package insights

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/driver"
)

// guardRenderedSQL gates checkRenderedSQL; only tests turn it off, to prove
// the DuckDB-level sandbox holds on its own (export_test.go).
var guardRenderedSQL = true

// syntaxFunctions are functions DuckDB's own parser desugars allowlisted
// syntax into (json_serialize_sql shows them as ordinary function calls),
// so they appear in a rendered query even though a user can't call them by
// name: EXTRACT -> date_part, [..] -> list_value, {..} -> struct_pack,
// SIMILAR TO -> regexp_full_match, INTERVAL 'n' unit -> to_<unit>s, etc.
var syntaxFunctions = map[string]bool{
	"count_star":        true,
	"date_part":         true,
	"list_value":        true,
	"struct_pack":       true,
	"row":               true,
	"regexp_full_match": true,
	"trunc":             true,
	"list_apply":        true, // [x FOR x IN l]
	"list_filter":       true, // [x FOR x IN l IF p]
	"to_years":          true,
	"to_months":         true,
	"to_weeks":          true,
	"to_days":           true,
	"to_hours":          true,
	"to_minutes":        true,
	"to_seconds":        true,
	"to_milliseconds":   true,
	"to_microseconds":   true,
}

// checkRenderedSQL is a last-line check on the exact SQL Execute is about to
// run, independent of this package's own parser and renderer: DuckDB parses
// it (json_serialize_sql, which never executes anything), and every function
// and table it references must be one Transpile could legitimately have
// produced — an allowlisted function, a syntax desugaring of one
// (syntaxFunctions), or an operator; a logical table's macro (only as a
// FROM-clause table function); or a CTE. A
// renderer or lexer gap that makes DuckDB read something other than what
// validate checked (a smuggled read_text, query(), or a bare physical
// inngest.* table that skips env scoping) is rejected here instead of run.
func checkRenderedSQL(ctx context.Context, db *sql.DB, query string, cat catalog) error {
	if !guardRenderedSQL {
		return nil
	}
	var serialized string
	if err := db.QueryRowContext(ctx, "SELECT json_serialize_sql(?)::VARCHAR;", query).Scan(&serialized); err != nil {
		return fmt.Errorf("checking rendered query: %w", err)
	}
	var tree struct {
		Error        bool   `json:"error"`
		ErrorMessage string `json:"error_message"`
		Statements   []any  `json:"statements"`
	}
	if err := json.Unmarshal([]byte(serialized), &tree); err != nil {
		return fmt.Errorf("checking rendered query: decoding: %w", err)
	}
	if tree.Error {
		return fmt.Errorf("checking rendered query: %s", tree.ErrorMessage)
	}
	if len(tree.Statements) != 1 {
		return fmt.Errorf("rendered query has %d statements, expected exactly 1", len(tree.Statements))
	}

	ctes := map[string]bool{}
	collectCTENames(tree.Statements[0], ctes)
	macros := map[string]bool{}
	for _, t := range cat.tables {
		if t.view != "" {
			macros[t.view] = true
		}
	}
	return checkRenderedNode(tree.Statements[0], ctes, macros)
}

// collectCTENames gathers every CTE name defined anywhere in n. Scope is
// ignored: a name used outside its CTE's scope still can't reach a physical
// table (those live under a schema, which checkRenderedNode rejects), and
// DuckDB itself rejects the out-of-scope reference.
func collectCTENames(n any, into map[string]bool) {
	switch v := n.(type) {
	case map[string]any:
		if cm, ok := v["cte_map"].(map[string]any); ok {
			entries, _ := cm["map"].([]any)
			for _, e := range entries {
				if em, ok := e.(map[string]any); ok {
					if k, ok := em["key"].(string); ok {
						into[strings.ToLower(k)] = true
					}
				}
			}
		}
		for _, child := range v {
			collectCTENames(child, into)
		}
	case []any:
		for _, child := range v {
			collectCTENames(child, into)
		}
	}
}

func checkRenderedNode(n any, ctes, macros map[string]bool) error {
	switch v := n.(type) {
	case map[string]any:
		// A FROM-clause table function may only be a logical table's macro
		// or UNNEST (the one table function scope.go admits); its arguments
		// are then checked like any other expression.
		if v["type"] == "TABLE_FUNCTION" {
			fn, _ := v["function"].(map[string]any)
			name, schema := strings.ToLower(str(fn["function_name"])), str(fn["schema"])
			macro := schema == driver.DuckLakeAlias && macros[name]
			unnest := schema == "" && name == "unnest"
			if str(fn["catalog"]) != "" || !macro && !unnest {
				return fmt.Errorf("rendered query reads from table function %q, which is not allowed", str(fn["function_name"]))
			}
			if err := checkRenderedNode(fn["arguments"], ctes, macros); err != nil {
				return err
			}
			for k, child := range v {
				if k == "function" {
					continue
				}
				if err := checkRenderedNode(child, ctes, macros); err != nil {
					return err
				}
			}
			return nil
		}
		if name, ok := v["function_name"].(string); ok {
			if err := checkRenderedFunction(name, str(v["schema"]), str(v["catalog"])); err != nil {
				return err
			}
		}
		if name, ok := v["table_name"].(string); ok {
			if str(v["schema_name"]) != "" || str(v["catalog_name"]) != "" || !ctes[strings.ToLower(name)] {
				return fmt.Errorf("rendered query references table %q, which is not a logical table or CTE", name)
			}
		}
		for _, child := range v {
			if err := checkRenderedNode(child, ctes, macros); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range v {
			if err := checkRenderedNode(child, ctes, macros); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkRenderedFunction(name, schema, catalog string) error {
	lower := strings.ToLower(name)
	if catalog == "" && schema == "" {
		if _, ok := allowedFunctions[lower]; ok || syntaxFunctions[lower] || isOperatorName(lower) {
			return nil
		}
	}
	qualified := name
	if schema != "" {
		qualified = schema + "." + qualified
	}
	return fmt.Errorf("rendered query calls function %q, which is not allowed", qualified)
}

// isOperatorName reports whether name is a symbolic operator (+, ~~, ->>,
// ||, ...), which DuckDB represents as a function call named after the
// operator itself.
func isOperatorName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if r == '_' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return false
		}
	}
	return true
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
