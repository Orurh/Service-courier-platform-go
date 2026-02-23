//go:build integration

package repository_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/repository"
)

type CourierRepositorySuite struct {
	suite.Suite
	pool *pgxpool.Pool
	repo *repository.CourierRepo
}

func (s *CourierRepositorySuite) SetupSuite() {
	s.Require().NotNil(tcPool, "tcPool must be initialized in TestMain")

	s.pool = tcPool
	s.repo = repository.NewCourierRepo(tcPool)
}

func (s *CourierRepositorySuite) SetupTest() {
	_, err := s.pool.Exec(context.Background(), `TRUNCATE couriers RESTART IDENTITY CASCADE`)
	s.Require().NoError(err)
}

func (s *CourierRepositorySuite) TestCreateAndGet() {
	ctx := context.Background()

	in := &domain.Courier{
		Name:          "Artem",
		Phone:         "+70000000000",
		Status:        domain.StatusAvailable,
		TransportType: domain.TransportTypeFoot,
	}

	id, err := s.repo.Create(ctx, in)
	s.Require().NoError(err)

	got, err := s.repo.Get(ctx, id)
	s.Require().NoError(err)
	s.Require().NotNil(got)

	s.Equal(id, got.ID)
	s.Equal(in.Name, got.Name)
	s.Equal(in.Phone, got.Phone)
	s.Equal(in.Status, got.Status)
	s.Equal(in.TransportType, got.TransportType)
}

func (s *CourierRepositorySuite) TestCreate_IsDublicate() {
	ctx := context.Background()

	phone := "+70000000000"
	in1 := &domain.Courier{
		Name:          "Artem",
		Phone:         phone,
		Status:        domain.StatusAvailable,
		TransportType: domain.TransportTypeFoot,
	}
	in2 := &domain.Courier{
		Name:          "Artem",
		Phone:         phone,
		Status:        domain.StatusAvailable,
		TransportType: domain.TransportTypeFoot,
	}
	_, err := s.repo.Create(ctx, in1)
	s.Require().NoError(err)

	_, err2 := s.repo.Create(ctx, in2)
	s.ErrorIs(err2, apperr.ErrConflict, "conflict conflict for dublicate phone")
}

func (s *CourierRepositorySuite) TestGetNotFound() {
	ctx := context.Background()

	got, err := s.repo.Get(ctx, 9999)
	s.Require().NoError(err)
	s.Require().Nil(got)
}

func (s *CourierRepositorySuite) TestListWithLimitOffset() {
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := s.repo.Create(ctx, &domain.Courier{
			Name:          fmt.Sprintf("C%d", i+1),
			Phone:         fmt.Sprintf("+7000000000%d", i+1),
			Status:        domain.StatusAvailable,
			TransportType: domain.TransportTypeFoot,
		})
		s.Require().NoError(err)
	}

	limit := 2
	offset := 1

	list, err := s.repo.List(ctx, &limit, &offset)
	s.Require().NoError(err)

	s.Len(list, 2)
	s.True(list[0].ID < list[1].ID)
}

func (s *CourierRepositorySuite) TestUpdatePartial() {
	ctx := context.Background()

	id, err := s.repo.Create(ctx, &domain.Courier{
		Name:          "Not Artem",
		Phone:         "+70000000000",
		Status:        domain.StatusAvailable,
		TransportType: domain.TransportTypeFoot,
	})
	s.Require().NoError(err)

	newName := "Artem"
	update := domain.PartialCourierUpdate{
		ID:   id,
		Name: &newName,
	}

	ok, err := s.repo.UpdatePartial(ctx, update)
	s.Require().NoError(err)
	s.True(ok)

	got, err := s.repo.Get(ctx, id)
	s.Require().NoError(err)

	s.Equal(newName, got.Name)
	s.Equal("+70000000000", got.Phone)
}

func (s *CourierRepositorySuite) TestUpdatePartial_IsDublicate() {
	ctx := context.Background()

	phone1 := "+70000000000"
	_, err := s.repo.Create(ctx, &domain.Courier{
		Name:          "Artem",
		Phone:         phone1,
		Status:        domain.StatusAvailable,
		TransportType: domain.TransportTypeFoot,
	})
	s.Require().NoError(err)

	phone2 := "+70000000001"
	id2, err := s.repo.Create(ctx, &domain.Courier{
		Name:          "Artem2",
		Phone:         phone2,
		Status:        domain.StatusAvailable,
		TransportType: domain.TransportTypeFoot,
	})
	s.Require().NoError(err)

	updatePhone := phone1
	update := domain.PartialCourierUpdate{
		ID:    id2,
		Phone: &updatePhone,
	}

	ok, err := s.repo.UpdatePartial(ctx, update)
	s.False(ok, "row must not be marked as updated on duplicate")
	s.Error(err)
	s.ErrorIs(err, apperr.ErrConflict, "expected apperr.ErrConflict on duplicate phone")
}

func (s *CourierRepositorySuite) TestGet_ContextCanceled_ReturnsError() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := s.repo.Get(ctx, 1)
	s.Nil(got)
	s.Error(err)
}

func (s *CourierRepositorySuite) TestCreate_ContextCanceled_ReturnsError() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.repo.Create(ctx, &domain.Courier{
		Name:          "Artem5",
		Phone:         "+70000000009",
		Status:        domain.StatusAvailable,
		TransportType: domain.TransportTypeFoot,
	})
	s.Error(err)
	s.ErrorIs(err, context.Canceled)
}

func (s *CourierRepositorySuite) TestList_ContextCanceled_ReturnsError() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	list, err := s.repo.List(ctx, nil, nil)
	s.Nil(list)
	s.Error(err)
	s.ErrorIs(err, context.Canceled)
}

func (s *CourierRepositorySuite) TestUpdatePartial_ContextCanceled_ReturnsError() {
	id, err := s.repo.Create(context.Background(), &domain.Courier{
		Name:          "Artem6",
		Phone:         "+70000000010",
		Status:        domain.StatusAvailable,
		TransportType: domain.TransportTypeFoot,
	})
	s.Require().NoError(err)

	newName := "Boom"
	u := domain.PartialCourierUpdate{ID: id, Name: &newName}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ok, err := s.repo.UpdatePartial(ctx, u)
	s.False(ok)
	s.Error(err)
	s.ErrorIs(err, context.Canceled)
}

func TestCourierRepositorySuite(t *testing.T) {
	suite.Run(t, new(CourierRepositorySuite))
}
