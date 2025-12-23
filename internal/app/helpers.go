package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"course-go-avito-Orurh/internal/repository"
)

var newPool = repository.NewPool

func connectDbWithRetry(ctx context.Context, logger *slog.Logger, dsn string, retries int, delay time.Duration) (*pgxpool.Pool, error) {
	if logger == nil {
		panic("db: logger is nil")
	}
	var lastErr error
	const attemptTimeout = 3 * time.Second
	for i := 1; i <= retries; i++ {
		retriesCtx, cancel := context.WithTimeout(ctx, attemptTimeout)
		pool, err := newPool(retriesCtx, dsn)
		cancel()
		if err == nil {
			logger.Info("db connected", slog.Int("attempt", i))
			return pool, nil
		}
		lastErr = err
		logger.Warn("db connect failed", slog.Int("attempt", i), slog.Int("retries", retries), slog.Any("err", err))
		if i < retries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	return nil, fmt.Errorf("db connect failed after %d attempts: %w", retries, lastErr)
}
