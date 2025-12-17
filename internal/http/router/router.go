package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"course-go-avito-Orurh/internal/http/handlers"
	obsmw "course-go-avito-Orurh/internal/http/middleware"
)

// New constructs a chi-based http.Handler with base middleware and routes.
func New(base *handlers.Handlers, cour *handlers.CourierHandler, delivery *handlers.DeliveryHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	// r.Use(middleware.Logger)
	r.Use(obsmw.Observability(base.Logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(5 * time.Second))

	r.Get("/ping", base.Ping)
	r.Get("/metrics", promhttp.Handler().ServeHTTP)
	r.Method(http.MethodHead, "/healthcheck", http.HandlerFunc(base.HealthcheckHead))
	r.NotFound(http.HandlerFunc(base.NotFound))

	r.Get("/courier/{id}", cour.GetByID)
	r.Get("/couriers", cour.List)
	r.Post("/courier", cour.Create)
	r.Put("/courier", cour.Update)

	r.Post("/delivery/assign", delivery.Assign)
	r.Post("/delivery/unassign", delivery.Unassign)
	return r
}
