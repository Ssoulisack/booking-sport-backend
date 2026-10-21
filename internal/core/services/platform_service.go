package services

import (
	"backend/internal/core/domain"
	"backend/internal/core/dto"
	"backend/internal/core/port/input"
	"backend/internal/core/port/output"
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type PlatformServiceImpl struct {
	plans       output.PlanRepository
	businesses  output.BusinessRepository
	memberships output.MembershipRepository
	invitations output.InvitationRepository
	users       output.UserRepository
	invites     invites
}

func NewPlatformService(
	plans output.PlanRepository,
	businesses output.BusinessRepository,
	memberships output.MembershipRepository,
	invitations output.InvitationRepository,
	users output.UserRepository,
	limits *Limits,
	mailer output.Mailer,
	frontendURL string,
) input.PlatformService {
	return &PlatformServiceImpl{
		plans: plans, businesses: businesses, memberships: memberships, invitations: invitations, users: users,
		invites: newInvites(invitations, memberships, users, limits, mailer, frontendURL),
	}
}

// ---------- plans ----------

func toPlanDTO(p domain.Plan) dto.PlanDTO {
	return dto.PlanDTO{ID: p.ID, Name: p.Name, Limits: p.Limits}
}

func (s *PlatformServiceImpl) ListPlans(ctx context.Context) ([]dto.PlanDTO, error) {
	plans, err := s.plans.List(ctx)
	if err != nil {
		return nil, err
	}
	return mapSlice(plans, toPlanDTO), nil
}

func validLimits(l domain.PlanLimits) bool {
	return l.MaxSports >= 0 && l.MaxResources >= 0 && l.MaxCoaches >= 0 && l.MaxClassSessionsPerMonth >= 0 &&
		l.MaxMembers >= 1 && l.MaxStorageMB >= 0 && l.MaxImages >= 0
}

func (s *PlatformServiceImpl) UpdatePlan(ctx context.Context, id uint, req dto.PlanRequest) (*dto.PlanDTO, error) {
	if !validLimits(req.Limits) {
		return nil, badRequest("INVALID_LIMITS")
	}
	p, err := s.plans.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	p.Limits = req.Limits
	if err := s.plans.Update(ctx, p); err != nil {
		return nil, err
	}
	d := toPlanDTO(*p)
	return &d, nil
}

// ---------- businesses ----------

func toBusinessDTO(b domain.Business) dto.BusinessDTO {
	return dto.BusinessDTO{
		ID: b.ID, Name: b.Name, Slug: b.Slug, Status: b.Status(), Plan: toPlanDTO(b.Plan),
		Overrides: b.Overrides, Limits: b.EffectiveLimits(), PublishedAt: b.PublishedAt, CreatedAt: b.CreatedAt,
	}
}

// uniqueSlug picks a free slug: the wanted one, then wanted-2 ... wanted-9, then a random suffix
func (s *PlatformServiceImpl) uniqueSlug(ctx context.Context, wanted string) (string, error) {
	for i := 1; i <= 9; i++ {
		candidate := wanted
		if i > 1 {
			candidate = fmt.Sprintf("%s-%d", wanted, i)
		}
		taken, err := s.businesses.SlugTaken(ctx, candidate, 0)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}
	return wanted + "-" + uuid.NewString()[:6], nil
}

// CreateBusiness starts a business in draft with the default booking rule and settings
func (s *PlatformServiceImpl) CreateBusiness(ctx context.Context, actor *domain.User, req dto.CreateBusinessRequest) (*dto.BusinessDTO, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Slug = strings.TrimSpace(req.Slug)
	if req.Name == "" {
		return nil, badRequest("NAME_IS_REQUIRED")
	}
	plan, err := s.plans.GetByID(ctx, req.PlanID)
	if err != nil {
		return nil, orBadRequest(err, "PLAN_NOT_FOUND")
	}

	slug := req.Slug
	if slug != "" {
		if !validSlug(slug) {
			return nil, badRequest("INVALID_SLUG")
		}
		taken, err := s.businesses.SlugTaken(ctx, slug, 0)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, conflict("SLUG_TAKEN")
		}
	} else if slug, err = s.uniqueSlug(ctx, slugify(req.Name, "business")); err != nil {
		return nil, err
	}

	b := domain.Business{
		Name: req.Name, Slug: slug, PlanID: plan.ID,
		Timezone: "Asia/Bangkok", Currency: "LAK", Language: "en", BookingMode: domain.BookingModeAccount,
	}
	if err := s.businesses.CreateWithDefaults(ctx, &b, domain.DefaultBookingRule(0)); err != nil {
		return nil, err
	}
	b.Plan = *plan
	d := toBusinessDTO(b)
	return &d, nil
}

