package order

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	testlog "course-go-avito-Orurh/internal/testutil"
)

type fakeGateway struct {
	getByIDFn func(context.Context, string) (*Order, error)
	listFn    func(context.Context, time.Time) ([]Order, error)
}

func (f *fakeGateway) GetByID(ctx context.Context, id string) (*Order, error) {
	return f.getByIDFn(ctx, id)
}
func (f *fakeGateway) ListFrom(ctx context.Context, from time.Time) ([]Order, error) {
	return f.listFn(ctx, from)
}

type counterStub struct{ n int64 }

func (c *counterStub) Inc() { atomic.AddInt64(&c.n, 1) }
func (c *counterStub) Count() int64 {
	return atomic.LoadInt64(&c.n)
}

func TestRetryingGateway_GetByID_RetriesThenSucceeds(t *testing.T) {
	t.Parallel()

	rec := testlog.New()

	var calls int32
	next := &fakeGateway{
		getByIDFn: func(context.Context, string) (*Order, error) {
			switch atomic.AddInt32(&calls, 1) {
			case 1, 2:
				return nil, status.Error(codes.Unavailable, "unavailable")
			default:
				return &Order{ID: "42"}, nil
			}
		},
	}
	ctr := &counterStub{}
	cfg := RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   0,
		MaxDelay:    0,
	}
	g := NewRetryingGateway(next, rec.Logger(), ctr, cfg)
	if g == nil {
		t.Fatalf("expected non-nil gw")
	}
	got, err := g.GetByID(context.Background(), "42")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got == nil || got.ID != "42" {
		t.Fatalf("unexpected order: %#v", got)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
	if ctr.Count() != 2 {
		t.Fatalf("expected 2 retries, got %d", ctr.Count())
	}
}

func TestRetryingGateway_GetByID_NoRetryOnNonRetryable(t *testing.T) {
	t.Parallel()

	rec := testlog.New()

	var calls int32
	next := &fakeGateway{
		getByIDFn: func(context.Context, string) (*Order, error) {
			atomic.AddInt32(&calls, 1)
			return nil, status.Error(codes.InvalidArgument, "bad request") // не retryable
		},
	}

	ctr := &counterStub{}
	cfg := RetryConfig{MaxAttempts: 5, BaseDelay: 0, MaxDelay: 0}

	g := NewRetryingGateway(next, rec.Logger(), ctr, cfg)

	_, err := g.GetByID(context.Background(), "42")
	if err == nil {
		t.Fatal("expected error")
	}

	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
	if ctr.Count() != 0 {
		t.Fatalf("expected 0 retries, got %d", ctr.Count())
	}
}

func TestRetryingGateway_ListFrom_RetriesThenSucceeds(t *testing.T) {
	t.Parallel()

	rec := testlog.New()

	var calls int32
	next := &fakeGateway{
		listFn: func(context.Context, time.Time) ([]Order, error) {
			switch atomic.AddInt32(&calls, 1) {
			case 1:
				return nil, status.Error(codes.ResourceExhausted, "rate limit")
			default:
				return []Order{{ID: "1"}}, nil
			}
		},
	}

	ctr := &counterStub{}
	cfg := RetryConfig{MaxAttempts: 3, BaseDelay: 0, MaxDelay: 0}

	g := NewRetryingGateway(next, rec.Logger(), ctr, cfg)

	got, err := g.ListFrom(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("unexpected result: %#v", got)
	}

	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
	if ctr.Count() != 1 {
		t.Fatalf("expected 1 retry, got %d", ctr.Count())
	}
}
