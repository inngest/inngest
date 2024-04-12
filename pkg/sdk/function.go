package sdk

import (
	"fmt"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/inngest"
)

// SDKFunction represents a function as specified via the SDK's registration request.
type SDKFunction struct {
	Name string `json:"name"`

	// ID is the function slug.
	Slug string `json:"id"`

	// Triggers represent the triggers which start this function.
	Triggers []inngest.Trigger `json:"triggers"`

	// Concurrency allows limiting the concurrency of running functions, optionally constrained
	// by an individual concurrency key.
	//
	// This may be an int OR a struct, for backwards compatibility.
	Concurrency *inngest.ConcurrencyLimits `json:"concurrency,omitempty"`

	// Priority represents the priority information for this function.
	Priority *inngest.Priority `json:"priority,omitempty"`

	// EventBatch determines how a function will process a list of incoming events
	EventBatch map[string]any `json:"batchEvents,omitempty"`

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

	// Throttle represents a soft rate limit for gating function starts.  Any function runs
	// over the throttle period will be enqueued in the backlog to run at the next available
	// time.
	Throttle *inngest.Throttle `json:"throttle,omitempty"`

	// Retries allows specifying the number of retries to attempt across all steps in the
	// function.
	Retries *int `json:"retries,omitempty"`

	Debounce *inngest.Debounce `json:"debounce,omitempty"`

	Timeouts *inngest.Timeouts `json:"timeouts,omitempty"`

	// Cancel specifies cancellation signals for the function
	Cancel []inngest.Cancel `json:"cancel,omitempty"`

	Steps map[string]SDKStep `json:"steps"`
}

func (s SDKFunction) Function() (*inngest.Function, error) {
	f := inngest.Function{
		Name:        s.Name,
		Slug:        s.Slug,
		Concurrency: s.Concurrency,
		Triggers:    s.Triggers,
		Priority:    s.Priority,
		RateLimit:   s.RateLimit,
		Throttle:    s.Throttle,
		Cancel:      s.Cancel,
		Debounce:    s.Debounce,
		Timeouts:    s.Timeouts,
	}
	// Ensure we set the slug here if s.ID is nil.  This defaults to using
	// the slugged version of the function name.
	f.Slug = f.GetSlug()

	eventbatch, err := inngest.NewEventBatchConfig(s.EventBatch)
	if err != nil {
		return nil, err
	}
	f.EventBatch = eventbatch
	if s.Idempotency != nil {
		f.RateLimit = &inngest.RateLimit{
			Limit:  1,
			Period: consts.FunctionIdempotencyPeriod.String(),
			Key:    s.Idempotency,
		}
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
			// Use the function's defaults provided as syntactic sugar when registering functions,
			// only if retries is nil
			funcStep.Retries = s.Retries
		}

		// Always enforce bounds.
		if funcStep.Retries != nil && *funcStep.Retries > consts.MaxRetries {
			max := consts.MaxRetries
			funcStep.Retries = &max
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
