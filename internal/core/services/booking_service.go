package services

import (
	"backend/internal/core/domain"
	"backend/internal/core/dto"
	"backend/internal/core/port/input"
	"backend/internal/core/port/output"
	"backend/pkg/response"
	"context"
	"crypto/rand"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode"
)

const (
	codeAlphabet  = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no 0/O or 1/I: easy to read out on the phone
	codeLength    = 8
	maxGuestName  = 120
	maxNotesChars = 1000
)

type BookingServiceImpl struct {
	businesses  output.BusinessRepository
	sports      output.SportRepository
	resources   output.ResourceRepository
	coaches     output.CoachRepository
	hours       output.OpeningHourRepository
	closures    output.ClosureRepository
	prices      output.PriceRepository
	classes     output.ClassSessionRepository
	bookings    output.BookingRepository
	rules       input.RuleService
	mailer      output.Mailer
	frontendURL string
	now         func() time.Time
}

func NewBookingService(
	businesses output.BusinessRepository,
	sports output.SportRepository,
	resources output.ResourceRepository,
	coaches output.CoachRepository,
	hours output.OpeningHourRepository,
	closures output.ClosureRepository,
	prices output.PriceRepository,
	classes output.ClassSessionRepository,
	bookings output.BookingRepository,
	rules input.RuleService,
	mailer output.Mailer,
	frontendURL string,
) input.BookingService {
	return &BookingServiceImpl{
		businesses: businesses, sports: sports, resources: resources, coaches: coaches, hours: hours,
		closures: closures, prices: prices, classes: classes, bookings: bookings, rules: rules,
		mailer: mailer, frontendURL: strings.TrimRight(frontendURL, "/"), now: time.Now,
	}
}

