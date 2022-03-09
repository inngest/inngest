package inngest

import (
	"fmt"

	"github.com/inngest/inngestctl/inngest/internal/cuedefs"
)

// ParseWorkflow parses a cue configuration defining a workflow.
func ParseWorkflow(input string) (*Workflow, error) {
	val, err := cuedefs.ParseWorkflow(input)
	if err != nil {
		return nil, fmt.Errorf("error parsing workflow: %w", err)
	}
	w := &Workflow{}
	if err := val.Decode(&w); err != nil {
		return nil, fmt.Errorf("error deserializing workflow: %w", err)
	}

	return w, nil
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

// Workflow represents a workflow encoded wtihin the Cue configuration language.
//
// This represents the logic for a workflow, but does not represent any specific
// workflow in the database.
type Workflow struct {
	// ID is the immutable human identifier for the workflow.  This acts
	// similarly to a git repository name;  a single workflow ID can contain
	// many workflow versions.
	//
	// When deploying a specific workflow version we read the cue configuration
	// and upsert a version to the given ID.
	ID   string `json:"id"`
	Name string `json:"name"`

	Triggers []Trigger `json:"triggers"`
	Actions  []Action  `json:"actions"`
	Edges    []Edge    `json:"edges"`
}

// Trigger represents the starting point for a workflow
type Trigger struct {
	*EventTrigger
	*CronTrigger
}

// EventTrigger represents an event that triggers this workflow.
type EventTrigger struct {
	Event      string  `json:"event"`
	Expression *string `json:"expression"`
}

// CronTrigger represents the cron schedule that triggers this workflow
type CronTrigger struct {
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
	Metadata *EdgeMetadata `json:"metadata,omitempty"`
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

var workflowTpl = `package main

import (
	"inngest.com/workflows"
)

workflow: workflows.#Workflow & %s`
