package app

import (
	"errors"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.uber.org/dig"

	"course-go-avito-Orurh/internal/config"
	"course-go-avito-Orurh/internal/logx"
	"course-go-avito-Orurh/internal/prometrics"
)

type httpServersIn struct {
	dig.In

	Main  *http.Server
	Pprof *http.Server `name:"pprof_server" optional:"true"`
}

func setupHTTPContainerWithCfg(t *testing.T, cfg *config.Config) *dig.Container {
	t.Helper()

	c := dig.New()

	require.NoError(t, c.Provide(func() *config.Config { return cfg }))
	require.NoError(t, c.Provide(logx.Nop))
	require.NoError(t, c.Provide(func() *pgxpool.Pool { return &pgxpool.Pool{} }))
	require.NoError(t, c.Provide(func() prometheus.Counter {
		return prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rate_limit_exceeded_total_unit",
			Help: "stub",
		})
	}, dig.Name("rate_limit_exceeded_total")))

	require.NoError(t, registerDomainServices(c))
	require.NoError(t, registerHTTP(c))

	return c
}

func TestRegisterHTTP_PprofDisabled_ReturnsNilPprofServer(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Port: 8080,
		Pprof: config.PprofConfig{
			Enabled: false,
			Addr:    "0.0.0.0:6060",
		},
	}
	c := setupHTTPContainerWithCfg(t, cfg)
	err := c.Invoke(func(in httpServersIn) {
		require.NotNil(t, in.Main)
		require.Equal(t, ":8080", in.Main.Addr)
		require.Nil(t, in.Pprof)
	})
	require.NoError(t, err)
}

func TestRegisterHTTP_PprofEnabled_ProvidesPprofServer(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Port: 8080,
		Pprof: config.PprofConfig{
			Enabled: true,
			Addr:    "127.0.0.1:6060",
			User:    "u",
			Pass:    "p",
		},
	}
	c := setupHTTPContainerWithCfg(t, cfg)
	err := c.Invoke(func(in httpServersIn) {
		require.NotNil(t, in.Main)
		require.NotNil(t, in.Pprof)
		require.Equal(t, "127.0.0.1:6060", in.Pprof.Addr)
		require.NotNil(t, in.Pprof.Handler)
	})
	require.NoError(t, err)
}

func TestProvideMetrics_Success_RegistersAndReturnsCounters(t *testing.T) {
	oldReg := prometheus.DefaultRegisterer
	oldGath := prometheus.DefaultGatherer
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg
	t.Cleanup(func() {
		prometheus.DefaultRegisterer = oldReg
		prometheus.DefaultGatherer = oldGath
	})
	out, err := provideMetrics()
	require.NoError(t, err)
	require.NotNil(t, out.RateLimitExceededTotal)
	require.NotNil(t, out.GatewayRetriesTotal)
}

func TestProvideMetrics_AlreadyRegistered_ReturnsExistingCounters(t *testing.T) {
	oldReg := prometheus.DefaultRegisterer
	oldGath := prometheus.DefaultGatherer
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg
	t.Cleanup(func() {
		prometheus.DefaultRegisterer = oldReg
		prometheus.DefaultGatherer = oldGath
	})

	// те же метрики юзаем
	existingRL := prometrics.NewRateLimitExceededTotal()
	existingGR := prometrics.NewGatewayRetriesTotal()

	require.NoError(t, reg.Register(existingRL))
	require.NoError(t, reg.Register(existingGR))

	out, err := provideMetrics()
	require.NoError(t, err)

	require.Same(t, existingRL, out.RateLimitExceededTotal)
	require.Same(t, existingGR, out.GatewayRetriesTotal)
}

type errRegisterer struct{ err error }

func (e errRegisterer) Register(prometheus.Collector) error  { return e.err }
func (e errRegisterer) MustRegister(...prometheus.Collector) {}
func (e errRegisterer) Unregister(prometheus.Collector) bool { return false }

func TestProvideMetrics_RegisterError_NotAlreadyRegistered(t *testing.T) {
	oldReg := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = errRegisterer{err: errors.New("boom")}
	t.Cleanup(func() { prometheus.DefaultRegisterer = oldReg })

	_, err := provideMetrics()
	require.Error(t, err)
	require.Contains(t, err.Error(), "register rate_limit_exceeded_total")
}
