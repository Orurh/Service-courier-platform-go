//go:generate mockgen -source=contracts.go -destination=delivery_mocks_test.go -package=delivery

package delivery

import (
	"context"
	"course-go-avito-Orurh/internal/domain"
	"time"
)

// Tx abstracts a delivery repository transaction.
type Tx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context)
}

type deliveryRepository interface {
	BeginTx(ctx context.Context) (Tx, error)
	FindAvailableCourierForUpdate(ctx context.Context, tx Tx) (*domain.Courier, error)
	UpdateCourierStatus(ctx context.Context, tx Tx, id int64, status string) error
	InsertDelivery(ctx context.Context, tx Tx, d *domain.Delivery) error
	GetByOrderID(ctx context.Context, tx Tx, orderID string) (*domain.Delivery, error)
	DeleteByOrderID(ctx context.Context, tx Tx, orderID string) error
	ReleaseCouriers(ctx context.Context, now time.Time) (int64, error)
}
