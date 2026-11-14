package history

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// captureDriver keeps every history row the lifecycle writes.
type captureDriver struct {
	rows []History
}

func (d *captureDriver) Close(context.Context) error { return nil }

func (d *captureDriver) Write(_ context.Context, h History) error {
	d.rows = append(d.rows, h)
	return nil
}

func TestOnFunctionFinishedRunCompleteOutput(t *testing.T) {
	md := sv2.Metadata{
		ID: sv2.ID{
			RunID: ulid.Make(),
		},
		Config: *sv2.InitConfig(&sv2.Config{}),
	}

	t.Run("output comes from the RunComplete opcode", func(t *testing.T) {
		d := &captureDriver{}
		l := NewLifecycleListener(nil, d)

		l.OnFunctionFinished(context.Background(), md, queue.Item{}, nil, state.DriverResponse{
			StatusCode: 206,
			Generator: []*state.GeneratorOpcode{
				{
					ID:   "a",
					Op:   enums.OpcodeStepRun,
					Data: json.RawMessage(`"step"`),
				},
				{
					ID:   "b",
					Op:   enums.OpcodeRunComplete,
					Data: json.RawMessage(`{"name":"","age":1}`),
				},
			},
		})

		require.Len(t, d.rows, 1)
		require.Equal(t, enums.HistoryTypeFunctionCompleted.String(), d.rows[0].Type)
		require.JSONEq(t, `{"name":"","age":1}`, d.rows[0].Result.Output)
		require.Equal(t, len(`{"name":"","age":1}`), d.rows[0].Result.SizeBytes)
	})

	t.Run("a plain response keeps its output", func(t *testing.T) {
		d := &captureDriver{}
		l := NewLifecycleListener(nil, d)

		l.OnFunctionFinished(context.Background(), md, queue.Item{}, nil, state.DriverResponse{
			StatusCode: 200,
			Output: map[string]any{
				"ok": true,
			},
		})

		require.Len(t, d.rows, 1)
		require.JSONEq(t, `{"ok":true}`, d.rows[0].Result.Output)
	})
}
