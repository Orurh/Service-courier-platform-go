package ratelimit

// Limiter is a rate limiter
type Limiter interface {
	Allow(key string) bool
}
