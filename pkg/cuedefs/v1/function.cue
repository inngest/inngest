package v1

#Function: {
	id:   string
	name: string
	// triggers represent how the function is invoked.
	triggers: [...#Trigger]
	// A function can have > 1 step, which is an individual "action" called in a DAG.
	steps?: [string]: #Step
	// idempotency allows the specification of an idempotency key using event data.
	idempotency?: string
	// throttle allows you to throttle workflows, only running them a given number
	// of times (count) per period.  This can optionally include a throttle key,
	// which is used to 
	throttle?: {
		key?:   string
		count:  uint & >=1 | *1
		period: string
	}
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
	name: string | *""

	// Runtime represents how the function is executed.  Each runtime specifies data
	// necessary for executing the image, eg. if this is an externally hosted serverless
	// function via an API this will include the URL to use in order to invoke the function.
	runtime: #Runtime
}
