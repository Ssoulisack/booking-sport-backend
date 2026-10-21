package application

import (
	"backend/internal/adapters/primary/http/handler"
	"backend/internal/core/mappers"
)

// handler.AuthHandler

type AppHandler struct {
	auth       handler.AuthHandler
	user       handler.UserHandler
	catalog    handler.CatalogHandler
	schedule   handler.ScheduleHandler
	rules      handler.RuleHandler
	media      handler.MediaHandler
	adminUsers handler.AdminUserHandler
}

func SetupHandlers(svc *AppServices) *AppHandler {
	classicMapper := mappers.NewClassicMapper[any, any]()
	mapperFactory := mappers.NewMapperFactory(classicMapper)
	return &AppHandler{
		auth:       handler.NewAuthHandler(svc.auth, mapperFactory),
		user:       handler.NewUserHandler(svc.user, mapperFactory),
		catalog:    handler.NewCatalogHandler(svc.catalog),
		schedule:   handler.NewScheduleHandler(svc.schedule),
		rules:      handler.NewRuleHandler(svc.rules),
		media:      handler.NewMediaHandler(svc.media),
		adminUsers: handler.NewAdminUserHandler(svc.adminUsers),
	}
}
