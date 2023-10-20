package inngest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/google/uuid"
	"github.com/gosimple/slug"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/xhit/go-str2duration/v2"
)

const (
	DefaultStepName = "step-1"
)

// Function represents a step function which is triggered whenever an event
// is received or on a schedule.  In essence, it contains:
//
// - Triggers, which represent when a function is invoked
//
// - Steps, which represent the individual steps of actions that the function calls.
//
// A function may be simple (ie. only having a single step) or complex (ie. many
// steps).  Simple functions are easy:  run the single step's action.  Complex functions
// represent steps as a DAG, with edges between the trigger and each step.
type Function struct {
	// ConfigVersion represents the configuration version.  This lets us add or change
	// JSON definitions within functions when unmarshalling.
	ConfigVersion int `json:"cv,omitempty"`

	// ID is an internal surrogate key representing this function.
	ID uuid.UUID `json:"id"`

	// FunctionVersion represents the version of this specific function.  The same
	// function ID may be updated many times over the lifetime of a function; this
	// represents the specific version for the functon ID.
	FunctionVersion int `json:"fv"`

	// Name is the descriptive name for the function
	Name string `json:"name"`

	// Slug is the human-friendly ID for the function
	Slug string `json:"slug"`

	Priority *Priority `json:"priority,omitempty"`

	// ConcurrencyLimits allows limiting the concurrency of running functions, optionally constrained
	// by individual concurrency keys.
	//
	// Users may specify up to 2 concurrency keys.
	Concurrency *ConcurrencyLimits `json:"concurrency,omitempty"`

	Debounce *Debounce `json:"debounce,omitempty"`

	// Trigger represnets the trigger for the function.
	Triggers []Trigger `json:"triggers"`

	// EventBatch determines how the function will process a list of incoming events
	EventBatch *EventBatchConfig `json:"batchEvents,omitempty"`

	// RateLimit allows specifying custom rate limiting for the function.
	RateLimit *RateLimit `json:"rateLimit,omitempty"`

	// Cancel specifies cancellation signals for the function
	Cancel []Cancel `json:"cancel,omitempty"`

	// Actions represents the actions to take for this function.  If empty, this assumes
	// that we have a single action specified in the current directory using
	Steps []Step `json:"steps,omitempty"`

	// Edges represent edges between steps in the dag.
	Edges []Edge `json:"edges,omitempty"`
}

type Priority struct {
	Run *string `json:"run"`
}

type Debounce struct {
	Key    *string `json:"key"`
	Period string  `json:"period"`
}

// Cancel represents a cancellation signal for a function.  When specified, this
// will set up pauses which automatically cancel the function based off of matching
// events and expressions.
type Cancel struct {
	Event   string  `json:"event"`
	Timeout *string `json:"timeout,omitempty"`
	If      *string `json:"if,omitempty"`
}

// GetSlug returns the function slug, defaulting to creating a slug of the function name.
func (f Function) GetSlug() string {
	if f.Slug != "" {
		return f.Slug
	}
	return strings.ToLower(slug.Make(f.Name))
}

func (f Function) IsScheduled() bool {
	for _, t := range f.Triggers {
		if t.CronTrigger != nil {
			return true
		}
	}
	return false
}

