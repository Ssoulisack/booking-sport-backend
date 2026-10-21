package router

import (
	handler "backend/internal/adapters/primary/http/handler"
	"backend/pkg/jwt"

	"github.com/gofiber/fiber/v2"
)

// SetupAuthRoutes registers all authentication related routes
func SetupAuthRoutes(router fiber.Router, authHandler handler.AuthHandler) {
	auth := router.Group("/auth")

	// Public - routes that don't require authentication
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)

	// Private - routes that require authentication
	auth.Post("/refresh", jwt.AccessRefreshToken, authHandler.Refresh)
	auth.Get("/me", jwt.AccessToken, authHandler.Me)
}
