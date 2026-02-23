package app

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.uber.org/dig"

	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/logx"
	"course-go-avito-Orurh/internal/service/delivery"
	testlog "course-go-avito-Orurh/internal/testutil"
	"course-go-avito-Orurh/internal/transport/kafka"
)

type fakeDeliveryRepo struct {
	mu           sync.Mutex
	releaseCalls int
	txCalls      int
}

func hasMsg(entries []testlog.Entry, msg string) bool {
	for _, e := range entries {
		if e.Msg == msg {
			return true
		}
	}
	return false
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

// requireEventually - делаем проверку, пока она не будет пройдена или не истекнет таймаут, для защиты в CI от флаков
// вдруг у нас планировщик не успеет
func requireEventually(t *testing.T, timeout time.Duration, tick time.Duration, condition func() bool, msgAndArgs ...any) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		if condition() {
			return
		}
		if time.Now().After(deadline) {
			if len(msgAndArgs) > 0 {
				t.Fatalf(msgAndArgs[0].(string), msgAndArgs[1:]...)
			}
			t.Fatalf("condition not satisfied within %s", timeout)
		}
		<-ticker.C
	}
}

func TestStartAutoReleaseLoop_CallsReleaseExpired(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := &fakeDeliveryRepo{}
	logger := logx.Nop()
	svc := delivery.NewDeliveryService(repo, fakeTimeFactory{}, time.Second, logger)

	startAutoReleaseLoop(ctx, logger, svc, 10*time.Millisecond)

	requireEventually(
		t,
		500*time.Millisecond,
		5*time.Millisecond,
		func() bool { return repo.ReleaseCalls() > 0 },
		"expected ReleaseExpired to be called at least once",
	)
	cancel()
}

func TestGracefulShutdown_DoesNotPanic(t *testing.T) {
	t.Parallel()

	srv := &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: http.NewServeMux(),
	}
	logger := logx.Nop()

	require.NotPanics(t, func() {
		gracefulShutdown(srv, logger, 100*time.Millisecond)
	})
}

func TestMustRun_ShutdownRequested(t *testing.T) {
	t.Parallel()

	rec := testlog.New()
	container := dig.New()
	require.NoError(t, container.Provide(func() logx.Logger {
		return rec.Logger()
	}))

	r := &Runner{
		runFn: func(_ *dig.Container) error {
			return context.Canceled
		},
	}
	r.MustRun(container)
	require.True(t, hasMsg(rec.Entries(), "shutdown requested, exiting"))
}

func TestRunner_MustRun_StartupTimeout(t *testing.T) {
	t.Parallel()

	rec := testlog.New()
	container := dig.New()
	require.NoError(t, container.Provide(func() logx.Logger {
		return rec.Logger()
	}))

	r := &Runner{
		runFn: func(_ *dig.Container) error {
			return context.DeadlineExceeded
		},
	}

	r.MustRun(container)
	require.True(t, hasMsg(rec.Entries(), "startup aborted: startup timeout exceeded"))
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

	require.NoError(t, container.Provide(func() logx.Logger {
		return logx.Nop()
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

	require.NoError(t, container.Provide(func(logger logx.Logger) *delivery.Service {
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
