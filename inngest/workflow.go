package inngest

import "github.com/google/uuid"

const (
	TriggerName = "$trigger"
)

var (
	SourceEdge = Edge{
		Incoming: TriggerName,
	}
)

// Workflow represents a workflow encoded wtihin the Cue configuration language.
//
// This represents the logic for a workflow, but does not represent any specific
// workflow in the database.
type Workflow struct {
	// UUID is a surrogate key.
	UUID uuid.UUID `json:"-"`
	// ID is the immutable human identifier for the workflow.  This acts
	// similarly to a git repository name;  a single workflow ID can contain
	// many workflow versions.
	//
	// When deploying a specific workflow version we read the cue configuration
	// and upsert a version to the given ID.
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Throttle *Throttle `json:"throttle,omitempty"`
	Triggers []Trigger `json:"triggers"`
	Steps    []Step    `json:"actions"`
	Edges    []Edge    `json:"edges"`
}

type Throttle struct {
	// Count is how often the function can be called within the specified period
	Count uint `json:"count"`
	// Period represents the time period for throttling the function
	Period string `json:"period"`
	// Key is an optional string to constrain throttling using event data.  For
	// example, if you want to throttle incoming notifications based off of a user's
	// ID in an event you can use the following key: "{{ event.user.id }}".  This ensures
	// that we throttle functions for each user independently.
	Key *string `json:"key"`
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

// Step is a reference to an action within a workflow.
type Step struct {
	ClientID string                 `json:"clientID"`
	Name     string                 `json:"name"`
	DSN      string                 `json:"dsn"`
	Version  *VersionConstraint     `json:"version,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type Edge struct {
	Outgoing string `json:"outgoing"`
	Incoming string `json:"incoming"`
	// Metadata specifies the type of edge to use.  This defaults
	// to EdgeTypeEdge - a basic link that can conditionally run.
	Metadata *EdgeMetadata `json:"metadata,omitempty"`
}

type EdgeMetadata struct {
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
	If   string `json:"if,omitempty"`
	// Wait specifies that the edge should only be traversed after the specified
	// duration.  This, in effect, allows you to delay jobs for a given amount of
	// time.
	Wait               *string `json:"wait,omitempty"`
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

// VersionCoinstraint represents version constraints for an action.  We use semver without
// patches:
// - Major versions are backwards-incompatible (eg. requesting different secrets,
//   incompatible APIs).
// - Minor versions are backwards compatible improvements, fixes, or additions.  We
//   automatically use the latest minor version within every step function.
type VersionConstraint struct {
	Major *uint `json:"version,omitempty"`
	Minor *uint `json:"minor,omitempty"`
}
