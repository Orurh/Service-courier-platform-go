package middleware

import (
	"log"
	"net/http"
	"strconv"
	"time"

	// "strconv"
	// "time"

	// "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
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
		[]string{"path"},
	)
)

// init регистрируем метрики
func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)
}

// Observability - middleware for prometheus
func Observability(logger *log.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = log.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor) // через прокси читаем ответ
			next.ServeHTTP(ww, r)                              // пропускаем дальше
			path := pathPattern(r)                             // что бы не взорвать прометеус
			tm := time.Since(start).Seconds()

			httpRequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(ww.Status())).Inc()
			httpRequestDuration.WithLabelValues(path).Observe(tm)

			logger.Printf("[INFO] method=%s path=%s status=%d duration=%v", r.Method, path, ww.Status(), tm)
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
