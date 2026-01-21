package deliverytx

import (
	"context"

	"course-go-avito-Orurh/internal/domain"
)

// Repository is a delivery repository
type Repository interface {
	FindAvailableCourierForUpdate(ctx context.Context) (*domain.Courier, error)
	GetByOrderID(ctx context.Context, orderID string) (*domain.Delivery, error)
	InsertDelivery(ctx context.Context, d *domain.Delivery) error
	DeleteByOrderID(ctx context.Context, orderID string) error
	UpdateCourierStatus(ctx context.Context, id int64, status domain.CourierStatus) error
}

// Runner is a transaction runner
type Runner interface {
	WithTx(ctx context.Context, fn func(tx Repository) error) error
}
