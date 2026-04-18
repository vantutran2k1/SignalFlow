package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vantutran2k1/SignalFlow/internal/domain"
	"github.com/vantutran2k1/SignalFlow/internal/service"
)

type ChannelHandler struct {
	svc *service.ChannelService
}

func NewChannelHandler(svc *service.ChannelService) *ChannelHandler {
	return &ChannelHandler{svc: svc}
}

func (h *ChannelHandler) List(w http.ResponseWriter, r *http.Request) {
	channels, err := h.svc.List(r.Context(), UserIDFromContext(r.Context()))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"channels": channels})
}

func (h *ChannelHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string             `json:"name"`
		Type    domain.ChannelType `json:"type"`
		Config  json.RawMessage    `json:"config"`
		Enabled *bool              `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	ch, err := h.svc.Create(r.Context(), service.CreateChannelInput{
		UserID:  UserIDFromContext(r.Context()),
		Name:    req.Name,
		Type:    req.Type,
		Config:  req.Config,
		Enabled: enabled,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, ch)
}

func (h *ChannelHandler) Get(w http.ResponseWriter, r *http.Request) {
	ch, err := h.svc.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, ch)
}

func (h *ChannelHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    *string             `json:"name"`
		Type    *domain.ChannelType `json:"type"`
		Config  *json.RawMessage    `json:"config"`
		Enabled *bool               `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	ch, err := h.svc.Update(r.Context(), chi.URLParam(r, "id"), service.UpdateChannelInput{
		Name:    req.Name,
		Type:    req.Type,
		Config:  req.Config,
		Enabled: req.Enabled,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, ch)
}

func (h *ChannelHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ChannelHandler) TestNotification(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotImplemented, "not implemented yet")
}
