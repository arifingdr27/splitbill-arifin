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
	provideStorage,
	provideExtractor,
	service.NewSplitbillService,
	httpadapter.NewSplitbillHandler,
	httpadapter.NewRouter,
)

func InitializeRouter(ctx context.Context, cfg *config.Config) (*httpadapter.Router, error) {
	wire.Build(appSet)
	return nil, nil
}
