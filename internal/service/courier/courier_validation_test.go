package courier

import (
	"errors"
	"testing"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"
)

func TestValidateCreate_NilCourier(t *testing.T) {
	t.Parallel()
	err := validateCreate(nil)
	if !errors.Is(err, apperr.Invalid) {
		t.Fatalf("expected Invalid for nil courier, got %v", err)
	}
}

func TestValidateCreate_EmptyName(t *testing.T) {
	t.Parallel()
	c := &domain.Courier{
		Name:          "    ",
		Phone:         "+70000000000",
		Status:        domain.StatusAvailable,
		TransportType: domain.TransportTypeFoot,
	}
	err := validateCreate(c)
	if !errors.Is(err, apperr.Invalid) {
		t.Fatalf("expected Invalid for empty name, got %v", err)
	}
}

func TestValidateCreate_InvalidPhone(t *testing.T) {
	t.Parallel()
	c := &domain.Courier{
		Name:          "Artem",
		Phone:         "123",
		Status:        domain.StatusAvailable,
		TransportType: domain.TransportTypeFoot,
	}
	err := validateCreate(c)
	if !errors.Is(err, apperr.Invalid) {
		t.Fatalf("expected Invalid for bad phone, got %v", err)
	}
}

func TestValidateCreate_InvalidStatus(t *testing.T) {
	t.Parallel()
	c := &domain.Courier{
		Name:          "Artem",
		Phone:         "+70000000000",
		Status:        domain.CourierStatus("boom"),
		TransportType: domain.TransportTypeFoot,
	}
	err := validateCreate(c)
	if !errors.Is(err, apperr.Invalid) {
		t.Fatalf("expected Invalid for bad status, got %v", err)
	}
}

func TestValidateCreate_InvalidTransportType(t *testing.T) {
	t.Parallel()
	c := &domain.Courier{
		Name:          "Artem",
		Phone:         "+70000000000",
		Status:        domain.StatusAvailable,
		TransportType: domain.CourierTransportType("teleport"),
	}
	err := validateCreate(c)
	if !errors.Is(err, apperr.Invalid) {
		t.Fatalf("expected Invalid for bad transport type, got %v", err)
	}
}

func TestValidateCreate_ValidCourier(t *testing.T) {
	t.Parallel()
	c := &domain.Courier{
		Name:          "Artem",
		Phone:         "+70000000000",
		Status:        domain.StatusAvailable,
		TransportType: domain.TransportTypeFoot,
	}
	if err := validateCreate(c); err != nil {
		t.Fatalf("expected nil error for valid courier, got %v", err)
	}
}

func TestValidateUpdate_IdLessOrEqualZero(t *testing.T) {
	t.Parallel()
	u := &domain.PartialCourierUpdate{
		ID: 0,
	}
	err := validateUpdate(u)
	if !errors.Is(err, apperr.Invalid) {
		t.Fatalf("expected Invalid for id <= 0, got %v", err)
	}
}

func TestValidateUpdate_AllFieldsNil(t *testing.T) {
	t.Parallel()
	u := &domain.PartialCourierUpdate{
		ID: 1,
	}
	err := validateUpdate(u)
	if !errors.Is(err, apperr.Invalid) {
		t.Fatalf("expected Invalid when all fields nil, got %v", err)
	}
}

func TestValidateUpdate_EmptyName(t *testing.T) {
	t.Parallel()
	name := "   "
	u := &domain.PartialCourierUpdate{
		ID:   1,
		Name: &name,
	}
	err := validateUpdate(u)
	if !errors.Is(err, apperr.Invalid) {
		t.Fatalf("expected Invalid for empty name, got %v", err)
	}
}

func TestValidateUpdate_InvalidPhone(t *testing.T) {
	t.Parallel()
	phone := "123"
	u := &domain.PartialCourierUpdate{
		ID:    1,
		Phone: &phone,
	}
	err := validateUpdate(u)
	if !errors.Is(err, apperr.Invalid) {
		t.Fatalf("expected Invalid for bad phone, got %v", err)
	}
}

func TestValidateUpdate_InvalidStatus(t *testing.T) {
	t.Parallel()
	status := domain.CourierStatus("bad")
	u := &domain.PartialCourierUpdate{
		ID:     1,
		Status: &status,
	}
	err := validateUpdate(u)
	if !errors.Is(err, apperr.Invalid) {
		t.Fatalf("expected Invalid for bad status, got %v", err)
	}
}

func TestValidateUpdate_InvalidTransportType(t *testing.T) {
	t.Parallel()

	transportType := domain.CourierTransportType("teleport")
	u := &domain.PartialCourierUpdate{
		ID:            1,
		TransportType: &transportType,
	}

	err := validateUpdate(u)
	if !errors.Is(err, apperr.Invalid) {
		t.Fatalf("expected Invalid for bad transport type, got %v", err)
	}
}

func TestValidateUpdate_ValidUpdatePasses(t *testing.T) {
	t.Parallel()
	name := "Artem"
	phone := "+70000000000"
	status := domain.StatusAvailable
	tt := domain.TransportTypeFoot

	u := &domain.PartialCourierUpdate{
		ID:            1,
		Name:          &name,
		Phone:         &phone,
		Status:        &status,
		TransportType: &tt,
	}
	if err := validateUpdate(u); err != nil {
		t.Fatalf("expected nil error for valid update, got %v", err)
	}
}
