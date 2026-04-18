package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/vantutran2k1/SignalFlow/internal/service"
)

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, map[string]string{"error": msg})
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		respondError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrInvalidInput):
		respondError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrInvalidCredentials):
		respondError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, service.ErrConflict):
		respondError(w, http.StatusConflict, err.Error())
	default:
		respondError(w, http.StatusInternalServerError, "internal error")
	}
}