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
	StartedAt attr[*time.Time]
	QueuedAt  attr[*time.Time]
	EndedAt   attr[*time.Time]

	// Run attributes
	AccountID       attr[*uuid.UUID]
	AppID           attr[*uuid.UUID]
	BatchID         attr[*ulid.ULID]
	BatchTimestamp  attr[*time.Time]
	CronSchedule    attr[*string]
	DropSpan        attr[*bool]
	EnvID           attr[*uuid.UUID]
	EventIDs        attr[*[]string]
	FunctionID      attr[*uuid.UUID]
	FunctionVersion attr[*int]
	RunID           attr[*ulid.ULID]

	// Dynamic span controls
	DynamicSpanID attr[*string]
	DynamicStatus attr[*enums.StepStatus]

	// Internal and debugging
	InternalLocation attr[*string]
	// Internal as we want this only to be set by internal logic.
	internalError attr[*string]

	// Function attributes
	IsFunctionOutput attr[*bool]

	// Step attributes
	StepID           attr[*string]
	StepName         attr[*string]
	StepOp           attr[*enums.Opcode]
	StepAttempt      attr[*int]
	StepMaxAttempts  attr[*int]
	StepCodeLocation attr[*string]
	// StepOutput is the data that has been returned from the step, in
	// its wrapped form for the SDK. This data may not be stored with the span
	// when it hits a store, and instead may be removed to be stored
	// separately.
	StepOutput    attr[*string]
	StepOutputRef attr[*string]
	// StepHasOutput is used to mark that a specific span has an output in the
	// attributes, in place of the output itself.
	StepHasOutput attr[*bool]

	// step.run attributes
	StepRunType attr[*string]

	// Pause-related attributes
	StepWaitExpired attr[*bool]
	StepWaitExpiry  attr[*time.Time]

	// Invoke attributes
	StepInvokeFunctionID     attr[*string]
	StepInvokeTriggerEventID attr[*ulid.ULID]
	StepInvokeFinishEventID  attr[*ulid.ULID]
	StepInvokeRunID          attr[*ulid.ULID]

	// step.sleep attributes
	StepSleepDuration attr[*time.Duration]

	// step.waitForEvent attributes
	StepWaitForEventIf        attr[*string]
	StepWaitForEventName      attr[*string]
	StepWaitForEventMatchedID attr[*ulid.ULID]

	// Signal attributes
	StepSignalName attr[*string]

	// Gateway attributes
	StepGatewayResponseStatusCode      attr[*int]
	StepGatewayResponseOutputSizeBytes attr[*int]

	// HTTP (serve) attributes
	RequestURL         attr[*string]
	ResponseHeaders    attr[*http.Header]
	ResponseStatusCode attr[*int]
	ResponseOutputSize attr[*int]

	// Debugger attributes
	DebugSessionID attr[*ulid.ULID]
	DebugRunID     attr[*ulid.ULID]
}{
	internalError: StringAttr("internal.error"),

	AccountID:                          UUIDAttr("account.id"),
	AppID:                              UUIDAttr("app.id"),
	BatchID:                            ULIDAttr("batch.id"),
	BatchTimestamp:                     TimeAttr("batch.ts"),
	CronSchedule:                       StringAttr("cron.schedule"),
	DropSpan:                           BoolAttr("executor.drop"),
	DynamicSpanID:                      StringAttr("dynamic.span.id"),
	DynamicStatus:                      StepStatusAttr("dynamic.status"),
	EndedAt:                            TimeAttr("ended_at"),
	EnvID:                              UUIDAttr("env.id"),
	EventIDs:                           StringSliceAttr("event.ids"),
	FunctionID:                         UUIDAttr("function.id"),
	FunctionVersion:                    IntAttr("function.version"),
	InternalLocation:                   StringAttr("internal.location"),
	IsFunctionOutput:                   BoolAttr("is.function.output"),
	QueuedAt:                           TimeAttr("queued_at"),
	RequestURL:                         StringAttr("request.url"),
	ResponseHeaders:                    HttpHeaderAttr("response.headers"),
	ResponseOutputSize:                 IntAttr("response.output_size"),
	ResponseStatusCode:                 IntAttr("response.status_code"),
	RunID:                              ULIDAttr("run.id"),
	StartedAt:                          TimeAttr("started_at"),
	StepAttempt:                        IntAttr("step.attempt"),
	StepCodeLocation:                   StringAttr("step.code_location"),
	StepGatewayResponseOutputSizeBytes: IntAttr("step.gateway.response.output_size_bytes"),
	StepGatewayResponseStatusCode:      IntAttr("step.gateway.response.status_code"),
	// StepHasOutput is used to mark that a specific span has an output in the
	// attributes, in place of the output itself.
	StepHasOutput:             BoolAttr("step.has_output"),
	StepID:                    StringAttr("step.id"),
	StepInvokeFinishEventID:   ULIDAttr("step.invoke.finish.event.id"),
	StepInvokeFunctionID:      StringAttr("step.invoke.function.id"),
	StepInvokeRunID:           ULIDAttr("step.invoke.run.id"),
	StepInvokeTriggerEventID:  ULIDAttr("step.invoke.trigger.event.id"),
	StepMaxAttempts:           IntAttr("step.max_attempts"),
	StepName:                  StringAttr("step.name"),
	StepOp:                    StepOpAttr("step.op"),
	StepOutput:                StringAttr("step.output"),
	StepOutputRef:             StringAttr("step.output_ref"),
	StepRunType:               StringAttr("step.run.type"),
	StepSignalName:            StringAttr("step.signal.name"),
	StepSleepDuration:         DurationAttr("step.sleep.duration"),
	StepWaitExpired:           BoolAttr("step.wait.expired"),
	StepWaitExpiry:            TimeAttr("step.wait.expiry"),
	StepWaitForEventIf:        StringAttr("step.wait_for_event.if"),
	StepWaitForEventMatchedID: ULIDAttr("step.wait_for_event.matched_id"),
	StepWaitForEventName:      StringAttr("step.wait_for_event.name"),
	DebugSessionID:            ULIDAttr("debug.session.id"),
	DebugRunID:                ULIDAttr("debug.run.id"),
}
