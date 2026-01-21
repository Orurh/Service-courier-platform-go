package ratelimit

// NopLimiter is a no-op limiter
type NopLimiter struct{}

// Allow always returns true
func (NopLimiter) Allow(string) bool { return true }

// NewNopLimiter returns NopLimiter
func NewNopLimiter() Limiter { return NopLimiter{} }
