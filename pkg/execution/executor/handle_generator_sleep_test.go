package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type sleepQueue struct {
	dedupeQueue
	opts queue.EnqueueOpts
	err  error
}

func (q *sleepQueue) Enqueue(ctx context.Context, item queue.Item, at time.Time, opts queue.EnqueueOpts) error {
	q.opts = opts
	if q.err != nil {
		return q.err
	}
	return q.dedupeQueue.Enqueue(ctx, item, at, opts)
}

func TestSleepIDsSeparateRuns(t *testing.T) {
	q := &sleepQueue{}
	e := &executor{queue: q, log: logger.From(t.Context()), tracerProvider: tracing.NewNoopTracerProvider()}
	rc := &mockRunContext{md: sv2.Metadata{ID: sv2.ID{RunID: ulid.Make(), FunctionID: uuid.New(), Tenant: sv2.Tenant{AccountID: uuid.New(), EnvID: uuid.New()}}, Config: *sv2.InitConfig(&sv2.Config{Idempotency: "reused-event-id"})}}
	gen := state.GeneratorOpcode{Op: enums.OpcodeSleep, ID: "same-step", Name: "1h"}
	for range 2 {
		require.NoError(t, e.handleGeneratorSleep(t.Context(), rc, gen, queue.PayloadEdge{}, OpcodeGroup{}))
	}
	require.Len(t, q.enqueued, 1, "retry in the same run must not schedule another wake-up")
	legacyID := q.opts.LegacySleepID
	require.NotEmpty(t, legacyID)
	rc.md.ID.RunID = ulid.Make()
	require.NoError(t, e.handleGeneratorSleep(t.Context(), rc, gen, queue.PayloadEdge{}, OpcodeGroup{}))
	require.Len(t, q.enqueued, 2, "a separate run with the same event key needs its own wake-up")
	require.NotEqual(t, *q.enqueued[0].JobID, *q.enqueued[1].JobID)
	require.Equal(t, legacyID, q.opts.LegacySleepID)
	q.err = errors.New("ambiguous legacy completion")
	require.ErrorIs(t, e.handleGeneratorSleep(t.Context(), rc, gen, queue.PayloadEdge{}, OpcodeGroup{}), q.err)
}
