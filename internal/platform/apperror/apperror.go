// Package apperror defines a small, transport-agnostic error taxonomy that
// the rest of the application builds on. Repositories and services classify
// failures by Kind (not found, conflict, invalid input, ...); the API layer
// later maps a Kind to an HTTP status without needing to know which package
// produced the error. This package contains no domain concepts.
package apperror

import (
	"errors"
	"fmt"
)

// Kind classifies what went wrong, independent of transport.
type Kind string

const (
	// KindInvalid means the caller supplied unusable input.
	KindInvalid Kind = "invalid"
	// KindNotFound means the requested resource does not exist.
	KindNotFound Kind = "not_found"
	// KindConflict means the request conflicts with the current state.
	KindConflict Kind = "conflict"
	// KindUnauthorized means the caller is not authenticated.
	KindUnauthorized Kind = "unauthorized"
	// KindForbidden means the caller is authenticated but not permitted.
	KindForbidden Kind = "forbidden"
	// KindUnavailable means a dependency (database, plugin, ...) could not
	// be reached.
	KindUnavailable Kind = "unavailable"
	// KindInternal means an unexpected, unclassified failure occurred.
	KindInternal Kind = "internal"
)

// Error is an error carrying a Kind and an optional wrapped cause.
type Error struct {
	Kind    Kind
	Message string
	Err     error
}

// New builds an Error with no wrapped cause.
func New(kind Kind, message string) *Error {
	return &Error{Kind: kind, Message: message}
}

// Newf builds an Error with a formatted message.
func Newf(kind Kind, format string, args ...any) *Error {
	return New(kind, fmt.Sprintf(format, args...))
}

// Wrap builds an Error that preserves err as its cause via Unwrap.
func Wrap(kind Kind, message string, err error) *Error {
	return &Error{Kind: kind, Message: message, Err: err}
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

// NotFound builds a KindNotFound error.
func NotFound(message string) *Error { return New(KindNotFound, message) }

// Conflict builds a KindConflict error.
func Conflict(message string) *Error { return New(KindConflict, message) }

// Invalid builds a KindInvalid error.
func Invalid(message string) *Error { return New(KindInvalid, message) }

// Unauthorized builds a KindUnauthorized error.
func Unauthorized(message string) *Error { return New(KindUnauthorized, message) }

// Forbidden builds a KindForbidden error.
func Forbidden(message string) *Error { return New(KindForbidden, message) }

// Unavailable builds a KindUnavailable error wrapping the underlying cause.
func Unavailable(message string, err error) *Error { return Wrap(KindUnavailable, message, err) }

// Internal builds a KindInternal error wrapping the underlying cause.
func Internal(message string, err error) *Error { return Wrap(KindInternal, message, err) }

// KindOf returns the Kind of err if it is (or wraps) an *Error, and
// KindInternal otherwise. It lets callers classify errors returned by code
// that may or may not have originated in this package.
func KindOf(err error) Kind {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Kind
	}
	return KindInternal
}

// Is reports whether err is (or wraps) an *Error of the given Kind.
func Is(err error, kind Kind) bool {
	return KindOf(err) == kind
}
