package diagnostics_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics"
)

func TestBasicONUCheckName(t *testing.T) {
	check := diagnostics.NewBasicONUCheck()

	if got := check.Name(); got != diagnostics.BasicONUCheckName {
		t.Errorf("Name() = %q, want %q", got, diagnostics.BasicONUCheckName)
	}
}

// TestBasicONUCheckRunReturnsPlaceholderResult proves goal 5's exact
// contract: a successful Result containing one Section whose Command and
// Output are both "not implemented."
func TestBasicONUCheckRunReturnsPlaceholderResult(t *testing.T) {
	check := diagnostics.NewBasicONUCheck()

	result, err := check.Run(context.Background(), diagnostics.Request{ONUID: uuid.New()})
	if err != nil {
		t.Fatalf("Run() = %v, want nil error", err)
	}
	if result == nil {
		t.Fatal("Run() returned a nil Result alongside a nil error")
	}

	if result.Name != diagnostics.BasicONUCheckName {
		t.Errorf("Result.Name = %q, want %q", result.Name, diagnostics.BasicONUCheckName)
	}
	if len(result.Sections) != 1 {
		t.Fatalf("len(Result.Sections) = %d, want 1", len(result.Sections))
	}
	if result.Sections[0].Command != "not implemented" {
		t.Errorf("Sections[0].Command = %q, want %q", result.Sections[0].Command, "not implemented")
	}
	if result.Sections[0].Output != "not implemented" {
		t.Errorf("Sections[0].Output = %q, want %q", result.Sections[0].Output, "not implemented")
	}
}

// TestBasicONUCheckRunSetsTimestamps proves StartedAt, FinishedAt, and
// Duration are all populated and internally consistent, even though the
// placeholder itself does no real work.
func TestBasicONUCheckRunSetsTimestamps(t *testing.T) {
	check := diagnostics.NewBasicONUCheck()

	result, err := check.Run(context.Background(), diagnostics.Request{ONUID: uuid.New()})
	if err != nil {
		t.Fatalf("Run() = %v", err)
	}

	if result.StartedAt.IsZero() {
		t.Error("Result.StartedAt is zero, want a real timestamp")
	}
	if result.FinishedAt.IsZero() {
		t.Error("Result.FinishedAt is zero, want a real timestamp")
	}
	if result.FinishedAt.Before(result.StartedAt) {
		t.Errorf("Result.FinishedAt (%v) is before StartedAt (%v)", result.FinishedAt, result.StartedAt)
	}
	if result.Duration != result.FinishedAt.Sub(result.StartedAt) {
		t.Errorf("Result.Duration = %v, want %v (FinishedAt - StartedAt)", result.Duration, result.FinishedAt.Sub(result.StartedAt))
	}
}

// TestBasicONUCheckRunIgnoresRequest proves Run's placeholder behavior
// does not depend on what ONUID (or its zero value) was requested — the
// framework has nothing yet to resolve topology against (see Request's
// doc comment).
func TestBasicONUCheckRunIgnoresRequest(t *testing.T) {
	check := diagnostics.NewBasicONUCheck()

	result, err := check.Run(context.Background(), diagnostics.Request{})
	if err != nil {
		t.Fatalf("Run() = %v, want nil error even for a zero-value Request", err)
	}
	if result.Name != diagnostics.BasicONUCheckName {
		t.Errorf("Result.Name = %q, want %q", result.Name, diagnostics.BasicONUCheckName)
	}
}
