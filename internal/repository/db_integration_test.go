//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"course-go-avito-Orurh/internal/config"
	"course-go-avito-Orurh/internal/repository"
)

func TestNewPool_Success(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load()
	require.NoError(t, err, "expected no error from config.Load")

	dsn := cfg.DB.DSN()
	require.NotEmpty(t, dsn, "cfg.DB.DSN must be set")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := repository.NewPool(ctx, dsn)
	require.NoError(t, err, "expected no error from NewPool")
	defer pool.Close()

	require.NoError(t, pool.Ping(ctx), "expected no error on ping")
}

func TestNewPool_InvalidDSN(t *testing.T) {
	t.Parallel()

	badDSN := "not-a-valid-dsn"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := repository.NewPool(ctx, badDSN)
	require.Error(t, err, "expected error for invalid DSN")
	require.Nil(t, pool, "expected nil pool on error")
}

func TestNewPool_PingError(t *testing.T) {
	t.Parallel()

	badPingDSN := "postgres://myuser:mypassword@127.0.0.1:65000/test_db?sslmode=disable"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := repository.NewPool(ctx, badPingDSN)
	require.Error(t, err, "expected ping error for unreachable DB")
	require.Nil(t, pool, "expected nil pool when ping fails")
}
