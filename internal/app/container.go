package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/dig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"course-go-avito-Orurh/internal/config"
	ordersgw "course-go-avito-Orurh/internal/gateway/orders"
	"course-go-avito-Orurh/internal/http/handlers"
	"course-go-avito-Orurh/internal/http/pprofserver"
	"course-go-avito-Orurh/internal/http/router"
	"course-go-avito-Orurh/internal/logx"
	"course-go-avito-Orurh/internal/metrics"
	ordersproto "course-go-avito-Orurh/internal/proto"
	"course-go-avito-Orurh/internal/repository"
	"course-go-avito-Orurh/internal/service/courier"
	"course-go-avito-Orurh/internal/service/delivery"
	"course-go-avito-Orurh/internal/service/orders"
	"course-go-avito-Orurh/internal/transport/kafka"
)

type metricsOut struct {
	dig.Out

	RateLimitExceededTotal prometheus.Counter `name:"rate_limit_exceeded_total"`
	GatewayRetriesTotal    prometheus.Counter `name:"gateway_retries_total"`
}

// MustBuildWorkerContainer builds and returns a new dig container
func MustBuildWorkerContainer(ctx context.Context) *dig.Container {
	return NewContainerBuilder().MustBuildWorker(ctx)
}

// ContainerBuilder is a dig container builder.
type ContainerBuilder struct {
	dbConnect func(context.Context, logx.Logger, string, int, time.Duration) (*pgxpool.Pool, error)

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
	fn func(context.Context, logx.Logger, string, int, time.Duration) (*pgxpool.Pool, error),
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
	if err := registerDomainServices(container); err != nil {
		return nil, fmt.Errorf("service: %w", err)
	}
	if err := registerHTTP(container); err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	return container, nil
}

// MustBuildWorker builds and returns a new dig container
func (b *ContainerBuilder) MustBuildWorker(ctx context.Context) *dig.Container {
	container, err := b.buildWorker(ctx)
	if err != nil {
		b.logFatalf("failed to build worker container: %v", err)
	}
	return container
}

func (b *ContainerBuilder) buildWorker(ctx context.Context) (*dig.Container, error) {
	container := dig.New()

	if err := registerCore(container, ctx); err != nil {
		return nil, fmt.Errorf("core: %w", err)
	}
	if err := registerDb(container, b.dbConnect); err != nil {
		return nil, fmt.Errorf("DB: %w", err)
	}
	if err := registerDomainServices(container); err != nil {
		return nil, fmt.Errorf("service: %w", err)
	}
	if err := registerWorker(container); err != nil {
		return nil, fmt.Errorf("worker: %w", err)
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
		provideMetrics,
		func(cfg *config.Config) autoReleaseInterval {
			return autoReleaseInterval(cfg.Delivery.AutoReleaseInterval)
		},
	)
}

func registerDb(
	container *dig.Container,
	dbConnect func(context.Context, logx.Logger, string, int, time.Duration) (*pgxpool.Pool, error),
) error {
	providerDB := func(ctx context.Context, cfg *config.Config, logger logx.Logger) (*pgxpool.Pool, error) {
		return dbConnect(ctx, logger, cfg.DB.DSN(), 10, time.Second)
	}
	return provideAll(container, providerDB)
}

func registerDomainServices(container *dig.Container) error {
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
			logger logx.Logger,
		) *delivery.Service {
			return delivery.NewDeliveryService(repo, factory, timeout, logger)
		},
	)
}

