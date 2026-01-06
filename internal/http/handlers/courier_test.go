package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/http/handlers"
)

type courierResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

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
			require.Equal(t, expected.ID, id)
			return expected, nil
		},
	}

	h := handlers.NewCourierHandler(testLogger(), uc)

	req := httptest.NewRequest(http.MethodGet, "/courier/99", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", "99")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rr := httptest.NewRecorder()

	h.GetByID(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp courierResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	require.Equal(t, expected.ID, resp.ID)
	require.Equal(t, expected.Name, resp.Name)
	require.Equal(t, expected.Phone, resp.Phone)
}

func TestCourierHandler_GetByID_InvalidID(t *testing.T) {
	t.Parallel()

	h := handlers.NewCourierHandler(testLogger(), &stubCourierUsecase{
		getFn: func(ctx context.Context, id int64) (*domain.Courier, error) {
			require.FailNow(t, "usecase.Get should not be called on invalid id")
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/courier/abc", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", "abc")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rr := httptest.NewRecorder()
	h.GetByID(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCourierHandler_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		getFn: func(ctx context.Context, id int64) (*domain.Courier, error) {
			return nil, apperr.ErrNotFound
		},
	}
	h := handlers.NewCourierHandler(testLogger(), uc)

	req := httptest.NewRequest(http.MethodGet, "/courier/10", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", "10")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rr := httptest.NewRecorder()
	h.GetByID(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestCourierHandler_GetByID_InternalError(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		getFn: func(ctx context.Context, id int64) (*domain.Courier, error) {
			return nil, errors.New("db down")
		},
	}
	h := handlers.NewCourierHandler(testLogger(), uc)

	req := httptest.NewRequest(http.MethodGet, "/courier/10", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", "10")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rr := httptest.NewRecorder()
	h.GetByID(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
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
	h := handlers.NewCourierHandler(testLogger(), uc)

	req := httptest.NewRequest(http.MethodGet, "/couriers?limit=10&offset=5", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.NotNil(t, gotLimit)
	require.Equal(t, 10, *gotLimit)
	require.NotNil(t, gotOffset)
	require.Equal(t, 5, *gotOffset)

	var resp []courierResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	require.Len(t, resp, len(expected))
}

func TestCourierHandler_List_InvalidLimit(t *testing.T) {
	t.Parallel()

	h := handlers.NewCourierHandler(testLogger(), &stubCourierUsecase{
		listFn: func(ctx context.Context, limit, offset *int) ([]domain.Courier, error) {
			require.FailNow(t, "List should not be called when limit is invalid")
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/couriers?limit=abc", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCourierHandler_List_InvalidOffset(t *testing.T) {
	t.Parallel()

	h := handlers.NewCourierHandler(testLogger(), &stubCourierUsecase{
		listFn: func(ctx context.Context, limit, offset *int) ([]domain.Courier, error) {
			require.FailNow(t, "List should not be called when offset is invalid")
			return nil, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/couriers?offset=-1", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCourierHandler_List_InternalError(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		listFn: func(ctx context.Context, limit, offset *int) ([]domain.Courier, error) {
			return nil, errors.New("db error")
		},
	}
	h := handlers.NewCourierHandler(testLogger(), uc)

	req := httptest.NewRequest(http.MethodGet, "/couriers", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
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
	h := handlers.NewCourierHandler(testLogger(), uc)

	body := `{"name":"Artem","phone":"+70000000000","status":"available","transport_type":"on_foot"}`
	req := httptest.NewRequest(http.MethodPost, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	require.Equal(t, "/courier/42", rr.Header().Get("Location"))
	require.NotNil(t, gotModel)
	require.Equal(t, "Artem", gotModel.Name)
}

func TestCourierHandler_Create_Invalid(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		createFn: func(ctx context.Context, c *domain.Courier) (int64, error) {
			return 0, apperr.ErrInvalid
		},
	}
	h := handlers.NewCourierHandler(testLogger(), uc)

	body := `{"name":"","phone":"bad"}`
	req := httptest.NewRequest(http.MethodPost, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCourierHandler_Create_Conflict(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		createFn: func(ctx context.Context, c *domain.Courier) (int64, error) {
			return 0, apperr.ErrConflict
		},
	}
	h := handlers.NewCourierHandler(testLogger(), uc)

	body := `{"name":"Artem","phone":"+70000000000"}`
	req := httptest.NewRequest(http.MethodPost, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	require.Equal(t, http.StatusConflict, rr.Code)
}

func TestCourierHandler_Create_InternalError(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		createFn: func(ctx context.Context, c *domain.Courier) (int64, error) {
			return 0, errors.New("db error")
		},
	}
	h := handlers.NewCourierHandler(testLogger(), uc)

	body := `{"name":"Artem","phone":"+70000000000"}`
	req := httptest.NewRequest(http.MethodPost, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
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
	h := handlers.NewCourierHandler(testLogger(), uc)

	body := `{"id":1,"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPut, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, int64(1), gotUpdate.ID)
	require.NotNil(t, gotUpdate.Name)
	require.Equal(t, "New Name", *gotUpdate.Name)
}

func TestCourierHandler_Update_Invalid(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		updatePartialFn: func(ctx context.Context, u domain.PartialCourierUpdate) (bool, error) {
			return false, apperr.ErrInvalid
		},
	}
	h := handlers.NewCourierHandler(testLogger(), uc)

	body := `{"id":0}`
	req := httptest.NewRequest(http.MethodPut, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCourierHandler_Update_Conflict(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		updatePartialFn: func(ctx context.Context, u domain.PartialCourierUpdate) (bool, error) {
			return false, apperr.ErrConflict
		},
	}
	h := handlers.NewCourierHandler(testLogger(), uc)

	body := `{"id":1,"phone":"+70000000000"}`
	req := httptest.NewRequest(http.MethodPut, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	require.Equal(t, http.StatusConflict, rr.Code)
}

func TestCourierHandler_Update_NotFound(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		updatePartialFn: func(ctx context.Context, u domain.PartialCourierUpdate) (bool, error) {
			return false, apperr.ErrNotFound
		},
	}
	h := handlers.NewCourierHandler(testLogger(), uc)

	body := `{"id":123,"name":"X"}`
	req := httptest.NewRequest(http.MethodPut, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestCourierHandler_Update_InternalError(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		updatePartialFn: func(ctx context.Context, u domain.PartialCourierUpdate) (bool, error) {
			return false, errors.New("db error")
		},
	}
	h := handlers.NewCourierHandler(testLogger(), uc)

	body := `{"id":1,"name":"X"}`
	req := httptest.NewRequest(http.MethodPut, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestCourierHandler_Create_BadJSON(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		createFn: func(ctx context.Context, c *domain.Courier) (int64, error) {
			require.FailNow(t, "Create must not be called on invalid JSON")
			return 0, nil
		},
	}
	h := handlers.NewCourierHandler(testLogger(), uc)

	body := `{"name": "Artem", "phone": "+70000000000",`
	req := httptest.NewRequest(http.MethodPost, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCourierHandler_Update_BadJSON(t *testing.T) {
	t.Parallel()

	uc := &stubCourierUsecase{
		updatePartialFn: func(ctx context.Context, u domain.PartialCourierUpdate) (bool, error) {
			require.FailNow(t, "UpdatePartial must not be called on invalid JSON")
			return false, nil
		},
	}
	h := handlers.NewCourierHandler(testLogger(), uc)

	body := `{"id": 1, "name": "New Name"`
	req := httptest.NewRequest(http.MethodPut, "/courier", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Update(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}
