package repository

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// IsDuplicate - signals that the error is a duplicate key violation.
func IsDuplicate(err error) bool {
	var pgerr *pgconn.PgError
	return errors.As(err, &pgerr) && pgerr.Code == "23505"
}

// IsNotFound - signals that the error is a not found error.
func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
