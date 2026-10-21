package application

import (
	"backend/internal/core/port/input"
	"backend/internal/core/services"
)

// AppServices holds all application services
type AppServices struct {
	user        input.UserService
	auth        input.AuthService
	access      input.AccessService
	platform    input.PlatformService
	business    input.BusinessService
	invitations input.InvitationService
	public      input.PublicService
	catalog     input.CatalogService
	schedule    input.ScheduleService
	rules       input.RuleService
	media       input.MediaService
	booking     input.BookingService
}

// setupServices initializes all core business logic services
func SetupServices(repos *AppRepositories) *AppServices {
	r := repos
	limits := services.NewLimits(r.businesses, r.sports, r.resources, r.coaches, r.classSessions, r.memberships, r.invitations, r.media)
	rules := services.NewRuleService(r.rules, r.sports, r.resources)
	return &AppServices{
		user:        services.NewUserService(r.user, r.redis, r.minio),
		auth:        services.NewAuthService(r.user, r.redis, r.memberships, r.passwordResets, r.mailer, r.app.FrontendURL),
		access:      services.NewAccessService(r.user, r.memberships, r.businesses, r.supportVisits),
		platform:    services.NewPlatformService(r.plans, r.businesses, r.memberships, r.invitations, r.user, limits, r.mailer, r.app.FrontendURL),
		invitations: services.NewInvitationService(r.invitations, r.memberships, r.businesses, r.user),
		public:      services.NewPublicService(r.businesses, r.sports, r.resources, r.coaches, r.classSessions, r.bookings, r.storage),
		business: services.NewBusinessService(
			r.businesses, r.memberships, r.invitations, r.user, r.sports, r.resources, r.openingHours, r.prices,
			r.supportVisits, r.storage, r.media, limits, r.mailer, r.app.FrontendURL,
		),
		catalog: services.NewCatalogService(r.sports, r.resources, r.coaches, r.classSessions, r.bookings, r.storage, r.media, limits),
		schedule: services.NewScheduleService(
			r.openingHours, r.closures, r.prices, r.classSessions, r.sports, r.resources, r.coaches, r.bookings, limits,
		),
		rules: rules,
		media: services.NewMediaService(r.storage, r.media, limits),
		booking: services.NewBookingService(
			r.businesses, r.sports, r.resources, r.coaches, r.openingHours, r.closures, r.prices, r.classSessions,
			r.bookings, rules, r.mailer, r.app.FrontendURL,
		),
	}
}
