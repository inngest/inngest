package inngest

import (
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/xhit/go-str2duration/v2"
)

// Step represents a single unit of code (action) which runs as part of a step function, in a DAG.
type Step struct {
	ID string `json:"id"`
	// Name is the name for the given step in a function.
	Name string `json:"name"`
	// URI represents how this function is invoked, eg https://example.com/api/inngest?step=foo,
	// or arn://xyz for lambda functions.
	URI string `json:"uri"`

	// Retries optionally overrides retries for this step, allowing steps to have differing retry
	// counts to the core function.
	Retries *int `json:"retries,omitempty"`

	// RetryDelay optionally specifies a fixed delay between retry attempts for this step.
	// This overrides the default exponential backoff table. The value should be a duration
	// string, e.g. "30s", "5m", "1h".
	RetryDelay *string `json:"retryDelay,omitempty"`
}

// RetryCount returns the number of retries for this step.
func (s Step) RetryCount() int {
	if s.Retries != nil {
		// This should be handled elsewhere, but we'll also handle it here just
		// in case.
		return min(*s.Retries, consts.MaxRetries)
	}
	return consts.DefaultRetryCount
}

// GetRetryDelay parses and returns the configured retry delay duration for this step.
// Returns nil if no custom retry delay is configured.
// The returned duration is clamped between consts.MinRetryDuration and consts.MaxRetryDuration.
func (s Step) GetRetryDelay() (*time.Duration, error) {
	if s.RetryDelay == nil {
		return nil, nil
	}
	dur, err := str2duration.ParseDuration(*s.RetryDelay)
	if err != nil {
		return nil, fmt.Errorf("invalid retryDelay %q: %w", *s.RetryDelay, err)
	}
	if dur < consts.MinRetryDuration {
		dur = consts.MinRetryDuration
	}
	if dur > consts.MaxRetryDuration {
		dur = consts.MaxRetryDuration
	}
	return &dur, nil
}
