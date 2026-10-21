package output

import (
	"backend/internal/core/domain"
	"context"
	"io"
)

type SportRepository = CRUDRepository[domain.Sport]
type ResourceRepository = CRUDRepository[domain.Resource]
type ClosureRepository = CRUDRepository[domain.Closure]
type ClassSessionRepository = CRUDRepository[domain.ClassSession]

// CoachRepository loads coaches with their sports
type CoachRepository interface {
	CRUDRepository[domain.Coach]
	SetSports(ctx context.Context, coachID uint, sportIDs []uint) error
}

// OpeningHourRepository replaces the whole weekly schedule of an owner in one transaction
type OpeningHourRepository interface {
	ListByOwner(ctx context.Context, ownerType string, ownerID uint) ([]domain.OpeningHour, error)
	ReplaceForOwner(ctx context.Context, ownerType string, ownerID uint, hours []domain.OpeningHour) error
}

type PriceRepository interface {
	GetByOwner(ctx context.Context, ownerType string, ownerID uint) (*domain.Price, error)
	Upsert(ctx context.Context, p *domain.Price) error
	DeleteByOwner(ctx context.Context, ownerType string, ownerID uint) error
}

type RuleRepository interface {
	// GetByScope returns (nil, nil) when no rule exists for the scope
	GetByScope(ctx context.Context, scopeType string, scopeID uint) (*domain.BookingRule, error)
	Upsert(ctx context.Context, r *domain.BookingRule) error
	List(ctx context.Context) ([]domain.BookingRule, error)
	DeleteByScope(ctx context.Context, scopeType string, scopeID uint) error
}

// FileStorage stores uploaded files. The database keeps only the object key.
type FileStorage interface {
	Upload(ctx context.Context, key, contentType string, body io.Reader) error
	Delete(ctx context.Context, key string) error
	URL(key string) string
}
