package function

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/google/uuid"
	"github.com/gosimple/slug"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/xhit/go-str2duration/v2"
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

	Concurrency int `json:"concurrency"`

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

	// Cancel specifies cancellation signals for the function
	Cancel []Cancel `json:"cancel,omitempty"`

	// dir is an internal field which maps the root directory for the function
	dir string
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

// Cancel represents a cancellation signal for a function.  When specified, this
// will set up pauses which automatically cancel the function based off of matching
// events and expressions.
type Cancel struct {
	Event   string  `json:"event"`
	Timeout *string `json:"timeout"`
	If      *string `json:"if"`
}

// New returns a new, empty function with a randomly generated ID.
func New() (*Function, error) {
	id, err := RandomID()
	if err != nil {
		return nil, err
	}
	return &Function{
		ID:    id,
		Steps: map[string]Step{},
	}, nil
}

// Dir returns the absolute directory that this function is written to
func (f Function) Dir() string {
	return f.dir
}

// Slug returns the function slug
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

	id := f.ID
	if strings.Contains(id, "/") {
		id = strings.Split(id, "/")[1]
	}
	if slug.Make(id) != id {
		err = multierror.Append(err, fmt.Errorf("A function ID must contain lowercase letters, numbers, and dashes only (eg. 'my-greatest-function-ef81b2')"))
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

		id := step.ID
		if strings.Contains(id, "/") {
			id = strings.Split(id, "/")[1]
		}
		if slug.Make(id) != id {
			err = multierror.Append(err, fmt.Errorf("A step ID must contain lowercase letters, numbers, and dashes only (eg. 'my-greatest-function-ef81b2')"))
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

	return err
}

// Workflow produces the workflow.cue definition for a function.  Our executor
// runs a "workflow", which is a DAG of the function steps.  Its a subset of
// the function used purely for execution.
func (f Function) Workflow(ctx context.Context) (*inngest.Workflow, error) {
	w := inngest.Workflow{
		Name:        f.Name,
		ID:          f.ID,
		Triggers:    make([]inngest.Trigger, len(f.Triggers)),
		Concurrency: f.Concurrency,
	}

	// TODO: Refactor these into shared structs and definitions, extend.
	// This is really ugly, and is a symptom of functions coming after
	// workflows and not being truly first class in the executor.

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

	for _, c := range f.Cancel {
		w.Cancel = append(w.Cancel, inngest.Cancel{
			Event:   c.Event,
			Timeout: c.Timeout,
			If:      c.If,
		})
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

		step := inngest.Step{
			ClientID: uint(n) + 1,
			ID:       found.ID,
			Name:     a.Name,
			DSN:      a.DSN,
			Retries:  a.Retries,
		}

		if a.Version != nil {
			step.Version = &inngest.VersionConstraint{
				Major: &a.Version.Major,
				Minor: &a.Version.Minor,
			}
		}

		w.Steps = append(w.Steps, step)
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

	if s.Runtime == nil {
		s.Runtime = &defaultRuntime
	}

	a := inngest.ActionVersion{
		Name:    s.Name,
		DSN:     id,
		Runtime: *s.Runtime,
		Retries: s.Retries,
	}

	if s.Version != nil {
		a.Version = &inngest.VersionInfo{
			Major: *s.Version.Major,
			Minor: *s.Version.Minor,
		}
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
	if strings.HasSuffix(path, JsonConfigName) || strings.HasSuffix(path, CueConfigName) {
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
		var majorVersion uint = 1
		var minorVersion uint = 1

		f.Steps = map[string]Step{}
		f.Steps[DefaultStepName] = Step{
			ID:      DefaultStepName,
			Name:    f.Name,
			Path:    "file://.",
			Runtime: &defaultRuntime,
			After: []After{
				{
					Step: inngest.TriggerName,
				},
			},
			Version: &inngest.VersionConstraint{
				Major: &majorVersion,
				Minor: &minorVersion,
			},
		}
	}

	for n, s := range f.Steps {
		if s.Runtime == nil {
			s.Runtime = DefaultRuntime()
			f.Steps[n] = s
		}

		version, err := f.action(ctx, s)
		if err != nil {
			return err
		}

		if version.Version != nil {
			s.Version = &inngest.VersionConstraint{
				Major: &version.Version.Major,
				Minor: &version.Version.Minor,
			}

			f.Steps[n] = s
		}

		if len(s.After) == 0 {
			s.After = []After{
				{
					Step: inngest.TriggerName,
				},
			}
			f.Steps[n] = s
		}
	}

	return nil
}

// WriteToDisk writes the function and associated event schemas to disk.
//
// When we read a function's `inngest` configuration file, we parse the
// event schemas from the triggers and add the definitions to the parsed
// struct.
//
// This function creates an interim function definition file which has
// simplified definitions for writing a consistent, small configuration
// file.
//
// In essence, this performs the _opposite_ of canonicalization: instead
// of adding defaults we remove them so that defaults aren't included
// within the JSON file.
func (f Function) WriteToDisk(ctx context.Context) error {
	// For new functions, dir might be empty.
	if f.dir == "" {
		dirname := f.Slug()
		relative := "./" + dirname
		f.dir, _ = filepath.Abs(relative)
	}

	if err := f.writeTriggersToDisk(ctx); err != nil {
		return err
	}

	// If Idempotency is set we don't need a throttle;  it's implied, and
	// we can remove this from the config.
	if f.Idempotency != nil {
		f.Throttle = nil
	}

	for n, s := range f.Steps {
		// If this is a basic docker runtime we can omit it.
		if s.Runtime != nil {
			ok := reflect.DeepEqual(*s.Runtime, defaultRuntime)
			if ok {
				// Remove this from the step.
				s.Runtime = nil
				f.Steps[n] = s
			}
		}

		// If this step is "after" and it has no metadata, name, expression, etc.
		// we can also omit this.
		if len(s.After) == 1 {
			ok := reflect.DeepEqual(s.After[0], defaultAfter)
			if ok {
				// Remove this from the step.
				s.After = nil
				f.Steps[n] = s
			}
		}
	}

	byt, err := MarshalJSON(f)
	if err != nil {
		return fmt.Errorf("Step '%s' already exists in this function", f.ID)
	}

	if err := os.WriteFile(filepath.Join(f.Dir(), "inngest.json"), byt, 0644); err != nil {
		return fmt.Errorf("Step '%s' already exists in this function", f.ID)
	}

	return nil
}

func (f Function) writeTriggersToDisk(ctx context.Context) error {
	if err := upsertDir(filepath.Join(f.dir, "events")); err != nil {
		return fmt.Errorf("error making events directory: %w", err)
	}

	// For each event within the function create a new event file.
	for n, trigger := range f.Triggers {
		if trigger.EventTrigger == nil {
			continue
		}

		if trigger.EventTrigger.Definition == nil || trigger.EventTrigger.Definition.Def == "" {
			// Use an empty event format.
			trigger.EventTrigger.Definition = &EventDefinition{
				Format: FormatCue,
				Synced: false,
				Def:    fmt.Sprintf(evtDefinition, strconv.Quote(trigger.Event)),
			}
		}

		cue, err := trigger.Definition.Cue(ctx)
		if err != nil {
			// XXX: We would like to log this as a warning.
			continue
		}

		name := fmt.Sprintf("%s.cue", slug.Make(trigger.Event))
		path := filepath.Join(f.dir, "events", name)
		if err := os.WriteFile(path, []byte(cue), 0644); err != nil {
			return fmt.Errorf("error writing event definition: %w", err)
		}
		f.Triggers[n].Definition.Def = fmt.Sprintf("file://./events/%s", name)
	}
	return nil
}

func upsertDir(path string) error {
	if exists(path) {
		return nil
	}
	return os.MkdirAll(path, 0755)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// DeterministicUUID returns a deterministic V3 UUID based off of the SHA1
// hash of the function's ID.
func DeterministicUUID(f Function) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(f.ID))
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
