// Package validate provides generic, field-oriented validation helpers.
// It knows nothing about any domain model; callers build up an Errors value
// field by field and convert it to a single apperror.Error when done.
package validate

import (
	"strings"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// FieldError describes why a single field failed validation.
type FieldError struct {
	Field   string
	Message string
}

// Errors accumulates FieldErrors for a single validation pass.
type Errors struct {
	fields []FieldError
}

// New returns an empty Errors accumulator.
func New() *Errors {
	return &Errors{}
}

// Add records a failure for field.
func (e *Errors) Add(field, message string) {
	e.fields = append(e.fields, FieldError{Field: field, Message: message})
}

// HasErrors reports whether any field has failed so far.
func (e *Errors) HasErrors() bool {
	return len(e.fields) > 0
}

// Fields returns a copy of the accumulated field errors.
func (e *Errors) Fields() []FieldError {
	return append([]FieldError(nil), e.fields...)
}

// Error implements the error interface so an *Errors can be used directly
// wherever an error is expected.
func (e *Errors) Error() string {
	parts := make([]string, len(e.fields))
	for i, f := range e.fields {
		parts[i] = f.Field + ": " + f.Message
	}
	return strings.Join(parts, "; ")
}

// Err returns nil if nothing failed, otherwise a *apperror.Error of
// KindInvalid describing every failed field.
func (e *Errors) Err() error {
	if !e.HasErrors() {
		return nil
	}
	return apperror.Invalid(e.Error())
}

// Required reports whether value contains anything besides whitespace.
func Required(value string) bool {
	return strings.TrimSpace(value) != ""
}

// MaxLength reports whether value is no longer than max runes.
func MaxLength(value string, max int) bool {
	return len([]rune(value)) <= max
}

// MinLength reports whether value is at least min runes long.
func MinLength(value string, min int) bool {
	return len([]rune(value)) >= min
}

// InRange reports whether value falls within [min, max] inclusive.
func InRange(value, min, max int) bool {
	return value >= min && value <= max
}
