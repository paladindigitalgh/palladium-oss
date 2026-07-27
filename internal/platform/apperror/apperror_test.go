package apperror_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

func TestKindOfUnwrapsWrappedErrors(t *testing.T) {
	cause := errors.New("connection refused")
	err := apperror.Unavailable("database ping failed", cause)

	wrapped := fmt.Errorf("startup: %w", err)

	if got := apperror.KindOf(wrapped); got != apperror.KindUnavailable {
		t.Errorf("KindOf() = %q, want %q", got, apperror.KindUnavailable)
	}
	if !errors.Is(wrapped, cause) {
		t.Error("errors.Is() did not find the wrapped cause")
	}
}

func TestKindOfDefaultsToInternalForForeignErrors(t *testing.T) {
	if got := apperror.KindOf(errors.New("boom")); got != apperror.KindInternal {
		t.Errorf("KindOf() = %q, want %q", got, apperror.KindInternal)
	}
}

func TestIsMatchesKind(t *testing.T) {
	err := apperror.NotFound("customer not found")

	if !apperror.Is(err, apperror.KindNotFound) {
		t.Error("Is() = false, want true for matching kind")
	}
	if apperror.Is(err, apperror.KindConflict) {
		t.Error("Is() = true, want false for non-matching kind")
	}
}

func TestErrorMessageIncludesCause(t *testing.T) {
	cause := errors.New("timeout")
	err := apperror.Wrap(apperror.KindUnavailable, "connect failed", cause)

	want := "connect failed: timeout"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
