package handler

import (
	"backend/internal/adapters/primary/http/middleware"
	"backend/internal/core/dto"
	"backend/internal/core/port/input"

	"github.com/gofiber/fiber/v2"
)

type (
	BusinessHandler interface {
		GetSettings(c *fiber.Ctx) error
		UpdateSettings(c *fiber.Ctx) error
		Publish(c *fiber.Ctx) error
		Unpublish(c *fiber.Ctx) error
		Usage(c *fiber.Ctx) error

		ListMembers(c *fiber.Ctx) error
		UpdateMember(c *fiber.Ctx) error
		RemoveMember(c *fiber.Ctx) error
		TransferOwnership(c *fiber.Ctx) error

		ListInvitations(c *fiber.Ctx) error
		Invite(c *fiber.Ctx) error
		CancelInvitation(c *fiber.Ctx) error
		ResendInvitation(c *fiber.Ctx) error

		ListSupportVisits(c *fiber.Ctx) error
	}
	businessHandler struct {
		svc input.BusinessService
	}
)

func NewBusinessHandler(svc input.BusinessService) BusinessHandler {
	return &businessHandler{svc: svc}
}

func (h *businessHandler) GetSettings(c *fiber.Ctx) error {
	res, err := h.svc.GetSettings(c.UserContext(), middleware.BusinessID(c))
	return ok(c, res, err)
}

func (h *businessHandler) UpdateSettings(c *fiber.Ctx) error {
	var req dto.SettingsRequest
	if err := bind(c, &req); err != nil {
		return fail(c, err)
	}
	res, err := h.svc.UpdateSettings(c.UserContext(), middleware.BusinessID(c), req)
	return ok(c, res, err)
}

func (h *businessHandler) Publish(c *fiber.Ctx) error {
	res, err := h.svc.Publish(c.UserContext(), middleware.BusinessID(c))
	return ok(c, res, err)
}

func (h *businessHandler) Unpublish(c *fiber.Ctx) error {
	res, err := h.svc.Unpublish(c.UserContext(), middleware.BusinessID(c))
	return ok(c, res, err)
}

func (h *businessHandler) Usage(c *fiber.Ctx) error {
	res, err := h.svc.Usage(c.UserContext(), middleware.BusinessID(c))
	return ok(c, res, err)
}

func (h *businessHandler) ListMembers(c *fiber.Ctx) error {
	res, err := h.svc.ListMembers(c.UserContext(), middleware.BusinessID(c))
	return ok(c, res, err)
}

func (h *businessHandler) UpdateMember(c *fiber.Ctx) error {
	userID, err := idParam(c, "userId")
	if err != nil {
		return fail(c, err)
	}
	var req dto.UpdateMemberRequest
	if err := bind(c, &req); err != nil {
		return fail(c, err)
	}
	res, err := h.svc.UpdateMember(c.UserContext(), middleware.BusinessID(c), userID, req)
	return ok(c, res, err)
}

func (h *businessHandler) RemoveMember(c *fiber.Ctx) error {
	userID, err := idParam(c, "userId")
	if err != nil {
		return fail(c, err)
	}
	return done(c, "DELETED", h.svc.RemoveMember(c.UserContext(), middleware.BusinessID(c), userID))
}

func (h *businessHandler) TransferOwnership(c *fiber.Ctx) error {
	var req dto.TransferOwnershipRequest
	if err := bind(c, &req); err != nil {
		return fail(c, err)
	}
	err := h.svc.TransferOwnership(c.UserContext(), middleware.BusinessID(c), middleware.CurrentUser(c).ID, req.UserID)
	return done(c, "UPDATED", err)
}

func (h *businessHandler) ListInvitations(c *fiber.Ctx) error {
	res, err := h.svc.ListInvitations(c.UserContext(), middleware.BusinessID(c))
	return ok(c, res, err)
}

func (h *businessHandler) Invite(c *fiber.Ctx) error {
	var req dto.InviteRequest
	if err := bind(c, &req); err != nil {
		return fail(c, err)
	}
	res, err := h.svc.Invite(c.UserContext(), middleware.BusinessID(c), middleware.CurrentUser(c), req)
	return created(c, res, err)
}

func (h *businessHandler) CancelInvitation(c *fiber.Ctx) error {
	return scopedRemove(c, h.svc.CancelInvitation)
}

func (h *businessHandler) ResendInvitation(c *fiber.Ctx) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	res, err := h.svc.ResendInvitation(c.UserContext(), middleware.BusinessID(c), middleware.CurrentUser(c), id)
	return ok(c, res, err)
}

func (h *businessHandler) ListSupportVisits(c *fiber.Ctx) error {
	return scopedList(c, h.svc.ListSupportVisits)
}
