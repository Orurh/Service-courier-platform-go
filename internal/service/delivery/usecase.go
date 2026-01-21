package delivery

import (
	"context"
	"strings"
	"time"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/logx"
	"course-go-avito-Orurh/internal/ports/deliverytx"
)

// Service - service for assigning deliveries to couriers.
type Service struct {
	repo             deliveryRepository
	factory          TimeFactory
	operationTimeout time.Duration
	logger           logx.Logger
	now              func() time.Time
}

func (s *Service) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, s.operationTimeout)
}

// NewDeliveryService - creates a new DeliveryService.
func NewDeliveryService(r deliveryRepository, f TimeFactory, timeout time.Duration, logger logx.Logger) *Service {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &Service{
		repo:             r,
		factory:          f,
		operationTimeout: timeout,
		logger:           logger,
		now:              func() time.Time { return time.Now().UTC() },
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
	var result domain.AssignResult

	err = s.repo.WithTx(ctx, func(tx deliverytx.Repository) error {
		courier, err := tx.FindAvailableCourierForUpdate(ctx)
		if err != nil {
			return err
		}
		if courier == nil {
			return apperr.ErrConflict
		}

		now := s.now()
		deadline, err := s.factory.Deadline(domain.CourierTransportType(courier.TransportType), now)
		if err != nil {
			return err
		}

		d := &domain.Delivery{
			CourierID:  courier.ID,
			OrderID:    orderID,
			AssignedAt: now,
			Deadline:   deadline,
		}
		if err := tx.InsertDelivery(ctx, d); err != nil {
			return err
		}

		if err := tx.UpdateCourierStatus(ctx, courier.ID, domain.StatusBusy); err != nil {
			return err
		}

		result = domain.AssignResult{
			CourierID:     courier.ID,
			OrderID:       orderID,
			TransportType: courier.TransportType,
			Deadline:      deadline,
		}

		return nil
	})
	if err != nil {
		return domain.AssignResult{}, err
	}

	s.logger.Info("courier assigned",
		logx.String("event", "courier_assigned"),
		logx.String("order_id", result.OrderID),
		logx.Int64("courier_id", result.CourierID),
		logx.String("transport", string(result.TransportType)),
		logx.Time("deadline", result.Deadline),
	)

	return result, nil
}

// Unassign unassigns a delivery from a courier.
func (s *Service) Unassign(ctx context.Context, orderID string) (domain.UnassignResult, error) {
	orderID, err := validateOrderID(orderID)
	if err != nil {
		return domain.UnassignResult{}, err
	}

	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	var result domain.UnassignResult

	err = s.repo.WithTx(ctx, func(tx deliverytx.Repository) error {
		d, err := tx.GetByOrderID(ctx, orderID)
		if err != nil {
			return err
		}
		if d == nil {
			return apperr.ErrNotFound
		}

		if err := tx.DeleteByOrderID(ctx, orderID); err != nil {
			return err
		}

		if err := tx.UpdateCourierStatus(ctx, d.CourierID, domain.StatusAvailable); err != nil {
			return err
		}

		result = domain.UnassignResult{
			CourierID: d.CourierID,
			OrderID:   orderID,
			Status:    "unassigned",
		}

		return nil
	})
	if err != nil {
		return domain.UnassignResult{}, err
	}

	return result, nil
}

func validateOrderID(raw string) (string, error) {
	orderID := strings.TrimSpace(raw)
	if orderID == "" {
		return "", apperr.ErrInvalid
	}
	return orderID, nil
}

// ReleaseExpired releases expired couriers.
func (s *Service) ReleaseExpired(ctx context.Context) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	now := s.now()
	_, err := s.repo.ReleaseCouriers(ctx, now)
	return err
}
