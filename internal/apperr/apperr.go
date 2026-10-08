// Package apperr classifies errors so the command-line entry point can map
// every failure to a stable exit code.
package apperr

import (
	"errors"
	"fmt"
)

// Kind describes the category of an error.
type Kind int

const (
	// Usage means invalid arguments or options (exit code 2).
	Usage Kind = iota
	// Config means missing or invalid configuration (exit code 1).
	Config
	// Transport means a network or transport failure (exit code 1).
	Transport
)

// Error carries a Kind alongside the underlying error.
type Error struct {
	Kind Kind
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }

// Unwrap allows errors.Is/errors.As to reach the wrapped error.
func (e *Error) Unwrap() error { return e.Err }

// New wraps err with the given Kind. It returns nil when err is nil.
func New(kind Kind, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Kind: kind, Err: err}
}

// Usagef builds a formatted Usage error.
func Usagef(format string, args ...any) error {
	return &Error{Kind: Usage, Err: fmt.Errorf(format, args...)}
}

// Configf builds a formatted Config error.
func Configf(format string, args ...any) error {
	return &Error{Kind: Config, Err: fmt.Errorf(format, args...)}
}

// Transportf builds a formatted Transport error.
func Transportf(format string, args ...any) error {
	return &Error{Kind: Transport, Err: fmt.Errorf(format, args...)}
}

// KindOf reports the Kind of err. Unclassified errors default to Transport so
// that an unexpected failure is never mistaken for success.
func KindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return Transport
}
