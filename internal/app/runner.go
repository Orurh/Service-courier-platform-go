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
	"course-go-avito-Orurh/internal/transport/kafka"
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
			r.logPrintln("run error:", err)
			return
		}
	}
}

func run(container *dig.Container) error {
	return container.Invoke(appRun)
}

func appRun(server *http.Server, appCtx context.Context, pool *pgxpool.Pool, logger *log.Logger, deliveryService *delivery.Service,
	autoReleaseInterval autoReleaseInterval, kafkaConsumer *kafka.Consumer, ordersCloser ordersConnCloser,
) error {
	defer closeResources(pool, server, logger, kafkaConsumer, ordersCloser)
	startAutoReleaseLoop(appCtx, logger, deliveryService, time.Duration(autoReleaseInterval))
	startKafkaLoop(appCtx, logger, kafkaConsumer)
	serverErrCh := startServer(server, logger)
	err := waitForShutdown(appCtx, logger, serverErrCh)
	logger.Println("shut down service-courier")
	gracefulShutdown(server, logger, shutdownTimeout)
	return err
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

func startServer(server *http.Server, logger *log.Logger) <-chan error {
	ch := make(chan error, 1)
	go func() {
		logger.Printf("service-courier listening on %s", server.Addr)
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			ch <- err
			return
		}
		ch <- nil
	}()
	return ch
}

func waitForShutdown(ctx context.Context, logger *log.Logger, serverErrCh <-chan error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-serverErrCh:
		if err != nil {
			logger.Printf("server stopped, error: %v", err)
		}
		return err
	}
}

func gracefulShutdown(srv *http.Server, logger *log.Logger, timeout time.Duration) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Printf("graceful shutdown error: %v", err)
	}
}

func closeResources(pool *pgxpool.Pool, server *http.Server, logger *log.Logger, kafkaConsumer *kafka.Consumer, ordersCloser ordersConnCloser) {
	if server != nil {
		if err := server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Printf("server close error: %v", err)
		}
	}
	if kafkaConsumer != nil {
		if err := kafkaConsumer.Close(); err != nil {
			logger.Printf("kafka close error: %v", err)
		}
	}
	if ordersCloser != nil {
		if err := ordersCloser(); err != nil {
			logger.Printf("orders close error: %v", err)
		}
	}
	if pool != nil {
		pool.Close()
	}
}

func startKafkaLoop(ctx context.Context, logger *log.Logger, c *kafka.Consumer) {
	if c == nil {
		logger.Println("kafka consumer: disabled")
		return
	}
	go func() {
		logger.Println("kafka consumer: started")
		if err := c.Run(ctx); err != nil {
			logger.Printf("kafka consumer: stopped: %v", err)
		}
	}()
}

// до перехода на Kafka
// func startOrderAssignLoop(
// 	ctx context.Context,
// 	logger *log.Logger,
// 	deliveryService *delivery.Service,
// 	orderGateway order.Gateway,
// 	interval time.Duration,
// ) {
// 	go func() {
// 		ticker := time.NewTicker(interval)
// 		defer ticker.Stop()
// 		lastFrom := time.Now().UTC()
// 		for {
// 			select {
// 			case <-ctx.Done():
// 				return
// 			case <-ticker.C:
// 				now := time.Now().UTC()
// 				orders, err := orderGateway.ListFrom(ctx, lastFrom)
// 				if err != nil {
// 					logger.Printf("order-assign: list from %v failed: %v", lastFrom, err)
// 					lastFrom = now
// 					continue
// 				}

// 				for _, o := range orders {
// 					_, err := deliveryService.Assign(ctx, o.ID)
// 					if err != nil {
// 						logger.Printf("order-assign: assign order %s failed: %v", o.ID, err)
// 					}
// 				}
// 				lastFrom = now
// 			}
// 		}
// 	}()
// }
