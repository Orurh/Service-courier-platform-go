package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/service/delivery"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/dig"
)

type fakeDeliveryRepo struct {
	mu           sync.Mutex
	releaseCalls int
}

func (f *fakeDeliveryRepo) BeginTx(ctx context.Context) (delivery.Tx, error) {
	return nil, nil
}

func (f *fakeDeliveryRepo) FindAvailableCourierForUpdate(ctx context.Context, tx delivery.Tx) (*domain.Courier, error) {
	panic("not used in tests")
}

func (f *fakeDeliveryRepo) UpdateCourierStatus(ctx context.Context, tx delivery.Tx, id int64, status string) error {
	panic("not used in tests")
}

func (f *fakeDeliveryRepo) InsertDelivery(ctx context.Context, tx delivery.Tx, d *domain.Delivery) error {
	panic("not used in tests")
}

func (f *fakeDeliveryRepo) GetByOrderID(ctx context.Context, tx delivery.Tx, orderID string) (*domain.Delivery, error) {
	panic("not used in tests")
}

func (f *fakeDeliveryRepo) DeleteByOrderID(ctx context.Context, tx delivery.Tx, orderID string) error {
	panic("not used in tests")
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

	if repo.ReleaseCalls() == 0 {
		t.Fatalf("expected ReleaseCouriers to be called at least once, got 0")
	}
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
		t.Fatal("waitForShutdown did not return after cancel")
	}

	if !bytes.Contains(buf.Bytes(), []byte("shutting down service-courier")) {
		t.Fatalf("expected shutdown log, got %q", buf.String())
	}
}

func TestGracefulShutdown_DoesNotPanic(t *testing.T) {
	t.Parallel()

	srv := &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: http.NewServeMux(),
	}
	logger := log.New(io.Discard, "", 0)

	gracefulShutdown(srv, logger, 100*time.Millisecond)
}

func TestCloseResources_DoesNotPanic(t *testing.T) {
	t.Parallel()

	srv := &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: http.NewServeMux(),
	}
	var pool *pgxpool.Pool
	logger := log.New(io.Discard, "", 0)

	closeResources(pool, srv, logger)
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

	if err := appRun(srv, ctx, pool, logger, svc); err != nil {
		t.Fatalf("appRun returned error: %v", err)
	}
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
			t.Fatalf("logFatalf must not be called for context.Canceled")
		},
	}
	r.MustRun(nil)
	if !strings.Contains(logged, "shutdown requested") {
		t.Fatalf("expected shutdown log, got %q", logged)
	}
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
			t.Fatalf("logFatalf must not be called for context.DeadlineExceeded")
		},
	}

	r.MustRun(nil)

	if !strings.Contains(logged, "startup aborted: startup timeout exceeded") {
		t.Fatalf("expected startup timeout log, got %q", logged)
	}
}

func TestRunner_MustRun_FatalOnOtherErrors(t *testing.T) {
	t.Parallel()

	var logged string

	r := &Runner{
		runFn: func(_ *dig.Container) error {
			return errors.New("boom")
		},
		logPrintln: func(...any) {
			t.Fatalf("logPrintln must not be called for generic error")
		},
		logFatalf: func(format string, args ...any) {
			logged = fmt.Sprintf(format, args...)
			//  os.Exit // убрал, чтобы не завершался
		},
	}

	r.MustRun(nil)

	if !strings.Contains(logged, "run error: boom") {
		t.Fatalf("expected fatal log with error, got %q", logged)
	}
}
