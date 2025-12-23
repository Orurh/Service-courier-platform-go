package delivery_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/service/delivery"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestDeliveryService(repo *MockdeliveryRepository, f delivery.TimeFactory) *delivery.Service {
	return delivery.NewDeliveryService(repo, f, 3*time.Second, testLogger())
}

func TestService_Assign_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	courier := &domain.Courier{
		ID:            10,
		TransportType: domain.TransportTypeFoot,
	}
	expectedDeadline := time.Date(2025, 1, 2, 15, 4, 5, 0, time.UTC)
	repo.EXPECT().
		WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(delivery.TxRepository) error) error {
			tx := NewMockTxRepository(ctrl)

			tx.EXPECT().FindAvailableCourierForUpdate(gomock.Any()).Return(courier, nil)

			factory.EXPECT().Deadline(domain.TransportTypeFoot, gomock.Any()).Return(expectedDeadline, nil)

			tx.EXPECT().InsertDelivery(gomock.Any(), gomock.AssignableToTypeOf(&domain.Delivery{})).
				DoAndReturn(func(ctx context.Context, d *domain.Delivery) error {
					require.Equal(t, courier.ID, d.CourierID)
					require.Equal(t, orderID, d.OrderID)
					require.True(t, d.Deadline.Equal(expectedDeadline))
					return nil
				})

			tx.EXPECT().UpdateCourierStatus(gomock.Any(), courier.ID, domain.StatusBusy).Return(nil)

			return fn(tx)
		})

	service := newTestDeliveryService(repo, factory)

	res, err := service.Assign(ctx, orderID)

	require.NoError(t, err)
	require.Equal(t, courier.ID, res.CourierID)
	require.Equal(t, orderID, res.OrderID)
	require.Equal(t, courier.TransportType, res.TransportType)
	require.True(t, res.Deadline.Equal(expectedDeadline))
}

func TestService_Unassign_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)

	existing := &domain.Delivery{
		ID:        100,
		CourierID: 10,
		OrderID:   orderID,
	}

	repo.EXPECT().
		WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(delivery.TxRepository) error) error {
			tx := NewMockTxRepository(ctrl)

			tx.EXPECT().GetByOrderID(gomock.Any(), orderID).Return(existing, nil)

			tx.EXPECT().DeleteByOrderID(gomock.Any(), orderID).Return(nil)

			tx.EXPECT().UpdateCourierStatus(gomock.Any(), existing.CourierID, domain.StatusAvailable).Return(nil)

			return fn(tx)
		})

	service := newTestDeliveryService(repo, nil)

	res, err := service.Unassign(ctx, orderID)
	require.NoError(t, err)
	require.Equal(t, orderID, res.OrderID)
	require.Equal(t, existing.CourierID, res.CourierID)
	require.Equal(t, "unassigned", res.Status)
}

func TestService_Assign_InvalidOrderID(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	ctx := context.Background()
	badOrderID := "   "

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	service := newTestDeliveryService(repo, factory)

	res, err := service.Assign(ctx, badOrderID)
	require.ErrorIs(t, err, apperr.ErrInvalid)
	require.Equal(t, domain.AssignResult{}, res)
}

func TestService_Assign_BeginTxError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)
	service := newTestDeliveryService(repo, factory)

	txErr := errors.New("begin tx failed")

	repo.EXPECT().WithTx(gomock.Any(), gomock.Any()).Return(txErr)

	res, err := service.Assign(ctx, orderID)

	require.ErrorIs(t, err, txErr)
	require.Equal(t, domain.AssignResult{}, res)
}

func TestService_Unassign_BeginTxError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	service := newTestDeliveryService(repo, factory)

	txErr := errors.New("tx error")

	repo.EXPECT().WithTx(gomock.Any(), gomock.Any()).Return(txErr)

	_, err := service.Unassign(ctx, orderID)
	require.ErrorIs(t, err, txErr)
}

func TestService_Unassign_InvalidOrderID(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	ctx := context.Background()
	badOrderID := "   "

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	service := newTestDeliveryService(repo, factory)

	res, err := service.Unassign(ctx, badOrderID)
	require.ErrorIs(t, err, apperr.ErrInvalid)
	require.Equal(t, domain.UnassignResult{}, res)
}

