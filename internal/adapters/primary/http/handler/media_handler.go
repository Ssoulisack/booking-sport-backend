package handler

import (
	"backend/internal/core/port/input"
	"backend/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type (
	MediaHandler interface {
		Upload(c *fiber.Ctx) error
		Delete(c *fiber.Ctx) error
	}
	mediaHandler struct {
		svc input.MediaService
	}
)

func NewMediaHandler(svc input.MediaService) MediaHandler {
	return &mediaHandler{svc: svc}
}

// Upload expects multipart/form-data with a "file" part and ?folder=sports|coaches|classes
func (h *mediaHandler) Upload(c *fiber.Ctx) error {
	header, err := c.FormFile("file")
	if err != nil {
		return fail(c, response.ErrorBadRequest("FILE_IS_REQUIRED"))
	}
	file, err := header.Open()
	if err != nil {
		return fail(c, response.ErrorBadRequest("INVALID_FILE"))
	}
	defer file.Close()

	res, err := h.svc.Upload(c.UserContext(), c.Query("folder"), file)
	if err != nil {
		return fail(c, err)
	}
	return response.NewCreatedResponse(c, res)
}

// Delete expects ?key= of a file the server generated earlier
func (h *mediaHandler) Delete(c *fiber.Ctx) error {
	if err := h.svc.Delete(c.UserContext(), c.Query("key")); err != nil {
		return fail(c, err)
	}
	return response.NewSuccessMessagesResponse(c, "DELETED")
}
