package inngest

import (
	"github.com/inngest/inngest/pkg/consts"
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
