package router

import (
	handler "backend/internal/adapters/primary/http/handler"
	"backend/pkg/jwt"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// RegisterAuthRoutes registers all authentication related routes
func SetupAuthRoutes(router fiber.Router, authHandler handler.AuthHandler) {
	// Public Router - routes that don't require authentication
	public := router.Group("/")
	fmt.Print(public)

	// Private Router - routes that require authentication
	private := router.Group("/auth", jwt.AccessToken)
	fmt.Print(private)
}
