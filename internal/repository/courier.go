package repository

import (
	"context"
	"fmt"

	"course-go-avito-Orurh/internal/apperr"
	"course-go-avito-Orurh/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CourierRepo represents courier repository.
type CourierRepo struct{ db *pgxpool.Pool }

// NewCourierRepo creates a new CourierRepo.
func NewCourierRepo(db *pgxpool.Pool) *CourierRepo { return &CourierRepo{db: db} }

// Get - returns courier by its ID.
func (r *CourierRepo) Get(ctx context.Context, id int64) (*domain.Courier, error) {
	var c domain.Courier
	err := r.db.QueryRow(ctx,
		`SELECT id, name, phone, status, transport_type FROM couriers WHERE id=$1`, id,
	).Scan(&c.ID, &c.Name, &c.Phone, &c.Status, &c.TransportType)
	if err != nil {
		if IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get courier %d: %w", id, err)
	}
	return &c, nil
}

// List returns couriers ordered by id. If limit/offset are nil, returns the full list.
func (r *CourierRepo) List(ctx context.Context, limit, offset *int) ([]domain.Courier, error) {
	q := `SELECT id, name, phone, status, transport_type FROM couriers ORDER BY id`
	args := make([]any, 0, 2)
	if limit != nil {
		q += fmt.Sprintf(" LIMIT $%d", len(args)+1)
		args = append(args, *limit)
	}
	if offset != nil {
		q += fmt.Sprintf(" OFFSET $%d", len(args)+1)
		args = append(args, *offset)
	}

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	capacity := 0
	if limit != nil && *limit > 0 {
		capacity = *limit
	}
	out := make([]domain.Courier, 0, capacity)
	for rows.Next() {
		var c domain.Courier
		if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &c.Status, &c.TransportType); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Create - creates a new courier.
func (r *CourierRepo) Create(ctx context.Context, c *domain.Courier) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx,
		`INSERT INTO couriers(name,phone,status,transport_type) VALUES($1,$2,$3,$4) RETURNING id`,
		c.Name, c.Phone, c.Status, c.TransportType).Scan(&id)
	if err != nil {
		if IsDuplicate(err) {
			return 0, apperr.Conflict
		}
		return 0, fmt.Errorf("create courier: %w", err)
	}
	return id, nil
}

// UpdatePartial applies a partial update to a courier and returns true if a row was affected.
func (r *CourierRepo) UpdatePartial(ctx context.Context, u domain.PartialCourierUpdate) (bool, error) {
	ct, err := r.db.Exec(ctx, `
        UPDATE couriers
        SET
            name           = COALESCE($2, name),
            phone          = COALESCE($3, phone),
            status         = COALESCE($4, status),
            transport_type = COALESCE($5, transport_type),
            updated_at     = now()
        WHERE id = $1
    `, u.ID, u.Name, u.Phone, u.Status, u.TransportType)

	if err != nil {
		if IsDuplicate(err) {
			return false, apperr.Conflict
		}
		return false, fmt.Errorf("update courier %d: %w", u.ID, err)
	}
	return ct.RowsAffected() > 0, nil
}
