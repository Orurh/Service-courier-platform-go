package handlers

import (
	"net/http"

	"course-go-avito-Orurh/internal/logx"
)

// Handlers holds HTTP handlers dependencies (logger, etc.).
type Handlers struct {
	Logger logx.Logger
}

// New creates a Handlers instance with the given logger (or a panic).
func New(logger logx.Logger) *Handlers {
	return &Handlers{Logger: logger}
}

// Ping godoc
// @Summary Liveness probe
// @Description Returns pong response in JSON
// @Tags system
// @Produce json
// @Success 200 {object} map[string]string
// @Router /ping [get]
func (h *Handlers) Ping(w http.ResponseWriter, r *http.Request) {
	writeJSON(h.Logger, w, r, http.StatusOK, map[string]string{"message": "pong"})
}

// HealthcheckHead godoc
// @Summary Healthcheck probe
// @Description Lightweight healthcheck endpoint
// @Tags system
// @Success 204
// @Router /healthcheck [head]
func (h *Handlers) HealthcheckHead(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// NotFound returns a JSON 404 error for unknown routes.
func (h *Handlers) NotFound(w http.ResponseWriter, r *http.Request) {
	writeError(h.Logger, w, r, http.StatusNotFound, "route not found")
}
