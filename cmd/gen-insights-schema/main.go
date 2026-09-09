// Command gen-insights-schema dumps pkg/duckdb/insights' static schema
// registry -- the six logical tables (with their columns, types, hints, and
// descriptions) and the allowed SQL function list (with descriptions and
// DuckDB docs links) -- as a single JSON file, embedded into the dev-server
// UI's SQL editor at build time rather than fetched live over GQL. This
// keeps the UI's schema explorer and autocomplete in sync with the real
// backend registry by construction: there is exactly one source of truth
// (pkg/duckdb/insights), and this tool is the only thing that reads it to
// produce the UI's copy.
//
// Usage:
//
//	go run ./cmd/gen-insights-schema -o path/to/output.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/inngest/inngest/pkg/duckdb/insights"
)

// schemaDump is the top-level JSON shape written to -o.
type schemaDump struct {
	Tables    []tableDump    `json:"tables"`
	Functions []functionDump `json:"functions"`
}

type tableDump struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Columns     []columnDump `json:"columns"`
}

type columnDump struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Hint        *string `json:"hint,omitempty"`
	Description string  `json:"description"`
}

type functionDump struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	DocsURL     string `json:"docsUrl"`
}

func main() {
	output := flag.String("o", "", "output JSON file path (required)")
	flag.Parse()

	if *output == "" {
		fmt.Fprintln(os.Stderr, "error: -o output file is required")
		os.Exit(1)
	}

	dump := schemaDump{
		Tables:    tableDumps(),
		Functions: functionDumps(),
	}

	b, err := json.MarshalIndent(dump, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling schema: %v\n", err)
		os.Exit(1)
	}
	b = append(b, '\n')

	if err := os.WriteFile(*output, b, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing %s: %v\n", *output, err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s with %d table(s) and %d function(s)\n", *output, len(dump.Tables), len(dump.Functions))
}

func tableDumps() []tableDump {
	tables := insights.AllTableSchemas()
	out := make([]tableDump, len(tables))
	for i, tbl := range tables {
		cols := make([]columnDump, len(tbl.Columns))
		for j, col := range tbl.Columns {
			cols[j] = columnDump{
				Name:        col.Name,
				Type:        col.Type.String(),
				Hint:        hintPtr(col.Hint),
				Description: col.Description,
			}
		}
		out[i] = tableDump{Name: tbl.Name, Description: tbl.Description, Columns: cols}
	}
	return out
}

// hintPtr omits the field entirely for HintNone rather than serializing an
// empty string -- matches ColumnHint.String()'s own "" == no hint contract.
func hintPtr(h insights.ColumnHint) *string {
	if h == insights.HintNone {
		return nil
	}
	s := h.String()
	return &s
}

func functionDumps() []functionDump {
	fns := insights.AllFunctionSchemas()
	out := make([]functionDump, len(fns))
	for i, fn := range fns {
		out[i] = functionDump{Name: fn.Name, Description: fn.Description, DocsURL: fn.DocsURL}
	}
	return out
}
