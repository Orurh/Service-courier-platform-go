package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// Handlers holds HTTP handlers dependencies (logger, etc.).
type Handlers struct {
	Logger *log.Logger
}

// New creates a Handlers instance with the given logger (or default if nil).
func New(logger *log.Logger) *Handlers {
	if logger == nil {
		logger = log.Default()
	}
	return &Handlers{Logger: logger}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// Ping handles GET /ping and returns 200 with {"message":"pong"}.
func (h *Handlers) Ping(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "pong"})
}

// HealthcheckHead handles HEAD /healthcheck and returns 204 No Content.
func (h *Handlers) HealthcheckHead(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// NotFound returns a JSON 404 error for unknown routes.
func (h *Handlers) NotFound(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotFound, "route not found")
}
