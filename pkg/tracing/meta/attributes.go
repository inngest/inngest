package meta

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

var Attrs = struct {
	// Timings
	StartedAt Attr[*time.Time]
	QueuedAt  Attr[*time.Time]
	EndedAt   Attr[*time.Time]

	// Run attributes
	AccountID       Attr[*uuid.UUID]
	AppID           Attr[*uuid.UUID]
	BatchID         Attr[*ulid.ULID]
	BatchTimestamp  Attr[*time.Time]
	CronSchedule    Attr[*string]
	DropSpan        Attr[*bool]
	EnvID           Attr[*uuid.UUID]
	EventIDs        Attr[*[]string]
	FunctionID      Attr[*uuid.UUID]
	FunctionVersion Attr[*int]
	RunID           Attr[*ulid.ULID]

	// Dynamic span controls
	DynamicSpanID Attr[*string]
	DynamicStatus Attr[*enums.StepStatus]

	// Internal and debugging
	InternalLocation Attr[*string]
	InternalError    Attr[*string]

	// Step attributes
	StepID           Attr[*string]
	StepName         Attr[*string]
	StepOp           Attr[*enums.Opcode]
	StepAttempt      Attr[*int]
	StepMaxAttempts  Attr[*int]
	StepCodeLocation Attr[*string]
	// StepOutput is the data that has been returned from the step, in
	// its wrapped form for the SDK. This data may not be stored with the span
	// when it hits a store, and instead may be removed to be stored
	// separately.
	StepOutput    Attr[*any]
	StepOutputRef Attr[*string]

	// step.run attributes
	StepRunType Attr[*string]

	// Pause-related attributes
	StepWaitExpired Attr[*bool]
	StepWaitExpiry  Attr[*time.Time]

	// Invoke attributes
	StepInvokeFunctionID     Attr[*uuid.UUID]
	StepInvokeTriggerEventID Attr[*string]
	StepInvokeFinishEventID  Attr[*string]
	StepInvokeRunID          Attr[*ulid.ULID]

	// step.sleep attributes
	StepSleepDuration Attr[*time.Duration]

	// step.waitForEvent attributes
	StepWaitForEventIf        Attr[*string]
	StepWaitForEventName      Attr[*string]
	StepWaitForEventMatchedID Attr[*string]

	// Signal attributes
	StepSignalName Attr[*string]

	// Gateway attributes
	StepGatewayResponseStatusCode      Attr[*int]
	StepGatewayResponseOutputSizeBytes Attr[*int]

	// HTTP (serve) attributes
	RequestURL         Attr[*string]
	ResponseHeaders    Attr[*http.Header]
	ResponseStatusCode Attr[*int]
	ResponseOutputSize Attr[*int]
}{
	StartedAt:      TimeAttr("_inngest.started_at"),      // _s.st
	QueuedAt:       TimeAttr("_inngest.queued_at"),       // _s.q
	EndedAt:        TimeAttr("_inngest.ended_at"),        // _s.e
	AccountID:      UUIDAttr("_inngest.account.id"),      // _acct
	AppID:          UUIDAttr("_inngest.app.id"),          // _app
	BatchID:        ULIDAttr("_inngest.batch.id"),        // _b.id
	BatchTimestamp: TimeAttr("_inngest.batch.ts"),        // _b.ts
	CronSchedule:   StringAttr("_inngest.cron.schedule"), // _cron
	DropSpan:       BoolAttr("_inngest.executor.drop"),   // _drop
	EnvID:          UUIDAttr("_inngest.env.id"),          // _env
	// EventIDs:       StringSliceAttr("_inngest.event.ids"), // _e.ids
	FunctionID:                         UUIDAttr("_inngest.function.id"),     // _fn
	FunctionVersion:                    IntAttr("_inngest.function.version"), // _fn.v
	RunID:                              ULIDAttr("_inngest.run.id"),
	DynamicSpanID:                      StringAttr("_inngest.dynamic.span.id"),    // _d.id
	DynamicStatus:                      StepStatusAttr("_inngest.dynamic.status"), // _d.st
	InternalLocation:                   StringAttr("_inngest.internal.location"),  // _i.loc
	InternalError:                      StringAttr("_inngest.internal.error"),     // _i.err
	StepID:                             StringAttr("_inngest.step.id"),            // _s.id
	StepName:                           StringAttr("_inngest.step.name"),          // _s.name
	StepOp:                             StepOpAttr("_inngest.step.op"),            // _s
	StepAttempt:                        IntAttr("_inngest.step.attempt"),          // _s.attempt
	StepMaxAttempts:                    IntAttr("_inngest.step.max_attempts"),     // _s.max_attempts
	StepCodeLocation:                   StringAttr("_inngest.step.code_location"), // _s.code_loc
	StepOutput:                         AnyAttr("_inngest.step.output"),
	StepOutputRef:                      StringAttr("_inngest.step.output_ref"),                      // _s.output_ref
	StepRunType:                        StringAttr("_inngest.step.run.type"),                        // _s.run.type
	StepWaitExpired:                    BoolAttr("_inngest.step.wait.expired"),                      // _s.wait
	StepWaitExpiry:                     TimeAttr("_inngest.step.wait.expiry"),                       // _s.wait.expiry
	StepInvokeFunctionID:               UUIDAttr("_inngest.step.invoke.function.id"),                // _s.invoke.fn.id
	StepInvokeTriggerEventID:           StringAttr("_inngest.step.invoke.trigger.event.id"),         // _s.invoke.trigger.event.id
	StepInvokeFinishEventID:            StringAttr("_inngest.step.invoke.finish.event.id"),          // _s.invoke.finish.event.id
	StepInvokeRunID:                    ULIDAttr("_inngest.step.invoke.run.id"),                     // _s.invoke.run.id
	StepSleepDuration:                  DurationAttr("_inngest.step.sleep.duration"),                // _s.sleep.duration
	StepWaitForEventIf:                 StringAttr("_inngest.step.wait_for_event.if"),               // _s.wfe.if
	StepWaitForEventName:               StringAttr("_inngest.step.wait_for_event.name"),             // _s.wfe.name
	StepWaitForEventMatchedID:          StringAttr("_inngest.step.wait_for_event.matched_id"),       // _s.wfe.matched_id
	StepSignalName:                     StringAttr("_inngest.step.signal.name"),                     // _s.signal.name
	StepGatewayResponseStatusCode:      IntAttr("_inngest.step.gateway.response.status_code"),       // _s.gateway.response.status_code
	StepGatewayResponseOutputSizeBytes: IntAttr("_inngest.step.gateway.response.output_size_bytes"), // _s.gateway.response.output_size_bytes
	RequestURL:                         StringAttr("_inngest.request.url"),                          // _req.url
	ResponseHeaders:                    HttpHeaderAttr("_inngest.response.headers"),                 // _res.headers
	ResponseStatusCode:                 IntAttr("_inngest.response.status_code"),                    // _res.status_code
	ResponseOutputSize:                 IntAttr("_inngest.response.output_size"),                    // _res.output_size
}
