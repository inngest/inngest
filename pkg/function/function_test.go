package function

/*
func TestValidate(t *testing.T) {
	tests := []struct {
		f   Function
		err error
	}{
		// Invalid expression
		{
			f: Function{
				Name: "Foo",
				ID:   "well-hello",
				Triggers: []Trigger{
					{
						EventTrigger: &EventTrigger{
							Event:      "lol",
							Expression: strptr("undefined.foo == 1"),
						},
					},
				},
			},
			err: fmt.Errorf("undeclared reference to 'undefined'"),
		},
		// invalid edge expression
		{
			f: Function{
				Name: "Foo",
				ID:   "well-hello",
				Triggers: []Trigger{
					{
						EventTrigger: &EventTrigger{
							Event:      "lol",
							Expression: strptr("event.data.ok == true"),
						},
					},
				},
				Steps: map[string]Step{
					"id": {
						ID:   "id",
						Path: "file://.",
						Name: "lol",
						Runtime: &inngest.RuntimeWrapper{
							Runtime: inngest.RuntimeHTTP{
								URL: "https://www.example.com",
							},
						},
					},
					"next": {
						ID:   "next",
						Path: "file://.",
						Name: "lol",
						Runtime: &inngest.RuntimeWrapper{
							Runtime: inngest.RuntimeHTTP{
								URL: "https://www.example.com",
							},
						},
						After: []After{
							{
								Step: "id",
								If:   "lol.what == 3.141",
							},
						},
					},
				},
			},
			err: fmt.Errorf("undeclared reference to 'lol'"),
		},
		// Invalid cron
		{
			f: Function{
				Name: "Foo",
				ID:   "well-hello",
				Triggers: []Trigger{
					{
						CronTrigger: &CronTrigger{
							Cron: "u wot m8",
						},
					},
				},
			},
			err: fmt.Errorf("'u wot m8' isn't a valid cron schedule"),
		},
		// valid cron
		{
			f: Function{
				Name: "Foo",
				ID:   "well-hello",
				Triggers: []Trigger{
					{
						CronTrigger: &CronTrigger{
							Cron: "0 * * * *",
						},
					},
				},
				Steps: map[string]Step{
					"id": {
						ID:   "id",
						Path: "file://.",
						Name: "lol",
						Runtime: &inngest.RuntimeWrapper{
							Runtime: inngest.RuntimeHTTP{
								URL: "https://www.example.com",
							},
						},
					},
				},
			},
			err: nil,
		},
		// valid trigger expression
		{
			f: Function{
				Name: "Foo",
				ID:   "well-hello",
				Triggers: []Trigger{
					{
						EventTrigger: &EventTrigger{
							Event:      "lol",
							Expression: strptr("event.data.ok == true"),
						},
					},
				},
				Steps: map[string]Step{
					"id": {
						ID:   "id",
						Path: "file://.",
						Name: "lol",
						Runtime: &inngest.RuntimeWrapper{
							Runtime: inngest.RuntimeHTTP{
								URL: "https://www.example.com",
							},
						},
					},
				},
			},
			err: nil,
		},
	}

	ctx := context.Background()
	for _, test := range tests {
		err := test.f.Validate(ctx)

		if test.err == nil {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
			require.ErrorContains(t, err, test.err.Error())
		}
	}
}

// TestDerivedConfigDefault asserts that the derived config for simple, default workflows
// is correct.
func TestDerivedConfigDefault(t *testing.T) {
	err := clistate.Clear(context.Background())
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
	path := "file:///Users/johnny/dev/repo/functions/inngest.json"

	var majorVersion uint = 1
	var minorVersion uint = 1

	err = fn.canonicalize(context.Background(), path)
	require.NoError(t, err)

	expectedActionVersion := inngest.ActionVersion{
		Name:   "Foo",
		DSN:    "magical-id-step-step-1-test",
		Scopes: []string{"secret:read:*"},
		Runtime: inngest.RuntimeWrapper{
			Runtime: inngest.RuntimeDocker{},
		},
		Version: &inngest.VersionInfo{
			Major: 1,
			Minor: 1,
		},
	}

	expectedActionConfig := `package main

import (
	"inngest.com/actions"
)

action: actions.#Action
action: {
  dsn:  "magical-id-step-step-1-test"
  name: "Foo"
  version: {
    major: 1
    minor: 1
  }
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
				ID:       DefaultStepName,
				ClientID: 1,
				Name:     expectedActionVersion.Name,
				DSN:      expectedActionVersion.DSN,
				Version: &inngest.VersionConstraint{
					Major: &majorVersion,
					Minor: &minorVersion,
				},
			},
		},
		Edges: []inngest.Edge{
			{
				Outgoing: inngest.TriggerName,
				Incoming: DefaultStepName,
			},
		},
	}

	expectedWorkflowConfig := `package main

import (
	"inngest.com/workflows"
)

workflow: workflows.#Workflow & {
  concurrency: 0
  id:          "magical-id"
  name:        "Foo"
  triggers: [{
    event:      "test.event.plz"
    expression: "event.version >= 2"
  }]
  actions: [{
    id:       "step-1"
    clientID: 1
    name:     "Foo"
    dsn:      "magical-id-step-step-1-test"
    version: {
      major: 1
      minor: 1
    }
  }]
  edges: [{
    incoming: "step-1"
    outgoing: "$trigger"
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
	err := clistate.Clear(context.Background())
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

	abs := "file://./events/event-def-in-file.cue"
	require.EqualValues(t, abs, fn.Triggers[0].EventTrigger.Definition.Def)
}

func TestFunctionActions_single(t *testing.T) {
	fn := Function{
		ID:   "hi",
		Name: "test",
		Triggers: []Trigger{{
			EventTrigger: &EventTrigger{
				Event: "test/foo.bar",
			},
		}},
		Steps: map[string]Step{
			"single": {
				ID:   "single",
				Name: "single",
				Runtime: &inngest.RuntimeWrapper{
					Runtime: &stubdriver{},
				},
			},
		},
	}
	err := fn.Validate(context.Background())
	require.NoError(t, err)

	actions, edges, err := fn.Actions(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, len(actions))
	require.Equal(t, 1, len(edges))
	require.Equal(t, inngest.TriggerName, edges[0].Outgoing)
	require.Equal(t, "single", edges[0].Incoming)
}

func TestDeterministicUUID(t *testing.T) {
	tests := []struct {
		f Function
		e uuid.UUID
	}{
		{
			f: Function{
				ID: "hello",
			},
			e: uuid.UUID{0x4d, 0x71, 0xd0, 0x3f, 0xf1, 0x9b, 0x5d, 0x9e, 0x85, 0x23, 0x96, 0x28, 0xba, 0x18, 0x6, 0x3c},
		},
		{
			f: Function{
				ID: "magic-whirl-8156df",
			},
			e: uuid.UUID{0x6d, 0x59, 0x9b, 0x84, 0xcf, 0x3a, 0x5c, 0x89, 0x9d, 0x12, 0xaf, 0xa8, 0xa, 0xf, 0x50, 0xc8},
		},
	}

	for _, test := range tests {
		out := DeterministicUUID(test.f)
		out2 := DeterministicUUID(test.f)
		require.Equal(t, test.e, out)
		require.Equal(t, test.e, out2)
	}
}

func strptr(s string) *string {
	return &s
}
*/
