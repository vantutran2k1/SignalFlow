package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/vantutran2k1/SignalFlow/internal/app"
	"github.com/vantutran2k1/SignalFlow/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	setupLogger(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	a, err := app.New(ctx, cfg)
	if err != nil {
		slog.Error("failed to initialize app", "error", err)
		os.Exit(1)
	}
	defer a.Close()

	if err := a.Run(ctx); err != nil {
		slog.Error("app terminated with error", "error", err)
		os.Exit(1)
	}
	slog.Info("shutdown complete")
}

func setupLogger(cfg *config.Config) {
	var h slog.Handler
	if cfg.LogFormat == "json" {
		h = slog.NewJSONHandler(os.Stdout, nil)
	} else {
		h = slog.NewTextHandler(os.Stdout, nil)
	}
	slog.SetDefault(slog.New(h))
}