func (s *PlatformServiceImpl) ListBusinesses(ctx context.Context, q dto.ListQuery) (*dto.PagedResult[dto.BusinessDTO], error) {
	lq, page, limit := newListQuery(q)
	switch q.Status {
	case "":
	case domain.StatusDraft:
		lq.Filters["published"], lq.Filters["suspended"] = false, false
	case domain.StatusPublished:
		lq.Filters["published"], lq.Filters["suspended"] = true, false
	case domain.StatusSuspended:
		lq.Filters["suspended"] = true
	default:
		return nil, badRequest("INVALID_STATUS")
	}
	items, total, err := s.businesses.List(ctx, lq)
	if err != nil {
		return nil, err
	}
	return pageOf(items, total, page, limit, toBusinessDTO), nil
}

func (s *PlatformServiceImpl) GetBusiness(ctx context.Context, id uint) (*dto.BusinessDTO, error) {
	b, err := s.businesses.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	d := toBusinessDTO(*b)
	return &d, nil
}

func validOverrides(o domain.LimitOverrides) bool {
	for _, v := range []*int{o.MaxSports, o.MaxResources, o.MaxCoaches, o.MaxClassSessionsPerMonth, o.MaxStorageMB, o.MaxImages} {
		if v != nil && *v < 0 {
			return false
		}
	}
	return o.MaxMembers == nil || *o.MaxMembers >= 1
}

// UpdateBusiness changes the name, plan and per-business limit overrides
func (s *PlatformServiceImpl) UpdateBusiness(ctx context.Context, id uint, req dto.UpdateBusinessRequest) (*dto.BusinessDTO, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, badRequest("NAME_IS_REQUIRED")
	}
	if !validOverrides(req.Overrides) {
		return nil, badRequest("INVALID_LIMITS")
	}
	b, err := s.businesses.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	plan, err := s.plans.GetByID(ctx, req.PlanID)
	if err != nil {
		return nil, orBadRequest(err, "PLAN_NOT_FOUND")
	}
	b.Name, b.PlanID, b.Plan, b.Overrides = req.Name, plan.ID, *plan, req.Overrides
	if err := s.businesses.Update(ctx, b); err != nil {
		return nil, err
	}
	d := toBusinessDTO(*b)
	return &d, nil
}

// SetSuspended switches the public page and member access off (or back on); data is kept
func (s *PlatformServiceImpl) SetSuspended(ctx context.Context, id uint, suspended bool) (*dto.BusinessDTO, error) {
	b, err := s.businesses.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	b.Suspended = suspended
	if err := s.businesses.Update(ctx, b); err != nil {
		return nil, err
	}
	d := toBusinessDTO(*b)
	return &d, nil
}

// InviteOwner invites the business's first owner; later owners come from TransferOwnership
func (s *PlatformServiceImpl) InviteOwner(ctx context.Context, actor *domain.User, businessID uint, email string) (*dto.InvitationDTO, error) {
	b, err := s.businesses.GetByID(ctx, businessID)
	if err != nil {
		return nil, err
	}
	hasOwner, err := s.memberships.HasOwner(ctx, businessID)
	if err != nil {
		return nil, err
	}
	if hasOwner {
		return nil, conflict("BUSINESS_ALREADY_HAS_OWNER")
	}
	pending, err := s.invitations.PendingOwnerInvite(ctx, businessID, s.invites.now())
	if err != nil {
		return nil, err
	}
	if pending != nil {
		return nil, conflict("OWNER_INVITATION_ALREADY_PENDING")
	}
	return s.invites.create(ctx, b, actor, email, domain.MemberOwner)
}

// ---------- users ----------

func (s *PlatformServiceImpl) ListUsers(ctx context.Context, q dto.ListQuery) (*dto.PagedResult[dto.UserDTO], error) {
	lq, page, limit := newListQuery(q)
	users, total, err := s.users.List(ctx, lq)
	if err != nil {
		return nil, err
	}
	return pageOf(users, total, page, limit, func(u domain.User) dto.UserDTO { return toUserDTO(&u) }), nil
}

// SetUserStatus disables or enables any account on the platform; it takes effect on the next request
func (s *PlatformServiceImpl) SetUserStatus(ctx context.Context, actorID, targetID uint, status bool) error {
	if actorID == targetID && !status {
		return badRequest("CANNOT_DISABLE_YOURSELF")
	}
	return s.users.SetStatus(ctx, targetID, status)
}
