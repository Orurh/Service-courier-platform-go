package app

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/dig"

	"course-go-avito-Orurh/internal/http/middleware/ratelimit"
	"course-go-avito-Orurh/internal/logx"
)

func newRateLimiter(clock ratelimit.Clock) ratelimit.Limiter {
	return ratelimit.NewTokenBucketLimiter(clock, ratelimit.Config{
		Rate:  5,
		Burst: 5,
		TTL:   10 * time.Minute,
	})
}

func newRateLimitClock() ratelimit.Clock {
	return ratelimit.RealClock{}
}

type rateLimitIn struct {
	dig.In
	Logger  logx.Logger
	Counter prometheus.Counter `name:"rate_limit_exceeded_total"`
	Limiter ratelimit.Limiter
}

func newRateLimitMiddleware(in rateLimitIn) *ratelimit.Middleware {
	return ratelimit.New(in.Logger, in.Counter, in.Limiter)
}
