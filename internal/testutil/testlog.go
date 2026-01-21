package testlog

import (
	"sync"

	"course-go-avito-Orurh/internal/logx"
)

// Entry is a log entry
type Entry struct {
	Level  string
	Msg    string
	Fields []logx.Field
}

// Recorder records log entries
type Recorder struct {
	mu      sync.Mutex
	entries []Entry
}

// New returns a new logger
func New() *Recorder { return &Recorder{} }

// Logger returns a bound logger
func (r *Recorder) Logger() logx.Logger {
	return bound{r: r}
}

// Entries returns a copy of the log entries
func (r *Recorder) Entries() []Entry {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Entry, len(r.entries))
	copy(out, r.entries)
	return out
}

func (r *Recorder) add(level, msg string, fields []logx.Field) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := append([]logx.Field(nil), fields...)
	r.entries = append(r.entries, Entry{Level: level, Msg: msg, Fields: cp})
}

type bound struct {
	r    *Recorder
	base []logx.Field
}

// Debug logs a debug message
func (b bound) Debug(msg string, f ...logx.Field) {
	b.r.add("debug", msg, append(b.base, f...))
}

// Info logs an info message
func (b bound) Info(msg string, f ...logx.Field) {
	b.r.add("info", msg, append(b.base, f...))
}

// Warn logs a warn message
func (b bound) Warn(msg string, f ...logx.Field) {
	b.r.add("warn", msg, append(b.base, f...))
}

// Error logs an error message
func (b bound) Error(msg string, f ...logx.Field) {
	b.r.add("error", msg, append(b.base, f...))
}

func (b bound) With(f ...logx.Field) logx.Logger {
	nb := bound{r: b.r, base: append([]logx.Field(nil), b.base...)}
	nb.base = append(nb.base, f...)
	return nb
}

func (b bound) Sync() error { return nil }

var _ logx.Logger = bound{}
