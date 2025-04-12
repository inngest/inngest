package meta

// TODO Comments are for potential shortenings later
const (
	// Implementation
	PropagationKey = "user-otel-ctx" // u-ctx

	// Top-level span names
	SpanNameRun              = "executor.run"            // run
	SpanNameStepDiscovery    = "executor.step.discovery" // step.disc
	SpanNameStep             = "executor.step"           // step
	SpanNameExecution        = "executor.execution"      // exec
	SpanNameDynamicExtension = "EXTEND"                  // EXT

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

	// Invoke attributes
	AttributeStepInvokeExpiry         = "_inngest.step.invoke_function.expiry"           // _s.if.exp
	AttributeStepInvokeFunctionID     = "_inngest.step.invoke_function.id"               // _s.if.id
	AttributeStepInvokeTriggerEventID = "_inngest.step.invoke_function.trigger_event_id" // _s.if.eid

	// Sleep attributes
	AttributeStepSleepDuration = "_inngest.step.sleep.duration" // _s.sleep

	// WaitForEvent attributes
	AttributeStepWaitForEventExpiry = "_inngest.step.wait_for_event.expiry" // _s.w.exp
	AttributeStepWaitForEventIf     = "_inngest.step.wait_for_event.if"     // _s.w.if
	AttributeStepWaitForEventName   = "_inngest.step.wait_for_event.name"   // _s.w.name

	// HTTP (serve) attributes
	AttributeResponseHeaders    = "_inngest.response.headers"     // _res.h
	AttributeResponseStatusCode = "_inngest.response.status_code" // _res.st
	AttributeResponseOutputSize = "_inngest.response.output_size" // _res.sz
)
