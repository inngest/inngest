package cli

import (
	"context"
	"fmt"
	"testing"

	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/pkg/function"
	"github.com/stretchr/testify/require"
)

func TestInitFunc(t *testing.T) {
	tests := []struct {
		name          string
		i             *initModel
		fn            *function.Function
		err           error
		validationErr error
	}{
		{
			name: "bare init model",
			i:    &initModel{},
			err:  fmt.Errorf("Unknown trigger type: "),
		},
		{
			name: "bare init model",
			i: &initModel{
				name:        "test fn",
				triggerType: "Event based",
				event:       "test/some-event",
			},
			fn: &function.Function{
				Name: "test fn",
				Triggers: []function.Trigger{
					{
						EventTrigger: &function.EventTrigger{
							Event: "test/some-event",
						},
					},
				},
				Steps: map[string]function.Step{
					function.DefaultStepName: {
						ID:   function.DefaultStepName,
						Name: "test fn",
						Path: function.DefaultStepPath,
						Runtime: inngest.RuntimeWrapper{
							Runtime: inngest.RuntimeDocker{},
						},
					},
				},
			},
		},
	}
	ctx := context.Background()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fn, err := test.i.Function(ctx)
			require.EqualValues(t, test.err, err)

			if fn != nil {
				// IDs are always randomly generated, so we can't assert that
				// here.  Update the ID of the test.
				test.fn.ID = fn.ID
			}

			require.EqualValues(t, test.fn, fn)
			if fn != nil {
				err := fn.Validate(ctx)
				require.EqualValues(t, test.validationErr, err)
			}
		})
	}
}
