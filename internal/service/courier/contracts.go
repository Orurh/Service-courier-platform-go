package courier

import (
	"context"
	"course-go-avito-Orurh/internal/domain"
)

// courierRepository defines storage operations required by the business layer.
type courierRepository interface {
	Get(ctx context.Context, id int64) (*domain.Courier, error)
	List(ctx context.Context, limit, offset *int) ([]domain.Courier, error)
	Create(ctx context.Context, c *domain.Courier) (int64, error)
	UpdatePartial(ctx context.Context, u domain.PartialCourierUpdate) (bool, error)
}
