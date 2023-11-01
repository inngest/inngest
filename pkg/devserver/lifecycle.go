package devserver

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
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

func (l lifecycle) OnFunctionFinished(
	ctx context.Context,
	id state.Identifier,
	item queue.Item,
	resp state.DriverResponse,
	s state.State,
) {
	now := time.Now()
	data := map[string]interface{}{
		"function_id": s.Function().Slug,
		"run_id":      id.RunID.String(),
	}

	origEvt := s.Event()

	if dataMap, ok := origEvt["data"].(map[string]interface{}); ok {
		if inngestObj, ok := dataMap[consts.InngestEventDataPrefix].(map[string]interface{}); ok {

			if dataValue, ok := inngestObj[consts.InvokeCorrelationId].(string); ok {
				logger.From(ctx).Debug().Str("data_value_str", dataValue).Msg("data_value")
				data[consts.InvokeCorrelationId] = dataValue
			}
		}
	}

	if resp.Err != nil {
		data["error"] = resp.UserError()
	} else {
		data["result"] = resp.Output
	}

	evt := event.Event{
		ID:        ulid.MustNew(uint64(now.UnixMilli()), rand.Reader).String(),
		Name:      event.FnFinishedName,
		Timestamp: now.UnixMilli(),
		Data:      data,
	}

	logger.From(ctx).Debug().Interface("event", evt).Msg("function completed event")

	byt, err := json.Marshal(evt)
	if err != nil {
		logger.From(ctx).Error().Err(err).Msg("error marshalling function completion event")
		return
	}

	_ = l.pb.Publish(
		ctx,
		l.eventTopic,
		pubsub.Message{
			Name:      event.EventReceivedName,
			Data:      string(byt),
			Timestamp: now,
		},
	)
}
