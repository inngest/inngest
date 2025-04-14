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
)
