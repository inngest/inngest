package inngest

const (
	TriggerName = "$trigger"
)

var (
	SourceEdge = Edge{
		Incoming: TriggerName,
	}
)

type Edge struct {
	// Incoming is the name of the step to run.  This is always the name of the
	// concrete step, even if we're running a generator.
	Incoming string `json:"incoming"`
	// StepPlanned is the ID of the generator step planned via enums.OpcodeStepPlanned.
	//
	// We cannot use "Incoming" here as the incoming name still needs to tbe the generator.
	IncomingGeneratorStep string `json:"gen,omitempty"`
	// IncomingGeneratorStepName is the name from step planned. it should be empty for
	// other cases
	IncomingGeneratorStepName string `json:"gen_name,omitempty"`
	// Outgoing is the name of the generator step or step that last ran.
	Outgoing string `json:"outgoing"`
}

func (e Edge) IsSource() bool {
	return e.Outgoing == "" && e.Incoming == TriggerName || e.Outgoing == TriggerName
}