func locationOf(b *domain.Business) *time.Location {
	loc, err := time.LoadLocation(b.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

func startOfDay(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// ---------- what is being booked ----------

// owner is something that can be booked and has its own hours: a resource or a coach
type owner struct {
	ownerType string
	ownerID   uint
}

// target is a resource, a coach, or both, for one sport
type target struct {
	resource *domain.Resource
	coach    *domain.Coach
	sportID  uint
}

func (t target) resourceID() uint {
	if t.resource == nil {
		return 0
	}
	return t.resource.ID
}

// resolveTarget loads and checks what the customer asked for; everything must be active
// and belong to the business
func (s *BookingServiceImpl) resolveTarget(ctx context.Context, businessID, resourceID, coachID, sportID uint) (*target, error) {
	if resourceID == 0 && coachID == 0 {
		return nil, badRequest("RESOURCE_OR_COACH_REQUIRED")
	}
	t := &target{sportID: sportID}
	if resourceID != 0 {
		r, err := s.resources.GetByID(ctx, businessID, resourceID)
		if err != nil {
			return nil, orBadRequest(err, "RESOURCE_NOT_FOUND")
		}
		if !r.Active {
			return nil, badRequest("RESOURCE_NOT_AVAILABLE")
		}
		if t.sportID != 0 && t.sportID != r.SportID {
			return nil, badRequest("RESOURCE_SPORT_MISMATCH")
		}
		t.resource, t.sportID = r, r.SportID
	}
	if t.sportID == 0 {
		return nil, badRequest("SPORT_ID_REQUIRED")
	}
	sport, err := s.sports.GetByID(ctx, businessID, t.sportID)
	if err != nil {
		return nil, orBadRequest(err, "SPORT_NOT_FOUND")
	}
	if !sport.Active {
		return nil, badRequest("SPORT_NOT_AVAILABLE")
	}
	if coachID != 0 {
		c, err := s.coaches.GetByID(ctx, businessID, coachID)
		if err != nil {
			return nil, orBadRequest(err, "COACH_NOT_FOUND")
		}
		if !c.Active {
			return nil, badRequest("COACH_NOT_AVAILABLE")
		}
		if !teaches(*c, t.sportID) {
			return nil, badRequest("COACH_DOES_NOT_TEACH_SPORT")
		}
		t.coach = c
	}
	return t, nil
}

// openIntervals turns an owner's weekly hours into concrete intervals on day
func (s *BookingServiceImpl) openIntervals(ctx context.Context, businessID uint, ownerType string, ownerID uint, day time.Time) ([]domain.Interval, error) {
	rows, err := s.hours.ListByOwner(ctx, businessID, ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	var out []domain.Interval
	for _, h := range rows {
		if h.Weekday != int(day.Weekday()) {
			continue
		}
		open, err1 := parseClock(h.OpenTime, false)
		closeAt, err2 := parseClock(h.CloseTime, true)
		if err1 != nil || err2 != nil {
			continue // stored hours were validated when saved
		}
		out = append(out, domain.Interval{
			Start: day.Add(time.Duration(open) * time.Minute),
			End:   day.Add(time.Duration(closeAt) * time.Minute),
		})
	}
	return out, nil
}

// busyIntervals collects the owner's active bookings and class holds on [from, to)
func (s *BookingServiceImpl) busyIntervals(ctx context.Context, businessID uint, ownerType string, ownerID uint, from, to time.Time) ([]domain.Interval, error) {
	items, err := s.bookings.ActiveItems(ctx, businessID, ownerType, ownerID, from, to)
	if err != nil {
		return nil, err
	}
	return mapSlice(items, func(i domain.BookingItem) domain.Interval {
		return domain.Interval{Start: i.StartsAt, End: i.EndsAt}
	}), nil
}

// slotsFor computes every bookable start on day for the target. It is the single definition
// of "free", used both to show availability and to accept a booking.
func (s *BookingServiceImpl) slotsFor(ctx context.Context, b *domain.Business, t *target, day time.Time) ([]domain.Slot, *domain.EffectiveRule, error) {
	rule, err := s.rules.Effective(ctx, b.ID, t.sportID, t.resourceID())
	if err != nil {
		return nil, nil, err
	}
	from, to := day, day.AddDate(0, 0, 1)

	var open []domain.Interval
	var busy []domain.Interval
	var owners []owner
	if t.resource != nil {
		owners = append(owners, owner{domain.OwnerResource, t.resource.ID})
	}
	if t.coach != nil {
		owners = append(owners, owner{domain.OwnerCoach, t.coach.ID})
	}
	for i, o := range owners {
		ownerOpen, err := s.openIntervals(ctx, b.ID, o.ownerType, o.ownerID, day)
		if err != nil {
			return nil, nil, err
		}
		if i == 0 {
			open = ownerOpen
		} else {
			open = domain.Intersect(open, ownerOpen) // coach + court: both must be open
		}
		ownerBusy, err := s.busyIntervals(ctx, b.ID, o.ownerType, o.ownerID, from, to)
		if err != nil {
			return nil, nil, err
		}
		busy = append(busy, ownerBusy...)
	}

	closures, err := s.closures.Overlapping(ctx, b.ID, from, to)
	if err != nil {
		return nil, nil, err
	}
	for _, c := range closures {
		applies := c.OwnerType == domain.OwnerVenue
		for _, o := range owners {
			applies = applies || (c.OwnerType == o.ownerType && c.OwnerID == o.ownerID)
		}
		if applies {
			busy = append(busy, domain.Interval{Start: c.StartsAt, End: c.EndsAt})
		}
	}

	now := s.now()
	window := domain.SlotWindow{
		Earliest: now.Add(time.Duration(rule.MinNoticeMin) * time.Minute),
		Latest:   startOfDay(now, day.Location()).AddDate(0, 0, rule.MaxDaysAhead+1),
	}
	return domain.FreeSlots(open, busy, *rule, window), rule, nil
}

// ---------- public business ----------

func (s *BookingServiceImpl) published(ctx context.Context, slug string) (*domain.Business, error) {
	return loadPublished(ctx, s.businesses, slug)
}

func (s *BookingServiceImpl) Availability(ctx context.Context, slug string, q dto.AvailabilityQuery) (*dto.AvailabilityDTO, error) {
	b, err := s.published(ctx, slug)
	if err != nil {
		return nil, err
	}
	loc := locationOf(b)
	day, err := time.ParseInLocation("2006-01-02", q.Date, loc)
	if err != nil {
		return nil, badRequest("INVALID_DATE")
	}
	t, err := s.resolveTarget(ctx, b.ID, q.ResourceID, q.CoachID, q.SportID)
	if err != nil {
		return nil, err
	}
	slots, rule, err := s.slotsFor(ctx, b, t, day)
	if err != nil {
		return nil, err
	}
	return &dto.AvailabilityDTO{
		Date: q.Date, Timezone: loc.String(), Rule: *rule,
		Slots: mapSlice(slots, func(sl domain.Slot) dto.SlotDTO {
			return dto.SlotDTO{StartsAt: sl.StartsAt.In(loc), Durations: sl.Durations}
		}),
	}, nil
}

// ---------- creating ----------

// booker is who the booking is for: a logged-in customer or a guest
type booker struct {
	customer *domain.User
	guest    *dto.GuestInfo
}

// whoBooks applies the business's booking mode
func whoBooks(b *domain.Business, actor *domain.User, guest *dto.GuestInfo) (*booker, error) {
	switch {
	case b.BookingMode == domain.BookingModeAccount || (b.BookingMode == domain.BookingModeBoth && actor != nil):
		if actor == nil {
			return nil, response.NewError(http.StatusUnauthorized, "LOGIN_REQUIRED")
		}
		return &booker{customer: actor}, nil
	default: // guest mode, or both without a login
		if guest == nil {
			return nil, badRequest("GUEST_DETAILS_REQUIRED")
		}
		g, err := cleanGuest(*guest)
		if err != nil {
			return nil, err
		}
		return &booker{guest: g}, nil
	}
}

// cleanGuest validates a guest's details; the phone keeps only digits and a leading +
func cleanGuest(g dto.GuestInfo) (*dto.GuestInfo, error) {
	g.Name = strings.TrimSpace(g.Name)
	if g.Name == "" || len([]rune(g.Name)) > maxGuestName {
		return nil, badRequest("GUEST_NAME_REQUIRED")
	}
	var phone strings.Builder
	digits := 0
	for i, r := range strings.TrimSpace(g.Phone) {
		switch {
		case unicode.IsDigit(r):
			phone.WriteRune(r)
			digits++
		case r == '+' && i == 0:
			phone.WriteRune(r)
		case r == ' ' || r == '-' || r == '(' || r == ')':
		default:
			return nil, badRequest("INVALID_PHONE")
		}
	}
	if digits < 6 || digits > 15 {
		return nil, badRequest("INVALID_PHONE")
	}
	g.Phone = phone.String()
	g.Email = normalizeEmail(g.Email)
	if g.Email != "" {
		if addr, err := mail.ParseAddress(g.Email); err != nil || addr.Address != g.Email {
			return nil, badRequest("INVALID_EMAIL")
		}
	}
	return &g, nil
}

// checkBookingLimit applies the per-customer rule (by account, or by phone for guests)
func (s *BookingServiceImpl) checkBookingLimit(ctx context.Context, businessID uint, who *booker, rule *domain.EffectiveRule) error {
	var active int64
	var err error
	if who.customer != nil {
		active, err = s.bookings.CountActiveForCustomer(ctx, businessID, who.customer.ID, s.now())
	} else {
		active, err = s.bookings.CountActiveForGuestPhone(ctx, businessID, who.guest.Phone, s.now())
	}
	if err != nil {
		return err
	}
	if active >= int64(rule.MaxActiveBookingsPerCustomer) {
		return conflict("BOOKING_LIMIT_REACHED")
	}
	return nil
}

func newBookingCode() (string, error) {
	b := make([]byte, codeLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = codeAlphabet[int(b[i])%len(codeAlphabet)]
	}
	return string(b), nil
}

// newBooking fills in what every booking has: who, status, code, cancel deadline, guest token
func (s *BookingServiceImpl) newBooking(b *domain.Business, who *booker, start, end time.Time, rule *domain.EffectiveRule, notes string) (*domain.Booking, string, error) {
	code, err := newBookingCode()
	if err != nil {
		return nil, "", err
	}
	status := domain.BookingConfirmed
	if b.RequireApproval {
		status = domain.BookingPending
	}
	bk := &domain.Booking{
		BusinessID: b.ID, Code: code, Status: status, StartsAt: start, EndsAt: end,
		PaymentStatus: domain.PaymentUnpaid, Notes: strings.TrimSpace(notes),
		CancellableUntil: start.Add(-time.Duration(rule.CancelBeforeHours) * time.Hour),
	}
	var rawToken string
	if who.customer != nil {
		id := who.customer.ID
		bk.CustomerID, bk.CreatedByType = &id, domain.CreatedByCustomer
	} else {
		raw, hash, err := newToken()
		if err != nil {
			return nil, "", err
		}
		rawToken = raw
		bk.GuestName, bk.GuestPhone, bk.GuestEmail = who.guest.Name, who.guest.Phone, who.guest.Email
		bk.GuestTokenHash, bk.CreatedByType = hash, domain.CreatedByGuest
	}
	return bk, rawToken, nil
}

func (s *BookingServiceImpl) Create(ctx context.Context, slug string, actor *domain.User, req dto.CreateBookingRequest) (*dto.CreatedBookingDTO, error) {
	b, err := s.published(ctx, slug)
	if err != nil {
		return nil, err
	}
	if len([]rune(req.Notes)) > maxNotesChars {
		return nil, badRequest("NOTES_TOO_LONG")
	}
	who, err := whoBooks(b, actor, req.Guest)
	if err != nil {
		return nil, err
	}

	var bk *domain.Booking
	var rawToken string
	if req.ClassSessionID != nil {
		bk, rawToken, err = s.createClassBooking(ctx, b, who, req)
	} else {
		bk, rawToken, err = s.createRental(ctx, b, who, req)
	}
	if err != nil {
		return nil, err
	}

	bk.Business = *b
	out := &dto.CreatedBookingDTO{BookingDTO: toBookingDTO(*bk)}
	if rawToken != "" {
		out.GuestToken = rawToken
		out.GuestLink = s.frontendURL + "/b/" + b.Slug + "/booking/" + rawToken
		if bk.GuestEmail != "" {
			if err := s.mailer.SendBookingLink(bk.GuestEmail, b.Name, bk.Code, out.GuestLink); err != nil {
				log.Printf("⚠️  booking link email for %s failed: %v", bk.Code, err) // the link is also in the response
			}
		}
	}
	return out, nil
}

// createRental books a resource, a coach, or both for starts_at + duration_min
func (s *BookingServiceImpl) createRental(ctx context.Context, b *domain.Business, who *booker, req dto.CreateBookingRequest) (*domain.Booking, string, error) {
	var resourceID, coachID uint
	if req.ResourceID != nil {
		resourceID = *req.ResourceID
	}
	if req.CoachID != nil {
		coachID = *req.CoachID
	}
	t, err := s.resolveTarget(ctx, b.ID, resourceID, coachID, req.SportID)
	if err != nil {
		return nil, "", err
	}
	if req.StartsAt.IsZero() {
		return nil, "", badRequest("STARTS_AT_REQUIRED")
	}
	loc := locationOf(b)
	slots, rule, err := s.slotsFor(ctx, b, t, startOfDay(req.StartsAt, loc))
	if err != nil {
		return nil, "", err
	}
	d := req.DurationMin
	if d < rule.MinDurationMin || d > rule.MaxDurationMin || (d-rule.MinDurationMin)%rule.DurationStepMin != 0 {
		return nil, "", badRequest("INVALID_DURATION")
	}
	if !domain.SlotAllows(slots, req.StartsAt, d) {
		return nil, "", conflict("SLOT_NOT_AVAILABLE")
	}
	if err := s.checkBookingLimit(ctx, b.ID, who, rule); err != nil {
		return nil, "", err
	}

	start := req.StartsAt.In(loc)
	end := start.Add(time.Duration(d) * time.Minute)
	var items []domain.BookingItem
	var total int64
	currency := ""
	add := func(ownerType string, ownerID uint) error {
		price, err := s.prices.GetByOwner(ctx, b.ID, ownerType, ownerID)
		if err != nil {
			if isNotFound(err) {
				return conflict("PRICE_NOT_SET")
			}
			return err
		}
		if currency != "" && currency != price.Currency {
			return conflict("PRICE_CURRENCY_MISMATCH")
		}
		currency = price.Currency
		item := domain.BookingItem{BusinessID: b.ID, Kind: ownerType, StartsAt: start, EndsAt: end, Price: domain.PriceFor(price.PricePerHour, d), Active: true}
		id := ownerID
		if ownerType == domain.OwnerResource {
			item.ResourceID = &id
		} else {
			item.CoachID = &id
		}
		total += item.Price
		items = append(items, item)
		return nil
	}
	if t.resource != nil {
		if err := add(domain.OwnerResource, t.resource.ID); err != nil {
			return nil, "", err
		}
	}
	if t.coach != nil {
		if err := add(domain.OwnerCoach, t.coach.ID); err != nil {
			return nil, "", err
		}
	}

	bk, rawToken, err := s.newBooking(b, who, start, end, rule, req.Notes)
	if err != nil {
		return nil, "", err
	}
	bk.TotalPrice, bk.Currency, bk.Items = total, currency, items
	// the database refuses the insert if another booking took the time meanwhile (409)
	if err := s.bookings.Create(ctx, bk); err != nil {
		return nil, "", err
	}
	return bk, rawToken, nil
}

// createClassBooking books seats in a group class
func (s *BookingServiceImpl) createClassBooking(ctx context.Context, b *domain.Business, who *booker, req dto.CreateBookingRequest) (*domain.Booking, string, error) {
	class, err := s.classes.GetByID(ctx, b.ID, *req.ClassSessionID)
	if err != nil {
		return nil, "", orBadRequest(err, "CLASS_NOT_FOUND")
	}
	if !class.Active {
		return nil, "", badRequest("CLASS_NOT_AVAILABLE")
	}
	seats := req.Seats
	if seats == 0 {
		seats = 1
	}
	if seats < 1 {
		return nil, "", badRequest("INVALID_SEATS")
	}
	var resourceID uint
	if class.ResourceID != nil {
		resourceID = *class.ResourceID
	}
	rule, err := s.rules.Effective(ctx, b.ID, class.SportID, resourceID)
	if err != nil {
		return nil, "", err
	}
	if class.StartsAt.Before(s.now().Add(time.Duration(rule.MinNoticeMin) * time.Minute)) {
		return nil, "", conflict("CLASS_STARTED_OR_TOO_SOON")
	}
	if err := s.checkBookingLimit(ctx, b.ID, who, rule); err != nil {
		return nil, "", err
	}

	loc := locationOf(b)
	start, end := class.StartsAt.In(loc), class.EndsAt.In(loc)
	bk, rawToken, err := s.newBooking(b, who, start, end, rule, req.Notes)
	if err != nil {
		return nil, "", err
	}
	classID := class.ID
	price := class.Price * int64(seats)
	bk.TotalPrice, bk.Currency = price, class.Currency
	bk.Items = []domain.BookingItem{{
		BusinessID: b.ID, Kind: domain.ItemClassSession, ClassSessionID: &classID, Seats: seats,
		StartsAt: start, EndsAt: end, Price: price, Active: true,
	}}
	// the class row is locked while seats are counted, so the last seat goes to one customer only
	if err := s.bookings.CreateClassBooking(ctx, bk, class.ID, seats); err != nil {
		return nil, "", err
	}
	return bk, rawToken, nil
}

// ---------- viewing and cancelling ----------

func toBookingDTO(b domain.Booking) dto.BookingDTO {
	return dto.BookingDTO{
		ID: b.ID, Code: b.Code, BusinessName: b.Business.Name, BusinessSlug: b.Business.Slug,
		Status: b.Status, StartsAt: b.StartsAt, EndsAt: b.EndsAt, TotalPrice: b.TotalPrice, Currency: b.Currency,
		PaymentStatus: b.PaymentStatus, CancellableUntil: b.CancellableUntil, GuestName: b.GuestName,
		Notes: b.Notes, CancelledAt: b.CancelledAt, CancelReason: b.CancelReason, CreatedAt: b.CreatedAt,
		Items: mapSlice(b.Items, func(i domain.BookingItem) dto.BookingItemDTO {
			return dto.BookingItemDTO{
				Kind: i.Kind, ResourceID: i.ResourceID, CoachID: i.CoachID, ClassSessionID: i.ClassSessionID,
				Seats: i.Seats, StartsAt: i.StartsAt, EndsAt: i.EndsAt, Price: i.Price,
			}
		}),
	}
}

// cancel applies the customer rules: only active bookings, and only until the cancel deadline
func (s *BookingServiceImpl) cancel(ctx context.Context, b *domain.Booking, reason string) (*dto.BookingDTO, error) {
	if !b.IsActive() {
		return nil, conflict("BOOKING_NOT_ACTIVE")
	}
	now := s.now()
	if now.After(b.CancellableUntil) {
		return nil, conflict("CANCELLATION_WINDOW_PASSED")
	}
	if len([]rune(reason)) > 255 {
		return nil, badRequest("REASON_TOO_LONG")
	}
	if err := s.bookings.Cancel(ctx, b, strings.TrimSpace(reason), now); err != nil {
		return nil, err
	}
	d := toBookingDTO(*b)
	return &d, nil
}

// guestBooking finds a booking by its private link; any business status, so guests can still
// see and cancel after a business unpublishes
func (s *BookingServiceImpl) guestBooking(ctx context.Context, slug, token string) (*domain.Booking, error) {
	gone := notFound("BOOKING_NOT_FOUND")
	b, err := s.businesses.GetBySlug(ctx, slug)
	if err != nil {
		if isNotFound(err) {
			return nil, gone
		}
		return nil, err
	}
	bk, err := s.bookings.GetByGuestToken(ctx, b.ID, hashToken(token))
	if err != nil {
		return nil, err
	}
	if bk == nil {
		return nil, gone
	}
	return bk, nil
}

func (s *BookingServiceImpl) GuestView(ctx context.Context, slug, token string) (*dto.BookingDTO, error) {
	bk, err := s.guestBooking(ctx, slug, token)
	if err != nil {
		return nil, err
	}
	d := toBookingDTO(*bk)
	return &d, nil
}

func (s *BookingServiceImpl) GuestCancel(ctx context.Context, slug, token string, req dto.CancelBookingRequest) (*dto.BookingDTO, error) {
	bk, err := s.guestBooking(ctx, slug, token)
	if err != nil {
		return nil, err
	}
	return s.cancel(ctx, bk, req.Reason)
}

var bookingStatuses = map[string]bool{
	domain.BookingPending: true, domain.BookingConfirmed: true, domain.BookingCancelled: true,
	domain.BookingCompleted: true, domain.BookingNoShow: true,
}

func (s *BookingServiceImpl) MyBookings(ctx context.Context, userID uint, q dto.ListQuery) (*dto.PagedResult[dto.BookingDTO], error) {
	lq, page, limit := newListQuery(q)
	if q.Status != "" {
		if !bookingStatuses[q.Status] {
			return nil, badRequest("INVALID_STATUS")
		}
		lq.Filters["status"] = q.Status
	}
	items, total, err := s.bookings.ListForCustomer(ctx, userID, lq)
	if err != nil {
		return nil, err
	}
	return pageOf(items, total, page, limit, toBookingDTO), nil
}

func (s *BookingServiceImpl) MyBooking(ctx context.Context, userID, id uint) (*dto.BookingDTO, error) {
	bk, err := s.bookings.GetForCustomer(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	d := toBookingDTO(*bk)
	return &d, nil
}

func (s *BookingServiceImpl) CancelMine(ctx context.Context, userID, id uint, req dto.CancelBookingRequest) (*dto.BookingDTO, error) {
	bk, err := s.bookings.GetForCustomer(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return s.cancel(ctx, bk, req.Reason)
}
