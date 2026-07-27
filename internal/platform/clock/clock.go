// Package clock abstracts time so callers that stamp records (created_at,
// audit history, workflow timestamps, ...) can be tested without depending
// on the wall clock.
package clock

import "time"

// Clock returns the current time.
type Clock interface {
	Now() time.Time
}

// real is the production Clock backed by time.Now.
type real struct{}

// New returns a Clock backed by the system wall clock.
func New() Clock {
	return real{}
}

func (real) Now() time.Time {
	return time.Now()
}

// Frozen is a Clock that always returns the same instant. It exists for
// tests that need deterministic timestamps.
type Frozen struct {
	at time.Time
}

// NewFrozen returns a Clock fixed at t.
func NewFrozen(t time.Time) Frozen {
	return Frozen{at: t}
}

func (f Frozen) Now() time.Time {
	return f.at
}
