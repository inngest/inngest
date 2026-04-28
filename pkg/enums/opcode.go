//go:generate go run github.com/dmarkham/enumer -trimprefix=Opcode -type=Opcode -json -text

package enums

type Opcode int

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
	OpcodeDeferCancel
)

// opcodeSyncMap explicitly represents the sync opcodes that can be checkpointed.
// Every other opcode is async by default, and this is always a subset.
var opcodeSyncMap = map[Opcode]struct{}{
	OpcodeStep:            {},
	OpcodeStepRun:         {},
	OpcodeRunComplete:     {},
	OpcodeSyncRunComplete: {},
	OpcodeStepFailed:      {},
	OpcodeDeferAdd:        {},
	OpcodeDeferCancel:     {},
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
	case OpcodeDeferAdd, OpcodeDeferCancel:
		return true
	}
	return false
}
