package history

/*
import (
	"context"
	"crypto/rand"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
)

type historyExecutor struct {
	exec    executor.Executor
	drivers []Driver
	// TODO: Logger.
}

func (he historyExecutor) write(ctx context.Context, h History) {
	for _, d := range he.drivers {
		go func() {
			if err := d.Write(ctx, h); err != nil {
				// TODO: Log error
				_ = err
			}
		}()
	}
}

func (he historyExecutor) Execute(
	ctx context.Context,
	id state.Identifier,
	edge inngest.Edge,
	attempt int,
	stackIndex int,
) (*state.DriverResponse, int, error) {
	// TODO: Write started.
	h := History{
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		CreatedAt:       time.Now(),
		RunStartedAt:    ulid.Time(id.RunID.Time()),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		RunID:           id.RunID,
		Type:            enums.HistoryTypeStepStarted.String(),
		Attempt:         int64(attempt),
		IdempotencyKey:  id.IdempotencyKey(),
		StepName:        &edge.Incoming,
		StepID:          &edge.Incoming, // TODO: Add step name to edge.
		// EventID: id.EventID(), // TODO: Add to ID
		// BatchID: id.EventID(), // TODO: Add to ID
		// URL: ,??? // TODO: Add URL.  What if this changes???
	}

	he.write(ctx, h)

	resp, i, err := he.exec.Execute(ctx, id, edge, attempt, stackIndex)

	if err == nil {
		// result := HistoryResult{}
		h.Type = enums.HistoryTypeStepCompleted.String()
		h.ID = ulid.MustNew(ulid.Now(), rand.Reader)
		h.CreatedAt = time.Now()
	}

	if err != nil {
		h.Typ
	}

	he.write(ctx, h)

	// TODO: Write ended
	return resp, i, err
}

func (he historyExecutor) HandleGenerator(ctx context.Context, gen state.GeneratorOpcode, item queue.Item) error {
	err := he.exec.HandleGenerator(ctx, gen, item)
	if err != nil {
		return err
	}
	// TODO: Write history.
}

func (he historyExecutor) Cancel(ctx context.Context, id state.Identifier, r CancelRequest) error {
	err := he.Cancel(ctx, id, r)
	if err != nil {
		return err
	}
	// TODO: write history
}

func (he historyExecutor) Resume(ctx context.Context, p state.Pause, r ResumeRequest) error {
	err := he.Resume(ctx, p, r)
	if err != nil {
		return err
	}
	// TODO: write history
}
*/
