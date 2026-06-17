package queue

import itrace "github.com/inngest/inngest/pkg/telemetry/trace"

func TraceScopeFromPartitionIdentifier(id PartitionIdentifier) itrace.Scope {
	if id.SystemQueueName != nil {
		return itrace.SystemScope{QueueName: id.SystemQueueName}
	}

	return itrace.UserScope{
		AccountID: id.AccountID,
		EnvID:     id.EnvID,
		FnID:      id.FunctionID,
	}
}

func TraceScopeFromQueueItem(i QueueItem) itrace.Scope {
	queueName := i.QueueName
	if queueName == nil {
		queueName = i.Data.QueueName
	}
	if queueName != nil {
		return itrace.SystemScope{QueueName: queueName}
	}

	return itrace.UserScope{
		AccountID: i.Data.Identifier.AccountID,
		EnvID:     i.Data.Identifier.WorkspaceID,
		FnID:      i.FunctionID,
	}
}
