package application

import (
	"backend/internal/core/port/input"
	"backend/internal/core/services"
)

// AppServices holds all application services
type AppServices struct {
	user input.UserService
	auth input.AuthService
}

// setupServices initializes all core business logic services
func SetupServices(repos *AppRepositories) *AppServices {
	userService := services.NewUserService(repos.user, repos.redis, repos.minio)
	authService := services.NewAuthService(repos.user, repos.redis)
	return &AppServices{
		user: userService,
		auth: authService,
	}
}
