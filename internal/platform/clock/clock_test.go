package clock_test

import (
	"testing"
	"time"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
)

func TestNewReflectsWallClock(t *testing.T) {
	c := clock.New()

	before := time.Now()
	got := c.Now()
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Errorf("Now() = %v, want between %v and %v", got, before, after)
	}
}

func TestFrozenAlwaysReturnsSameInstant(t *testing.T) {
	fixed := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	c := clock.NewFrozen(fixed)

	if got := c.Now(); !got.Equal(fixed) {
		t.Errorf("Now() = %v, want %v", got, fixed)
	}
	if got := c.Now(); !got.Equal(fixed) {
		t.Errorf("second Now() = %v, want %v", got, fixed)
	}
}
