//go:build integration

package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"course-go-avito-Orurh/internal/app"
	"course-go-avito-Orurh/internal/config"
)

func TestMustBuildContainer_Integration(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c := app.MustBuildContainer(ctx)
	require.NotNil(t, c)

	err := c.Invoke(func(cfg *config.Config, pool *pgxpool.Pool) {
		require.NotNil(t, cfg)
		require.NotNil(t, pool)
	})
	require.NoError(t, err)
}
