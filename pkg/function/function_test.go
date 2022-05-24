package function

import (
	"context"
	"testing"

	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/inngest/state"
	"github.com/inngest/inngest-cli/internal/cuedefs"
	"github.com/stretchr/testify/require"
)

// TestDerivedConfigDefault asserts that the derived config for simple, default workflows
// is correct.
func TestDerivedConfigDefault(t *testing.T) {
	err := state.Clear(context.Background())
	require.NoError(t, err)

	expr := "event.version >= 2"
	fn := Function{
		Name: "Foo",
		ID:   "magical-id",
		Triggers: []Trigger{
			{
				EventTrigger: &EventTrigger{
					Event:      "test.event.plz",
					Expression: &expr,
					Definition: &EventDefinition{
						Format: FormatCue,
						Synced: false,
						Def:    `{ name: string }`,
					},
				},
			},
		},
	}
	path := "/Users/johnny/dev/repo/functions/inngest.json"

	err = fn.canonicalize(context.Background(), path)
	require.NoError(t, err)

	expectedActionVersion := inngest.ActionVersion{
		Name:   "Foo",
		DSN:    "magical-id-step-foo-test",
		Scopes: []string{"secret:read:*"},
		Runtime: inngest.RuntimeWrapper{
			Runtime: inngest.RuntimeDocker{},
		},
	}

	expectedActionConfig := `package main

import (
	"inngest.com/actions"
)

action: actions.#Action
action: {
  dsn:  "magical-id-step-foo-test"
  name: "Foo"
  scopes: ["secret:read:*"]
  runtime: type: "docker"
}`

	actions, edges, err := fn.Actions(context.Background())
	_ = edges
	require.NoError(t, err)
	require.Equal(t, 1, len(actions))

	def, err := cuedefs.FormatAction(actions[0])
	require.NoError(t, err)
	require.EqualValues(t, expectedActionVersion, actions[0])
	require.EqualValues(t, expectedActionConfig, string(def))

	expectedWorkflow := inngest.Workflow{
		ID:   "magical-id",
		Name: "Foo",
		Triggers: []inngest.Trigger{
			{
				EventTrigger: &inngest.EventTrigger{
					Event:      "test.event.plz",
					Expression: &expr,
				},
			},
		},
		Steps: []inngest.Step{
			{
				ClientID: "Foo",
				Name:     expectedActionVersion.Name,
				DSN:      expectedActionVersion.DSN,
			},
		},
		Edges: []inngest.Edge{
			{
				Outgoing: inngest.TriggerName,
				Incoming: "Foo",
			},
		},
	}

	expectedWorkflowConfig := `package main

import (
	"inngest.com/workflows"
)

workflow: workflows.#Workflow & {
  id:   "magical-id"
  name: "Foo"
  triggers: [{
    event:      "test.event.plz"
    expression: "event.version >= 2"
  }]
  actions: [{
    clientID: "Foo"
    name:     "Foo"
    dsn:      "magical-id-step-foo-test"
  }]
  edges: [{
    outgoing: "$trigger"
    incoming: "Foo"
    metadata: {}
  }]
}`

	wflow, err := fn.Workflow(context.Background())
	require.NoError(t, err)
	require.NotNil(t, wflow)
	require.EqualValues(t, expectedWorkflow, *wflow)
	def, err = cuedefs.FormatWorkflow(*wflow)
	require.NoError(t, err)
	require.EqualValues(t, expectedWorkflowConfig, string(def))
}

// TestEventDefinitionAbsolutePath asserts that the event definition file path is not relative
func TestEventDefinitionAbsolutePath(t *testing.T) {
	err := state.Clear(context.Background())
	require.NoError(t, err)

	expr := "event.version >= 2"
	fn := Function{
		Name: "Foo",
		ID:   "relative-id",
		Triggers: []Trigger{
			{
				EventTrigger: &EventTrigger{
					Event:      "event.def.in.file",
					Expression: &expr,
					Definition: &EventDefinition{
						Format: FormatCue,
						Synced: false,
						Def:    "file://./events/event-def-in-file.cue",
					},
				},
			},
		},
	}

	path := "/Users/johnny/dev/repo/functions/inngest.json"

	err = fn.canonicalize(context.Background(), path)
	require.NoError(t, err)

	abs := "file:///Users/johnny/dev/repo/functions/events/event-def-in-file.cue"
	require.EqualValues(t, abs, fn.Triggers[0].EventTrigger.Definition.Def)
}
