package handler

import (
	"encoding/json"
	"net/http"

	"github.com/vantutran2k1/SignalFlow/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	user, err := h.svc.Register(r.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	token, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"token": token})
}