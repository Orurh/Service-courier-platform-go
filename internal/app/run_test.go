package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.uber.org/dig"

	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/service/delivery"
	"course-go-avito-Orurh/internal/transport/kafka"
)

func testLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

type fakeDeliveryRepo struct {
	mu           sync.Mutex
	releaseCalls int
	txCalls      int
}

func (f *fakeDeliveryRepo) WithTx(ctx context.Context, fn func(delivery.TxRepository) error) error {
	f.mu.Lock()
	f.txCalls++
	f.mu.Unlock()
	return nil
}

func (f *fakeDeliveryRepo) ReleaseCouriers(ctx context.Context, now time.Time) (int64, error) {
	f.mu.Lock()
	f.releaseCalls++
	f.mu.Unlock()
	return 0, nil
}

func (f *fakeDeliveryRepo) ReleaseCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.releaseCalls
}

func (f *fakeDeliveryRepo) TxCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.txCalls
}

type fakeTimeFactory struct{}

func (fakeTimeFactory) Deadline(_ domain.CourierTransportType, assignedAt time.Time) (time.Time, error) {
	return assignedAt.Add(time.Hour), nil
}

func TestStartAutoReleaseLoop_CallsReleaseExpired(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := &fakeDeliveryRepo{}
	logger := testLogger(io.Discard)
	svc := delivery.NewDeliveryService(repo, fakeTimeFactory{}, time.Second, logger)

	startAutoReleaseLoop(ctx, logger, svc, 10*time.Millisecond)

	time.Sleep(50 * time.Millisecond)
	cancel()

	require.Greater(t, repo.ReleaseCalls(), 0)
}

func TestGracefulShutdown_DoesNotPanic(t *testing.T) {
	t.Parallel()

	srv := &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: http.NewServeMux(),
	}
	logger := testLogger(io.Discard)

	require.NotPanics(t, func() {
		gracefulShutdown(srv, logger, 100*time.Millisecond)
	})
}

func TestCloseResources_DoesNotPanic(t *testing.T) {
	t.Parallel()

	srv := &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: http.NewServeMux(),
	}
	var pool *pgxpool.Pool
	logger := testLogger(io.Discard)

	require.NotPanics(t, func() {
		closeResources(pool, srv, logger, nil, nil)
	})
}

func TestAppRun_FullFlow_WithCancelledCtx(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: http.NewServeMux(),
	}
	var pool *pgxpool.Pool
	logger := testLogger(io.Discard)

	repo := &fakeDeliveryRepo{}
	svc := delivery.NewDeliveryService(repo, fakeTimeFactory{}, time.Second, logger)

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := appRun(srv, ctx, pool, logger, svc, autoReleaseInterval(10*time.Millisecond), nil, nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestMustRun_ShutdownRequested(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	container := dig.New()
	require.NoError(t, container.Provide(func() *slog.Logger {
		return testLogger(&buf)
	}))

	r := &Runner{
		runFn: func(_ *dig.Container) error {
			return context.Canceled
		},
	}
	r.MustRun(container)
	require.Contains(t, buf.String(), "shutdown requested, exiting")
}

func TestRunner_MustRun_StartupTimeout(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	container := dig.New()
	require.NoError(t, container.Provide(func() *slog.Logger {
		return testLogger(&buf)
	}))

	r := &Runner{
		runFn: func(_ *dig.Container) error {
			return context.DeadlineExceeded
		},
	}

	r.MustRun(container)
	require.Contains(t, buf.String(), "startup aborted: startup timeout exceeded")
}

func TestNewRunner_DefaultFields(t *testing.T) {
	t.Parallel()

	r := NewRunner()
	require.NotNil(t, r)

	require.NotNil(t, r.runFn)
	require.Equal(t, fmt.Sprintf("%p", run), fmt.Sprintf("%p", r.runFn))
}

func TestRun_InvokesAppRunViaContainer(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	container := dig.New()

	require.NoError(t, container.Provide(func() context.Context {
		return ctx
	}))

	require.NoError(t, container.Provide(func() *slog.Logger {
		return testLogger(io.Discard)
	}))

	require.NoError(t, container.Provide(func() *pgxpool.Pool {
		return nil
	}))

	require.NoError(t, container.Provide(func() *http.Server {
		return &http.Server{
			Addr:    "127.0.0.1:0",
			Handler: http.NewServeMux(),
		}
	}))

	require.NoError(t, container.Provide(func() autoReleaseInterval {
		return autoReleaseInterval(10 * time.Millisecond)
	}))

	require.NoError(t, container.Provide(func(logger *slog.Logger) *delivery.Service {
		repo := &fakeDeliveryRepo{}
		return delivery.NewDeliveryService(repo, fakeTimeFactory{}, time.Second, logger)
	}))

	require.NoError(t, container.Provide(func() *kafka.Consumer {
		return nil
	}))

	require.NoError(t, container.Provide(func() ordersConnCloser {
		return func() error { return nil }
	}))

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := run(container)
	require.ErrorIs(t, err, context.Canceled)
}
