package inngest

import "github.com/inngest/inngest/pkg/consts"

// Step represents a single unit of code (action) which runs as part of a step function, in a DAG.
type Step struct {
	// Name is the name for the given step in a function.
	Name string `json:"name"`
	// URI represents how this function is invoked, eg https://example.com/api/inngest?step=foo,
	// or arn://xyz for lambda functions.
	URI string `json:"uri"`

	// Retries optionally overrides retries for this step, allowing steps to have differing retry
	// counts to the core function.
	Retries *int `json:"retries"`

	// ConcurrencyKey allows steps to share concurrency slots across multiple functions, eg. for
	// rate limiting across multiple functions.
	ConcurrencyKey *string `json:"concurrencyKey"`
}

// RetryCount returns the number of retries for this step.
func (s Step) RetryCount() int {
	if s.Retries != nil {
		return *s.Retries
	}
	return consts.DefaultRetryCount
}
