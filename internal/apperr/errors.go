package apperr

import "errors"

// Invalid is returned when the input fails domain validation.
var Invalid = errors.New("invalid input")

// Conflict indicates a uniqueness or state conflict (HTTP 409).
var Conflict = errors.New("conflict")

// NotFound indicates that the requested resource does not exist.
var NotFound = errors.New("not found")
