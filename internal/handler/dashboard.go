package handler

import (
	"encoding/json"
	"errors"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
	"github.com/vantutran2k1/SignalFlow/internal/service"
	"github.com/vantutran2k1/SignalFlow/web"
)

type DashboardHandler struct {
	jobSvc   *service.JobService
	chanSvc  *service.ChannelService
	execSvc  *service.ExecutionService
	dashSvc  *service.DashboardService
	pages    map[string]*template.Template
	partials map[string]*template.Template
}

func NewDashboardHandler(
	jobSvc *service.JobService,
	chanSvc *service.ChannelService,
	execSvc *service.ExecutionService,
	dashSvc *service.DashboardService,
) *DashboardHandler {
	tmplFS, err := fs.Sub(web.FS, "templates")
	if err != nil {
		panic(err)
	}

	page := func(name string) *template.Template {
		return template.Must(template.ParseFS(tmplFS, "layout.html", name))
	}
	partial := func(name string) *template.Template {
		return template.Must(template.ParseFS(tmplFS, name))
	}

	return &DashboardHandler{
		jobSvc:  jobSvc,
		chanSvc: chanSvc,
		execSvc: execSvc,
		dashSvc: dashSvc,
		pages: map[string]*template.Template{
			"dashboard": page("dashboard.html"),
			"jobs":      page("jobs.html"),
			"jobDetail": page("job_detail.html"),
			"channels":  page("channels.html"),
		},
		partials: map[string]*template.Template{
			"recentExecutions": partial("partials/recent_executions.html"),
		},
	}
}

// ------------------------- Page handlers -------------------------

func (h *DashboardHandler) Index(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	stats, err := h.dashSvc.Stats(r.Context(), userID)
	if err != nil {
		slog.Error("dashboard stats failed", "error", err)
	}
	h.render(w, "dashboard", map[string]any{
		"Title": "Dashboard",
		"Stats": stats,
	})
}

func (h *DashboardHandler) JobsPage(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	jobs, _, err := h.jobSvc.List(r.Context(), userID, 0, 100)
	if err != nil {
		h.renderError(w, "jobs", "Failed to load jobs", nil)
		return
	}
	h.render(w, "jobs", map[string]any{
		"Title": "Jobs",
		"Jobs":  jobs,
	})
}

func (h *DashboardHandler) JobDetailPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	job, err := h.jobSvc.GetByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	execs, _, err := h.execSvc.ListByJob(r.Context(), id, 0, 50)
	if err != nil {
		slog.Error("list executions failed", "job_id", id, "error", err)
	}
	h.render(w, "jobDetail", map[string]any{
		"Title":      job.Name,
		"Job":        job,
		"Executions": execs,
	})
}

func (h *DashboardHandler) ChannelsPage(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	channels, err := h.chanSvc.List(r.Context(), userID)
	if err != nil {
		h.renderError(w, "channels", "Failed to load channels", nil)
		return
	}
	h.render(w, "channels", map[string]any{
		"Title":    "Channels",
		"Channels": channels,
	})
}

// ------------------------- Partials -------------------------

func (h *DashboardHandler) RecentExecutionsPartial(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	execs, err := h.dashSvc.RecentExecutions(r.Context(), userID, limit)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.partials["recentExecutions"].Execute(w, execs)
}

// ------------------------- Job actions -------------------------

func (h *DashboardHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderError(w, "jobs", "invalid form data", nil)
		return
	}

	cfg, err := parseFormJSON(r.FormValue("config"), "{}")
	if err != nil {
		h.renderJobsError(w, r, "invalid config JSON: "+err.Error())
		return
	}
	cond, err := parseFormJSON(r.FormValue("condition"), `{"on":"failure"}`)
	if err != nil {
		h.renderJobsError(w, r, "invalid condition JSON: "+err.Error())
		return
	}

	timeout, _ := strconv.Atoi(r.FormValue("timeout_seconds"))

	_, err = h.jobSvc.Create(r.Context(), service.CreateJobInput{
		UserID:         UserIDFromContext(r.Context()),
		Name:           r.FormValue("name"),
		Type:           domain.JobType(r.FormValue("type")),
		Schedule:       r.FormValue("schedule"),
		Config:         cfg,
		NotifyChannels: parseCSV(r.FormValue("notify_channels")),
		Condition:      cond,
		TimeoutSeconds: timeout,
	})
	if err != nil {
		h.renderJobsError(w, r, err.Error())
		return
	}
	http.Redirect(w, r, "/jobs", http.StatusSeeOther)
}

