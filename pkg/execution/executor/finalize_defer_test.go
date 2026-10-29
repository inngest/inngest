package executor

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func buildDeferEventsFor(t *testing.T, resp execution.FinalizeResponse, cancel bool) map[string]any {
	t.Helper()

	opts := execution.FinalizeOpts{
		Metadata: sv2.Metadata{ID: sv2.ID{
			RunID:      ulid.Make(),
			FunctionID: uuid.New(),
			Tenant:     sv2.Tenant{AppID: uuid.New(), EnvID: uuid.New(), AccountID: uuid.New()},
		}},
		Response: resp,
		Optional: execution.FinalizeOptional{FnSlug: "parent-fn", Cancel: cancel},
	}
	defers := map[string]sv2.Defer{
		"h1": {
			FnSlug:         "app-child",
			HashedID:       "h1",
			ScheduleStatus: enums.DeferStatusAfterRun,
			Input:          json.RawMessage(`{"a":1}`),
		},
	}

	events, err := (&executor{}).buildDeferEvents(context.Background(), opts, defers)
	require.NoError(t, err)
	require.Len(t, events, 1)

	raw, ok := events[0].Data[consts.InngestEventDataPrefix]
	require.True(t, ok)
	byt, err := json.Marshal(raw)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(byt, &out))
	require.Equal(t, float64(1), events[0].Data["a"])
	return out
}

func TestBuildDeferEventsParentOutcome(t *testing.T) {
	t.Run("completed", func(t *testing.T) {
		got := buildDeferEventsFor(t, execution.FinalizeResponse{
			Type:        execution.FinalizeResponseRunComplete,
			RunComplete: state.GeneratorOpcode{Data: []byte(`"ok"`)},
		}, false)
		require.Equal(t, "Completed", got["parent_status"])
		require.NotContains(t, got, "parent_error")
		require.NotZero(t, got["parent_ended_at"])
	})

	t.Run("failed carries the error", func(t *testing.T) {
		got := buildDeferEventsFor(t, execution.FinalizeResponse{
			Type: execution.FinalizeResponseDriver,
			DriverResponse: state.DriverResponse{
				Err:       new("boom"),
				UserError: &state.UserError{Name: "NonRetriableError", Message: "nope", Stack: "at x"},
			},
		}, false)
		require.Equal(t, "Failed", got["parent_status"])
		require.Equal(t, map[string]any{
			"name":    "NonRetriableError",
			"message": "nope",
			"stack":   "at x",
		}, got["parent_error"])
	})

	t.Run("failed with a standard error", func(t *testing.T) {
		got := buildDeferEventsFor(t, execution.FinalizeResponse{
			Type: execution.FinalizeResponseDriver,
			DriverResponse: state.DriverResponse{
				Err:    new("internal"),
				Output: map[string]any{"name": "Oops", "message": "bad"},
			},
		}, false)
		require.Equal(t, "Failed", got["parent_status"])
		errMap, ok := got["parent_error"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "Oops", errMap["name"])
	})

	t.Run("cancelled", func(t *testing.T) {
		got := buildDeferEventsFor(t, execution.FinalizeResponse{
			Type: execution.FinalizeResponseDriver,
		}, true)
		require.Equal(t, "Cancelled", got["parent_status"])
		require.NotContains(t, got, "parent_error")
	})

	t.Run("oversized error is reduced", func(t *testing.T) {
		got := buildDeferEventsFor(t, execution.FinalizeResponse{
			Type: execution.FinalizeResponseDriver,
			DriverResponse: state.DriverResponse{
				Err: new("boom"),
				UserError: &state.UserError{
					Name:    "Big",
					Message: strings.Repeat("m", consts.MaxDeferParentErrorSize),
					Stack:   strings.Repeat("s", consts.MaxDeferParentErrorSize),
				},
			},
		}, false)
		errMap, ok := got["parent_error"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "Big", errMap["name"])
		require.Equal(t, true, errMap["truncated"])
		require.NotContains(t, errMap, "stack")
		require.Len(t, errMap["message"], consts.MaxDeferParentErrorSize/2)
	})
}

func TestCapDeferParentError(t *testing.T) {
	require.Nil(t, capDeferParentError(nil))

	small := capDeferParentError(map[string]string{"name": "n"})
	require.JSONEq(t, `{"name":"n"}`, string(small))

	multibyte := strings.Repeat("é", consts.MaxDeferParentErrorSize)
	got := capDeferParentError(map[string]string{"name": "n", "message": multibyte})
	require.LessOrEqual(t, len(got), consts.MaxDeferParentErrorSize)
	require.True(t, json.Valid(got))
}

func TestCapDeferParentErrorEscapeExpansion(t *testing.T) {
	got := capDeferParentError(map[string]string{
		"name":    "n",
		"message": strings.Repeat("\x01", consts.MaxDeferParentErrorSize),
	})
	require.LessOrEqual(t, len(got), consts.MaxDeferParentErrorSize)
	require.True(t, json.Valid(got))
}
