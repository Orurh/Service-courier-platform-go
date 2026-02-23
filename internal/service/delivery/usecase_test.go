package delivery_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/logx"
	"course-go-avito-Orurh/internal/service/delivery"
)

func newCtrl(t *testing.T) *gomock.Controller {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	return ctrl
}

type stubTimeFactory struct {
	fn func(transport domain.CourierTransportType, now time.Time) (time.Time, error)
}

func (s stubTimeFactory) Deadline(transport domain.CourierTransportType, now time.Time) (time.Time, error) {
	if s.fn == nil {
		return time.Time{}, errors.New("stubTimeFactory: nil")
	}
	return s.fn(transport, now)
}

type stubTx struct {
	findFn   func(context.Context) (*domain.Courier, error)
	insertFn func(context.Context, *domain.Delivery) error
	getFn    func(context.Context, string) (*domain.Delivery, error)
	delFn    func(context.Context, string) error
	updFn    func(context.Context, int64, domain.CourierStatus) error
}

func (s *stubTx) FindAvailableCourierForUpdate(ctx context.Context) (*domain.Courier, error) {
	if s.findFn == nil {
		return nil, nil
	}
	return s.findFn(ctx)
}
func (s *stubTx) InsertDelivery(ctx context.Context, d *domain.Delivery) error {
	if s.insertFn == nil {
		return nil
	}
	return s.insertFn(ctx, d)
}
func (s *stubTx) GetByOrderID(ctx context.Context, orderID string) (*domain.Delivery, error) {
	if s.getFn == nil {
		return nil, nil
	}
	return s.getFn(ctx, orderID)
}
func (s *stubTx) DeleteByOrderID(ctx context.Context, orderID string) error {
	if s.delFn == nil {
		return nil
	}
	return s.delFn(ctx, orderID)
}
func (s *stubTx) UpdateCourierStatus(ctx context.Context, id int64, status domain.CourierStatus) error {
	if s.updFn == nil {
		return nil
	}
	return s.updFn(ctx, id, status)
}

func testLogger(_ io.Writer) logx.Logger {
	return logx.Nop()
}

func newTestDeliveryService(repo *MockdeliveryRepository, f delivery.TimeFactory) *delivery.Service {
	return delivery.NewDeliveryService(repo, f, 3*time.Second, testLogger(io.Discard))
}

func TestService_Assign_Success(t *testing.T) {
	t.Parallel()

	ctrl := newCtrl(t)

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	expectedDeadline := time.Date(2025, 1, 2, 15, 4, 5, 0, time.UTC)
	factory := stubTimeFactory{
		fn: func(transport domain.CourierTransportType, _ time.Time) (time.Time, error) {
			require.Equal(t, domain.TransportTypeFoot, transport)
			return expectedDeadline, nil
		},
	}

	courier := &domain.Courier{
		ID:            10,
		TransportType: domain.TransportTypeFoot,
	}

	repo.EXPECT().
		WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(delivery.TxRepository) error) error {
			tx := &stubTx{
				findFn: func(context.Context) (*domain.Courier, error) { return courier, nil },
				insertFn: func(_ context.Context, d *domain.Delivery) error {
					require.Equal(t, courier.ID, d.CourierID)
					require.Equal(t, orderID, d.OrderID)
					require.True(t, d.Deadline.Equal(expectedDeadline))
					return nil
				},
				updFn: func(_ context.Context, id int64, st domain.CourierStatus) error {
					require.Equal(t, courier.ID, id)
					require.Equal(t, domain.StatusBusy, st)
					return nil
				},
			}
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

func TestService_Assign_InvalidOrderID(t *testing.T) {
	t.Parallel()

	ctrl := newCtrl(t)

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

	ctrl := newCtrl(t)

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

func TestService_Assign_NoCourierAvailable(t *testing.T) {
	t.Parallel()

	ctrl := newCtrl(t)

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	repo.EXPECT().WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(delivery.TxRepository) error) error {
			tx := &stubTx{
				findFn: func(context.Context) (*domain.Courier, error) { return nil, nil },
			}
			return fn(tx)
		})

	service := newTestDeliveryService(repo, factory)

	res, err := service.Assign(ctx, orderID)

	require.ErrorIs(t, err, apperr.ErrConflict)
	require.Equal(t, domain.AssignResult{}, res)
}

