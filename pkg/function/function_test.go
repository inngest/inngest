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
		Triggers: []Trigger{
			{
				EventTrigger: &EventTrigger{
					Event:      "test.event.plz",
					Expression: &expr,
					Definition: &EventDefinition{
						Format: FormatCue,
						Synced: false,
						Def:    []byte(`{ name: string }`),
					},
				},
			},
		},
	}

	expectedActionVersion := inngest.ActionVersion{
		Name:   "Foo",
		DSN:    "foo-action",
		Scopes: []string{"secret:read:*"},
		Runtime: inngest.RuntimeWrapper{
			Runtime: inngest.RuntimeDocker{
				Image: "foo",
			},
		},
	}

	expectedActionConfig := `package main

import (
	"inngest.com/actions"
)

action: actions.#Action
action: {
  dsn:  "foo-action"
  name: "Foo"
  scopes: ["secret:read:*"]
  runtime: {
    image: "foo"
    type:  "docker"
  }
}`

	actions, err := fn.Actions()
	require.NoError(t, err)
	require.Equal(t, 1, len(actions))
	require.EqualValues(t, expectedActionVersion, actions[0].definition)
	require.EqualValues(t, expectedActionConfig, string(actions[0].config))

	expectedWorkflow := inngest.Workflow{
		ID:   "foo",
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
  id:   "foo"
  name: "Foo"
  triggers: [{
    event:      "test.event.plz"
    expression: "event.version >= 2"
  }]
  actions: [{
    clientID: 1
    name:     "Foo"
    dsn:      "foo-action"
  }]
  edges: [{
    outgoing: "trigger"
    incoming: 1
  }]
}`

	wflow, err := fn.Workflow()
	require.NoError(t, err)
	require.EqualValues(t, expectedWorkflow, wflow.definition)
	require.EqualValues(t, expectedWorkflowConfig, string(wflow.config))
}
