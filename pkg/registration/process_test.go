package registration

import (
	"context"
	"fmt"
	"testing"

	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/stretchr/testify/require"
)

func TestProcessFunctions(t *testing.T) {
	tests := []struct {
		name string
		r    sdk.RegisterRequest
		err  error
	}{
		{
			name: "No functions",
			r: sdk.RegisterRequest{
				Functions: []sdk.SDKFunction{},
			},
			err: sdk.ErrNoFunctions,
		},
		{
			name: "no steps",
			r: sdk.RegisterRequest{
				Functions: []sdk.SDKFunction{
					{
						Name: "lol",
						Triggers: []inngest.Trigger{
							{
								EventTrigger: &inngest.EventTrigger{
									Event: "my/event",
								},
							},
						},
					},
				},
			},
			err: fmt.Errorf("Functions must contain one step"),
		},
		{
			name: "no driver",
			r: sdk.RegisterRequest{
				Functions: []sdk.SDKFunction{
					{
						Name: "lol",
						Triggers: []inngest.Trigger{
							{
								EventTrigger: &inngest.EventTrigger{
									Event: "my/event",
								},
							},
						},
						Steps: map[string]sdk.SDKStep{
							"step-id": {
								ID:   "step-id",
								Name: "This is my first step.  It's a goodun, but it uses docker",
							},
						},
					},
				},
			},
			err: fmt.Errorf("No SDK URL"),
		},
		{
			name: "docker driver",
			r: sdk.RegisterRequest{
				Functions: []sdk.SDKFunction{
					{
						Name: "lol",
						Triggers: []inngest.Trigger{
							{
								EventTrigger: &inngest.EventTrigger{
									Event: "my/event",
								},
							},
						},
						Steps: map[string]sdk.SDKStep{
							"step-id": {
								ID:   "step-id",
								Name: "This is my first step.  It's a goodun, but it's not http",
								Runtime: map[string]any{
									"url": "docker://some/image:foo",
								},
							},
						},
					},
				},
			},
			err: fmt.Errorf("Non-supported step schema: docker"),
		},
		{
			name: "valid",
			r: sdk.RegisterRequest{
				Functions: []sdk.SDKFunction{
					{
						Name: "lol",
						Triggers: []inngest.Trigger{
							{
								EventTrigger: &inngest.EventTrigger{
									Event: "my/event",
								},
							},
						},
						Steps: map[string]sdk.SDKStep{
							"step-id": {
								ID:   "step-id",
								Name: "This is my first step.  It's a goodun",
								Runtime: map[string]any{
									"url": "https://www.example.net/lol/what",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, actual := ProcessFunctions(context.Background(), test.r, ProcessOpts{})
			if test.err == nil {
				require.Nil(t, actual)
			} else {
				require.NotNil(t, actual)
				require.Contains(t, actual.Error(), test.err.Error())
			}
		})
	}
}

// TestProcessFunctionsEmptyReturnsNonNilResult is a regression test ensuring
// that ProcessFunctions returns a non-nil *ProcessResult alongside
// sdk.ErrNoFunctions when the request contains no functions. Callers rely on
// the result being non-nil (e.g. to access Functions) even on this error path.
func TestProcessFunctionsEmptyReturnsNonNilResult(t *testing.T) {
	req := sdk.RegisterRequest{Functions: []sdk.SDKFunction{}}

	result, err := ProcessFunctions(context.Background(), req, ProcessOpts{})

	require.ErrorIs(t, err, sdk.ErrNoFunctions)
	require.NotNil(t, result)
	require.NotNil(t, result.Functions)
	require.Len(t, result.Functions, 0)
}
