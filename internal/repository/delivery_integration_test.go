//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"

	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/repository"
	"course-go-avito-Orurh/internal/service/delivery"
)

type DeliveryRepositorySuite struct {
	suite.Suite
	pool         *pgxpool.Pool
	deliveryRepo *repository.DeliveryRepo
	courierRepo  *repository.CourierRepo
}

func withTxDelivery(ctx context.Context, repo *repository.DeliveryRepo, fn func(tx delivery.TxRepository) error) error {
	return repo.WithTx(ctx, func(tx delivery.TxRepository) error {

		return fn(tx)
	})
}

func (s *DeliveryRepositorySuite) SetupSuite() {
	s.Require().NotNil(tcPool, "tcPool must be initialized in TestMain")

	s.pool = tcPool
	s.deliveryRepo = repository.NewDeliveryRepo(tcPool)
	s.courierRepo = repository.NewCourierRepo(tcPool)
}

func (s *DeliveryRepositorySuite) SetupTest() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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

func (s *DeliveryRepositorySuite) TestInsertDeliveryAndGetByOrderID() {
	ctx := context.Background()

	courierID := s.createCourier("Artem", "+70000000000", domain.StatusAvailable)

	assignedAt := time.Now().Add(-time.Minute)
	deadline := time.Now().Add(30 * time.Minute)

	var insertedID int64

	err := withTxDelivery(ctx, s.deliveryRepo, func(tx delivery.TxRepository) error {
		d := &domain.Delivery{
			CourierID:  courierID,
			OrderID:    "order-1",
			AssignedAt: assignedAt,
			Deadline:   deadline,
		}

		if err := tx.InsertDelivery(ctx, d); err != nil {
			return err
		}
		s.Require().Positive(d.ID)
		insertedID = d.ID

		got, err := tx.GetByOrderID(ctx, "order-1")
		if err != nil {
			return err
		}
		s.Require().NotNil(got)
		s.Equal(d.ID, got.ID)
		s.Equal(courierID, got.CourierID)

		return nil
	})
	s.Require().NoError(err)

	var got2 *domain.Delivery
	err = withTxDelivery(ctx, s.deliveryRepo, func(tx delivery.TxRepository) error {
		var err error
		got2, err = tx.GetByOrderID(ctx, "order-1")
		return err
	})
	s.Require().NoError(err)
	s.Require().NotNil(got2)
	s.Equal(insertedID, got2.ID)
}

func (s *DeliveryRepositorySuite) TestDeleteByOrderID() {
	ctx := context.Background()

	courierID := s.createCourier("Artem", "+70000000000", domain.StatusAvailable)

	err := withTxDelivery(ctx, s.deliveryRepo, func(tx delivery.TxRepository) error {
		d := &domain.Delivery{
			CourierID:  courierID,
			OrderID:    "order-2",
			AssignedAt: time.Now(),
			Deadline:   time.Now().Add(10 * time.Minute),
		}
		return tx.InsertDelivery(ctx, d)
	})
	s.Require().NoError(err)

	err = withTxDelivery(ctx, s.deliveryRepo, func(tx delivery.TxRepository) error {
		return tx.DeleteByOrderID(ctx, "order-2")
	})
	s.Require().NoError(err)

	err = withTxDelivery(ctx, s.deliveryRepo, func(tx delivery.TxRepository) error {
		got, err := tx.GetByOrderID(ctx, "order-2")
		if err != nil {
			return err
		}
		s.Nil(got)
		return nil
	})
	s.Require().NoError(err)
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

	var courier *domain.Courier

	err := withTxDelivery(ctx, s.deliveryRepo, func(tx delivery.TxRepository) error {
		var err error
		courier, err = tx.FindAvailableCourierForUpdate(ctx)
		return err
	})
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

func (s *DeliveryRepositorySuite) TestUpdateCourierStatus_Success() {
	ctx := context.Background()

	id := s.createCourier("Artem", "+70000000020", domain.StatusAvailable)

	err := withTxDelivery(ctx, s.deliveryRepo, func(tx delivery.TxRepository) error {
		return tx.UpdateCourierStatus(ctx, id, domain.StatusBusy)
	})
	s.Require().NoError(err)

	got, err := s.courierRepo.Get(ctx, id)
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Equal(domain.StatusBusy, got.Status)
}

func (s *DeliveryRepositorySuite) TestUpdateCourierStatus_NotFound() {
	ctx := context.Background()

	const badID int64 = 999999

	err := withTxDelivery(ctx, s.deliveryRepo, func(tx delivery.TxRepository) error {
		return tx.UpdateCourierStatus(ctx, badID, domain.StatusBusy)
	})
	s.Require().Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *DeliveryRepositorySuite) TestFindAvailableCourierForUpdate_NoAvailableCouriers() {
	ctx := context.Background()

	_ = s.createCourier("Busy1", "+70000000030", domain.StatusBusy)
	_ = s.createCourier("Busy2", "+70000000031", domain.StatusBusy)

	var courier *domain.Courier

	err := withTxDelivery(ctx, s.deliveryRepo, func(tx delivery.TxRepository) error {
		var err error
		courier, err = tx.FindAvailableCourierForUpdate(ctx)
		return err
	})
	s.Require().NoError(err)
	s.Nil(courier)
}

func (s *DeliveryRepositorySuite) TestDeleteByOrderID_NotFound() {
	ctx := context.Background()

	const orderID = "no-order"

	err := withTxDelivery(ctx, s.deliveryRepo, func(tx delivery.TxRepository) error {
		return tx.DeleteByOrderID(ctx, orderID)
	})
	s.Require().Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *DeliveryRepositorySuite) TestWithTx_BeginTx_ContextCanceled() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := withTxDelivery(ctx, s.deliveryRepo, func(tx delivery.TxRepository) error {
		return nil
	})
	s.Error(err)
	s.ErrorIs(err, context.Canceled)
}

