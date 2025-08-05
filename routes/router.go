package routes

import (
	"github.com/arifin2018/splitbill-arifin.git/injector"
	"github.com/gofiber/fiber/v2"
)

func Router(app *fiber.App) {
	allController := injector.InitializeController()

	api_v2 := app.Group("/api/v2/")

	api_v2.Post("/", allController.SplitbilController.Splitbil)
}
