package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/arifin2018/splitbill-arifin.git/config"
	appconfig "github.com/arifin2018/splitbill-arifin.git/config/appConfig"
	"github.com/arifin2018/splitbill-arifin.git/routes"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	_ "github.com/arifin2018/splitbill-arifin.git/docs" // Import generated docs
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

// @title Splitbill API
// @version 1.0
// @description API untuk mengekstrak informasi splitbill dari gambar struk menggunakan OCR dan AI
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:3000
// @BasePath /
func main() {
	app := fiber.New()

	// Logging ke stdout
	// app.Use(logger.New())

	// Allow semua origin agar frontend bisa akses dari mana saja
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	serverShutdown := make(chan struct{})

	go func() {
		_ = <-c
		fmt.Println("Gracefully shutting down...")
		_ = app.Shutdown()
		serverShutdown <- struct{}{}
	}()
	appconfig.InitApplication()
	app.Use(cors.New())

	config.ConnectFirebase()
	config.Logger(app)

	// Add Swagger route
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	routes.Router(app)
	if err := app.Listen(":3000"); err != nil {
		panic(err.Error())
	}

	<-serverShutdown

	config.GeneralLogger.Println("Running cleanup tasks...")
	fmt.Println("Running cleanup tasks...")
}
