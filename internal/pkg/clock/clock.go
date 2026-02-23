package clock

import "time"

// Clock returns the current time.
type Clock interface {
	Now() time.Time
}

// Real is a Clock that returns the actual system time.
type Real struct{}

// Now returns time.Now().
func (Real) Now() time.Time { return time.Now() }

// Fixed is a Clock that always returns the same time. Useful in tests.
type Fixed struct {
	T time.Time
}

// Now returns the fixed time.
func (f Fixed) Now() time.Time { return f.T }