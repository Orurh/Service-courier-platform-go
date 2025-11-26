//go:build integration

package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/repository"
	"course-go-avito-Orurh/internal/service/delivery"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
)

type DeliveryRepositorySuite struct {
	suite.Suite
	pool         *pgxpool.Pool
	deliveryRepo *repository.DeliveryRepo
	courierRepo  *repository.CourierRepo
}

func (s *DeliveryRepositorySuite) SetupSuite() {
	dsn := getenvOrSkip(s.T(), "TEST_DB_DSN")

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	s.Require().NoError(err)
	s.Require().NoError(pool.Ping(ctx))

	s.pool = pool
	s.deliveryRepo = repository.NewDeliveryRepo(pool)
	s.courierRepo = repository.NewCourierRepo(pool)
}

func (s *DeliveryRepositorySuite) TearDownSuite() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *DeliveryRepositorySuite) SetupTest() {
	ctx := context.Background()
	_, err := s.pool.Exec(ctx, `TRUNCATE delivery RESTART IDENTITY CASCADE`)
	s.Require().NoError(err)
	_, err = s.pool.Exec(ctx, `TRUNCATE couriers RESTART IDENTITY CASCADE`)
	s.Require().NoError(err)
}

func (s *DeliveryRepositorySuite) createCourier(name, phone string, status domain.CourierStatus) int64 {
	ctx := context.Background()
	id, err := s.courierRepo.Create(ctx, &domain.Courier{
		Name:          name,
		Phone:         phone,
		Status:        status,
		TransportType: domain.TransportTypeFoot,
	})
	s.Require().NoError(err)
	return id
}

type fakeTx struct{}

func (f *fakeTx) Commit(ctx context.Context) error { return nil }
func (f *fakeTx) Rollback(ctx context.Context)     {}

func (s *DeliveryRepositorySuite) TestInsertDeliveryAndGetByOrderID() {
	ctx := context.Background()

	courierID := s.createCourier("Artem", "+70000000000", domain.StatusAvailable)

	tx, err := s.deliveryRepo.BeginTx(ctx)
	s.Require().NoError(err)

	assignedAt := time.Now().Add(-time.Minute)
	deadline := time.Now().Add(30 * time.Minute)

	d := &domain.Delivery{
		CourierID:  courierID,
		OrderID:    "order-1",
		AssignedAt: assignedAt,
		Deadline:   deadline,
	}

	err = s.deliveryRepo.InsertDelivery(ctx, tx, d)
	s.Require().NoError(err)
	s.Require().Positive(d.ID)

	got, err := s.deliveryRepo.GetByOrderID(ctx, tx, "order-1")
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Equal(d.ID, got.ID)
	s.Equal(courierID, got.CourierID)

	s.Require().NoError(tx.Commit(ctx))

	tx2, err := s.deliveryRepo.BeginTx(ctx)
	s.Require().NoError(err)
	defer tx2.Rollback(ctx)

	got2, err := s.deliveryRepo.GetByOrderID(ctx, tx2, "order-1")
	s.Require().NoError(err)
	s.Require().NotNil(got2)
	s.Equal(d.ID, got2.ID)
}

func (s *DeliveryRepositorySuite) TestDeleteByOrderID() {
	ctx := context.Background()

	courierID := s.createCourier("Artem", "+70000000000", domain.StatusAvailable)

	tx, err := s.deliveryRepo.BeginTx(ctx)
	s.Require().NoError(err)

	d := &domain.Delivery{
		CourierID:  courierID,
		OrderID:    "order-2",
		AssignedAt: time.Now(),
		Deadline:   time.Now().Add(10 * time.Minute),
	}
	s.Require().NoError(s.deliveryRepo.InsertDelivery(ctx, tx, d))
	s.Require().NoError(tx.Commit(ctx))

	tx2, err := s.deliveryRepo.BeginTx(ctx)
	s.Require().NoError(err)
	err = s.deliveryRepo.DeleteByOrderID(ctx, tx2, "order-2")
	s.Require().NoError(err)
	s.Require().NoError(tx2.Commit(ctx))

	tx3, err := s.deliveryRepo.BeginTx(ctx)
	s.Require().NoError(err)
	defer tx3.Rollback(ctx)

	got, err := s.deliveryRepo.GetByOrderID(ctx, tx3, "order-2")
	s.Require().NoError(err)
	s.Nil(got)
}

