package services

import (
	"backend/internal/core/domain"
	"backend/internal/core/dto"
	"backend/internal/core/port/output"
	"context"
	"strings"
	"time"
)

const bytesPerMB = 1 << 20

// Limits enforces each business's plan limits. Checks run before creating; two requests
// racing at the very last free slot can both pass (acceptable for cost limits).
type Limits struct {
	businesses  output.BusinessRepository
	sports      output.SportRepository
	resources   output.ResourceRepository
	coaches     output.CoachRepository
	classes     output.ClassSessionRepository
	members     output.MembershipRepository
	invitations output.InvitationRepository
	media       output.MediaRepository
	now         func() time.Time
}

func NewLimits(
	businesses output.BusinessRepository,
	sports output.SportRepository,
	resources output.ResourceRepository,
	coaches output.CoachRepository,
	classes output.ClassSessionRepository,
	members output.MembershipRepository,
	invitations output.InvitationRepository,
	media output.MediaRepository,
) *Limits {
	return &Limits{
		businesses: businesses, sports: sports, resources: resources, coaches: coaches,
		classes: classes, members: members, invitations: invitations, media: media, now: time.Now,
	}
}

// limitReached names the limit, e.g. PLAN_LIMIT_REACHED_SPORTS
func limitReached(name string) error {
	return conflict("PLAN_LIMIT_REACHED_" + strings.ToUpper(name))
}

func (l *Limits) limitsOf(ctx context.Context, businessID uint) (*domain.Business, domain.PlanLimits, error) {
	b, err := l.businesses.GetByID(ctx, businessID)
	if err != nil {
		return nil, domain.PlanLimits{}, err
	}
	return b, b.EffectiveLimits(), nil
}

// checkOneMore fails when one more item would exceed max
func checkOneMore(used int64, max int, name string) error {
	if used+1 > int64(max) {
		return limitReached(name)
	}
	return nil
}

func (l *Limits) CheckSports(ctx context.Context, businessID uint) error {
	_, lim, err := l.limitsOf(ctx, businessID)
	if err != nil {
		return err
	}
	used, err := l.sports.Count(ctx, businessID, nil)
	if err != nil {
		return err
	}
	return checkOneMore(used, lim.MaxSports, "sports")
}

func (l *Limits) CheckResources(ctx context.Context, businessID uint) error {
	_, lim, err := l.limitsOf(ctx, businessID)
	if err != nil {
		return err
	}
	used, err := l.resources.Count(ctx, businessID, nil)
	if err != nil {
		return err
	}
	return checkOneMore(used, lim.MaxResources, "resources")
}

func (l *Limits) CheckCoaches(ctx context.Context, businessID uint) error {
	_, lim, err := l.limitsOf(ctx, businessID)
	if err != nil {
		return err
	}
	used, err := l.coaches.Count(ctx, businessID, nil)
	if err != nil {
		return err
	}
	return checkOneMore(used, lim.MaxCoaches, "coaches")
}

// monthOf returns the calendar month containing t, in the business's timezone
func monthOf(t time.Time, timezone string) (from, to time.Time) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	local := t.In(loc)
	from = time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
	return from, from.AddDate(0, 1, 0)
}

// CheckClassSession checks the monthly class-session limit for the month a session starts in
func (l *Limits) CheckClassSession(ctx context.Context, businessID uint, startsAt time.Time) error {
	b, lim, err := l.limitsOf(ctx, businessID)
	if err != nil {
		return err
	}
	from, to := monthOf(startsAt, b.Timezone)
	used, err := l.classes.CountStartingBetween(ctx, businessID, from, to)
	if err != nil {
		return err
	}
	return checkOneMore(used, lim.MaxClassSessionsPerMonth, "class_sessions")
}

// CheckClassSessionMove checks the limit only when a session moves into another month
// (in the business's timezone); staying in the same month uses no extra allowance
func (l *Limits) CheckClassSessionMove(ctx context.Context, businessID uint, oldStart, newStart time.Time) error {
	b, err := l.businesses.GetByID(ctx, businessID)
	if err != nil {
		return err
	}
	oldFrom, _ := monthOf(oldStart, b.Timezone)
	newFrom, _ := monthOf(newStart, b.Timezone)
	if oldFrom.Equal(newFrom) {
		return nil
	}
	return l.CheckClassSession(ctx, businessID, newStart)
}