func (h *DashboardHandler) RunJob(w http.ResponseWriter, r *http.Request) {
	if _, err := h.jobSvc.TriggerRun(r.Context(), chi.URLParam(r, "id")); err != nil && !errors.Is(err, service.ErrNotFound) {
		slog.Error("trigger run failed", "error", err)
	}
	http.Redirect(w, r, "/jobs", http.StatusSeeOther)
}

func (h *DashboardHandler) PauseJob(w http.ResponseWriter, r *http.Request) {
	if _, err := h.jobSvc.Pause(r.Context(), chi.URLParam(r, "id")); err != nil && !errors.Is(err, service.ErrNotFound) {
		slog.Error("pause failed", "error", err)
	}
	http.Redirect(w, r, "/jobs", http.StatusSeeOther)
}

func (h *DashboardHandler) ResumeJob(w http.ResponseWriter, r *http.Request) {
	if _, err := h.jobSvc.Resume(r.Context(), chi.URLParam(r, "id")); err != nil && !errors.Is(err, service.ErrNotFound) {
		slog.Error("resume failed", "error", err)
	}
	http.Redirect(w, r, "/jobs", http.StatusSeeOther)
}

func (h *DashboardHandler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	if err := h.jobSvc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		slog.Error("delete job failed", "error", err)
	}
	http.Redirect(w, r, "/jobs", http.StatusSeeOther)
}

// ------------------------- Channel actions -------------------------

func (h *DashboardHandler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderError(w, "channels", "invalid form data", nil)
		return
	}
	cfg, err := parseFormJSON(r.FormValue("config"), "{}")
	if err != nil {
		h.renderChannelsError(w, r, "invalid config JSON: "+err.Error())
		return
	}

	_, err = h.chanSvc.Create(r.Context(), service.CreateChannelInput{
		UserID:  UserIDFromContext(r.Context()),
		Name:    r.FormValue("name"),
		Type:    domain.ChannelType(r.FormValue("type")),
		Config:  cfg,
		Enabled: true,
	})
	if err != nil {
		h.renderChannelsError(w, r, err.Error())
		return
	}
	http.Redirect(w, r, "/channels", http.StatusSeeOther)
}

func (h *DashboardHandler) ToggleChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ch, err := h.chanSvc.GetByID(r.Context(), id)
	if err != nil {
		http.Redirect(w, r, "/channels", http.StatusSeeOther)
		return
	}
	if _, err := h.chanSvc.Update(r.Context(), id, service.UpdateChannelInput{Enabled: new(!ch.Enabled)}); err != nil {
		slog.Error("channel toggle failed", "error", err)
	}
	http.Redirect(w, r, "/channels", http.StatusSeeOther)
}

func (h *DashboardHandler) DeleteChannel(w http.ResponseWriter, r *http.Request) {
	if err := h.chanSvc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		slog.Error("channel delete failed", "error", err)
	}
	http.Redirect(w, r, "/channels", http.StatusSeeOther)
}

// ------------------------- Helpers -------------------------

func (h *DashboardHandler) render(w http.ResponseWriter, page string, data map[string]any) {
	h.renderStatus(w, page, http.StatusOK, data)
}

func (h *DashboardHandler) renderStatus(w http.ResponseWriter, page string, status int, data map[string]any) {
	tmpl, ok := h.pages[page]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		slog.Error("template render failed", "page", page, "error", err)
	}
}

func (h *DashboardHandler) renderError(w http.ResponseWriter, page string, msg string, extra map[string]any) {
	data := map[string]any{"Error": msg, "Title": page}
	for k, v := range extra {
		data[k] = v
	}
	h.renderStatus(w, page, http.StatusBadRequest, data)
}

func (h *DashboardHandler) renderJobsError(w http.ResponseWriter, r *http.Request, msg string) {
	userID := UserIDFromContext(r.Context())
	jobs, _, _ := h.jobSvc.List(r.Context(), userID, 0, 100)
	h.renderError(w, "jobs", msg, map[string]any{"Jobs": jobs})
}

func (h *DashboardHandler) renderChannelsError(w http.ResponseWriter, r *http.Request, msg string) {
	userID := UserIDFromContext(r.Context())
	channels, _ := h.chanSvc.List(r.Context(), userID)
	h.renderError(w, "channels", msg, map[string]any{"Channels": channels})
}

func parseFormJSON(raw, fallback string) (json.RawMessage, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = fallback
	}
	if !json.Valid([]byte(raw)) {
		return nil, errors.New("not valid JSON")
	}
	return json.RawMessage(raw), nil
}

func parseCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
