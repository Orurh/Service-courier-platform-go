package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/dig"

	"course-go-avito-Orurh/internal/service/delivery"
)

type autoReleaseInterval time.Duration

const (
	shutdownTimeout = 15 * time.Second
)

// Runner runs the HTTP server
type Runner struct {
	runFn      func(*dig.Container) error
	logPrintln func(...any)
	logFatalf  func(string, ...any)
}

// NewRunner returns a new Runner
func NewRunner() *Runner {
	return &Runner{
		runFn:      run,
		logPrintln: log.Println,
		logFatalf:  log.Fatalf,
	}
}

// MustRun starts the HTTP server using the provided DI container
func (r *Runner) MustRun(container *dig.Container) {
	if err := r.runFn(container); err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			r.logPrintln("shutdown requested, exiting")
			return
		case errors.Is(err, context.DeadlineExceeded):
			r.logPrintln("startup aborted: startup timeout exceeded")
			return
		default:
			r.logFatalf("run error: %v", err)
		}
	}
}

func run(container *dig.Container) error {
	return container.Invoke(appRun)
}

func appRun(server *http.Server, appCtx context.Context, pool *pgxpool.Pool, logger *log.Logger,
	deliveryService *delivery.Service, autoReleaseInterval autoReleaseInterval) error {
	startAutoReleaseLoop(appCtx, logger, deliveryService, time.Duration(autoReleaseInterval))
	startServer(server, logger)
	waitForShutdown(appCtx, logger)
	gracefulShutdown(server, logger, shutdownTimeout)
	closeResources(pool, server, logger)
	return nil
}

func startAutoReleaseLoop(ctx context.Context, logger *log.Logger, deliveryService *delivery.Service, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := deliveryService.ReleaseExpired(ctx); err != nil {
					logger.Printf("auto-release failed: %v", err)
				}
			}
		}
	}()
}

func startServer(server *http.Server, logger *log.Logger) {
	go func() {
		logger.Printf("service-courier listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("listen error: %v", err)
		}
	}()
}

func waitForShutdown(ctx context.Context, logger *log.Logger) {
	<-ctx.Done()
	logger.Println("shutting down service-courier...")
}

func gracefulShutdown(srv *http.Server, logger *log.Logger, timeout time.Duration) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Printf("graceful shutdown error: %v", err)
	}
}

func closeResources(pool *pgxpool.Pool, server *http.Server, logger *log.Logger) {
	if err := server.Close(); err != nil {
		logger.Printf("server close error: %v", err)
	}
	if pool != nil {
		pool.Close()
	}
}
