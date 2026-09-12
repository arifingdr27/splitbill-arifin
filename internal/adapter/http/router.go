package httpadapter

import (
	"github.com/arifin2018/splitbill-arifin.git/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

type Router struct {
	handler *SplitbillHandler
	cfg     *config.Config
	log     *logrus.Logger
}

func NewRouter(handler *SplitbillHandler, cfg *config.Config, log *logrus.Logger) *Router {
	return &Router{handler: handler, cfg: cfg, log: log}
}

func (r *Router) Mount(app *fiber.App) {
	if !r.cfg.IsProduction() {
		app.Get("/swagger/*", fiberSwagger.WrapHandler)
	}

	v2 := app.Group("/api/v2", rateLimitMiddleware(r.cfg.RateLimitRPM, r.log))
	v2.Post("", r.handler.Extract)
}
