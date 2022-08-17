package function

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/inngest/clistate"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/txtar"
)

const exampleTxtar = `
-- function.json --
{
  "name": "test",
  "id": "some-id",
  "triggers": [
    {
      "event": "test.event",
      "expression": "event.data.run == true"
    }
  ],
  "idempotency": "{{ event.data.foo }}",
  "throttle": {
    "count": 1,
    "period": "24h",
    "key": "{{ event.data.foo }}"
  },
  "steps": {
    "first": {
      "id": "first",
      "path": "",
      "name": "My first func",
      "runtime": {
        "type": "docker"
      },
      "after": [
        {
          "step": "$trigger",
          "wait": "5m"
        }
      ]
    },
    "second": {
      "id": "second",
      "path": "",
      "name": "A second func that does something cool!",
      "runtime": {
        "type": "docker"
      },
      "after": [
        {
          "step": "first"
        }
      ]
    }
  }
}
-- input --
package main

import (
        defs "inngest.com/defs/v1"
)

function: defs.#Function & {
        id:   "some-id"
        name: "test"
        triggers: [defs.#EventTrigger & {
                event:      "test.event"
                expression: "event.data.run == true"
        }]
        idempotency: "{{ event.data.foo }}"
        steps: {
                first: {
                        name: "My first func"
                        runtime: defs.#RuntimeDocker
                        after: [{
                                step: "$trigger"
                                wait: "5m"
                        }]
                }
                second: {
                        name: "A second func that does something cool!"
                        runtime: defs.#RuntimeDocker
                        after: [{
                                step: "first"
                        }]
                }
        }
}
-- workflow.json --
{
  "id": "some-id",
  "name": "test",
  "throttle": {
    "count": 1,
    "period": "24h",
    "key": "{{ event.data.foo }}"
  },
  "triggers": [
    {
      "event": "test.event",
      "expression": "event.data.run == true"
    }
  ],
  "actions": [
    {
      "id": "first",
      "clientID": 1,
      "name": "My first func",
      "dsn": "some-id-step-first-test"
    },
    {
      "id": "second",
      "clientID": 2,
      "name": "A second func that does something cool!",
      "dsn": "some-id-step-second-test"
    }
  ],
  "edges": [
    {
      "outgoing": "$trigger",
      "incoming": "first",
      "metadata": {
        "wait": "5m"
      }
    },
    {
      "outgoing": "first",
      "incoming": "second"
    }
  ]
}
`

func TestUnmarshal_testdata(t *testing.T) {
	ctx := context.Background()

	type testdata struct {
		input    []byte
		function []byte
		workflow []byte
	}

	err := clistate.Clear(context.Background())
	require.NoError(t, err)

	archive := txtar.Parse([]byte(exampleTxtar))
	if err != nil {
		require.NoError(t, err)
	}

	fmt.Println(string(archive.Comment))

	td := testdata{}
	for _, f := range archive.Files {
		switch f.Name {
		case "input":
			td.input = f.Data
		case "function.json":
			td.function = f.Data
		case "workflow.json":
			td.workflow = f.Data
		}
	}

	require.NotEmpty(t, td.input)
	require.NotEmpty(t, td.function)
	require.NotEmpty(t, td.workflow)

	fn, err := Unmarshal(ctx, td.input, "/dir/inngest.json")
	require.NoError(t, err)

	marshalled, err := json.MarshalIndent(fn, "", "  ")
	require.NoError(t, err)
	require.EqualValues(t, strings.TrimSpace(string(td.function)), string(marshalled))

	flow, err := fn.Workflow(context.Background())
	require.NoError(t, err)

	marshalled, err = json.MarshalIndent(flow, "", "  ")
	require.NoError(t, err)
	require.EqualValues(t, strings.TrimSpace(string(td.workflow)), string(marshalled))
}

