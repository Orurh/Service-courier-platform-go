package service

import (
	"context"
	"strings"
	"time"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"
)

// CourierService coordinates courier business logic and orchestrates repository calls.
type CourierService struct {
	repo             CourierRepository
	operationTimeout time.Duration
}

// NewCourierService creates and configures a CourierService.
func NewCourierService(r CourierRepository, timeout time.Duration) *CourierService {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &CourierService{repo: r, operationTimeout: timeout}
}

// withOperationTimeout
func (s *CourierService) withOperationTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
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

// Get retrieves a courier by its ID. It returns (nil, nil) if the courier is not found.
func (s *CourierService) Get(ctx context.Context, id int64) (*domain.Courier, error) {
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

// List returns couriers with optional pagination
func (s *CourierService) List(ctx context.Context, limit, offset *int) ([]domain.Courier, error) {
	ctx, cancel := s.withOperationTimeout(ctx)
	defer cancel()
	return s.repo.List(ctx, limit, offset)
}

// Create persists a new courier and returns its generated ID.
func (s *CourierService) Create(ctx context.Context, c *domain.Courier) (int64, error) {
	if err := validateCreate(c); err != nil {
		return 0, err
	}
	ctx, cancel := s.withOperationTimeout(ctx)
	defer cancel()
	return s.repo.Create(ctx, c)
}

// UpdatePartial applies a partial update to a courier. It returns true if a row was updated.
func (s *CourierService) UpdatePartial(ctx context.Context, u domain.PartialCourierUpdate) (bool, error) {
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
