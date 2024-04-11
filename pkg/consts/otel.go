package consts

const (
	OtelSpanEvent   = "event"
	OtelSpanCron    = "cron"
	OtelSpanTrigger = "trigger"

	// system attributes
	OtelSysAccountID      = "sys.account.id"
	OtelSysWorkspaceID    = "sys.workspace.id"
	OtelSysAppID          = "sys.app.id"
	OtelSysIdempotencyKey = "sys.idempotency.key"

	OtelSysEventData       = "sys.event"
	OtelSysEventRequestID  = "sys.event.request.id"
	OtelSysEventInternalID = "sys.event.internal.id"
	OtelSysEventIDs        = "sys.event.ids"
	OtelSysBatchID         = "sys.batch.id"

	OtelSysFunctionID         = "sys.function.id"
	OtelSysFunctionSlug       = "sys.function.slug"
	OtelSysFunctionVersion    = "sys.function.version"
	OtelSysFunctionScheduleAt = "sys.function.time.schedule"
	OtelSysFunctionStartAt    = "sys.function.time.start"
	OtelSysFunctionEndAt      = "sys.function.time.end"
	OtelSysFunctionStatus     = "sys.function.status"
	OtelSysFunctionStatusCode = "sys.function.status.code"
	OtelSysFunctionOutput     = "sys.function.output"

	OtelSysStepScheduleAt      = "sys.step.time.schedule"
	OtelSysStepStartAt         = "sys.step.time.start"
	OtelSysStepEndAt           = "sys.step.time.end"
	OtelSysStepStatus          = "sys.step.status"
	OtelSysStepStatusCode      = "sys.step.status.code"
	OtelSysStepAttempt         = "sys.step.attempt"
	OtelSysStepOutput          = "sys.step.output"
	OtelSysStepOutputSizeBytes = "sys.step.output.size.bytes"
	OtelSysStepFirst           = "sys.step.first"
	OtelSysStepGroupID         = "sys.step.group.id"

	OtelSysStepRetry         = "sys.step.retry"
	OtelSysStepNextOpcode    = "sys.step.next.opcode"
	OtelSysStepNextTimestamp = "sys.step.next.time"
	OtelSysStepNextExpires   = "sys.step.next.expires"
	OtelSysStepDelete        = "sys.step.delete"

	OtelSysCronTimestamp = "sys.cron.timestamp"
	OtelSysCronExpr      = "sys.cron.expr"

	// tracking delays
	OtelSysDelaySystem  = "sys.delay.system.ms"
	OtelSysDelaySojourn = "sys.delay.sojourn.ms"

	// SDK attributes
	OtelAttrSDKServiceName = "sdk.app.id"
	OtelAttrSDKRunID       = "sdk.run.id"

	// otel scopes
	OtelScopeEvent     = "event.inngest"
	OtelScopeTrigger   = "trigger.inngest"
	OtelScopeCron      = "cron.inngest"
	OtelScopeEnv       = "env.inngest"
	OtelScopeApp       = "app.env.inngest"
	OtelScopeFunction  = "function.app.env.inngest"
	OtelScopeStep      = "step.function.app.env.inngest"
	OtelScopeExecution = "execution.function.app.env.inngest"

	// otel collector filter keys
	OtelUserTraceFilterKey = "inngest.user"

	OtelPropagationKey = "sys.trace"
)
