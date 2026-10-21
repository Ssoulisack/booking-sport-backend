package output

import (
	"backend/internal/core/domain"
	"context"
	"time"
)

type PlanRepository interface {
	List(ctx context.Context) ([]domain.Plan, error)
	GetByID(ctx context.Context, id uint) (*domain.Plan, error)
	Update(ctx context.Context, p *domain.Plan) error
}

type BusinessRepository interface {
	// CreateWithDefaults stores the business and its default booking rule in one transaction
	CreateWithDefaults(ctx context.Context, b *domain.Business, rule domain.BookingRule) error
	// GetByID and GetBySlug load the plan too
	GetByID(ctx context.Context, id uint) (*domain.Business, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Business, error)
	// List supports the filters "published" and "suspended" and a keyword on name / slug
	List(ctx context.Context, q ListQuery) ([]domain.Business, int64, error)
	Update(ctx context.Context, b *domain.Business) error
	SlugTaken(ctx context.Context, slug string, exceptID uint) (bool, error)
}

type MembershipRepository interface {
	Create(ctx context.Context, m *domain.Membership) error
	// Get returns (nil, nil) when the user is not a member
	Get(ctx context.Context, businessID, userID uint) (*domain.Membership, error)
	ListByBusiness(ctx context.Context, businessID uint) ([]domain.Membership, error)
	ListByUser(ctx context.Context, userID uint) ([]domain.Membership, error)
	Update(ctx context.Context, m *domain.Membership) error
	Delete(ctx context.Context, businessID, userID uint) error
	Count(ctx context.Context, businessID uint) (int64, error)
	HasOwner(ctx context.Context, businessID uint) (bool, error)
	// TransferOwnership makes to the owner and from a manager, in one transaction
	TransferOwnership(ctx context.Context, businessID, fromUserID, toUserID uint) error
}

type InvitationRepository interface {
	Create(ctx context.Context, inv *domain.Invitation) error
	// GetByTokenHash returns (nil, nil) when no invitation matches
	GetByTokenHash(ctx context.Context, hash string) (*domain.Invitation, error)
	GetByID(ctx context.Context, businessID, id uint) (*domain.Invitation, error)
	// Pending lists invitations not yet accepted and not expired at now
	ListPending(ctx context.Context, businessID uint, now time.Time) ([]domain.Invitation, error)
	CountPending(ctx context.Context, businessID uint, now time.Time) (int64, error)
	PendingForEmail(ctx context.Context, businessID uint, email string, now time.Time) (*domain.Invitation, error)
	PendingOwnerInvite(ctx context.Context, businessID uint, now time.Time) (*domain.Invitation, error)
	Update(ctx context.Context, inv *domain.Invitation) error
	Delete(ctx context.Context, businessID, id uint) error
}

type PasswordResetRepository interface {
	Create(ctx context.Context, r *domain.PasswordReset) error
	// GetByTokenHash returns (nil, nil) when no reset matches
	GetByTokenHash(ctx context.Context, hash string) (*domain.PasswordReset, error)
	// UseAll marks every unused reset of the user as used
	UseAll(ctx context.Context, userID uint, at time.Time) error
}

type SupportVisitRepository interface {
	// Record stores a visit unless the same admin visited the same business within window
	Record(ctx context.Context, userID, businessID uint, at time.Time, window time.Duration) error
	List(ctx context.Context, businessID uint, q ListQuery) ([]domain.SupportVisit, int64, error)
}

// Mailer sends the platform's transactional emails
type Mailer interface {
	SendInvitation(to, businessName, inviterName, role, acceptURL string) error
	SendPasswordReset(to, name, resetURL string) error
}