func TestService_Assign_NoCourierAvailable(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	repo.EXPECT().WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(delivery.TxRepository) error) error {
			tx := NewMockTxRepository(ctrl)

			tx.EXPECT().FindAvailableCourierForUpdate(gomock.Any()).
				Return(nil, nil)

			return fn(tx)
		})

	service := newTestDeliveryService(repo, factory)

	res, err := service.Assign(ctx, orderID)

	require.ErrorIs(t, err, apperr.ErrConflict)
	require.Equal(t, domain.AssignResult{}, res)
}

func TestService_ReleaseExpired_RepoError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	ctx := context.Background()

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	wantErr := errors.New("boom")
	repo.EXPECT().ReleaseCouriers(gomock.Any(), gomock.Any()).Return(int64(0), wantErr)

	svc := newTestDeliveryService(repo, factory)

	err := svc.ReleaseExpired(ctx)
	require.ErrorIs(t, err, wantErr)
}

func TestDefaultTimeFactory_Deadline(t *testing.T) {
	t.Parallel()

	f := delivery.NewTimeFactory()
	now := time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		transport domain.CourierTransportType
		wantDelta time.Duration
		errAssert require.ErrorAssertionFunc
	}{
		{
			name:      "on_foot",
			transport: domain.TransportTypeFoot,
			wantDelta: 30 * time.Minute,
			errAssert: require.NoError,
		},
		{
			name:      "scooter",
			transport: domain.TransportTypeScooter,
			wantDelta: 15 * time.Minute,
			errAssert: require.NoError,
		},
		{
			name:      "car",
			transport: domain.TransportTypeCar,
			wantDelta: 5 * time.Minute,
			errAssert: require.NoError,
		},
		{
			name:      "unknown transport returns error",
			transport: domain.CourierTransportType("horse"),
			errAssert: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := f.Deadline(tt.transport, now)

			tt.errAssert(t, err)
			if err != nil {
				return
			}

			require.NoError(t, err)

			want := now.Add(tt.wantDelta)
			require.Equal(t, want, got)
		})
	}
}

func TestService_Assign_FindAvailableCourierError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	wantErr := errors.New("find available courier error")

	repo.EXPECT().WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(delivery.TxRepository) error) error {
			tx := NewMockTxRepository(ctrl)
			tx.EXPECT().FindAvailableCourierForUpdate(gomock.Any()).Return(nil, wantErr)
			return fn(tx)
		})

	service := newTestDeliveryService(repo, factory)

	res, err := service.Assign(ctx, orderID)

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, domain.AssignResult{}, res)
}

func TestService_Assign_DeadlineError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	courier := &domain.Courier{
		ID:            10,
		TransportType: domain.TransportTypeFoot,
	}

	wantErr := errors.New("deadline calculation error")

	repo.EXPECT().WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(delivery.TxRepository) error) error {
			tx := NewMockTxRepository(ctrl)
			tx.EXPECT().FindAvailableCourierForUpdate(gomock.Any()).Return(courier, nil)
			factory.EXPECT().Deadline(domain.TransportTypeFoot, gomock.Any()).Return(time.Time{}, wantErr)
			return fn(tx)
		})

	service := newTestDeliveryService(repo, factory)

	res, err := service.Assign(ctx, orderID)

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, domain.AssignResult{}, res)
}

func TestService_Assign_InsertDeliveryError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	courier := &domain.Courier{
		ID:            10,
		TransportType: domain.TransportTypeFoot,
	}
	expectedDeadline := time.Date(2025, 1, 2, 15, 4, 5, 0, time.UTC)

	wantErr := errors.New("insert delivery error")

	repo.EXPECT().WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(delivery.TxRepository) error) error {
			tx := NewMockTxRepository(ctrl)
			tx.EXPECT().FindAvailableCourierForUpdate(gomock.Any()).Return(courier, nil)
			factory.EXPECT().Deadline(domain.TransportTypeFoot, gomock.Any()).Return(expectedDeadline, nil)
			tx.EXPECT().InsertDelivery(gomock.Any(), gomock.Any()).Return(wantErr)
			return fn(tx)
		})

	service := newTestDeliveryService(repo, factory)

	res, err := service.Assign(ctx, orderID)

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, domain.AssignResult{}, res)
}

