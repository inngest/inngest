package function

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/google/uuid"
	"github.com/gosimple/slug"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/inngest/clistate"
	"github.com/inngest/inngest-cli/pkg/expressions"
)

var (
	// pathCtxKey stores the function path within context,
	// necessary for validation.
	pathCtxKey = struct{}{}
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
	// Name is the descriptive name for the function
	Name string `json:"name"`

	// ID is the immutable random ID for the function.
	ID string `json:"id"`

	// Trigger represnets the trigger for the function.
	Triggers []Trigger `json:"triggers"`

	// Idempotency allows the specification of an idempotency key by templating event
	// data, eg:
	//
	//  `{{ event.data.order_id }}`.
	//
	// When specified, a function will run at most once per 24 hours for the given unique
	// key.
	Idempotency *string `json:"idempotency,omitempty"`

	// Throttle allows specifying custom throttling for the function.
	Throttle *inngest.Throttle `json:"throttle,omitempty"`

	// Actions represents the actions to take for this function.  If empty, this assumes
	// that we have a single action specified in the current directory using
	Steps map[string]Step `json:"steps,omitempty"`

	// dir is an internal field which maps the root directory for the function
	dir string
}

// Step represents a single unit of code (action) which runs as part of a step function, in a DAG.
type Step struct {
	ID      string                 `json:"id"`
	Path    string                 `json:"path"`
	Name    string                 `json:"name"`
	Runtime inngest.RuntimeWrapper `json:"runtime"`
	After   []After                `json:"after,omitempty"`
}

type After struct {
	// Step represents the step name to run after.
	Step string `json:"step,omitempty"`

	// Wait represents a duration that we should wait before continuing with
	// this step, eg. "24h" or "1h30m".
	Wait *string `json:"wait,omitempty"`

	If string `json:"if,omitempty"`

	// Async, when specified, indicates that we must wait for another event
	// to be received before continuing with this step.  Note that we may
	// specify expressions within an Async block to only continue with specific
	// event data.
	Async *inngest.AsyncEdgeMetadata `json:"async,omitempty"`

	// TODO: support multiple steps all finishing prior to running this once.
	// Steps []string `json:"steps,omitempty"`
}

// New returns a new, empty function with a randomly generated ID.
func New() (*Function, error) {
	id, err := randomID()
	if err != nil {
		return nil, err
	}
	return &Function{
		ID:    id,
		Steps: map[string]Step{},
	}, nil
}

func (f Function) Dir() string {
	return f.dir
}

func (f Function) Slug() string {
	return strings.ToLower(slug.Make(f.Name))
}

// MarshalCUE formats a function into canonical cue configuration.
func (f Function) MarshalCUE() ([]byte, error) {
	return formatCue(f)
}

// Validate returns an error if the function definition is invalid.
func (f Function) Validate(ctx context.Context) error {
	// Store the fn path in context for validating triggers.
	ctx = context.WithValue(ctx, pathCtxKey, f.dir)

	var err error
	if f.ID == "" {
		err = multierror.Append(err, fmt.Errorf("A function ID is required"))
	}
	if f.Name == "" {
		err = multierror.Append(err, fmt.Errorf("A function name is required"))
	}
	if len(f.Triggers) == 0 {
		err = multierror.Append(err, fmt.Errorf("At least one trigger is required"))
	}
	for _, t := range f.Triggers {
		if terr := t.Validate(ctx); terr != nil {
			err = multierror.Append(err, terr)
		}
	}

	for k, step := range f.Steps {
		if k == "" || step.ID == "" {
			return fmt.Errorf("A step must have an ID defined")
		}
	}

	_, edges, aerr := f.Actions(ctx)
	if aerr != nil {
		err = multierror.Append(err, aerr)
		return err
	}

	// Validate edges exist.
	for _, edge := range edges {
		_, incoming := f.Steps[edge.Incoming]
		_, outgoing := f.Steps[edge.Outgoing]
		if edge.Outgoing != inngest.TriggerName && !outgoing {
			err = multierror.Append(err, fmt.Errorf("unknown step '%s' for edge '%v'", edge.Outgoing, edge))
		}
		if !incoming {
			err = multierror.Append(err, fmt.Errorf("unknown step '%s' for edge '%v'", edge.Incoming, edge))
		}

		// Ensure that any expressions are also valid.
		if edge.Metadata != nil && edge.Metadata.If != "" {
			if _, verr := expressions.NewExpressionEvaluator(ctx, edge.Metadata.If); verr != nil {
				err = multierror.Append(err, verr)
			}
		}
	}

	return err
}

