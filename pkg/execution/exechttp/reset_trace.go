package exechttp

import (
	"errors"
	"syscall"

	"github.com/inngest/go-httpstat"
)

// ResetTraceError carries a snapshot of the HTTP trace for a TCP reset. Unwrap
// returns the original client error, which may normalize a response-body failure.
type ResetTraceError struct {
	err      error
	Cause    error
	Snapshot httpstat.Snapshot
}

func (e *ResetTraceError) Error() string { return e.err.Error() }
func (e *ResetTraceError) Unwrap() error { return e.err }

func (e ExtendedClient) withResetTrace(result *httpstat.Result, err, cause error) error {
	if !e.resetTrace || !errors.Is(cause, syscall.ECONNRESET) {
		return err
	}
	return &ResetTraceError{err: err, Cause: cause, Snapshot: result.Snapshot()}
}
