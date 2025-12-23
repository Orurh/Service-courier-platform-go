package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/dig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"course-go-avito-Orurh/internal/config"
	ordersgw "course-go-avito-Orurh/internal/gateway/orders"
	"course-go-avito-Orurh/internal/http/handlers"
	"course-go-avito-Orurh/internal/http/router"
	ordersproto "course-go-avito-Orurh/internal/proto"
	"course-go-avito-Orurh/internal/repository"
	"course-go-avito-Orurh/internal/service/courier"
	"course-go-avito-Orurh/internal/service/delivery"
	"course-go-avito-Orurh/internal/service/orders"
	"course-go-avito-Orurh/internal/transport/kafka"
)

// ContainerBuilder is a dig container builder.
type ContainerBuilder struct {
	dbConnect func(context.Context, *slog.Logger, string, int, time.Duration) (*pgxpool.Pool, error)

	logFatalf func(string, ...any)
}

// NewContainerBuilder returns a new dig container builder
func NewContainerBuilder() *ContainerBuilder {
	return &ContainerBuilder{
		dbConnect: connectDbWithRetry,
		logFatalf: func(format string, args ...any) { panic(fmt.Sprintf(format, args...)) },
	}
}

// WithDBConnect sets the database connection function
func (b *ContainerBuilder) WithDBConnect(
	fn func(context.Context, *slog.Logger, string, int, time.Duration) (*pgxpool.Pool, error),
) *ContainerBuilder {
	if fn != nil {
		b.dbConnect = fn
	}
	return b
}

// WithLogFatalf sets the log.Fatalf function
func (b *ContainerBuilder) WithLogFatalf(fn func(string, ...any)) *ContainerBuilder {
	if fn != nil {
		b.logFatalf = fn
	}
	return b
}

// MustBuild builds and returns a new dig container
func (b *ContainerBuilder) MustBuild(ctx context.Context) *dig.Container {
	container, err := b.build(ctx)
	if err != nil {
		b.logFatalf("failed to build container: %v", err)
	}
	return container
}

// build builds and returns a new dig container
func (b *ContainerBuilder) build(ctx context.Context) (*dig.Container, error) {
	container := dig.New()

	if err := registerCore(container, ctx); err != nil {
		return nil, fmt.Errorf("core: %w", err)
	}
	if err := registerDb(container, b.dbConnect); err != nil {
		return nil, fmt.Errorf("DB: %w", err)
	}
	if err := registerService(container); err != nil {
		return nil, fmt.Errorf("service: %w", err)
	}
	if err := registerHTTP(container); err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	return container, nil
}

// MustBuildContainer builds and returns a new dig container
func MustBuildContainer(ctx context.Context) *dig.Container {
	return NewContainerBuilder().MustBuild(ctx)
}

func provideAll(container *dig.Container, providers ...any) error {
	for _, provider := range providers {
		if err := container.Provide(provider); err != nil {
			return fmt.Errorf("provide %T: %w", provider, err)
		}
	}
	return nil
}

func registerCore(container *dig.Container, ctx context.Context) error {
	return provideAll(container,
		func() context.Context { return ctx },
		NewLogger,
		config.Load,
		func(cfg *config.Config) autoReleaseInterval {
			return autoReleaseInterval(cfg.Delivery.AutoReleaseInterval)
		},
	)
}

func registerDb(
	container *dig.Container,
	dbConnect func(context.Context, *slog.Logger, string, int, time.Duration) (*pgxpool.Pool, error),
) error {
	providerDB := func(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*pgxpool.Pool, error) {
		return dbConnect(ctx, logger, cfg.DB.DSN(), 10, time.Second)
	}
	return provideAll(container, providerDB)
}

func registerService(container *dig.Container) error {
	return provideAll(container,
		repository.NewCourierRepo,
		repository.NewDeliveryRepo,

		func() time.Duration { return 3 * time.Second },
		func(repo *repository.CourierRepo, timeout time.Duration) *courier.Service {
			return courier.NewService(repo, timeout)
		},
		delivery.NewTimeFactory,
		func(
			repo *repository.DeliveryRepo,
			timeout time.Duration,
			factory delivery.TimeFactory,
			logger *slog.Logger,
		) *delivery.Service {
			return delivery.NewDeliveryService(repo, factory, timeout, logger)
		},
		provideOrdersGateway,
		orders.NewProcessor,
		makeOrdersKafka,

		func(cfg *config.Config, h kafka.HandleFunc, logger *slog.Logger) (*kafka.Consumer, error) {
			return kafka.NewConsumer(logger, cfg.Kafka.Brokers, cfg.Kafka.GroupID, cfg.Kafka.Topic, h)
		},
	)
}

func registerHTTP(container *dig.Container) error {
	serverProvider := func(cfg *config.Config, mux http.Handler) *http.Server {
		return &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Port),
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
		}
	}
	return provideAll(container,
		handlers.New,
		handlers.NewCourierUsecase,
		handlers.NewCourierHandler,
		handlers.NewDeliveryUsecase,
		handlers.NewDeliveryHandler,
		router.New,
		serverProvider,
	)
}

type ordersConnCloser func() error

func provideOrdersGateway(ctx context.Context, cfg *config.Config) (*ordersgw.GRPCGateway, ordersConnCloser, error) {
	addr := strings.TrimSpace(cfg.OrderService)
	if addr == "" {
		return nil, nil, nil
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("provideOrdersGateway grpc: %w", err)
	}
	client := ordersproto.NewOrdersServiceClient(conn)
	gw := ordersgw.NewGRPCGateway(client)
	return gw, func() error { return conn.Close() }, nil
}
