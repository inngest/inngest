package insights

import "github.com/inngest/inngest/pkg/duckdb/parser"

// DiagnosticSeverity mirrors the reference ClickHouse pkg/insights'
// diagnostics severity levels.
type DiagnosticSeverity int

const (
	DiagnosticInfo DiagnosticSeverity = iota
	DiagnosticWarning
	DiagnosticError
)

// Diagnostic is a non-fatal, position-aware note about a query: something
// worth surfacing but that doesn't reject it the way a *ValidationError
// does (a ValidationError stops the pipeline; a Diagnostic rides along in
// TranspileResult next to a successful result). Start/End are a full
// source range, matching parser.Node's Pos()/End(), so a UI can render it
// as a Monaco range marker rather than a single-position caret.
type Diagnostic struct {
	Start    parser.Position
	End      parser.Position
	Severity DiagnosticSeverity
	Code     string
	Message  string
}

func diagnosticAt(n parser.Node, severity DiagnosticSeverity, code, message string) Diagnostic {
	return Diagnostic{Start: n.Pos(), End: n.End(), Severity: severity, Code: code, Message: message}
}

// Diagnostic converts a *ValidationError into an ERROR-severity Diagnostic,
// for a caller (e.g. the GQL resolver) that wants to render "the query is
// invalid" as an inline marker instead of a bare top-level error. Transpile
// itself still rejects invalid queries outright; this doesn't change that.
// Zero-width range (Start == End == e.Pos): ValidationError only carries a
// single position, not a span.
func (e *ValidationError) Diagnostic() Diagnostic {
	return Diagnostic{Start: e.Pos, End: e.Pos, Severity: DiagnosticError, Code: "validation-error", Message: e.Message}
}
