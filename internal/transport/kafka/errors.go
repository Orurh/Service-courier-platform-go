package kafka

// PermanentError is a permanent error.
type PermanentError struct {
	Err error
}

func (e PermanentError) Error() string {
	if e.Err == nil {
		return "permanent error"
	}
	return e.Err.Error()
}

func (e PermanentError) Unwrap() error { return e.Err }

// Permanent returns a permanent error.
func Permanent(err error) error {
	return PermanentError{Err: err}
}
