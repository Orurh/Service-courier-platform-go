package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/dig"

	"course-go-avito-Orurh/internal/logx"
	"course-go-avito-Orurh/internal/service/delivery"
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
	invErr := container.Invoke(func(logger logx.Logger) {
		switch {
		case errors.Is(err, context.Canceled):
			logger.Info("shutdown requested, exiting")
		case errors.Is(err, context.DeadlineExceeded):
			logger.Info("startup aborted: startup timeout exceeded")
		default:
			logger.Error("run error", logx.Any("err", err))
		}
	})
	if invErr != nil {
		return
	}
}

func run(container *dig.Container) error {
	return container.Invoke(appRun)
}

type appDeps struct {
	dig.In

	Server              *http.Server
	AppCtx              context.Context
	Pool                *pgxpool.Pool
	Logger              logx.Logger
	DeliveryService     *delivery.Service
	AutoReleaseInterval autoReleaseInterval

	OrdersCloser ordersConnCloser `optional:"true"`
}

func appRun(d appDeps) error {
	defer closeResources(d.Pool, d.Server, d.Logger, d.OrdersCloser)
	startAutoReleaseLoop(d.AppCtx, d.Logger, d.DeliveryService, time.Duration(d.AutoReleaseInterval))
	serverErrCh := startServer(d.Server, d.Logger)
	err := waitForShutdown(d.AppCtx, d.Logger, serverErrCh)
	d.Logger.Info("shut down service-courier")
	gracefulShutdown(d.Server, d.Logger, shutdownTimeout)
	return err
}

func startAutoReleaseLoop(ctx context.Context, logger logx.Logger, deliveryService *delivery.Service, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := deliveryService.ReleaseExpired(ctx); err != nil {
					logger.Error("auto-release failed", logx.Any("err", err))
				}
			}
		}
	}()
}

func startServer(server *http.Server, logger logx.Logger) <-chan error {
	ch := make(chan error, 1)
	go func() {
		logger.Info("service-courier listening", logx.String("addr", server.Addr))
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			ch <- err
			return
		}
		ch <- nil
	}()
	return ch
}

func waitForShutdown(ctx context.Context, logger logx.Logger, serverErrCh <-chan error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-serverErrCh:
		if err != nil {
			logger.Error("server stopped", logx.Any("err", err))
		}
		return err
	}
}

func gracefulShutdown(srv *http.Server, logger logx.Logger, timeout time.Duration) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown error", logx.Any("err", err))
	}
}

func closeResources(pool *pgxpool.Pool, server *http.Server, logger logx.Logger, ordersCloser ordersConnCloser) {
	if server != nil {
		if err := server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server close error", logx.Any("err", err))
		}
	}
	if ordersCloser != nil {
		if err := ordersCloser(); err != nil {
			logger.Error("orders close error", logx.Any("err", err))
		}
	}
	if pool != nil {
		pool.Close()
	}
}
