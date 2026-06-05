package apiv1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCheckpointAsyncSteps_GenerationIDUnmarshals defends the JSON tag on
// GenerationID. The SDK sends `generation_id`; if that tag drifts the
// dispatch fence silently fails open and stale dispatches go unfenced.
func TestCheckpointAsyncSteps_GenerationIDUnmarshals(t *testing.T) {
	body := []byte(`{
		"run_id": "01HW9YQX2SQXVK6K4RHKZG4Z6N",
		"fn_id": "11111111-1111-1111-1111-111111111111",
		"qi_id": "job-123:shard-1",
		"request_id": "01HW9YQX2SQXVK6K4RHKZG4Z6P",
		"generation_id": 5,
		"steps": [],
		"ts": 1700000000000,
		"request_started_at": 1700000000123
	}`)

	var got checkpointAsyncSteps
	require.NoError(t, json.Unmarshal(body, &got))
	require.Equal(t, 5, got.GenerationID, "generation_id must populate GenerationID")
}
