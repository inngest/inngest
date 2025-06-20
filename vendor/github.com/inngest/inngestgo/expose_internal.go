package inngestgo

import (
	"github.com/inngest/inngestgo/internal/fn"
)

// Expose concurrency options.

type (
	// ServableFunction defines a function which can be called by a handler's Serve method.
	ServableFunction = fn.ServableFunction

	// Trigger represents a function trigger - either an EventTrigger or a CronTrigger
	Trigger = fn.Trigger

	// FunctionOpts represents the options available to configure functions.  This includes
	// concurrency, retry, and flow control configuration.
	FunctionOpts = fn.FunctionOpts

	// FnDebounce represents debounce configuration.
	FnDebounce = fn.Debounce

	// FnThrottle represents concurrency over time.  This limits the maximum number of new
	// function runs over time.  Any runs over the limit are enqueued for the future.
	//
	// Note that this does not limit the number of steps executing at once and only limits
	// how frequently runs can start.  To limit the number of steps executing at once, use
	// concurrency limits.
	FnThrottle = fn.Throttle

	// FnRateLimit rate limits a function to a maximum number of runs over a given period.
	// Any runs over the limit are ignored and are NOT enqueued for the future.
	FnRateLimit = fn.RateLimit

	// FnTimeouts represents timeouts for the function. If any of the timeouts are hit, the function
	// will be marked as cancelled with a cancellation reason.
	FnTimeouts = fn.Timeouts

	// FnConcurrency represents a single concurrency limit for a function.  Concurrency limits
	// the number of running steps for a given key at a time.  Other steps will be enqueued
	// for the future and executed as soon as there's capacity.
	//
	// # Concurrency keys: virtual queues.
	//
	// The `Key` parameter is an optional CEL expression evaluated using the run's events.
	// The output from the expression is used to create new virtual queues, which limits
	// the number of runs for each virtual queue.
	//
	// For example, to limit the number of running steps for every account in your system,
	// you can send the `account_id` in the triggering event and use the following key:
	//
	// 		event.data.account_id
	//
	// Concurrency is then limited for each unique account_id field in parent events.
	FnConcurrency = fn.Concurrency

	// FnCancel represents a cancellation signal for a function.  When specified, this
	// will set up pauses which automatically cancel the function based off of matching
	// events and expressions.
	FnCancel = fn.Cancel

	// FnBatchEvents allows you run functions with a batch of events, instead of executing
	// a new run for every event received.
	//
	// The MaxSize option configures how many events will be collected into a batch before
	// executing a new function run.
	//
	// The timeout option limits how long Inngest waits for a batch to fill to MaxSize before
	// executing the function with a smaller batch.  This allows you to ensure functions run
	// without waiting for a batch to fill indefinitely.
	//
	// Inngest will execute your function as soon as MaxSize is reached or the Timeout is
	// reached.
	FnBatchEvents = fn.EventBatchConfig

	// FnSingleton configures a function to run as a singleton, ensuring that only one
	// instance of the function is active at a time for a given key. This is useful for
	// deduplicating runs or enforcing exclusive execution.
	//
	// If a new run is triggered while another instance with the same key is active,
	// it is skipped.
	FnSingleton = fn.Singleton

	// FnPriority allows you to dynamically execute some runs ahead or behind others based
	// on any data. This allows you to prioritize some jobs ahead of others without the need
	// for a separate queue. Some use cases for priority include:
	//
	// - Giving higher priority based on a user's subscription level, for example, free vs. paid users.
	// - Ensuring that critical work is executed before other work in the queue.
	// - Prioritizing certain jobs during onboarding to give the user a better first-run experience.
	FnPriority = fn.Priority
)

type (
	// Input is the input for a given function run.
	Input[T any] = fn.Input[T]

	// InputCtx is the additional context for a given function run, including the run ID,
	// function ID, step ID, attempt, etc.
	InputCtx = fn.InputCtx
)
