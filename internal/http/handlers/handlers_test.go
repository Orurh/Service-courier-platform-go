package handlers_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"course-go-avito-Orurh/internal/http/handlers"

	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHandlers_Ping(t *testing.T) {
	t.Parallel()

	h := handlers.New(testLogger())

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rr := httptest.NewRecorder()

	h.Ping(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var body map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
	require.Equal(t, "pong", body["message"])
}

func TestHandlers_HealthcheckHead(t *testing.T) {
	t.Parallel()

	h := handlers.New(testLogger())

	req := httptest.NewRequest(http.MethodHead, "/healthcheck", nil)
	rr := httptest.NewRecorder()

	h.HealthcheckHead(rr, req)

	require.Equal(t, http.StatusNoContent, rr.Code)
	require.Empty(t, rr.Body.String(), "HEAD request should not have a body")
}

func TestHandlers_NotFound(t *testing.T) {
	t.Parallel()

	h := handlers.New(testLogger())

	req := httptest.NewRequest(http.MethodGet, "/nonexistent-route", nil)
	rr := httptest.NewRecorder()

	h.NotFound(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)

	var body map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
	require.Contains(t, body["error"], "route not found")
}

func TestHandlers_New_WithNilLogger(t *testing.T) {
	t.Parallel()

	require.PanicsWithValue(t, "handlers: logger is nil", func() {
		_ = handlers.New(nil)
	})
}
