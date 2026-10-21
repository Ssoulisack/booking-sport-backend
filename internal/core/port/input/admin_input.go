package input

import (
	"backend/internal/core/domain"
	"backend/internal/core/dto"
	"context"
	"io"
)

// Every business-scoped method takes the business it acts on as businessID.
// The HTTP layer has already checked that the caller may act on that business.

// CatalogService manages sports, resources and coaches of one business
type CatalogService interface {
	CreateSport(ctx context.Context, businessID uint, req dto.SportRequest) (*dto.SportDTO, error)
	GetSport(ctx context.Context, businessID, id uint) (*dto.SportDTO, error)
	ListSports(ctx context.Context, businessID uint, q dto.ListQuery) (*dto.PagedResult[dto.SportDTO], error)
	UpdateSport(ctx context.Context, businessID, id uint, req dto.SportRequest) (*dto.SportDTO, error)
	DeleteSport(ctx context.Context, businessID, id uint) error

	CreateResource(ctx context.Context, businessID uint, req dto.ResourceRequest) (*dto.ResourceDTO, error)
	GetResource(ctx context.Context, businessID, id uint) (*dto.ResourceDTO, error)
	ListResources(ctx context.Context, businessID uint, q dto.ListQuery) (*dto.PagedResult[dto.ResourceDTO], error)
	UpdateResource(ctx context.Context, businessID, id uint, req dto.ResourceRequest) (*dto.ResourceDTO, error)
	DeleteResource(ctx context.Context, businessID, id uint) error

	CreateCoach(ctx context.Context, businessID uint, req dto.CoachRequest) (*dto.CoachDTO, error)
	GetCoach(ctx context.Context, businessID, id uint) (*dto.CoachDTO, error)
	ListCoaches(ctx context.Context, businessID uint, q dto.ListQuery) (*dto.PagedResult[dto.CoachDTO], error)
	UpdateCoach(ctx context.Context, businessID, id uint, req dto.CoachRequest) (*dto.CoachDTO, error)
	DeleteCoach(ctx context.Context, businessID, id uint) error
}

// ScheduleService manages opening hours, closures, prices and class sessions of one business
type ScheduleService interface {
	GetOpeningHours(ctx context.Context, businessID uint, ownerType string, ownerID uint) ([]dto.OpeningHourDTO, error)
	SetOpeningHours(ctx context.Context, businessID uint, ownerType string, ownerID uint, req dto.OpeningHoursRequest) ([]dto.OpeningHourDTO, error)

	CreateClosure(ctx context.Context, businessID uint, req dto.ClosureRequest) (*dto.ClosureDTO, error)
	GetClosure(ctx context.Context, businessID, id uint) (*dto.ClosureDTO, error)
	ListClosures(ctx context.Context, businessID uint, q dto.ListQuery) (*dto.PagedResult[dto.ClosureDTO], error)
	UpdateClosure(ctx context.Context, businessID, id uint, req dto.ClosureRequest) (*dto.ClosureDTO, error)
	DeleteClosure(ctx context.Context, businessID, id uint) error

	GetPrice(ctx context.Context, businessID uint, ownerType string, ownerID uint) (*dto.PriceDTO, error)
	SetPrice(ctx context.Context, businessID uint, ownerType string, ownerID uint, req dto.PriceRequest) (*dto.PriceDTO, error)
	DeletePrice(ctx context.Context, businessID uint, ownerType string, ownerID uint) error

	CreateClassSession(ctx context.Context, businessID uint, req dto.ClassSessionRequest) (*dto.ClassSessionDTO, error)
	GetClassSession(ctx context.Context, businessID, id uint) (*dto.ClassSessionDTO, error)
	ListClassSessions(ctx context.Context, businessID uint, q dto.ListQuery) (*dto.PagedResult[dto.ClassSessionDTO], error)
	UpdateClassSession(ctx context.Context, businessID, id uint, req dto.ClassSessionRequest) (*dto.ClassSessionDTO, error)
	DeleteClassSession(ctx context.Context, businessID, id uint) error
}

// RuleService manages booking rules and resolves the rule that applies to a sport or resource
type RuleService interface {
	GetRule(ctx context.Context, businessID uint, scopeType string, scopeID uint) (*dto.RuleDTO, error)
	ListRules(ctx context.Context, businessID uint) ([]dto.RuleDTO, error)
	SetRule(ctx context.Context, businessID uint, scopeType string, scopeID uint, req dto.RuleRequest) (*dto.RuleDTO, error)
	DeleteRule(ctx context.Context, businessID uint, scopeType string, scopeID uint) error
	// Effective merges resource -> sport -> default. resourceID implies its sport.
	Effective(ctx context.Context, businessID, sportID, resourceID uint) (*domain.EffectiveRule, error)
}

