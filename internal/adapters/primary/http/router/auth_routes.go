package router

import (
	handler "backend/internal/adapters/primary/http/handler"
	"backend/internal/adapters/primary/http/middleware"
	"backend/pkg/jwt"

	"github.com/gofiber/fiber/v2"
)

// SetupAuthRoutes registers all authentication related routes
func SetupAuthRoutes(router fiber.Router, authHandler handler.AuthHandler, auth *middleware.Auth) {
	a := router.Group("/auth")

	// Public - routes that don't require authentication
	a.Post("/register", authHandler.Register)
	a.Post("/login", authHandler.Login)
	a.Post("/forgot-password", authHandler.ForgotPassword)
	a.Post("/reset-password", authHandler.ResetPassword)

	// Private - routes that require authentication
	a.Post("/refresh", jwt.AccessRefreshToken, authHandler.Refresh)
	a.Get("/me", auth.RequireUser, authHandler.Me)
	router.Get("/me/businesses", auth.RequireUser, authHandler.MyBusinesses)
}