func (s *DeliveryRepositorySuite) TestWithTx_Commit_ContextCanceled() {
	ctx, cancel := context.WithCancel(context.Background())
	err := withTxDelivery(ctx, s.deliveryRepo, func(tx delivery.TxRepository) error {
		cancel()
		return nil
	})
	s.Error(err)
	s.ErrorIs(err, context.Canceled)
}

func (s *DeliveryRepositorySuite) TestInsertDelivery_FKViolation_CoversErrorBranch() {
	ctx := context.Background()

	err := withTxDelivery(ctx, s.deliveryRepo, func(tx delivery.TxRepository) error {
		d := &domain.Delivery{
			CourierID:  999999,
			OrderID:    "order-100",
			AssignedAt: time.Now(),
			Deadline:   time.Now().Add(10 * time.Minute),
		}
		return tx.InsertDelivery(ctx, d)
	})
	s.Require().Error(err)
	s.Contains(err.Error(), "insert delivery")
}

func (s *DeliveryRepositorySuite) TestGetByOrderID_ContextCanceled_CoversErrorBranch() {
	ctx := context.Background()

	courierID := s.createCourier("ArtemX", "+70000000111", domain.StatusAvailable)
	now := time.Now()

	err := withTxDelivery(ctx, s.deliveryRepo, func(tx delivery.TxRepository) error {
		return tx.InsertDelivery(ctx, &domain.Delivery{
			CourierID:  courierID,
			OrderID:    "order-cancel-get",
			AssignedAt: now,
			Deadline:   now.Add(10 * time.Minute),
		})
	})
	s.Require().NoError(err)
	err = withTxDelivery(ctx, s.deliveryRepo, func(tx delivery.TxRepository) error {
		cctx, cancel := context.WithCancel(ctx)
		cancel()

		_, err := tx.GetByOrderID(cctx, "order-cancel-get")
		return err
	})
	s.Require().Error(err)
	s.ErrorIs(err, context.Canceled)
	s.Contains(err.Error(), "get delivery by order")
}

func (s *DeliveryRepositorySuite) TestDeleteByOrderID_ContextCanceled_CoversErrorBranch() {
	ctx := context.Background()

	courierID := s.createCourier("ArtemY", "+70000000112", domain.StatusAvailable)
	now := time.Now()

	err := withTxDelivery(ctx, s.deliveryRepo, func(tx delivery.TxRepository) error {
		return tx.InsertDelivery(ctx, &domain.Delivery{
			CourierID:  courierID,
			OrderID:    "order-cancel-del",
			AssignedAt: now,
			Deadline:   now.Add(10 * time.Minute),
		})
	})
	s.Require().NoError(err)

	err = withTxDelivery(ctx, s.deliveryRepo, func(tx delivery.TxRepository) error {
		cctx, cancel := context.WithCancel(ctx)
		cancel()

		return tx.DeleteByOrderID(cctx, "order-cancel-del")
	})
	s.Require().Error(err)
	s.ErrorIs(err, context.Canceled)
	s.Contains(err.Error(), "delete delivery by order")
}

func TestDeliveryRepositorySuite(t *testing.T) {
	suite.Run(t, new(DeliveryRepositorySuite))
}
