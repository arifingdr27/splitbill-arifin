package httpadapter

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/sirupsen/logrus"
)

func RegisterMiddleware(app *fiber.App, corsOrigins string, log *logrus.Logger) {
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: corsOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))
	app.Use(requestLogger(log))
}

func requestLogger(log *logrus.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		log.WithFields(logrus.Fields{
			"method":   c.Method(),
			"path":     c.Path(),
			"status":   c.Response().StatusCode(),
			"latency":  time.Since(start).String(),
			"ip":       c.IP(),
			"error":    err,
		}).Info("request")
		return err
	}
}
