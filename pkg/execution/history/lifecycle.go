package history

import (
	"context"
	"crypto/rand"
	"errors"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"golang.org/x/exp/slog"
)

func NewLifecycleListener(l *slog.Logger, d ...Driver) execution.LifecycleListener {
	if l == nil {
		l = slog.Default()
	}
	return lifecycle{
		log:     l,
		drivers: d,
	}
}

type lifecycle struct {
	log     *slog.Logger
	drivers []Driver
}

func (l lifecycle) Close() error {
	var err error
	for _, d := range l.drivers {
		err = errors.Join(err, d.Close())
	}
	return err
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
		StepID:          &edge.Incoming, // TODO: Add step name to edge.
		EventID:         id.EventID,
		BatchID:         id.BatchID,
		URL:             &step.URI,
	}

	for _, d := range l.drivers {
		if err := d.Write(ctx, h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onStepStarted", "error", err)
		}
	}
}

func (l lifecycle) OnStepFinished(
	ctx context.Context,
	id state.Identifier,
	item queue.Item,
	edge inngest.Edge,
	step inngest.Step,
	resp state.DriverResponse,
) {

	h := History{
		ID:              ulid.MustNew(ulid.Now(), rand.Reader),
		CreatedAt:       time.Now(),
		FunctionID:      id.WorkflowID,
		FunctionVersion: int64(id.WorkflowVersion),
		RunID:           id.RunID,
		Type:            enums.HistoryTypeStepCompleted.String(),
		Attempt:         int64(item.Attempt),
		IdempotencyKey:  id.IdempotencyKey(),
		StepName:        &resp.Step.Name,
		StepID:          &edge.Incoming,
		EventID:         id.EventID,
		BatchID:         id.BatchID,
		URL:             &step.URI,
		Result: &Result{
			Output:     resp.Output,
			DurationMS: int(resp.Duration.Milliseconds()),
			SizeBytes:  resp.OutputSize,
			// XXX: Add more fields here
		},
	}

	if resp.Err != nil && resp.Retryable() {
		h.Type = enums.HistoryTypeStepErrored.String()
	}
	if resp.Err != nil && !resp.Retryable() {
		h.Type = enums.HistoryTypeStepFailed.String()
	}

	for _, d := range l.drivers {
		if err := d.Write(ctx, h); err != nil {
			l.log.Error("execution lifecycle error", "lifecycle", "onStepFinished", "error", err)
		}
	}
}

func (l lifecycle) OnFunctionFailed(
	ctx context.Context,
	id state.Identifier,
	resp state.DriverResponse,
) {
}

func (l lifecycle) OnWaitForEvent(
	context.Context,
	state.Identifier,
	queue.Item,
	state.GeneratorOpcode,
) {
	// nothing right now.
}
