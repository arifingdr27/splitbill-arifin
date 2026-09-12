package di

import (
	"context"
	"fmt"

	"github.com/arifin2018/splitbill-arifin.git/internal/adapter/failover"
	"github.com/arifin2018/splitbill-arifin.git/internal/adapter/gemini"
	"github.com/arifin2018/splitbill-arifin.git/internal/adapter/googleauth"
	"github.com/arifin2018/splitbill-arifin.git/internal/adapter/groq"
	httpadapter "github.com/arifin2018/splitbill-arifin.git/internal/adapter/http"
	"github.com/arifin2018/splitbill-arifin.git/internal/adapter/jwtauth"
	"github.com/arifin2018/splitbill-arifin.git/internal/adapter/postgres"
	"github.com/arifin2018/splitbill-arifin.git/internal/adapter/storage"
	"github.com/arifin2018/splitbill-arifin.git/internal/config"
	"github.com/arifin2018/splitbill-arifin.git/internal/migrate"
	"github.com/arifin2018/splitbill-arifin.git/internal/port"
	"github.com/arifin2018/splitbill-arifin.git/internal/service"
	applogger "github.com/arifin2018/splitbill-arifin.git/pkg/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

func provideLogger(cfg *config.Config) (*logrus.Logger, error) {
	return applogger.New(cfg.LogLevel, cfg.LogDir)
}

func providePool(ctx context.Context, cfg *config.Config, log *logrus.Logger) (*pgxpool.Pool, error) {
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, log)
	if err != nil {
		return nil, err
	}
	if err := migrate.New(pool, log).Up(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func provideUserRepo(pool *pgxpool.Pool) port.UserRepository {
	return postgres.NewUserRepo(pool)
}

func provideUsageRepo(pool *pgxpool.Pool) port.UsageRepository {
	return postgres.NewUsageRepo(pool)
}

func provideGoogleVerifier(cfg *config.Config) port.GoogleTokenVerifier {
	return googleauth.NewVerifier(cfg.GoogleClientID)
}

func provideTokenIssuer(cfg *config.Config) port.TokenIssuer {
	return jwtauth.NewIssuer(cfg.JWTSecret, cfg.JWTTTLDays)
}

func provideAuthService(users port.UserRepository, verifier port.GoogleTokenVerifier, tokens port.TokenIssuer, log *logrus.Logger) *service.AuthService {
	return service.NewAuthService(users, verifier, tokens, log)
}

func provideQuotaService(pool *pgxpool.Pool, users port.UserRepository, usage port.UsageRepository, cfg *config.Config, log *logrus.Logger) *service.QuotaService {
	return service.NewQuotaService(pool, users, usage, cfg.FreeOCRLimit, log)
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
			labels = append(labels, "gemini("+cfg.GeminiModel+")")
		case config.ProviderGroq:
			ex, err := groq.NewExtractor(cfg.GroqAPIKey, cfg.GroqModel, cfg.GroqTimeout, log)
			if err != nil {
				return nil, err
			}
			chain = append(chain, ex)
			labels = append(labels, "groq("+cfg.GroqModel+")")
		default:
			return nil, fmt.Errorf("unknown extract provider %q", name)
		}
	}

	log.Infof("extract providers: %v", labels)
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

func InitializeRouter(ctx context.Context, cfg *config.Config) (*httpadapter.Router, error) {
	log, err := provideLogger(cfg)
	if err != nil {
		return nil, err
	}
	pool, err := providePool(ctx, cfg, log)
	if err != nil {
		return nil, err
	}
	users := provideUserRepo(pool)
	usage := provideUsageRepo(pool)
	verifier := provideGoogleVerifier(cfg)
	tokens := provideTokenIssuer(cfg)
	authSvc := provideAuthService(users, verifier, tokens, log)
	quotaSvc := provideQuotaService(pool, users, usage, cfg, log)

	uploader, err := provideStorage(ctx, cfg, log)
	if err != nil {
		return nil, err
	}
	extractor, err := provideExtractor(ctx, cfg, log)
	if err != nil {
		return nil, err
	}
	svc := service.NewSplitbillService(uploader, extractor, log, cfg)
	handler := httpadapter.NewSplitbillHandler(svc, quotaSvc, log, cfg)
	authHandler := httpadapter.NewAuthHandler(authSvc, log)
	quotaHandler := httpadapter.NewQuotaHandler(quotaSvc, log)
	return httpadapter.NewRouter(handler, authHandler, quotaHandler, tokens, cfg, log), nil
}
