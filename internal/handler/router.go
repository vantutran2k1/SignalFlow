package handler

import (
	"io/fs"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/vantutran2k1/SignalFlow/web"
)

type Handlers struct {
	Auth         *AuthHandler
	Job          *JobHandler
	Channel      *ChannelHandler
	Execution    *ExecutionHandler
	Notification *NotificationHandler
	Session      *SessionHandler
	Dashboard    *DashboardHandler
}

func NewRouter(jwtSecret string, h Handlers) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Static assets served from embedded FS.
	staticFS, _ := fs.Sub(web.FS, "static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Public browser routes
	r.Get("/login", h.Session.ShowLogin)
	r.Post("/login", h.Session.DoLogin)
	r.Post("/logout", h.Session.Logout)

	// Dashboard (cookie auth; redirects to /login on failure)
	r.Group(func(r chi.Router) {
		r.Use(CookieAuthMiddleware(jwtSecret))

		r.Get("/", h.Dashboard.Index)
		r.Get("/jobs", h.Dashboard.JobsPage)
		r.Get("/jobs/{id}", h.Dashboard.JobDetailPage)
		r.Get("/channels", h.Dashboard.ChannelsPage)

		r.Get("/dashboard/partials/recent-executions", h.Dashboard.RecentExecutionsPartial)

		r.Post("/dashboard/jobs", h.Dashboard.CreateJob)
		r.Post("/dashboard/jobs/{id}/run", h.Dashboard.RunJob)
		r.Post("/dashboard/jobs/{id}/pause", h.Dashboard.PauseJob)
		r.Post("/dashboard/jobs/{id}/resume", h.Dashboard.ResumeJob)
		r.Post("/dashboard/jobs/{id}/delete", h.Dashboard.DeleteJob)

		r.Post("/dashboard/channels", h.Dashboard.CreateChannel)
		r.Post("/dashboard/channels/{id}/toggle", h.Dashboard.ToggleChannel)
		r.Post("/dashboard/channels/{id}/delete", h.Dashboard.DeleteChannel)
	})

	// JSON API (Bearer token auth)
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