// membersUsed counts members plus pending invites: an invite reserves a seat
func (l *Limits) membersUsed(ctx context.Context, businessID uint) (int64, error) {
	members, err := l.members.Count(ctx, businessID)
	if err != nil {
		return 0, err
	}
	pending, err := l.invitations.CountPending(ctx, businessID, l.now())
	if err != nil {
		return 0, err
	}
	return members + pending, nil
}

func (l *Limits) CheckMembers(ctx context.Context, businessID uint) error {
	_, lim, err := l.limitsOf(ctx, businessID)
	if err != nil {
		return err
	}
	used, err := l.membersUsed(ctx, businessID)
	if err != nil {
		return err
	}
	return checkOneMore(used, lim.MaxMembers, "members")
}

// CheckStorage checks that one more image of addBytes fits both the image count and the storage size
func (l *Limits) CheckStorage(ctx context.Context, businessID uint, addBytes int64) error {
	_, lim, err := l.limitsOf(ctx, businessID)
	if err != nil {
		return err
	}
	images, bytes, err := l.media.Usage(ctx, businessID)
	if err != nil {
		return err
	}
	if err := checkOneMore(images, lim.MaxImages, "images"); err != nil {
		return err
	}
	if bytes+addBytes > int64(lim.MaxStorageMB)*bytesPerMB {
		return limitReached("storage")
	}
	return nil
}

// Usage reports current use against every limit, for the dashboard
func (l *Limits) Usage(ctx context.Context, businessID uint) (*dto.UsageDTO, error) {
	b, lim, err := l.limitsOf(ctx, businessID)
	if err != nil {
		return nil, err
	}
	sports, err := l.sports.Count(ctx, businessID, nil)
	if err != nil {
		return nil, err
	}
	resources, err := l.resources.Count(ctx, businessID, nil)
	if err != nil {
		return nil, err
	}
	coaches, err := l.coaches.Count(ctx, businessID, nil)
	if err != nil {
		return nil, err
	}
	from, to := monthOf(l.now(), b.Timezone)
	classes, err := l.classes.CountStartingBetween(ctx, businessID, from, to)
	if err != nil {
		return nil, err
	}
	members, err := l.membersUsed(ctx, businessID)
	if err != nil {
		return nil, err
	}
	images, bytes, err := l.media.Usage(ctx, businessID)
	if err != nil {
		return nil, err
	}
	item := func(used int64, limit int) dto.UsageItem { return dto.UsageItem{Used: used, Limit: int64(limit)} }
	return &dto.UsageDTO{
		Plan:                   b.Plan.Name,
		Sports:                 item(sports, lim.MaxSports),
		Resources:              item(resources, lim.MaxResources),
		Coaches:                item(coaches, lim.MaxCoaches),
		ClassSessionsThisMonth: item(classes, lim.MaxClassSessionsPerMonth),
		Members:                item(members, lim.MaxMembers),
		StorageMB:              item((bytes+bytesPerMB-1)/bytesPerMB, lim.MaxStorageMB),
		Images:                 item(images, lim.MaxImages),
	}, nil
}

// Location is the business's timezone (UTC if it cannot be loaded)
func (l *Limits) Location(ctx context.Context, businessID uint) (*time.Location, error) {
	b, err := l.businesses.GetByID(ctx, businessID)
	if err != nil {
		return nil, err
	}
	return locationOf(b), nil
}

// CheckClassSessionsBatch checks the monthly limit for many new sessions at once: for every month
// they fall in, the sessions already there plus the new ones must fit
func (l *Limits) CheckClassSessionsBatch(ctx context.Context, businessID uint, starts []time.Time) error {
	b, lim, err := l.limitsOf(ctx, businessID)
	if err != nil {
		return err
	}
	// keyed by "2006-01": time.Time keys would differ by their *Location even for the same month
	type month struct {
		from  time.Time
		added int64
	}
	perMonth := map[string]*month{}
	for _, s := range starts {
		from, _ := monthOf(s, b.Timezone)
		key := from.Format("2006-01")
		if perMonth[key] == nil {
			perMonth[key] = &month{from: from}
		}
		perMonth[key].added++
	}
	for _, mo := range perMonth {
		from, added := mo.from, mo.added
		used, err := l.classes.CountStartingBetween(ctx, businessID, from, from.AddDate(0, 1, 0))
		if err != nil {
			return err
		}
		if used+added > int64(lim.MaxClassSessionsPerMonth) {
			return limitReached("class_sessions")
		}
	}
	return nil
}
