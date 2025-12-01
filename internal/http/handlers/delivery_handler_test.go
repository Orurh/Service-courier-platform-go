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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
			require.Equal(t, "order-123", orderID)
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

	assert.Equal(t, http.StatusOK, rr.Code)

	expectedJSON := `{
        "courier_id": 42,
        "order_id": "order-123",
        "transport_type": "car",
        "delivery_deadline": "2025-01-02T03:04:05Z"
    }`
	assert.JSONEq(t, expectedJSON, rr.Body.String())
}

func TestDeliveryHandler_Assign_Invalid(t *testing.T) {
	t.Parallel()

	body := `{"order_id":""}`
	req := httptest.NewRequest(http.MethodPost, "/delivery/assign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	uc := &stubDeliveryUsecase{
		assignFn: func(ctx context.Context, orderID string) (domain.AssignResult, error) {
			return domain.AssignResult{}, apperr.ErrInvalid
		},
	}

	h := NewDeliveryHandler(uc)
	h.Assign(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.JSONEq(t, `{"error": "invalid input"}`, rr.Body.String())
}

func TestDeliveryHandler_Assign_Conflict(t *testing.T) {
	t.Parallel()
	body := `{"order_id":"order-123"}`
	req := httptest.NewRequest(http.MethodPost, "/delivery/assign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	uc := &stubDeliveryUsecase{
		assignFn: func(ctx context.Context, orderID string) (domain.AssignResult, error) {
			require.Equal(t, "order-123", orderID)
			return domain.AssignResult{}, apperr.ErrConflict
		},
	}

	h := NewDeliveryHandler(uc)

	h.Assign(rr, req)

	require.Equal(t, http.StatusConflict, rr.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Contains(t, resp, "error")
	require.NotEmpty(t, resp["error"])
	require.Contains(t, resp["error"], "no available couriers")
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

	require.Equal(t, http.StatusInternalServerError, rr.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Contains(t, resp, "error")
	require.NotEmpty(t, resp["error"])
}

func TestDeliveryHandler_Assign_InvalidJSON(t *testing.T) {
	t.Parallel()

	body := `{"order_id":`
	req := httptest.NewRequest(http.MethodPost, "/delivery/assign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	uc := &stubDeliveryUsecase{
		assignFn: func(ctx context.Context, orderID string) (domain.AssignResult, error) {
			require.FailNow(t, "usecase.Assign must not be called on invalid json")
			return domain.AssignResult{}, nil
		},
	}

	h := NewDeliveryHandler(uc)

	h.Assign(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Contains(t, resp, "error")
	require.NotEmpty(t, resp["error"])
}

func TestDeliveryHandler_Unassign_OK(t *testing.T) {
	t.Parallel()

	body := `{"order_id":"order-123"}`
	req := httptest.NewRequest(http.MethodPost, "/delivery/unassign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	uc := &stubDeliveryUsecase{
		unassignFn: func(ctx context.Context, orderID string) (domain.UnassignResult, error) {
			require.Equal(t, "order-123", orderID)
			return domain.UnassignResult{
				CourierID: 10,
				OrderID:   orderID,
				Status:    "unassigned",
			}, nil
		},
	}

	h := NewDeliveryHandler(uc)
	h.Unassign(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	expectedJSON := `{
        "order_id": "order-123",
        "status": "unassigned",
        "courier_id": 10
    }`
	assert.JSONEq(t, expectedJSON, rr.Body.String())
}

func TestDeliveryHandler_Unassign_NotFound(t *testing.T) {
	t.Parallel()

	body := `{"order_id":"order-404"}`
	req := httptest.NewRequest(http.MethodPost, "/delivery/unassign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	uc := &stubDeliveryUsecase{
		unassignFn: func(ctx context.Context, orderID string) (domain.UnassignResult, error) {
			require.Equal(t, "order-404", orderID)
			return domain.UnassignResult{}, apperr.ErrNotFound
		},
	}

	h := NewDeliveryHandler(uc)

	h.Unassign(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Contains(t, resp, "error")
	require.NotEmpty(t, resp["error"])
	require.Contains(t, resp["error"], "delivery not found")
}

func TestDeliveryHandler_Unassign_Invalid(t *testing.T) {
	t.Parallel()

	body := `{"order_id":"bad"}`
	req := httptest.NewRequest(http.MethodPost, "/delivery/unassign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	uc := &stubDeliveryUsecase{
		unassignFn: func(ctx context.Context, orderID string) (domain.UnassignResult, error) {
			return domain.UnassignResult{}, apperr.ErrInvalid
		},
	}

	h := NewDeliveryHandler(uc)

	h.Unassign(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Contains(t, resp, "error")
	require.NotEmpty(t, resp["error"])
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

	require.Equal(t, http.StatusInternalServerError, rr.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	require.Contains(t, resp, "error")
	require.NotEmpty(t, resp["error"])
}

func TestDeliveryHandler_Unassign_InvalidJSON(t *testing.T) {
	t.Parallel()

	body := `{"order_id":`
	req := httptest.NewRequest(http.MethodPost, "/delivery/unassign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	uc := &stubDeliveryUsecase{
		unassignFn: func(ctx context.Context, orderID string) (domain.UnassignResult, error) {
			require.FailNow(t, "usecase.Unassign must not be called on invalid json")
			return domain.UnassignResult{}, nil
		},
	}

	h := NewDeliveryHandler(uc)
	h.Unassign(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	assert.JSONEq(t, `{"error": "invalid json"}`, rr.Body.String())
}
