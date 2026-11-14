package inngest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/require"
)

func TestURI(t *testing.T) {
	str := `{
	  "fv": 1,
	  "id": "d1ca3d3e-9976-437d-b3e9-e2a11218bbb2",
	  "name": "Stuff",
	  "slug": "fn-stuff",
	  "steps": [
	    {
	      "id": "step",
	      "uri": "https://example.com/api/inngest?&fnId=fn-stuff&stepId=step",
	      "name": "step"
	    }
	  ],
	  "triggers": [
	    {
	      "event": "run/init"
	    }
	  ],
	  "concurrency": {
	    "limit": 1
	  }
	}`

	fn := Function{}
	err := json.Unmarshal([]byte(str), &fn)
	require.NoError(t, err)

	expected, err := url.Parse("https://example.com/api/inngest?&fnId=fn-stuff&stepId=step")
	require.NoError(t, err)

	actual := fn.URI()
	require.EqualValues(t, *expected, *actual)
}

func TestValidate(t *testing.T) {
	t.Run("Failures", func(t *testing.T) {
		t.Run("With a non-HTTP URI", func(t *testing.T) {
			f := Function{
				Name: "hi",
				Triggers: []Trigger{
					{
						EventTrigger: &EventTrigger{
							Event: "fail",
						},
					},
				},
				Steps: []Step{
					{
						ID:   "step",
						Name: "Function body",
						URI:  "htt://lol/what.xml.api",
					},
				},
			}

			err := f.Validate(context.Background())
			require.NotNil(t, err)
			require.Contains(t, err.Error(), "Non-supported step schema: htt")
		})

		t.Run("With an invalid cache expression", func(t *testing.T) {
			f := Function{
				Name: "hi",
				Triggers: []Trigger{
					{
						EventTrigger: &EventTrigger{
							Event: "fail",
						},
					},
				},
				Concurrency: &ConcurrencyLimits{
					Limits: []StepConcurrency{
						{
							Limit: 5,
							Key:   new("invalid because not a string"),
						},
					},
				},
				Steps: []Step{
					{
						ID:   "step",
						Name: "Function body",
						URI:  "http://lol/what.xml.api",
					},
				},
			}

			err := f.Validate(context.Background())
			require.NotNil(t, err)
			require.Contains(t, err.Error(), "Invalid concurrency key")
		})

		t.Run("Without edges", func(t *testing.T) {
			f := Function{
				Name: "hi",
				Triggers: []Trigger{
					{
						EventTrigger: &EventTrigger{
							Event: "fail",
						},
					},
				},
			}

			err := f.Validate(context.Background())
			require.NotNil(t, err)
			require.Contains(t, err.Error(), "Functions must contain one step")
		})
	})
}

func TestRunPriorityFactor(t *testing.T) {
	ctx := context.Background()
	f := Function{}

	pf, err := f.RunPriorityFactor(ctx, map[string]any{})
	require.NoError(t, err)
	require.EqualValues(t, 0, pf)

	t.Run("With ternaries", func(t *testing.T) {
		f.Priority = &Priority{
			Run: new("event.data.plan == 'paid' ? 100 : 0"),
		}

		pf, err := f.RunPriorityFactor(ctx, map[string]any{
			"data": map[string]any{"plan": "free"},
		})
		require.NoError(t, err)
		require.EqualValues(t, 0, pf)

		pf, err = f.RunPriorityFactor(ctx, map[string]any{
			"data": map[string]any{"plan": "paid"},
		})
		require.NoError(t, err)
		require.EqualValues(t, 100, pf)
	})

	t.Run("With an int return value in the expression", func(t *testing.T) {
		f.Priority = &Priority{
			Run: new("event.data.priority"),
		}

		pf, err := f.RunPriorityFactor(ctx, map[string]any{
			"data": map[string]any{"priority": 1},
		})
		require.NoError(t, err)
		require.EqualValues(t, 1, pf)

		pf, err = f.RunPriorityFactor(ctx, map[string]any{
			"data": map[string]any{"priority": 100},
		})
		require.NoError(t, err)
		require.EqualValues(t, 100, pf)

		pf, err = f.RunPriorityFactor(ctx, map[string]any{
			"data": map[string]any{"priority": consts.PriorityFactorMax + 1},
		})
		require.NoError(t, err)
		require.EqualValues(t, consts.PriorityFactorMax, pf)

		pf, err = f.RunPriorityFactor(ctx, map[string]any{
			"data": map[string]any{"priority": -1},
		})
		require.NoError(t, err)
		require.EqualValues(t, -1, pf)

		pf, err = f.RunPriorityFactor(ctx, map[string]any{
			"data": map[string]any{"priority": consts.PriorityFactorMin - 1},
		})
		require.NoError(t, err)
		require.EqualValues(t, consts.PriorityFactorMin, pf)
	})

	t.Run("With missing data", func(t *testing.T) {
		f.Priority = &Priority{
			Run: new("event.data.priority"),
		}

		pf, err := f.RunPriorityFactor(ctx, map[string]any{
			"data": map[string]any{},
		})
		require.EqualValues(t, 0, pf)
		require.ErrorContains(t, err, "Priority.Run expression returned non-int: false")
	})

	t.Run("With an invalid expression", func(t *testing.T) {
		f.Priority = &Priority{
			Run: new("event.data.priority = 123"),
		}

		pf, err := f.RunPriorityFactor(ctx, map[string]any{
			"data": map[string]any{},
		})
		require.EqualValues(t, 0, pf)
		require.ErrorContains(t, err, "Priority.Run expression is invalid")
	})
}

