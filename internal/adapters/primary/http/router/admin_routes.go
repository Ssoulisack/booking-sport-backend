package router

import (
	handler "backend/internal/adapters/primary/http/handler"
	"backend/pkg/rbac"

	"github.com/gofiber/fiber/v2"
)

// SetupAdminRoutes registers the back-office routes. Every route needs a staff or admin token;
// user management needs an admin.
func SetupAdminRoutes(
	router fiber.Router,
	catalog handler.CatalogHandler,
	schedule handler.ScheduleHandler,
	rules handler.RuleHandler,
	media handler.MediaHandler,
	users handler.AdminUserHandler,
) {
	admin := router.Group("/admin", rbac.StaffUp)

	sports := admin.Group("/sports")
	sports.Post("/", catalog.CreateSport)
	sports.Get("/", catalog.ListSports)
	sports.Get("/:id", catalog.GetSport)
	sports.Put("/:id", catalog.UpdateSport)
	sports.Delete("/:id", catalog.DeleteSport)

	resources := admin.Group("/resources")
	resources.Post("/", catalog.CreateResource)
	resources.Get("/", catalog.ListResources)
	resources.Get("/:id", catalog.GetResource)
	resources.Put("/:id", catalog.UpdateResource)
	resources.Delete("/:id", catalog.DeleteResource)

	coaches := admin.Group("/coaches")
	coaches.Post("/", catalog.CreateCoach)
	coaches.Get("/", catalog.ListCoaches)
	coaches.Get("/:id", catalog.GetCoach)
	coaches.Put("/:id", catalog.UpdateCoach)
	coaches.Delete("/:id", catalog.DeleteCoach)

	// ownerType is "resource" or "coach"
	admin.Get("/opening-hours/:ownerType/:ownerId", schedule.GetOpeningHours)
	admin.Put("/opening-hours/:ownerType/:ownerId", schedule.SetOpeningHours)

	admin.Get("/prices/:ownerType/:ownerId", schedule.GetPrice)
	admin.Put("/prices/:ownerType/:ownerId", schedule.SetPrice)
	admin.Delete("/prices/:ownerType/:ownerId", schedule.DeletePrice)

	closures := admin.Group("/closures")
	closures.Post("/", schedule.CreateClosure)
	closures.Get("/", schedule.ListClosures)
	closures.Get("/:id", schedule.GetClosure)
	closures.Put("/:id", schedule.UpdateClosure)
	closures.Delete("/:id", schedule.DeleteClosure)

	classes := admin.Group("/class-sessions")
	classes.Post("/", schedule.CreateClassSession)
	classes.Get("/", schedule.ListClassSessions)
	classes.Get("/:id", schedule.GetClassSession)
	classes.Put("/:id", schedule.UpdateClassSession)
	classes.Delete("/:id", schedule.DeleteClassSession)

	// scopeType is "global" (id 0), "sport" or "resource"
	admin.Get("/rules", rules.ListRules)
	admin.Get("/rules/effective", rules.Effective)
	admin.Get("/rules/:scopeType/:scopeId", rules.GetRule)
	admin.Put("/rules/:scopeType/:scopeId", rules.SetRule)
	admin.Delete("/rules/:scopeType/:scopeId", rules.DeleteRule)

	admin.Post("/uploads", media.Upload)
	admin.Delete("/uploads", media.Delete)

	adminOnly := admin.Group("/users", rbac.AdminOnly)
	adminOnly.Post("/", users.CreateStaff)
	adminOnly.Get("/", users.ListUsers)
	adminOnly.Patch("/:id/status", users.SetUserStatus)
}
