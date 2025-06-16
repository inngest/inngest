package meta

type SpanReference struct {
	TraceParent string `json:"tp"`
	TraceState  string `json:"ts"`

	// If this is a dynamic span, store enough information to be able to safely
	// extend the span with only this context.
	DynamicSpanTraceParent string `json:"dstp,omitempty"`
	DynamicSpanTraceState  string `json:"dsts,omitempty"`
	DynamicSpanID          string `json:"dsid,omitempty"`
}