func TestIsScheduled(t *testing.T) {
	t.Run("returns false for function with no triggers", func(t *testing.T) {
		f := Function{}
		require.False(t, f.IsScheduled())
	})

	t.Run("returns false for function with only event triggers", func(t *testing.T) {
		f := Function{
			Triggers: []Trigger{
				{
					EventTrigger: &EventTrigger{
						Event: "user.created",
					},
				},
				{
					EventTrigger: &EventTrigger{
						Event: "user.updated",
					},
				},
			},
		}
		require.False(t, f.IsScheduled())
	})

	t.Run("returns true for function with cron trigger", func(t *testing.T) {
		f := Function{
			Triggers: []Trigger{
				{
					CronTrigger: &CronTrigger{
						Cron: "0 9 * * *",
					},
				},
			},
		}
		require.True(t, f.IsScheduled())
	})

	t.Run("returns true for function with mixed triggers including cron", func(t *testing.T) {
		f := Function{
			Triggers: []Trigger{
				{
					EventTrigger: &EventTrigger{
						Event: "user.created",
					},
				},
				{
					CronTrigger: &CronTrigger{
						Cron: "*/15 * * * *",
					},
				},
			},
		}
		require.True(t, f.IsScheduled())
	})

	t.Run("returns true for function with multiple cron triggers", func(t *testing.T) {
		f := Function{
			Triggers: []Trigger{
				{
					CronTrigger: &CronTrigger{
						Cron: "0 9 * * *",
					},
				},
				{
					CronTrigger: &CronTrigger{
						Cron: "0 17 * * *",
					},
				},
			},
		}
		require.True(t, f.IsScheduled())
	})
}

func TestScheduleExpressions(t *testing.T) {
	t.Run("returns empty slice for function with no triggers", func(t *testing.T) {
		f := Function{}
		require.Empty(t, f.ScheduleExpressions())
	})

	t.Run("returns empty slice for function with only event triggers", func(t *testing.T) {
		f := Function{
			Triggers: []Trigger{
				{
					EventTrigger: &EventTrigger{
						Event: "user.created",
					},
				},
				{
					EventTrigger: &EventTrigger{
						Event: "user.updated",
					},
				},
			},
		}
		require.Empty(t, f.ScheduleExpressions())
	})

	t.Run("returns cron expression for function with single cron trigger", func(t *testing.T) {
		cronExpr := "0 9 * * *"
		f := Function{
			Triggers: []Trigger{
				{
					CronTrigger: &CronTrigger{
						Cron: cronExpr,
					},
				},
			},
		}
		cronExprs := f.ScheduleExpressions()
		require.Len(t, cronExprs, 1)
		require.Equal(t, cronExpr, cronExprs[0])
	})

	t.Run("returns all cron expressions for function with multiple cron triggers", func(t *testing.T) {
		firstCronExpr := "0 9 * * *"
		secondCronExpr := "0 17 * * *"
		f := Function{
			Triggers: []Trigger{
				{
					CronTrigger: &CronTrigger{
						Cron: firstCronExpr,
					},
				},
				{
					CronTrigger: &CronTrigger{
						Cron: secondCronExpr,
					},
				},
			},
		}
		cronExprs := f.ScheduleExpressions()
		require.Len(t, cronExprs, 2)
		require.Equal(t, firstCronExpr, cronExprs[0])
		require.Equal(t, secondCronExpr, cronExprs[1])
	})

	t.Run("returns cron expression for mixed triggers with cron", func(t *testing.T) {
		cronExpr := "*/15 * * * *"
		f := Function{
			Triggers: []Trigger{
				{
					EventTrigger: &EventTrigger{
						Event: "user.created",
					},
				},
				{
					CronTrigger: &CronTrigger{
						Cron: cronExpr,
					},
				},
				{
					EventTrigger: &EventTrigger{
						Event: "user.updated",
					},
				},
			},
		}
		cronExprs := f.ScheduleExpressions()
		require.Len(t, cronExprs, 1)
		require.Equal(t, cronExpr, cronExprs[0])
	})

	t.Run("handles various valid cron expressions", func(t *testing.T) {
		testCases := []string{
			"* * * * *",      // every minute
			"0 * * * *",      // every hour
			"0 0 * * *",      // daily at midnight
			"0 0 * * 0",      // weekly on Sunday
			"0 0 1 * *",      // monthly on 1st
			"0 0 1 1 *",      // yearly on Jan 1st
			"*/5 * * * *",    // every 5 minutes
			"0 9-17 * * 1-5", // business hours weekdays
		}

		for _, cronExpr := range testCases {
			t.Run(fmt.Sprintf("cron: %s", cronExpr), func(t *testing.T) {
				f := Function{
					Triggers: []Trigger{
						{
							CronTrigger: &CronTrigger{
								Cron: cronExpr,
							},
						},
					},
				}
				cronExprs := f.ScheduleExpressions()
				require.Len(t, cronExprs, 1)
				require.Equal(t, cronExpr, cronExprs[0])
			})
		}
	})
}

