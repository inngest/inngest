package golang

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/group"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParallelSteps(t *testing.T) {
	ctx := context.Background()

	c := client.New(t)
	inngestClient, server, registerFuncs := NewSDKHandler(t, "parallel")
	defer server.Close()

	var (
		counter int32
		runID   string
	)

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "concurrent", Concurrency: []inngestgo.ConfigStepConcurrency{
			{Limit: 2, Scope: enums.ConcurrencyScopeFn},
		}},
		inngestgo.EventTrigger("test/parallel", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}

			res := group.Parallel(ctx,
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "p1", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&counter, 1)
						<-time.After(5 * time.Second)
						return "p1", nil
					})
				},
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "p2", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&counter, 1)
						<-time.After(5 * time.Second)
						return "p2", nil
					})
				},
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "p3", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&counter, 1)
						<-time.After(5 * time.Second)
						return "p3", nil
					})
				},
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "p4", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&counter, 1)
						<-time.After(5 * time.Second)
						return "p4", nil
					})
				},
			)

			return res, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	_, err = inngestClient.Send(ctx, inngestgo.Event{
		Name: "test/parallel",
		Data: map[string]any{"hello": "world"},
	})
	require.NoError(t, err)

	t.Run("verify in-progress", func(t *testing.T) {
		<-time.After(2 * time.Second)
		require.Equal(t, int32(2), atomic.LoadInt32(&counter))

		_ = c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{
			Status:         models.FunctionStatusRunning,
			ChildSpanCount: 2,
			Timeout:        2 * time.Second,
			Interval:       200 * time.Millisecond,
		})
	})

	t.Run("verify completion", func(t *testing.T) {
		<-time.After(10 * time.Second)
		require.Equal(t, int32(4), atomic.LoadInt32(&counter))

		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{
			Status:         models.FunctionStatusCompleted,
			ChildSpanCount: 4,
			Timeout:        5 * time.Second,
			Interval:       250 * time.Millisecond,
		})

		// check on spans
		for _, cspan := range run.Trace.ChildSpans {
			if cspan.StepOp == "" {
				continue
			}
			t.Run(fmt.Sprintf("child: %s", cspan.Name), func(t *testing.T) {
				assert.Equal(t, 0, cspan.Attempts)
				assert.Equal(t, models.StepOpRun.String(), cspan.StepOp)
				assert.Equal(t, models.RunTraceSpanStatusCompleted.String(), cspan.Status)
			})
		}
	})
}

func TestParallelCoalesce(t *testing.T) {
	// Steps are only called once, regardless of whether they're inside or
	// outside of a parallel group. This test:
	// 1. Diverges into 3 parallel steps.
	// 2. Converges into a single step.
	// 3. Diverges again into 2 parallel steps.
	// 4. Converges again into a single step.

	r := require.New(t)
	ctx := context.Background()
	c := client.New(t)
	ic, server, sync := NewSDKHandler(t, randomSuffix("app"))
	defer server.Close()

	eventName := randomSuffix("event")
	var (
		stepA1Counter      int32
		stepA2Counter      int32
		stepA3Counter      int32
		stepBetweenCounter int32
		stepB1Counter      int32
		stepB2Counter      int32
		stepAfterCounter   int32
		runID              string
	)
	_, err := inngestgo.CreateFunction(
		ic,
		inngestgo.FunctionOpts{ID: "fn"},
		inngestgo.EventTrigger(eventName, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}

			res := group.Parallel(ctx,
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "a1", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&stepA1Counter, 1)
						<-time.After(100 * time.Millisecond)
						return nil, nil
					})
				},
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "a2", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&stepA2Counter, 1)
						<-time.After(100 * time.Millisecond)
						return nil, nil
					})
				},
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "a3", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&stepA3Counter, 1)
						<-time.After(100 * time.Millisecond)
						return nil, nil
					})
				},
			)
			err := res.AnyError()
			if err != nil {
				return nil, err
			}

			_, err = step.Run(ctx, "between", func(ctx context.Context) (any, error) {
				atomic.AddInt32(&stepBetweenCounter, 1)
				time.Sleep(100 * time.Millisecond)
				return nil, nil
			})
			if err != nil {
				return nil, err
			}

			res = group.Parallel(ctx,
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "b1", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&stepB1Counter, 1)
						<-time.After(100 * time.Millisecond)
						return nil, nil
					})
				},
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "b2", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&stepB2Counter, 1)
						<-time.After(100 * time.Millisecond)
						return nil, nil
					})
				},
			)
			err = res.AnyError()
			if err != nil {
				return nil, err
			}

			_, err = step.Run(ctx, "after", func(ctx context.Context) (any, error) {
				atomic.AddInt32(&stepAfterCounter, 1)
				time.Sleep(100 * time.Millisecond)
				return nil, nil
			})
			if err != nil {
				return nil, err
			}
			return res, nil
		},
	)
	r.NoError(err)
	sync()

	_, err = ic.Send(ctx, inngestgo.Event{Name: eventName})
	r.NoError(err)

	c.WaitForRunStatus(ctx, t, "COMPLETED", &runID)
	r.Equal(int32(1), stepA1Counter)
	r.Equal(int32(1), stepA2Counter)
	r.Equal(int32(1), stepA3Counter)
	r.Equal(int32(1), stepBetweenCounter)
	r.Equal(int32(1), stepAfterCounter)
}

