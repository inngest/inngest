package tracing

const (
	SpanNameRun       = "executor.run"
	SpanNameStep      = "executor.step"
	SpanNameExecution = "executor.execution"

	AttributeRunID            = "_inngest.run.id"
	AttributeFunctionID       = "_inngest.function.id"
	AttributeFunctionVersion  = "_inngest.function.version"
	AttributeEventIDs         = "_inngest.event.ids"
	AttributeAccountID        = "_inngest.account.id"
	AttributeEnvID            = "_inngest.env.id"
	AttributeAppID            = "_inngest.app.id"
	AttributeCronSchedule     = "_inngest.cron.schedule"
	AttributeBatchID          = "_inngest.batch.id"
	AttributeBatchTimestamp   = "_inngest.batch.ts"
	AttributeDropSpan         = "_inngest.executor.drop"
	AttributeDynamicDuration  = "_inngest.dynamic.duration"
	AttributeInternalLocation = "_inngest.internal.location"
)
