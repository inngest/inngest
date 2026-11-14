package executor

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/loops"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/stretchr/testify/require"
)

// loopDriverV1 records the loop block that run puts on the context.  it
// answers with a retryable error, so the run stays open and Execute never
// reaches Finalize.
type loopDriverV1 struct {
	calls int
	seen  *driver.SDKLoopRequest
}

func (d *loopDriverV1) Name() string {
	return "http"
}

func (d *loopDriverV1) Execute(ctx context.Context, _ sv2.StateLoader, _ sv2.Metadata, _ queue.Item, _ inngest.Edge, step inngest.Step, _ int, _ int) (*state.DriverResponse, error) {
	d.calls++
	d.seen = driver.LoopFromContext(ctx)
	msg := "loop driver stub"
	return &state.DriverResponse{
		Step: step,
		Err:  &msg,
	}, nil
}

// loopDriverV2 stands in for the sync driver.  a loop run must fail before
// it is called.
type loopDriverV2 struct {
	calls int
}

func (d *loopDriverV2) Name() string {
	return "httpv2"
}

func (d *loopDriverV2) Do(context.Context, sv2.StateLoader, driver.V2RequestOpts) (*state.DriverResponse, errs.UserError, errs.InternalError) {
	d.calls++
	return nil, nil, nil
}

// execute runs the one queued start item of the run through Execute, the
// way the executor service does.
func (h *loopHarness) execute(t *testing.T, id sv2.ID) (*state.DriverResponse, error) {
	t.Helper()
	scope := queue.Scope{
		AccountID:  id.Tenant.AccountID,
		EnvID:      id.Tenant.EnvID,
		FunctionID: id.FunctionID,
	}
	items, err := h.q.ItemsByRunID(h.ctx, h.shard, scope, id.RunID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	item := items[0].Data
	edge, err := queue.GetEdge(item)
	require.NoError(t, err)
	return h.e.Execute(h.ctx, item.Identifier, item, edge.Edge)
}

// openAndCommit schedules the trigger run, which opens the instance, and
// commits one state so the next request carries a version above zero.
func (h *loopHarness) openAndCommit(t *testing.T) {
	t.Helper()
	first, firstID := h.request(nil)
	_, _, err := h.e.Schedule(h.ctx, first)
	require.NoError(t, err)
	_, err = h.store.Commit(h.ctx, loops.Commit{
		ID:      h.loopID("user-1"),
		RunID:   firstID.RunID,
		Counter: 0,
		State:   json.RawMessage(`{"n":1}`),
	})
	require.NoError(t, err)
}

func TestExecuteLoopRequest(t *testing.T) {
	t.Run("async driver sees the loop block", func(t *testing.T) {
		h := newLoopHarness(t)
		d := &loopDriverV1{}
		h.e.driverv1 = map[string]driver.DriverV1{
			d.Name(): d,
		}
		h.openAndCommit(t)

		req, id := h.request(new(loops.NewWakeEvent("user-1", "wake-1")))
		_, _, err := h.e.Schedule(h.ctx, req)
		require.NoError(t, err)

		resp, err := h.execute(t, id)
		require.ErrorContains(t, err, "loop driver stub")
		require.NotNil(t, resp)
		require.Equal(t, 1, d.calls)
		require.NotNil(t, d.seen)
		require.JSONEq(t, `{"n":1}`, string(d.seen.State))
		require.Equal(t, int64(1), d.seen.Version)
		require.Equal(t, "user-1", d.seen.Key)
		require.Equal(t, driver.SDKLoopInput{
			Kind: loops.InputWake,
			ID:   "wake-1",
		}, d.seen.Input)
	})

	t.Run("sync driver fails without retry", func(t *testing.T) {
		h := newLoopHarness(t)
		d := &loopDriverV2{}
		h.e.driverv2 = map[string]driver.DriverV2{
			d.Name(): d,
		}
		fn := h.req.Function
		fn.Driver.Metadata = map[string]any{
			"type": "sync",
		}
		h.e.fl = singletonRetryFunctionLoader{fn: &fn}
		h.openAndCommit(t)

		req, id := h.request(new(loops.NewTickEvent("user-1")))
		_, _, err := h.e.Schedule(h.ctx, req)
		require.NoError(t, err)

		resp, err := h.execute(t, id)
		require.ErrorContains(t, err, "sync endpoint")
		require.False(t, queue.ShouldRetry(err, 0, 3))
		require.Nil(t, resp)
		require.Equal(t, 0, d.calls)
	})
}