func registerWorker(container *dig.Container) error {
	return provideAll(container,
		provideOrdersGateway,
		func(deliverySvc orders.DeliveryPort, repo *repository.DeliveryRepo) *orders.Processor {
			return orders.NewProcessorWithDeps(deliverySvc, repo)
		},

		makeOrdersKafka,

		func(cfg *config.Config, h kafka.HandleFunc, logger logx.Logger) (*kafka.Consumer, error) {
			c, err := kafka.NewConsumer(logger, cfg.Kafka.Brokers, cfg.Kafka.GroupID, cfg.Kafka.Topic, h)
			if err != nil {
				return nil, err
			}
			if c == nil {
				return nil, fmt.Errorf("kafka config is missing: worker requires KAFKA_BROKERS/KAFKA_GROUP_ID/KAFKA_TOPIC")
			}
			return c, nil
		},
	)
}

type serversOut struct {
	dig.Out

	Server *http.Server
	Pprof  *http.Server `name:"pprof_server"`
}

func registerHTTP(container *dig.Container) error {
	serverProvider := func(cfg *config.Config, mux http.Handler) serversOut {
		main := &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Port),
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
		}

		var pprofS *http.Server

		if cfg.Pprof.Enabled {
			pprofS = &http.Server{

				Addr:              cfg.Pprof.Addr,
				Handler:           pprofserver.Handler(pprofserver.Config{User: cfg.Pprof.User, Pass: cfg.Pprof.Pass}),
				ReadHeaderTimeout: 5 * time.Second,
				IdleTimeout:       60 * time.Second,
			}
		}
		return serversOut{Server: main, Pprof: pprofS}
	}

	return provideAll(container,
		handlers.New,
		handlers.NewCourierUsecase,
		handlers.NewCourierHandler,
		handlers.NewDeliveryUsecase,
		handlers.NewDeliveryHandler,
		newRateLimitClock,
		newRateLimiter,
		newRateLimitMiddleware,
		router.New,
		serverProvider,
	)
}

type ordersConnCloser func() error

type ordersGatewayIn struct {
	dig.In
	Ctx     context.Context
	Cfg     *config.Config
	Logger  logx.Logger
	Retries prometheus.Counter `name:"gateway_retries_total"`
}

func provideOrdersGateway(in ordersGatewayIn) (ordersGateway, ordersConnCloser, error) {
	addr := strings.TrimSpace(in.Cfg.OrderService)

	if addr == "" {
		return nil, nil, nil
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("provideOrdersGateway grpc: %w", err)
	}
	client := ordersproto.NewOrdersServiceClient(conn)
	base := ordersgw.NewGRPCGateway(client)

	gw := ordersgw.NewRetryingGateway(
		base,
		in.Logger,
		in.Retries,
		ordersgw.RetryConfig{
			MaxAttempts: in.Cfg.OrdersGateway.MaxAttempts,
			BaseDelay:   in.Cfg.OrdersGateway.BaseDelay,
			MaxDelay:    in.Cfg.OrdersGateway.MaxDelay,
		},
	)
	return gw, func() error { return conn.Close() }, nil
}

func provideMetrics() (metricsOut, error) {
	rl := metrics.NewRateLimitExceededTotal()
	if err := prometheus.Register(rl); err != nil {
		var are prometheus.AlreadyRegisteredError
		if !errors.As(err, &are) {
			return metricsOut{}, fmt.Errorf("register rate_limit_exceeded_total: %w", err)
		}
		existing, ok := are.ExistingCollector.(prometheus.Counter)
		if !ok {
			return metricsOut{}, fmt.Errorf("register rate_limit_exceeded_total: %w", err)
		}
		rl = existing
	}

	gr := metrics.NewGatewayRetriesTotal()
	if err := prometheus.Register(gr); err != nil {
		var are prometheus.AlreadyRegisteredError
		if !errors.As(err, &are) {
			return metricsOut{}, fmt.Errorf("register gateway_retries_total: %w", err)
		}
		existing, ok := are.ExistingCollector.(prometheus.Counter)
		if !ok {
			return metricsOut{}, fmt.Errorf("register gateway_retries_total: %w", err)
		}
		gr = existing
	}

	return metricsOut{
		RateLimitExceededTotal: rl,
		GatewayRetriesTotal:    gr,
	}, nil
}
