package di

import (
	"context"
	"fmt"
	"strings"

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
	var chain []port.ReceiptExtractor
	var labels []string

	for _, name := range cfg.ExtractProviders {
		switch name {
		case config.ProviderGemini:
			ex, err := gemini.NewExtractor(ctx, cfg.GeminiAPIKey, cfg.GeminiModel, cfg.GeminiTimeout, log)
			if err != nil {
				return nil, err
			}
			chain = append(chain, ex)
			labels = append(labels, fmt.Sprintf("gemini(%s)", cfg.GeminiModel))
		case config.ProviderGroq:
			ex, err := groq.NewExtractor(cfg.GroqAPIKey, cfg.GroqModel, cfg.GroqTimeout, log)
			if err != nil {
				return nil, err
			}
			chain = append(chain, ex)
			labels = append(labels, fmt.Sprintf("groq(%s)", cfg.GroqModel))
		default:
			return nil, fmt.Errorf("unknown extract provider %q", name)
		}
	}

	log.Infof("extract providers: %s", strings.Join(labels, " -> "))
	if len(chain) == 1 {
		return chain[0], nil
	}
	return failover.New(chain[0], chain[1], log), nil
}

func provideStorage(ctx context.Context, cfg *config.Config, log *logrus.Logger) (port.StorageUploader, error) {
	return storage.NewUploader(ctx, cfg, log)
}

func InitializeLogger(cfg *config.Config) (*logrus.Logger, error) {
	return provideLogger(cfg)
}
