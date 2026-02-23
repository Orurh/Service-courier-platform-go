package app

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/dig"

	"course-go-avito-Orurh/internal/config"
	"course-go-avito-Orurh/internal/http/middleware/ratelimit"
	"course-go-avito-Orurh/internal/logx"
)

func newRateLimiter(cfg *config.Config, clock ratelimit.Clock) ratelimit.Limiter {
	rl := cfg.RateLimit
	if !rl.Enabled {
		return ratelimit.NopLimiter{}
	}
	return ratelimit.NewTokenBucketLimiter(clock, ratelimit.Config{
		Rate:       rl.Rate,
		Burst:      rl.Burst,
		TTL:        rl.TTL,
		MaxBuckets: rl.MaxBuckets,
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
