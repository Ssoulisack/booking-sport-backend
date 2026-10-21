package handler

import (
	"backend/internal/adapters/primary/http/middleware"
	"backend/internal/core/dto"
	"backend/internal/core/port/input"

	"github.com/gofiber/fiber/v2"
)

type (
	PlatformHandler interface {
		ListPlans(c *fiber.Ctx) error
		UpdatePlan(c *fiber.Ctx) error

		CreateBusiness(c *fiber.Ctx) error
		ListBusinesses(c *fiber.Ctx) error
		GetBusiness(c *fiber.Ctx) error
		UpdateBusiness(c *fiber.Ctx) error
		Suspend(c *fiber.Ctx) error
		Reactivate(c *fiber.Ctx) error
		InviteOwner(c *fiber.Ctx) error

		ListUsers(c *fiber.Ctx) error
		SetUserStatus(c *fiber.Ctx) error
	}
	platformHandler struct {
		svc input.PlatformService
	}
)

func NewPlatformHandler(svc input.PlatformService) PlatformHandler {
	return &platformHandler{svc: svc}
}

func (h *platformHandler) ListPlans(c *fiber.Ctx) error {
	res, err := h.svc.ListPlans(c.UserContext())
	return ok(c, res, err)
}

func (h *platformHandler) UpdatePlan(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.PlanRequest
	if err := bind(c, &req); err != nil {
		return fail(c, err)
	}
	res, err := h.svc.UpdatePlan(c.UserContext(), id, req)
	return ok(c, res, err)
}

func (h *platformHandler) CreateBusiness(c *fiber.Ctx) error {
	var req dto.CreateBusinessRequest
	if err := bind(c, &req); err != nil {
		return fail(c, err)
	}
	res, err := h.svc.CreateBusiness(c.UserContext(), middleware.CurrentUser(c), req)
	return created(c, res, err)
}

func (h *platformHandler) ListBusinesses(c *fiber.Ctx) error {
	q, err := listQuery(c)
	if err != nil {
		return fail(c, err)
	}
	res, err := h.svc.ListBusinesses(c.UserContext(), q)
	return paged(c, res, err)
}

func (h *platformHandler) GetBusiness(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	res, err := h.svc.GetBusiness(c.UserContext(), id)
	return ok(c, res, err)
}

func (h *platformHandler) UpdateBusiness(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.UpdateBusinessRequest
	if err := bind(c, &req); err != nil {
		return fail(c, err)
	}
	res, err := h.svc.UpdateBusiness(c.UserContext(), id, req)
	return ok(c, res, err)
}

func (h *platformHandler) setSuspended(c *fiber.Ctx, suspended bool) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	res, err := h.svc.SetSuspended(c.UserContext(), id, suspended)
	return ok(c, res, err)
}

func (h *platformHandler) Suspend(c *fiber.Ctx) error    { return h.setSuspended(c, true) }
func (h *platformHandler) Reactivate(c *fiber.Ctx) error { return h.setSuspended(c, false) }

func (h *platformHandler) InviteOwner(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.InviteOwnerRequest
	if err := bind(c, &req); err != nil {
		return fail(c, err)
	}
	res, err := h.svc.InviteOwner(c.UserContext(), middleware.CurrentUser(c), id, req.Email)
	return created(c, res, err)
}

func (h *platformHandler) ListUsers(c *fiber.Ctx) error {
	q, err := listQuery(c)
	if err != nil {
		return fail(c, err)
	}
	res, err := h.svc.ListUsers(c.UserContext(), q)
	return paged(c, res, err)
}

func (h *platformHandler) SetUserStatus(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.UserStatusRequest
	if err := bind(c, &req); err != nil {
		return fail(c, err)
	}
	return done(c, "UPDATED", h.svc.SetUserStatus(c.UserContext(), middleware.CurrentUser(c).ID, id, req.Status))
}
