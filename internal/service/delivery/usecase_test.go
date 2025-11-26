package delivery

import (
	"context"
	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
)

func newTestDeliveryService(repo deliveryRepository, f TimeFactory) *Service {
	return NewDeliveryService(repo, f, 3*time.Second)
}

func TestNewDeliveryService_ZeroTimeoutUsesDefault(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	service := NewDeliveryService(repo, factory, 0)
	if service.operationTimeout != 3*time.Second {
		t.Fatalf("default timeout 3s, got %v", service.operationTimeout)
	}
}

func TestService_Assign_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	tx := NewMockTx(ctrl)
	factory := NewMockTimeFactory(ctrl)

	courier := &domain.Courier{
		ID:            10,
		TransportType: domain.TransportTypeFoot,
	}
	expectedDeadline := time.Date(2025, 1, 2, 15, 4, 5, 0, time.UTC)

	tx.EXPECT().Rollback(gomock.Any()).AnyTimes()

	repo.EXPECT().
		BeginTx(gomock.Any()).
		Return(tx, nil)

	repo.EXPECT().
		FindAvailableCourierForUpdate(gomock.Any(), tx).
		Return(courier, nil)

	factory.EXPECT().
		Deadline(domain.TransportTypeFoot, gomock.Any()).
		Return(expectedDeadline, nil)

	repo.EXPECT().
		InsertDelivery(gomock.Any(), tx, gomock.AssignableToTypeOf(&domain.Delivery{})).
		DoAndReturn(func(ctx context.Context, gotTx Tx, d *domain.Delivery) error {
			if d.CourierID != courier.ID {
				t.Fatalf("expected inserted.CourierID %d, got %d", courier.ID, d.CourierID)
			}
			if d.OrderID != orderID {
				t.Fatalf("expected inserted.OrderID %q, got %q", orderID, d.OrderID)
			}
			if !d.Deadline.Equal(expectedDeadline) {
				t.Fatalf("expected inserted.Deadline %v, got %v", expectedDeadline, d.Deadline)
			}
			return nil
		})

	repo.EXPECT().
		UpdateCourierStatus(gomock.Any(), tx, courier.ID, string(domain.StatusBusy)).
		Return(nil)

	tx.EXPECT().
		Commit(gomock.Any()).
		Return(nil)

	service := newTestDeliveryService(repo, factory)

	res, err := service.Assign(ctx, orderID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.CourierID != courier.ID {
		t.Fatalf("expected CourierID %d, got %d", courier.ID, res.CourierID)
	}
	if res.OrderID != orderID {
		t.Fatalf("expected OrderID %q, got %q", orderID, res.OrderID)
	}
	if res.TransportType != courier.TransportType {
		t.Fatalf("expected TransportType %q, got %q", courier.TransportType, res.TransportType)
	}
	if !res.Deadline.Equal(expectedDeadline) {
		t.Fatalf("expected Deadline %v, got %v", expectedDeadline, res.Deadline)
	}
}

func TestService_Unassign_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	tx := NewMockTx(ctrl)

	existing := &domain.Delivery{
		ID:        100,
		CourierID: 10,
		OrderID:   orderID,
	}

	tx.EXPECT().Rollback(gomock.Any()).AnyTimes()

	repo.EXPECT().
		BeginTx(gomock.Any()).
		Return(tx, nil)

	repo.EXPECT().
		GetByOrderID(gomock.Any(), tx, orderID).
		Return(existing, nil)

	repo.EXPECT().
		DeleteByOrderID(gomock.Any(), tx, orderID).
		Return(nil)

	repo.EXPECT().
		UpdateCourierStatus(gomock.Any(), tx, existing.CourierID, string(domain.StatusAvailable)).
		Return(nil)

	tx.EXPECT().
		Commit(gomock.Any()).
		Return(nil)

	service := newTestDeliveryService(repo, nil)

	res, err := service.Unassign(ctx, orderID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.OrderID != orderID {
		t.Fatalf("expected OrderID %q, got %q", orderID, res.OrderID)
	}
	if res.CourierID != existing.CourierID {
		t.Fatalf("expected CourierID %d, got %d", existing.CourierID, res.CourierID)
	}
	if res.Status != "unassigned" {
		t.Fatalf("expected Status %q, got %q", "unassigned", res.Status)
	}
}

