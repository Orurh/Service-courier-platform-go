package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"course-go-avito-Orurh/internal/config"
	"course-go-avito-Orurh/internal/http/handlers"
	"course-go-avito-Orurh/internal/http/router"
	"course-go-avito-Orurh/internal/repository"
	"course-go-avito-Orurh/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
)

func connectDbWithRetry(ctx context.Context, dsn string, retries int, delay time.Duration) (*pgxpool.Pool, error) {
	var lastErr error
	const attemptTimeout = 3 * time.Second
	for i := 1; i <= retries; i++ {
		retriesCtx, cancel := context.WithTimeout(ctx, attemptTimeout)
		pool, err := repository.NewPool(retriesCtx, dsn)
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

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load error: %v", err)
	}
	ctxSignals, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := connectDbWithRetry(ctxSignals, cfg.DB.DSN(), 10, time.Second)
	if err != nil {
		log.Fatalf("database connection error: %v", err)
	}
	defer pool.Close()

	base := handlers.New(log.Default())
	repo := repository.NewCourierRepo(pool)
	uc := service.NewCourierService(repo, 3*time.Second)

	courier := handlers.NewCourierHandler(uc)
	mux := router.New(base, courier)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	defer func() {
		if err := srv.Close(); err != nil {
			log.Printf("server close error: %v", err)
		}
	}()

	go func() {
		log.Printf("service-courier listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %v", err)
		}
	}()

	<-ctxSignals.Done()
	log.Println("Shutting down service-courier")

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}
}
