//go:build integration

package repository

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewPool_Succses(t *testing.T) {
	t.Parallel()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN is not set")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("expected no error on ping, got %v", err)
	}
	pool.Close()
}

func TestNewPool_InvalidDSN(t *testing.T) {
	t.Parallel()

	badDSN := "postgres://myuser:mypassword@127.0.0.1:1/test_db?sslmode=disable"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := NewPool(ctx, badDSN)
	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Fatal("expected error for invalid DSN, got nil")
	}
	if pool != nil {
		t.Fatal("expected nil pool on error")
	}
}
