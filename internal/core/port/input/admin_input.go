package input

import (
	"backend/internal/core/domain"
	"backend/internal/core/dto"
	"context"
	"io"
)

// CatalogService manages sports, resources and coaches
type CatalogService interface {
	CreateSport(ctx context.Context, req dto.SportRequest) (*dto.SportDTO, error)
	GetSport(ctx context.Context, id uint) (*dto.SportDTO, error)
	ListSports(ctx context.Context, q dto.ListQuery) (*dto.PagedResult[dto.SportDTO], error)
	UpdateSport(ctx context.Context, id uint, req dto.SportRequest) (*dto.SportDTO, error)
	DeleteSport(ctx context.Context, id uint) error

	CreateResource(ctx context.Context, req dto.ResourceRequest) (*dto.ResourceDTO, error)
	GetResource(ctx context.Context, id uint) (*dto.ResourceDTO, error)
	ListResources(ctx context.Context, q dto.ListQuery) (*dto.PagedResult[dto.ResourceDTO], error)
	UpdateResource(ctx context.Context, id uint, req dto.ResourceRequest) (*dto.ResourceDTO, error)
	DeleteResource(ctx context.Context, id uint) error

	CreateCoach(ctx context.Context, req dto.CoachRequest) (*dto.CoachDTO, error)
	GetCoach(ctx context.Context, id uint) (*dto.CoachDTO, error)
	ListCoaches(ctx context.Context, q dto.ListQuery) (*dto.PagedResult[dto.CoachDTO], error)
	UpdateCoach(ctx context.Context, id uint, req dto.CoachRequest) (*dto.CoachDTO, error)
	DeleteCoach(ctx context.Context, id uint) error
}

// ScheduleService manages opening hours, closures, prices and class sessions
type ScheduleService interface {
	GetOpeningHours(ctx context.Context, ownerType string, ownerID uint) ([]dto.OpeningHourDTO, error)
	SetOpeningHours(ctx context.Context, ownerType string, ownerID uint, req dto.OpeningHoursRequest) ([]dto.OpeningHourDTO, error)

	CreateClosure(ctx context.Context, req dto.ClosureRequest) (*dto.ClosureDTO, error)
	GetClosure(ctx context.Context, id uint) (*dto.ClosureDTO, error)
	ListClosures(ctx context.Context, q dto.ListQuery) (*dto.PagedResult[dto.ClosureDTO], error)
	UpdateClosure(ctx context.Context, id uint, req dto.ClosureRequest) (*dto.ClosureDTO, error)
	DeleteClosure(ctx context.Context, id uint) error

	GetPrice(ctx context.Context, ownerType string, ownerID uint) (*dto.PriceDTO, error)
	SetPrice(ctx context.Context, ownerType string, ownerID uint, req dto.PriceRequest) (*dto.PriceDTO, error)
	DeletePrice(ctx context.Context, ownerType string, ownerID uint) error

	CreateClassSession(ctx context.Context, req dto.ClassSessionRequest) (*dto.ClassSessionDTO, error)
	GetClassSession(ctx context.Context, id uint) (*dto.ClassSessionDTO, error)
	ListClassSessions(ctx context.Context, q dto.ListQuery) (*dto.PagedResult[dto.ClassSessionDTO], error)
	UpdateClassSession(ctx context.Context, id uint, req dto.ClassSessionRequest) (*dto.ClassSessionDTO, error)
	DeleteClassSession(ctx context.Context, id uint) error
}

// RuleService manages booking rules and resolves the rule that applies to a sport or resource
type RuleService interface {
	GetRule(ctx context.Context, scopeType string, scopeID uint) (*dto.RuleDTO, error)
	ListRules(ctx context.Context) ([]dto.RuleDTO, error)
	SetRule(ctx context.Context, scopeType string, scopeID uint, req dto.RuleRequest) (*dto.RuleDTO, error)
	DeleteRule(ctx context.Context, scopeType string, scopeID uint) error
	// Effective merges resource -> sport -> global. resourceID implies its sport.
	Effective(ctx context.Context, sportID, resourceID uint) (*domain.EffectiveRule, error)
}

// MediaService validates and stores uploaded images
type MediaService interface {
	Upload(ctx context.Context, folder string, file io.Reader) (*dto.MediaDTO, error)
	Delete(ctx context.Context, key string) error
}

// AdminUserService lets an admin manage staff accounts
type AdminUserService interface {
	CreateStaff(ctx context.Context, req dto.CreateStaffRequest) (*dto.UserDTO, error)
	ListUsers(ctx context.Context, q dto.ListQuery) (*dto.PagedResult[dto.UserDTO], error)
	SetUserStatus(ctx context.Context, actorID, targetID uint, status bool) error
}
