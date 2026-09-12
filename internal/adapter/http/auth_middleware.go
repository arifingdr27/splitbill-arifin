package httpadapter

import (
	"strings"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/arifin2018/splitbill-arifin.git/internal/port"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const localsUserID = "user_id"

func AuthMiddleware(tokens port.TokenIssuer, log *logrus.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if header == "" || !strings.HasPrefix(strings.ToLower(header), "bearer ") {
			return writeError(c, domain.ErrUnauthorized)
		}
		raw := strings.TrimSpace(header[7:])
		userID, _, err := tokens.Parse(raw)
		if err != nil {
			log.WithError(err).Warn("jwt parse failed")
			return writeError(c, domain.ErrUnauthorized)
		}
		c.Locals(localsUserID, userID)
		return c.Next()
	}
}

func UserIDFromCtx(c *fiber.Ctx) (uuid.UUID, error) {
	v := c.Locals(localsUserID)
	id, ok := v.(uuid.UUID)
	if !ok || id == uuid.Nil {
		return uuid.Nil, domain.ErrUnauthorized
	}
	return id, nil
}
