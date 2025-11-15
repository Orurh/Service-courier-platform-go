package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/dig"
)

// MustRun starts the HTTP server using the provided DI container
func MustRun(container *dig.Container) {
	if err := run(container); err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			log.Println("shutdown requested, exiting")
			return
		case errors.Is(err, context.DeadlineExceeded):
			log.Println("startup aborted: startup timeout exceeded")
			return
		default:
			log.Fatalf("run error: %v", err)
		}
	}
}

func run(container *dig.Container) error {
	return container.Invoke(func(server *http.Server, ctx context.Context, pool *pgxpool.Pool, logger *log.Logger) error {
		startServer(server, logger)
		waitForShutdown(ctx, logger)
		gracefulShutdown(server, logger, 15*time.Second)
		closeResources(pool, server, logger)
		return nil
	})
}

func startServer(server *http.Server, logger *log.Logger) {
	go func() {
		logger.Printf("service-courier listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %v", err)
		}
	}()
}

func waitForShutdown(ctx context.Context, logger *log.Logger) {
	<-ctx.Done()
	logger.Println("shutting down service-courier...")
}

func gracefulShutdown(srv *http.Server, logger *log.Logger, timeout time.Duration) {
	shCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := srv.Shutdown(shCtx); err != nil {
		logger.Printf("graceful shutdown error: %v", err)
	}
}

func closeResources(pool *pgxpool.Pool, server *http.Server, logger *log.Logger) {
	if err := server.Close(); err != nil {
		logger.Printf("server close error: %v", err)
	}
	pool.Close()
}
