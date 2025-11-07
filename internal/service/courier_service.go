package service

import (
	"context"
	"strings"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"
)

type courierService struct {
	repo CourierRepository
}

// NewCourierService creates a CourierUsecase implementation backed by a repository.
func NewCourierService(r CourierRepository) CourierUsecase {
	return &courierService{repo: r}
}

// General validation
func validateCreate(c *domain.Courier) error {
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
	return s.repo.List(ctx, limit, offset)
}

func (s *courierService) Create(ctx context.Context, c *domain.Courier) (int64, error) {
	if err := validateCreate(c); err != nil {
		return 0, apperr.Invalid
	}
	id, err := s.repo.Create(ctx, c)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *courierService) UpdatePartial(ctx context.Context, u domain.PartialCourierUpdate) (bool, error) {
	if err := validateUpdate(&u); err != nil {
		return false, apperr.Invalid
	}
	ok, err := s.repo.UpdatePartial(ctx, u)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, apperr.NotFound
	}
	return true, nil
}