func TestScheduleTriggers(t *testing.T) {
	t.Run("returns parsed cron trigger metadata", func(t *testing.T) {
		jitter := "5m"
		f := Function{
			Triggers: []Trigger{
				{
					EventTrigger: &EventTrigger{
						Event: "user.created",
					},
				},
				{
					CronTrigger: &CronTrigger{
						Cron:   "0 9 * * *",
						Jitter: &jitter,
					},
				},
			},
		}

		triggers := f.ScheduleTriggers()
		require.Len(t, triggers, 1)
		require.Equal(t, "0 9 * * *", triggers[0].Expression)
		require.Equal(t, 5*time.Minute, triggers[0].Jitter)
	})

	t.Run("defaults jitter to zero for unparseable value", func(t *testing.T) {
		jitter := "not-a-duration"
		f := Function{
			Triggers: []Trigger{
				{
					CronTrigger: &CronTrigger{
						Cron:   "0 9 * * *",
						Jitter: &jitter,
					},
				},
			},
		}

		triggers := f.ScheduleTriggers()
		require.Len(t, triggers, 1)
		require.Equal(t, time.Duration(0), triggers[0].Jitter)
	})
}

func TestCronJitter(t *testing.T) {
	jitter5m := "5m"
	jitter1h := "1h"

	f := Function{
		Triggers: []Trigger{
			{CronTrigger: &CronTrigger{Cron: "0 9 * * *", Jitter: &jitter5m}},
			{CronTrigger: &CronTrigger{Cron: "0 * * * *", Jitter: &jitter1h}},
			{CronTrigger: &CronTrigger{Cron: "*/5 * * * *"}},
			{EventTrigger: &EventTrigger{Event: "user.created"}},
		},
	}

	t.Run("returns jitter for matching expression", func(t *testing.T) {
		require.Equal(t, 5*time.Minute, f.CronJitter("0 9 * * *"))
		require.Equal(t, time.Hour, f.CronJitter("0 * * * *"))
	})

	t.Run("returns zero for expression without jitter", func(t *testing.T) {
		require.Equal(t, time.Duration(0), f.CronJitter("*/5 * * * *"))
	})

	t.Run("returns zero for unknown expression", func(t *testing.T) {
		require.Equal(t, time.Duration(0), f.CronJitter("0 0 * * *"))
	})
}

