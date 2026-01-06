package courier_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/service/courier"
)

func TestService_Get_Success(t *testing.T) {
	t.Parallel()

	expected := &domain.Courier{
		ID:            50,
		Name:          "Artem",
		Phone:         "+71111111111",
		Status:        domain.StatusAvailable,
		TransportType: domain.TransportTypeFoot,
	}

	ctrl := gomock.NewController(t)

	repo := NewMockcourierRepository(ctrl)
	repo.EXPECT().
		Get(gomock.Any(), expected.ID).
		Return(expected, nil)
	service := courier.NewService(repo, time.Second)

	got, err := service.Get(context.Background(), expected.ID)
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestService_Get_NotFound(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	repo := NewMockcourierRepository(ctrl)
	repo.EXPECT().
		Get(gomock.Any(), int64(1)).
		Return(nil, nil)

	service := courier.NewService(repo, time.Second)

	got, err := service.Get(context.Background(), 1)
	require.Error(t, err)
	require.Nil(t, got)
}

func TestService_Get_RepoError(t *testing.T) {
	t.Parallel()

	wantErr := assert.AnError

	ctrl := gomock.NewController(t)

	repo := NewMockcourierRepository(ctrl)
	repo.EXPECT().
		Get(gomock.Any(), int64(1)).
		Return(nil, wantErr)

	service := courier.NewService(repo, time.Second)

	_, err := service.Get(context.Background(), 1)
	require.ErrorIs(t, err, wantErr)
}

func TestService_List_Success(t *testing.T) {
	t.Parallel()

	limit, offset := 10, 5

	expected := []domain.Courier{
		{ID: 1, Name: "first"},
		{ID: 2, Name: "second"},
	}

	ctrl := gomock.NewController(t)

	repo := NewMockcourierRepository(ctrl)
	repo.EXPECT().
		List(gomock.Any(), &limit, &offset).
		Return(expected, nil)

	service := courier.NewService(repo, time.Second)

	res, err := service.List(context.Background(), &limit, &offset)
	require.NoError(t, err)
	require.Len(t, res, len(expected))

	for i := range expected {
		require.Equal(t, expected[i].ID, res[i].ID)
		require.Equal(t, expected[i].Name, res[i].Name)
	}
}

func TestService_List_RepoError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("db down")

	ctrl := gomock.NewController(t)

	repo := NewMockcourierRepository(ctrl)
	repo.EXPECT().
		List(gomock.Any(), gomock.Nil(), gomock.Nil()).
		Return(nil, wantErr)

	service := courier.NewService(repo, time.Second)

	_, err := service.List(context.Background(), nil, nil)
	require.ErrorIs(t, err, wantErr)
}

func TestService_Create_InvalidInput(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	repo := NewMockcourierRepository(ctrl)

	service := courier.NewService(repo, time.Second)

	c := &domain.Courier{
		Name:          " ",
		Phone:         "123",
		Status:        domain.StatusAvailable,
		TransportType: domain.TransportTypeFoot,
	}

	_, err := service.Create(context.Background(), c)
	require.ErrorIs(t, err, apperr.ErrInvalid)
}

func TestService_Create_SetsDefaultTransportAndCallsRepo(t *testing.T) {
	t.Parallel()

	var got *domain.Courier

	ctrl := gomock.NewController(t)

	repo := NewMockcourierRepository(ctrl)
	repo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, c *domain.Courier) (int64, error) {
			got = c
			return 123, nil
		})

	service := courier.NewService(repo, time.Second)

	c := &domain.Courier{
		Name:   "Artem",
		Phone:  "+79990000000",
		Status: domain.StatusAvailable,
	}

	id, err := service.Create(context.Background(), c)
	require.NoError(t, err)
	require.Equal(t, int64(123), id)
	require.NotNil(t, got)
	require.Equal(t, domain.TransportTypeFoot, got.TransportType)
}

