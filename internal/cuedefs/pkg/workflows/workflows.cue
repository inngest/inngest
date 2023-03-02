package workflows

import (
	"inngest.com/edges"
)

#EdgeMetadata: edges.#Edge

// a workflow is an entire workflow for our app
#Workflow: {
	// id represents the immutable identifier for the workflow, which groups all
	// versions into a single workflow.  If this were GitHub, this would be the
	// repository name.
	id: =~"^[a-z0-9-]+$"

	// The workflow name.
	name: string

	// The concurrency limit for this function.  This indicates the total number
	// of parallel steps that can occur within copies of this function.
	// 0 means use the max available within your account.
	concurrency?: >=0

	// The triggers which start a workflow.
	//
	// If this is a scheduled trigger, only one trigger may exist.
	// Workflows triggered by events may contain multiple event triggers which are exclusive -
	// any of these triggers will start a workflow.
	triggers?: [ ...#Trigger]
	actions?: [ ...#Action]
	edges?: [ ...#Edge]
	alerts?: [ ...#Alert]
	cancel?: [...#Cancel]
}

#Alert: {
	workflowID: string
}

// trigger represents the event that starts our the workflow
#Trigger: #EventTrigger | #ScheduleTrigger

#EventTrigger: {
	event:       string
	expression?: string
}

#ScheduleTrigger: {
	cron: string
}

#Action: {
	// clientID represents the ID of the action as represented by edges and
	// by the frontend's rendering.
	clientID: uint & >=1
	id:       string | *""
	// name of the action
	name: string
	// dsn of the action.  eg "com.datosapp.logic.if" to test a predicate
	// or "com.datosapp.comms.email" to send an email
	dsn: string
	// version of the action DSN to run.  If this is undefined, this defaults
	// to the latest version of the action.
	version?: {
		major?: uint
		minor?: uint
	}
	// Metadata about how the action will be used.  Each action requires custom
	// input to work, eg. what data to transform, what email template to use, etc.
	metadata?: [string]: _

	retries?: {
		attempts?: int & >=0 & <=20
	}
}

#Edge: {
	outgoing:  string | "$trigger" // Either the action ID or 'trigger'
	incoming:  string
	metadata?: #EdgeMetadata
}

#Cancel: {
	// event is the event name that will cancel this function
	event: string
	// timeout is the time at which the function can be cancelled, defaulting
	// to the max runtime length.
	timeout?: string
	// if is an optional expression to match when cancelling the function.
	if?: string
}
