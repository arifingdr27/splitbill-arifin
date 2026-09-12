//go:build wireinject
// +build wireinject

package di

import (
	"context"

	httpadapter "github.com/arifin2018/splitbill-arifin.git/internal/adapter/http"
	"github.com/arifin2018/splitbill-arifin.git/internal/config"
	"github.com/arifin2018/splitbill-arifin.git/internal/service"
	"github.com/google/wire"
)

var appSet = wire.NewSet(
	provideLogger,
	providePool,
	provideUserRepo,
	provideUsageRepo,
	provideGoogleVerifier,
	provideTokenIssuer,
	provideAuthService,
	provideQuotaService,
	provideStorage,
	provideExtractor,
	service.NewSplitbillService,
	httpadapter.NewSplitbillHandler,
	httpadapter.NewAuthHandler,
	httpadapter.NewQuotaHandler,
	httpadapter.NewRouter,
)

func InitializeRouterWire(ctx context.Context, cfg *config.Config) (*httpadapter.Router, error) {
	wire.Build(appSet)
	return nil, nil
}
