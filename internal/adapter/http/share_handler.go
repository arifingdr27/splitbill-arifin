package httpadapter

import (
	"encoding/json"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/arifin2018/splitbill-arifin.git/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type ShareHandler struct {
	shares *service.ShareService
	log    *logrus.Logger
}

func NewShareHandler(shares *service.ShareService, log *logrus.Logger) *ShareHandler {
	return &ShareHandler{shares: shares, log: log}
}

func (h *ShareHandler) Create(c *fiber.Ctx) error {
	userID, err := UserIDFromCtx(c)
	if err != nil {
		return writeError(c, err)
	}

	body := c.Body()
	if len(body) == 0 {
		return writeError(c, domain.ErrSharePayloadInvalid)
	}
	if !json.Valid(body) {
		return writeError(c, domain.ErrSharePayloadInvalid)
	}

	result, err := h.shares.Create(c.UserContext(), userID, json.RawMessage(body))
	if err != nil {
		return writeError(c, err)
	}
	return writeSuccess(c, result)
}

func (h *ShareHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return writeError(c, domain.ErrShareNotFound)
	}
	view, err := h.shares.Get(c.UserContext(), id)
	if err != nil {
		return writeError(c, err)
	}
	return writeSuccess(c, view)
}