func TestService_Assign_FindAvailableCourierError(t *testing.T) {
	t.Parallel()

	ctrl := newCtrl(t)

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	wantErr := errors.New("find available courier error")

	repo.EXPECT().WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(delivery.TxRepository) error) error {
			tx := &stubTx{
				findFn: func(context.Context) (*domain.Courier, error) { return nil, wantErr },
			}
			return fn(tx)
		})

	service := newTestDeliveryService(repo, factory)

	res, err := service.Assign(ctx, orderID)

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, domain.AssignResult{}, res)
}

func TestService_Assign_DeadlineError(t *testing.T) {
	t.Parallel()

	ctrl := newCtrl(t)

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
			tx := &stubTx{
				findFn: func(context.Context) (*domain.Courier, error) { return courier, nil },
			}
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

	ctrl := newCtrl(t)

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
			tx := &stubTx{
				findFn: func(context.Context) (*domain.Courier, error) { return courier, nil },
				insertFn: func(context.Context, *domain.Delivery) error {
					return wantErr
				},
			}
			factory.EXPECT().Deadline(domain.TransportTypeFoot, gomock.Any()).Return(expectedDeadline, nil)
			return fn(tx)
		})

	service := newTestDeliveryService(repo, factory)

	res, err := service.Assign(ctx, orderID)

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, domain.AssignResult{}, res)
}

func TestService_Assign_UpdateCourierStatusError(t *testing.T) {
	t.Parallel()

	ctrl := newCtrl(t)

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
			tx := &stubTx{
				findFn:   func(context.Context) (*domain.Courier, error) { return courier, nil },
				insertFn: func(context.Context, *domain.Delivery) error { return nil },
				updFn:    func(context.Context, int64, domain.CourierStatus) error { return wantErr },
			}
			factory.EXPECT().Deadline(domain.TransportTypeFoot, gomock.Any()).Return(expectedDeadline, nil)
			return fn(tx)
		})

	service := newTestDeliveryService(repo, factory)

	res, err := service.Assign(ctx, orderID)

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, domain.AssignResult{}, res)
}

func TestNewDeliveryService_ZeroTimeoutUsesDefault_Behavior(t *testing.T) {
	t.Parallel()

	ctrl := newCtrl(t)

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	svc := delivery.NewDeliveryService(repo, factory, 0, testLogger(io.Discard))

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

	res, err := svc.Assign(ctx, orderID)
	require.Equal(t, domain.AssignResult{}, res)

	require.ErrorIs(t, err, wantErr)
	require.NotNil(t, capturedCtx, "context must be captured")

	deadline, ok := capturedCtx.Deadline()
	require.True(t, ok, "expected context with deadline")

	remaining := time.Until(deadline)

	require.Greater(t, remaining, 2*time.Second)
	require.Less(t, remaining, 4*time.Second)
}

func TestService_Unassign_Success(t *testing.T) {
	t.Parallel()

	ctrl := newCtrl(t)

	ctx := context.Background()
	orderID := "order_1"
	repo := NewMockdeliveryRepository(ctrl)

	factory := stubTimeFactory{
		fn: func(domain.CourierTransportType, time.Time) (time.Time, error) {
			return time.Time{}, nil
		},
	}

	existing := &domain.Delivery{
		ID:        100,
		CourierID: 10,
		OrderID:   orderID,
	}

	repo.EXPECT().
		WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(delivery.TxRepository) error) error {
			tx := &stubTx{
				getFn: func(_ context.Context, gotOrderID string) (*domain.Delivery, error) {
					require.Equal(t, orderID, gotOrderID)
					return existing, nil
				},
				delFn: func(_ context.Context, gotOrderID string) error {
					require.Equal(t, orderID, gotOrderID)
					return nil
				},
				updFn: func(_ context.Context, id int64, st domain.CourierStatus) error {
					require.Equal(t, existing.CourierID, id)
					require.Equal(t, domain.StatusAvailable, st)
					return nil
				},
			}
			return fn(tx)
		})

	service := newTestDeliveryService(repo, factory)

	res, err := service.Unassign(ctx, orderID)
	require.NoError(t, err)
	require.Equal(t, orderID, res.OrderID)
	require.Equal(t, existing.CourierID, res.CourierID)
	require.Equal(t, "unassigned", res.Status)
}

