package devserver

import (
	"context"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/pubsub"
	"github.com/oklog/ulid/v2"
)

type lifecycle struct {
	execution.NoopLifecyceListener

	cqrs       cqrs.Manager
	pb         pubsub.Publisher
	eventTopic string
}

func (l lifecycle) OnFunctionScheduled(
	ctx context.Context,
	md state.Metadata,
	item queue.Item,
) {
	_ = l.cqrs.InsertFunctionRun(ctx, cqrs.FunctionRun{
		RunID:         md.ID.RunID,
		RunStartedAt:  ulid.Time(md.ID.RunID.Time()),
		FunctionID:    md.ID.FunctionID,
		EventID:       md.Config.FirstEventID(),
		Cron:          md.Config.CronSchedule(),
		OriginalRunID: md.Config.OriginalRunID,
	})

	/*
		if id.BatchID != nil {
			executedTime := ulid.Time(id.RunID.Time())

			batch := cqrs.NewEventBatch(
				cqrs.WithEventBatchID(*id.BatchID),
				cqrs.WithEventBatchAccountID(id.AccountID),
				cqrs.WithEventBatchWorkspaceID(id.WorkspaceID),
				cqrs.WithEventBatchAppID(id.AppID),
				cqrs.WithEventBatchFunctionID(id.WorkflowID),
				cqrs.WithEventBatchRunID(id.RunID),
				cqrs.WithEventBatchEventIDs(id.EventIDs),
				cqrs.WithEventBatchExecutedTime(executedTime),
			)

			if batch.IsMulti() {
				_ = l.cqrs.InsertEventBatch(ctx, *batch)
			}
		}
	*/
}
