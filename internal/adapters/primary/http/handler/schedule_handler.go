package handler

import (
	"backend/internal/core/dto"
	"backend/internal/core/port/input"
	"backend/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type (
	ScheduleHandler interface {
		GetOpeningHours(c *fiber.Ctx) error
		SetOpeningHours(c *fiber.Ctx) error

		CreateClosure(c *fiber.Ctx) error
		GetClosure(c *fiber.Ctx) error
		ListClosures(c *fiber.Ctx) error
		UpdateClosure(c *fiber.Ctx) error
		DeleteClosure(c *fiber.Ctx) error

		GetPrice(c *fiber.Ctx) error
		SetPrice(c *fiber.Ctx) error
		DeletePrice(c *fiber.Ctx) error

		CreateClassSession(c *fiber.Ctx) error
		GetClassSession(c *fiber.Ctx) error
		ListClassSessions(c *fiber.Ctx) error
		UpdateClassSession(c *fiber.Ctx) error
		DeleteClassSession(c *fiber.Ctx) error
	}
	scheduleHandler struct {
		svc input.ScheduleService
	}
)

func NewScheduleHandler(svc input.ScheduleService) ScheduleHandler {
	return &scheduleHandler{svc: svc}
}

func (h *scheduleHandler) GetOpeningHours(c *fiber.Ctx) error {
	ownerType, ownerID, err := ownerParams(c)
	if err != nil {
		return fail(c, err)
	}
	res, err := h.svc.GetOpeningHours(c.UserContext(), ownerType, ownerID)
	if err != nil {
		return fail(c, err)
	}
	return response.NewSuccessResponse(c, res)
}

func (h *scheduleHandler) SetOpeningHours(c *fiber.Ctx) error {
	ownerType, ownerID, err := ownerParams(c)
	if err != nil {
		return fail(c, err)
	}
	var req dto.OpeningHoursRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, response.ErrorBadRequest("INVALID_BODY"))
	}
	res, err := h.svc.SetOpeningHours(c.UserContext(), ownerType, ownerID, req)
	if err != nil {
		return fail(c, err)
	}
	return response.NewSuccessResponse(c, res)
}

func (h *scheduleHandler) CreateClosure(c *fiber.Ctx) error { return create(c, h.svc.CreateClosure) }
func (h *scheduleHandler) GetClosure(c *fiber.Ctx) error    { return get(c, h.svc.GetClosure) }
func (h *scheduleHandler) ListClosures(c *fiber.Ctx) error  { return list(c, h.svc.ListClosures) }
func (h *scheduleHandler) UpdateClosure(c *fiber.Ctx) error { return update(c, h.svc.UpdateClosure) }
func (h *scheduleHandler) DeleteClosure(c *fiber.Ctx) error { return remove(c, h.svc.DeleteClosure) }

func (h *scheduleHandler) GetPrice(c *fiber.Ctx) error {
	ownerType, ownerID, err := ownerParams(c)
	if err != nil {
		return fail(c, err)
	}
	res, err := h.svc.GetPrice(c.UserContext(), ownerType, ownerID)
	if err != nil {
		return fail(c, err)
	}
	return response.NewSuccessResponse(c, res)
}

func (h *scheduleHandler) SetPrice(c *fiber.Ctx) error {
	ownerType, ownerID, err := ownerParams(c)
	if err != nil {
		return fail(c, err)
	}
	var req dto.PriceRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, response.ErrorBadRequest("INVALID_BODY"))
	}
	res, err := h.svc.SetPrice(c.UserContext(), ownerType, ownerID, req)
	if err != nil {
		return fail(c, err)
	}
	return response.NewSuccessResponse(c, res)
}

func (h *scheduleHandler) DeletePrice(c *fiber.Ctx) error {
	ownerType, ownerID, err := ownerParams(c)
	if err != nil {
		return fail(c, err)
	}
	if err := h.svc.DeletePrice(c.UserContext(), ownerType, ownerID); err != nil {
		return fail(c, err)
	}
	return response.NewSuccessMessagesResponse(c, "DELETED")
}

func (h *scheduleHandler) CreateClassSession(c *fiber.Ctx) error {
	return create(c, h.svc.CreateClassSession)
}
func (h *scheduleHandler) GetClassSession(c *fiber.Ctx) error { return get(c, h.svc.GetClassSession) }
func (h *scheduleHandler) ListClassSessions(c *fiber.Ctx) error {
	return list(c, h.svc.ListClassSessions)
}
func (h *scheduleHandler) UpdateClassSession(c *fiber.Ctx) error {
	return update(c, h.svc.UpdateClassSession)
}
func (h *scheduleHandler) DeleteClassSession(c *fiber.Ctx) error {
	return remove(c, h.svc.DeleteClassSession)
}
