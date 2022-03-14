package function

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/gosimple/slug"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/inngest/state"
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

	// Workflow is the in memory representation of the workflow based off the function
	Workflow *inngest.Workflow

	// Actions represents the actions to take for this function.
	Actions []*inngest.ActionVersion
}

// New returns a new, empty function with a randomly generated ID.
func New() (*Function, error) {
	id, err := randomID()
	if err != nil {
		return nil, err
	}
	return &Function{ID: id}, nil
}

func (f Function) Slug() string {
	return slug.Make(f.Name)
}

// Validate returns an error if the function definition is invalid.
func (f *Function) Validate() error {
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
		if terr := t.Validate(); terr != nil {
			err = multierror.Append(err, terr)
		}
	}
	return err
}

// Workflow produces the workflow.cue definition for a function.  Our executor
// runs a "workflow", which is a DAG of the function steps.  Its a subset of
// the function used purely for execution.
func (f *Function) GetWorkflow(s *state.State) (*inngest.Workflow, error) {
	if f.Workflow != nil {
		return f.Workflow, nil
	}

	w := &inngest.Workflow{
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

	// This has references to actions.  Create the actions then reference them
	// from the workflow.
	actions, err := f.GetActions(s)
	if err != nil {
		return nil, err
	}

	for n, a := range actions {
		w.Actions = append(w.Actions, inngest.Action{
			ClientID: uint(n) + 1, // 0 is the trigger; use 1 offset
			Name:     a.Name,
			DSN:      a.DSN,
		})
	}

	// TODO: When supporting > 1 step, the function must define edges itself.
	// Right now we assume a single action (all that's supported) and build an
	// edge from the trigger to the action.
	w.Edges = []inngest.Edge{{
		Outgoing: "trigger",
		Incoming: 1,
	}}

	f.Workflow = w

	return w, nil
}

// Actions produces configuration for each step of the function.  Each config
// file specifies how to run the code.
func (f *Function) GetActions(s *state.State) ([]*inngest.ActionVersion, error) {
	if f.Actions != nil {
		return f.Actions, nil
	}
	// XXX: In the very near future we'll adapt this function package to
	// support step functions in the same way that a workflow does.  This
	// means that we have to support returning many actions.

	// This has no defined actions, which means its an implicit
	// single action invocation.  We assume that a Dockerfile
	// exists in the project root, and that we can build the
	// image which contains all of the code necessary to run
	// the function.
	actions, err := f.defaultAction(s)
	if err != nil {
		return nil, err
	}
	f.Actions = actions
	return actions, nil
}

func (f *Function) defaultAction(s *state.State) ([]*inngest.ActionVersion, error) {
	prefix := s.Account.Identifier.DSNPrefix
	if s.Account.Identifier.Domain != nil {
		prefix = *s.Account.Identifier.Domain
	}

	var sb strings.Builder
	sb.WriteString(prefix)
	sb.WriteString("/")
	sb.WriteString(f.ID)
	sb.WriteString("-action")

	a := &inngest.ActionVersion{
		Name: f.Name,
		DSN:  sb.String(),
		// This is a custom action, so allow reading any secret.
		Scopes: []string{"secret:read:*"},
		Runtime: inngest.RuntimeWrapper{
			Runtime: inngest.RuntimeDocker{
				Image: f.ID,
			},
		},
		Version: &inngest.VersionInfo{},
	}

	err := populateActionVersion(context.Background(), s, a)
	if err != nil {
		return nil, err
	}

	return []*inngest.ActionVersion{a}, nil
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

// populateActionVersion attempts to pull an existing action from Inngest and if it exists
// populates the version info. We currently hardcode scopes which is the only time a major version is required.
func populateActionVersion(ctx context.Context, state *state.State, action *inngest.ActionVersion) error {
	fmt.Println(state.Client)
	a, err := state.Client.Action(ctx, action.DSN)
	if err != nil {
		fmt.Println(err)
		if err.Error() == "action not found" {
			// Set default values
			action.Version.Major = 1
			action.Version.Minor = 1
			return nil
		}
		return err
	}

	action.Version.Major = a.Latest.VersionMajor
	action.Version.Minor = a.Latest.VersionMinor

	fmt.Println(action.Version.Major, action.Version.Minor)
	return nil
}
