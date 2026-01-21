package ratelimit

import (
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock(t time.Time) *fakeClock { return &fakeClock{now: t} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Add(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

func TestTokenBucketLimiter_BurstThenBlocksThenRefills(t *testing.T) {
	t.Parallel()

	clk := newFakeClock(time.Unix(0, 0))
	l := NewTokenBucketLimiter(clk, Config{
		Rate:  1, // 1 token/sec
		Burst: 2, // capacity 2
	})

	// full burst at start => 2 allowed
	if !l.Allow("ip1") {
		t.Fatalf("expected allow #1")
	}
	if !l.Allow("ip1") {
		t.Fatalf("expected allow #2")
	}
	if l.Allow("ip1") {
		t.Fatalf("expected block when bucket empty")
	}

	// +1 sec => +1 token => allow once
	clk.Add(1 * time.Second)
	if !l.Allow("ip1") {
		t.Fatalf("expected allow after refill")
	}
	if l.Allow("ip1") {
		t.Fatalf("expected block (no tokens left)")
	}

	// +10 sec => should cap at burst=2
	clk.Add(10 * time.Second)
	if !l.Allow("ip1") {
		t.Fatalf("expected allow #1 after long refill (capped by burst)")
	}
	if !l.Allow("ip1") {
		t.Fatalf("expected allow #2 after long refill (capped by burst)")
	}
	if l.Allow("ip1") {
		t.Fatalf("expected block after consuming burst again")
	}
}

func TestTokenBucketLimiter_IsPerKey(t *testing.T) {
	t.Parallel()

	clk := newFakeClock(time.Unix(0, 0))
	l := NewTokenBucketLimiter(clk, Config{Rate: 1, Burst: 1})

	// keyA consumes its only token
	if !l.Allow("keyA") {
		t.Fatalf("expected allow keyA #1")
	}
	if l.Allow("keyA") {
		t.Fatalf("expected block keyA #2")
	}

	if !l.Allow("keyB") {
		t.Fatalf("expected allow keyB #1 (independent bucket)")
	}
}

func TestTokenBucketLimiter_TTLCleanupRemovesIdleBuckets(t *testing.T) {
	t.Parallel()

	clk := newFakeClock(time.Unix(0, 0))
	l := NewTokenBucketLimiter(clk, Config{
		Rate:  10,
		Burst: 1,
		TTL:   2 * time.Second,
	})

	_ = l.Allow("A")
	_ = l.Allow("B")

	if got := len(l.buckets); got != 2 {
		t.Fatalf("expected 2 buckets, got %d", got)
	}

	// добавим 59 секунд)
	clk.Add(59 * time.Second)
	_ = l.Allow("B")

	// добавим еще 2 секунды
	clk.Add(2 * time.Second)
	_ = l.Allow("B")

	if _, ok := l.buckets["A"]; ok {
		t.Fatalf("expected bucket A to be cleaned up")
	}
	if _, ok := l.buckets["B"]; !ok {
		t.Fatalf("expected bucket B to remain")
	}
}

func TestNewTokenBucketPerWindow_UsesLimitAsBurst(t *testing.T) {
	t.Parallel()

	clk := newFakeClock(time.Unix(0, 0))
	l := NewTokenBucketPerWindow(clk, 3, time.Second, 0)

	for i := 1; i <= 3; i++ {
		if !l.Allow("k") {
			t.Fatalf("expected allow #%d for burst=limit", i)
		}
	}
	if l.Allow("k") {
		t.Fatalf("expected block after consuming burst")
	}
}
