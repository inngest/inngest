package meta

// TODO
type SpanMetadata struct {
	TraceParent   string `json:"tp"`
	TraceState    string `json:"ts"`
	DynamicSpanID string `json:"dsid,omitempty"`
}
