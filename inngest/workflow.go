package inngest

import (
	"fmt"

	"github.com/inngest/inngestctl/inngest/internal/cuedefs"
)

// Workflow represents a workflow encoded wtihin the Cue configuration language.
//
// This represents the logic for a workflow, but does not represent any specific
// workflow in the database.
type Workflow struct {
	Name string `json:"name"`

	Triggers []Trigger `json:"triggers"`
	Actions  []Action  `json:"actions"`
	Edges    []Edge    `json:"edges"`
}

// Trigger represents the starting point for a workflow
type Trigger struct {
	*EventTrigger
	*ScheduleTrigger
}

// EventTrigger represents an event that triggers this workflow.
type EventTrigger struct {
	Event string `json:"event"`
}

// ScheduleTrigger represents the cron schedule that triggers this workflow
type ScheduleTrigger struct {
	Cron string `json:"cron"`
}

// Action represents a serialized action within a workflow.  This represents
// the set of information to run a single action.Action for a workflow.  It
// is not the action itself.
type Action struct {
	ClientID uint                   `json:"clientID"`
	Name     string                 `json:"name"`
	DSN      string                 `json:"dsn"`
	Version  *uint                  `json:"version,omitempty"`
	Metadata map[string]interface{} `json:"metadata"`
}

type Edge struct {
	Outgoing interface{} `json:"outgoing"`
	Incoming uint        `json:"incoming"`
	// Metadata specifies the type of edge to use.  This defaults
	// to EdgeTypeEdge - a basic link that can conditionally run.
	Metadata EdgeMetadata `json:"metadata,omitempty"`
}

type EdgeMetadata struct {
	Type               string `json:"type"`
	Name               string `json:"name"`
	If                 string `json:"if"`
	*AsyncEdgeMetadata `json:"async,omitempty"`
}

type AsyncEdgeMetadata struct {
	TTL string `json:"ttl"`
	// Event specifies the event name to listen for, which can coninue this workflow.
	Event string `json:"event"`
	// Match represents the optional expression to use when matching the event.
	// If specified, the event name must match and this expression must evaluate
	// to true for the workflow to continue.  This allows you to filter events
	// to eg. the same user.
	Match *string `json:"match"`
}

// FormatWorkflow formats a workflow struct into a canonical cue string representation
func FormatWorkflow(a Workflow) (string, error) {
	def, err := cuedefs.FormatDef(a)
	if err != nil {
		return "", err
	}
	// XXX: Inspect cue and implement packages.
	return fmt.Sprintf(workflowTpl, def), nil
}

var workflowTpl = `
package main

import (
	"inngest.com/workflows"
)

workflow: workflows.#Workflow & %s`