func (s *DeliveryRepositorySuite) TestFindAvailableCourierForUpdate_PicksLeastLoaded() {
	ctx := context.Background()

	id1 := s.createCourier("C1", "+70000000001", domain.StatusAvailable)
	id2 := s.createCourier("C2", "+70000000002", domain.StatusAvailable)
	id3 := s.createCourier("C3", "+70000000003", domain.StatusAvailable)

	now := time.Now()
	insertDeliveryRaw := func(courierID int64) {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO delivery (courier_id, order_id, assigned_at, deadline)
			VALUES ($1, gen_random_uuid()::text, $2, $3)
		`, courierID, now, now.Add(10*time.Minute))
		s.Require().NoError(err)
	}
	insertDeliveryRaw(id1)
	insertDeliveryRaw(id1)
	insertDeliveryRaw(id2)

	tx, err := s.deliveryRepo.BeginTx(ctx)
	s.Require().NoError(err)
	defer tx.Rollback(ctx)

	courier, err := s.deliveryRepo.FindAvailableCourierForUpdate(ctx, tx)
	s.Require().NoError(err)
	s.Require().NotNil(courier)

	s.Equal(id3, courier.ID)
}

func (s *DeliveryRepositorySuite) TestReleaseCouriers() {
	ctx := context.Background()

	id1 := s.createCourier("Busy1", "+70000000010", domain.StatusBusy)
	id2 := s.createCourier("Busy2", "+70000000011", domain.StatusBusy)

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	_, err := s.pool.Exec(ctx, `
		INSERT INTO delivery (courier_id, order_id, assigned_at, deadline)
		VALUES ($1, 'o1', $2, $3)
	`, id1, past, past)
	s.Require().NoError(err)

	_, err = s.pool.Exec(ctx, `
		INSERT INTO delivery (courier_id, order_id, assigned_at, deadline)
		VALUES ($1, 'o2', $2, $3)
	`, id2, now, future)
	s.Require().NoError(err)

	affected, err := s.deliveryRepo.ReleaseCouriers(ctx, now)
	s.Require().NoError(err)
	s.Equal(int64(1), affected)

	c1, err := s.courierRepo.Get(ctx, id1)
	s.Require().NoError(err)
	s.Equal(domain.StatusAvailable, c1.Status)

	c2, err := s.courierRepo.Get(ctx, id2)
	s.Require().NoError(err)
	s.Equal(domain.StatusBusy, c2.Status)
}

func TestDeliveryRepositorySuite(t *testing.T) {
	suite.Run(t, new(DeliveryRepositorySuite))
}

func getenvOrSkip(t *testing.T, key string) string {
	val := os.Getenv(key)
	if val == "" {
		t.Skipf("%s nt set, skipping integration tests", key)
	}
	return val
}

func (s *DeliveryRepositorySuite) TestUpdateCourierStatus_Success() {
	ctx := context.Background()

	id := s.createCourier("Artem", "+70000000020", domain.StatusAvailable)

	tx, err := s.deliveryRepo.BeginTx(ctx)
	s.Require().NoError(err)
	defer tx.Rollback(ctx)

	err = s.deliveryRepo.UpdateCourierStatus(ctx, tx, id, string(domain.StatusBusy))
	s.Require().NoError(err)

	s.Require().NoError(tx.Commit(ctx))

	got, err := s.courierRepo.Get(ctx, id)
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Equal(domain.StatusBusy, got.Status)
}

func (s *DeliveryRepositorySuite) TestUpdateCourierStatus_NotFound() {
	ctx := context.Background()

	tx, err := s.deliveryRepo.BeginTx(ctx)
	s.Require().NoError(err)
	defer tx.Rollback(ctx)

	const badID int64 = 999999

	err = s.deliveryRepo.UpdateCourierStatus(ctx, tx, badID, string(domain.StatusBusy))
	s.Require().Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *DeliveryRepositorySuite) TestFindAvailableCourierForUpdate_NoAvailableCouriers() {
	ctx := context.Background()

	_ = s.createCourier("Busy1", "+70000000030", domain.StatusBusy)
	_ = s.createCourier("Busy2", "+70000000031", domain.StatusBusy)

	tx, err := s.deliveryRepo.BeginTx(ctx)
	s.Require().NoError(err)
	defer tx.Rollback(ctx)

	courier, err := s.deliveryRepo.FindAvailableCourierForUpdate(ctx, tx)

	s.Require().NoError(err)
	s.Nil(courier)
}

func (s *DeliveryRepositorySuite) TestFindAvailableCourierForUpdate_UnexpectedTxImpl() {
	ctx := context.Background()

	var tx delivery.Tx = &fakeTx{}

	courier, err := s.deliveryRepo.FindAvailableCourierForUpdate(ctx, tx)

	s.Nil(courier)
	s.Require().Error(err)
	s.Contains(err.Error(), "unexpected tx implementation")
}

func (s *DeliveryRepositorySuite) TestDeleteByOrderID_NotFound() {
	ctx := context.Background()

	tx, err := s.deliveryRepo.BeginTx(ctx)
	s.Require().NoError(err)
	defer tx.Rollback(ctx)

	const orderID = "no-order"

	err = s.deliveryRepo.DeleteByOrderID(ctx, tx, orderID)

	s.Require().Error(err)
	s.Contains(err.Error(), "not found")
}
