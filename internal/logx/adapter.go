package logx

import "log/slog"

// SlogAdapter adapts the standard library slog.Logger to the logx.Logger interface.
type SlogAdapter struct {
	l *slog.Logger
}

// NewSlogAdapter returns a Logger implementation backed by the provided *slog.Logger.
func NewSlogAdapter(l *slog.Logger) Logger {
	return &SlogAdapter{l: l}
}

// Debug logs a debug-level message with optional structured fields.
func (s *SlogAdapter) Debug(msg string, fields ...Field) { s.l.Debug(msg, toSlogArgs(fields)...) }

// Info logs an info-level message with optional structured fields.
func (s *SlogAdapter) Info(msg string, fields ...Field) { s.l.Info(msg, toSlogArgs(fields)...) }

// Warn logs a warning-level message with optional structured fields.
func (s *SlogAdapter) Warn(msg string, fields ...Field) { s.l.Warn(msg, toSlogArgs(fields)...) }

// Error logs an error-level message with optional structured fields.
func (s *SlogAdapter) Error(msg string, fields ...Field) { s.l.Error(msg, toSlogArgs(fields)...) }

// With returns a new logger with the provided fields attached to every subsequent log entry.
func (s *SlogAdapter) With(fields ...Field) Logger {
	return &SlogAdapter{l: s.l.With(toSlogArgs(fields)...)}
}

// Sync flushes buffered logs if supported; slog.Logger does not require flushing.
func (s *SlogAdapter) Sync() error { return nil }

// toSlogArgs converts logx fields into slog arguments.
func toSlogArgs(fields []Field) []any {
	args := make([]any, 0, len(fields))
	for _, f := range fields {
		args = append(args, slog.Any(f.Key, f.Value))
	}
	return args
}
