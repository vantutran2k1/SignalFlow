package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/vantutran2k1/SignalFlow/internal/domain"
	"github.com/vantutran2k1/SignalFlow/internal/service"
)

type JobHandler struct {
	svc *service.JobService
}

func NewJobHandler(svc *service.JobService) *JobHandler {
	return &JobHandler{svc: svc}
}

func (h *JobHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	jobs, total, err := h.svc.List(r.Context(), userID, offset, limit)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"jobs": jobs, "total": total})
}

func (h *JobHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name           string          `json:"name"`
		Type           domain.JobType  `json:"type"`
		Schedule       string          `json:"schedule"`
		Config         json.RawMessage `json:"config"`
		NotifyChannels []string        `json:"notify_channels"`
		Condition      json.RawMessage `json:"condition"`
		TimeoutSeconds int             `json:"timeout_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	job, err := h.svc.Create(r.Context(), service.CreateJobInput{
		UserID:         UserIDFromContext(r.Context()),
		Name:           req.Name,
		Type:           req.Type,
		Schedule:       req.Schedule,
		Config:         req.Config,
		NotifyChannels: req.NotifyChannels,
		Condition:      req.Condition,
		TimeoutSeconds: req.TimeoutSeconds,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, job)
}

func (h *JobHandler) Get(w http.ResponseWriter, r *http.Request) {
	job, err := h.svc.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, job)
}

func (h *JobHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name           *string          `json:"name"`
		Schedule       *string          `json:"schedule"`
		Config         *json.RawMessage `json:"config"`
		NotifyChannels *[]string        `json:"notify_channels"`
		Condition      *json.RawMessage `json:"condition"`
		TimeoutSeconds *int             `json:"timeout_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	job, err := h.svc.Update(r.Context(), chi.URLParam(r, "id"), service.UpdateJobInput{
		Name:           req.Name,
		Schedule:       req.Schedule,
		Config:         req.Config,
		NotifyChannels: req.NotifyChannels,
		Condition:      req.Condition,
		TimeoutSeconds: req.TimeoutSeconds,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, job)
}

func (h *JobHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *JobHandler) TriggerRun(w http.ResponseWriter, r *http.Request) {
	if _, err := h.svc.TriggerRun(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "triggered"})
}

func (h *JobHandler) Pause(w http.ResponseWriter, r *http.Request) {
	job, err := h.svc.Pause(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, job)
}

func (h *JobHandler) Resume(w http.ResponseWriter, r *http.Request) {
	job, err := h.svc.Resume(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, job)
}