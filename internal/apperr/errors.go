package apperr

import "errors"

// ErrInvalid is returned when the input fails domain validation.
var ErrInvalid = errors.New("invalid input")

// ErrConflict indicates a uniqueness or state conflict (HTTP 409).
var ErrConflict = errors.New("conflict")

// ErrNotFound indicates that the requested resource does not exist.
var ErrNotFound = errors.New("not found")
