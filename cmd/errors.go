package cmd

import (
	"errors"
	"fmt"
)

// Exit codes returned by aipim. Documented in README.
const (
	ExitOK            = 0
	ExitGeneric       = 1
	ExitUsage         = 2
	ExitConfig        = 3
	ExitAgentNotFound = 4
	ExitCancelled     = 5
)

// ExitError carries a process exit code alongside its error.
// main.go unwraps this to set os.Exit appropriately.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit code %d", e.Code)
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

// withCode wraps an error with a specific exit code.
func withCode(code int, err error) error {
	if err == nil {
		return nil
	}
	return &ExitError{Code: code, Err: err}
}

// usageErrorf returns an ExitUsage error formatted from a format string.
func usageErrorf(format string, args ...any) error {
	return withCode(ExitUsage, fmt.Errorf(format, args...))
}

// configErrorf returns an ExitConfig error.
func configErrorf(format string, args ...any) error {
	return withCode(ExitConfig, fmt.Errorf(format, args...))
}

// agentNotFoundErrorf returns an ExitAgentNotFound error.
func agentNotFoundErrorf(format string, args ...any) error {
	return withCode(ExitAgentNotFound, fmt.Errorf(format, args...))
}

// cancelledError signals user-initiated cancellation.
var cancelledError = &ExitError{Code: ExitCancelled, Err: errors.New("cancelled")}

// ExitCodeFor unwraps an error chain to find an ExitError and returns its code.
// Returns ExitGeneric if no ExitError is found, or ExitOK for nil.
func ExitCodeFor(err error) int {
	if err == nil {
		return ExitOK
	}
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}
	return ExitGeneric
}
