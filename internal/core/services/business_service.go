package services

import (
	"backend/internal/core/domain"
	"backend/internal/core/dto"
	"backend/internal/core/port/input"
	"backend/internal/core/port/output"
	"context"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	brandColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	languages         = map[string]bool{"en": true, "lo": true, "th": true}
	bookingModes      = map[string]bool{domain.BookingModeAccount: true, domain.BookingModeGuest: true, domain.BookingModeBoth: true}
)

const (
	maxSocialLinks       = 10
	maxDescriptionChars  = 2000
	maxBookingPolicyChar = 5000
)

type BusinessServiceImpl struct {
	businesses  output.BusinessRepository
	memberships output.MembershipRepository
	invitations output.InvitationRepository
	sports      output.SportRepository
	resources   output.ResourceRepository
	hours       output.OpeningHourRepository
	prices      output.PriceRepository
	visits      output.SupportVisitRepository
	files       mediaFiles
	limits      *Limits
	invites     invites
	now         func() time.Time
}

func NewBusinessService(
	businesses output.BusinessRepository,
	memberships output.MembershipRepository,
	invitations output.InvitationRepository,
	users output.UserRepository,
	sports output.SportRepository,
	resources output.ResourceRepository,
	hours output.OpeningHourRepository,
	prices output.PriceRepository,
	visits output.SupportVisitRepository,
	storage output.FileStorage,
	media output.MediaRepository,
	limits *Limits,
	mailer output.Mailer,
	frontendURL string,
) input.BusinessService {
	return &BusinessServiceImpl{
		businesses: businesses, memberships: memberships, invitations: invitations,
		sports: sports, resources: resources, hours: hours, prices: prices, visits: visits,
		files: mediaFiles{storage: storage, media: media}, limits: limits,
		invites: newInvites(invitations, memberships, users, limits, mailer, frontendURL),
		now:     time.Now,
	}
}

// ---------- settings ----------

func (s *BusinessServiceImpl) toSettingsDTO(b *domain.Business) *dto.SettingsDTO {
	return &dto.SettingsDTO{
		SettingsRequest: dto.SettingsRequest{
			Name: b.Name, Slug: b.Slug, Timezone: b.Timezone, Currency: b.Currency, Language: b.Language,
			LogoKey: b.LogoKey, CoverKey: b.CoverKey, BrandColor: b.BrandColor, Description: b.Description,
			Address: b.Address, Province: b.Province, Phone: b.Phone, MapURL: b.MapURL, SocialLinks: b.SocialLinks,
			BookingPolicy: b.BookingPolicy, BookingMode: b.BookingMode,
			RequireApproval: b.RequireApproval,
		},
		ID: b.ID, Status: b.Status(), LogoURL: s.files.url(b.LogoKey), CoverURL: s.files.url(b.CoverKey),
		PublicPath: "/b/" + b.Slug, PublishedAt: b.PublishedAt,
	}
}

func (s *BusinessServiceImpl) GetSettings(ctx context.Context, businessID uint) (*dto.SettingsDTO, error) {
	b, err := s.businesses.GetByID(ctx, businessID)
	if err != nil {
		return nil, err
	}
	return s.toSettingsDTO(b), nil
}

