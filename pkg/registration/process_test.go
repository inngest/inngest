package registration

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/driver/schemes"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/stretchr/testify/require"
)

func init() {
	schemes.Register("test", func(u *url.URL) string {
		return "test"
	})
}

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
		{
			name: "registered scheme",
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
							"step": {
								ID:   "step",
								Name: "step",
								Runtime: map[string]any{
									"url": "test://serve?fnId=app-hello&stepId=step",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "unregistered scheme",
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
							"step": {
								ID:   "step",
								Name: "step",
								Runtime: map[string]any{
									"url": "foo://serve?fnId=app-hello&stepId=step",
								},
							},
						},
					},
				},
			},
			err: fmt.Errorf("Non-supported step schema: foo"),
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

func TestProcessFunctionsLoopCapability(t *testing.T) {
	loopFn := func() sdk.SDKFunction {
		return sdk.SDKFunction{
			Name: "loop",
			Triggers: []inngest.Trigger{
				{
					EventTrigger: &inngest.EventTrigger{
						Event: "loop/start",
					},
				},
			},
			Loop: &inngest.Loop{
				Kind:    "loop",
				Key:     "event.data.id",
				IdleTTL: time.Hour,
			},
			Steps: map[string]sdk.SDKStep{
				"step": {
					ID:   "step",
					Name: "step",
					Runtime: map[string]any{
						"url": "https://www.example.net/api/inngest",
					},
				},
			},
		}
	}

	t.Run("rejected without the capability", func(t *testing.T) {
		req := sdk.RegisterRequest{
			Functions: []sdk.SDKFunction{loopFn()},
		}
		_, err := ProcessFunctions(context.Background(), req, ProcessOpts{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "the SDK must declare the loop capability")
	})

	t.Run("rejected with an unknown capability version", func(t *testing.T) {
		for _, v := range []string{"false", "v0", "v2"} {
			req := sdk.RegisterRequest{
				Functions: []sdk.SDKFunction{loopFn()},
				Capabilities: sdk.Capabilities{
					Loop: v,
				},
			}
			_, err := ProcessFunctions(context.Background(), req, ProcessOpts{})
			require.Error(t, err, v)
			require.Contains(t, err.Error(), "the SDK must declare the loop capability")
		}
	})

	t.Run("accepted with the capability", func(t *testing.T) {
		req := sdk.RegisterRequest{
			Functions: []sdk.SDKFunction{loopFn()},
			Capabilities: sdk.Capabilities{
				Loop: sdk.LoopV1,
			},
		}
		res, err := ProcessFunctions(context.Background(), req, ProcessOpts{})
		require.NoError(t, err)
		require.Len(t, res.Functions, 1)
		require.NotNil(t, res.Functions[0].Function.Loop)
	})

	t.Run("gets a keyed function semaphore with limit 1", func(t *testing.T) {
		req := sdk.RegisterRequest{
			Functions: []sdk.SDKFunction{loopFn()},
			Capabilities: sdk.Capabilities{
				Loop: sdk.LoopV1,
			},
		}
		res, err := ProcessFunctions(context.Background(), req, ProcessOpts{})
		require.NoError(t, err)
		require.Len(t, res.Functions, 1)
		fn := res.Functions[0]
		require.NotNil(t, fn.Function.Concurrency)
		require.Len(t, fn.Function.Concurrency.Fn, 1)
		fc := fn.Function.Concurrency.Fn[0]
		require.Equal(t, 1, fc.Limit)
		require.Equal(t, inngest.FnConcurrencyScopeFn, fc.EffectiveScope())
		require.NotNil(t, fc.Key)
		require.Equal(t, fn.Function.Loop.Key, *fc.Key)

		sm := &capturingSemaphoreManager{}
		res.SetSemaphoreCapacity(context.Background(), sm)
		require.Len(t, sm.set, 1)
		require.Equal(t, constraintapi.SemaphoreIDFnKey(fn.ID, fn.Function.Loop.Key), sm.set[0].name)
		require.Equal(t, int64(1), sm.set[0].capacity)
	})

	t.Run("user function concurrency fails sync", func(t *testing.T) {
		fn := loopFn()
		fn.Concurrency = &inngest.ConcurrencyLimits{
			Fn: []inngest.FnConcurrency{
				{
					Limit: 5,
				},
			},
		}
		req := sdk.RegisterRequest{
			Functions: []sdk.SDKFunction{fn},
			Capabilities: sdk.Capabilities{
				Loop: sdk.LoopV1,
			},
		}
		_, err := ProcessFunctions(context.Background(), req, ProcessOpts{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "A loop cannot set function concurrency")
	})

	t.Run("a plain function needs no capability", func(t *testing.T) {
		fn := loopFn()
		fn.Loop = nil
		req := sdk.RegisterRequest{
			Functions: []sdk.SDKFunction{fn},
		}
		_, err := ProcessFunctions(context.Background(), req, ProcessOpts{})
		require.NoError(t, err)
	})
}

type semaphoreCapacity struct {
	name     string
	capacity int64
}

// capturingSemaphoreManager records SetCapacity calls.  every other method
// panics through the nil embedded interface.
type capturingSemaphoreManager struct {
	constraintapi.SemaphoreManager
	set []semaphoreCapacity
}

func (m *capturingSemaphoreManager) SetCapacity(_ context.Context, _ uuid.UUID, name, _ string, capacity int64) (constraintapi.SetResult, error) {
	m.set = append(m.set, semaphoreCapacity{
		name:     name,
		capacity: capacity,
	})
	return constraintapi.SetResult{}, nil
}
