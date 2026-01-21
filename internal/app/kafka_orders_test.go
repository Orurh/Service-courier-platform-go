package app

import (
	"context"
	"errors"
	"testing"
	"time"

	ordersgw "course-go-avito-Orurh/internal/gateway/orders"
	"course-go-avito-Orurh/internal/service/orders"

	"github.com/stretchr/testify/require"
)

type ctxKey struct{}

type spyHandler struct {
	called int
	ctx    context.Context
	event  orders.Event
	err    error
}

func (s *spyHandler) Handle(ctx context.Context, e orders.Event) error {
	s.called++
	s.ctx = ctx
	s.event = e
	return s.err
}

type stubOrdersGateway struct {
	getFn       func(ctx context.Context, id string) (*ordersgw.Order, error)
	capturedCtx context.Context
	capturedID  string
}

func (g *stubOrdersGateway) GetByID(ctx context.Context, id string) (*ordersgw.Order, error) {
	g.capturedCtx = ctx
	g.capturedID = id
	if g.getFn == nil {
		return nil, nil
	}
	return g.getFn(ctx, id)
}

func requireTimeout2s(t *testing.T, ctx context.Context) {
	t.Helper()
	deadline, ok := ctx.Deadline()
	require.True(t, ok, "expected context with deadline")

	remaining := time.Until(deadline)
	require.Greater(t, remaining, 1*time.Second)
	require.Less(t, remaining, 3*time.Second)
}

func requireCanceled(t *testing.T, ctx context.Context) {
	t.Helper()
	select {
	case <-ctx.Done():
	default:
		t.Fatalf("expected gateway context to be canceled after handler returns")
	}
}

func TestMakeOrdersKafka_NoGateway_DelegatesToHandler(t *testing.T) {
	t.Parallel()

	hSpy := &spyHandler{}
	h := makeOrdersKafka(hSpy, nil)

	ctx := context.WithValue(context.Background(), ctxKey{}, "v")
	in := orders.Event{OrderID: "order-1", Status: "created"}

	err := h(ctx, in)
	require.NoError(t, err)

	require.Equal(t, 1, hSpy.called)
	require.Equal(t, "v", hSpy.ctx.Value(ctxKey{}))
	require.Equal(t, in, hSpy.event)
}

func TestMakeOrdersKafka_GatewayError_ReturnsError_AndDoesNotCallHandler(t *testing.T) {
	t.Parallel()

	hSpy := &spyHandler{}

	sentinel := errors.New("gw boom")
	gw := &stubOrdersGateway{
		getFn: func(ctx context.Context, id string) (*ordersgw.Order, error) {
			return nil, sentinel
		},
	}

	h := makeOrdersKafka(hSpy, gw)

	ctx := context.WithValue(context.Background(), ctxKey{}, "v")
	err := h(ctx, orders.Event{OrderID: "order-2", Status: "created"})
	require.ErrorIs(t, err, sentinel)

	require.Equal(t, 0, hSpy.called)

	require.Equal(t, "order-2", gw.capturedID)
	requireTimeout2s(t, gw.capturedCtx)
	requireCanceled(t, gw.capturedCtx)
}

func TestMakeOrdersKafka_OrderNotFound_ReturnsNil_AndDoesNotCallHandler(t *testing.T) {
	t.Parallel()

	hSpy := &spyHandler{}
	gw := &stubOrdersGateway{
		getFn: func(ctx context.Context, id string) (*ordersgw.Order, error) {
			return nil, nil
		},
	}

	h := makeOrdersKafka(hSpy, gw)

	ctx := context.WithValue(context.Background(), ctxKey{}, "v")
	err := h(ctx, orders.Event{OrderID: "order-3", Status: "created"})
	require.NoError(t, err)

	require.Equal(t, 0, hSpy.called)

	require.Equal(t, "order-3", gw.capturedID)
	requireTimeout2s(t, gw.capturedCtx)
	requireCanceled(t, gw.capturedCtx)
}

func TestMakeOrdersKafka_OrderFound_OverridesEvent_AndCallsHandler(t *testing.T) {
	t.Parallel()

	hSpy := &spyHandler{}

	ts := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	gw := &stubOrdersGateway{
		getFn: func(ctx context.Context, id string) (*ordersgw.Order, error) {
			return &ordersgw.Order{
				ID:        id,
				Status:    "canceled",
				CreatedAt: ts,
			}, nil
		},
	}

	h := makeOrdersKafka(hSpy, gw)

	ctx := context.WithValue(context.Background(), ctxKey{}, "v")
	in := orders.Event{OrderID: "order-4", Status: "created"}

	err := h(ctx, in)
	require.NoError(t, err)

	require.Equal(t, 1, hSpy.called)
	require.Equal(t, "v", hSpy.ctx.Value(ctxKey{}))

	require.Equal(t, "order-4", hSpy.event.OrderID)
	require.Equal(t, "canceled", hSpy.event.Status)
	require.Equal(t, ts, hSpy.event.CreatedAt)

	require.Equal(t, "order-4", gw.capturedID)
	requireTimeout2s(t, gw.capturedCtx)
	requireCanceled(t, gw.capturedCtx)
}
