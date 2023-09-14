package inngest

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultEdges(t *testing.T) {
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

	edges, err := fn.AllEdges(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, len(edges))
	require.Equal(t, Edge{
		Outgoing: TriggerName,
		Incoming: "step",
	}, edges[0])
}

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
			require.Contains(t, err.Error(), "Non-HTTP steps are not yet supported")
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