func TestParallelSequential(t *testing.T) {
	// 2 parallel groups with 2 sequential steps in each

	t.Parallel()
	r := require.New(t)
	ctx := context.Background()
	c := client.New(t)
	inngestClient, server, registerFuncs := NewSDKHandler(t, randomSuffix("app"))
	defer server.Close()

	eventName := randomSuffix("event")
	var (
		counterA1 int32
		counterA2 int32
		counterB1 int32
		counterB2 int32
		runID     string
		stepOrder []string
	)
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "fn", Retries: inngestgo.Ptr(0)},
		inngestgo.EventTrigger(eventName, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}

			res := group.Parallel(ctx,
				func(ctx context.Context) (any, error) {
					_, err := step.Run(ctx, "a1", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&counterA1, 1)
						stepOrder = append(stepOrder, "a1")
						return nil, nil
					})
					if err != nil {
						return nil, err
					}

					_, err = step.Run(ctx, "a2", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&counterA2, 1)
						stepOrder = append(stepOrder, "a2")
						return nil, nil
					})
					return nil, err
				},
				func(ctx context.Context) (any, error) {
					_, err := step.Run(ctx, "b1", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&counterB1, 1)
						time.Sleep(2 * time.Second)
						stepOrder = append(stepOrder, "b1")
						return nil, nil
					})
					if err != nil {
						return nil, err
					}

					_, err = step.Run(ctx, "b2", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&counterB2, 1)
						time.Sleep(2 * time.Second)
						stepOrder = append(stepOrder, "b2")
						return nil, nil
					})
					return nil, err
				},
			)

			stepOrder = append(stepOrder, "end")
			return res, nil
		},
	)
	r.NoError(err)
	registerFuncs()

	_, err = inngestClient.Send(ctx, inngestgo.Event{Name: eventName})
	r.NoError(err)

	c.WaitForRunStatus(ctx, t, "COMPLETED", &runID, client.WaitForRunStatusOpts{
		Timeout: 20 * time.Second,
	})
	r.Equal(int32(1), atomic.LoadInt32(&counterA1))
	r.Equal(int32(1), atomic.LoadInt32(&counterA2))
	r.Equal(int32(1), atomic.LoadInt32(&counterB1))
	r.Equal(int32(1), atomic.LoadInt32(&counterB2))

	// a2 completes after b1 because "optimized parallelism" doesn't continue
	// until all parallel steps end
	r.Equal([]string{"a1", "b1", "a2", "b2", "end"}, stepOrder)
}

