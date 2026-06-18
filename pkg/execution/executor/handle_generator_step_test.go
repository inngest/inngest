package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type stubRunService struct {
	sv2.RunService
	saveStepErr error
}

func (s *stubRunService) SaveStep(_ context.Context, _ sv2.ID, _ string, _ []byte) (bool, error) {
	return false, s.saveStepErr
}

type stubQueue struct {
	queue.Queue
	enqueued []queue.Item
}

func (q *stubQueue) Enqueue(_ context.Context, item queue.Item, _ time.Time, _ queue.EnqueueOpts) error {
	q.enqueued = append(q.enqueued, item)
	return nil
}

// dedupeQueue mimics queue-level dedup: the first enqueue for a given JobID
// succeeds; subsequent calls for the same ID return ErrQueueItemExists.
type dedupeQueue struct {
	queue.Queue
	mu       sync.Mutex
	enqueued []queue.Item
}

func (q *dedupeQueue) Enqueue(_ context.Context, item queue.Item, _ time.Time, _ queue.EnqueueOpts) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, e := range q.enqueued {
		if e.JobID != nil && item.JobID != nil && *e.JobID == *item.JobID {
			return queue.ErrQueueItemExists
		}
	}
	q.enqueued = append(q.enqueued, item)
	return nil
}

func TestMaybeEnqueueDiscoveryStepCoalesces(t *testing.T) {
	runID := ulid.MustNew(ulid.Now(), nil)
	ck := computeParallelCoalesceKey(runID.String(), []string{"step-a", "step-b"})

	q := &dedupeQueue{}
	e := &executor{
		queue:          q,
		log:            logger.From(context.Background()),
		tracerProvider: tracing.NewNoopTracerProvider(),
	}

	rc := &mockRunContext{
		md: sv2.Metadata{
			ID:     sv2.ID{RunID: runID, FunctionID: uuid.New()},
			Config: *sv2.InitConfig(&sv2.Config{}),
		},
		lifecycleItem: queue.Item{ParallelCoalesceKey: &ck},
	}

	gen := state.GeneratorOpcode{Op: enums.OpcodeStepRun, ID: "step-a"}
	edge := queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}}

	for i := 0; i < 3; i++ {
		err := e.maybeEnqueueDiscoveryStep(context.Background(), rc, gen, edge, uuid.New().String(), false)
		require.NoError(t, err)
	}

	require.Len(t, q.enqueued, 1, "all parallel completions must coalesce to a single discovery enqueue")
	require.Equal(t, fmt.Sprintf("%s-%s-discover", runID, ck), *q.enqueued[0].JobID)
}

// EXE-1625: when the SDK responds with a step that the checkpoint path
// already saved with different bytes, SaveStep returns ErrDuplicateResponse.
// The executor must still enqueue the next discovery edge or the run
// strands with no in-flight queue items.
func TestHandleGeneratorStepEnqueuesDiscoveryOnDuplicate(t *testing.T) {
	q := &stubQueue{}
	e := &executor{
		smv2:           &stubRunService{saveStepErr: state.ErrDuplicateResponse},
		queue:          q,
		log:            logger.From(context.Background()),
		tracerProvider: tracing.NewNoopTracerProvider(),
	}

	rc := &mockRunContext{md: sv2.Metadata{
		ID:     sv2.ID{RunID: ulid.MustNew(ulid.Now(), nil), FunctionID: uuid.New()},
		Config: *sv2.InitConfig(&sv2.Config{}),
	}}

	err := e.handleGeneratorStep(
		context.Background(),
		rc,
		state.GeneratorOpcode{Op: enums.OpcodeStepRun, ID: "step", Data: json.RawMessage(`null`)},
		queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}},
	)
	require.NoError(t, err)
	require.Len(t, q.enqueued, 1, "expected discovery enqueue despite duplicate save")
}
