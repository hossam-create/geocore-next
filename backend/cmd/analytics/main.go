// Binary analytics runs the data intelligence pipeline.
// Phase 4 — Data Intelligence Layer.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/geocore-next/backend/pkg/tracing"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	slog.Info("analytics: starting")

	metrics.Init()
	tracing.Init()

	// Analytics pipeline will be wired here in Phase 4.
	// For now, block until signal.

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("analytics: shutting down")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	tracing.Shutdown(shutdownCtx)
}
