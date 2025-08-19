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
)

// opcodeSyncMap explicitly represents the sync opcodes that can be checkpointed.
// Every other opcode is async by default, and this is always a subset.
var opcodeSyncMap = map[Opcode]struct{}{
	OpcodeStep:        {},
	OpcodeStepRun:     {},
	OpcodeRunComplete: {},
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
