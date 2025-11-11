package handlers

import (
	"context"
	"course-go-avito-Orurh/internal/domain"
)

// CourierUsecase exposes courier-related business operations to the HTTP layer.
type CourierUsecase interface {
	Get(ctx context.Context, id int64) (*domain.Courier, error)
	List(ctx context.Context, limit, offset *int) ([]domain.Courier, error)
	Create(ctx context.Context, c *domain.Courier) (int64, error)
	UpdatePartial(ctx context.Context, u domain.PartialCourierUpdate) (bool, error)
}