// validLink accepts empty or an absolute http(s) URL
func validLink(raw string) bool {
	if raw == "" {
		return true
	}
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func (s *BusinessServiceImpl) validateSettings(ctx context.Context, b *domain.Business, req *dto.SettingsRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	req.Slug = strings.TrimSpace(req.Slug)
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	switch {
	case req.Name == "":
		return badRequest("NAME_IS_REQUIRED")
	case !domain.ValidProvince(req.Province):
		return badRequest("INVALID_PROVINCE")
	case !validCurrency(req.Currency):
		return badRequest("INVALID_CURRENCY")
	case !languages[req.Language]:
		return badRequest("INVALID_LANGUAGE")
	case req.BrandColor != "" && !brandColorPattern.MatchString(req.BrandColor):
		return badRequest("INVALID_BRAND_COLOR")
	case !bookingModes[req.BookingMode]:
		return badRequest("INVALID_BOOKING_MODE")
	case !validLink(req.MapURL):
		return badRequest("INVALID_MAP_URL")
	case len(req.SocialLinks) > maxSocialLinks:
		return badRequest("TOO_MANY_SOCIAL_LINKS")
	case utf8.RuneCountInString(req.Description) > maxDescriptionChars:
		return badRequest("DESCRIPTION_TOO_LONG")
	case utf8.RuneCountInString(req.BookingPolicy) > maxBookingPolicyChar:
		return badRequest("BOOKING_POLICY_TOO_LONG")
	}
	if _, err := time.LoadLocation(req.Timezone); err != nil || req.Timezone == "" {
		return badRequest("INVALID_TIMEZONE")
	}
	for name, link := range req.SocialLinks {
		if name == "" || len(name) > 30 || !validLink(link) || link == "" {
			return badRequest("INVALID_SOCIAL_LINK")
		}
	}
	if req.Slug != b.Slug {
		if b.PublishedAt != nil {
			return conflict("SLUG_LOCKED_AFTER_PUBLISH")
		}
		if !validSlug(req.Slug) {
			return badRequest("INVALID_SLUG")
		}
		taken, err := s.businesses.SlugTaken(ctx, req.Slug, b.ID)
		if err != nil {
			return err
		}
		if taken {
			return conflict("SLUG_TAKEN")
		}
	}
	if err := s.files.requireOwned(ctx, b.ID, req.LogoKey); err != nil {
		return err
	}
	return s.files.requireOwned(ctx, b.ID, req.CoverKey)
}

// UpdateSettings replaces every setting at once (like PUT elsewhere)
func (s *BusinessServiceImpl) UpdateSettings(ctx context.Context, businessID uint, req dto.SettingsRequest) (*dto.SettingsDTO, error) {
	b, err := s.businesses.GetByID(ctx, businessID)
	if err != nil {
		return nil, err
	}
	if err := s.validateSettings(ctx, b, &req); err != nil {
		return nil, err
	}
	oldLogo, oldCover := b.LogoKey, b.CoverKey
	b.Name, b.Slug, b.Timezone, b.Currency, b.Language = req.Name, req.Slug, req.Timezone, req.Currency, req.Language
	b.LogoKey, b.CoverKey, b.BrandColor = req.LogoKey, req.CoverKey, req.BrandColor
	b.Description, b.Address, b.Phone, b.MapURL = req.Description, strings.TrimSpace(req.Address), strings.TrimSpace(req.Phone), req.MapURL
	b.Province = req.Province
	b.SocialLinks, b.BookingPolicy, b.BookingMode = req.SocialLinks, req.BookingPolicy, req.BookingMode
	b.RequireApproval = req.RequireApproval
	if err := s.businesses.Update(ctx, b); err != nil {
		return nil, err
	}
	s.files.replaced(ctx, businessID, oldLogo, b.LogoKey)
	s.files.replaced(ctx, businessID, oldCover, b.CoverKey)
	return s.toSettingsDTO(b), nil
}

// bookable reports whether at least one active resource of an active sport has hours and a price
func (s *BusinessServiceImpl) bookable(ctx context.Context, businessID uint) (bool, error) {
	resources, _, err := s.resources.List(ctx, businessID, output.ListQuery{Filters: map[string]any{"active": true}})
	if err != nil {
		return false, err
	}
	for _, r := range resources {
		sport, err := s.sports.GetByID(ctx, businessID, r.SportID)
		if err != nil || !sport.Active {
			continue
		}
		hours, err := s.hours.ListByOwner(ctx, businessID, domain.OwnerResource, r.ID)
		if err != nil {
			return false, err
		}
		if len(hours) == 0 {
			continue
		}
		if _, err := s.prices.GetByOwner(ctx, businessID, domain.OwnerResource, r.ID); err == nil {
			return true, nil
		} else if !isNotFound(err) {
			return false, err
		}
	}
	return false, nil
}

// Publish makes the public page live once something can actually be booked
func (s *BusinessServiceImpl) Publish(ctx context.Context, businessID uint) (*dto.SettingsDTO, error) {
	b, err := s.businesses.GetByID(ctx, businessID)
	if err != nil {
		return nil, err
	}
	ok, err := s.bookable(ctx, businessID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, badRequest("PUBLISH_NEEDS_ACTIVE_RESOURCE_WITH_HOURS_AND_PRICE")
	}
	b.Published = true
	if b.PublishedAt == nil {
		now := s.now()
		b.PublishedAt = &now
	}
	if err := s.businesses.Update(ctx, b); err != nil {
		return nil, err
	}
	return s.toSettingsDTO(b), nil
}

func (s *BusinessServiceImpl) Unpublish(ctx context.Context, businessID uint) (*dto.SettingsDTO, error) {
	b, err := s.businesses.GetByID(ctx, businessID)
	if err != nil {
		return nil, err
	}
	b.Published = false
	if err := s.businesses.Update(ctx, b); err != nil {
		return nil, err
	}
	return s.toSettingsDTO(b), nil
}

func (s *BusinessServiceImpl) Usage(ctx context.Context, businessID uint) (*dto.UsageDTO, error) {
	return s.limits.Usage(ctx, businessID)
}

// ---------- members ----------

func toMemberDTO(m domain.Membership) dto.MemberDTO {
	return dto.MemberDTO{
		UserID: m.UserID, Name: m.User.Name, Email: m.User.ContactEmail(),
		Role: m.Role, Active: m.Active, JoinedAt: m.CreatedAt,
	}
}

func (s *BusinessServiceImpl) ListMembers(ctx context.Context, businessID uint) ([]dto.MemberDTO, error) {
	members, err := s.memberships.ListByBusiness(ctx, businessID)
	if err != nil {
		return nil, err
	}
	return mapSlice(members, toMemberDTO), nil
}

// member loads one membership or 404
func (s *BusinessServiceImpl) member(ctx context.Context, businessID, userID uint) (*domain.Membership, error) {
	m, err := s.memberships.Get(ctx, businessID, userID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, notFound("MEMBER_NOT_FOUND")
	}
	return m, nil
}

// UpdateMember changes a manager's or staff member's role or disables them.
// The owner is changed only through TransferOwnership.
func (s *BusinessServiceImpl) UpdateMember(ctx context.Context, businessID, userID uint, req dto.UpdateMemberRequest) (*dto.MemberDTO, error) {
	m, err := s.member(ctx, businessID, userID)
	if err != nil {
		return nil, err
	}
	if m.Role == domain.MemberOwner {
		return nil, badRequest("USE_TRANSFER_OWNERSHIP")
	}
	if req.Role != nil {
		if *req.Role != domain.MemberManager && *req.Role != domain.MemberStaff {
			return nil, badRequest("INVALID_ROLE")
		}
		m.Role = *req.Role
	}
	if req.Active != nil {
		m.Active = *req.Active
	}
	if err := s.memberships.Update(ctx, m); err != nil {
		return nil, err
	}
	members, err := s.memberships.ListByBusiness(ctx, businessID)
	if err != nil {
		return nil, err
	}
	for _, x := range members {
		if x.UserID == userID {
			d := toMemberDTO(x)
			return &d, nil
		}
	}
	return nil, notFound("MEMBER_NOT_FOUND")
}

func (s *BusinessServiceImpl) RemoveMember(ctx context.Context, businessID, userID uint) error {
	m, err := s.member(ctx, businessID, userID)
	if err != nil {
		return err
	}
	if m.Role == domain.MemberOwner {
		return badRequest("OWNER_CANNOT_BE_REMOVED")
	}
	return s.memberships.Delete(ctx, businessID, userID)
}

// TransferOwnership hands the business to another member; the old owner becomes a manager
func (s *BusinessServiceImpl) TransferOwnership(ctx context.Context, businessID, actorID, toUserID uint) error {
	if actorID == toUserID {
		return badRequest("ALREADY_THE_OWNER")
	}
	if _, err := s.member(ctx, businessID, toUserID); err != nil {
		return err
	}
	return s.memberships.TransferOwnership(ctx, businessID, actorID, toUserID)
}

// ---------- invitations ----------

func (s *BusinessServiceImpl) ListInvitations(ctx context.Context, businessID uint) ([]dto.InvitationDTO, error) {
	items, err := s.invitations.ListPending(ctx, businessID, s.now())
	if err != nil {
		return nil, err
	}
	return mapSlice(items, toInvitationDTO), nil
}

// Invite adds a manager or staff member; the owner comes from the Super Admin's invite
func (s *BusinessServiceImpl) Invite(ctx context.Context, businessID uint, actor *domain.User, req dto.InviteRequest) (*dto.InvitationDTO, error) {
	if req.Role != domain.MemberManager && req.Role != domain.MemberStaff {
		return nil, badRequest("INVALID_ROLE")
	}
	b, err := s.businesses.GetByID(ctx, businessID)
	if err != nil {
		return nil, err
	}
	return s.invites.create(ctx, b, actor, req.Email, req.Role)
}

func (s *BusinessServiceImpl) CancelInvitation(ctx context.Context, businessID, id uint) error {
	return s.invitations.Delete(ctx, businessID, id)
}

func (s *BusinessServiceImpl) ResendInvitation(ctx context.Context, businessID uint, actor *domain.User, id uint) (*dto.InvitationDTO, error) {
	b, err := s.businesses.GetByID(ctx, businessID)
	if err != nil {
		return nil, err
	}
	inv, err := s.invitations.GetByID(ctx, businessID, id)
	if err != nil {
		return nil, err
	}
	return s.invites.resend(ctx, b, actor, inv)
}

// ---------- support visits ----------

func (s *BusinessServiceImpl) ListSupportVisits(ctx context.Context, businessID uint, q dto.ListQuery) (*dto.PagedResult[dto.SupportVisitDTO], error) {
	lq, page, limit := newListQuery(q)
	items, total, err := s.visits.List(ctx, businessID, lq)
	if err != nil {
		return nil, err
	}
	return pageOf(items, total, page, limit, func(v domain.SupportVisit) dto.SupportVisitDTO {
		return dto.SupportVisitDTO{UserID: v.UserID, VisitedAt: v.VisitedAt}
	}), nil
}