func TestCronTriggerValidate(t *testing.T) {
	ctx := context.Background()

	t.Run("accepts valid jitter", func(t *testing.T) {
		jitter := "5m"
		err := (CronTrigger{Cron: "0 9 * * *", Jitter: &jitter}).Validate(ctx)
		require.NoError(t, err)
	})

	t.Run("rejects invalid jitter", func(t *testing.T) {
		jitter := "oops"
		err := (CronTrigger{Cron: "0 9 * * *", Jitter: &jitter}).Validate(ctx)
		require.Error(t, err)
		require.ErrorContains(t, err, "isn't a valid jitter duration")
	})

	t.Run("rejects negative jitter", func(t *testing.T) {
		jitter := "-1m"
		err := (CronTrigger{Cron: "0 9 * * *", Jitter: &jitter}).Validate(ctx)
		require.Error(t, err)
		require.ErrorContains(t, err, "greater than or equal to zero")
	})

	t.Run("rejects jitter below minimum", func(t *testing.T) {
		jitter := "500ms"
		err := (CronTrigger{Cron: "0 9 * * *", Jitter: &jitter}).Validate(ctx)
		require.Error(t, err)
		require.ErrorContains(t, err, "at least")
	})

	t.Run("rejects jitter above hard cap", func(t *testing.T) {
		jitter := "25h"
		err := (CronTrigger{Cron: "0 9 * * *", Jitter: &jitter}).Validate(ctx)
		require.Error(t, err)
		require.ErrorContains(t, err, "less than or equal to")
	})
}

func TestDuplicateCronExpressionRejected(t *testing.T) {
	ctx := context.Background()

	jitter1 := "5m"
	jitter2 := "10m"
	triggers := MultipleTriggers{
		{CronTrigger: &CronTrigger{Cron: "0 9 * * *", Jitter: &jitter1}},
		{CronTrigger: &CronTrigger{Cron: "0 9 * * *", Jitter: &jitter2}},
	}

	err := triggers.Validate(ctx)
	require.Error(t, err)
	require.ErrorContains(t, err, "duplicate cron expression")
}

func TestLoopJSON(t *testing.T) {
	t.Run("round trip stores durations as strings", func(t *testing.T) {
		in := Loop{
			Kind:    "loop",
			Key:     "event.data.id",
			Tick:    30 * time.Second,
			IdleTTL: 90 * time.Minute,
		}

		byt, err := json.Marshal(in)
		require.NoError(t, err)
		require.Contains(t, string(byt), `"tick":"30s"`)
		require.Contains(t, string(byt), `"idle_ttl":"1h30m"`)
		require.NotContains(t, string(byt), "30000000000")

		out := Loop{}
		require.NoError(t, json.Unmarshal(byt, &out))
		require.Equal(t, in, out)
	})

	t.Run("zero durations are omitted and read back as zero", func(t *testing.T) {
		in := Loop{
			Kind: "fsm",
			Key:  "event.data.id",
		}
		byt, err := json.Marshal(in)
		require.NoError(t, err)
		require.NotContains(t, string(byt), "tick")
		require.NotContains(t, string(byt), "idle_ttl")

		out := Loop{}
		require.NoError(t, json.Unmarshal([]byte(`{"kind":"fsm","key":"event.data.id","tick":"","idle_ttl":""}`), &out))
		require.Equal(t, in, out)
	})

	t.Run("round trips inside a function", func(t *testing.T) {
		fn := Function{
			Name: "loop",
			Loop: &Loop{
				Kind: "agent",
				Key:  "event.data.id",
				Tick: time.Minute,
			},
		}
		byt, err := json.Marshal(fn)
		require.NoError(t, err)
		require.Contains(t, string(byt), `"loop":{"kind":"agent","key":"event.data.id","tick":"1m"}`)

		out := Function{}
		require.NoError(t, json.Unmarshal(byt, &out))
		require.Equal(t, fn.Loop, out.Loop)
	})

	t.Run("invalid duration errors", func(t *testing.T) {
		out := Loop{}
		err := json.Unmarshal([]byte(`{"kind":"loop","key":"event.data.id","tick":"soon"}`), &out)
		require.Error(t, err)
		require.Contains(t, err.Error(), "loop tick")
	})

	t.Run("invalid idle_ttl errors", func(t *testing.T) {
		out := Loop{}
		err := json.Unmarshal([]byte(`{"kind":"loop","key":"event.data.id","idle_ttl":"later"}`), &out)
		require.Error(t, err)
		require.Contains(t, err.Error(), "loop idle_ttl")
	})

	t.Run("numeric tick errors", func(t *testing.T) {
		out := Loop{}
		err := json.Unmarshal([]byte(`{"kind":"loop","key":"event.data.id","tick":30}`), &out)
		require.Error(t, err)
	})

	t.Run("null loop decodes to nil", func(t *testing.T) {
		out := Function{}
		require.NoError(t, json.Unmarshal([]byte(`{"name":"fn","loop":null}`), &out))
		require.Nil(t, out.Loop)
	})
}

