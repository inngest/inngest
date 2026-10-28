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

// SubscriptionParkName starts the name of the index of the markers of a run that
// is parked on a step.subscribe subscription: the name, then the run and a hash of
// the subscription. The string is the one the earlier wait pauses used.
const SubscriptionParkName = InternalNamePrefix + "subscription.wait"
