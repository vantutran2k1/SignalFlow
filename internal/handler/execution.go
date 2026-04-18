package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/vantutran2k1/SignalFlow/internal/service"
)

type ExecutionHandler struct {
	svc *service.ExecutionService
}

func NewExecutionHandler(svc *service.ExecutionService) *ExecutionHandler {
	return &ExecutionHandler{svc: svc}
}

func (h *ExecutionHandler) Get(w http.ResponseWriter, r *http.Request) {
	exec, err := h.svc.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, exec)
}

func (h *ExecutionHandler) ListByJob(w http.ResponseWriter, r *http.Request) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	execs, total, err := h.svc.ListByJob(r.Context(), chi.URLParam(r, "id"), offset, limit)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"executions": execs, "total": total})
}

func (h *ExecutionHandler) ListRecent(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	execs, err := h.svc.ListRecent(r.Context(), limit)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"executions": execs})
}
