package handler

import (
	"backend/internal/core/dto"
	"backend/internal/core/port/input"
	"backend/pkg/jwt"
	"backend/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type (
	AdminUserHandler interface {
		CreateStaff(c *fiber.Ctx) error
		ListUsers(c *fiber.Ctx) error
		SetUserStatus(c *fiber.Ctx) error
	}
	adminUserHandler struct {
		svc input.AdminUserService
	}
)

func NewAdminUserHandler(svc input.AdminUserService) AdminUserHandler {
	return &adminUserHandler{svc: svc}
}

func (h *adminUserHandler) CreateStaff(c *fiber.Ctx) error { return create(c, h.svc.CreateStaff) }
func (h *adminUserHandler) ListUsers(c *fiber.Ctx) error   { return list(c, h.svc.ListUsers) }

func (h *adminUserHandler) SetUserStatus(c *fiber.Ctx) error {
	targetID, err := idParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req dto.UserStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, response.ErrorBadRequest("INVALID_BODY"))
	}
	claims, err := jwt.GetOwnerAccessToken(c)
	if err != nil {
		return response.NewErrorErrMsgUnauthorizedErrMsgInvalidToken(c)
	}
	if err := h.svc.SetUserStatus(c.UserContext(), uint(claims.ID), targetID, req.Status); err != nil {
		return fail(c, err)
	}
	return response.NewSuccessMessagesResponse(c, "UPDATED")
}
