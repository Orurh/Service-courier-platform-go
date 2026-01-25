package courier_test

import (
	"context"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/service/courier"
)

func TestService_Create_Validation(t *testing.T) {
	t.Parallel()

	for _, tt := range createValidationCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertCreateValidationCase(t, tt.courier, tt.wantErr)
		})
	}
}

var createValidationCases = []struct {
	name    string
	courier *domain.Courier
	wantErr bool
}{
	{
		name:    "nil courier",
		courier: nil,
		wantErr: true,
	},
	{
		name: "empty name",
		courier: &domain.Courier{
			Name:          "    ",
			Phone:         "+70000000000",
			Status:        domain.StatusAvailable,
			TransportType: domain.TransportTypeFoot,
		},
		wantErr: true,
	},
	{
		name: "invalid phone",
		courier: &domain.Courier{
			Name:          "Artem",
			Phone:         "123",
			Status:        domain.StatusAvailable,
			TransportType: domain.TransportTypeFoot,
		},
		wantErr: true,
	},
	{
		name: "invalid status",
		courier: &domain.Courier{
			Name:          "Artem",
			Phone:         "+70000000000",
			Status:        domain.CourierStatus("boom"),
			TransportType: domain.TransportTypeFoot,
		},
		wantErr: true,
	},
	{
		name: "invalid transport type",
		courier: &domain.Courier{
			Name:          "Artem",
			Phone:         "+70000000000",
			Status:        domain.StatusAvailable,
			TransportType: domain.CourierTransportType("teleport"),
		},
		wantErr: true,
	},
	{
		name: "valid courier",
		courier: &domain.Courier{
			Name:          "Artem",
			Phone:         "+70000000000",
			Status:        domain.StatusAvailable,
			TransportType: domain.TransportTypeFoot,
		},
		wantErr: false,
	},
}

func assertCreateValidationCase(t *testing.T, c *domain.Courier, wantErr bool) {
	t.Helper()

	ctrl := gomock.NewController(t)
	repo := NewMockcourierRepository(ctrl)

	if !wantErr {
		repo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(int64(123), nil)
	}

	svc := courier.NewService(repo, time.Second)
	id, err := svc.Create(context.Background(), c)

	if wantErr {
		require.ErrorIs(t, err, apperr.ErrInvalid)
		require.Zero(t, id)
		return
	}
	require.NoError(t, err)
	require.Equal(t, int64(123), id)
}

func ptr[T any](v T) *T { return &v }

func TestService_Update_Validation(t *testing.T) {
	t.Parallel()

	for _, tt := range updateValidationCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertUpdateValidationCase(t, tt.update, tt.wantErr)
		})
	}
}

var updateValidationCases = func() []struct {
	name    string
	update  *domain.PartialCourierUpdate
	wantErr bool
} {
	validStatus := domain.StatusAvailable
	busyStatus := domain.StatusBusy
	footTransport := domain.TransportTypeFoot
	carTransport := domain.TransportTypeCar

	invalidStatus := domain.CourierStatus("bad")
	invalidTransport := domain.CourierTransportType("teleport")

	return []struct {
		name    string
		update  *domain.PartialCourierUpdate
		wantErr bool
	}{
		{
			name: "id <= 0",
			update: &domain.PartialCourierUpdate{
				ID: 0,
			},
			wantErr: true,
		},
		{
			name: "all fields nil",
			update: &domain.PartialCourierUpdate{
				ID: 1,
			},
			wantErr: true,
		},
		{
			name: "empty name",
			update: &domain.PartialCourierUpdate{
				ID:   1,
				Name: ptr("   "),
			},
			wantErr: true,
		},
		{
			name: "invalid phone",
			update: &domain.PartialCourierUpdate{
				ID:    1,
				Phone: ptr("123"),
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			update: &domain.PartialCourierUpdate{
				ID:     1,
				Status: &invalidStatus,
			},
			wantErr: true,
		},
		{
			name: "invalid transport type",
			update: &domain.PartialCourierUpdate{
				ID:            1,
				TransportType: &invalidTransport,
			},
			wantErr: true,
		},
		{
			name: "valid update with all fields",
			update: &domain.PartialCourierUpdate{
				ID:            1,
				Name:          ptr("Artem"),
				Phone:         ptr("+70000000000"),
				Status:        &validStatus,
				TransportType: &footTransport,
			},
			wantErr: false,
		},
		{
			name: "valid update with different status",
			update: &domain.PartialCourierUpdate{
				ID:            1,
				Status:        &busyStatus,
				TransportType: &footTransport,
			},
			wantErr: false,
		},
		{
			name: "valid update with different transport type",
			update: &domain.PartialCourierUpdate{
				ID:            1,
				Status:        &validStatus,
				TransportType: &carTransport,
			},
			wantErr: false,
		},
	}
}()

func assertUpdateValidationCase(t *testing.T, upd *domain.PartialCourierUpdate, wantErr bool) {
	t.Helper()

	ctrl := gomock.NewController(t)
	repo := NewMockcourierRepository(ctrl)

	if !wantErr {
		repo.EXPECT().
			UpdatePartial(gomock.Any(), gomock.Any()).
			Return(true, nil)
	}

	svc := courier.NewService(repo, time.Second)

	update := domain.PartialCourierUpdate{}
	if upd != nil {
		update = *upd
	}

	ok, err := svc.UpdatePartial(context.Background(), update)

	if wantErr {
		require.ErrorIs(t, err, apperr.ErrInvalid)
		require.False(t, ok)
		return
	}

	require.NoError(t, err)
	require.True(t, ok)
}
