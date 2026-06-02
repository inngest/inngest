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
