package httputil

import (
	"github.com/gofiber/fiber/v2"

	"github.com/team-attention/cops/api/internal/platform/util/errutil"
)

// ErrorResponse sends an error response based on the error type.
func ErrorResponse(ctx *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	message := "internal server error"

	if appErr, ok := err.(*errutil.AppError); ok {
		message = appErr.Message
		switch appErr.Type {
		case errutil.ErrorTypeBadRequest:
			status = fiber.StatusBadRequest
		case errutil.ErrorTypeUnauthorized:
			status = fiber.StatusUnauthorized
		case errutil.ErrorTypeForbidden:
			status = fiber.StatusForbidden
		case errutil.ErrorTypeNotFound:
			status = fiber.StatusNotFound
		case errutil.ErrorTypeInternal:
			status = fiber.StatusInternalServerError
		}
	}

	return ctx.Status(status).JSON(fiber.Map{
		"error": message,
	})
}
