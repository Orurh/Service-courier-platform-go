package logx

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// наверно не надо тестить это
func TestFields_Constructors(t *testing.T) {
	now := time.Now()

	require.Equal(t, Field{Key: "k", Value: "v"}, String("k", "v"))
	require.Equal(t, Field{Key: "k", Value: 1}, Int("k", 1))
	require.Equal(t, Field{Key: "k", Value: int64(2)}, Int64("k", int64(2)))
	require.Equal(t, Field{Key: "k", Value: now}, Time("k", now))
	require.Equal(t, Field{Key: "k", Value: time.Second}, Duration("k", time.Second))
	require.Equal(t, Field{Key: "k", Value: struct{ A int }{A: 1}}, Any("k", struct{ A int }{A: 1}))
}

func TestNopLogger_NoPanic(t *testing.T) {
	l := Nop()
	l.Debug("d", String("k", "v"))
	l.Info("i", Int("n", 1))
	l.Warn("w")
	l.Error("e")

	l2 := l.With(String("x", "y"))
	require.NotNil(t, l2)

	require.NoError(t, l.Sync())
	require.NoError(t, l2.Sync())
}

func TestSlogAdapter_WithAndToSlogArgs(t *testing.T) {
	base := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	l := NewSlogAdapter(base)

	args := toSlogArgs([]Field{
		String("a", "b"),
		Int("n", 1),
	})
	require.Len(t, args, 2)

	l2 := l.With(String("x", "y"))
	require.NotNil(t, l2)

	l2.Info("msg", String("k", "v"))
	require.NoError(t, l2.Sync())
}

func TestSlogAdapter_Warn(t *testing.T) {
	base := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	l := NewSlogAdapter(base)

	l.Warn("msg", String("k", "v"))
	require.NoError(t, l.Sync())
}

func TestSlogAdapter_Error(t *testing.T) {
	base := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	l := NewSlogAdapter(base)

	l.Error("msg", String("k", "v"))
	require.NoError(t, l.Sync())
}

func TestSlogAdapter_Debug(t *testing.T) {
	base := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	l := NewSlogAdapter(base)

	l.Debug("msg", String("k", "v"))
	require.NoError(t, l.Sync())
}
