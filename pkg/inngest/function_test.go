package inngest

import (
	"context"
	"encoding/json"
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
