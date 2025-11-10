package service

import (
	"context"
	"strings"
	"time"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"
)

type courierRepository interface {
	Get(ctx context.Context, id int64) (*domain.Courier, error)
	List(ctx context.Context, limit, offset *int) ([]domain.Courier, error)
	Create(ctx context.Context, c *domain.Courier) (int64, error)
	UpdatePartial(ctx context.Context, u domain.PartialCourierUpdate) (bool, error)
}

type courierService struct {
	repo             courierRepository
	operationTimeout time.Duration
}

// NewCourierService creates a service...
func NewCourierService(r courierRepository, timeout time.Duration) *courierService {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &courierService{repo: r, operationTimeout: timeout}
}

// withOperationTimeout
func (s *courierService) withOperationTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, s.operationTimeout)
}

// General validation
func validateCreate(c *domain.Courier) error {
	if c == nil {
		return apperr.Invalid
	}
	if strings.TrimSpace(c.Name) == "" {
		return apperr.Invalid
	}
	if !domain.ValidatePhone(c.Phone) {
		return apperr.Invalid
	}
	if !domain.CourierStatus(c.Status).Valid() {
		return apperr.Invalid
	}
	return nil
}

func validateUpdate(u *domain.PartialCourierUpdate) error {
	if u.ID <= 0 {
		return apperr.Invalid
	}
	if u.Name == nil && u.Phone == nil && u.Status == nil {
		return apperr.Invalid
	}
	if u.Name != nil && strings.TrimSpace(*u.Name) == "" {
		return apperr.Invalid
	}
	if u.Phone != nil && !domain.ValidatePhone(*u.Phone) {
		return apperr.Invalid
	}
	if u.Status != nil && !domain.CourierStatus(*u.Status).Valid() {
		return apperr.Invalid
	}
	return nil
}

func (s *courierService) Get(ctx context.Context, id int64) (*domain.Courier, error) {
	ctx, cancel := s.withOperationTimeout(ctx)
	defer cancel()
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, apperr.NotFound
	}
	return c, nil
}

func (s *courierService) List(ctx context.Context, limit, offset *int) ([]domain.Courier, error) {
	ctx, cancel := s.withOperationTimeout(ctx)
	defer cancel()
	return s.repo.List(ctx, limit, offset)
}

func (s *courierService) Create(ctx context.Context, c *domain.Courier) (int64, error) {
	if err := validateCreate(c); err != nil {
		return 0, err
	}
	ctx, cancel := s.withOperationTimeout(ctx)
	defer cancel()
	return s.repo.Create(ctx, c)
}

func (s *courierService) UpdatePartial(ctx context.Context, u domain.PartialCourierUpdate) (bool, error) {
	if err := validateUpdate(&u); err != nil {
		return false, err
	}
	ctx, cancel := s.withOperationTimeout(ctx)
	defer cancel()
	ok, err := s.repo.UpdatePartial(ctx, u)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, apperr.NotFound
	}
	return true, nil
}
