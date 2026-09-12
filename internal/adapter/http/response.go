package httpadapter

import (
	"errors"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/gofiber/fiber/v2"
)

func writeSuccess(c *fiber.Ctx, data any) error {
	return c.Status(fiber.StatusOK).JSON(data)
}

func writeError(c *fiber.Ctx, err error) error {
	status := fiber.StatusUnprocessableEntity
	msg := "failed to process receipt"

	switch {
	case errors.Is(err, domain.ErrImageRequired):
		status = fiber.StatusBadRequest
		msg = domain.ErrImageRequired.Error()
	case errors.Is(err, domain.ErrInvalidImageType):
		status = fiber.StatusBadRequest
		msg = domain.ErrInvalidImageType.Error()
	case errors.Is(err, domain.ErrPayloadTooLarge):
		status = fiber.StatusRequestEntityTooLarge
		msg = domain.ErrPayloadTooLarge.Error()
	case errors.Is(err, domain.ErrTooManyRequests):
		status = fiber.StatusTooManyRequests
		msg = domain.ErrTooManyRequests.Error()
		c.Set("Retry-After", "60")
	case errors.Is(err, domain.ErrStorageUnavailable), errors.Is(err, domain.ErrServiceBusy):
		status = fiber.StatusServiceUnavailable
		msg = "service temporarily unavailable"
	case errors.Is(err, domain.ErrExtractFailed):
		status = fiber.StatusUnprocessableEntity
		msg = "failed to process receipt"
	}

	return c.Status(status).JSON(domain.ErrorResponse{
		Data:   "",
		Status: msg,
	})
}
