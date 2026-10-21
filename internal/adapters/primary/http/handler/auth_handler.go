package handler

import (
	"backend/internal/core/dto"
	"backend/internal/core/mappers"
	"backend/internal/core/port/input"
	"backend/pkg/jwt"
	"backend/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type (
	AuthHandler interface {
		Register(c *fiber.Ctx) error
		Login(c *fiber.Ctx) error
		Refresh(c *fiber.Ctx) error
		Me(c *fiber.Ctx) error
	}
	authHandler struct {
		authService   input.AuthService
		mapperFactory mappers.MapperFactory
	}
)

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(
	authService input.AuthService,
	mapperFactory mappers.MapperFactory,
) AuthHandler {
	return &authHandler{
		authService:   authService,
		mapperFactory: mapperFactory,
	}
}

func (h *authHandler) Register(c *fiber.Ctx) error {
	var req dto.RegisterDTO
	if err := c.BodyParser(&req); err != nil {
		return response.NewErrorBadRequest(c)
	}
	res, err := h.authService.Register(c.UserContext(), req)
	if err != nil {
		return response.NewErrorResponses(c, err)
	}
	return response.NewCreatedResponse(c, res)
}

func (h *authHandler) Login(c *fiber.Ctx) error {
	var req dto.LoginDTO
	if err := c.BodyParser(&req); err != nil {
		return response.NewErrorBadRequest(c)
	}
	res, err := h.authService.Login(c.UserContext(), req)
	if err != nil {
		return response.NewErrorResponses(c, err)
	}
	return response.NewSuccessResponse(c, res)
}

// Refresh expects the refresh token as the bearer token (see jwt.AccessRefreshToken)
func (h *authHandler) Refresh(c *fiber.Ctx) error {
	claims, err := jwt.GetOwnerRefresh(c)
	if err != nil {
		return response.NewErrorErrMsgUnauthorizedErrMsgInvalidToken(c)
	}
	res, err := h.authService.Refresh(c.UserContext(), uint(claims.ID))
	if err != nil {
		return response.NewErrorResponses(c, err)
	}
	return response.NewSuccessResponse(c, res)
}

func (h *authHandler) Me(c *fiber.Ctx) error {
	claims, err := jwt.GetOwnerAccessToken(c)
	if err != nil {
		return response.NewErrorErrMsgUnauthorizedErrMsgInvalidToken(c)
	}
	res, err := h.authService.Me(c.UserContext(), uint(claims.ID))
	if err != nil {
		return response.NewErrorResponses(c, err)
	}
	return response.NewSuccessResponse(c, res)
}
