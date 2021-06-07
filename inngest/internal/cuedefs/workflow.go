package cuedefs

import (
	"cuelang.org/go/cue"
)

const (
	workflowConst     = "workflow"
	workflowDefSuffix = "workflow: workflows.#Workflow"
)

// ParseWorkflow parses a cue configuration defining a workflow.  It returns
// the cue.Value of the given workflow.
func ParseWorkflow(input string) (*cue.Value, error) {
	// TODO: (tonyhb) define workflow struct, and parse a concrete type here.
	return parseDef(input, workflowConst, workflowDefSuffix)
}
