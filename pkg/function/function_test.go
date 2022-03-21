package function

import (
	"testing"

	"github.com/inngest/inngestctl/inngest"
	"github.com/stretchr/testify/require"
)

// TestDerivedConfigDefault asserts that the derived config for simple, default workflows
// is correct.
func TestDerivedConfigDefault(t *testing.T) {
	expr := "event.version >= 2"
	fn := Function{
		Name: "Foo",
		ID:   "magical-id-1",
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

	expectedActionVersion := inngest.ActionVersion{
		Name:   "Foo",
		DSN:    "magical-id-1-action",
		Scopes: []string{"secret:read:*"},
		Runtime: inngest.RuntimeWrapper{
			Runtime: inngest.RuntimeDocker{
				Image: "magical-id-1",
			},
		},
	}

	expectedActionConfig := `package main

import (
	"inngest.com/actions"
)

action: actions.#Action
action: {
  dsn:  "magical-id-1-action"
  name: "Foo"
  scopes: ["secret:read:*"]
  runtime: {
    image: "magical-id-1"
    type:  "docker"
  }
}`

	actions, err := fn.Actions()
	require.NoError(t, err)
	require.Equal(t, 1, len(actions))

	def, err := inngest.FormatAction(actions[0])
	require.NoError(t, err)
	require.EqualValues(t, expectedActionVersion, actions[0])
	require.EqualValues(t, expectedActionConfig, string(def))

	expectedWorkflow := inngest.Workflow{
		ID:   "magical-id-1",
		Name: "Foo",
		Triggers: []inngest.Trigger{
			{
				EventTrigger: &inngest.EventTrigger{
					Event:      "test.event.plz",
					Expression: &expr,
				},
			},
		},
		Actions: []inngest.Action{
			{
				ClientID: 1,
				Name:     expectedActionVersion.Name,
				DSN:      expectedActionVersion.DSN,
			},
		},
		Edges: []inngest.Edge{
			{
				Outgoing: "trigger",
				Incoming: 1,
			},
		},
	}

	expectedWorkflowConfig := `package main

import (
	"inngest.com/workflows"
)

workflow: workflows.#Workflow & {
  id:   "magical-id-1"
  name: "Foo"
  triggers: [{
    event:      "test.event.plz"
    expression: "event.version >= 2"
  }]
  actions: [{
    clientID: 1
    name:     "Foo"
    dsn:      "magical-id-1-action"
  }]
  edges: [{
    outgoing: "trigger"
    incoming: 1
  }]
}`

	wflow, err := fn.Workflow()
	require.NoError(t, err)
	require.NotNil(t, wflow)
	require.EqualValues(t, expectedWorkflow, *wflow)
	def, err = inngest.FormatWorkflow(*wflow)
	require.NoError(t, err)
	require.EqualValues(t, expectedWorkflowConfig, string(def))
}
