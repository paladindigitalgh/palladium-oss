package id_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

func TestNewGeneratorProducesUniqueValues(t *testing.T) {
	gen := id.New()

	first := gen.New()
	second := gen.New()

	if first == uuid.Nil {
		t.Fatal("New() returned the nil UUID")
	}
	if first == second {
		t.Errorf("two calls to New() returned the same UUID: %s", first)
	}
	if first.Version() != 4 {
		t.Errorf("version = %d, want 4 (random)", first.Version())
	}
}

func TestStaticGeneratorReturnsFixedValue(t *testing.T) {
	want := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	gen := id.Static{Value: want}

	if got := gen.New(); got != want {
		t.Errorf("New() = %s, want %s", got, want)
	}
	if got := gen.New(); got != want {
		t.Errorf("second New() = %s, want %s", got, want)
	}
}
