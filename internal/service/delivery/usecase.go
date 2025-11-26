package delivery

import (
	"context"
	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"
	"strings"
	"time"
)

// Service - service for assigning deliveries to couriers.
type Service struct {
	repo             deliveryRepository
	factory          TimeFactory
	operationTimeout time.Duration
}

func (s *Service) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, s.operationTimeout)
}

// NewDeliveryService - creates a new DeliveryService.
func NewDeliveryService(r deliveryRepository, f TimeFactory, timeout time.Duration) *Service {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &Service{
		repo:             r,
		factory:          f,
		operationTimeout: timeout,
	}
}

// Assign assigns a delivery to a courier.
func (s *Service) Assign(ctx context.Context, orderID string) (domain.AssignResult, error) {
	orderID, err := validateOrderID(orderID)
	if err != nil {
		return domain.AssignResult{}, err
	}
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return domain.AssignResult{}, err
	}
	defer tx.Rollback(ctx)

	courier, err := s.repo.FindAvailableCourierForUpdate(ctx, tx)
	if err != nil {
		return domain.AssignResult{}, err
	}
	if courier == nil {
		return domain.AssignResult{}, apperr.Conflict
	}

	now := time.Now().UTC()
	deadline, err := s.factory.Deadline(domain.CourierTransportType(courier.TransportType), now)
	if err != nil {
		return domain.AssignResult{}, err
	}

	d := &domain.Delivery{
		CourierID:  courier.ID,
		OrderID:    orderID,
		AssignedAt: now,
		Deadline:   deadline,
	}
	if err := s.repo.InsertDelivery(ctx, tx, d); err != nil {
		return domain.AssignResult{}, err
	}

	if err := s.repo.UpdateCourierStatus(ctx, tx, courier.ID, string(domain.StatusBusy)); err != nil {
		return domain.AssignResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.AssignResult{}, err
	}

	return domain.AssignResult{
		CourierID:     courier.ID,
		OrderID:       orderID,
		TransportType: courier.TransportType,
		Deadline:      deadline,
	}, nil
}

// Unassign unassigns a delivery from a courier.
func (s *Service) Unassign(ctx context.Context, orderID string) (domain.UnassignResult, error) {
	orderID, err := validateOrderID(orderID)
	if err != nil {
		return domain.UnassignResult{}, err
	}

	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return domain.UnassignResult{}, err
	}
	defer tx.Rollback(ctx)

	d, err := s.repo.GetByOrderID(ctx, tx, orderID)
	if err != nil {
		return domain.UnassignResult{}, err
	}
	if d == nil {
		return domain.UnassignResult{}, apperr.NotFound
	}

	if err := s.repo.DeleteByOrderID(ctx, tx, orderID); err != nil {
		return domain.UnassignResult{}, err
	}

	if err := s.repo.UpdateCourierStatus(ctx, tx, d.CourierID, string(domain.StatusAvailable)); err != nil {
		return domain.UnassignResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.UnassignResult{}, err
	}

	return domain.UnassignResult{
		CourierID: d.CourierID,
		OrderID:   orderID,
		Status:    "unassigned",
	}, nil
}

func validateOrderID(raw string) (string, error) {
	orderID := strings.TrimSpace(raw)
	if orderID == "" {
		return "", apperr.Invalid
	}
	return orderID, nil
}

// ReleaseExpired releases expired couriers.
func (s *Service) ReleaseExpired(ctx context.Context) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	now := time.Now().UTC()
	_, err := s.repo.ReleaseCouriers(ctx, now)
	return err
}
