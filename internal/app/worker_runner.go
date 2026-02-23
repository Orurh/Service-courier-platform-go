package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/dig"

	"course-go-avito-Orurh/internal/logx"
	"course-go-avito-Orurh/internal/transport/kafka"
)

// WorkerRunner runs the HTTP server
type WorkerRunner struct {
	runFn func(*dig.Container) error
}

// NewWorkerRunner returns a new WorkerRunner
func NewWorkerRunner() *WorkerRunner {
	return &WorkerRunner{runFn: runWorker}
}

// MustRun starts the HTTP server using the provided DI container
func (r *WorkerRunner) MustRun(container *dig.Container) {
	err := r.runFn(container)
	if err == nil || errors.Is(err, context.Canceled) {
		return
	}
	panic(err)
}

func runWorker(container *dig.Container) error {
	return container.Invoke(workerRun)
}

func workerRun(
	ctx context.Context,
	pool *pgxpool.Pool,
	logger logx.Logger,
	consumer *kafka.Consumer,
	ordersCloser ordersConnCloser,
) error {
	if consumer == nil {
		return fmt.Errorf("kafka consumer is nil: worker container misconfigured")
	}
	defer closeWorker(pool, logger, consumer, ordersCloser)

	logger.Info("service-courier-worker started")
	return consumer.Run(ctx)
}

func closeWorker(pool *pgxpool.Pool, logger logx.Logger, kafkaConsumer *kafka.Consumer, ordersCloser ordersConnCloser) {
	if kafkaConsumer != nil {
		if err := kafkaConsumer.Close(); err != nil {
			logger.Error("kafka close error", logx.Any("err", err))
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
