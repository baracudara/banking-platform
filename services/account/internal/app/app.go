// Package app wires the service together: it owns every resource that has a
// lifecycle (the database pool, the gRPC server) so that main only has to
// start it and stop it.
package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	grpcapp "banking/services/account/internal/app/grpc"
	"banking/services/account/internal/config"
	"banking/services/account/internal/repository"
	"banking/services/account/internal/service"
)

type App struct {
	GRPCServer *grpcapp.App

	log  *slog.Logger
	pool *pgxpool.Pool
}

func New(ctx context.Context, log *slog.Logger, cfg *config.Config) (*App, error) {
	pool, err := repository.NewPool(
		ctx,
		cfg.Postgres.DSN(),
		cfg.Postgres.MinConns,
		cfg.Postgres.MaxConns,
		cfg.Postgres.ConnectTimeout,
	)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}

	repo := repository.NewAccountRepository(pool)
	svc := service.NewAccountService(repo)

	return &App{
		GRPCServer: grpcapp.New(log, svc, cfg.GRPC.Port, cfg.GRPC.ShutdownTimeout, cfg.ReflectionEnabled()),
		log:        log,
		pool:       pool,
	}, nil
}

// Close releases resources after the server has stopped accepting traffic.
// Order matters: the pool is closed only once no handler can still be using
// a connection, otherwise in-flight queries fail during a shutdown that was
// supposed to be graceful.
func (a *App) Close() {
	a.log.Info("closing database pool")
	a.pool.Close()
}
