package history

import (
	"context"
	"crypto/rand"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
)

type lifecycle struct {
	drivers []Driver
}

func (l lifecycle) OnStepStarted(
	ctx context.Context,
	id state.Identifier,
	item queue.Item,
	edge inngest.Edge,
	step inngest.Step,
	state state.State,
) {

	h := History{
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		RunID:           id.RunID,
		Type:            enums.HistoryTypeStepStarted.String(),
		Attempt:         int64(item.Attempt),
		IdempotencyKey:  id.IdempotencyKey(),
		StepName:        &edge.Incoming,
		StepID:          edge.Incoming, // TODO: Add step name to edge.
		// EventID: id.EventID(), // TODO: Add to ID
		// BatchID: id.EventID(), // TODO: Add to ID
		URL: &step.URI,
	}

	for _, d := range l.drivers {
		if err := d.Write(ctx, h); err != nil {
			// TODO: Log
		}
	}
}

func (l lifecycle) OnStepFinished(
	context.Context,
	state.Identifier,
	queue.Item,
	inngest.Step,
	*state.DriverResponse,
	error,
) {
}

func (l lifecycle) OnWaitForEvent(
	context.Context,
	state.Identifier,
	queue.Item,
	state.GeneratorOpcode,
) {
}
