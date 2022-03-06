package function

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUnmarshal asserts that unmarshalling a function definition works as expected, producing
// the correct struct defintions or errors.
func TestUnmarshal(t *testing.T) {
	valid := []struct {
		name     string
		input    string
		expected Function
	}{
		{
			name:  "simplest json defintion",
			input: `{"name":"test", triggers: [{ "event": "test.event" }] }`,
			expected: Function{
				Name: "test",
				Triggers: []Trigger{
					{EventTrigger: &EventTrigger{Event: "test.event"}},
				},
			},
		},
		{
			name: "simplest plain cue definition",
			input: `
			{
				name: "test"
				triggers: [{
					event: "test.event"
				}]
			}`,
			expected: Function{
				Name: "test",
				Triggers: []Trigger{
					{EventTrigger: &EventTrigger{Event: "test.event"}},
				},
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
					name: "test"
					triggers: [{
						event: "test.event"
					}]
				}`,
			expected: Function{
				Name: "test",
				Triggers: []Trigger{
					{EventTrigger: &EventTrigger{Event: "test.event"}},
				},
			},
		},
	}

	for _, i := range valid {
		f, err := Unmarshal([]byte(i.input))
		require.NoError(t, err, i.name)
		require.NotNil(t, f, i.name)
		require.EqualValues(t, i.expected, *f, i.name)
	}

	invalid := []struct {
		name  string
		input string
		// The error message contains this string
		msg string
	}{
		{
			name:  "no trigger specified",
			input: `{"name":"test"}`,
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
		f, err := Unmarshal([]byte(i.input))
		require.Error(t, err, i.name)
		require.Contains(t, err.Error(), i.msg, i.name)
		require.Nil(t, f, i.name)
	}
}
