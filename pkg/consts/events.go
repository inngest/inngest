package consts

const (
	EventReceivedName  = "event/event.received"
	InternalNamePrefix = "inngest/"

	FnFailedName        = InternalNamePrefix + "function.failed"
	FnFinishedName      = InternalNamePrefix + "function.finished"
	FnCancelledName     = InternalNamePrefix + "function.cancelled"
	FnInvokeName        = InternalNamePrefix + "function.invoked"
	FnCronName          = InternalNamePrefix + "scheduled.timer"
	FnDeferScheduleName = InternalNamePrefix + "deferred.schedule"
	HttpRequestName     = InternalNamePrefix + "http.request"
)

// SubscriptionWaitName names the index of the pauses a run blocks on while it
// waits for the next message of a step.subscribe subscription. No event has
// this name.
const SubscriptionWaitName = InternalNamePrefix + "subscription.wait"
