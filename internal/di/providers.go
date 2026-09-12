package di

import (
	"context"

	"github.com/arifin2018/splitbill-arifin.git/internal/adapter/failover"
	"github.com/arifin2018/splitbill-arifin.git/internal/adapter/gemini"
	"github.com/arifin2018/splitbill-arifin.git/internal/adapter/groq"
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
	primary, err := gemini.NewExtractor(ctx, cfg.GeminiAPIKey, cfg.GeminiModel, cfg.GeminiTimeout, log)
	if err != nil {
		return nil, err
	}
	if cfg.GroqAPIKey == "" {
		log.Info("GROQ_API_KEY not set; Gemini-only extract (no fallback)")
		return primary, nil
	}

	secondary, err := groq.NewExtractor(cfg.GroqAPIKey, cfg.GroqModel, cfg.GroqTimeout, log)
	if err != nil {
		return nil, err
	}
	log.Infof("extract failover enabled: gemini(%s) -> groq(%s)", cfg.GeminiModel, cfg.GroqModel)
	return failover.New(primary, secondary, log), nil
}

func provideStorage(ctx context.Context, cfg *config.Config, log *logrus.Logger) (port.StorageUploader, error) {
	return storage.NewUploader(ctx, cfg, log)
}

func InitializeLogger(cfg *config.Config) (*logrus.Logger, error) {
	return provideLogger(cfg)
}
