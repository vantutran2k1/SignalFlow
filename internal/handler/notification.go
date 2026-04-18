package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/vantutran2k1/SignalFlow/internal/service"
)

type NotificationHandler struct {
	svc *service.NotificationService
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

func (h *NotificationHandler) ListByExecution(w http.ResponseWriter, r *http.Request) {
	notifs, err := h.svc.ListByExecution(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"notifications": notifs})
}

func (h *NotificationHandler) ListRecent(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	notifs, err := h.svc.ListRecent(r.Context(), limit)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"notifications": notifs})
}
