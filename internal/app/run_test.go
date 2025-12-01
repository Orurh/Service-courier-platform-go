package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.uber.org/dig"

	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/service/delivery"
)

type fakeDeliveryRepo struct {
	mu           sync.Mutex
	releaseCalls int
}

func (f *fakeDeliveryRepo) WithTx(ctx context.Context, fn func(delivery.TxRepository) error) error {
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

type fakeTimeFactory struct{}

func (fakeTimeFactory) Deadline(_ domain.CourierTransportType, assignedAt time.Time) (time.Time, error) {
	return assignedAt.Add(time.Hour), nil
}

func TestStartAutoReleaseLoop_CallsReleaseExpired(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := &fakeDeliveryRepo{}
	svc := delivery.NewDeliveryService(repo, fakeTimeFactory{}, time.Second)

	logger := log.New(io.Discard, "", 0)

	startAutoReleaseLoop(ctx, logger, svc, 10*time.Millisecond)

	time.Sleep(50 * time.Millisecond)
	cancel()

	require.Greater(t, repo.ReleaseCalls(), 0)
}

func TestWaitForShutdown_LogsOnCtxDone(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	done := make(chan struct{})
	go func() {
		waitForShutdown(ctx, logger)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		require.FailNow(t, "waitForShutdown did not return after cancel")
	}
	require.Contains(t, buf.String(), "shutting down service-courier")
}

func TestGracefulShutdown_DoesNotPanic(t *testing.T) {
	t.Parallel()

	srv := &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: http.NewServeMux(),
	}
	logger := log.New(io.Discard, "", 0)

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
	logger := log.New(io.Discard, "", 0)

	require.NotPanics(t, func() {
		closeResources(pool, srv, logger)
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
	logger := log.New(io.Discard, "", 0)

	repo := &fakeDeliveryRepo{}
	svc := delivery.NewDeliveryService(repo, fakeTimeFactory{}, time.Second)

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := appRun(srv, ctx, pool, logger, svc, autoReleaseInterval(10*time.Millisecond))
	require.NoError(t, err)
}

func TestMustRun_ShutdownRequested(t *testing.T) {
	t.Parallel()

	var logged string

	r := &Runner{
		runFn: func(_ *dig.Container) error {
			return context.Canceled
		},
		logPrintln: func(v ...any) {
			logged = fmt.Sprint(v...)
		},
		logFatalf: func(string, ...any) {
			require.FailNow(t, "logFatalf must not be called for context. Canceled")
		},
	}
	r.MustRun(nil)
	require.Contains(t, logged, "shutdown requested")
}

func TestRunner_MustRun_StartupTimeout(t *testing.T) {
	t.Parallel()

	var logged string

	r := &Runner{
		runFn: func(_ *dig.Container) error {
			return context.DeadlineExceeded
		},
		logPrintln: func(v ...any) {
			logged = fmt.Sprint(v...)
		},
		logFatalf: func(string, ...any) {
			require.FailNow(t, "logFatalf must not be called for context.Canceled")
		},
	}

	r.MustRun(nil)

	require.Contains(t, logged, "startup aborted: startup timeout exceeded")
}

func TestRunner_MustRun_FatalOnOtherErrors(t *testing.T) {
	t.Parallel()

	var logged string

	r := &Runner{
		runFn: func(_ *dig.Container) error {
			return errors.New("boom")
		},
		logPrintln: func(...any) {
			require.FailNow(t, "logPrintln must not be called for generic error")
		},
		logFatalf: func(format string, args ...any) {
			logged = fmt.Sprintf(format, args...)
		},
	}

	r.MustRun(nil)
	require.Equal(t, "run error: boom", logged)
}

func TestNewRunner_DefaultFields(t *testing.T) {
	t.Parallel()

	r := NewRunner()
	require.NotNil(t, r)

	require.NotNil(t, r.runFn)
	require.NotNil(t, r.logPrintln)
	require.NotNil(t, r.logFatalf)
	require.Equal(t, fmt.Sprintf("%p", run), fmt.Sprintf("%p", r.runFn))
	require.Equal(t, fmt.Sprintf("%p", log.Println), fmt.Sprintf("%p", r.logPrintln))
	require.Equal(t, fmt.Sprintf("%p", log.Fatalf), fmt.Sprintf("%p", r.logFatalf))
}

func TestRun_InvokesAppRunViaContainer(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	container := dig.New()

	require.NoError(t, container.Provide(func() context.Context {
		return ctx
	}))

	require.NoError(t, container.Provide(func() *log.Logger {
		return log.New(io.Discard, "", 0)
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

	require.NoError(t, container.Provide(func() *delivery.Service {
		repo := &fakeDeliveryRepo{}
		return delivery.NewDeliveryService(repo, fakeTimeFactory{}, time.Second)
	}))

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := run(container)
	require.NoError(t, err)
}
