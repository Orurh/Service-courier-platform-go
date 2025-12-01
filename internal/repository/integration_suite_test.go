package repository_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var tcPool *pgxpool.Pool

var tcDSN string

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_pass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		log.Fatalf("failed to start postgres testcontainer: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		if termErr := pgContainer.Terminate(ctx); termErr != nil {
			log.Printf("failed to terminate container after conn string error: %v", termErr)
		}
		log.Fatalf("failed to get connection string from container: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		if termErr := pgContainer.Terminate(ctx); termErr != nil {
			log.Printf("failed to terminate container after pool create error: %v", termErr)
		}
		log.Fatalf("failed to create pgx pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		if termErr := pgContainer.Terminate(ctx); termErr != nil {
			log.Printf("failed to terminate container after ping error: %v", termErr)
		}
		log.Fatalf("failed to ping postgres in testcontainer: %v", err)
	}

	tcPool = pool
	tcDSN = connStr

	if err := createTables(ctx, tcPool); err != nil {
		pool.Close()
		if termErr := pgContainer.Terminate(ctx); termErr != nil {
			log.Printf("failed to terminate container after createTables error: %v", termErr)
		}
		log.Fatalf("failed to create test tables: %v", err)
	}

	code := m.Run()

	pool.Close()
	if err := pgContainer.Terminate(ctx); err != nil {
		log.Printf("failed to terminate postgres container: %v", err)
	}

	os.Exit(code)
}

func createTables(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS couriers (
			id             BIGSERIAL PRIMARY KEY,
			name           TEXT NOT NULL,
			phone          TEXT NOT NULL UNIQUE,
			status         TEXT NOT NULL,
			transport_type TEXT NOT NULL,
			created_at     TIMESTAMP WITHOUT TIME ZONE DEFAULT now() NOT NULL,
			updated_at     TIMESTAMP WITHOUT TIME ZONE DEFAULT now() NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("create couriers table: %w", err)
	}

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS delivery (
			id          BIGSERIAL PRIMARY KEY,
			courier_id  BIGINT NOT NULL REFERENCES couriers(id) ON DELETE CASCADE,
			order_id    TEXT NOT NULL UNIQUE,
			assigned_at TIMESTAMP WITHOUT TIME ZONE NOT NULL,
			deadline    TIMESTAMP WITHOUT TIME ZONE NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("create delivery table: %w", err)
	}

	return nil
}
