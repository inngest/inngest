package fn

import "net/url"

func GetFnSyncConfig(fn ServableFunction) *SyncConfig {
	config := fn.Config()

	return &SyncConfig{
		Name:        fn.Name(),
		Slug:        fn.FullyQualifiedID(),
		Triggers:    fn.Trigger().Triggers(),
		Concurrency: config.Concurrency,
		Priority:    config.Priority,
		EventBatch:  config.BatchEvents,
		Idempotency: config.Idempotency,
		RateLimit:   config.RateLimit,
		Throttle:    config.Throttle,
		Debounce:    config.Debounce,
		Timeouts:    config.Timeouts,
		Cancel:      config.Cancel,
		Retries:     config.Retries,
		Singleton:   config.Singleton,
	}
}

// SyncConfig represents an github.com/inngest/inngest/pkg/sdk.SDKFunction
// type used to sync functions.
type SyncConfig struct {
	Name string `json:"name"`

	// ID is the function slug.
	Slug string `json:"id"`

	// Triggers represent the triggers which start this function.
	Triggers []Trigger `json:"triggers"`

	// Concurrency allows limiting the concurrency of running functions, optionally constrained
	// by an individual concurrency key.
	//
	// This may be an int OR a struct, for backwards compatibility.
	Concurrency []Concurrency `json:"concurrency,omitempty"`

	// Priority represents the priority information for this function.
	Priority *Priority `json:"priority,omitempty"`

	// EventBatch determines how a function will process a list of incoming events
	EventBatch *EventBatchConfig `json:"batchEvents,omitempty"`

	// Idempotency allows the specification of an idempotency key by templating event
	// data, eg:
	//
	//  `event.data.order_id`.
	//
	// When specified, a function will run at most once per 24 hours for the given unique
	// key.
	Idempotency *string `json:"idempotency,omitempty"`

	// RateLimit allows specifying custom rate limiting for the function.
	RateLimit *RateLimit `json:"rateLimit,omitempty"`

	// Throttle represents a soft rate limit for gating function starts.  Any function runs
	// over the throttle period will be enqueued in the backlog to run at the next available
	// time.
	Throttle *Throttle `json:"throttle,omitempty"`

	// Retries allows specifying the number of retries to attempt across all steps in the
	// function.
	Retries *int `json:"retries,omitempty"`

	Debounce *Debounce `json:"debounce,omitempty"`

	Timeouts *Timeouts `json:"timeouts,omitempty"`

	// Cancel specifies cancellation signals for the function
	Cancel []Cancel `json:"cancel,omitempty"`

	// Singleton represents a mechanism to ensure that only one instance of a function
	// runs at a time for a given key. Additional invocations with the same key will either
	// be ignored or cause the current instance to be canceled and replaced, depending on
	// the specified mode.
	Singleton *Singleton

	Steps map[string]SDKStep `json:"steps"`
}

func (c *SyncConfig) UpdateSteps(endpoint url.URL) {
	// Modify URL to contain fn ID, step params
	values := endpoint.Query()
	values.Set("fnId", c.Slug) // This should match the Slug below
	values.Set("step", "step")
	endpoint.RawQuery = values.Encode()

	var r *StepRetries
	if c.Retries != nil {
		r = &StepRetries{
			Attempts: *c.Retries,
		}
	}

	c.Steps = map[string]SDKStep{
		"step": {
			ID:      "step",
			Name:    c.Name,
			Retries: r,
			Runtime: map[string]any{
				"url": endpoint.String(),
			},
		},
	}
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
