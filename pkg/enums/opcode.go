//go:generate go run github.com/dmarkham/enumer -trimprefix=Opcode -type=Opcode -json -text

package enums

type Opcode int

const (
	// OpcodeNone represents the default opcode 0, which does nothing
	OpcodeNone Opcode = iota
	OpcodeStep
	OpcodeStepPlanned
	OpcodeSleep
	OpcodeWaitForEvent
)
