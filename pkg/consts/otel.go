package consts

const (
	OtelSpanEvent        = "event"
	OtelSpanCron         = "cron"
	OtelSpanBatch        = "batch"
	OtelSpanDebounce     = "debounce"
	OtelSpanTrigger      = "trigger"
	OtelSpanInvoke       = "invoke"
	OtelSpanWaitForEvent = "wait"
	OtelSpanSleep        = "sleep"
	OtelSpanExecute      = "execute"
	OtelSpanRerun        = "rerun"

	// system attributes
	OtelSysAccountID      = "sys.account.id"
	OtelSysWorkspaceID    = "sys.workspace.id"
	OtelSysAppID          = "sys.app.id"
	OtelSysIdempotencyKey = "sys.idempotency.key"
	OtelSysLifecycleID    = "sys.lifecycle.id"

	OtelSysEventData       = "sys.event"
	OtelSysEventRequestID  = "sys.event.request.id"
	OtelSysEventInternalID = "sys.event.internal.id"
	OtelSysEventIDs        = "sys.event.ids"

	OtelSysBatchID      = "sys.batch.id"
	OtelSysBatchTS      = "sys.batch.timestamp"
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
	OtelSysFunctionLink       = "sys.function.link"
	OtelSysFunctionHasAI      = "sys.function.hasAI"

	OtelSysStepID              = "sys.step.id"
	OtelSysStepDisplayName     = "sys.step.display.name"
	OtelSysStepOpcode          = "sys.step.opcode"
	OtelSysStepScheduleAt      = "sys.step.time.schedule"
	OtelSysStepStartAt         = "sys.step.time.start"
	OtelSysStepEndAt           = "sys.step.time.end"
	OtelSysStepStatus          = "sys.step.status"
	OtelSysStepStatusCode      = "sys.step.status.code"
	OtelSysStepAttempt         = "sys.step.attempt"
	OtelSysStepMaxAttempt      = "sys.step.attempt.max"
	OtelSysStepInput           = "sys.step.input"
	OtelSysStepOutput          = "sys.step.output"
	OtelSysStepOutputSizeBytes = "sys.step.output.size.bytes"
	OtelSysStepFirst           = "sys.step.first"
	OtelSysStepGroupID         = "sys.step.group.id"
	OtelSysStepStack           = "sys.step.stack"
	OtelSysStepAIRequest       = "sys.step.ai.req" // ai request metadata
	OtelSysStepAIResponse      = "sys.step.ai.res" // ai response metadata
	OtelSysStepRunType         = "sys.step.run.type"
	OtelSysStepPlan            = "sys.step.plan" // indicate this is a planning step

	OtelSysStepSleepEndAt = "sys.step.sleep.end"

	OtelSysStepWaitExpires        = "sys.step.wait.expires"
	OtelSysStepWaitExpired        = "sys.step.wait.expired"
	OtelSysStepWaitEventName      = "sys.step.wait.event"
	OtelSysStepWaitExpression     = "sys.step.wait.expr"
	OtelSysStepWaitMatchedEventID = "sys.step.wait.matched.event.id"

	OtelSysStepInvokeExpires           = "sys.step.invoke.expires"
	OtelSysStepInvokeTargetFnID        = "sys.step.invoke.fn.id"
	OtelSysStepInvokeTriggeringEventID = "sys.step.invoke.event.outgoing.id"
	OtelSysStepInvokeReturnedEventID   = "sys.step.invoke.event.incoming.id"
	OtelSysStepInvokeRunID             = "sys.step.invoke.run.id"
	OtelSysStepInvokeExpired           = "sys.step.invoke.expired"

	OtelSysStepRetry  = "sys.step.retry"
	OtelSysStepDelete = "sys.step.delete"

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
	OtelScopeInvoke    = "invoke.inngest"
	OtelScopeRerun     = "rerun.inngest"
	OtelScopeEnv       = "env.inngest"
	OtelScopeApp       = "app.env.inngest"
	OtelScopeFunction  = "function.app.env.inngest"
	OtelScopeStep      = "step.function.app.env.inngest"
	OtelScopeExecution = "execution.function.app.env.inngest"

	// Propagation keys
	OtelPropagationKey     = "sys.trace"
	OtelPropagationLinkKey = "sys.trace.link"

	// execution copies
	OtelExecPlaceholder = "execute"
	OtelExecFnOk        = "function success"
	OtelExecFnErr       = "function error"
)
