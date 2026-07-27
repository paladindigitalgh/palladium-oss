// Package id abstracts UUID generation. Every table in Palladium uses a UUID
// primary key (see CLAUDE.md database rules); this package is the single
// place that decision is implemented, so it can be swapped or stubbed in
// tests without every caller importing a UUID library directly.
package id

import "github.com/google/uuid"

// Generator produces UUIDs.
type Generator interface {
	New() uuid.UUID
}

// generator is the production Generator backed by google/uuid's random
// (version 4) generation.
type generator struct{}

// New returns a Generator backed by cryptographically random UUIDs.
func New() Generator {
	return generator{}
}

func (generator) New() uuid.UUID {
	return uuid.New()
}

// Static is a Generator that always returns the same value. It exists for
// tests that need deterministic identifiers.
type Static struct {
	Value uuid.UUID
}

func (s Static) New() uuid.UUID {
	return s.Value
}