// MediaService validates and stores uploaded images within the business's storage limits
type MediaService interface {
	Upload(ctx context.Context, businessID uint, folder string, file io.Reader) (*dto.MediaDTO, error)
	Delete(ctx context.Context, businessID uint, key string) error
}

// BusinessService is what a business's own members manage: settings, publishing, members, invites
type BusinessService interface {
	GetSettings(ctx context.Context, businessID uint) (*dto.SettingsDTO, error)
	UpdateSettings(ctx context.Context, businessID uint, req dto.SettingsRequest) (*dto.SettingsDTO, error)
	Publish(ctx context.Context, businessID uint) (*dto.SettingsDTO, error)
	Unpublish(ctx context.Context, businessID uint) (*dto.SettingsDTO, error)
	Usage(ctx context.Context, businessID uint) (*dto.UsageDTO, error)

	ListMembers(ctx context.Context, businessID uint) ([]dto.MemberDTO, error)
	UpdateMember(ctx context.Context, businessID, userID uint, req dto.UpdateMemberRequest) (*dto.MemberDTO, error)
	RemoveMember(ctx context.Context, businessID, userID uint) error
	TransferOwnership(ctx context.Context, businessID, actorID, toUserID uint) error

	ListInvitations(ctx context.Context, businessID uint) ([]dto.InvitationDTO, error)
	Invite(ctx context.Context, businessID uint, actor *domain.User, req dto.InviteRequest) (*dto.InvitationDTO, error)
	CancelInvitation(ctx context.Context, businessID, id uint) error
	ResendInvitation(ctx context.Context, businessID uint, actor *domain.User, id uint) (*dto.InvitationDTO, error)

	ListSupportVisits(ctx context.Context, businessID uint, q dto.ListQuery) (*dto.PagedResult[dto.SupportVisitDTO], error)
}

// PlatformService is the Super Admin's view of the whole platform
type PlatformService interface {
	ListPlans(ctx context.Context) ([]dto.PlanDTO, error)
	UpdatePlan(ctx context.Context, id uint, req dto.PlanRequest) (*dto.PlanDTO, error)

	CreateBusiness(ctx context.Context, actor *domain.User, req dto.CreateBusinessRequest) (*dto.BusinessDTO, error)
	ListBusinesses(ctx context.Context, q dto.ListQuery) (*dto.PagedResult[dto.BusinessDTO], error)
	GetBusiness(ctx context.Context, id uint) (*dto.BusinessDTO, error)
	UpdateBusiness(ctx context.Context, id uint, req dto.UpdateBusinessRequest) (*dto.BusinessDTO, error)
	SetSuspended(ctx context.Context, id uint, suspended bool) (*dto.BusinessDTO, error)
	InviteOwner(ctx context.Context, actor *domain.User, businessID uint, email string) (*dto.InvitationDTO, error)

	ListUsers(ctx context.Context, q dto.ListQuery) (*dto.PagedResult[dto.UserDTO], error)
	SetUserStatus(ctx context.Context, actorID, targetID uint, status bool) error
}

// InvitationService serves the invite link: look at an invite, then accept it
type InvitationService interface {
	GetInvitation(ctx context.Context, token string) (*dto.InvitationInfoDTO, error)
	// Accept joins the business. actor is the logged-in user (nil if none). A new user
	// is created from req; an existing account must be logged in as the invited email.
	Accept(ctx context.Context, token string, actor *domain.User, req dto.AcceptInvitationRequest) (*dto.AuthResponseDTO, error)
}

// AccessService answers "who is calling and what may they do", for the HTTP middleware
type AccessService interface {
	// Authenticate loads the token's user and refuses disabled users and tokens issued before a password reset
	Authenticate(ctx context.Context, userID uint, issuedAt int64) (*domain.User, error)
	// BusinessRole returns the caller's role in the business if it ranks at least minRole.
	// A Super Admin gets read-only support access (readOnly must be true), which is recorded.
	BusinessRole(ctx context.Context, user *domain.User, businessID uint, minRole string, readOnly bool) (string, error)
}

// PublicService serves a published business's booking page data
type PublicService interface {
	Business(ctx context.Context, slug string) (*dto.PublicBusinessDTO, error)
	Sports(ctx context.Context, slug string) ([]dto.SportDTO, error)
	Resources(ctx context.Context, slug string, sportID uint) ([]dto.ResourceDTO, error)
	Coaches(ctx context.Context, slug string, sportID uint) ([]dto.PublicCoachDTO, error)
	ClassSessions(ctx context.Context, slug string, q dto.ListQuery) (*dto.PagedResult[dto.ClassSessionDTO], error)
}
