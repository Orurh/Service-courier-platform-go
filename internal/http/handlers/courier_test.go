package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"

	"github.com/go-chi/chi/v5"
)

type stubCourierUsecase struct {
	getFn           func(ctx context.Context, id int64) (*domain.Courier, error)
	listFn          func(ctx context.Context, limit, offset *int) ([]domain.Courier, error)
	createFn        func(ctx context.Context, c *domain.Courier) (int64, error)
	updatePartialFn func(ctx context.Context, u domain.PartialCourierUpdate) (bool, error)
}

func (s *stubCourierUsecase) Get(ctx context.Context, id int64) (*domain.Courier, error) {
	return s.getFn(ctx, id)
}

func (s *stubCourierUsecase) List(ctx context.Context, limit, offset *int) ([]domain.Courier, error) {
	return s.listFn(ctx, limit, offset)
}

func (s *stubCourierUsecase) Create(ctx context.Context, c *domain.Courier) (int64, error) {
	return s.createFn(ctx, c)
}

func (s *stubCourierUsecase) UpdatePartial(ctx context.Context, u domain.PartialCourierUpdate) (bool, error) {
	return s.updatePartialFn(ctx, u)
}

func TestCourierHandler_GetByID_OK(t *testing.T) {
	t.Parallel()

	expected := &domain.Courier{
		ID:    99,
		Name:  "Artem",
		Phone: "+70000000000",
	}

	uc := &stubCourierUsecase{
		getFn: func(ctx context.Context, id int64) (*domain.Courier, error) {
			if id != expected.ID {
				t.Fatalf("expected id %d, got %d", expected.ID, id)
			}
			return expected, nil
		},
	}

	h := NewCourierHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/courier/99", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", "99")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rr := httptest.NewRecorder()

	h.GetByID(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp courierDTO
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.ID != expected.ID {
		t.Fatalf("expected ID %d, got %d", expected.ID, resp.ID)
	}
	if resp.Name != expected.Name {
		t.Fatalf("expected Name %q, got %q", expected.Name, resp.Name)
	}
	if resp.Phone != expected.Phone {
		t.Fatalf("expected Phone %q, got %q", expected.Phone, resp.Phone)
	}
}

func TestCourierHandler_GetByID_InvalidID(t *testing.T) {
	t.Parallel()

	h := NewCourierHandler(&stubCourierUsecase{
		getFn: func(ctx context.Context, id int64) (*domain.Courier, error) {
			t.Fatalf("usecase.Get should not be called on invalid id")
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/courier/abc", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", "abc")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rr := httptest.NewRecorder()
	h.GetByID(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCourierHandler_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		getFn: func(ctx context.Context, id int64) (*domain.Courier, error) {
			return nil, apperr.NotFound
		},
	}
	h := NewCourierHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/courier/10", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", "10")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rr := httptest.NewRecorder()
	h.GetByID(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestCourierHandler_GetByID_InternalError(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		getFn: func(ctx context.Context, id int64) (*domain.Courier, error) {
			return nil, errors.New("db down")
		},
	}
	h := NewCourierHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/courier/10", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", "10")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rr := httptest.NewRecorder()
	h.GetByID(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestCourierHandler_List_OK(t *testing.T) {
	t.Parallel()

	expected := []domain.Courier{
		{ID: 1, Name: "A"},
		{ID: 2, Name: "B"},
	}

	var gotLimit, gotOffset *int

	uc := &stubCourierUsecase{
		listFn: func(ctx context.Context, limit, offset *int) ([]domain.Courier, error) {
			gotLimit, gotOffset = limit, offset
			return expected, nil
		},
	}
	h := NewCourierHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/couriers?limit=10&offset=5", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	if gotLimit == nil || *gotLimit != 10 {
		t.Fatalf("expected limit=10, got %#v", gotLimit)
	}
	if gotOffset == nil || *gotOffset != 5 {
		t.Fatalf("expected offset=5, got %#v", gotOffset)
	}

	var resp []courierDTO
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(resp))
	}
}

