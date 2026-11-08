package insights

import (
	"regexp"
	"slices"
	"strings"
	"sync"
)

// ExecutionDiagnostic is e.Diagnostic() w/ its message made user facing
// (see UserFacingExecutionMessage). It's what both the REST and GQL
// endpoints report for an *ExecutionError.
func ExecutionDiagnostic(e *ExecutionError) Diagnostic {
	d := e.Diagnostic()
	d.Message = UserFacingExecutionMessage(d.Message)
	return d
}

var (
	macroRefOnce sync.Once
	macroRefRe   *regexp.Regexp
	macroToTable map[string]string

	// lineContextRe matches DuckDB's "LINE N: <sql>" excerpt and the caret
	// line under it.
	lineContextRe = regexp.MustCompile(`(?m)\n*^LINE \d+:.*(\n[ \t]*\^+[ \t]*)?$`)
)

// UserFacingExecutionMessage rewrites DuckDB error text about the SQL
// Execute actually ran (remapTables' `inngest.insights_runs('<account>',
// '<env>') AS runs`, ...) back to what the user wrote:
//
//   - every macro reference, w/ its scoping arguments (whole, or truncated
//     by DuckDB's "..."), becomes its logical table name, e.g.
//     `inngest.insights_runs('...', '...')` -> `runs`.
//   - DuckDB's "LINE N: ..." excerpt and its caret are dropped: they quote
//     the rewritten SQL and point at a column within it, not within the
//     user's query (see ExecutionError's doc comment), and can start
//     mid-way through the macro's arguments.
func UserFacingExecutionMessage(msg string) string {
	macroRefOnce.Do(buildMacroRefRe)
	msg = lineContextRe.ReplaceAllString(msg, "")
	return macroRefRe.ReplaceAllStringFunc(msg, func(m string) string {
		return macroToTable[macroRefRe.FindStringSubmatch(m)[1]]
	})
}

func buildMacroRefRe() {
	macroToTable = make(map[string]string, len(logicalTables))
	views := make([]string, 0, len(logicalTables))
	for name, tbl := range logicalTables {
		macroToTable[tbl.view] = name
		views = append(views, regexp.QuoteMeta(tbl.view))
	}
	// Longest first so e.g. insights_step_attempts never matches as
	// insights_steps (the \b already prevents that, this is just belt and
	// braces).
	slices.SortFunc(views, func(a, b string) int { return len(b) - len(a) })
	// An optional `inngest.` schema, the macro name, then optionally its
	// argument list: either complete, or cut off by DuckDB's "...".
	macroRefRe = regexp.MustCompile(`(?:\binngest\.)?\b(` + strings.Join(views, "|") + `)\b(?:\s*\((?:[^()]*\)|[^()]*?\.\.\.))?`)
}
