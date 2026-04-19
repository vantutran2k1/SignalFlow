package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/vantutran2k1/SignalFlow/internal/config"
	"github.com/vantutran2k1/SignalFlow/internal/database"
	"github.com/vantutran2k1/SignalFlow/internal/domain"
	"github.com/vantutran2k1/SignalFlow/internal/executor"
	"github.com/vantutran2k1/SignalFlow/internal/handler"
	"github.com/vantutran2k1/SignalFlow/internal/repository/postgres"
	"github.com/vantutran2k1/SignalFlow/internal/scheduler"
	"github.com/vantutran2k1/SignalFlow/internal/service"
)

const shutdownTimeout = 10 * time.Second

type App struct {
	cfg       *config.Config
	pool      *pgxpool.Pool
	router    http.Handler
	scheduler *scheduler.Scheduler
}

func New(ctx context.Context, cfg *config.Config) (*App, error) {
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	slog.Info("connected to database")

	userRepo := postgres.NewUserRepository(pool)
	jobRepo := postgres.NewJobRepository(pool)
	channelRepo := postgres.NewChannelRepository(pool)
	execRepo := postgres.NewExecutionRepository(pool)
	notifRepo := postgres.NewNotificationRepository(pool)

	handlers := handler.Handlers{
		Auth:         handler.NewAuthHandler(service.NewAuthService(userRepo, cfg.JWTSecret)),
		Job:          handler.NewJobHandler(service.NewJobService(jobRepo)),
		Channel:      handler.NewChannelHandler(service.NewChannelService(channelRepo)),
		Execution:    handler.NewExecutionHandler(service.NewExecutionService(execRepo)),
		Notification: handler.NewNotificationHandler(service.NewNotificationService(notifRepo)),
	}

	executors := map[domain.JobType]executor.Executor{
		domain.JobTypeHTTPCheck: executor.NewHTTPCheck(),
		domain.JobTypeCommand:   executor.NewCommand(),
	}
	sched := scheduler.New(jobRepo, execRepo, executors, cfg.WorkerCount, slog.Default())

	return &App{
		cfg:       cfg,
		pool:      pool,
		router:    handler.NewRouter(cfg.JWTSecret, handlers),
		scheduler: sched,
	}, nil
}

func (a *App) Close() {
	if a.pool != nil {
		a.pool.Close()
	}
}

func (a *App) Run(ctx context.Context) error {
	g, gctx := errgroup.WithContext(ctx)

	srv := &http.Server{Addr: a.cfg.ListenAddr, Handler: a.router}

	g.Go(func() error {
		slog.Info("starting server", "addr", a.cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	g.Go(func() error {
		<-gctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	})

	g.Go(func() error {
		return a.scheduler.Run(gctx)
	})

	return g.Wait()
}
