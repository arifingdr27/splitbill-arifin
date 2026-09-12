package httpadapter

import (
	"github.com/gofiber/fiber/v2"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

type Router struct {
	handler *SplitbillHandler
}

func NewRouter(handler *SplitbillHandler) *Router {
	return &Router{handler: handler}
}

func (r *Router) Mount(app *fiber.App) {
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	v2 := app.Group("/api/v2")
	v2.Post("", r.handler.Extract)
}
