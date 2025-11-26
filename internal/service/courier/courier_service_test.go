package courier

import (
	"context"
	"errors"
	"testing"
	"time"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"
)

type mockCourierRepo struct {
	getFn           func(ctx context.Context, id int64) (*domain.Courier, error)
	listFn          func(ctx context.Context, limit, offset *int) ([]domain.Courier, error)
	createFn        func(ctx context.Context, c *domain.Courier) (int64, error)
	updatePartialFn func(ctx context.Context, u domain.PartialCourierUpdate) (bool, error)
}

func (m *mockCourierRepo) Get(ctx context.Context, id int64) (*domain.Courier, error) {
	return m.getFn(ctx, id)
}

func (m *mockCourierRepo) List(ctx context.Context, limit, offset *int) ([]domain.Courier, error) {
	return m.listFn(ctx, limit, offset)
}

func (m *mockCourierRepo) Create(ctx context.Context, c *domain.Courier) (int64, error) {
	return m.createFn(ctx, c)
}

func (m *mockCourierRepo) UpdatePartial(ctx context.Context, u domain.PartialCourierUpdate) (bool, error) {
	return m.updatePartialFn(ctx, u)
}

func TestNewService_ZeroTimeoutUsesDefault(t *testing.T) {
	t.Parallel()

	repo := &mockCourierRepo{}
	service := NewService(repo, 0)
	if service.operationTimeout != 3*time.Second {
		t.Fatalf("default timeout 3s, got %v", service.operationTimeout)
	}
}

func TestNewService_PositiveTimeoutKept(t *testing.T) {
	t.Parallel()

	repo := &mockCourierRepo{}
	service := NewService(repo, 5*time.Second)
	if service.operationTimeout != 5*time.Second {
		t.Fatalf("expected timeout 5s, got %v", service.operationTimeout)
	}
}

func TestNewService_NegativeTimeoutUsesDefault(t *testing.T) {
	t.Parallel()

	repo := &mockCourierRepo{}
	service := NewService(repo, -10*time.Second)
	if service.operationTimeout != 3*time.Second {
		t.Fatalf("negative timeout should default to 3s, got %v", service.operationTimeout)
	}
}

func TestService_Get_Success(t *testing.T) {
	t.Parallel()

	expected := &domain.Courier{
		ID:            50,
		Name:          "courier",
		Phone:         "+71111111111",
		Status:        domain.StatusAvailable,
		TransportType: domain.TransportTypeFoot,
	}

	repo := &mockCourierRepo{
		getFn: func(ctx context.Context, id int64) (*domain.Courier, error) {
			if id != expected.ID {
				t.Fatalf("expected id %d, got %d", expected.ID, id)
			}
			return expected, nil
		},
	}

	service := NewService(repo, time.Second)

	got, err := service.Get(context.Background(), expected.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Fatalf("expected %#v, got %#v", expected, got)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	t.Parallel()

	repo := &mockCourierRepo{
		getFn: func(ctx context.Context, id int64) (*domain.Courier, error) {
			return nil, nil
		},
	}

	service := NewService(repo, time.Second)

	got, err := service.Get(context.Background(), 1)
	if !errors.Is(err, apperr.NotFound) {
		t.Fatalf("expected NotFound, got err=%v", err)
	}
	if got != nil {
		t.Fatalf("expected nil courier, got %#v", got)
	}
}

func TestService_Get_RepoError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	repo := &mockCourierRepo{
		getFn: func(ctx context.Context, id int64) (*domain.Courier, error) {
			return nil, wantErr
		},
	}

	service := NewService(repo, time.Second)

	_, err := service.Get(context.Background(), 1)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected repo error %v, got %v", wantErr, err)
	}
}

func TestService_List_Success(t *testing.T) {
	t.Parallel()

	limit, offset := 10, 5

	expected := []domain.Courier{
		{ID: 1, Name: "first"},
		{ID: 2, Name: "second"},
	}

	repo := &mockCourierRepo{
		listFn: func(ctx context.Context, gotLimit, gotOffset *int) ([]domain.Courier, error) {
			if gotLimit == nil || *gotLimit != limit {
				t.Fatalf("expected limit %d, got %v", limit, gotLimit)
			}
			if gotOffset == nil || *gotOffset != offset {
				t.Fatalf("expected offset %d, got %v", offset, gotOffset)
			}
			return expected, nil
		},
	}

	service := NewService(repo, time.Second)

	res, err := service.List(context.Background(), &limit, &offset)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(res))
	}
}

