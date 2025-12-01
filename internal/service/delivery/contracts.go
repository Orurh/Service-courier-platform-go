//go:generate mockgen -source=contracts.go -destination=delivery_mocks_test.go -package=delivery_test

package delivery

import (
	"context"
	"time"

	"course-go-avito-Orurh/internal/domain"
)

// TxRepository abstracts a delivery repository transaction.
type TxRepository interface {
	FindAvailableCourierForUpdate(ctx context.Context) (*domain.Courier, error)
	UpdateCourierStatus(ctx context.Context, id int64, status domain.CourierStatus) error
	InsertDelivery(ctx context.Context, d *domain.Delivery) error
	GetByOrderID(ctx context.Context, orderID string) (*domain.Delivery, error)
	DeleteByOrderID(ctx context.Context, orderID string) error
}

// deliveryRepository is an interface for the service layer.
type deliveryRepository interface {
	WithTx(ctx context.Context, fn func(tx TxRepository) error) error
	ReleaseCouriers(ctx context.Context, now time.Time) (int64, error)
}

// TimeFactory is a factory for calculating delivery deadline.
type TimeFactory interface {
	Deadline(transport domain.CourierTransportType, now time.Time) (time.Time, error)
}