func TestValidateLoop(t *testing.T) {
	ctx := context.Background()

	// base returns a valid loop function.  each case changes one thing.
	base := func() Function {
		return Function{
			Name: "loop",
			Triggers: []Trigger{
				{
					EventTrigger: &EventTrigger{
						Event: "loop/start",
					},
				},
			},
			Steps: []Step{
				{
					ID:   "step",
					Name: "Function body",
					URI:  "http://example.com/api/inngest",
				},
			},
			Loop: &Loop{
				Kind:    "loop",
				Key:     "event.data.id",
				Tick:    30 * time.Second,
				IdleTTL: time.Hour,
			},
		}
	}

	t.Run("valid loop passes", func(t *testing.T) {
		f := base()
		require.NoError(t, f.Validate(ctx))
	})

	t.Run("every kind passes", func(t *testing.T) {
		for _, kind := range []string{"loop", "fsm", "agent"} {
			f := base()
			f.Loop.Kind = kind
			require.NoError(t, f.Validate(ctx), kind)
		}
	})

	t.Run("unkeyed step concurrency passes", func(t *testing.T) {
		f := base()
		f.Concurrency = &ConcurrencyLimits{
			Step: []StepConcurrency{
				{
					Limit: 5,
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		}
		require.NoError(t, f.Validate(ctx))
	})

	t.Run("cancel passes", func(t *testing.T) {
		f := base()
		f.Cancel = []Cancel{
			{
				Event: "loop/stop",
			},
		}
		require.NoError(t, f.Validate(ctx))
	})

	t.Run("missing key", func(t *testing.T) {
		f := base()
		f.Loop.Key = ""
		err := f.Validate(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "A loop key is required")
	})

	t.Run("invalid key expression", func(t *testing.T) {
		f := base()
		f.Loop.Key = "event.data.id ==="
		err := f.Validate(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Loop key expression is invalid")
	})

	t.Run("bad kind", func(t *testing.T) {
		f := base()
		f.Loop.Kind = "workflow"
		err := f.Validate(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "The loop kind of 'workflow' is invalid")
	})

	t.Run("negative tick and idle ttl", func(t *testing.T) {
		f := base()
		f.Loop.Tick = -time.Second
		f.Loop.IdleTTL = -time.Second
		err := f.Validate(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "loop tick")
		require.Contains(t, err.Error(), "loop idle_ttl")
	})

	t.Run("sync endpoint rejected", func(t *testing.T) {
		f := base()
		f.Driver = FunctionDriver{
			URI: "https://example.com/api/inngest",
			Metadata: map[string]any{
				"type": "sync",
			},
		}
		err := f.Validate(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "A loop cannot run on a sync endpoint")
	})

	t.Run("async endpoint with driver metadata passes", func(t *testing.T) {
		f := base()
		f.Driver = FunctionDriver{
			URI: "https://example.com/api/inngest",
			Metadata: map[string]any{
				"type": "async",
			},
		}
		require.NoError(t, f.Validate(ctx))
	})

	t.Run("singleton rejected", func(t *testing.T) {
		f := base()
		f.Singleton = &Singleton{
			Key:  new("event.data.id"),
			Mode: enums.SingletonModeSkip,
		}
		err := f.Validate(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Loops and singletons are mutually exclusive")
	})

	t.Run("batching rejected", func(t *testing.T) {
		f := base()
		f.EventBatch = &EventBatchConfig{
			MaxSize: 10,
			Timeout: "10s",
		}
		err := f.Validate(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Loops and batching are mutually exclusive")
	})

	t.Run("debounce rejected", func(t *testing.T) {
		f := base()
		f.Debounce = &Debounce{
			Period: "10s",
		}
		err := f.Validate(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Loops and debouncing are mutually exclusive")
	})

	t.Run("function concurrency rejected", func(t *testing.T) {
		f := base()
		f.Concurrency = &ConcurrencyLimits{
			Fn: []FnConcurrency{
				{
					Limit: 5,
				},
			},
		}
		err := f.Validate(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "A loop cannot set function concurrency")
	})

	t.Run("keyed step concurrency rejected", func(t *testing.T) {
		f := base()
		f.Concurrency = &ConcurrencyLimits{
			Step: []StepConcurrency{
				{
					Limit: 5,
					Key:   new("event.data.id"),
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		}
		err := f.Validate(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Loops and keyed step concurrency limits are mutually exclusive")
	})

	t.Run("keyed legacy concurrency rejected", func(t *testing.T) {
		f := base()
		f.Concurrency = &ConcurrencyLimits{
			Limits: []StepConcurrency{
				{
					Limit: 5,
					Key:   new("event.data.id"),
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		}
		err := f.Validate(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Loops and keyed step concurrency limits are mutually exclusive")
	})
}
