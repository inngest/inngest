package executor

import (
	"context"
	"encoding/json"
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
