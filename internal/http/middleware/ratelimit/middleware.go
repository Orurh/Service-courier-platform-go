package ratelimit

import (
	"io"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"

	"course-go-avito-Orurh/internal/logx"
)

// Middleware представляет собой middleware для ограничения количества запросов
type Middleware struct {
	logger  logx.Logger        // логгер
	counter prometheus.Counter // счетчик
	limiter Limiter            // лимитер
}

// New создает новый Middleware
func New(logger logx.Logger, counter prometheus.Counter, limiter Limiter) *Middleware {
	if limiter == nil {
		limiter = NopLimiter{}
	}
	return &Middleware{
		logger:  logger,
		counter: counter,
		limiter: limiter,
	}
}

// Handler returns chi-style middleware.
func (m *Middleware) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)

			if !m.limiter.Allow(ip) {
				// считаю отказы
				if m.counter != nil {
					m.counter.Inc()
				}
				m.logger.Warn("rate limit exceeded",
					logx.String("ip", ip),
					logx.String("method", r.Method),
					logx.String("path", r.URL.Path),
				)
				// отвечаю
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "1")
				// ошибка 429
				w.WriteHeader(http.StatusTooManyRequests)
				// сообщение о том, что слишком много запросов
				if _, err := io.WriteString(w, `{"error":"too many requests"}`); err != nil {
					// клиент мог оборвать соединение; это не ошибка бизнес-логики
					m.logger.Debug("rate limit response write failed",
						logx.String("ip", ip),
						logx.Any("err", err),
					)
				}
				// не вызываю next мы уже ответили
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	// пока без нормализации
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "unknown"
}
