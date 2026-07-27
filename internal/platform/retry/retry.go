// Package retry provides a minimal exponential backoff abstraction and a
// small helper for retrying an operation. It has no knowledge of what it is
// retrying; callers (e.g. the database connection warm-up) supply the
// operation.
package retry

import (
	"context"
	"time"
)

// Backoff computes how long to wait before a given attempt. attempt is
// zero-indexed: Next(0) is the delay before the first retry.
type Backoff interface {
	Next(attempt int) time.Duration
}

// ExponentialBackoff doubles its delay on every attempt, capped at Max.
type ExponentialBackoff struct {
	Base time.Duration
	Max  time.Duration
}

// NewExponentialBackoff returns an ExponentialBackoff starting at base and
// never exceeding max.
func NewExponentialBackoff(base, max time.Duration) ExponentialBackoff {
	return ExponentialBackoff{Base: base, Max: max}
}

func (b ExponentialBackoff) Next(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	delay := b.Base
	for i := 0; i < attempt; i++ {
		delay *= 2
		if delay <= 0 || delay > b.Max { // overflow or exceeded cap
			return b.Max
		}
	}
	if delay > b.Max {
		return b.Max
	}
	return delay
}

// Do calls fn until it succeeds, ctx is cancelled, or maxAttempts have been
// made. maxAttempts <= 0 means retry indefinitely (bounded only by ctx).
// The delay between attempts is taken from backoff.
func Do(ctx context.Context, backoff Backoff, maxAttempts int, fn func() error) error {
	var err error

	for attempt := 0; maxAttempts <= 0 || attempt < maxAttempts; attempt++ {
		if err = fn(); err == nil {
			return nil
		}

		if maxAttempts > 0 && attempt == maxAttempts-1 {
			break
		}

		timer := time.NewTimer(backoff.Next(attempt))
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	return err
}
