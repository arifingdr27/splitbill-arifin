package httpadapter

import (
	"github.com/arifin2018/splitbill-arifin.git/internal/config"
	"github.com/arifin2018/splitbill-arifin.git/internal/port"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

type Router struct {
	handler      *SplitbillHandler
	authHandler  *AuthHandler
	quotaHandler *QuotaHandler
	tokens       port.TokenIssuer
	cfg          *config.Config
	log          *logrus.Logger
}

func NewRouter(
	handler *SplitbillHandler,
	authHandler *AuthHandler,
	quotaHandler *QuotaHandler,
	tokens port.TokenIssuer,
	cfg *config.Config,
	log *logrus.Logger,
) *Router {
	return &Router{
		handler:      handler,
		authHandler:  authHandler,
		quotaHandler: quotaHandler,
		tokens:       tokens,
		cfg:          cfg,
		log:          log,
	}
}

func (r *Router) Mount(app *fiber.App) {
	if !r.cfg.IsProduction() {
		app.Get("/swagger/*", fiberSwagger.WrapHandler)
	}

	authMW := AuthMiddleware(r.tokens, r.log)
	globalRL := rateLimitMiddleware(r.cfg.GlobalRateLimitRPM, r.log)
	authRL := rateLimitMiddleware(r.cfg.AuthRateLimitRPM, r.log)
	extractRL := rateLimitMiddleware(r.cfg.RateLimitRPM, r.log)

	api := app.Group("/api", globalRL)

	v1 := api.Group("/v1")
	v1.Post("/auth/google", authRL, r.authHandler.GoogleLogin)
	v1.Get("/me/quota", authMW, r.quotaHandler.MeQuota)

	v2 := api.Group("/v2", extractRL, authMW)
	v2.Post("", r.handler.Extract)
}
