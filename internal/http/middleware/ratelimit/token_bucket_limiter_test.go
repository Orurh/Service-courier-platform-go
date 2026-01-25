package ratelimit

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
	require.True(t, l.Allow("ip1"), "expected allow #1")
	require.True(t, l.Allow("ip1"), "expected allow #2")
	require.False(t, l.Allow("ip1"), "expected block when bucket empty")

	// +1 sec => +1 token => allow once
	clk.Add(1 * time.Second)
	require.True(t, l.Allow("ip1"), "expected allow after refill")
	require.False(t, l.Allow("ip1"), "expected block (no tokens left)")

	// +10 sec => should cap at burst=2
	clk.Add(10 * time.Second)
	require.True(t, l.Allow("ip1"), "expected allow #1 after long refill")
	require.True(t, l.Allow("ip1"), "expected allow #2 after long refill")
	require.False(t, l.Allow("ip1"), "expected block after consuming burst again")
}

func TestTokenBucketLimiter_IsPerKey(t *testing.T) {
	t.Parallel()

	clk := newFakeClock(time.Unix(0, 0))
	l := NewTokenBucketLimiter(clk, Config{Rate: 1, Burst: 1})

	require.True(t, l.Allow("keyA"), "expected allow keyA #1")
	require.False(t, l.Allow("keyA"), "expected block keyA #2")
	require.True(t, l.Allow("keyB"), "expected allow keyB #1 (independent bucket)")
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

	require.Len(t, l.buckets, 2, "expected 2 buckets")

	// добавим 59 секунд)
	clk.Add(59 * time.Second)
	_ = l.Allow("B")

	// добавим еще 2 секунды
	clk.Add(2 * time.Second)
	_ = l.Allow("B")

	_, okA := l.buckets["A"]
	require.False(t, okA, "expected bucket A to be cleaned up")
	_, okB := l.buckets["B"]
	require.True(t, okB, "expected bucket B to remain")
}

func TestNewTokenBucketPerWindow_UsesLimitAsBurst(t *testing.T) {
	t.Parallel()

	clk := newFakeClock(time.Unix(0, 0))
	l := NewTokenBucketPerWindow(clk, 3, time.Second, 0, 0)

	for i := 1; i <= 3; i++ {
		require.True(t, l.Allow("k"), "expected allow #%d for burst=limit", i)
	}
	require.False(t, l.Allow("k"), "expected block after consuming burst")
}

func TestTokenBucketLimiter_MaxBucketsCapsNewKeys(t *testing.T) {
	t.Parallel()

	clk := newFakeClock(time.Unix(0, 0))
	l := NewTokenBucketLimiter(clk, Config{
		Rate:       10,
		Burst:      1,
		TTL:        0,
		MaxBuckets: 1,
	})

	require.True(t, l.Allow("A"), "expected allow for key A")
	require.False(t, l.Allow("B"), "expected deny for key B due to MaxBuckets cap")

	clk.Add(1 * time.Second)
	require.True(t, l.Allow("A"), "expected allow for key A after refill")
}

func TestTokenBucketLimiter_MaxBucketsWithTTL_AllowsNewKeyAfterCleanup(t *testing.T) {
	t.Parallel()

	clk := newFakeClock(time.Unix(0, 0))
	l := NewTokenBucketLimiter(clk, Config{
		Rate:       10,
		Burst:      1,
		TTL:        2 * time.Second,
		MaxBuckets: 1,
	})

	require.True(t, l.Allow("A"), "expected allow for key A")
	require.False(t, l.Allow("B"), "expected deny for key B due to MaxBuckets cap")

	clk.Add(61 * time.Second)

	require.True(t, l.Allow("B"), "expected allow for key B after cleanup freed bucket slot")
	require.False(t, l.Allow("A"), "expected deny for key A because MaxBuckets=1 and bucket B exists")
}