func TestService_Assign_InvalidOrderID(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	badOrderID := "   "

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	service := newTestDeliveryService(repo, factory)

	res, err := service.Assign(ctx, badOrderID)
	if !errors.Is(err, apperr.Invalid) {
		t.Fatalf("expected Invalid for bad orderID, got %v", err)
	}
	if res != (domain.AssignResult{}) {
		t.Fatalf("expected zero AssignResult, got %#v", res)
	}
}

func TestService_Assign_BeginTxError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)
	service := newTestDeliveryService(repo, factory)

	txErr := errors.New("begin tx failed")

	repo.EXPECT().
		BeginTx(gomock.Any()).
		Return(nil, txErr)

	res, err := service.Assign(ctx, orderID)

	if !errors.Is(err, txErr) {
		t.Fatalf("expected BeginTx error %v, got %v", txErr, err)
	}
	if res != (domain.AssignResult{}) {
		t.Fatalf("expected zero AssignResult, got %#v", res)
	}
}

func TestService_Unassign_BeginTxError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	service := newTestDeliveryService(repo, factory)

	txErr := errors.New("tx error")

	repo.EXPECT().
		BeginTx(gomock.Any()).
		Return(nil, txErr)

	_, err := service.Unassign(ctx, orderID)
	if !errors.Is(err, txErr) {
		t.Fatalf("expected tx error, got %v", err)
	}
}

func TestService_Unassign_InvalidOrderID(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	badOrderID := "   "

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	service := newTestDeliveryService(repo, factory)

	res, err := service.Unassign(ctx, badOrderID)
	if !errors.Is(err, apperr.Invalid) {
		t.Fatalf("expected Invalid for bad orderID, got %v", err)
	}
	if res != (domain.UnassignResult{}) {
		t.Fatalf("expected zero UnassignResult, got %#v", res)
	}
}

func TestService_Assign_NoCourierAvailable(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	orderID := "order_1"

	repo := NewMockdeliveryRepository(ctrl)
	tx := NewMockTx(ctrl)
	factory := NewMockTimeFactory(ctrl)

	tx.EXPECT().Rollback(gomock.Any()).AnyTimes()

	repo.EXPECT().
		BeginTx(gomock.Any()).
		Return(tx, nil)

	repo.EXPECT().
		FindAvailableCourierForUpdate(gomock.Any(), tx).
		Return(nil, nil)

	service := newTestDeliveryService(repo, factory)

	res, err := service.Assign(ctx, orderID)
	if !errors.Is(err, apperr.Conflict) {
		t.Fatalf("expected apperr.Conflict, got %v", err)
	}
	if res != (domain.AssignResult{}) {
		t.Fatalf("expected zero AssignResult, got %#v", res)
	}
}

func TestService_ReleaseExpired_RepoError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	repo := NewMockdeliveryRepository(ctrl)
	factory := NewMockTimeFactory(ctrl)

	wantErr := errors.New("boom")
	repo.EXPECT().
		ReleaseCouriers(gomock.Any(), gomock.Any()).
		Return(int64(0), wantErr)

	svc := newTestDeliveryService(repo, factory)

	err := svc.ReleaseExpired(ctx)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestDefaultTimeFactory_Deadline(t *testing.T) {
	t.Parallel()

	f := NewTimeFactory()
	now := time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		transport domain.CourierTransportType
		wantDelta time.Duration
		wantErr   bool
	}{
		{
			name:      "on_foot",
			transport: domain.TransportTypeFoot,
			wantDelta: 30 * time.Minute,
		},
		{
			name:      "scooter",
			transport: domain.TransportTypeScooter,
			wantDelta: 15 * time.Minute,
		},
		{
			name:      "car",
			transport: domain.TransportTypeCar,
			wantDelta: 5 * time.Minute,
		},
		{
			name:      "unknown transport returns error",
			transport: domain.CourierTransportType("horse"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := f.Deadline(tt.transport, now)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for transport %q, got nil", tt.transport)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			want := now.Add(tt.wantDelta)
			if !got.Equal(want) {
				t.Fatalf("expected deadline %v, got %v", want, got)
			}
		})
	}
}
