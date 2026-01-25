package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"course-go-avito-Orurh/internal/logx"
)

type stubLimiter struct {
	allow bool
}

func (s stubLimiter) Allow(string) bool { return s.allow }

func TestMiddleware_Allows_RequestPassesToNext(t *testing.T) {
	t.Parallel()

	nextCalled := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	m := New(logx.Nop(), nil, stubLimiter{allow: true})
	h := m.Handler()(next)

	r := httptest.NewRequest(http.MethodGet, "http://example/test", nil)
	r.RemoteAddr = "1.2.3.4:5678"
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code, "expected 200")
	require.Equal(t, 1, nextCalled, "expected next called once")
}

func TestMiddleware_Blocks_Returns429AndIncrementsCounter(t *testing.T) {
	t.Parallel()

	nextCalled := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled++
		w.WriteHeader(http.StatusOK)
	})

	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ratelimit_denied_total",
		Help: "denied requests",
	})

	m := New(logx.Nop(), counter, stubLimiter{allow: false})
	h := m.Handler()(next)

	r := httptest.NewRequest(http.MethodGet, "http://example/test", nil)
	r.RemoteAddr = "1.2.3.4:5678"
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	require.Equal(t, 0, nextCalled, "expected next not called")
	require.Equal(t, http.StatusTooManyRequests, w.Code, "expected 429")
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))
	require.Equal(t, "1", w.Header().Get("Retry-After"))
	require.Equal(t, `{"error":"too many requests"}`, w.Body.String())
	require.Equal(t, float64(1), testutil.ToFloat64(counter), "expected counter=1")
}
