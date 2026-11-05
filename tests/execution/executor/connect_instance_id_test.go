package executor

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runConnectInstanceScenario runs one request whose driver response reports
// the given opcode and connect worker instance ID.  it returns the persisted
// executor.step span for the opcode.
func runConnectInstanceScenario(t *testing.T, op *state.GeneratorOpcode, instanceID string) *cqrs.OtelSpan {
	t.Helper()

	infra := newExecTestInfra(t, "step")
	ctx := infra.ctx

	mockDriver := &mockDriverV1{
		t: t,
		response: &state.DriverResponse{
			StatusCode:     206,
			RequestVersion: 2,
			Generator:      []*state.GeneratorOpcode{op},
			InstanceID:     instanceID,
		},
	}

	exec := infra.newExecutor(t,
		executor.WithInvokeEventHandler(func(ctx context.Context, evt event.TrackedEvent) error { return nil }),
		executor.WithDriverV1(mockDriver),
	)

	run := infra.scheduleRun(t, exec)

	jobs, err := infra.rq.RunJobs(ctx, infra.queueShard.Name(), queue.Scope{
		AccountID:  run.ID.Tenant.AccountID,
		EnvID:      run.ID.Tenant.EnvID,
		FunctionID: run.ID.FunctionID,
	}, run.ID.RunID, 1000, 0)
	require.NoError(t, err)
	require.NotEmpty(t, jobs)

	rawItem, ok := jobs[0].Raw.(*queue.QueueItem)
	require.True(t, ok, "RunJobs Raw must be a *queue.QueueItem, got %T", jobs[0].Raw)

	jobCtx := queue.WithJobID(ctx, jobs[0].JobID)
	id := sv2.V1FromMetadata(*run)
	edge := inngest.Edge{
		Incoming: "$trigger",
		Outgoing: "step",
	}

	_, err = exec.Execute(jobCtx, id, rawItem.Data, edge)
	require.NoError(t, err)

	var span *cqrs.OtelSpan
	require.Eventually(t, func() bool {
		root, err := infra.dbcqrs.GetSpansByRunID(ctx, run.ID.RunID)
		if err != nil || root == nil {
			return false
		}
		span = findStepSpan(root, op.ID)
		return span != nil && span.Attributes != nil
	}, 10*time.Second, 100*time.Millisecond, "the step span for the opcode must be persisted")

	return span
}

func TestStepSpansCarryConnectInstanceID(t *testing.T) {
	for _, tc := range []struct {
		name string
		op   *state.GeneratorOpcode
	}{
		{
			name: "sleep",
			op: &state.GeneratorOpcode{
				Op:          enums.OpcodeSleep,
				ID:          "c2640f79b4ed481b838ce4ad75330aa3f825d4d9",
				Name:        "5s",
				Opts:        map[string]any{},
				DisplayName: new("nap"),
			},
		},
		{
			name: "step run",
			op: &state.GeneratorOpcode{
				Op:          enums.OpcodeStepRun,
				ID:          "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3",
				Name:        "work",
				Data:        json.RawMessage(`"done"`),
				DisplayName: new("work"),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("with instance ID", func(t *testing.T) {
				span := runConnectInstanceScenario(t, tc.op, "worker-1")
				require.NotNil(t, span.Attributes.ConnectInstanceID)
				assert.Equal(t, "worker-1", *span.Attributes.ConnectInstanceID)
			})

			t.Run("without instance ID", func(t *testing.T) {
				span := runConnectInstanceScenario(t, tc.op, "")
				assert.Nil(t, span.Attributes.ConnectInstanceID)
			})
		})
	}
}
