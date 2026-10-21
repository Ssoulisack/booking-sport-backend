package router

import (
	handler "backend/internal/adapters/primary/http/handler"
	"backend/internal/adapters/primary/http/middleware"
	"backend/internal/core/domain"

	"github.com/gofiber/fiber/v2"
)

// SetupManageRoutes registers the business dashboard routes under /manage/businesses/:businessId.
// Each route names the lowest member role allowed: staff < manager < owner.
// A Super Admin who is not a member may call the GET routes (read-only support view).
func SetupManageRoutes(
	router fiber.Router,
	auth *middleware.Auth,
	catalog handler.CatalogHandler,
	schedule handler.ScheduleHandler,
	rules handler.RuleHandler,
	media handler.MediaHandler,
	business handler.BusinessHandler,
) {
	staff := auth.Business(domain.MemberStaff)
	manager := auth.Business(domain.MemberManager)
	owner := auth.Business(domain.MemberOwner)

	b := router.Group("/manage/businesses/:businessId", auth.RequireUser)

	// settings and publishing
	b.Get("/settings", staff, business.GetSettings)
	b.Put("/settings", owner, business.UpdateSettings)
	b.Post("/publish", owner, business.Publish)
	b.Post("/unpublish", owner, business.Unpublish)
	b.Get("/usage", manager, business.Usage)

	// members and invitations
	b.Get("/members", manager, business.ListMembers)
	b.Patch("/members/:userId", owner, business.UpdateMember)
	b.Delete("/members/:userId", owner, business.RemoveMember)
	b.Post("/transfer-ownership", owner, business.TransferOwnership)
	b.Get("/invitations", owner, business.ListInvitations)
	b.Post("/invitations", owner, business.Invite)
	b.Delete("/invitations/:id", owner, business.CancelInvitation)
	b.Post("/invitations/:id/resend", owner, business.ResendInvitation)
	b.Get("/support-visits", owner, business.ListSupportVisits)

	// catalog: everyone reads, managers change
	b.Get("/sports", staff, catalog.ListSports)
	b.Get("/sports/:id", staff, catalog.GetSport)
	b.Post("/sports", manager, catalog.CreateSport)
	b.Put("/sports/:id", manager, catalog.UpdateSport)
	b.Delete("/sports/:id", manager, catalog.DeleteSport)

	b.Get("/resources", staff, catalog.ListResources)
	b.Get("/resources/:id", staff, catalog.GetResource)
	b.Post("/resources", manager, catalog.CreateResource)
	b.Put("/resources/:id", manager, catalog.UpdateResource)
	b.Delete("/resources/:id", manager, catalog.DeleteResource)

	b.Get("/coaches", staff, catalog.ListCoaches)
	b.Get("/coaches/:id", staff, catalog.GetCoach)
	b.Post("/coaches", manager, catalog.CreateCoach)
	b.Put("/coaches/:id", manager, catalog.UpdateCoach)
	b.Delete("/coaches/:id", manager, catalog.DeleteCoach)

	// ownerType is "resource" or "coach"
	b.Get("/opening-hours/:ownerType/:ownerId", staff, schedule.GetOpeningHours)
	b.Put("/opening-hours/:ownerType/:ownerId", manager, schedule.SetOpeningHours)

	b.Get("/prices/:ownerType/:ownerId", staff, schedule.GetPrice)
	b.Put("/prices/:ownerType/:ownerId", manager, schedule.SetPrice)
	b.Delete("/prices/:ownerType/:ownerId", manager, schedule.DeletePrice)

	b.Get("/closures", staff, schedule.ListClosures)
	b.Get("/closures/:id", staff, schedule.GetClosure)
	b.Post("/closures", manager, schedule.CreateClosure)
	b.Put("/closures/:id", manager, schedule.UpdateClosure)
	b.Delete("/closures/:id", manager, schedule.DeleteClosure)

	b.Get("/class-sessions", staff, schedule.ListClassSessions)
	b.Get("/class-sessions/:id", staff, schedule.GetClassSession)
	b.Post("/class-sessions", manager, schedule.CreateClassSession)
	b.Put("/class-sessions/:id", manager, schedule.UpdateClassSession)
	b.Delete("/class-sessions/:id", manager, schedule.DeleteClassSession)

	// scopeType is "default" (id 0), "sport" or "resource"
	b.Get("/rules", staff, rules.ListRules)
	b.Get("/rules/effective", staff, rules.Effective)
	b.Get("/rules/:scopeType/:scopeId", staff, rules.GetRule)
	b.Put("/rules/:scopeType/:scopeId", manager, rules.SetRule)
	b.Delete("/rules/:scopeType/:scopeId", manager, rules.DeleteRule)

	b.Post("/uploads", manager, media.Upload)
	b.Delete("/uploads", manager, media.Delete)
}
