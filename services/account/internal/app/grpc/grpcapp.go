// Package grpcapp owns the lifecycle of the gRPC server: construction with
// its interceptors, serving, and shutdown. Keeping it separate from main
// means the wiring is testable and main stays a thin entry point.
package grpcapp

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"banking/services/account/internal/service"
	"banking/services/account/internal/transport/grpcserver"
)

type App struct {
	log             *slog.Logger
	server          *grpc.Server
	port            int
	shutdownTimeout time.Duration
}

func New(log *slog.Logger, svc *service.AccountService, port int, shutdownTimeout time.Duration, enableReflection bool) *App {
	recoveryOpts := []recovery.Option{
		recovery.WithRecoveryHandler(func(p any) error {
			// A panic in one handler must not take the whole process down
			// and must not leak the stack trace to the caller.
			log.Error("recovered from panic", slog.Any("panic", p))
			return status.Error(codes.Internal, "internal error")
		}),
	}

	loggingOpts := []logging.Option{
		// Log the start and end of each call, not the payloads: request
		// bodies here contain account identifiers and balances.
		logging.WithLogOnEvents(logging.StartCall, logging.FinishCall),
	}

	server := grpc.NewServer(grpc.ChainUnaryInterceptor(
		recovery.UnaryServerInterceptor(recoveryOpts...),
		logging.UnaryServerInterceptor(interceptorLogger(log), loggingOpts...),
	))

	grpcserver.NewServer(svc).Register(server)

	// Off in production: reflection would let anyone who can reach the port
	// enumerate every service, method and message type this server exposes.
	if enableReflection {
		reflection.Register(server)
		log.Info("grpc reflection enabled")
	}

	return &App{
		log:             log,
		server:          server,
		port:            port,
		shutdownTimeout: shutdownTimeout,
	}
}

func (a *App) Run() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("listen on :%d: %w", a.port, err)
	}

	a.log.Info("grpc server started", slog.String("addr", lis.Addr().String()))

	if err := a.server.Serve(lis); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

// Stop drains in-flight RPCs, but only for shutdownTimeout. GracefulStop on
// its own waits indefinitely, so a single stuck client would hold the
// process open until the orchestrator SIGKILLs it - which is exactly the
// abrupt termination graceful shutdown was meant to avoid.
func (a *App) Stop() {
	a.log.Info("stopping grpc server", slog.Duration("timeout", a.shutdownTimeout))

	stopped := make(chan struct{})
	go func() {
		a.server.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		a.log.Info("grpc server stopped gracefully")
	case <-time.After(a.shutdownTimeout):
		a.log.Warn("graceful shutdown timed out, forcing stop")
		a.server.Stop()
	}
}

func interceptorLogger(l *slog.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}