// TestUnmarshal asserts that unmarshalling a function definition works as expected, producing
// the correct struct defintions or errors.
func TestUnmarshal(t *testing.T) {
	ctx := context.Background()

	var int1 int = 1
	var uint1 uint = 1
	var uint2 uint = 2
	var uint3 uint = 3

	version23 := &inngest.VersionConstraint{
		Major: &uint2,
		Minor: &uint3,
	}

	version11 := &inngest.VersionConstraint{
		Major: &uint1,
		Minor: &uint1,
	}

	valid := []struct {
		name     string
		input    string
		expected Function
	}{
		{
			name:  "simplest json defintion",
			input: `{"id":"wut", "name":"test", triggers: [{ "event": "test.event" }] }`,
			expected: Function{
				Name: "test",
				ID:   "wut",
				Triggers: []Trigger{
					{EventTrigger: &EventTrigger{Event: "test.event"}},
				},
				Steps: map[string]Step{
					DefaultStepName: {
						ID:   DefaultStepName,
						Name: "test",
						Path: "file://.",
						Runtime: inngest.RuntimeWrapper{
							Runtime: inngest.RuntimeDocker{},
						},
						After: []After{
							{
								Step: inngest.TriggerName,
							},
						},
						Version: version11,
					},
				},
				dir: filepath.FromSlash("/dir"),
			},
		},
		{
			name: "simplest json defintion with step version constraints",
			input: `{
				"id": "wut",
				"name": "test",
				"triggers": [{ "event": "test.event" }],
				"steps": {
					"step-1": {
						"id": "step-1",
						"path": "file://.",
						"name": "test",
						"runtime": { "type": "docker" },
						"after": [
							{
								"step": "$trigger"
							}
						],
						"version": {
							"major": 2,
							"minor": 3
						}
					}
				}
			}`,
			expected: Function{
				Name: "test",
				ID:   "wut",
				Triggers: []Trigger{
					{EventTrigger: &EventTrigger{Event: "test.event"}},
				},
				Steps: map[string]Step{
					DefaultStepName: {
						ID:   DefaultStepName,
						Name: "test",
						Path: "file://.",
						Runtime: inngest.RuntimeWrapper{
							Runtime: inngest.RuntimeDocker{},
						},
						After: []After{
							{
								Step: inngest.TriggerName,
							},
						},
						Version: version23,
					},
				},
				dir: filepath.FromSlash("/dir"),
			},
		},
		{
			name: "simplest json defintion with step retry attempts",
			input: `{
				"id": "wut",
				"name": "test",
				"triggers": [{ "event": "test.event" }],
				"steps": {
					"step-1": {
						"id": "step-1",
						"path": "file://.",
						"name": "test",
						"runtime": { "type": "docker" },
						"after": [
							{
								"step": "$trigger"
							}
						],
						"retries": {
							"attempts": 1
						}
					}
				}
			}`,
			expected: Function{
				Name: "test",
				ID:   "wut",
				Triggers: []Trigger{
					{EventTrigger: &EventTrigger{Event: "test.event"}},
				},
				Steps: map[string]Step{
					DefaultStepName: {
						ID:   DefaultStepName,
						Name: "test",
						Path: "file://.",
						Runtime: inngest.RuntimeWrapper{
							Runtime: inngest.RuntimeDocker{},
						},
						After: []After{
							{
								Step: inngest.TriggerName,
							},
						},
						Retries: &inngest.RetryOptions{
							Attempts: &int1,
						},
					},
				},
				dir: "/dir",
			},
		},
		{
			name: "simplest plain cue definition",
			input: `
			{
				id: "wut"
				name: "test"
				triggers: [{
					event: "test.event"
				}]
			}`,
			expected: Function{
				Name: "test",
				ID:   "wut",
				Triggers: []Trigger{
					{EventTrigger: &EventTrigger{Event: "test.event"}},
				},
				Steps: map[string]Step{
					DefaultStepName: {
						ID:   DefaultStepName,
						Name: "test",
						Path: "file://.",
						Runtime: inngest.RuntimeWrapper{
							Runtime: inngest.RuntimeDocker{},
						},
						After: []After{
							{
								Step: inngest.TriggerName,
							},
						},
						Version: version11,
					},
				},
				dir: filepath.FromSlash("/dir"),
			},
		},
		{
			name: "simplest cue definition with imports",
			input: `
				package whatevs
				import (
					defs "inngest.com/defs/v1"
				)

				function: defs.#Function & {
					id: "hellz-yea"
					name: "test"
					triggers: [{
						event: "test.event"
					}]
				}`,
			expected: Function{
				Name: "test",
				ID:   "hellz-yea",
				Triggers: []Trigger{
					{EventTrigger: &EventTrigger{Event: "test.event"}},
				},
				Steps: map[string]Step{
					DefaultStepName: {
						ID:   DefaultStepName,
						Name: "test",
						Path: "file://.",
						Runtime: inngest.RuntimeWrapper{
							Runtime: inngest.RuntimeDocker{},
						},
						After: []After{
							{
								Step: inngest.TriggerName,
							},
						},
						Version: version11,
					},
				},
				dir: filepath.FromSlash("/dir"),
			},
		},
		{
			name: "simplest cue definition with step version constraints",
			input: `
			package whatevs

			import (
				defs "inngest.com/defs/v1"
			)

			function: defs.#Function & {
				id:   "hellz-yea"
				name: "test"
				triggers: [{
					event: "test.event"
				}]
				steps: {
					"step-1": {
						id:   "step-1"
						path: "file://."
						name: "test"
						runtime: {"type": "docker"}
						after: [
							{
								step: "$trigger"
							},
						]
						version: {
							major:   2
							minor:   3
						}
					}
				}
			}`,
			expected: Function{
				Name: "test",
				ID:   "hellz-yea",
				Triggers: []Trigger{
					{EventTrigger: &EventTrigger{Event: "test.event"}},
				},
				Steps: map[string]Step{
					DefaultStepName: {
						ID:   DefaultStepName,
						Name: "test",
						Path: "file://.",
						Runtime: inngest.RuntimeWrapper{
							Runtime: inngest.RuntimeDocker{},
						},
						After: []After{
							{
								Step: inngest.TriggerName,
							},
						},
						Version: version23,
					},
				},
				dir: filepath.FromSlash("/dir"),
			},
		},
	}

	for _, i := range valid {
		t.Run(i.name, func(t *testing.T) {
			f, err := Unmarshal(ctx, []byte(i.input), "/dir/inngest.json")
			require.NoError(t, err, i.name)
			require.NotNil(t, f, i.name)
			require.EqualValues(t, i.expected, *f, i.name)
		})
	}

	invalid := []struct {
		name  string
		input string
		// The error message contains this string
		msg string
	}{
		{
			name:  "no ID specified",
			input: `{"name":"test"}`,
			msg:   "ID is required",
		},
		{
			name:  "no trigger specified",
			input: `{"name":"test", "id": "wut"}`,
			msg:   "trigger is required",
		},
		{
			name:  "no trigger in array",
			input: `{"name":"test", triggers: [] }`,
			msg:   "trigger is required",
		},
		{
			name:  "no event trigger",
			input: `{"name":"test", triggers: [{ "event": "" }] }`,
			msg:   "event name",
		},
	}

	for _, i := range invalid {
		f, err := Unmarshal(ctx, []byte(i.input), "/dir/inngest.json")
		require.Error(t, err, i.name)
		require.Contains(t, err.Error(), i.msg, i.name)
		require.Nil(t, f, i.name)
	}
}

func TestFormatCue(t *testing.T) {
	// Parse input JSON
	ctx := context.Background()
	input := `{"id":"wut", "name":"test", triggers: [{ "event": "test.event" }] }`
	f, err := Unmarshal(ctx, []byte(input), ".")
	require.Nil(t, err)
	str, err := formatCue(*f)
	require.Nil(t, err)

	expected := `package main

import (
	defs "inngest.com/defs/v1"
)

function: defs.#Function & {
  name: "test"
  id:   "wut"
  triggers: [{
    event: "test.event"
  }]
  steps: "step-1": {
    id:   "step-1"
    path: "file://."
    name: "test"
    runtime: type: "docker"
    after: [{
      step: "$trigger"
    }]
    version: {
      major: 1
      minor: 1
    }
  }
}`
	require.Equal(t, []byte(expected), str)

	// Ensure parsing this works.
	f2, err := Unmarshal(ctx, []byte(str), ".")
	require.Nil(t, err)
	require.EqualValues(t, *f, *f2)
}
