package router

import (
	handler "backend/internal/adapters/primary/http/handler"
	"backend/internal/adapters/primary/http/middleware"
	"time"

	"github.com/gofiber/fiber/v2"
)

// SetupPlatformRoutes registers the Super Admin routes
func SetupPlatformRoutes(router fiber.Router, auth *middleware.Auth, platform handler.PlatformHandler) {
	p := router.Group("/platform", auth.RequireUser, auth.RequireSuperAdmin)

	p.Get("/plans", platform.ListPlans)
	p.Put("/plans/:id", platform.UpdatePlan)

	p.Post("/businesses", platform.CreateBusiness)
	p.Get("/businesses", platform.ListBusinesses)
	p.Get("/businesses/:id", platform.GetBusiness)
	p.Put("/businesses/:id", platform.UpdateBusiness)
	p.Post("/businesses/:id/suspend", platform.Suspend)
	p.Post("/businesses/:id/reactivate", platform.Reactivate)
	p.Post("/businesses/:id/owner-invitation", platform.InviteOwner)

	p.Get("/users", platform.ListUsers)
	p.Patch("/users/:id/status", platform.SetUserStatus)
	p.Post("/users/:id/password", platform.SetUserPassword)
}

// SetupPublicRoutes registers routes anyone can call: invite links and published booking pages
func SetupPublicRoutes(
	router fiber.Router,
	auth *middleware.Auth,
	invitations handler.InvitationHandler,
	public handler.PublicHandler,
) {
	router.Get("/invitations/:token", invitations.Get)
	router.Post("/invitations/:token/accept", middleware.RateLimit(10, 15*time.Minute), auth.OptionalUser, invitations.Accept)

	pub := router.Group("/public/:slug")
	pub.Get("/", public.Business)
	pub.Get("/sports", public.Sports)
	pub.Get("/resources", public.Resources)
	pub.Get("/coaches", public.Coaches)
	pub.Get("/class-sessions", public.ClassSessions)
}
