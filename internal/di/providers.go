package di

import (
	"context"

	"github.com/arifin2018/splitbill-arifin.git/internal/adapter/gemini"
	"github.com/arifin2018/splitbill-arifin.git/internal/adapter/storage"
	"github.com/arifin2018/splitbill-arifin.git/internal/config"
	"github.com/arifin2018/splitbill-arifin.git/internal/port"
	applogger "github.com/arifin2018/splitbill-arifin.git/pkg/logger"
	"github.com/sirupsen/logrus"
)

func provideLogger(cfg *config.Config) (*logrus.Logger, error) {
	return applogger.New(cfg.LogLevel, cfg.LogDir)
}

func provideExtractor(ctx context.Context, cfg *config.Config, log *logrus.Logger) (port.ReceiptExtractor, error) {
	return gemini.NewExtractor(ctx, cfg.GeminiAPIKey, cfg.GeminiModel, log)
}

func provideStorage(ctx context.Context, cfg *config.Config, log *logrus.Logger) (port.StorageUploader, error) {
	return storage.NewUploader(ctx, cfg, log)
}

func InitializeLogger(cfg *config.Config) (*logrus.Logger, error) {
	return provideLogger(cfg)
}