func TestService_UpdatePartial_Invalid(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := NewMockcourierRepository(ctrl)

	service := courier.NewService(repo, time.Second)
	u := domain.PartialCourierUpdate{}

	_, err := service.UpdatePartial(context.Background(), u)
	require.ErrorIs(t, err, apperr.ErrInvalid)
}

func TestService_UpdatePartial_Success(t *testing.T) {
	t.Parallel()

	name := "Artem"
	u := domain.PartialCourierUpdate{
		ID:   1,
		Name: &name,
	}

	ctrl := gomock.NewController(t)

	var gotUpdate domain.PartialCourierUpdate

	repo := NewMockcourierRepository(ctrl)
	repo.EXPECT().
		UpdatePartial(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, upd domain.PartialCourierUpdate) (bool, error) {
			gotUpdate = upd
			return true, nil
		})

	service := courier.NewService(repo, time.Second)

	ok, err := service.UpdatePartial(context.Background(), u)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, u.ID, gotUpdate.ID)
	require.NotNil(t, gotUpdate.Name)
	require.Equal(t, *u.Name, *gotUpdate.Name)
}

func TestService_UpdatePartial_NotFound(t *testing.T) {
	t.Parallel()

	name := "Artem"
	u := domain.PartialCourierUpdate{
		ID:   50,
		Name: &name,
	}

	ctrl := gomock.NewController(t)

	repo := NewMockcourierRepository(ctrl)
	repo.EXPECT().
		UpdatePartial(gomock.Any(), u).
		Return(false, nil)

	service := courier.NewService(repo, time.Second)

	ok, err := service.UpdatePartial(context.Background(), u)
	require.False(t, ok)
	require.ErrorIs(t, err, apperr.ErrNotFound)
}

func TestService_UpdatePartial_RepoError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("repo err")
	name := "Artem"
	u := domain.PartialCourierUpdate{
		ID:   1,
		Name: &name,
	}

	ctrl := gomock.NewController(t)

	repo := NewMockcourierRepository(ctrl)
	repo.EXPECT().
		UpdatePartial(gomock.Any(), u).
		Return(false, wantErr)

	service := courier.NewService(repo, time.Second)

	_, err := service.UpdatePartial(context.Background(), u)
	require.ErrorIs(t, err, wantErr)
}

func TestService_TimeoutConfiguration_Behavior(t *testing.T) {
	t.Parallel()

	type wantRange struct {
		min time.Duration
		max time.Duration
	}

	tests := []struct {
		name      string
		timeout   time.Duration
		wantRange wantRange
	}{
		{
			name:      "zero_uses_default_3s",
			timeout:   0,
			wantRange: wantRange{min: 2 * time.Second, max: 4 * time.Second},
		},
		{
			name:      "positive_kept",
			timeout:   5 * time.Second,
			wantRange: wantRange{min: 4 * time.Second, max: 6 * time.Second},
		},
		{
			name:      "negative_uses_default_3s",
			timeout:   -10 * time.Second,
			wantRange: wantRange{min: 2 * time.Second, max: 4 * time.Second},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			repo := NewMockcourierRepository(ctrl)
			svc := courier.NewService(repo, tt.timeout)

			ctx := context.Background()
			const id int64 = 1

			var capturedCtx context.Context
			wantErr := errors.New("stop here")

			repo.EXPECT().
				Get(gomock.Any(), id).
				DoAndReturn(func(c context.Context, gotID int64) (*domain.Courier, error) {
					capturedCtx = c
					require.Equal(t, id, gotID)
					return nil, wantErr
				})

			_, err := svc.Get(ctx, id)
			require.ErrorIs(t, err, wantErr)
			require.NotNil(t, capturedCtx, "expected captured context")

			deadline, ok := capturedCtx.Deadline()
			require.True(t, ok, "expected context with deadline")

			remaining := time.Until(deadline)

			require.Greater(t, remaining, tt.wantRange.min)
			require.Less(t, remaining, tt.wantRange.max)
		})
	}
}
