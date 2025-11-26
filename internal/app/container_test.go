package app

import (
	"context"
	"log"
	"net/http"
	"testing"
	"time"

	"course-go-avito-Orurh/internal/config"
	"course-go-avito-Orurh/internal/http/handlers"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/dig"
)

func newTestLogger() *log.Logger {
	return log.New(log.Writer(), "", 0)
}

func setupTestContainer(t *testing.T) *dig.Container {
	t.Helper()

	c := dig.New()

	providers := []struct {
		name     string
		provider interface{}
	}{
		{"context", func() context.Context { return context.Background() }},
		{"logger", func() *log.Logger { return newTestLogger() }},
		{"config", func() *config.Config { return &config.Config{Port: 8080} }},
		{"pgxpool", func() *pgxpool.Pool { return &pgxpool.Pool{} }},
	}

	for _, p := range providers {
		if err := c.Provide(p.provider); err != nil {
			t.Fatalf("provide %s: %v", p.name, err)
		}
	}

	if err := registerService(c); err != nil {
		t.Fatalf("registerService error: %v", err)
	}
	if err := registerHTTP(c); err != nil {
		t.Fatalf("registerHTTP error: %v", err)
	}

	return c
}

func verifyServer(t *testing.T, srv *http.Server) {
	t.Helper()

	if srv == nil {
		t.Fatal("http.Server is nil")
	}
	if srv.Addr != ":8080" {
		t.Fatalf("expected server Addr ':8080', got %q", srv.Addr)
	}
	if srv.ReadHeaderTimeout <= 0 {
		t.Fatalf("expected positive ReadHeaderTimeout, got %v", srv.ReadHeaderTimeout)
	}
	if srv.ReadTimeout <= 0 {
		t.Fatalf("expected positive ReadTimeout, got %v", srv.ReadTimeout)
	}
	if srv.WriteTimeout <= 0 {
		t.Fatalf("expected positive WriteTimeout, got %v", srv.WriteTimeout)
	}
	if srv.IdleTimeout <= 0 {
		t.Fatalf("expected positive IdleTimeout, got %v", srv.IdleTimeout)
	}
}

func verifyHandlers(t *testing.T, base *handlers.Handlers, courierHandler *handlers.CourierHandler, deliveryHandler *handlers.DeliveryHandler) {
	t.Helper()

	if base == nil {
		t.Fatal("*handlers.Handlers is nil")
	}
	if courierHandler == nil {
		t.Fatal("*handlers.CourierHandler is nil")
	}
	if deliveryHandler == nil {
		t.Fatal("*handlers.DeliveryHandler is nil")
	}
}

func TestRegisterServiceAndHTTP_ProvidesHttpServerAndHandlers(t *testing.T) {
	t.Parallel()

	c := setupTestContainer(t)

	err := c.Invoke(func(
		srv *http.Server,
		base *handlers.Handlers,
		courierHandler *handlers.CourierHandler,
		deliveryHandler *handlers.DeliveryHandler,
	) {
		verifyServer(t, srv)
		verifyHandlers(t, base, courierHandler, deliveryHandler)
	})
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}
}

func TestProvideAll_Success(t *testing.T) {
	t.Parallel()

	c := dig.New()

	err := provideAll(c,
		func() context.Context { return context.Background() },
		func() time.Duration { return 3 * time.Second },
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = c.Invoke(func(ctx context.Context, d time.Duration) {
		if ctx == nil {
			t.Fatalf("context is nil")
		}
		if d != 3*time.Second {
			t.Fatalf("expected 3s, got %v", d)
		}
	})
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}
}

func TestProvideAll_InvalidProvider(t *testing.T) {
	t.Parallel()

	c := dig.New()

	type bad struct{}
	err := provideAll(c, bad{})
	if err == nil {
		t.Fatal("expected error for invalid provider, got nil")
	}
}
