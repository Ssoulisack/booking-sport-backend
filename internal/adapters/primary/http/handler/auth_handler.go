package handler

import (
	"backend/internal/core/mappers"
	"backend/internal/core/port/input"
)

type (
	AuthHandler interface {
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
