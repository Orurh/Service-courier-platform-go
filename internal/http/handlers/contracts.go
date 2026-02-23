package handlers

import (
	"context"

	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/service/courier"
	"course-go-avito-Orurh/internal/service/delivery"
)

type courierUsecase interface {
	Get(ctx context.Context, id int64) (*domain.Courier, error)
	List(ctx context.Context, limit, offset *int) ([]domain.Courier, error)
	Create(ctx context.Context, c *domain.Courier) (int64, error)
	UpdatePartial(ctx context.Context, u domain.PartialCourierUpdate) (bool, error)
}

// NewCourierUsecase wires a CourierService into a courierUsecase.
func NewCourierUsecase(service *courier.Service) courierUsecase {
	return service
}

type deliveryUsecase interface {
	Assign(ctx context.Context, orderID string) (domain.AssignResult, error)
	Unassign(ctx context.Context, orderID string) (domain.UnassignResult, error)
}

// NewDeliveryUsecase wires a DeliveryService into a deliveryUsecase.
func NewDeliveryUsecase(svc *delivery.Service) deliveryUsecase {
	return svc
}
