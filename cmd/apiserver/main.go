package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/Saaghh/lamoda-hr/internal/apiserver"
	"github.com/Saaghh/lamoda-hr/internal/config"
	"github.com/Saaghh/lamoda-hr/internal/logger"
	"github.com/Saaghh/lamoda-hr/internal/service"
	"github.com/Saaghh/lamoda-hr/internal/store"
	migrate "github.com/rubenv/sql-migrate"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
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

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		if err = server.Run(ctx); err != nil {
			return fmt.Errorf("server.Run(ctx): %w", err)
		}

		return nil
	})

	eg.Go(func() error {
		period, err := time.ParseDuration(cfg.DeactivatorPeriod)
		if err != nil {
			return fmt.Errorf("time.ParseDuration(cfg.DeactivatorPeriod): %w", err)
		}

		if err = serviceLayer.RunReservationsDeactivations(ctx, period); err != nil {
			return fmt.Errorf("serviceLayer.RunReservationsDeactivations(ctx, period): %w", err)
		}

		return nil
	})

	if err = eg.Wait(); err != nil {
		zap.L().With(zap.Error(err)).Panic("main/eg.Wait()")
	}
}
