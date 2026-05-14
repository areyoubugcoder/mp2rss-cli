// Package errs defines exit-code-aware typed errors for the CLI.
//
// Commands return *Error values; cmd.Execute() inspects the code and exits
// with the matching numeric exit code per the mp2rss CLI contract:
//
//	0 success | 1 generic | 2 args | 3 auth | 4 not found | 5 upstream unavailable
package errs

import (
	"errors"
	"fmt"
)

// Exit codes per the CLI contract.
const (
	CodeOK           = 0
	CodeGeneric      = 1
	CodeArgs         = 2
	CodeAuth         = 3
	CodeNotFound     = 4
	CodeUpstreamDown = 5
)

// Error is a CLI-visible error with both a human message and an exit code.
//
// HTTPStatus is the upstream HTTP status code when the error originated from
// an API response. It is 0 for purely local errors.
type Error struct {
	Code       int    // exit code
	HTTPStatus int    // upstream HTTP status if any
	Message    string // user-facing message
	Cause      error  // wrapped underlying error (for `errors.Is/As`)
}

// Error returns the message.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// Unwrap exposes the wrapped error.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// Newf creates a typed CLI error.
func Newf(code int, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Wrap turns any error into a CLI error with the given exit code.
func Wrap(code int, err error) *Error {
	if err == nil {
		return nil
	}
	return &Error{Code: code, Message: err.Error(), Cause: err}
}

// As converts err to *Error. Returns (nil, false) for non-typed errors.
func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}
