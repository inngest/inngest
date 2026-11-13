package executor

import (
	"testing"

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

func TestSleepIDsSeparateRuns(t *testing.T) {
	q := &dedupeQueue{}
	e := &executor{queue: q, log: logger.From(t.Context()), tracerProvider: tracing.NewNoopTracerProvider()}
	rc := &mockRunContext{md: sv2.Metadata{ID: sv2.ID{RunID: ulid.Make(), FunctionID: uuid.New(), Tenant: sv2.Tenant{AccountID: uuid.New(), EnvID: uuid.New()}}, Config: *sv2.InitConfig(&sv2.Config{Idempotency: "reused-event-id"})}}
	gen := state.GeneratorOpcode{Op: enums.OpcodeSleep, ID: "same-step", Name: "1h"}
	for range 2 {
		require.NoError(t, e.handleGeneratorSleep(t.Context(), rc, gen, queue.PayloadEdge{}, OpcodeGroup{}))
	}
	require.Len(t, q.enqueued, 1, "retry in the same run must not schedule another wake-up")
	rc.md.ID.RunID = ulid.Make()
	require.NoError(t, e.handleGeneratorSleep(t.Context(), rc, gen, queue.PayloadEdge{}, OpcodeGroup{}))
	require.Len(t, q.enqueued, 2, "a separate run with the same event key needs its own wake-up")
	require.NotEqual(t, *q.enqueued[0].JobID, *q.enqueued[1].JobID)
}