func TestParallelDisabledOptimization(t *testing.T) {
	t.Parallel()

	t.Run("parallel race", func(t *testing.T) {
		// 2 parallel groups with 2 sequential steps in each, but optimized
		// parallelism is disabled. This allows the 2 groups to "race" (i.e. all
		// the steps in one group can complete irrespective of the other group)

		t.Parallel()
		r := require.New(t)
		ctx := context.Background()
		c := client.New(t)
		inngestClient, server, registerFuncs := NewSDKHandler(t, randomSuffix("app"))
		defer server.Close()

		eventName := randomSuffix("event")
		var (
			counterA1 int32
			counterA2 int32
			counterB1 int32
			counterB2 int32
			runID     string
			stepOrder []string
		)
		_, err := inngestgo.CreateFunction(
			inngestClient,
			inngestgo.FunctionOpts{ID: "fn", Retries: inngestgo.Ptr(0)},
			inngestgo.EventTrigger(eventName, nil),
			func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
				if runID == "" {
					runID = input.InputCtx.RunID
				}

				res := group.ParallelWithOpts(ctx,
					group.ParallelOpts{ParallelMode: enums.ParallelModeRace},
					func(ctx context.Context) (any, error) {
						_, err := step.Run(ctx, "a1", func(ctx context.Context) (any, error) {
							atomic.AddInt32(&counterA1, 1)
							stepOrder = append(stepOrder, "a1")
							return nil, nil
						})
						if err != nil {
							return nil, err
						}

						_, err = step.Run(ctx, "a2", func(ctx context.Context) (any, error) {
							atomic.AddInt32(&counterA2, 1)
							stepOrder = append(stepOrder, "a2")
							return nil, nil
						})
						return nil, err
					},
					func(ctx context.Context) (any, error) {
						_, err := step.Run(ctx, "b1", func(ctx context.Context) (any, error) {
							atomic.AddInt32(&counterB1, 1)
							time.Sleep(2 * time.Second)
							stepOrder = append(stepOrder, "b1")
							return nil, nil
						})
						if err != nil {
							return nil, err
						}

						_, err = step.Run(ctx, "b2", func(ctx context.Context) (any, error) {
							atomic.AddInt32(&counterB2, 1)
							time.Sleep(2 * time.Second)
							stepOrder = append(stepOrder, "b2")
							return nil, nil
						})
						return nil, err
					},
				)

				stepOrder = append(stepOrder, "end")
				return res, nil
			},
		)
		r.NoError(err)
		registerFuncs()

		_, err = inngestClient.Send(ctx, inngestgo.Event{Name: eventName})
		r.NoError(err)

		c.WaitForRunStatus(ctx, t, "COMPLETED", &runID, client.WaitForRunStatusOpts{
			Timeout: 20 * time.Second,
		})
		r.Equal(int32(1), atomic.LoadInt32(&counterA1))
		r.Equal(int32(1), atomic.LoadInt32(&counterA2))
		r.Equal(int32(1), atomic.LoadInt32(&counterB1))
		r.Equal(int32(1), atomic.LoadInt32(&counterB2))

		// Steps are in logical order because disabling optimized parallelism causes
		// the parallel groups to race
		r.Equal([]string{"a1", "a2", "b1", "b2", "end"}, stepOrder)
	})

	t.Run("all step kinds", func(t *testing.T) {
		// Ensure that all step kinds work with the "optimize parallel" control. This is
		// necessary because we need to write Executor logic for each step kind

		t.Parallel()

		type testCase struct {
			parallelMode enums.ParallelMode

			// The number of expected SDK requests. Discovery steps count
			expRequestCount int32
		}
		cases := []testCase{
			{parallelMode: enums.ParallelModeNone, expRequestCount: 4},
			{parallelMode: enums.ParallelModeWait, expRequestCount: 4},
			{parallelMode: enums.ParallelModeRace, expRequestCount: 8},
		}

		for _, tc := range cases {
			t.Run(fmt.Sprintf("optimize=%s", tc.parallelMode.String()), func(t *testing.T) {
				t.Parallel()
				r := require.New(t)
				ctx := context.Background()
				c := client.New(t)
				inngestClient, server, registerFuncs := NewSDKHandler(t, randomSuffix("app"))
				defer server.Close()

				eventName := randomSuffix("event")
				var (
					counterRequest int32
					counterRun     int32
					runID          string
				)
				_, err := inngestgo.CreateFunction(
					inngestClient,
					inngestgo.FunctionOpts{ID: "fn", Retries: inngestgo.Ptr(0)},
					inngestgo.EventTrigger(eventName, nil),
					func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
						atomic.AddInt32(&counterRequest, 1)
						if runID == "" {
							runID = input.InputCtx.RunID
						}

						res := group.ParallelWithOpts(ctx,
							group.ParallelOpts{ParallelMode: tc.parallelMode},
							func(ctx context.Context) (any, error) {
								return step.Fetch[string](ctx, "fetch", step.FetchOpts{
									URL:    "http://0.0.0.0:0",
									Method: "GET",
								})
							},
							func(ctx context.Context) (any, error) {
								return step.Invoke[any](
									ctx,
									"invoke",
									step.InvokeOpts{
										FunctionId: "does-not-exist",
										Timeout:    1 * time.Second,
									},
								)
							},
							func(ctx context.Context) (any, error) {
								return step.Run(ctx, "run", func(ctx context.Context) (any, error) {
									atomic.AddInt32(&counterRun, 1)
									time.Sleep(3 * time.Second)
									return nil, nil
								})
							},
							func(ctx context.Context) (any, error) {
								step.Sleep(ctx, "sleep", 5*time.Second)
								return nil, nil
							},
							func(ctx context.Context) (any, error) {
								return step.WaitForEvent[any](ctx, "wait", step.WaitForEventOpts{
									Event:   "does-not-exist",
									Timeout: 7 * time.Second,
								})
							},
						)

						// Sleep long enough to ensure all the discovery steps
						// had enough time to run
						step.Sleep(ctx, "coalesce", 5*time.Second)

						return res, nil
					},
				)
				r.NoError(err)
				registerFuncs()

				_, err = inngestClient.Send(ctx, inngestgo.Event{Name: eventName})
				r.NoError(err)

				c.WaitForRunStatus(ctx, t, "COMPLETED", &runID, client.WaitForRunStatusOpts{
					Timeout: 20 * time.Second,
				})
				r.Equal(tc.expRequestCount, atomic.LoadInt32(&counterRequest))
				r.Equal(1, int(atomic.LoadInt32(&counterRun)))
			})
		}
	})
}

