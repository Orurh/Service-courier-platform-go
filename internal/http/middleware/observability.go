package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"

	"course-go-avito-Orurh/internal/logx"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)
)

// init регистрируем метрики
func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)
}

// Observability - middleware for prometheus
func Observability(logger logx.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor) // через прокси читаем ответ
			next.ServeHTTP(ww, r)                              // пропускаем дальше
			path := pathPattern(r)                             // что бы не взорвать прометеус))
			tm := time.Since(start)
			status := strconv.Itoa(ww.Status())

			httpRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
			httpRequestDuration.WithLabelValues(r.Method, path, status).Observe(tm.Seconds())

			logger.Info("http request",
				logx.String("method", r.Method),
				logx.String("path", path),
				logx.Int("status", ww.Status()),
				logx.Duration("duration", tm),
			)
		})
	}
}

func pathPattern(r *http.Request) string {
	rc := chi.RouteContext(r.Context())
	if rc != nil {
		if p := rc.RoutePattern(); p != "" {
			return p
		}
	}
	return r.URL.Path
}