// Validate returns an error if the function definition is invalid.
func (f Function) Validate(ctx context.Context) error {
	var err error
	if f.Name == "" {
		err = multierror.Append(err, fmt.Errorf("A function name is required"))
	}
	if len(f.Triggers) == 0 {
		err = multierror.Append(err, fmt.Errorf("At least one trigger is required"))
	}

	if f.Concurrency != nil {
		if cerr := f.Concurrency.Validate(ctx); cerr != nil {
			err = multierror.Append(err, cerr)
		}
	}

	for _, t := range f.Triggers {
		if terr := t.Validate(ctx); terr != nil {
			err = multierror.Append(err, terr)
		}
	}

	if f.EventBatch != nil {
		if berr := f.EventBatch.IsValid(); berr != nil {
			err = multierror.Append(err, berr)
		}
	}

	for _, step := range f.Steps {
		if step.Name == "" {
			err = multierror.Append(err, fmt.Errorf("All steps must have a name"))
		}
		uri, serr := url.Parse(step.URI)
		if serr != nil {
			err = multierror.Append(err, fmt.Errorf("Steps must have a valid URI"))
		}
		switch uri.Scheme {
		case "http", "https":
			continue
		default:
			err = multierror.Append(err, fmt.Errorf("Non-HTTP steps are not yet supported"))
		}
	}

	edges, aerr := f.AllEdges(ctx)
	if aerr != nil {
		return multierror.Append(err, aerr)
	}

	if len(f.Steps) != 1 {
		err = multierror.Append(err, fmt.Errorf("Functions must contain one step"))
	}

	// Validate edges.
	for _, edge := range edges {
		// Ensure that any expressions are also valid.
		if edge.Metadata == nil {
			continue
		}
		if edge.Metadata.If != "" {
			if _, verr := expressions.NewExpressionEvaluator(ctx, edge.Metadata.If); verr != nil {
				err = multierror.Append(err, verr)
			}
		}
		if edge.Metadata.Wait != nil {
			// Ensure that this is a valid duration or expression.
			if _, err := str2duration.ParseDuration(*edge.Metadata.Wait); err == nil {
				continue
			}
			if _, err := expressions.NewExpressionEvaluator(ctx, *edge.Metadata.Wait); err == nil {
				continue
			}
			err = multierror.Append(err, fmt.Errorf("Unable to parse wait as a duration or expression: %s", *edge.Metadata.Wait))
		}
	}

	// Validate priority expression
	if f.Priority != nil && f.Priority.Run != nil {
		if _, exprErr := expressions.NewExpressionEvaluator(ctx, *f.Priority.Run); exprErr != nil {
			err = multierror.Append(err, fmt.Errorf("Priority.Run expression is invalid: %s", exprErr))
		}
		// NOTE: Priority.Run is not valid when batch is enabled.
		if f.EventBatch != nil {
			err = multierror.Append(err, fmt.Errorf("A function cannot specify Priority.Run and Batch together"))
		}
	}

	// Validate cancellation expressions
	for _, c := range f.Cancel {
		if c.If != nil {
			if _, exprErr := expressions.NewExpressionEvaluator(ctx, *c.If); exprErr != nil {
				err = multierror.Append(err, fmt.Errorf("Cancellation expression is invalid: %s", exprErr))
			}
		}
	}

	if len(f.Cancel) > consts.MaxCancellations {
		err = multierror.Append(err, fmt.Errorf("This function exceeds the max number of cancellation events: %d", consts.MaxCancellations))
	}

	if f.Debounce != nil && f.Debounce.Key != nil {
		if _, exprErr := expressions.NewExpressionEvaluator(ctx, *f.Debounce.Key); exprErr != nil {
			err = multierror.Append(err, fmt.Errorf("Debounce expression is invalid: %s", exprErr))
		}
		// NOTE: Debounce is not valid when batch is enabled.
		if f.EventBatch != nil {
			err = multierror.Append(err, fmt.Errorf("A function cannot specify batch and debounce"))
		}
		period, perr := str2duration.ParseDuration(f.Debounce.Period)
		if perr != nil {
			err = multierror.Append(err, fmt.Errorf("The debounce period of '%s' is invalid: %w", f.Debounce.Period, perr))
		}
		if period < consts.MinDebouncePeriod {
			err = multierror.Append(err, fmt.Errorf("The debounce period of '%s' is less than the min of: %s", f.Debounce.Period, consts.MinDebouncePeriod))
		}
		if period > consts.MaxDebouncePeriod {
			err = multierror.Append(err, fmt.Errorf("The debounce period of '%s' is greater than the max of: %s", f.Debounce.Period, consts.MaxDebouncePeriod))
		}
	}

	// Validate rate limit expression
	if f.RateLimit != nil && f.RateLimit.Key != nil {
		if _, exprErr := expressions.NewExpressionEvaluator(ctx, *f.RateLimit.Key); exprErr != nil {
			err = multierror.Append(err, fmt.Errorf("Rate limit expression is invalid: %s", exprErr))
		}
	}

	return err
}

