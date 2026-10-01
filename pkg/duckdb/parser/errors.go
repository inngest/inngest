package parser

import "fmt"

// ParseError is returned by ParseString when the input doesn't parse as a
// DuckDB SELECT statement. Pos locates the furthest point the parser
// reached before giving up (see peg.Parser.Parse's doc comment on
// furthestFail) — a standard PEG error-reporting heuristic, not
// necessarily the single "most helpful" position for every input, but
// always a real position in the source worth pointing a diagnostic at.
type ParseError struct {
	Pos     Position
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Pos.Line, e.Pos.Column, e.Message)
}
