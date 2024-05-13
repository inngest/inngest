package consts

const (
	OtelSpanEvent    = "event"
	OtelSpanCron     = "cron"
	OtelSpanBatch    = "batch"
	OtelSpanDebounce = "debounce"
	OtelSpanTrigger  = "trigger"

	// system attributes
	OtelSysAccountID      = "sys.account.id"
	OtelSysWorkspaceID    = "sys.workspace.id"
	OtelSysAppID          = "sys.app.id"
	OtelSysIdempotencyKey = "sys.idempotency.key"

	OtelSysEventData       = "sys.event"
	OtelSysEventRequestID  = "sys.event.request.id"
	OtelSysEventInternalID = "sys.event.internal.id"
	OtelSysEventIDs        = "sys.event.ids"

	OtelSysBatchID      = "sys.batch.id"
	OtelSysBatchFull    = "sys.batch.full"
	OtelSysBatchTimeout = "sys.batch.timeout"

	OtelSysDebounceID      = "sys.debounce.id"
	OtelSysDebounceTimeout = "sys.debounce.timeout"

	OtelSysFunctionID         = "sys.function.id"
	OtelSysFunctionSlug       = "sys.function.slug"
	OtelSysFunctionVersion    = "sys.function.version"
	OtelSysFunctionScheduleAt = "sys.function.time.schedule"
	OtelSysFunctionStartAt    = "sys.function.time.start"
	OtelSysFunctionEndAt      = "sys.function.time.end"
	OtelSysFunctionStatusCode = "sys.function.status.code"
	OtelSysFunctionOutput     = "sys.function.output"

	OtelSysStepDisplayName     = "sys.step.display.name"
	OtelSysStepOpcode          = "sys.step.opcode"
	OtelSysStepScheduleAt      = "sys.step.time.schedule"
	OtelSysStepStartAt         = "sys.step.time.start"
	OtelSysStepEndAt           = "sys.step.time.end"
	OtelSysStepStatus          = "sys.step.status"
	OtelSysStepStatusCode      = "sys.step.status.code"
	OtelSysStepAttempt         = "sys.step.attempt"
	OtelSysStepMaxAttempt      = "sys.step.attempt.max"
	OtelSysStepOutput          = "sys.step.output"
	OtelSysStepOutputSizeBytes = "sys.step.output.size.bytes"
	OtelSysStepFirst           = "sys.step.first"
	OtelSysStepGroupID         = "sys.step.group.id"

	OtelSysStepSleepEndAt = "sys.step.sleep.end"

	OtelSysStepInvokeExpires           = "sys.step.invoke.expires"
	OtelSysStepInvokeTargetFnID        = "sys.step.invoke.fn.id"
	OtelSysStepInvokeTriggeringEventID = "sys.step.invoke.event.outgoing.id"
	OtelSysStepInvokeReturnedEventID   = "sys.step.invoke.event.incoming.id"
	OtelSysStepInvokeRunID             = "sys.step.invoke.run.id"
	OtelSysStepInvokeExpired           = "sys.step.invoke.expired"

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
	OtelScopeBatch     = "batch.inngest"
	OtelScopeDebounce  = "debounce.inngest"
	OtelScopeTrigger   = "trigger.inngest"
	OtelScopeCron      = "cron.inngest"
	OtelScopeEnv       = "env.inngest"
	OtelScopeApp       = "app.env.inngest"
	OtelScopeFunction  = "function.app.env.inngest"
	OtelScopeStep      = "step.function.app.env.inngest"
	OtelScopeExecution = "execution.function.app.env.inngest"

	// otel collector filter keys
	OtelUserTraceFilterKey = "inngest.user"

	OtelPropagationKey     = "sys.trace"
	OtelPropagationLinkKey = "sys.trace.link"
)
