package testlog

import (
	"sync"

	"course-go-avito-Orurh/internal/logx"
)

type Entry struct {
	Level  string
	Msg    string
	Fields []logx.Field
}

type Recorder struct {
	mu      sync.Mutex
	entries []Entry
}

func New() *Recorder { return &Recorder{} }

func (r *Recorder) Logger() logx.Logger {
	return bound{r: r}
}

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

func (b bound) Debug(msg string, f ...logx.Field) {
	b.r.add("debug", msg, append(b.base, f...))
}

func (b bound) Info(msg string, f ...logx.Field) {
	b.r.add("info", msg, append(b.base, f...))
}

func (b bound) Warn(msg string, f ...logx.Field) {
	b.r.add("warn", msg, append(b.base, f...))
}

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
