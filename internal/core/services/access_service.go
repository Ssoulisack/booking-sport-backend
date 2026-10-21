package services

import (
	"backend/internal/core/domain"
	"backend/internal/core/port/input"
	"backend/internal/core/port/output"
	"backend/pkg/response"
	"context"
	"net/http"
	"time"
)

// RoleSupport is returned instead of a member role when a Super Admin views a business
const RoleSupport = "support"

// a Super Admin's repeated requests to one business within this window count as one visit
const supportVisitWindow = 30 * time.Minute

type AccessServiceImpl struct {
	users       output.UserRepository
	memberships output.MembershipRepository
	businesses  output.BusinessRepository
	visits      output.SupportVisitRepository
	now         func() time.Time
}

func NewAccessService(
	users output.UserRepository,
	memberships output.MembershipRepository,
	businesses output.BusinessRepository,
	visits output.SupportVisitRepository,
) input.AccessService {
	return &AccessServiceImpl{users: users, memberships: memberships, businesses: businesses, visits: visits, now: time.Now}
}

func (s *AccessServiceImpl) Authenticate(ctx context.Context, userID uint, issuedAt int64) (*domain.User, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return nil, response.NewError(http.StatusUnauthorized, response.ErrMsgUnauthorized)
		}
		return nil, err
	}
	if err := checkUsable(user, issuedAt); err != nil {
		return nil, err
	}
	return user, nil
}

// BusinessRole is checked on every request, so removing or disabling a member, or suspending
// the business, takes effect immediately. Non-members get 404 so they cannot probe which ids exist.
func (s *AccessServiceImpl) BusinessRole(ctx context.Context, user *domain.User, businessID uint, minRole string, readOnly bool) (string, error) {
	hidden := notFound("BUSINESS_NOT_FOUND")
	b, err := s.businesses.GetByID(ctx, businessID)
	if err != nil {
		if isNotFound(err) {
			return "", hidden
		}
		return "", err
	}

	m, err := s.memberships.Get(ctx, businessID, user.ID)
	if err != nil {
		return "", err
	}
	if m != nil {
		switch {
		case b.Suspended:
			return "", forbidden("BUSINESS_SUSPENDED")
		case !m.Active:
			return "", forbidden("MEMBERSHIP_DISABLED")
		case domain.MemberRank(m.Role) < domain.MemberRank(minRole):
			return "", forbidden("ROLE_NOT_ALLOWED")
		}
		return m.Role, nil
	}

	if user.IsSuperAdmin() {
		if !readOnly {
			return "", forbidden("SUPPORT_VIEW_IS_READ_ONLY")
		}
		if err := s.visits.Record(ctx, user.ID, businessID, s.now(), supportVisitWindow); err != nil {
			return "", err
		}
		return RoleSupport, nil
	}
	return "", hidden
}
