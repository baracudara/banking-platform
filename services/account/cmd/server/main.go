package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	"banking/services/account/internal/repository"
	"banking/services/account/internal/service"
	"banking/services/account/internal/transport/grpcserver"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://banking:banking@localhost:5432/account?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		logger.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	repo := repository.NewAccountRepository(pool)
	svc := service.NewAccountService(repo)
	handler := grpcserver.NewServer(svc)

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		logger.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	handler.Register(grpcServer)

	logger.Info("account service listening", "port", grpcPort)
	if err := grpcServer.Serve(lis); err != nil {
		logger.Error("grpc server error", "error", err)
		os.Exit(1)
	}
}
