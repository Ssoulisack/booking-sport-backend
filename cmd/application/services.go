package application

import (
	"backend/internal/core/port/input"
	"backend/internal/core/services"
)

// AppServices holds all application services
type AppServices struct {
	user       input.UserService
	auth       input.AuthService
	catalog    input.CatalogService
	schedule   input.ScheduleService
	rules      input.RuleService
	media      input.MediaService
	adminUsers input.AdminUserService
}

// setupServices initializes all core business logic services
func SetupServices(repos *AppRepositories) *AppServices {
	userService := services.NewUserService(repos.user, repos.redis, repos.minio)
	authService := services.NewAuthService(repos.user, repos.redis)
	return &AppServices{
		user:    userService,
		auth:    authService,
		catalog: services.NewCatalogService(repos.sports, repos.resources, repos.coaches, repos.storage),
		schedule: services.NewScheduleService(
			repos.openingHours, repos.closures, repos.prices, repos.classSessions,
			repos.sports, repos.resources, repos.coaches,
		),
		rules:      services.NewRuleService(repos.rules, repos.sports, repos.resources),
		media:      services.NewMediaService(repos.storage),
		adminUsers: services.NewAdminUserService(repos.user),
	}
}
