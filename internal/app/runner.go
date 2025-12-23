package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/dig"

	"course-go-avito-Orurh/internal/service/delivery"
	"course-go-avito-Orurh/internal/transport/kafka"
)

type autoReleaseInterval time.Duration

const (
	shutdownTimeout = 15 * time.Second
)

// Runner runs the HTTP server
type Runner struct{ runFn func(*dig.Container) error }

// NewRunner returns a new Runner
func NewRunner() *Runner {
	return &Runner{runFn: run}
}

// MustRun starts the HTTP server using the provided DI container
func (r *Runner) MustRun(container *dig.Container) {
	err := r.runFn(container)
	if err == nil {
		return
	}
	if container == nil {
		return
	}
	invErr := container.Invoke(func(logger *slog.Logger) {
		if logger == nil {
			panic("logger is nil")
		}
		switch {
		case errors.Is(err, context.Canceled):
			logger.Info("shutdown requested, exiting")
		case errors.Is(err, context.DeadlineExceeded):
			logger.Info("startup aborted: startup timeout exceeded")
		default:
			logger.Error("run error", slog.Any("err", err))
		}
	})
	if invErr != nil {
		return
	}
}

func run(container *dig.Container) error {
	return container.Invoke(appRun)
}

func appRun(server *http.Server, appCtx context.Context, pool *pgxpool.Pool, logger *slog.Logger, deliveryService *delivery.Service,
	autoReleaseInterval autoReleaseInterval, kafkaConsumer *kafka.Consumer, ordersCloser ordersConnCloser,
) error {
	defer closeResources(pool, server, logger, kafkaConsumer, ordersCloser)
	startAutoReleaseLoop(appCtx, logger, deliveryService, time.Duration(autoReleaseInterval))
	startKafkaLoop(appCtx, logger, kafkaConsumer)
	serverErrCh := startServer(server, logger)
	err := waitForShutdown(appCtx, logger, serverErrCh)
	logger.Info("shut down service-courier")
	gracefulShutdown(server, logger, shutdownTimeout)
	return err
}

func startAutoReleaseLoop(ctx context.Context, logger *slog.Logger, deliveryService *delivery.Service, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := deliveryService.ReleaseExpired(ctx); err != nil {
					logger.Error("auto-release failed", slog.Any("err", err))
				}
			}
		}
	}()
}

func startServer(server *http.Server, logger *slog.Logger) <-chan error {
	ch := make(chan error, 1)
	go func() {
		logger.Info("service-courier listening", slog.String("addr", server.Addr))
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			ch <- err
			return
		}
		ch <- nil
	}()
	return ch
}

func waitForShutdown(ctx context.Context, logger *slog.Logger, serverErrCh <-chan error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-serverErrCh:
		if err != nil {
			logger.Error("server stopped", slog.Any("err", err))
		}
		return err
	}
}

func gracefulShutdown(srv *http.Server, logger *slog.Logger, timeout time.Duration) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown error", slog.Any("err", err))
	}
}

func closeResources(pool *pgxpool.Pool, server *http.Server, logger *slog.Logger, kafkaConsumer *kafka.Consumer, ordersCloser ordersConnCloser) {
	if server != nil {
		if err := server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server close error", slog.Any("err", err))
		}
	}
	if kafkaConsumer != nil {
		if err := kafkaConsumer.Close(); err != nil {
			logger.Error("kafka close error", slog.Any("err", err))
		}
	}
	if ordersCloser != nil {
		if err := ordersCloser(); err != nil {
			logger.Error("orders close error", slog.Any("err", err))
		}
	}
	if pool != nil {
		pool.Close()
	}
}

func startKafkaLoop(ctx context.Context, logger *slog.Logger, c *kafka.Consumer) {
	if c == nil {
		logger.Info("kafka consumer disabled")
		return
	}
	go func() {
		logger.Info("kafka consumer started")
		if err := c.Run(ctx); err != nil {
			logger.Error("kafka consumer stopped", slog.Any("err", err))
		}
	}()
}