func TestParallelStepsDuplicatePlan(t *testing.T) {
	// A regression test for a bug where a parallel `step.run` would execute
	// multiple times. The bug would happen because the SDK asked the Executor
	// to plan the same step multiple times, and then the Executor would
	// mistakenly not reuse the same job ID for each (so idempotency didn't
	// prevent multiple scheduled jobs).
	//
	// To replicate this bug, we need to resume the run while the `step.run`
	// is still executing. We achieve this by using multiple `step.WaitForEvent`
	// calls with different timeouts.

	t.Parallel()
	r := require.New(t)
	ctx := context.Background()

	c := client.New(t)
	inngestClient, server, registerFuncs := NewSDKHandler(t, randomSuffix("app"))
	defer server.Close()

	var (
		counter int32
		runID   string
	)
	eventName := randomSuffix("event")
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:      "fn",
			Retries: inngestgo.Ptr(0),
		},
		inngestgo.EventTrigger(eventName, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}

			// Need 2 waits (each with a different timeout) to replicate the bug
			waitTimeouts := []time.Duration{
				1 * time.Second,
				2 * time.Second,
			}

			steps := []func(ctx context.Context) (any, error){
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "a", func(ctx context.Context) (any, error) {
						atomic.AddInt32(&counter, 1)

						// Sleep as long as the longest wait timeout to
						// replicate the bug
						<-time.After(waitTimeouts[len(waitTimeouts)-1])

						return nil, nil
					})
				},
			}

			for i, timeout := range waitTimeouts {
				steps = append(steps, func(ctx context.Context) (any, error) {
					return step.WaitForEvent[any](ctx, fmt.Sprintf("wait-%d", i),
						step.WaitForEventOpts{
							Event: randomSuffix("never"),

							// Each wait needs a different timeout to replicate
							// the bug
							Timeout: timeout,
						},
					)
				})
			}

			res := group.ParallelWithOpts(ctx,
				// Need to use race mode to replicate the bug
				group.ParallelOpts{ParallelMode: enums.ParallelModeRace},
				steps...,
			)

			return res, nil
		},
	)
	r.NoError(err)
	registerFuncs()

	_, err = inngestClient.Send(ctx, inngestgo.Event{Name: eventName})
	r.NoError(err)
	c.WaitForRunStatus(ctx, t, "COMPLETED", &runID, client.WaitForRunStatusOpts{
		Timeout: 10 * time.Second,
	})

	r.Equal(1, int(atomic.LoadInt32(&counter)))
}

