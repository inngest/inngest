//go:generate go run github.com/dmarkham/enumer -trimprefix=Opcode -type=Opcode -json -text

package enums

import "fmt"

type Opcode int

// UnknownOpcodeError indicates the SDK returned an opcode this server version
// doesn't recognize, which almost always means the server is older than the
// SDK. Its message prompts a server upgrade rather than surfacing the opaque
// decode error.
type UnknownOpcodeError struct {
	Opcode string
}

func (e *UnknownOpcodeError) Error() string {
	return fmt.Sprintf(
		"this Inngest server does not recognize the %q operation returned by the SDK. This usually means the server is older than the SDK; update your Inngest server to the latest version.",
		e.Opcode,
	)
}

// Retryable satisfies the execution queue's structural RetryableError interface
// without importing it: version skew is deterministic, so retrying never helps.
func (e *UnknownOpcodeError) Retryable() bool { return false }

// ParseOpcode is like OpcodeString but distinguishes an otherwise valid opcode
// name that this server version doesn't know about from a malformed value,
// returning an *UnknownOpcodeError so callers can detect server/SDK version
// skew.
func ParseOpcode(s string) (Opcode, error) {
	op, err := OpcodeString(s)
	if err != nil {
		return op, &UnknownOpcodeError{Opcode: s}
	}
	return op, nil
}

const (
	// OpcodeNone represents the default opcode 0, which does nothing
	OpcodeNone        Opcode = iota
	OpcodeStep               // A step run repsonse, _maybe_ with wrapped data.
	OpcodeStepRun            // Same as OpcodeStep, but guarantees data is not wrapped
	OpcodeStepError          // A step errored.  The response contains error information.  This can only exist for `step.Run`
	OpcodeStepPlanned        // A step is reported and should be executed next.
	OpcodeSleep
	OpcodeWaitForEvent
	OpcodeInvokeFunction
	OpcodeAIGateway // AI gateway inference call
	OpcodeGateway   // Gateway call
	OpcodeWaitForSignal
	OpcodeRunComplete
	OpcodeStepFailed
	// OpcodeSyncRunComplete represents a sync API-based function completion.  This is
	// distinct from OpcodeRunComplete as it always contains a specific shape of data.
	OpcodeSyncRunComplete
	// OpcodeDiscoveryRequest indicates that an SDK wants another discovery
	// request to be sent to resume execution.
	OpcodeDiscoveryRequest

	OpcodeDeferAdd
	OpcodeDeferAbort
)

// opcodeSyncMap explicitly represents the sync opcodes that can be checkpointed.
// Every other opcode is async by default, and this is always a subset.
var opcodeSyncMap = map[Opcode]struct{}{
	OpcodeStep:            {},
	OpcodeStepRun:         {},
	OpcodeStepPlanned:     {},
	OpcodeRunComplete:     {},
	OpcodeSyncRunComplete: {},
	OpcodeStepFailed:      {},
	OpcodeDeferAdd:        {},
	OpcodeDeferAbort:     {},
}

// OpcodeIsSync returns whether the given opcode is synchronous.  This
// allows us to process specific opcodes in sync runs without switching
// to async execution.
func OpcodeIsSync(o Opcode) bool {
	_, ok := opcodeSyncMap[o]
	return ok
}

func OpcodeIsAsync(o Opcode) bool {
	return !OpcodeIsSync(o)
}

// OpcodeIsLazy reports whether the opcode is a lazy op piggybacked onto a host
// op rather than a standalone step. Lazy ops travel alongside another opcode
// (e.g. [StepRun, DeferAdd]) and shouldn't trigger parallel-step gating like
// ForceStepPlan or per-step history grouping.
func OpcodeIsLazy(o Opcode) bool {
	switch o {
	case OpcodeDeferAdd, OpcodeDeferAbort:
		return true
	}
	return false
}

// OpcodeIsPriority reports whether the opcode must be processed before
// non-priority opcodes in the same batch.
func OpcodeIsPriority(o Opcode) bool {
	switch o {
	case OpcodeWaitForEvent:
		// Prioritize in case the user wrote an invoke-esque "[WaitForEvent,
		// SendEvent]" pattern. For example, they want to wait for an event
		// that's sent by a function triggered by the SendEvent. If we don't
		// finish processing the WaitForEvent before the SendEvent does its
		// thing, then we may miss the event.
		return true
	case OpcodeDeferAdd, OpcodeDeferAbort:
		// Prioritize in case the SDK returned something like "[DeferAdd,
		// RunComplete]". If we don't finish processing the DeferAdd before
		// finalizing the run, then we'll won't send the defer event.
		return true
	}
	return false
}
