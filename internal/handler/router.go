package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Handlers struct {
	Auth         *AuthHandler
	Job          *JobHandler
	Channel      *ChannelHandler
	Execution    *ExecutionHandler
	Notification *NotificationHandler
}

func NewRouter(jwtSecret string, h Handlers) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", h.Auth.Register)
		r.Post("/auth/login", h.Auth.Login)

		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(jwtSecret))

			r.Route("/jobs", func(r chi.Router) {
				r.Get("/", h.Job.List)
				r.Post("/", h.Job.Create)
				r.Get("/{id}", h.Job.Get)
				r.Put("/{id}", h.Job.Update)
				r.Delete("/{id}", h.Job.Delete)
				r.Post("/{id}/run", h.Job.TriggerRun)
				r.Post("/{id}/pause", h.Job.Pause)
				r.Post("/{id}/resume", h.Job.Resume)
				r.Get("/{id}/executions", h.Execution.ListByJob)
			})

			r.Route("/channels", func(r chi.Router) {
				r.Get("/", h.Channel.List)
				r.Post("/", h.Channel.Create)
				r.Get("/{id}", h.Channel.Get)
				r.Put("/{id}", h.Channel.Update)
				r.Delete("/{id}", h.Channel.Delete)
			})

			r.Get("/executions/recent", h.Execution.ListRecent)
			r.Get("/executions/{id}", h.Execution.Get)
			r.Get("/executions/{id}/notifications", h.Notification.ListByExecution)
			r.Get("/notifications/recent", h.Notification.ListRecent)
		})
	})

	return r
}
