package output

import (
	"backend/internal/core/domain"
	"context"
	"io"
	"time"
)

type SportRepository = ScopedRepository[domain.Sport]
type ResourceRepository = ScopedRepository[domain.Resource]

// ClosureRepository also finds the closures (of any owner, including the venue) overlapping a period
type ClosureRepository interface {
	ScopedRepository[domain.Closure]
	Overlapping(ctx context.Context, businessID uint, from, to time.Time) ([]domain.Closure, error)
}

// CoachRepository loads coaches with their sports
type CoachRepository interface {
	ScopedRepository[domain.Coach]
	// SetSports replaces the coach's sports; every sport must belong to the same business
	SetSports(ctx context.Context, businessID, coachID uint, sportIDs []uint) error
}

type ClassSessionRepository interface {
	ScopedRepository[domain.ClassSession]
	CountStartingBetween(ctx context.Context, businessID uint, from, to time.Time) (int64, error)
	// ListFiltered lists sessions starting in the filter's range, soonest first
	ListFiltered(ctx context.Context, businessID uint, f ClassFilter, q ListQuery) ([]domain.ClassSession, int64, error)
	// CreateMany stores every session (and its court / coach hold) in one transaction: all or nothing
	CreateMany(ctx context.Context, sessions []domain.ClassSession) error
	// ListBetween lists sessions (active or not) overlapping [from, to)
	ListBetween(ctx context.Context, businessID uint, from, to time.Time) ([]domain.ClassSession, error)
}

// ClassFilter narrows a class list; zero values do not filter
type ClassFilter struct {
	From, To   *time.Time // sessions starting in [From, To)
	SportID    uint
	ActiveOnly bool
	Active     *bool
}

// OpeningHourRepository replaces the whole weekly schedule of an owner in one transaction
type OpeningHourRepository interface {
	ListByOwner(ctx context.Context, businessID uint, ownerType string, ownerID uint) ([]domain.OpeningHour, error)
	ReplaceForOwner(ctx context.Context, businessID uint, ownerType string, ownerID uint, hours []domain.OpeningHour) error
}

type PriceRepository interface {
	GetByOwner(ctx context.Context, businessID uint, ownerType string, ownerID uint) (*domain.Price, error)
	Upsert(ctx context.Context, p *domain.Price) error
	DeleteByOwner(ctx context.Context, businessID uint, ownerType string, ownerID uint) error
}

type RuleRepository interface {
	// GetByScope returns (nil, nil) when no rule exists for the scope
	GetByScope(ctx context.Context, businessID uint, scopeType string, scopeID uint) (*domain.BookingRule, error)
	Upsert(ctx context.Context, r *domain.BookingRule) error
	List(ctx context.Context, businessID uint) ([]domain.BookingRule, error)
	DeleteByScope(ctx context.Context, businessID uint, scopeType string, scopeID uint) error
}

// FileStorage stores uploaded files. The database keeps only the object key.
type FileStorage interface {
	Upload(ctx context.Context, key, contentType string, body io.Reader) error
	Delete(ctx context.Context, key string) error
	URL(key string) string
}

// MediaRepository tracks uploaded files per business
type MediaRepository interface {
	Create(ctx context.Context, m *domain.Media) error
	// Get returns (nil, nil) when the key does not belong to the business
	Get(ctx context.Context, businessID uint, key string) (*domain.Media, error)
	Delete(ctx context.Context, businessID uint, key string) error
	Usage(ctx context.Context, businessID uint) (images int64, bytes int64, err error)
}
