package ratelimit

import "time"

// Clock provides current time.
type Clock interface {
	Now() time.Time
}

// RealClock is the default clock.
type RealClock struct{}

// Now returns current time.
func (RealClock) Now() time.Time { return time.Now() }
