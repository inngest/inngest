package inngest

import (
	"context"
	"encoding/json"
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

	actual, err := fn.URI()
	require.NoError(t, err)
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

func strptr(s string) *string { return &s }
