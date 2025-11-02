package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"course-go-avito-Orurh/internal/http/handlers"
)

// New constructs a chi-based http.Handler with base middleware and routes.
func New(h *handlers.Handlers) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(5 * time.Second))

	r.Get("/ping", h.Ping)
	r.Method(http.MethodHead, "/healthcheck", http.HandlerFunc(h.HealthcheckHead))
	r.NotFound(http.HandlerFunc(h.NotFound))

	return r
}
