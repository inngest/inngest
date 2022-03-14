package function

import (
	"testing"

	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/inngest/client"
	"github.com/inngest/inngestctl/inngest/state"
	"github.com/inngest/inngestctl/test/mocks"
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

	expectedActionVersion := &inngest.ActionVersion{
		Name:   "Foo",
		DSN:    "coolprefix/magical-id-1-action",
		Scopes: []string{"secret:read:*"},
		Runtime: inngest.RuntimeWrapper{
			Runtime: inngest.RuntimeDocker{
				Image: "magical-id-1",
			},
		},
		Version: &inngest.VersionInfo{
			Major: 1,
			Minor: 1,
		},
	}

	s := &state.State{
		Client: mocks.NewMockClient(),
		Account: client.Account{
			Identifier: client.AccountIdentifier{
				DSNPrefix: "coolprefix",
			},
		},
	}

	actions, err := fn.GetActions(s)
	require.NoError(t, err)
	require.Equal(t, 1, len(actions))
	require.EqualValues(t, expectedActionVersion, actions[0])
	// require.EqualValues(t, expectedActionConfig, string(actions[0].config))

	expectedWorkflow := &inngest.Workflow{
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

	wflow, err := fn.GetWorkflow(s)
	require.NoError(t, err)
	require.EqualValues(t, expectedWorkflow, wflow)
}
