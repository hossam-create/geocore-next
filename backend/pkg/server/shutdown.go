package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/geocore-next/backend/pkg/tracing"
)

// WaitForShutdown blocks until SIGINT/SIGTERM, then performs graceful cleanup.
// cleanup receives a context with a 30-second timeout.
func WaitForShutdown(srv *http.Server, cleanup func(ctx context.Context)) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("shutdown: signal received", "signal", sig.String())

	// 30-second hard deadline for cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server (stops accepting new connections, drains in-flight)
	if srv != nil {
		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("shutdown: HTTP server error", "error", err)
		} else {
			slog.Info("shutdown: HTTP server stopped")
		}
	}

	// Flush OTEL traces before exit
	tracing.Shutdown(ctx)

	// Run custom cleanup (Kafka consumers, DB, Redis, etc.)
	if cleanup != nil {
		cleanup(ctx)
	}

	slog.Info("shutdown: complete")
}

// WaitForCancel blocks until SIGINT/SIGTERM, then calls cancel + cleanup.
// Use this for non-HTTP services (workers, fraud-engine, etc.).
func WaitForCancel(cancel context.CancelFunc, cleanup func(ctx context.Context)) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("shutdown: signal received", "signal", sig.String())

	// Cancel all contexts — signals goroutines to stop
	cancel()

	// 10-second deadline for cleanup
	ctx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cleanupCancel()

	// Flush OTEL traces
	tracing.Shutdown(ctx)

	// Run custom cleanup
	if cleanup != nil {
		cleanup(ctx)
	}

	slog.Info("shutdown: complete")
}
