package main

import (
	"log/slog"
	"os"

	"github.com/vantutran2k1/SignalFlow/internal/config"
	"github.com/vantutran2k1/SignalFlow/internal/database"
	"github.com/vantutran2k1/SignalFlow/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	setupLogger(cfg)

	if err := database.Migrate(cfg.DatabaseURL, migrations.FS); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	slog.Info("migrations applied")
}

func setupLogger(cfg *config.Config) {
	var handler slog.Handler
	if cfg.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, nil)
	} else {
		handler = slog.NewTextHandler(os.Stdout, nil)
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
