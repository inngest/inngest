package meta

const (
	AttrKeyPrefix = "_inngest."

	// Implementation
	PropagationKey = "user-otel-ctx"
	// Used when an internal error has occurred and may have resulted in a span
	// being mishandled or have incorrect or imcomplete data. In this case, we
	// should store any errors under this attribute.
	InternalError = "internal.error"

	// Top-level span names. These are the real (v2) production tracing
	// pipeline's names — pkg/tracing/v3 (the DuckDB dual-write POC's own
	// TracerProvider) has its own, separate const block for span names that
	// don't exist here, aliasing the ones below it as needed.
	SpanNameRun              = "executor.run"
	SpanNameStepDiscovery    = "executor.step.discovery"
	SpanNameStep             = "executor.step"
	SpanNameExecution        = "executor.execution"
	SpanNameStepFailed       = "executor.failed"
	SpanNameDefer            = "executor.defer"
	SpanNameDynamicExtension = "EXTEND"
	SpanNameUserland         = "userland"
	SpanNameMetadata         = "metadata"
	SpanNameNonStep          = "executor.nonstep" // TODO: better name

	// SDKExecutionSpanName is the name of the execution wrapper span
	// created by SDKs (e.g., "inngest.execution"). This span houses
	// metadata about the environment, versions, and scope, but should
	// not be displayed to the user directly.
	SDKExecutionSpanName = "inngest.execution"

	// Link attributes
	LinkAttributeType            = "_inngest.link.type"
	LinkAttributeTypeFollowsFrom = "follows_from"
)
