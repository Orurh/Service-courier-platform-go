package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

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

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if nextCalled != 1 {
		t.Fatalf("expected next called once, got %d", nextCalled)
	}
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

	if nextCalled != 0 {
		t.Fatalf("expected next not called, got %d", nextCalled)
	}
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type=application/json, got %q", ct)
	}
	if ra := w.Header().Get("Retry-After"); ra != "1" {
		t.Fatalf("expected Retry-After=1, got %q", ra)
	}
	if body := w.Body.String(); body != `{"error":"too many requests"}` {
		t.Fatalf("unexpected body: %q", body)
	}

	if got := testutil.ToFloat64(counter); got != 1 {
		t.Fatalf("expected counter=1, got %v", got)
	}
}
