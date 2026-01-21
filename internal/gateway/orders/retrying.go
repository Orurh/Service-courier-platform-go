package order

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"course-go-avito-Orurh/internal/logx"
)

type gateway interface {
	GetByID(context.Context, string) (*Order, error)
	ListFrom(context.Context, time.Time) ([]Order, error)
}

type counter interface {
	Inc()
}

// RetryConfig описывает поведение RetryingGateway
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// RetryingGateway реализует поведение RetryingGateway
type RetryingGateway struct {
	next    gateway
	logger  logx.Logger
	retries counter
	cfg     RetryConfig
	sleep   func(time.Duration)
}

// NewRetryingGateway конструктор который проверяет, что next не nil и возвращает RetryingGateway
func NewRetryingGateway(next gateway, logger logx.Logger, retries counter, cfg RetryConfig) *RetryingGateway {
	if next == nil {
		return nil
	}
	return &RetryingGateway{next: next, logger: logger, retries: retries, cfg: cfg, sleep: time.Sleep}
}

// GetByID реализует поведение RetryingGateway
func (g *RetryingGateway) GetByID(ctx context.Context, id string) (*Order, error) {
	// объявляем переменную lastErr
	var lastErr error
	// цикл по повторам
	for attempt := 1; attempt <= g.cfg.MaxAttempts; attempt++ {
		// вызываем GetByID
		ord, err := g.next.GetByID(ctx, id)
		if err == nil {
			return ord, nil
		}
		lastErr = err
		// проверяем условия повтора
		if ctx.Err() != nil || attempt == g.cfg.MaxAttempts || !isRetryable(err) {
			break
		}
		// вычисляем задержку
		delay := backoff(g.cfg.BaseDelay, g.cfg.MaxDelay, attempt)
		if g.retries != nil {
			g.retries.Inc()
		}
		// выводим лог о повторе
		g.logger.Warn("orders gateway retry",
			logx.String("method", "GetByID"),
			logx.Int("attempt", attempt),
			logx.Duration("delay", delay),
			logx.Any("err", err),
		)
		// ждем
		if !sleepWithContext(ctx, g.sleep, delay) {
			break
		}
	}
	return nil, lastErr
}

// ListFrom реализует поведение RetryingGateway
func (g *RetryingGateway) ListFrom(ctx context.Context, from time.Time) ([]Order, error) {
	var lastErr error
	for attempt := 1; attempt <= g.cfg.MaxAttempts; attempt++ {
		orders, err := g.next.ListFrom(ctx, from)
		if err == nil {
			return orders, nil
		}
		lastErr = err

		if ctx.Err() != nil || attempt == g.cfg.MaxAttempts || !isRetryable(err) {
			break
		}

		delay := backoff(g.cfg.BaseDelay, g.cfg.MaxDelay, attempt)
		if g.retries != nil {
			g.retries.Inc()
		}
		g.logger.Warn("orders gateway retry",
			logx.String("method", "ListFrom"),
			logx.Int("attempt", attempt),
			logx.Duration("delay", delay),
			logx.Any("err", err),
		)
		if !sleepWithContext(ctx, g.sleep, delay) {
			break
		}
	}
	return nil, lastErr
}

// isRetryable определяет, является ли ошибка повторяемой
func isRetryable(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	switch st.Code() {
	case codes.ResourceExhausted,
		codes.Unavailable,
		codes.DeadlineExceeded:
		return true
	default:
		return false
	}
}

// backoff вычисляет задержку повтора
func backoff(base, max time.Duration, attempt int) time.Duration {
	d := base << (attempt - 1)
	if d > max {
		return max
	}
	return d
}

func sleepWithContext(ctx context.Context, sleep func(time.Duration), d time.Duration) bool {
	if d <= 0 {
		return true
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
