package logx

import "time"

// Logger is a minimal structured logging interface based on key-value fields.
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	With(fields ...Field) Logger
	Sync() error
}

// Field represents a single structured log field (key-value pair).
type Field struct {
	Key   string
	Value any
}

// Any creates a field with an arbitrary value type.
func Any(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// String creates a string field.
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an int field.
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field.
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Time creates a time.Time field.
func Time(key string, value time.Time) Field {
	return Field{Key: key, Value: value}
}

// Duration creates a time.Duration field.
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value}
}
