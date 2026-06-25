package queue

import itrace "github.com/inngest/inngest/pkg/telemetry/trace"

func TraceScopeFromQueueItem(i QueueItem, queueShardName string) itrace.Scope {
	queueName := i.QueueName
	if queueName == nil {
		queueName = i.Data.QueueName
	}
	if queueName != nil {
		return itrace.SystemScope{
			QueueName:      queueName,
			QueueShardName: queueShardName,
			ItemKind:       i.Data.Kind,
		}
	}

	return itrace.UserScope{
		AccountID: i.Data.Identifier.AccountID,
		EnvID:     i.Data.Identifier.WorkspaceID,
		FnID:      i.FunctionID,
	}
}
