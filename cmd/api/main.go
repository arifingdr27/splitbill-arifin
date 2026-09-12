package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/arifin2018/splitbill-arifin.git/docs"
	httpadapter "github.com/arifin2018/splitbill-arifin.git/internal/adapter/http"
	"github.com/arifin2018/splitbill-arifin.git/internal/adapter/storage"
	"github.com/arifin2018/splitbill-arifin.git/internal/config"
	"github.com/arifin2018/splitbill-arifin.git/internal/di"
	"github.com/gofiber/fiber/v2"
)

// @title Splitbill API
// @version 1.0
// @description API untuk mengekstrak informasi splitbill dari gambar struk menggunakan OCR dan AI
// @host localhost:3000
// @BasePath /
func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	log, err := di.InitializeLogger(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger error: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	router, err := di.InitializeRouter(ctx, cfg)
	if err != nil {
		log.Fatalf("dependency init failed: %v", err)
	}

	if cfg.BucketStorage == config.StorageVM {
		storage.StartLocalImageCleanup(ctx, cfg.StorageLocalPath, cfg.StorageRetentionDays, cfg.StorageCleanupIntervalHrs, log)
	}

	app := fiber.New(fiber.Config{
		AppName:       "splitbill-api",
		BodyLimit:     cfg.HTTPBodyLimitBytes,
		ReadTimeout:   cfg.HTTPReadTimeout,
		WriteTimeout:  cfg.HTTPWriteTimeout,
		IdleTimeout:   cfg.HTTPIdleTimeout,
		Concurrency:   cfg.HTTPConcurrency,
		ProxyHeader:   fiber.HeaderXForwardedFor,
		ErrorHandler:  httpadapter.FiberErrorHandler,
	})
	httpadapter.RegisterMiddleware(app, cfg.CORSOrigins, log)
	router.Mount(app)

	go func() {
		addr := ":" + cfg.AppPort
		log.Infof("listening on %s (env=%s storage=%s)", addr, cfg.AppEnv, cfg.BucketStorage)
		if err := app.Listen(addr); err != nil {
			log.Errorf("server stopped: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info("shutting down...")
	cancel()
	_ = app.Shutdown()
	log.Info("cleanup done")
}
