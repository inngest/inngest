package duckdb

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// TableMacro is one of views.sql's `CREATE MACRO ... AS TABLE`s, parsed, for
// callers that need to see through a macro call to what it reads (IE
// federate's column lineage).
type TableMacro struct {
	// Name is the macro's name in schema.MacroSchema, IE "insights_runs".
	Name string
	// Params are its parameters in declaration order. A positional one has
	// no default; a named one (`p := 'x'`) defaults to Defaults[p].
	Params   []string
	Defaults map[string]string
	// Body is its SELECT. It reads its tables through query_table(param)
	// or through other macros, and its parameters appear as bare Idents.
	Body *parser.SelectStatement
}

var tableMacros = sync.OnceValues(func() (map[string]*TableMacro, error) {
	var b strings.Builder
	// The collapsing raw_runs: its QUALIFY reads a superset of the flat
	// one's columns, so what the macros read holds for either mode.
	if err := viewsTmpl.Execute(&b, struct{ CollapseRuns bool }{true}); err != nil {
		return nil, fmt.Errorf("duckdb: rendering views: %w", err)
	}
	out := map[string]*TableMacro{}
	for _, stmt := range splitStatements(b.String()) {
		m := macroHeader.FindStringSubmatch(stmt)
		if m == nil {
			continue
		}
		macro := &TableMacro{Name: m[1], Defaults: map[string]string{}}
		for _, p := range strings.Split(m[2], ",") {
			name, def, named := strings.Cut(p, ":=")
			name = strings.TrimSpace(name)
			macro.Params = append(macro.Params, name)
			if !named {
				continue
			}
			lit, err := parser.ParseString("SELECT " + def)
			if err != nil {
				return nil, fmt.Errorf("duckdb: macro %s: parameter %s's default: %w", macro.Name, name, err)
			}
			l, ok := lit.Columns[0].Expr.(*parser.Literal)
			if !ok || l.Kind != parser.LitString {
				return nil, fmt.Errorf("duckdb: macro %s: parameter %s's default isn't a string", macro.Name, name)
			}
			macro.Defaults[name] = l.Text
		}
		body, err := parser.ParseString(strings.TrimSuffix(strings.TrimSpace(m[3]), ";"))
		if err != nil {
			return nil, fmt.Errorf("duckdb: macro %s: %w", macro.Name, err)
		}
		macro.Body = body
		out[macro.Name] = macro
	}
	return out, nil
})

// macroHeader matches a table macro's statement: its name (w/o the schema),
// its parameter list and its body.
var macroHeader = regexp.MustCompile(`(?s)^\s*CREATE OR REPLACE MACRO \w+\.(\w+)\((.*?)\) AS TABLE\s+(.*)$`)

// TableMacros returns views.sql's table macros by name, parsed.
func TableMacros() (map[string]*TableMacro, error) { return tableMacros() }
