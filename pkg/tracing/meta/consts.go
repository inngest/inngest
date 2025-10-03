package meta

const (
	AttrKeyPrefix = "_inngest."

	// Implementation
	PropagationKey = "user-otel-ctx"
	// Used when an internal error has occurred and may have resulted in a span
	// being mishandled or have incorrect or imcomplete data. In this case, we
	// should store any errors under this attribute.
	InternalError = "internal.error"

	// Top-level span names
	SpanNameRun              = "executor.run"
	SpanNameStepDiscovery    = "executor.step.discovery"
	SpanNameStep             = "executor.step"
	SpanNameExecution        = "executor.execution"
	SpanNameStepFailed       = "executor.failed"
	SpanNameDynamicExtension = "EXTEND"
	SpanNameUserland         = "userland"

	// Link attributes
	LinkAttributeType            = "_inngest.link.type"
	LinkAttributeTypeFollowsFrom = "follows_from"
)
