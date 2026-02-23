//go:generate mockgen -source=contracts.go -destination=orders_mocks_test.go -package=orders_test

package orders

import (
	"context"

	"course-go-avito-Orurh/internal/domain"
)

// DeliveryPort abstracts the subset of delivery service operations
// needed by orders Processor when handling order events
type DeliveryPort interface {
	Assign(ctx context.Context, orderID string) (domain.AssignResult, error)
	Unassign(ctx context.Context, orderID string) (domain.UnassignResult, error)
}
