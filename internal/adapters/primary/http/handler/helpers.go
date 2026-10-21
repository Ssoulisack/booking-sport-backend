package handler

import (
	"backend/internal/core/dto"
	"backend/pkg/response"
	"context"

	"github.com/gofiber/fiber/v2"
)

// idParam reads a positive numeric path parameter
func idParam(c *fiber.Ctx, name string) (uint, error) {
	id, err := c.ParamsInt(name)
	if err != nil || id < 1 {
		return 0, response.ErrorBadRequest("INVALID_ID")
	}
	return uint(id), nil
}

// ownerParams reads /:ownerType/:ownerId; the venue has no id, so 0 is allowed here
func ownerParams(c *fiber.Ctx) (string, uint, error) {
	id, err := c.ParamsInt("ownerId")
	if err != nil || id < 0 {
		return "", 0, response.ErrorBadRequest("INVALID_ID")
	}
	return c.Params("ownerType"), uint(id), nil
}

func fail(c *fiber.Ctx, err error) error { return response.NewErrorResponses(c, err) }

func create[Req, Res any](c *fiber.Ctx, fn func(context.Context, Req) (*Res, error)) error {
	var req Req
	if err := c.BodyParser(&req); err != nil {
		return fail(c, response.ErrorBadRequest("INVALID_BODY"))
	}
	res, err := fn(c.UserContext(), req)
	if err != nil {
		return fail(c, err)
	}
	return response.NewCreatedResponse(c, res)
}

func get[Res any](c *fiber.Ctx, fn func(context.Context, uint) (*Res, error)) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	res, err := fn(c.UserContext(), id)
	if err != nil {
		return fail(c, err)
	}
	return response.NewSuccessResponse(c, res)
}

func list[Res any](c *fiber.Ctx, fn func(context.Context, dto.ListQuery) (*dto.PagedResult[Res], error)) error {
	var q dto.ListQuery
	if err := c.QueryParser(&q); err != nil {
		return fail(c, response.ErrorBadRequest("INVALID_QUERY"))
	}
	res, err := fn(c.UserContext(), q)
	if err != nil {
		return fail(c, err)
	}
	return response.NewPaginatedResponse(c, res.Items, res.Meta)
}

func update[Req, Res any](c *fiber.Ctx, fn func(context.Context, uint, Req) (*Res, error)) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	var req Req
	if err := c.BodyParser(&req); err != nil {
		return fail(c, response.ErrorBadRequest("INVALID_BODY"))
	}
	res, err := fn(c.UserContext(), id, req)
	if err != nil {
		return fail(c, err)
	}
	return response.NewSuccessResponse(c, res)
}

func remove(c *fiber.Ctx, fn func(context.Context, uint) error) error {
	id, err := idParam(c, "id")
	if err != nil {
		return fail(c, err)
	}
	if err := fn(c.UserContext(), id); err != nil {
		return fail(c, err)
	}
	return response.NewSuccessMessagesResponse(c, "DELETED")
}
