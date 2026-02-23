package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"course-go-avito-Orurh/internal/http/handlers"
	"course-go-avito-Orurh/internal/logx"
)

func testLogger() logx.Logger { return logx.Nop() }

func TestHandlers_Ping(t *testing.T) {
	t.Parallel()

	h := handlers.New(testLogger())

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rr := httptest.NewRecorder()

	h.Ping(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Header().Get("Content-Type"), "application/json")

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
	require.Contains(t, rr.Header().Get("Content-Type"), "application/json")

	var body struct {
		Error string `json:"error"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
	require.Contains(t, body.Error, "route not found")
}
