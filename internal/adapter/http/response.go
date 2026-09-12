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
	msg := err.Error()

	switch {
	case errors.Is(err, domain.ErrImageRequired), errors.Is(err, domain.ErrInvalidImageType):
		status = fiber.StatusBadRequest
	case errors.Is(err, domain.ErrStorageUnavailable):
		status = fiber.StatusServiceUnavailable
	}

	return c.Status(status).JSON(domain.ErrorResponse{
		Data:   "",
		Status: msg,
	})
}
