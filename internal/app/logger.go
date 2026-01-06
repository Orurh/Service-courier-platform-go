package app

import (
	"log/slog"
	"os"

	"course-go-avito-Orurh/internal/logx"
)

func NewLogger() logx.Logger {
	base := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	return logx.NewSlogAdapter(base)
}