func TestParallelStepFailuresOnFailureDeduplication(t *testing.T) {
	ctx := context.Background()
	c := client.New(t)

	testID := fmt.Sprintf("parallel-fail-%d", time.Now().UnixNano())
	inngestClient, server, registerFuncs := NewSDKHandler(t, testID)
	defer server.Close()

	failureCount := int32(0)
	runID := ""

	functionID := fmt.Sprintf("parallel-fail-test-%d", time.Now().UnixNano())
	eventName := fmt.Sprintf("test/parallel-fail-%d", time.Now().UnixNano())

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:      functionID,
			Retries: inngestgo.Ptr(0),
		},
		inngestgo.EventTrigger(eventName, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			if runID == "" {
				runID = input.InputCtx.RunID
			}

			res := group.Parallel(ctx,
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "step-a", func(ctx context.Context) (any, error) {
						return nil, fmt.Errorf("step a failed")
					})
				},
				func(ctx context.Context) (any, error) {
					return step.Run(ctx, "step-b", func(ctx context.Context) (any, error) {
						return nil, fmt.Errorf("step b failed")
					})
				},
				)

			// Check if any parallel step failed and propagate error to trigger onFailure
			if err := res.AnyError(); err != nil {
				return nil, err
			}

			return res, nil
		},
		)
	require.NoError(t, err)

	// onFailure equivalent function
	handlerID := fmt.Sprintf("handle-parallel-failures-%d", time.Now().UnixNano())
	expectedFunctionID := fmt.Sprintf("%s-%s", testID, functionID)

	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:      handlerID,
			Retries: inngestgo.Ptr(0),
		},
		inngestgo.EventTrigger(
			"inngest/function.failed",
			inngestgo.StrPtr(fmt.Sprintf("event.data.function_id == '%s'", expectedFunctionID)),
			),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			atomic.AddInt32(&failureCount, 1)

			assert.NotEmpty(t, input.Event.Data["run_id"])
			assert.Equal(t, expectedFunctionID, input.Event.Data["function_id"])

			return "handled", nil
		},
		)
	require.NoError(t, err)

	registerFuncs()

	_, err = inngestClient.Send(ctx, inngestgo.Event{
		Name: eventName,
		Data: map[string]any{"test": true},
	})
	require.NoError(t, err)

	require.EventuallyWithT(t, func(t *assert.CollectT) {
		require.Equal(t, 1, int(atomic.LoadInt32(&failureCount)))
	}, 15*time.Second, 500*time.Millisecond)

	run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{
		Status:  models.FunctionStatusFailed,
		Timeout: 10 * time.Second,
	})
	require.Equal(t, models.RunTraceSpanStatusFailed.String(), run.Trace.Status)
}