func TestCourierHandler_List_InvalidLimit(t *testing.T) {
	t.Parallel()

	h := NewCourierHandler(&stubCourierUsecase{
		listFn: func(ctx context.Context, limit, offset *int) ([]domain.Courier, error) {
			t.Fatalf("List should not be called when limit is invalid")
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/couriers?limit=abc", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCourierHandler_List_InvalidOffset(t *testing.T) {
	t.Parallel()

	h := NewCourierHandler(&stubCourierUsecase{
		listFn: func(ctx context.Context, limit, offset *int) ([]domain.Courier, error) {
			t.Fatalf("List should not be called when offset is invalid")
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/couriers?offset=-1", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCourierHandler_List_InternalError(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		listFn: func(ctx context.Context, limit, offset *int) ([]domain.Courier, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewCourierHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/couriers", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestCourierHandler_Create_OK(t *testing.T) {
	t.Parallel()

	var gotModel *domain.Courier

	uc := &stubCourierUsecase{
		createFn: func(ctx context.Context, c *domain.Courier) (int64, error) {
			gotModel = c
			return 42, nil
		},
	}
	h := NewCourierHandler(uc)

	body := `{"name":"Artem","phone":"+70000000000","status":"available","transport_type":"on_foot"}`
	req := httptest.NewRequest(http.MethodPost, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/courier/42" {
		t.Fatalf("expected Location /courier/42, got %q", loc)
	}
	if gotModel == nil || gotModel.Name != "Artem" {
		t.Fatalf("unexpected model: %#v", gotModel)
	}
}

func TestCourierHandler_Create_Invalid(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		createFn: func(ctx context.Context, c *domain.Courier) (int64, error) {
			return 0, apperr.Invalid
		},
	}
	h := NewCourierHandler(uc)

	body := `{"name":"","phone":"bad"}`
	req := httptest.NewRequest(http.MethodPost, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCourierHandler_Create_Conflict(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		createFn: func(ctx context.Context, c *domain.Courier) (int64, error) {
			return 0, apperr.Conflict
		},
	}
	h := NewCourierHandler(uc)

	body := `{"name":"Artem","phone":"+70000000000"}`
	req := httptest.NewRequest(http.MethodPost, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected %d, got %d", http.StatusConflict, rr.Code)
	}
}

func TestCourierHandler_Create_InternalError(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		createFn: func(ctx context.Context, c *domain.Courier) (int64, error) {
			return 0, errors.New("db error")
		},
	}
	h := NewCourierHandler(uc)

	body := `{"name":"Artem","phone":"+70000000000"}`
	req := httptest.NewRequest(http.MethodPost, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestCourierHandler_Update_OK(t *testing.T) {
	t.Parallel()

	var gotUpdate domain.PartialCourierUpdate

	uc := &stubCourierUsecase{
		updatePartialFn: func(ctx context.Context, u domain.PartialCourierUpdate) (bool, error) {
			gotUpdate = u
			return true, nil
		},
	}
	h := NewCourierHandler(uc)

	body := `{"id":1,"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPut, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}
	if gotUpdate.ID != 1 || gotUpdate.Name == nil || *gotUpdate.Name != "New Name" {
		t.Fatalf("unexpected update: %#v", gotUpdate)
	}
}

func TestCourierHandler_Update_Invalid(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		updatePartialFn: func(ctx context.Context, u domain.PartialCourierUpdate) (bool, error) {
			return false, apperr.Invalid
		},
	}
	h := NewCourierHandler(uc)

	body := `{"id":0}`
	req := httptest.NewRequest(http.MethodPut, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCourierHandler_Update_Conflict(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		updatePartialFn: func(ctx context.Context, u domain.PartialCourierUpdate) (bool, error) {
			return false, apperr.Conflict
		},
	}
	h := NewCourierHandler(uc)

	body := `{"id":1,"phone":"+70000000000"}`
	req := httptest.NewRequest(http.MethodPut, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected %d, got %d", http.StatusConflict, rr.Code)
	}
}

func TestCourierHandler_Update_NotFound(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		updatePartialFn: func(ctx context.Context, u domain.PartialCourierUpdate) (bool, error) {
			return false, apperr.NotFound
		},
	}
	h := NewCourierHandler(uc)

	body := `{"id":123,"name":"X"}`
	req := httptest.NewRequest(http.MethodPut, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestCourierHandler_Update_InternalError(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		updatePartialFn: func(ctx context.Context, u domain.PartialCourierUpdate) (bool, error) {
			return false, errors.New("db error")
		},
	}
	h := NewCourierHandler(uc)

	body := `{"id":1,"name":"X"}`
	req := httptest.NewRequest(http.MethodPut, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}
