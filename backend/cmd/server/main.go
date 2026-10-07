package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dataolympus/rtsp-stream-console/backend/internal/api"
	"github.com/dataolympus/rtsp-stream-console/backend/internal/config"
	"github.com/dataolympus/rtsp-stream-console/backend/internal/health"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	healthHandler := health.New(func() bool {
		return true
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.NewRouter(healthHandler),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		logger.Info("server starting",
			"addr", cfg.HTTPAddr,
		)

		err := server.ListenAndServe()

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	stop := make(chan os.Signal, 1)

	signal.Notify(
		stop,
		os.Interrupt,
		syscall.SIGTERM,
	)

	select {
	case sig := <-stop:
		logger.Info("shutdown signal received",
			"signal", sig.String(),
		)

	case err := <-serverErrors:
		logger.Error("server failed",
			"error", err,
		)
		return
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		cfg.ShutdownTimeout,
	)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed",
			"error", err,
		)
		return
	}

	logger.Info("server stopped")
}
