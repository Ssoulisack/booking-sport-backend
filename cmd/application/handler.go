package application

import (
	"backend/internal/adapters/primary/http/handler"
	"backend/internal/adapters/primary/http/middleware"
	"backend/internal/core/mappers"
)

// handler.AuthHandler

type AppHandler struct {
	authMiddleware *middleware.Auth
	auth           handler.AuthHandler
	user           handler.UserHandler
	platform       handler.PlatformHandler
	business       handler.BusinessHandler
	invitations    handler.InvitationHandler
	public         handler.PublicHandler
	catalog        handler.CatalogHandler
	schedule       handler.ScheduleHandler
	rules          handler.RuleHandler
	media          handler.MediaHandler
	booking        handler.BookingHandler
	desk           handler.DeskHandler
	reports        handler.ReportHandler
}

func SetupHandlers(svc *AppServices) *AppHandler {
	classicMapper := mappers.NewClassicMapper[any, any]()
	mapperFactory := mappers.NewMapperFactory(classicMapper)
	return &AppHandler{
		authMiddleware: middleware.NewAuth(svc.access),
		auth:           handler.NewAuthHandler(svc.auth, mapperFactory),
		user:           handler.NewUserHandler(svc.user, mapperFactory),
		platform:       handler.NewPlatformHandler(svc.platform),
		business:       handler.NewBusinessHandler(svc.business),
		invitations:    handler.NewInvitationHandler(svc.invitations),
		public:         handler.NewPublicHandler(svc.public),
		catalog:        handler.NewCatalogHandler(svc.catalog),
		schedule:       handler.NewScheduleHandler(svc.schedule),
		rules:          handler.NewRuleHandler(svc.rules),
		media:          handler.NewMediaHandler(svc.media),
		booking:        handler.NewBookingHandler(svc.booking),
		desk:           handler.NewDeskHandler(svc.desk),
		reports:        handler.NewReportHandler(svc.reports),
	}
}
