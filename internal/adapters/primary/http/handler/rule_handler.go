package handler

import (
	"backend/internal/core/dto"
	"backend/internal/core/port/input"
	"backend/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type (
	RuleHandler interface {
		ListRules(c *fiber.Ctx) error
		GetRule(c *fiber.Ctx) error
		SetRule(c *fiber.Ctx) error
		DeleteRule(c *fiber.Ctx) error
		Effective(c *fiber.Ctx) error
	}
	ruleHandler struct {
		svc input.RuleService
	}
)

func NewRuleHandler(svc input.RuleService) RuleHandler {
	return &ruleHandler{svc: svc}
}

// scopeParams reads /:scopeType/:scopeId; the global rule uses id 0
func scopeParams(c *fiber.Ctx) (string, uint, error) {
	id, err := c.ParamsInt("scopeId")
	if err != nil || id < 0 {
		return "", 0, response.ErrorBadRequest("INVALID_ID")
	}
	return c.Params("scopeType"), uint(id), nil
}

func (h *ruleHandler) ListRules(c *fiber.Ctx) error {
	res, err := h.svc.ListRules(c.UserContext())
	if err != nil {
		return fail(c, err)
	}
	return response.NewSuccessResponse(c, res)
}

func (h *ruleHandler) GetRule(c *fiber.Ctx) error {
	scopeType, scopeID, err := scopeParams(c)
	if err != nil {
		return fail(c, err)
	}
	res, err := h.svc.GetRule(c.UserContext(), scopeType, scopeID)
	if err != nil {
		return fail(c, err)
	}
	return response.NewSuccessResponse(c, res)
}

func (h *ruleHandler) SetRule(c *fiber.Ctx) error {
	scopeType, scopeID, err := scopeParams(c)
	if err != nil {
		return fail(c, err)
	}
	var req dto.RuleRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, response.ErrorBadRequest("INVALID_BODY"))
	}
	res, err := h.svc.SetRule(c.UserContext(), scopeType, scopeID, req)
	if err != nil {
		return fail(c, err)
	}
	return response.NewSuccessResponse(c, res)
}

func (h *ruleHandler) DeleteRule(c *fiber.Ctx) error {
	scopeType, scopeID, err := scopeParams(c)
	if err != nil {
		return fail(c, err)
	}
	if err := h.svc.DeleteRule(c.UserContext(), scopeType, scopeID); err != nil {
		return fail(c, err)
	}
	return response.NewSuccessMessagesResponse(c, "DELETED")
}

// Effective answers "which rule applies?" for ?sport_id= or ?resource_id=
func (h *ruleHandler) Effective(c *fiber.Ctx) error {
	var q dto.ListQuery
	if err := c.QueryParser(&q); err != nil {
		return fail(c, response.ErrorBadRequest("INVALID_QUERY"))
	}
	resourceID := uint(c.QueryInt("resource_id", 0))
	res, err := h.svc.Effective(c.UserContext(), q.SportID, resourceID)
	if err != nil {
		return fail(c, err)
	}
	return response.NewSuccessResponse(c, res)
}
