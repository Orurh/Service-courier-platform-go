//go:build integration

package app

import (
	"context"
	"course-go-avito-Orurh/internal/logx"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func testLogger(_ io.Writer) logx.Logger { return logx.Nop() }

func withStubNewPool(t *testing.T, stub func(context.Context, string) (*pgxpool.Pool, error)) {
	t.Helper()
	orig := newPool
	newPool = stub
	t.Cleanup(func() { newPool = orig })
}

func TestConnectDbWithRetry_SuccessFirstAttempt(t *testing.T) {
	ctx := context.Background()
	dsn := "postgres://stub"

	wantPool := &pgxpool.Pool{}
	calls := 0

	withStubNewPool(t, func(_ context.Context, _ string) (*pgxpool.Pool, error) {
		calls++
		return wantPool, nil
	})

	pool, err := connectDbWithRetry(ctx, testLogger(io.Discard), dsn, 3, 10*time.Millisecond)
	require.NoError(t, err)
	require.Equal(t, wantPool, pool)
	require.Equal(t, 1, calls)
}

func TestConnectDbWithRetry_ExhaustsRetries(t *testing.T) {
	ctx := context.Background()
	dsn := "postgres://stub"

	sentinelErr := errors.New("db boom")
	calls := 0

	withStubNewPool(t, func(_ context.Context, _ string) (*pgxpool.Pool, error) {
		calls++
		return nil, sentinelErr
	})

	pool, err := connectDbWithRetry(ctx, testLogger(io.Discard), dsn, 3, 0)
	require.Error(t, err)
	require.Nil(t, pool)
	require.Equal(t, 3, calls)
	require.ErrorIs(t, err, sentinelErr)
}

func TestConnectDbWithRetry_ContextCanceledBetweenRetries(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dsn := "postgres://stub"
	sentinelErr := errors.New("db boom")

	withStubNewPool(t, func(_ context.Context, _ string) (*pgxpool.Pool, error) {
		return nil, sentinelErr
	})

	pool, err := connectDbWithRetry(ctx, testLogger(io.Discard), dsn, 3, 50*time.Millisecond)
	require.Error(t, err)
	require.Nil(t, pool)
	require.ErrorIs(t, err, context.Canceled)
}
