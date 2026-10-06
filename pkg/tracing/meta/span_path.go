package meta

// SpanPathElement is one span group a step was called in. A step's span path
// lists its groups outermost first, as the SDK sends them in `opts.span`.
type SpanPathElement struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
