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
	"github.com/vantutran2k1/SignalFlow/internal/notifier"
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

	dispatcher := notifier.NewDispatcher(channelRepo, notifRepo, slog.Default())
	dispatcher.Register(notifier.NewWebhookNotifier())
	dispatcher.Register(notifier.NewSlackNotifier())
	dispatcher.Register(notifier.NewDiscordNotifier())
	dispatcher.Register(notifier.NewTelegramNotifier())
	dispatcher.Register(notifier.NewEmailNotifier())

	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
	jobSvc := service.NewJobService(jobRepo)
	channelSvc := service.NewChannelService(channelRepo)
	execSvc := service.NewExecutionService(execRepo)
	notifSvc := service.NewNotificationService(notifRepo)
	dashSvc := service.NewDashboardService(jobRepo, execRepo, notifRepo)

	handlers := handler.Handlers{
		Auth:         handler.NewAuthHandler(authSvc),
		Job:          handler.NewJobHandler(jobSvc),
		Channel:      handler.NewChannelHandler(channelSvc),
		Execution:    handler.NewExecutionHandler(execSvc),
		Notification: handler.NewNotificationHandler(notifSvc),
		Session:      handler.NewSessionHandler(authSvc, cfg.JWTSecret),
		Dashboard:    handler.NewDashboardHandler(jobSvc, channelSvc, execSvc, dashSvc),
	}

	executors := map[domain.JobType]executor.Executor{
		domain.JobTypeHTTPCheck: executor.NewHTTPCheck(),
		domain.JobTypeCommand:   executor.NewCommand(),
	}
	sched := scheduler.New(jobRepo, execRepo, executors, dispatcher, cfg.WorkerCount, slog.Default())

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
