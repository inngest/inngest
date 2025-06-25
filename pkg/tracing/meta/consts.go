package meta

// TODO Comments are for potential shortenings later
const (
	// Implementation
	PropagationKey = "user-otel-ctx" // u-ctx
	// Used when an internal error has occurred and may have resulted in a span
	// being mishandled or have incorrect or imcomplete data. In this case, we
	// should store any errors under this attribute.
	InternalError = "_inngest.internal.error" // _int.err

	// Top-level span names
	SpanNameRun              = "executor.run"            // run
	SpanNameStepDiscovery    = "executor.step.discovery" // step.disc
	SpanNameStep             = "executor.step"           // step
	SpanNameExecution        = "executor.execution"      // exec
	SpanNameDynamicExtension = "EXTEND"                  // EXT

	// Timings
	AttributeQueuedAt  = "_inngest.queued_at"  // _q.at
	AttributeStartedAt = "_inngest.started_at" // _s.at
	AttributeEndedAt   = "_inngest.ended_at"   // _e.at

	// Run attributes
	AttributeAccountID        = "_inngest.account.id"        // _acct
	AttributeAppID            = "_inngest.app.id"            // _app
	AttributeBatchID          = "_inngest.batch.id"          // _b.id
	AttributeBatchTimestamp   = "_inngest.batch.ts"          // _b.ts
	AttributeCronSchedule     = "_inngest.cron.schedule"     // _cron
	AttributeDropSpan         = "_inngest.executor.drop"     // _drop
	AttributeEnvID            = "_inngest.env.id"            // _env
	AttributeEventIDs         = "_inngest.event.ids"         // _e.ids
	AttributeFunctionID       = "_inngest.function.id"       // _fn
	AttributeFunctionVersion  = "_inngest.function.version"  // _fn.v
	AttributeInternalLocation = "_inngest.internal.location" // _loc
	AttributeRunID            = "_inngest.run.id"            // _run

	// Dynamic span controls
	AttributeDynamicSpanID = "_inngest.dynamic.span.id" // _d.id
	AttributeDynamicStatus = "_inngest.dynamic.status"  // _d.st

	// Link attributes
	LinkAttributeType            = "_inngest.link.type" // _l.type
	LinkAttributeTypeFollowsFrom = "follows_from"       // follows

	// Generic step attributes
	AttributeStepID          = "_inngest.step.id"           // _s.id
	AttributeStepName        = "_inngest.step.name"         // _s.n
	AttributeStepOp          = "_inngest.step.op"           // _s.op
	AttributeStepAttempt     = "_inngest.step.attempt"      // _s.a
	AttributeStepMaxAttempts = "_inngest.step.max_attempts" // _s.m
	AttributeCodeLocation    = "_inngest.code.location"     // _s.loc
	// AttributeStepOutput is the data that has been returned from the step, in
	// its wrapped form for the SDK. This data may not be stored with the span
	// when it hits a store, and instead may be removed to be stored
	// separately.
	AttributeStepOutput    = "_inngest.step.output"     // _s.out
	AttributeStepOutputRef = "_inngest.step.output.ref" // _s.out.ref

	// Run attributes
	AttributeStepRunType = "_inngest.step.run.type" // _s.r.type

	// Pause-related attributes
	AttributeStepWaitExpired = "_inngest.step.wait.expired" // _s.w.expired
	AttributeStepWaitExpiry  = "_inngest.step.wait.expiry"  // _s.w.expiry

	// Invoke attributes
	AttributeStepInvokeFunctionID     = "_inngest.step.invoke_function.id"               // _s.if.id
	AttributeStepInvokeTriggerEventID = "_inngest.step.invoke_function.trigger_event_id" // _s.if.eid
	AttributeStepInvokeFinishEventID  = "_inngest.step.invoke_function.finish_event_id"  // _s.if.fid
	AttributeStepInvokeRunID          = "_inngest.step.invoke_function.run_id"           // _s.if.rid

	// Sleep attributes
	AttributeStepSleepDuration = "_inngest.step.sleep.duration" // _s.sleep

	// WaitForEvent attributes
	AttributeStepWaitForEventIf        = "_inngest.step.wait_for_event.if"         // _s.w.if
	AttributeStepWaitForEventName      = "_inngest.step.wait_for_event.name"       // _s.w.name
	AttributeStepWaitForEventMatchedID = "_inngest.step.wait_for_event.matched_id" // _s.w.mid

	// Signal attributes
	AttributeStepSignalName = "_inngest.step.signal.name" // _s.sig.name

	// Gateway attributes
	AttributeStepGatewayResponseStatusCode      = "_inngest.step.gateway.response.status_code" // _s.gw.res.st
	AttributeStepGatewayResponseOutputSizeBytes = "_inngest.step.gateway.response.output_size" // _s.gw.res.sz

	// HTTP (serve) attributes
	AttributeRequestURL         = "_inngest.request.uri"          // _req.u
	AttributeResponseHeaders    = "_inngest.response.headers"     // _res.h
	AttributeResponseStatusCode = "_inngest.response.status_code" // _res.st
	AttributeResponseOutputSize = "_inngest.response.output_size" // _res.sz
)
