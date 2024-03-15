package consts

const (
	// system attributes
	OtelSysRootSpan        = "sys.root"
	OtelSysAccountID       = "sys.account.id"
	OtelSysWorkspaceID     = "sys.workspace.id"
	OtelSysAppID           = "sys.app.id"
	OtelSysEventData       = "sys.event"
	OtelSysEventIDs        = "sys.event.ids"
	OtelSysBatchID         = "sys.batch.id"
	OtelSysIdempotencyKey  = "sys.idempotency.key"
	OtelSysFunctionID      = "sys.function.id"
	OtelSysFunctionSlug    = "sys.function.slug"
	OtelSysFunctionVersion = "sys.function.version"
	OtelSysFunctionOutput  = "sys.function.output"
	OtelSysStepOutput      = "sys.step.output"

	// SDK attributes
	OtelAttrSDKServiceName = "sdk.app.name"
	OtelAttrSDKRunID       = "sdk.run.id"

	// otel scopes
	OtelScopeEventAPI       = "event.api.inngest"
	OtelScopeEventIngestion = "event.inngest"
	OtelScopeEnv            = "env.inngest"
	OtelScopeApp            = "app.env.inngest"
	OtelScopeFunction       = "function.app.env.inngest"
	OtelScopeStep           = "step.function.app.env.inngest"

	// otel collector filter keys
	OtelUserTraceFilterKey = "inngest.user"

	OtelCtxQueuePropKey  = "trace"
	OtelCtxPubsubPropKey = "data"
)
