package consts

const (
	EventReceivedName  = "event/event.received"
	InternalNamePrefix = "inngest/"

	FnFailedName    = InternalNamePrefix + "function.failed"
	FnFinishedName  = InternalNamePrefix + "function.finished"
	FnCancelledName = InternalNamePrefix + "function.cancelled"
	FnInvokeName    = InternalNamePrefix + "function.invoked"
	FnCronName      = InternalNamePrefix + "scheduled.timer"
	HttpRequestName = InternalNamePrefix + "http.request"
)
