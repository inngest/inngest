package v1

#Function: {
	id:           string
	name:         string
	concurrency?: uint

	// triggers represent how the function is invoked.
	triggers: [...#Trigger]

	// A function can have > 1 step, which is an individual "action" called in a DAG.
	steps?: [ID=string]: #Step & {id: ID}

	// idempotency allows the specification of an idempotency key using event data.
	// If specified, this overrides the throttle object.
	idempotency?: string
	// throttle allows you to throttle workflows, only running them a given number
	// of times (count) per period.  This can optionally include a throttle key,
	// which is used to  further constrain throttling similar to idempotency.
	throttle?: {
		key?:   string
		count:  uint & >=1 | *1
		period: string
	}

	cancel?: [...#Cancel]
}

#EventTrigger: {
	// Event is the name of the event that triggers the function.
	event: string

	// Expression allows you to write custom expressions for specifying conditions
	//  for the trigger.  For example, you may want a function to run if an order
	// is above a specific value (eg. "event.data.total >= 500"), or if the event
	// is a specific version (eg. "event.version >= '2').
	expression?: string

	// Definition stores the type definitions for the event.
	//
	// Inngest is fully typed, and events may come from integrations with built-in
	// event schemas or from your own API. In many cases you'll write functions
	// with events which are not yet stored within Inngest.  We allow you to store
	// a type for the event directly here.
	definition?: #EventDefinition
}

#CronTrigger: {
	cron: string
}

#Trigger: #EventTrigger | #CronTrigger

#EventDefinition: {
	format: "cue" | "json-schema"
	// Whether this is synced within Inngest.  This allows us to always fetch the
	// latest version of an event.
	synced: bool
	// The definition may be a cue type embedded within the definition, or
	// it may be a JSON object representing a JSON schema.
	//
	// If this is a string, it is assumed that this represents a filepath
	// to load the definition from.
	def: string | {[string]: _}
}

// Step represents a single action within a function.  An action is an individual unit
// of code which is scheduled as part of the function execution.
#Step: {
	id:   string
	name: string | *""

	// path represents the location on disk for the step defintiion.  A single function
	// may have >1 docker-based step.  This lists the directory which contains the step.
	path: string | *""

	// Runtime represents how the function is executed.  Each runtime specifies data
	// necessary for executing the image, eg. if this is an externally hosted serverless
	// function via an API this will include the URL to use in order to invoke the function.
	runtime?: #Runtime

	// after specifies that this step should run after each of the following steps.
	//
	// If more than one item is supplied in this array, the step will run multiple times after
	// each preceeding step finishes.
	after?: [...#After]

	// version is the version constraint for the step when resolving the action to
	// run.
	version?: {
		major?: uint
		minor?: uint
	}

	retries?: {
		attempts?: int & >=0 & <=20
	}
}

#After: {
	step: string | "$trigger"
	// TODO: support Promise.all() like support in which we wait after all steps
	// specified in an array are finished before running this once.
	// steps?: [...string]
	if?: string
	// wait allows you to delay a step from running for a set amount of time, eg.
	// to delay a step from running you can set wait to "10m".  This will enqueue
	// the step to run after 10 minutes.
	wait?: string

	// async allows you to specify an event that must be received within a specific
	// amount of time (ttl) to continue with the specified step.
	async?: {
		ttl:    string
		event:  string
		match?: string
		// onTimeout specifies that this edge should be traversed on timeout only,
		// if the event is not received within the TTL.
		onTimeout?: bool
	}
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
