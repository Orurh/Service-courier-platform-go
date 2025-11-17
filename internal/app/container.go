package app

import (
	"context"
	"course-go-avito-Orurh/internal/config"
	"course-go-avito-Orurh/internal/http/handlers"
	"course-go-avito-Orurh/internal/http/router"
	"course-go-avito-Orurh/internal/repository"
	"course-go-avito-Orurh/internal/service/courier"

	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/dig"
)

// MustBuildContainer constructs the application's DI container
func MustBuildContainer(ctx context.Context) *dig.Container {
	container, err := buildContainer(ctx)
	if err != nil {
		log.Fatalf("failed to build container: %v", err)
	}
	return container
}

func buildContainer(ctx context.Context) (*dig.Container, error) {
	container := dig.New()

	if err := registerCore(container, ctx); err != nil {
		return nil, fmt.Errorf("core: %w", err)
	}
	if err := registerDb(container); err != nil {
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
	)
}

func registerDb(container *dig.Container) error {
	providerDB := func(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
		return connectDbWithRetry(ctx, cfg.DB.DSN(), 10, time.Second)
	}
	return provideAll(container, providerDB)
}

func registerService(container *dig.Container) error {
	if err := container.Provide(repository.NewCourierRepo); err != nil {
		return fmt.Errorf("provide repo: %w", err)
	}
	if err := container.Provide(func() time.Duration { return 3 * time.Second }); err != nil {
		return fmt.Errorf("provide timeout: %w", err)
	}
	if err := container.Provide(func(r *repository.CourierRepo, d time.Duration) *courier.Service {
		return courier.NewService(r, d)
	}); err != nil {
		return fmt.Errorf("provide service: %w", err)
	}
	return nil
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
		handlers.NewCourierHandler,
		router.New,
		serverProvider,
	)
}
