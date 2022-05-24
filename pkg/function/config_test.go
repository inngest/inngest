package function

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/inngest/inngest-cli/inngest"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/txtar"
)

func TestUnmarshal_testdata(t *testing.T) {
	entries, err := os.ReadDir("./testdata")
	require.NoError(t, err)
	ctx := context.Background()

	type testdata struct {
		input    []byte
		function []byte
		workflow []byte
	}

	for _, e := range entries {
		t.Run(e.Name(), func(t *testing.T) {
			if !strings.HasSuffix(e.Name(), ".txtar") {
				return
			}

			archive, err := txtar.ParseFile(path.Join("./testdata", e.Name()))
			if err != nil {
				log.Fatal(err)
			}

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
		})
	}

}

// TestUnmarshal asserts that unmarshalling a function definition works as expected, producing
// the correct struct defintions or errors.
func TestUnmarshal(t *testing.T) {
	ctx := context.Background()
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
					"test": {
						Name: "test",
						Runtime: inngest.RuntimeWrapper{
							Runtime: inngest.RuntimeDocker{},
						},
						After: []After{
							{
								Step: inngest.TriggerName,
							},
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
					"test": {
						Name: "test",
						Runtime: inngest.RuntimeWrapper{
							Runtime: inngest.RuntimeDocker{},
						},
						After: []After{
							{
								Step: inngest.TriggerName,
							},
						},
					},
				},
				dir: "/dir",
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
					"test": {
						Name: "test",
						Runtime: inngest.RuntimeWrapper{
							Runtime: inngest.RuntimeDocker{},
						},
						After: []After{
							{
								Step: inngest.TriggerName,
							},
						},
					},
				},
				dir: "/dir",
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
