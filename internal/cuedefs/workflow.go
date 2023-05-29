package cuedefs

import (
	"fmt"

	"cuelang.org/go/cue"
	"github.com/inngest/inngest/pkg/inngest"
)

const (
	workflowConst     = "workflow"
	workflowDefSuffix = "workflow: workflows.#Workflow"
)

// ParseWorkflow parses a cue configuration defining a workflow.  It returns
// the cue.Value of the given workflow.
func ReadWorkflow(input string) (*cue.Value, error) {
	// TODO: (tonyhb) define workflow struct, and parse a concrete type here.
	return parseDef(input, workflowConst, workflowDefSuffix)
}

// ParseWorkflow parses a cue configuration defining a workflow.
func ParseWorkflow(input string) (*inngest.Workflow, error) {
	val, err := ReadWorkflow(input)
	if err != nil {
		return nil, fmt.Errorf("error parsing workflow: %w", err)
	}
	w := &inngest.Workflow{}
	if err := val.Decode(&w); err != nil {
		return nil, fmt.Errorf("error deserializing workflow: %w", err)
	}

	return w, nil
}

// FormatWorkflow formats a workflow struct into a canonical cue string representation
func FormatWorkflow(a inngest.Workflow) (string, error) {
	def, err := FormatDef(a)
	if err != nil {
		return "", err
	}
	// XXX: Inspect cue and implement packages.
	return fmt.Sprintf(workflowTpl, def), nil
}

const workflowTpl = `package main

import (
	"inngest.com/workflows"
)

workflow: workflows.#Workflow & %s`