func TestService_Assign_UpdateCourierStatusError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	courier := &domain.Courier{
		ID:            10,
		TransportType: domain.TransportTypeFoot,
	}
	expectedDeadline := time.Date(2025, 1, 2, 15, 4, 5, 0, time.UTC)

	wantErr := errors.New("update courier status error")

	repo.EXPECT().WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(delivery.TxRepository) error) error {
			tx := NewMockTxRepository(ctrl)
			tx.EXPECT().FindAvailableCourierForUpdate(gomock.Any()).Return(courier, nil)
			factory.EXPECT().Deadline(domain.TransportTypeFoot, gomock.Any()).Return(expectedDeadline, nil)
			tx.EXPECT().InsertDelivery(gomock.Any(), gomock.Any()).Return(nil)
			tx.EXPECT().UpdateCourierStatus(gomock.Any(), courier.ID, domain.StatusBusy).Return(wantErr)
			return fn(tx)
		})

	service := newTestDeliveryService(repo, factory)

	res, err := service.Assign(ctx, orderID)

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, domain.AssignResult{}, res)
}

func TestService_Unassign_GetByOrderIDError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	wantErr := errors.New("get by order id error")

	repo.EXPECT().WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(delivery.TxRepository) error) error {
			tx := NewMockTxRepository(ctrl)
			tx.EXPECT().GetByOrderID(gomock.Any(), orderID).Return(nil, wantErr)
			return fn(tx)
		})

	service := newTestDeliveryService(repo, factory)

	res, err := service.Unassign(ctx, orderID)

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, domain.UnassignResult{}, res)
}

func TestService_Unassign_DeleteByOrderIDError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	existing := &domain.Delivery{
		ID:        100,
		CourierID: 10,
		OrderID:   orderID,
	}

	wantErr := errors.New("delete error")

	repo.EXPECT().WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(delivery.TxRepository) error) error {
			tx := NewMockTxRepository(ctrl)
			tx.EXPECT().GetByOrderID(gomock.Any(), orderID).Return(existing, nil)
			tx.EXPECT().DeleteByOrderID(gomock.Any(), orderID).Return(wantErr)
			return fn(tx)
		})

	service := newTestDeliveryService(repo, factory)

	res, err := service.Unassign(ctx, orderID)

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, domain.UnassignResult{}, res)
}

func TestService_Unassign_UpdateCourierStatusError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	existing := &domain.Delivery{
		ID:        100,
		CourierID: 10,
		OrderID:   orderID,
	}

	wantErr := errors.New("update status error")

	repo.EXPECT().WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(delivery.TxRepository) error) error {
			tx := NewMockTxRepository(ctrl)
			tx.EXPECT().GetByOrderID(gomock.Any(), orderID).Return(existing, nil)
			tx.EXPECT().DeleteByOrderID(gomock.Any(), orderID).Return(nil)
			tx.EXPECT().UpdateCourierStatus(gomock.Any(), existing.CourierID, domain.StatusAvailable).Return(wantErr)
			return fn(tx)
		})

	service := newTestDeliveryService(repo, factory)

	res, err := service.Unassign(ctx, orderID)

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, domain.UnassignResult{}, res)
}

func TestService_Unassign_DeliveryNotFound(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	repo.EXPECT().WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(delivery.TxRepository) error) error {
			tx := NewMockTxRepository(ctrl)
			tx.EXPECT().GetByOrderID(gomock.Any(), orderID).Return(nil, nil)
			return fn(tx)
		})

	service := newTestDeliveryService(repo, factory)

	res, err := service.Unassign(ctx, orderID)

	require.ErrorIs(t, err, apperr.ErrNotFound)
	require.Equal(t, domain.UnassignResult{}, res)
}

func TestNewDeliveryService_ZeroTimeoutUsesDefault_Behavior(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	svc := delivery.NewDeliveryService(repo, factory, 0, testLogger())

	ctx := context.Background()
	orderID := "order_1"

	var capturedCtx context.Context
	wantErr := errors.New("stopped")

	repo.EXPECT().
		WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(c context.Context, fn func(delivery.TxRepository) error) error {
			capturedCtx = c
			return wantErr
		})

	_, err := svc.Assign(ctx, orderID)

	require.ErrorIs(t, err, wantErr)
	require.NotNil(t, capturedCtx, "context must be captured")

	deadline, ok := capturedCtx.Deadline()
	require.True(t, ok, "expected context with deadline")

	remaining := time.Until(deadline)

	require.Greater(t, remaining, 2*time.Second)
	require.Less(t, remaining, 4*time.Second)
}
