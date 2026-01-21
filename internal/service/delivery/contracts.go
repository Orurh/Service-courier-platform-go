//go:generate mockgen -source=contracts.go -destination=delivery_mocks_test.go -package=delivery_test

package delivery

import (
	"context"
	"time"

	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/ports/deliverytx"
)

// TxRepository aliases deliverytx.Repository
type TxRepository = deliverytx.Repository

// deliveryRepository is an interface for the service layer.
type deliveryRepository interface {
	WithTx(ctx context.Context, fn func(tx TxRepository) error) error
	ReleaseCouriers(ctx context.Context, now time.Time) (int64, error)
}

// TimeFactory is a factory for calculating delivery deadline.
type TimeFactory interface {
	Deadline(transport domain.CourierTransportType, now time.Time) (time.Time, error)
}
