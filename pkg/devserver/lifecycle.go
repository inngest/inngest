package devserver

import (
	"context"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/pubsub"
	"github.com/oklog/ulid/v2"
)

type lifecycle struct {
	execution.NoopLifecyceListener

	sm         state.Manager
	cqrs       cqrs.Manager
	pb         pubsub.Publisher
	eventTopic string
}

func (l lifecycle) OnFunctionScheduled(
	ctx context.Context,
	id state.Identifier,
	item queue.Item,
	event event.Event,
) {
	state, err := l.sm.Load(ctx, id.RunID)
	if err != nil {
		return
	}

	evt := state.Event()

	triggerType := "event"
	if name, _ := evt["name"].(string); name == "inngest/scheduled.timer" {
		triggerType = "cron"
	}

	_ = l.cqrs.InsertFunctionRun(ctx, cqrs.FunctionRun{
		RunID:        id.RunID,
		RunStartedAt: ulid.Time(id.RunID.Time()),
		FunctionID:   id.WorkflowID,
		EventID:      id.EventID,
		TriggerType:  triggerType,
		Cron:         event.Cron,
	})
}
