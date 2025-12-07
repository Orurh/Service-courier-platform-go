package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/dig"

	"course-go-avito-Orurh/internal/config"
	"course-go-avito-Orurh/internal/http/handlers"
	"course-go-avito-Orurh/internal/http/router"
	"course-go-avito-Orurh/internal/repository"
	"course-go-avito-Orurh/internal/service/courier"
	"course-go-avito-Orurh/internal/service/delivery"
)

// ContainerBuilder is a dig container builder.
type ContainerBuilder struct {
	dbConnect func(context.Context, string, int, time.Duration) (*pgxpool.Pool, error)
	logFatalf func(string, ...interface{})
}

// NewContainerBuilder returns a new dig container builder
func NewContainerBuilder() *ContainerBuilder {
	return &ContainerBuilder{
		dbConnect: connectDbWithRetry,
		logFatalf: log.Fatalf,
	}
}

// WithDBConnect sets the database connection function
func (b *ContainerBuilder) WithDBConnect(
	fn func(context.Context, string, int, time.Duration) (*pgxpool.Pool, error),
) *ContainerBuilder {
	if fn != nil {
		b.dbConnect = fn
	}
	return b
}

// WithLogFatalf sets the log.Fatalf function
func (b *ContainerBuilder) WithLogFatalf(fn func(string, ...interface{})) *ContainerBuilder {
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
		func() *log.Logger { return log.Default() },
		config.Load,
		func(cfg *config.Config) autoReleaseInterval {
			return autoReleaseInterval(cfg.Delivery.AutoReleaseInterval)
		},
	)
}

func registerDb(
	container *dig.Container,
	dbConnect func(context.Context, string, int, time.Duration) (*pgxpool.Pool, error),
) error {
	providerDB := func(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
		return dbConnect(ctx, cfg.DB.DSN(), 10, time.Second)
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
		func() delivery.TimeFactory {
			return delivery.NewTimeFactory()
		},
		func(
			repo *repository.DeliveryRepo,
			timeout time.Duration,
			factory delivery.TimeFactory,
		) *delivery.Service {
			return delivery.NewDeliveryService(repo, factory, timeout)
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
