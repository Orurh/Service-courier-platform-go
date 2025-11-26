package repository

import (
	"context"
	"fmt"
	"time"

	"course-go-avito-Orurh/internal/domain"
	"course-go-avito-Orurh/internal/service/delivery" // <-- важно

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DeliveryRepo represents delivery repository.
type DeliveryRepo struct {
	db *pgxpool.Pool
}

// NewDeliveryRepo creates a new DeliveryRepo.
func NewDeliveryRepo(db *pgxpool.Pool) *DeliveryRepo {
	return &DeliveryRepo{db: db}
}

type txWrapper struct {
	tx pgx.Tx
}

// Commit - commit the transaction.
func (w *txWrapper) Commit(ctx context.Context) error {
	return w.tx.Commit(ctx)
}

// Rollback - rollback the transaction.
func (w *txWrapper) Rollback(ctx context.Context) {
	_ = w.tx.Rollback(ctx) //nolint:errcheck
}

// BeginTx - begin the transaction.
func (r *DeliveryRepo) BeginTx(ctx context.Context) (delivery.Tx, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	return &txWrapper{tx: tx}, nil
}

func unwrapTx(tx delivery.Tx) (pgx.Tx, error) {
	w, ok := tx.(*txWrapper)
	if !ok {
		return nil, fmt.Errorf("unexpected tx implementation: %T", tx)
	}
	return w.tx, nil
}

// FindAvailableCourierForUpdate - find available courier for update.
func (r *DeliveryRepo) FindAvailableCourierForUpdate(ctx context.Context, tx delivery.Tx) (*domain.Courier, error) {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return nil, err
	}

	row := pgxTx.QueryRow(ctx, `
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
func (r *DeliveryRepo) UpdateCourierStatus(ctx context.Context, tx delivery.Tx, id int64, status string) error {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return err
	}
	ct, err := pgxTx.Exec(ctx, `
        UPDATE couriers
        SET status = $2, updated_at = now()
        WHERE id = $1
    `, id, status)
	if err != nil {
		return fmt.Errorf("update courier status %d: %w", id, err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("courier %d not found", id)
	}
	return nil
}

// InsertDelivery - insert a new delivery.
func (r *DeliveryRepo) InsertDelivery(ctx context.Context, tx delivery.Tx, d *domain.Delivery) error {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return err
	}
	err = pgxTx.QueryRow(ctx, `
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
func (r *DeliveryRepo) GetByOrderID(ctx context.Context, tx delivery.Tx, orderID string) (*domain.Delivery, error) {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return nil, err
	}
	row := pgxTx.QueryRow(ctx, `
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
func (r *DeliveryRepo) DeleteByOrderID(ctx context.Context, tx delivery.Tx, orderID string) error {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return err
	}

	ct, err := pgxTx.Exec(ctx, `DELETE FROM delivery WHERE order_id = $1`, orderID)
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