func TestService_Unassign_BeginTxError(t *testing.T) {
	t.Parallel()

	ctrl := newCtrl(t)

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

	ctrl := newCtrl(t)

	ctx := context.Background()
	badOrderID := "   "

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	service := newTestDeliveryService(repo, factory)

	res, err := service.Unassign(ctx, badOrderID)
	require.ErrorIs(t, err, apperr.ErrInvalid)
	require.Equal(t, domain.UnassignResult{}, res)
}

func TestService_Unassign_GetByOrderIDError(t *testing.T) {
	t.Parallel()

	ctrl := newCtrl(t)

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	wantErr := errors.New("get by order id error")

	repo.EXPECT().WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(delivery.TxRepository) error) error {
			tx := &stubTx{
				getFn: func(context.Context, string) (*domain.Delivery, error) { return nil, wantErr },
			}
			return fn(tx)
		})

	service := newTestDeliveryService(repo, factory)

	res, err := service.Unassign(ctx, orderID)

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, domain.UnassignResult{}, res)
}

func TestService_Unassign_DeleteByOrderIDError(t *testing.T) {
	t.Parallel()

	ctrl := newCtrl(t)

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
			tx := &stubTx{
				getFn: func(_ context.Context, gotOrderID string) (*domain.Delivery, error) {
					require.Equal(t, orderID, gotOrderID)
					return existing, nil
				},
				delFn: func(_ context.Context, gotOrderID string) error {
					require.Equal(t, orderID, gotOrderID)
					return wantErr
				},
			}
			return fn(tx)
		})

	service := newTestDeliveryService(repo, factory)

	res, err := service.Unassign(ctx, orderID)

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, domain.UnassignResult{}, res)
}

func TestService_Unassign_UpdateCourierStatusError(t *testing.T) {
	t.Parallel()

	ctrl := newCtrl(t)

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
			tx := &stubTx{
				getFn: func(_ context.Context, gotOrderID string) (*domain.Delivery, error) {
					require.Equal(t, orderID, gotOrderID)
					return existing, nil
				},
				delFn: func(_ context.Context, gotOrderID string) error {
					require.Equal(t, orderID, gotOrderID)
					return nil
				},
				updFn: func(_ context.Context, id int64, st domain.CourierStatus) error {
					require.Equal(t, existing.CourierID, id)
					require.Equal(t, domain.StatusAvailable, st)
					return wantErr
				},
			}
			return fn(tx)
		})

	service := newTestDeliveryService(repo, factory)

	res, err := service.Unassign(ctx, orderID)

	require.ErrorIs(t, err, wantErr)
	require.Equal(t, domain.UnassignResult{}, res)
}

func TestService_Unassign_DeliveryNotFound(t *testing.T) {
	t.Parallel()

	ctrl := newCtrl(t)

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	repo.EXPECT().WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(delivery.TxRepository) error) error {
			tx := &stubTx{
				getFn: func(context.Context, string) (*domain.Delivery, error) { return nil, nil },
			}
			return fn(tx)
		})

	service := newTestDeliveryService(repo, factory)

	res, err := service.Unassign(ctx, orderID)

	require.ErrorIs(t, err, apperr.ErrNotFound)
	require.Equal(t, domain.UnassignResult{}, res)
}

func TestService_ReleaseExpired_RepoError(t *testing.T) {
	t.Parallel()

	ctrl := newCtrl(t)

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

			want := now.Add(tt.wantDelta)
			require.Equal(t, want, got)
		})
	}
}
