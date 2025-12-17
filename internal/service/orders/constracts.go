package orders

import (
	"context"

	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/service/delivery"
)

// TxRunner abstracts running a function within a delivery transaction
type TxRunner interface {
	WithTx(ctx context.Context, fn func(tx delivery.TxRepository) error) error
}

// DeliveryPort abstracts the subset of delivery service operations
// needed by orders Processor when handling order events
type DeliveryPort interface {
	Assign(ctx context.Context, orderID string) (domain.AssignResult, error)
	Unassign(ctx context.Context, orderID string) (domain.UnassignResult, error)
}
