package ratelimit

import (
	"sync"
	"time"
)

// Config stores TokenBucketLimiter settings.
type Config struct {
	Rate  float64
	Burst int
	TTL   time.Duration
}

// TokenBucketLimiter per-key token bucket limiter.
type TokenBucketLimiter struct {
	cfg         Config
	clock       Clock
	mu          sync.RWMutex
	buckets     map[string]*bucket
	lastCleanup time.Time
}

type bucket struct {
	mu       sync.Mutex
	tokens   float64
	last     time.Time
	lastSeen time.Time
}

// NewTokenBucketLimiter creates limiter with explicit config and injected clock.
func NewTokenBucketLimiter(clock Clock, cfg Config) *TokenBucketLimiter {
	if clock == nil {
		clock = RealClock{}
	}
	if cfg.Rate <= 0 {
		cfg.Rate = 1
	}
	if cfg.Burst <= 0 {
		cfg.Burst = 1
	}
	return &TokenBucketLimiter{
		cfg:     cfg,
		clock:   clock,
		buckets: make(map[string]*bucket),
	}
}

// NewTokenBucketPerWindow is a convenience ctor for "limit per window".
func NewTokenBucketPerWindow(clock Clock, limit int, window time.Duration, ttl time.Duration) *TokenBucketLimiter {
	if window <= 0 {
		window = time.Second
	}
	if limit <= 0 {
		limit = 1
	}
	return NewTokenBucketLimiter(clock, Config{
		Rate:  float64(limit) / window.Seconds(),
		Burst: limit,
		TTL:   ttl,
	})
}

// Allow returns true if key is allowed to proceed.
func (l *TokenBucketLimiter) Allow(key string) bool {
	now := l.clock.Now()

	b := l.getOrCreateBucket(key, now)
	allowed := b.allow(now, l.cfg.Rate, float64(l.cfg.Burst))

	l.maybeCleanup(now)
	return allowed
}

func (l *TokenBucketLimiter) getOrCreateBucket(key string, now time.Time) *bucket {
	l.mu.RLock()
	b := l.buckets[key]
	l.mu.RUnlock()
	if b != nil {
		return b
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if b = l.buckets[key]; b != nil {
		return b
	}
	b = &bucket{
		tokens:   float64(l.cfg.Burst),
		last:     now,
		lastSeen: now,
	}
	l.buckets[key] = b
	return b
}

func (b *bucket) allow(now time.Time, rate float64, burst float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if dt := now.Sub(b.last); dt > 0 {
		b.tokens += dt.Seconds() * rate
		if b.tokens > burst {
			b.tokens = burst
		}
		b.last = now
	}
	b.lastSeen = now

	if b.tokens < 1.0 {
		return false
	}
	b.tokens -= 1.0
	return true
}

func (l *TokenBucketLimiter) maybeCleanup(now time.Time) {
	if l.cfg.TTL <= 0 {
		return
	}

	interval := time.Minute
	if half := l.cfg.TTL / 2; half > interval {
		interval = half
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.lastCleanup.IsZero() && now.Sub(l.lastCleanup) < interval {
		return
	}
	l.lastCleanup = now

	ttl := l.cfg.TTL
	for k, b := range l.buckets {
		b.mu.Lock()
		seen := b.lastSeen
		b.mu.Unlock()

		if now.Sub(seen) > ttl {
			delete(l.buckets, k)
		}
	}
}
