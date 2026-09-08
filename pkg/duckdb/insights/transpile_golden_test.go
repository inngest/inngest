package insights

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sebdah/goldie/v2"
)

// transpileGoldenCase mirrors pkg/duckdb/parser's TestParser golden fixture
// shape, capturing everything docs/plans/010's Testing section calls for:
// the rewritten SQL, QueryInfo, and any diagnostic/error.
type transpileGoldenCase struct {
	SQL          string                 `json:"sql,omitempty"`
	Args         []any                  `json:"args,omitempty"`
	PrimaryTable string                 `json:"primaryTable,omitempty"`
	Tables       []string               `json:"tables,omitempty"`
	Limited      bool                   `json:"limited,omitempty"`
	ColumnHints  []int                  `json:"columnHints,omitempty"`
	Diagnostics  []diagnosticGoldenCase `json:"diagnostics,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

type diagnosticGoldenCase struct {
	Start    string `json:"start"`
	End      string `json:"end"`
	Severity int    `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

func TestTranspileGolden(t *testing.T) {
	dir := "testdata/queries"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	g := goldie.New(t,
		goldie.WithFixtureDir(filepath.Join(dir, "transpile")),
		goldie.WithNameSuffix(".out"),
		goldie.WithDiffEngine(goldie.ColoredDiff),
	)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}

			var out transpileGoldenCase
			tr, err := Transpile(string(raw), testAccountID, testEnvID)
			if err != nil {
				out.Error = err.Error()
			} else {
				hints := make([]int, len(tr.ColumnHints))
				for i, h := range tr.ColumnHints {
					hints[i] = int(h)
				}
				diags := make([]diagnosticGoldenCase, len(tr.Diagnostics))
				for i, d := range tr.Diagnostics {
					diags[i] = diagnosticGoldenCase{
						Start: d.Start.String(), End: d.End.String(),
						Severity: int(d.Severity), Code: d.Code, Message: d.Message,
					}
				}
				out = transpileGoldenCase{
					SQL: tr.SQL, Args: tr.Args, PrimaryTable: tr.PrimaryTable,
					Tables: tr.Tables, Limited: tr.Limited, ColumnHints: hints,
					Diagnostics: diags,
				}
			}

			data, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			g.Assert(t, filepath.Base(name), data)
		})
	}
}
