package middleware

import (
	"course-go-avito-Orurh/internal/logx"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestObservability_UsesRoutePatternForLabels(t *testing.T) {
	t.Parallel()
	routePrefix := "/test/" + sanitizeLabel(t.Name())
	pattern := routePrefix + "/{id}"
	r := chi.NewRouter()
	r.Use(Observability(logx.Nop()))
	r.Get(pattern, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, routePrefix+"/123", nil)
	rec := httptest.NewRecorder()

	before := testutil.ToFloat64(httpRequestsTotal.WithLabelValues(http.MethodGet, pattern, "204"))
	beforeCount := histogramCount(t, httpRequestDuration, http.MethodGet, pattern, "204")

	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNoContent, rec.Code)

	after := testutil.ToFloat64(httpRequestsTotal.WithLabelValues(http.MethodGet, pattern, "204"))
	afterCount := histogramCount(t, httpRequestDuration, http.MethodGet, pattern, "204")

	require.Equal(t, before+1, after)
	require.Equal(t, beforeCount+1, afterCount)
}

func sanitizeLabel(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "\t", "_")
	return s
}

func histogramCount(t *testing.T, hv *prometheus.HistogramVec, method, path, status string) uint64 {
	t.Helper()

	obs, err := hv.GetMetricWithLabelValues(method, path, status)
	require.NoError(t, err)

	metric, ok := obs.(prometheus.Metric)
	require.True(t, ok, "must implement prometheus.Metric")

	m := &dto.Metric{}
	err = metric.Write(m)
	require.NoError(t, err)

	h := m.GetHistogram()
	require.NotNil(t, h)
	return h.GetSampleCount()
}
