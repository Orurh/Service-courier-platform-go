package orders_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/ports/deliverytx"
	"course-go-avito-Orurh/internal/service/orders"
)

type stubTx struct {
	getFn    func(ctx context.Context, orderID string) (*domain.Delivery, error)
	updateFn func(ctx context.Context, id int64, status domain.CourierStatus) error
}

func (s *stubTx) FindAvailableCourierForUpdate(ctx context.Context) (*domain.Courier, error) {
	panic("not used in orders processor tests")
}

func (s *stubTx) InsertDelivery(ctx context.Context, d *domain.Delivery) error {
	panic("not used in orders processor tests")
}

func (s *stubTx) DeleteByOrderID(ctx context.Context, orderID string) error {
	panic("not used in orders processor tests")
}

func (s *stubTx) GetByOrderID(ctx context.Context, orderID string) (*domain.Delivery, error) {
	if s.getFn == nil {
		return nil, nil
	}
	return s.getFn(ctx, orderID)
}

func (s *stubTx) UpdateCourierStatus(ctx context.Context, id int64, status domain.CourierStatus) error {
	if s.updateFn == nil {
		return nil
	}
	return s.updateFn(ctx, id, status)
}

type noopRunner struct{}

func (noopRunner) WithTx(ctx context.Context, fn func(tx deliverytx.Repository) error) error {
	panic("WithTx must not be called in this test")
}

type stubRunner struct {
	withTx func(ctx context.Context, fn func(tx deliverytx.Repository) error) error
}

func (s stubRunner) WithTx(ctx context.Context, fn func(tx deliverytx.Repository) error) error {
	if s.withTx == nil {
		return fn(nil)
	}
	return s.withTx(ctx, fn)
}

func TestProcessor_Handle_Created_AssignOK(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := NewMockDeliveryPort(ctrl)
	r := noopRunner{}

	p := orders.NewProcessorWithDeps(d, r)

	d.EXPECT().
		Assign(gomock.Any(), "order-1").
		Return(domain.AssignResult{}, nil)

	err := p.Handle(context.Background(), orders.Event{
		OrderID:   "order-1",
		Status:    "  CREATED  ",
		CreatedAt: time.Now(),
	})
	require.NoError(t, err)
}

func TestProcessor_Handle_Created_ConflictIsIgnored(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := NewMockDeliveryPort(ctrl)
	r := noopRunner{}

	p := orders.NewProcessorWithDeps(d, r)

	d.EXPECT().
		Assign(gomock.Any(), "order-1").
		Return(domain.AssignResult{}, apperr.ErrConflict)

	err := p.Handle(context.Background(), orders.Event{OrderID: "order-1", Status: "created"})
	require.NoError(t, err)
}

func TestProcessor_Handle_Created_OtherErrorReturned(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := NewMockDeliveryPort(ctrl)
	r := noopRunner{}

	p := orders.NewProcessorWithDeps(d, r)

	wantErr := errors.New("boom")
	d.EXPECT().
		Assign(gomock.Any(), "order-1").
		Return(domain.AssignResult{}, wantErr)

	err := p.Handle(context.Background(), orders.Event{OrderID: "order-1", Status: "created"})
	require.ErrorIs(t, err, wantErr)
}

func TestProcessor_Handle_Canceled_UnassignOK(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := NewMockDeliveryPort(ctrl)
	r := noopRunner{}

	p := orders.NewProcessorWithDeps(d, r)

	d.EXPECT().
		Unassign(gomock.Any(), "order-2").
		Return(domain.UnassignResult{}, nil)

	err := p.Handle(context.Background(), orders.Event{OrderID: "order-2", Status: "canceled"})
	require.NoError(t, err)
}

func TestProcessor_Handle_Canceled_NotFoundIsIgnored(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := NewMockDeliveryPort(ctrl)
	r := noopRunner{}

	p := orders.NewProcessorWithDeps(d, r)

	d.EXPECT().
		Unassign(gomock.Any(), "order-2").
		Return(domain.UnassignResult{}, apperr.ErrNotFound)

	err := p.Handle(context.Background(), orders.Event{OrderID: "order-2", Status: "deleted"})
	require.NoError(t, err)
}

func TestProcessor_Handle_Completed_NoDelivery_NoUpdate(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := NewMockDeliveryPort(ctrl)
	r := stubRunner{
		withTx: func(ctx context.Context, fn func(tx deliverytx.Repository) error) error {
			tx := &stubTx{
				getFn: func(ctx context.Context, orderID string) (*domain.Delivery, error) {
					require.Equal(t, "order-3", orderID)
					return nil, nil
				},
				updateFn: func(ctx context.Context, id int64, status domain.CourierStatus) error {
					t.Fatalf("UpdateCourierStatus must not be called when delivery is nil")
					return nil
				},
			}
			return fn(tx)
		},
	}

	p := orders.NewProcessorWithDeps(d, r)

	err := p.Handle(context.Background(), orders.Event{OrderID: "order-3", Status: "completed"})
	require.NoError(t, err)
}

func TestProcessor_Handle_Completed_UpdateCourierStatus(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := NewMockDeliveryPort(ctrl)
	r := stubRunner{
		withTx: func(ctx context.Context, fn func(tx deliverytx.Repository) error) error {
			tx := &stubTx{
				getFn: func(ctx context.Context, orderID string) (*domain.Delivery, error) {
					return &domain.Delivery{OrderID: orderID, CourierID: 42}, nil
				},
				updateFn: func(ctx context.Context, id int64, status domain.CourierStatus) error {
					require.Equal(t, int64(42), id)
					require.Equal(t, domain.StatusAvailable, status)
					return nil
				},
			}
			return fn(tx)
		},
	}

	p := orders.NewProcessorWithDeps(d, r)

	err := p.Handle(context.Background(), orders.Event{OrderID: "order-4", Status: "completed"})
	require.NoError(t, err)
}

func TestProcessor_Handle_UnknownStatus_NoOps(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := NewMockDeliveryPort(ctrl)
	r := noopRunner{}

	p := orders.NewProcessorWithDeps(d, r)

	err := p.Handle(context.Background(), orders.Event{OrderID: "order-x", Status: "some-new-status"})
	require.NoError(t, err)
}
