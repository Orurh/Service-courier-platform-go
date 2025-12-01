package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"course-go-avito-Orurh/internal/repository"
)

var newPool = repository.NewPool

func connectDbWithRetry(ctx context.Context, dsn string, retries int, delay time.Duration) (*pgxpool.Pool, error) {
	var lastErr error
	const attemptTimeout = 3 * time.Second
	for i := 1; i <= retries; i++ {
		retriesCtx, cancel := context.WithTimeout(ctx, attemptTimeout)
		pool, err := newPool(retriesCtx, dsn)
		cancel()
		if err == nil {
			log.Printf("db connected on attempt %d", i)
			return pool, nil
		}
		lastErr = err
		log.Printf("db connect failed (attempt %d/%d): %v", i, retries, err)
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