func TestService_List_RepoError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("db down")
	repo := &mockCourierRepo{
		listFn: func(ctx context.Context, limit, offset *int) ([]domain.Courier, error) {
			return nil, wantErr
		},
	}

	service := NewService(repo, time.Second)

	_, err := service.List(context.Background(), nil, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected repo error %v, got %v", wantErr, err)
	}
}

func TestService_Create_InvalidInput(t *testing.T) {
	t.Parallel()

	repo := &mockCourierRepo{
		createFn: func(ctx context.Context, c *domain.Courier) (int64, error) {
			t.Fatal("Create should not be called on invalid input")
			return 0, nil
		},
	}

	service := NewService(repo, time.Second)

	c := &domain.Courier{
		Name:          " ",
		Phone:         "123",
		Status:        domain.StatusAvailable,
		TransportType: domain.TransportTypeFoot,
	}

	_, err := service.Create(context.Background(), c)
	if !errors.Is(err, apperr.Invalid) {
		t.Fatalf("expected Invalid error, got %v", err)
	}
}

func TestService_Create_SetsDefaultTransportAndCallsRepo(t *testing.T) {
	t.Parallel()

	var got *domain.Courier
	repo := &mockCourierRepo{
		createFn: func(ctx context.Context, c *domain.Courier) (int64, error) {
			got = c
			return 123, nil
		},
	}

	service := NewService(repo, time.Second)

	c := &domain.Courier{
		Name:   "John",
		Phone:  "+79990000000",
		Status: domain.StatusAvailable,
	}

	id, err := service.Create(context.Background(), c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 123 {
		t.Fatalf("expected id 123, got %d", id)
	}
	if got == nil {
		t.Fatal("repo.Create was not called")
	}
	if got.TransportType != domain.TransportTypeFoot {
		t.Fatalf("expected default transport type %q, got %q", domain.TransportTypeFoot, got.TransportType)
	}
}

func TestService_UpdatePartial_Invalid(t *testing.T) {
	t.Parallel()

	repo := &mockCourierRepo{
		updatePartialFn: func(ctx context.Context, u domain.PartialCourierUpdate) (bool, error) {
			t.Fatal("UpdatePartial should not be called on invalid input")
			return false, nil
		},
	}

	service := NewService(repo, time.Second)
	u := domain.PartialCourierUpdate{}

	_, err := service.UpdatePartial(context.Background(), u)
	if !errors.Is(err, apperr.Invalid) {
		t.Fatalf("expected Invalid error, got %v", err)
	}
}

func TestService_UpdatePartial_Success(t *testing.T) {
	t.Parallel()

	name := "New Name"
	u := domain.PartialCourierUpdate{
		ID:   1,
		Name: &name,
	}

	var gotUpdate domain.PartialCourierUpdate
	repo := &mockCourierRepo{
		updatePartialFn: func(ctx context.Context, upd domain.PartialCourierUpdate) (bool, error) {
			gotUpdate = upd
			return true, nil
		},
	}

	service := NewService(repo, time.Second)

	ok, err := service.UpdatePartial(context.Background(), u)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true, got false")
	}
	if gotUpdate.ID != u.ID || gotUpdate.Name == nil || *gotUpdate.Name != *u.Name {
		t.Fatalf("repo received wrong update: %#v", gotUpdate)
	}
}

func TestService_UpdatePartial_NotFound(t *testing.T) {
	t.Parallel()

	name := "New Name"
	u := domain.PartialCourierUpdate{
		ID:   50,
		Name: &name,
	}

	repo := &mockCourierRepo{
		updatePartialFn: func(ctx context.Context, upd domain.PartialCourierUpdate) (bool, error) {
			return false, nil
		},
	}

	service := NewService(repo, time.Second)

	ok, err := service.UpdatePartial(context.Background(), u)
	if ok {
		t.Fatalf("expected ok=false on not found")
	}
	if !errors.Is(err, apperr.NotFound) {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestService_UpdatePartial_RepoError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("repo error")
	name := "New Name"
	u := domain.PartialCourierUpdate{
		ID:   1,
		Name: &name,
	}

	repo := &mockCourierRepo{
		updatePartialFn: func(ctx context.Context, upd domain.PartialCourierUpdate) (bool, error) {
			return false, wantErr
		},
	}

	service := NewService(repo, time.Second)

	_, err := service.UpdatePartial(context.Background(), u)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected repo error %v, got %v", wantErr, err)
	}
}