// Workflow produces the workflow.cue definition for a function.  Our executor
// runs a "workflow", which is a DAG of the function steps.  Its a subset of
// the function used purely for execution.
func (f Function) Workflow(ctx context.Context) (*inngest.Workflow, error) {
	w := inngest.Workflow{
		Name:     f.Name,
		ID:       f.ID,
		Triggers: make([]inngest.Trigger, len(f.Triggers)),
	}

	for n, t := range f.Triggers {
		if t.EventTrigger != nil {
			w.Triggers[n].EventTrigger = &inngest.EventTrigger{
				Event:      t.EventTrigger.Event,
				Expression: t.EventTrigger.Expression,
			}
			continue
		}
		if t.CronTrigger != nil {
			w.Triggers[n].CronTrigger = &inngest.CronTrigger{
				Cron: t.CronTrigger.Cron,
			}
			continue
		}
		return nil, fmt.Errorf("unknown trigger type")
	}

	if f.Throttle != nil {
		w.Throttle = f.Throttle
	}

	if f.Idempotency != nil {
		w.Throttle = &inngest.Throttle{
			Key:    f.Idempotency,
			Count:  1,
			Period: "24h",
		}
	}

	// This has references to actions.  Create the actions then reference them
	// from the workflow.
	versions, edges, err := f.Actions(ctx)
	if err != nil {
		return nil, err
	}

	for n, a := range versions {
		// TODO: remove this n^n loop with a refactoring of how we consider
		// actions to be defined within a workflow, plus data type changes.
		var found Step
		for _, s := range f.Steps {
			if s.DSN(ctx, f) == a.DSN {
				found = s
				break
			}
		}

		w.Steps = append(w.Steps, inngest.Step{
			ClientID: uint(n) + 1,
			ID:       found.ID,
			Name:     a.Name,
			DSN:      a.DSN,
		})
	}

	w.Edges = edges

	return &w, nil
}

// Actions produces configuration for each step of the function.  Each config
// file specifies how to run the code.
func (f Function) Actions(ctx context.Context) ([]inngest.ActionVersion, []inngest.Edge, error) {
	// This has no defined actions, which means its an implicit
	// single action invocation.  We assume that a Dockerfile
	// exists in the project root, and that we can build the
	// image which contains all of the code necessary to run
	// the function.
	if len(f.Steps) == 0 {
		return nil, nil, fmt.Errorf("This function has no steps")
	}

	avs := []inngest.ActionVersion{}
	edges := []inngest.Edge{}

	for _, s := range f.Steps {
		step := s
		av, err := f.action(ctx, step)
		if err != nil {
			return nil, nil, err
		}
		avs = append(avs, av)

		// We support barebones function definitions with a single step.  Any time
		// a single step is specified without an After block, it's ran automatically
		// from the trigger.
		if len(step.After) == 0 {
			edges = append(edges, inngest.Edge{
				Outgoing: inngest.TriggerName,
				Incoming: step.ID,
			})
			continue
		}

		// For each of the "after" items, add an edge.
		for _, after := range step.After {
			var metadata *inngest.EdgeMetadata
			if after.Async != nil || after.Wait != nil || after.If != "" {
				metadata = &inngest.EdgeMetadata{
					If:                after.If,
					Wait:              after.Wait,
					AsyncEdgeMetadata: after.Async,
				}
			}

			edges = append(edges, inngest.Edge{
				Outgoing: after.Step,
				Incoming: step.ID,
				Metadata: metadata,
			})
		}
	}

	// Ensure that the actions and edges are sorted by name, giving us
	// deterministic output.
	sort.SliceStable(avs, func(i, j int) bool {
		return avs[i].DSN < avs[j].DSN
	})
	sort.SliceStable(edges, func(i, j int) bool {
		return edges[i].Outgoing < edges[j].Outgoing
	})

	return avs, edges, nil
}

func (f Function) action(ctx context.Context, s Step) (inngest.ActionVersion, error) {
	id := s.DSN(ctx, f)

	a := inngest.ActionVersion{
		Name:    s.Name,
		DSN:     id,
		Runtime: s.Runtime,
	}
	if s.Runtime.Runtime == nil {
		return a, fmt.Errorf("no runtime specified")
	}
	if s.Runtime.RuntimeType() != "http" {
		// Non-HTTP actions can read secrets;  http actions are external APIs and so
		// don't need secret access.
		a.Scopes = []string{"secret:read:*"}
	}
	return a, nil
}

func (f *Function) canonicalize(ctx context.Context, path string) error {
	f.dir = path
	// dir should point to the dir, not the file.
	if strings.HasSuffix(path, jsonConfigName) || strings.HasSuffix(path, cueConfigName) {
		f.dir = filepath.Dir(path)
	}

	if f.Idempotency != nil {
		// Replace the throttle field with idempotency.
		f.Throttle = &inngest.Throttle{
			Key:    f.Idempotency,
			Count:  1,
			Period: "24h",
		}
	}

	if len(f.Steps) == 0 {
		// Create the default action used when no steps are specified.
		// This assumes that we're writing a single step function using
		// custom code with the docker executor, and that the code is
		// in the current directory.
		f.Steps = map[string]Step{}
		f.Steps[DefaultStepName] = Step{
			ID:   DefaultStepName,
			Name: f.Name,
			Path: "file://.",
			Runtime: inngest.RuntimeWrapper{
				Runtime: inngest.RuntimeDocker{},
			},
			After: []After{
				{
					Step: inngest.TriggerName,
				},
			},
		}
	}

	return nil
}

func (s Step) DSN(ctx context.Context, f Function) string {
	suffix := "test"
	if clistate.IsProd() {
		suffix = "prod"
	}

	slug := strings.ToLower(slug.Make(s.ID))

	id := fmt.Sprintf("%s-step-%s-%s", f.ID, slug, suffix)
	if prefix, err := clistate.AccountIdentifier(ctx); err == nil && prefix != "" {
		id = fmt.Sprintf("%s/%s", prefix, id)
	}

	return id
}

// DeterministicUUID returns a deterministic V3 UUID based off of the SHA1
// hash of the function's ID.
func DeterministicUUID(f Function) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(f.ID))
}

func randomID() (string, error) {
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
