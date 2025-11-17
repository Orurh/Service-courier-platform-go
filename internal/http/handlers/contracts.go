package handlers

import (
	"context"
	"course-go-avito-Orurh/internal/domain"
)

// сourierUsecase exposes courier-related business operations to the HTTP layer.
type courierUsecase interface {
	Get(ctx context.Context, id int64) (*domain.Courier, error)
	List(ctx context.Context, limit, offset *int) ([]domain.Courier, error)
	Create(ctx context.Context, c *domain.Courier) (int64, error)
	UpdatePartial(ctx context.Context, u domain.PartialCourierUpdate) (bool, error)
}

// можно через адаптер сделать, тогда не придется имплементировать сервис, дальше по заданиям решу, что делать
// func NewCourierUsecase(service *courier.Service) courierUsecase {
// 	return service // CourierService реализует все методы интерфейса
// }
