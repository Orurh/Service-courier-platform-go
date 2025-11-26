// internal/http/handlers/delivery_handler_test.go
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"
)

type stubDeliveryUsecase struct {
	assignFn   func(ctx context.Context, orderID string) (domain.AssignResult, error)
	unassignFn func(ctx context.Context, orderID string) (domain.UnassignResult, error)
}

func (s *stubDeliveryUsecase) Assign(ctx context.Context, orderID string) (domain.AssignResult, error) {
	if s.assignFn == nil {
		panic("Assign not expected in this test")
	}
	return s.assignFn(ctx, orderID)
}

func (s *stubDeliveryUsecase) Unassign(ctx context.Context, orderID string) (domain.UnassignResult, error) {
	if s.unassignFn == nil {
		panic("Unassign not expected in this test")
	}
	return s.unassignFn(ctx, orderID)
}

func TestDeliveryHandler_Assign_OK(t *testing.T) {
	t.Parallel()

	body := `{"order_id":"order-123"}`
	req := httptest.NewRequest(http.MethodPost, "/delivery/assign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	deadline := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)

	uc := &stubDeliveryUsecase{
		assignFn: func(ctx context.Context, orderID string) (domain.AssignResult, error) {
			if orderID != "order-123" {
				t.Fatalf("expected orderID %q, got %q", "order-123", orderID)
			}
			return domain.AssignResult{
				CourierID:     42,
				OrderID:       orderID,
				TransportType: domain.TransportTypeCar,
				Deadline:      deadline,
			}, nil
		},
	}

	h := NewDeliveryHandler(uc)

	h.Assign(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp assignDeliveryResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.CourierID != 42 ||
		resp.OrderID != "order-123" ||
		resp.TransportType != string(domain.TransportTypeCar) ||
		!resp.DeliveryDeadline.Equal(deadline) {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestDeliveryHandler_Assign_Invalid(t *testing.T) {
	t.Parallel()

	body := `{"order_id":""}` // но это неважно, всё равно вернём apperr.Invalid из usecase
	req := httptest.NewRequest(http.MethodPost, "/delivery/assign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	uc := &stubDeliveryUsecase{
		assignFn: func(ctx context.Context, orderID string) (domain.AssignResult, error) {
			return domain.AssignResult{}, apperr.Invalid
		},
	}

	h := NewDeliveryHandler(uc)

	h.Assign(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] == "" {
		t.Fatalf("expected non-empty error message, got %#v", resp)
	}
}

func TestDeliveryHandler_Assign_Conflict(t *testing.T) {
	t.Parallel()
	body := `{"order_id":"order-123"}`
	req := httptest.NewRequest(http.MethodPost, "/delivery/assign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	uc := &stubDeliveryUsecase{
		assignFn: func(ctx context.Context, orderID string) (domain.AssignResult, error) {
			if orderID != "order-123" {
				t.Fatalf("expected orderID %q, got %q", "order-123", orderID)
			}
			return domain.AssignResult{}, apperr.Conflict
		},
	}

	h := NewDeliveryHandler(uc)

	h.Assign(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["error"] == "" {
		t.Fatalf(`expected non-empty "error" message in body, got: %#v`, resp)
	}
}

func TestDeliveryHandler_Assign_InternalError(t *testing.T) {
	t.Parallel()

	body := `{"order_id":"order-123"}`
	req := httptest.NewRequest(http.MethodPost, "/delivery/assign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	uc := &stubDeliveryUsecase{
		assignFn: func(ctx context.Context, orderID string) (domain.AssignResult, error) {
			return domain.AssignResult{}, errors.New("boom")
		},
	}

	h := NewDeliveryHandler(uc)

	h.Assign(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] == "" {
		t.Fatalf("expected non-empty error message, got %#v", resp)
	}
}

func TestDeliveryHandler_Assign_InvalidJSON(t *testing.T) {
	t.Parallel()

	// Некорректный JSON, decodeJSON должен отстрелить 400 и не вызвать usecase
	body := `{"order_id":`
	req := httptest.NewRequest(http.MethodPost, "/delivery/assign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	uc := &stubDeliveryUsecase{
		assignFn: func(ctx context.Context, orderID string) (domain.AssignResult, error) {
			t.Fatalf("usecase.Assign must not be called on invalid json")
			return domain.AssignResult{}, nil
		},
	}

	h := NewDeliveryHandler(uc)

	h.Assign(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] == "" {
		t.Fatalf("expected non-empty error message, got %#v", resp)
	}
}

func TestDeliveryHandler_Unassign_OK(t *testing.T) {
	t.Parallel()

	body := `{"order_id":"order-123"}`
	req := httptest.NewRequest(http.MethodPost, "/delivery/unassign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	uc := &stubDeliveryUsecase{
		unassignFn: func(ctx context.Context, orderID string) (domain.UnassignResult, error) {
			if orderID != "order-123" {
				t.Fatalf("expected orderID %q, got %q", "order-123", orderID)
			}
			return domain.UnassignResult{
				CourierID: 10,
				OrderID:   orderID,
				Status:    "unassigned",
			}, nil
		},
	}

	h := NewDeliveryHandler(uc)

	h.Unassign(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp unassignDeliveryResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.OrderID != "order-123" || resp.Status == "" || resp.CourierID != 10 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestDeliveryHandler_Unassign_NotFound(t *testing.T) {
	t.Parallel()

	body := `{"order_id":"order-404"}`
	req := httptest.NewRequest(http.MethodPost, "/delivery/unassign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	uc := &stubDeliveryUsecase{
		unassignFn: func(ctx context.Context, orderID string) (domain.UnassignResult, error) {
			if orderID != "order-404" {
				t.Fatalf("expected orderID %q, got %q", "order-404", orderID)
			}
			return domain.UnassignResult{}, apperr.NotFound
		},
	}

	h := NewDeliveryHandler(uc)

	h.Unassign(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] == "" {
		t.Fatalf(`expected non-empty "error" message, got: %#v`, resp)
	}
}

func TestDeliveryHandler_Unassign_Invalid(t *testing.T) {
	t.Parallel()

	body := `{"order_id":"bad"}` // значение не важно, ошибка сгенерирует usecase
	req := httptest.NewRequest(http.MethodPost, "/delivery/unassign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	uc := &stubDeliveryUsecase{
		unassignFn: func(ctx context.Context, orderID string) (domain.UnassignResult, error) {
			return domain.UnassignResult{}, apperr.Invalid
		},
	}

	h := NewDeliveryHandler(uc)

	h.Unassign(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] == "" {
		t.Fatalf("expected non-empty error message, got %#v", resp)
	}
}

func TestDeliveryHandler_Unassign_InternalError(t *testing.T) {
	t.Parallel()

	body := `{"order_id":"order-123"}`
	req := httptest.NewRequest(http.MethodPost, "/delivery/unassign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	uc := &stubDeliveryUsecase{
		unassignFn: func(ctx context.Context, orderID string) (domain.UnassignResult, error) {
			return domain.UnassignResult{}, errors.New("boom")
		},
	}

	h := NewDeliveryHandler(uc)

	h.Unassign(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] == "" {
		t.Fatalf("expected non-empty error message, got %#v", resp)
	}
}

func TestDeliveryHandler_Unassign_InvalidJSON(t *testing.T) {
	t.Parallel()

	body := `{"order_id":` // сломанный JSON
	req := httptest.NewRequest(http.MethodPost, "/delivery/unassign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	uc := &stubDeliveryUsecase{
		unassignFn: func(ctx context.Context, orderID string) (domain.UnassignResult, error) {
			t.Fatalf("usecase.Unassign must not be called on invalid json")
			return domain.UnassignResult{}, nil
		},
	}

	h := NewDeliveryHandler(uc)

	h.Unassign(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] == "" {
		t.Fatalf("expected non-empty error message, got %#v", resp)
	}
}
