package sdk

import (
	"context"
	"fmt"
	"testing"

	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/function"
	"github.com/stretchr/testify/require"
)

func TestRegisterRequestValidate(t *testing.T) {
	tests := []struct {
		name string
		r    RegisterRequest
		err  error
	}{
		{
			name: "No functions",
			r: RegisterRequest{
				Functions: []function.Function{},
			},
			err: ErrNoFunctions,
		},
		{
			name: "no steps",
			r: RegisterRequest{
				Functions: []function.Function{
					{
						ID:   "lol",
						Name: "lol",
						Triggers: []function.Trigger{
							{
								EventTrigger: &function.EventTrigger{
									Event: "my/event",
								},
							},
						},
					},
				},
			},
			err: fmt.Errorf("Function has no steps: lol"),
		},
		{
			name: "no driver",
			r: RegisterRequest{
				Functions: []function.Function{
					{
						ID:   "lol",
						Name: "lol",
						Triggers: []function.Trigger{
							{
								EventTrigger: &function.EventTrigger{
									Event: "my/event",
								},
							},
						},
						Steps: map[string]function.Step{
							"step-id": {
								ID:   "step-id",
								Name: "This is my first step.  It's a goodun, but it uses docker",
							},
						},
					},
				},
			},
			err: fmt.Errorf("Step 'step-id' has an invalid driver. Only HTTP drivers may be used with SDK functions."),
		},
		{
			name: "docker driver",
			r: RegisterRequest{
				Functions: []function.Function{
					{
						ID:   "lol",
						Name: "lol",
						Triggers: []function.Trigger{
							{
								EventTrigger: &function.EventTrigger{
									Event: "my/event",
								},
							},
						},
						Steps: map[string]function.Step{
							"step-id": {
								ID:   "step-id",
								Name: "This is my first step.  It's a goodun, but it uses docker",
								Runtime: &inngest.RuntimeWrapper{
									Runtime: inngest.RuntimeDocker{},
								},
							},
						},
					},
				},
			},
			err: fmt.Errorf("Step 'step-id' has an invalid driver. Only HTTP drivers may be used with SDK functions."),
		},
		{
			name: "valid",
			r: RegisterRequest{
				Functions: []function.Function{
					{
						ID:   "lol",
						Name: "lol",
						Triggers: []function.Trigger{
							{
								EventTrigger: &function.EventTrigger{
									Event: "my/event",
								},
							},
						},
						Steps: map[string]function.Step{
							"step-id": {
								ID:   "step-id",
								Name: "This is my first step.  It's a goodun, but it uses docker",
								Runtime: &inngest.RuntimeWrapper{
									Runtime: inngest.RuntimeHTTP{
										URL: "https://www.example.net/lol/what",
									},
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
			actual := test.r.Validate(context.Background())
			if test.err == nil {
				require.Nil(t, actual)
			} else {
				require.NotNil(t, actual)
				require.Contains(t, actual.Error(), test.err.Error())
			}
		})
	}
}
