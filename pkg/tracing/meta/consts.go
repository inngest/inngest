package meta

const (
	// Implementation
	PropagationKey = "user-otel-ctx"

	// Top-level span names
	SpanNameRun              = "executor.run"
	SpanNameStepDiscovery    = "executor.step.discovery"
	SpanNameStep             = "executor.step"
	SpanNameExecution        = "executor.execution"
	SpanNameDynamicExtension = "EXTEND"

	// Run attributes
	AttributeAccountID        = "_inngest.account.id"
	AttributeAppID            = "_inngest.app.id"
	AttributeBatchID          = "_inngest.batch.id"
	AttributeBatchTimestamp   = "_inngest.batch.ts"
	AttributeCronSchedule     = "_inngest.cron.schedule"
	AttributeDropSpan         = "_inngest.executor.drop"
	AttributeEnvID            = "_inngest.env.id"
	AttributeEventIDs         = "_inngest.event.ids"
	AttributeFunctionID       = "_inngest.function.id"
	AttributeFunctionVersion  = "_inngest.function.version"
	AttributeInternalLocation = "_inngest.internal.location"
	AttributeRunID            = "_inngest.run.id"

	// Dynamic span controls
	AttributeDynamicSpanID = "_inngest.dynamic.span.id"
	AttributeDynamicStatus = "_inngest.dynamic.status"

	// Link attributes
	LinkAttributeType            = "_inngest.link.type"
	LinkAttributeTypeFollowsFrom = "follows_from"

	// Generic step attributes
	AttributeStepID          = "_inngest.step.id"
	AttributeStepName        = "_inngest.step.name"
	AttributeStepOp          = "_inngest.step.op"
	AttributeStepAttempt     = "_inngest.step.attempt"
	AttributeStepMaxAttempts = "_inngest.step.max_attempts"

	// Invoke attributes
	AttributeStepInvokeExpiry         = "_inngest.step.invoke_function.expiry"
	AttributeStepInvokeFunctionID     = "_inngest.step.invoke_function.id"
	AttributeStepInvokeTriggerEventID = "_inngest.step.invoke_function.trigger_event_id"

	// Sleep attributes
	AttributeStepSleepDuration = "_inngest.step.sleep.duration"

	// WaitForEvent attributes
	AttributeStepWaitForEventExpiry = "_inngest.step.wait_for_event.expiry"
	AttributeStepWaitForEventIf     = "_inngest.step.wait_for_event.if"
	AttributeStepWaitForEventName   = "_inngest.step.wait_for_event.name"

	// HTTP (serve) attributes
	AttributeResponseHeaders    = "_inngest.response.headers"
	AttributeResponseStatusCode = "_inngest.response.status_code"
	AttributeResponseOutputSize = "_inngest.response.output_size"
)
