package inngest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"testing"

	"github.com/inngest/inngest/pkg/consts"
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
					Limits: []Concurrency{
						{
							Limit: 5,
							Key:   strptr("invalid because not a string"),
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
			Run: strptr("event.data.plan == 'paid' ? 100 : 0"),
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
			Run: strptr("event.data.priority"),
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
			Run: strptr("event.data.priority"),
		}

		pf, err := f.RunPriorityFactor(ctx, map[string]any{
			"data": map[string]any{},
		})
		require.EqualValues(t, 0, pf)
		require.ErrorContains(t, err, "Priority.Run expression returned non-int: false")
	})

	t.Run("With an invalid expression", func(t *testing.T) {
		f.Priority = &Priority{
			Run: strptr("event.data.priority = 123"),
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

func strptr(s string) *string { return &s }
