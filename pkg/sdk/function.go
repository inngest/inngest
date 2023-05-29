package sdk

import (
	"fmt"

	"github.com/inngest/inngest/pkg/inngest"
)

// SDKFunction represents a function as specified via the SDK's registration request.
type SDKFunction struct {
	Name string `json:"name"`
	// ID is the function slug.
	ID string `json:"id"`

	// Triggers represent the triggers which start this function.
	Triggers []inngest.Trigger `json:"triggers"`

	// Concurrency allows limiting the concurrency of running functions, optionally constrained
	// by an individual concurrency key.
	Concurrency *inngest.Concurrency `json:"concurrency,omitempty"`
	// Idempotency allows the specification of an idempotency key by templating event
	// data, eg:
	//
	//  `event.data.order_id`.
	//
	// When specified, a function will run at most once per 24 hours for the given unique
	// key.
	Idempotency *string `json:"idempotency,omitempty"`

	// RateLimit allows specifying custom rate limiting for the function.
	RateLimit *inngest.RateLimit `json:"rateLimit,omitempty"`

	// Retries allows specifying the number of retries to attempt across all steps in the
	// function.
	Retries *int `json:"retries,omitempty"`

	// Cancel specifies cancellation signals for the function
	Cancel []inngest.Cancel `json:"cancel,omitempty"`

	Steps map[string]SDKStep `json:"steps"`
}

func (s SDKFunction) Function() (*inngest.Function, error) {
	f := inngest.Function{
		Name: s.Name,
		// Slug:        s.ID,
		Concurrency: s.Concurrency,
		Triggers:    s.Triggers,
		Idempotency: s.Idempotency,
		RateLimit:   s.RateLimit,
		Cancel:      s.Cancel,
	}

	for _, step := range s.Steps {
		url, ok := step.Runtime["url"].(string)
		if !ok {
			return nil, fmt.Errorf("No SDK URL provided for function '%s'", f.Name)
		}

		funcStep := inngest.Step{
			ID:   step.ID,
			Name: step.Name,
			URI:  url,
			// no concurrency keys are yet provided by the SDK
		}
		if step.Retries != nil {
			atts := step.Retries.Attempts
			funcStep.Retries = &atts
		}
		if step.Retries == nil && s.Retries != nil {
			// Use the function's defaults provided as syntactic sugar when registering functions
			funcStep.Retries = s.Retries
		}
		f.Steps = append(f.Steps, funcStep)
	}

	return &f, nil
}

// SDKStep represents the SDK's definition of a step;  a step is a node in a DAG of steps
// to be triggered by the function.
//
// Within an SDK, there is only one step right now (v1).
type SDKStep struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Runtime map[string]any `json:"runtime"`
	Retries *StepRetries   `json:"retries"`
}

type StepRetries struct {
	Attempts int `json:"attempts"`
}
