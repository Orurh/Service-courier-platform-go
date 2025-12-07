package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.uber.org/dig"

	"course-go-avito-Orurh/internal/config"
	"course-go-avito-Orurh/internal/http/handlers"
)

func newTestLogger() *log.Logger {
	return log.New(log.Writer(), "", 0)
}

func setupTestContainer(t *testing.T) *dig.Container {
	t.Helper()

	c := dig.New()

	providers := []struct {
		name     string
		provider any
	}{
		{"context", func() context.Context { return context.Background() }},
		{"logger", func() *log.Logger { return newTestLogger() }},
		{"config", func() *config.Config { return &config.Config{Port: 8080} }},
		{"pgxpool", func() *pgxpool.Pool { return &pgxpool.Pool{} }},
	}

	for _, p := range providers {
		err := c.Provide(p.provider)
		require.NoErrorf(t, err, "provide %s", p.name)
	}

	require.NoError(t, registerService(c))
	require.NoError(t, registerHTTP(c))

	return c
}

func verifyServer(t *testing.T, srv *http.Server) {
	t.Helper()

	require.NotNil(t, srv, "http.Server is nil")
	require.Equal(t, ":8080", srv.Addr)
	require.Greater(t, srv.ReadHeaderTimeout, time.Duration(0))
	require.Greater(t, srv.ReadTimeout, time.Duration(0))
	require.Greater(t, srv.WriteTimeout, time.Duration(0))
	require.Greater(t, srv.IdleTimeout, time.Duration(0))
}

func verifyHandlers(t *testing.T,
	base *handlers.Handlers,
	courierHandler *handlers.CourierHandler,
	deliveryHandler *handlers.DeliveryHandler,
) {
	t.Helper()

	require.NotNil(t, base)
	require.NotNil(t, courierHandler)
	require.NotNil(t, deliveryHandler)
}

func TestRegisterServiceAndHTTP_ProvidesHttpServerAndHandlers(t *testing.T) {
	t.Parallel()

	c := setupTestContainer(t)

	err := c.Invoke(func(
		srv *http.Server,
		base *handlers.Handlers,
		courierHandler *handlers.CourierHandler,
		deliveryHandler *handlers.DeliveryHandler,
	) {
		verifyServer(t, srv)
		verifyHandlers(t, base, courierHandler, deliveryHandler)
	})
	require.NoError(t, err)
}

func TestProvideAll_Success(t *testing.T) {
	t.Parallel()

	c := dig.New()

	err := provideAll(c,
		func() context.Context { return context.Background() },
		func() time.Duration { return 3 * time.Second },
	)
	require.NoError(t, err)

	err = c.Invoke(func(ctx context.Context, d time.Duration) {
		require.NotNil(t, ctx)
		require.Equal(t, 3*time.Second, d)
	})
	require.NoError(t, err)
}

func TestProvideAll_InvalidProvider(t *testing.T) {
	t.Parallel()

	c := dig.New()

	type bad struct{}
	err := provideAll(c, bad{})
	require.Error(t, err)
}

func TestRegisterCore_ProvidesDependencies(t *testing.T) {
	t.Parallel()

	c := dig.New()
	ctx := context.Background()

	err := registerCore(c, ctx)
	require.NoError(t, err)

	err = c.Invoke(func(
		gotCtx context.Context,
		logger *log.Logger,
		cfg *config.Config,
		interval autoReleaseInterval,
	) {
		require.Equal(t, ctx, gotCtx)
		require.NotNil(t, logger)
		require.NotNil(t, cfg)
		require.Equal(t, autoReleaseInterval(cfg.Delivery.AutoReleaseInterval), interval)
	})
	require.NoError(t, err)
}

func TestRegisterDb_UsesDbConnectAndProvidesPool(t *testing.T) {
	t.Parallel()

	c := dig.New()
	ctx := context.Background()

	cfg := &config.Config{
		DB: config.DB{
			Host: "localhost",
			Port: "5432",
			User: "user",
			Pass: "pass",
			Name: "db",
		},
	}

	require.NoError(t, c.Provide(func() context.Context { return ctx }))
	require.NoError(t, c.Provide(func() *config.Config { return cfg }))

	stubPool := &pgxpool.Pool{}

	stubConnect := func(
		gotCtx context.Context,
		dsn string,
		retries int,
		delay time.Duration,
	) (*pgxpool.Pool, error) {
		require.Equal(t, ctx, gotCtx)
		require.Equal(t, cfg.DB.DSN(), dsn)
		require.Equal(t, 10, retries)
		require.Equal(t, time.Second, delay)
		return stubPool, nil
	}

	err := registerDb(c, stubConnect)
	require.NoError(t, err)

	err = c.Invoke(func(pool *pgxpool.Pool) {
		require.Equal(t, stubPool, pool)
	})
	require.NoError(t, err)
}

func TestContainerBuilder_Build_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	builder := NewContainerBuilder().
		WithDBConnect(func(context.Context, string, int, time.Duration) (*pgxpool.Pool, error) {
			return &pgxpool.Pool{}, nil
		})

	c, err := builder.build(ctx)
	require.NoError(t, err)
	require.NotNil(t, c)

	err = c.Invoke(func(pool *pgxpool.Pool) {
		require.NotNil(t, pool)
	})
	require.NoError(t, err)
}

func TestContainerBuilder_Build_DBError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	builder := NewContainerBuilder().
		WithDBConnect(func(context.Context, string, int, time.Duration) (*pgxpool.Pool, error) {
			return nil, fmt.Errorf("db failed")
		})

	c, err := builder.build(ctx)
	require.NoError(t, err)
	require.NotNil(t, c)

	err = c.Invoke(func(pool *pgxpool.Pool) {
		_ = pool
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "db failed")
}

func TestContainerBuilder_MustBuild_LogsFatalOnError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	builder := NewContainerBuilder().
		WithDBConnect(func(context.Context, string, int, time.Duration) (*pgxpool.Pool, error) {
			return &pgxpool.Pool{}, nil
		}).
		WithLogFatalf(func(format string, args ...interface{}) {
			require.FailNowf(t, "logFatalf must not be called", format, args...)
		})

	c := builder.MustBuild(ctx)
	require.NotNil(t, c)
}
