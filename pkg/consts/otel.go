package consts

const (
	// system attributes
	OtelSysAccountID      = "sys.account.id"
	OtelSysWorkspaceID    = "sys.workspace.id"
	OtelSysAppID          = "sys.app.id"
	OtelSysEventData      = "sys.event"
	OtelSysEventRequestID = "sys.event.request.id"
	OtelSysEventIDs       = "sys.event.ids"
	OtelSysBatchID        = "sys.batch.id"
	OtelSysIdempotencyKey = "sys.idempotency.key"

	OtelSysFunctionID         = "sys.function.id"
	OtelSysFunctionSlug       = "sys.function.slug"
	OtelSysFunctionVersion    = "sys.function.version"
	OtelSysFunctionScheduleAt = "sys.function.time.schedule"
	OtelSysFunctionStartAt    = "sys.function.time.start"
	OtelSysFunctionEndAt      = "sys.function.time.end"
	OtelSysFunctionStatus     = "sys.function.status"
	OtelSysFunctionOutput     = "sys.function.output"

	OtelSysStepScheduleAt      = "sys.step.time.schedule"
	OtelSysStepStartAt         = "sys.step.time.start"
	OtelSysStepEndAt           = "sys.step.time.end"
	OtelSysStepStatus          = "sys.step.status"
	OtelSysStepAttempt         = "sys.step.attempt"
	OtelSysStepOutput          = "sys.step.output"
	OtelSysStepOutputSizeBytes = "sys.step.output.size.bytes"

	// SDK attributes
	OtelAttrSDKServiceName = "sdk.app.id"
	OtelAttrSDKRunID       = "sdk.run.id"

	// otel scopes
	OtelScopeEventAPI       = "event.api.inngest"
	OtelScopeEventIngestion = "event.inngest"
	OtelScopeCron           = "cron.inngest"
	OtelScopeEnv            = "env.inngest"
	OtelScopeApp            = "app.env.inngest"
	OtelScopeFunction       = "function.app.env.inngest"
	OtelScopeStep           = "step.function.app.env.inngest"

	// otel collector filter keys
	OtelUserTraceFilterKey = "inngest.user"

	OtelPropagationKey = "sys.trace"
)