// RunPriorityFactor returns the run priority factor for this function, given an input event.
func (f Function) RunPriorityFactor(ctx context.Context, event map[string]any) (int64, error) {
	if f.Priority == nil || f.Priority.Run == nil {
		return 0, nil
	}

	expr, err := expressions.NewExpressionEvaluator(ctx, *f.Priority.Run)
	if err != nil {
		return 0, fmt.Errorf("Priority.Run expression is invalid: %s", err)
	}

	val, _, err := expr.Evaluate(ctx, expressions.NewData(map[string]any{"event": event}))
	if err != nil {
		return 0, fmt.Errorf("Priority.Run expression errored: %s", err)
	}

	var result int64

	switch v := val.(type) {
	case int:
		result = int64(v)
	case int64:
		result = v
	default:
		return 0, fmt.Errorf("Priority.Run expression returned non-int: %v", val)
	}

	// Apply bounds
	if result > consts.PriorityFactorMax {
		return consts.PriorityFactorMax, nil
	}
	if result < consts.PriorityFactorMin {
		return consts.PriorityFactorMin, nil
	}

	return result, nil
}

// URI returns the function's URI.  It is expected that the function has already been
// validated.
func (f Function) URI() (*url.URL, error) {
	if len(f.Steps) >= 1 {
		return url.Parse(f.Steps[0].URI)
	}
	return nil, fmt.Errorf("No steps configured")
}

// AllEdges produces edge configuration for steps defined within the function.
// If no edges for a step exists, an automatic step from the tirgger is added.
func (f Function) AllEdges(ctx context.Context) ([]Edge, error) {
	edges := []Edge{}

	// O1 lookup of steps.
	stepmap := map[string]Step{}
	// Track whether incoming edges exist for each step
	seen := map[string]bool{}
	for _, s := range f.Steps {
		stepmap[s.ID] = s
		seen[s.ID] = false
	}

	var err error

	// Map all edges for incoming steps.
	for _, edge := range f.Edges {
		if _, ok := seen[edge.Incoming]; !ok {
			err = multierror.Append(
				err,
				fmt.Errorf("Step '%s' doesn't exist within edge", edge.Incoming),
			)
			continue
		}
		seen[edge.Incoming] = true
		edges = append(edges, edge)
	}

	// For all unseen edges, add a trigger edge.
	for step, ok := range seen {
		if ok {
			continue
		}
		edges = append(edges, Edge{
			Outgoing: TriggerName,
			Incoming: step,
		})
	}

	// Ensure that the edges are sorted by name, giving us
	// deterministic output.
	sort.SliceStable(edges, func(i, j int) bool {
		return edges[i].Outgoing < edges[j].Outgoing
	})
	return edges, nil
}

// DeterministicUUID returns a deterministic V3 UUID based off of the SHA1
// hash of the function's name.
func DeterministicUUID(f Function) uuid.UUID {
	str := f.Name + f.Steps[0].URI
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(str))
}

func RandomID() (string, error) {
	// Generate a 6 character long hex string.  This is the suffix to
	// our DSN prefix, which decreases the chance of collosion by 1/16,777,216.
	// This makes the total chance of collisions from an _empty_ keyspace
	// 1 in 3,435,034,312,704 (we'll ignore birthday problems).
	byt := make([]byte, 3)
	if _, err := rand.Read(byt); err != nil {
		return "", fmt.Errorf("error generating ID: %w", err)
	}
	petname.NonDeterministicMode()
	return fmt.Sprintf("%s-%s", petname.Generate(2, "-"), hex.EncodeToString(byt)), nil
}
