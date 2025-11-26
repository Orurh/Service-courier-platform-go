//go:build integration

package app

import (
	"context"
	"course-go-avito-Orurh/internal/config"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMustBuildContainer_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c := MustBuildContainer(ctx)
	if c == nil {
		t.Fatal("MustBuildContainer returned nil container")
	}

	err := c.Invoke(func(cfg *config.Config, pool *pgxpool.Pool) {
		if cfg == nil {
			t.Fatalf("*config.Config is nil")
		}
		if pool == nil {
			t.Fatalf("*pgxpool.Pool is nil")
		}
	})
	if err != nil {
		t.Fatalf("container.Invoke failed: %v", err)
	}
}
