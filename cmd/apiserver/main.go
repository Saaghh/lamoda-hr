package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/Saaghh/lamoda-hr/internal/apiserver"
	"github.com/Saaghh/lamoda-hr/internal/config"
	"github.com/Saaghh/lamoda-hr/internal/logger"
	"github.com/Saaghh/lamoda-hr/internal/service"
	"github.com/Saaghh/lamoda-hr/internal/store"
	migrate "github.com/rubenv/sql-migrate"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	cfg := config.New()

	logger.InitLogger(logger.Config{Level: cfg.LogLevel})

	// no error handling for now
	// check https://github.com/uber-go/zap/issues/991
	//nolint: errcheck
	defer zap.L().Sync()

	pgStore, err := store.New(ctx, cfg)
	if err != nil {
		zap.L().With(zap.Error(err)).Panic("main/pgStore.New(ctx, cfg)")
	}

	if err = pgStore.Migrate(migrate.Up); err != nil {
		zap.L().With(zap.Error(err)).Panic("main/pgStore.Migrate(migrate.Up)")
	}

	zap.L().Info("successful migration")

	serviceLayer := service.New(pgStore)

	server := apiserver.New(
		apiserver.Config{BindAddress: cfg.BindAddress},
		serviceLayer,
	)

	if err = server.Run(ctx); err != nil {
		zap.L().With(zap.Error(err)).Panic("main/server.Run(ctx)")
	}
}
