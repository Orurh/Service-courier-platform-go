package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/ports/deliverytx"
)

// DeliveryRepo represents delivery repository.
type DeliveryRepo struct {
	db *pgxpool.Pool
}

// NewDeliveryRepo creates a new DeliveryRepo.
func NewDeliveryRepo(db *pgxpool.Pool) *DeliveryRepo {
	return &DeliveryRepo{db: db}
}

// WithTx opens a transaction and executes fn within it.
func (r *DeliveryRepo) WithTx(ctx context.Context, fn func(tx deliverytx.Repository) error) (err error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	// отменяем в случае паники
	defer func() {
		if p := recover(); p != nil {
			err = tx.Rollback(ctx)
			if err != nil {
				panic(err)
			}
			panic(p)
		}
	}()

	wrapped := &TxRepo{tx: tx}

	if err := fn(wrapped); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("rollback tx: %w (original error: %s)", rbErr, err.Error())
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

// TxRepo represents transaction repository.
type TxRepo struct {
	tx pgx.Tx
}

// FindAvailableCourierForUpdate - find available courier for update.
func (r *TxRepo) FindAvailableCourierForUpdate(ctx context.Context) (*domain.Courier, error) {
	row := r.tx.QueryRow(ctx, `
        SELECT c.id, c.name, c.phone, c.status, c.transport_type
        FROM couriers c
        WHERE c.status = 'available'
        ORDER BY
            (SELECT COUNT(*) FROM delivery d WHERE d.courier_id = c.id) ASC,
            c.id ASC
        FOR UPDATE
        LIMIT 1
    `)

	var c domain.Courier
	if err := row.Scan(&c.ID, &c.Name, &c.Phone, &c.Status, &c.TransportType); err != nil {
		if IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("find available courier: %w", err)
	}
	return &c, nil
}

// UpdateCourierStatus - update courier status.
func (r *TxRepo) UpdateCourierStatus(ctx context.Context, id int64, status domain.CourierStatus) error {
	ct, err := r.tx.Exec(ctx, `
        UPDATE couriers
        SET status = $2, updated_at = now()
        WHERE id = $1
    `, id, string(status))
	if err != nil {
		return fmt.Errorf("update courier status %d: %w", id, err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("courier %d not found", id)
	}
	return nil
}

// InsertDelivery - insert a new delivery.
func (r *TxRepo) InsertDelivery(ctx context.Context, d *domain.Delivery) error {
	err := r.tx.QueryRow(ctx, `
        INSERT INTO delivery (courier_id, order_id, assigned_at, deadline)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `, d.CourierID, d.OrderID, d.AssignedAt, d.Deadline).Scan(&d.ID)
	if err != nil {
		return fmt.Errorf("insert delivery: %w", err)
	}
	return nil
}

// GetByOrderID - get delivery by order ID.
func (r *TxRepo) GetByOrderID(ctx context.Context, orderID string) (*domain.Delivery, error) {
	row := r.tx.QueryRow(ctx, `
        SELECT id, courier_id, order_id, assigned_at, deadline
        FROM delivery
        WHERE order_id = $1
    `, orderID)

	var d domain.Delivery
	if err := row.Scan(&d.ID, &d.CourierID, &d.OrderID, &d.AssignedAt, &d.Deadline); err != nil {
		if IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get delivery by order %q: %w", orderID, err)
	}
	return &d, nil
}

// DeleteByOrderID - delete delivery by order ID.
func (r *TxRepo) DeleteByOrderID(ctx context.Context, orderID string) error {
	ct, err := r.tx.Exec(ctx, `DELETE FROM delivery WHERE order_id = $1`, orderID)
	if err != nil {
		return fmt.Errorf("delete delivery by order %q: %w", orderID, err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("delivery for order %q not found", orderID)
	}
	return nil
}

// ReleaseCouriers - release expired couriers.
func (r *DeliveryRepo) ReleaseCouriers(ctx context.Context, now time.Time) (int64, error) {
	cmd, err := r.db.Exec(ctx, `
        UPDATE couriers c
        SET status = $1,
            updated_at = now()
        WHERE c.status = $2
          AND c.id IN (
              SELECT d.courier_id
              FROM delivery d
              WHERE d.deadline < $3
          )
    `, string(domain.StatusAvailable), string(domain.StatusBusy), now)
	if err != nil {
		return 0, fmt.Errorf("release expired couriers: %w", err)
	}
	return cmd.RowsAffected(), nil
}
