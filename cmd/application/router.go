package application

import (
	"backend/internal/adapters/primary/http/router"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app fiber.Router, handlers *AppHandler) {
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World! 👋")
	})
	// API group
	api := app.Group("/api/v1")
	router.SetupAuthRoutes(api, handlers.auth)
	router.SetupUserRoutes(api, handlers.user)
}
