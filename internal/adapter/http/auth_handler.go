package httpadapter

import (
	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/arifin2018/splitbill-arifin.git/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

type AuthHandler struct {
	auth *service.AuthService
	log  *logrus.Logger
}

func NewAuthHandler(auth *service.AuthService, log *logrus.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, log: log}
}

type googleLoginRequest struct {
	IDToken string `json:"id_token"`
}

func (h *AuthHandler) GoogleLogin(c *fiber.Ctx) error {
	var req googleLoginRequest
	if err := c.BodyParser(&req); err != nil || req.IDToken == "" {
		return writeError(c, domain.ErrInvalidGoogleToken)
	}
	result, err := h.auth.LoginWithGoogle(c.UserContext(), req.IDToken)
	if err != nil {
		return writeError(c, err)
	}
	return writeSuccess(c, result)
}

type QuotaHandler struct {
	quota *service.QuotaService
	log   *logrus.Logger
}

func NewQuotaHandler(quota *service.QuotaService, log *logrus.Logger) *QuotaHandler {
	return &QuotaHandler{quota: quota, log: log}
}

func (h *QuotaHandler) MeQuota(c *fiber.Ctx) error {
	userID, err := UserIDFromCtx(c)
	if err != nil {
		return writeError(c, err)
	}
	snap, err := h.quota.Snapshot(c.UserContext(), userID)
	if err != nil {
		h.log.WithError(err).Error("get quota failed")
		return writeError(c, err)
	}
	return writeSuccess(c, snap)
}
