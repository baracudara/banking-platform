package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"banking/services/account/internal/app"
	"banking/services/account/internal/config"
)

func main() {
	cfg := config.MustLoad()
	log := setupLogger(cfg.Env)

	log.Info("starting account service", slog.String("env", cfg.Env))

	application, err := app.New(context.Background(), log, cfg)
	if err != nil {
		log.Error("failed to start", "error", err)
		os.Exit(1)
	}

	// Serve blocks, so it runs in its own goroutine while main waits for a
	// termination signal.
	go func() {
		if err := application.GRPCServer.Run(); err != nil {
			log.Error("grpc server error", "error", err)
			os.Exit(1)
		}
	}()

	// Buffered so that a signal arriving before the receive is not dropped:
	// signal.Notify never blocks on send, it discards instead.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	sig := <-stop
	log.Info("shutdown signal received", slog.String("signal", sig.String()))

	application.GRPCServer.Stop()
	application.Close()

	log.Info("stopped")
}

func setupLogger(env string) *slog.Logger {
	switch env {
	case config.EnvProd:
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	case config.EnvDev:
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case config.EnvLocal:
		return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	default:
		return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}
}
