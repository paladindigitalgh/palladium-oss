package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/retry"
)

// zeroBackoff never actually waits, so tests run instantly.
type zeroBackoff struct{}

func (zeroBackoff) Next(attempt int) time.Duration { return 0 }

func TestExponentialBackoffDoublesAndCaps(t *testing.T) {
	b := retry.NewExponentialBackoff(10*time.Millisecond, 100*time.Millisecond)

	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 10 * time.Millisecond},
		{1, 20 * time.Millisecond},
		{2, 40 * time.Millisecond},
		{3, 80 * time.Millisecond},
		{4, 100 * time.Millisecond}, // would be 160ms uncapped
		{10, 100 * time.Millisecond},
	}

	for _, c := range cases {
		if got := b.Next(c.attempt); got != c.want {
			t.Errorf("Next(%d) = %v, want %v", c.attempt, got, c.want)
		}
	}
}

func TestDoSucceedsWithoutRetrying(t *testing.T) {
	calls := 0
	err := retry.Do(context.Background(), zeroBackoff{}, 3, func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("Do() = %v, want nil", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestDoRetriesUntilSuccess(t *testing.T) {
	calls := 0
	err := retry.Do(context.Background(), zeroBackoff{}, 5, func() error {
		calls++
		if calls < 3 {
			return errors.New("not yet")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Do() = %v, want nil", err)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestDoReturnsLastErrorAfterMaxAttempts(t *testing.T) {
	calls := 0
	sentinel := errors.New("still failing")
	err := retry.Do(context.Background(), zeroBackoff{}, 3, func() error {
		calls++
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("Do() = %v, want %v", err, sentinel)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestDoStopsWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	err := retry.Do(ctx, retry.NewExponentialBackoff(time.Hour, time.Hour), 5, func() error {
		calls++
		return errors.New("always fails")
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Do() = %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (one attempt before the cancelled wait)", calls)
	}
}
