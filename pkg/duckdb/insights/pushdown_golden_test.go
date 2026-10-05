package insights

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/federate"
	"github.com/sebdah/goldie/v2"
)

// pushdownGoldenCase is how one query's WHERE clause is pushed down: every
// predicate extracted per logical table, and what became of each on its
// federated physical table. Conjuncts that weren't extracted at all (OR,
// NOT, functions, casts, ...) don't appear: they're evaluated only in
// DuckDB.
type pushdownGoldenCase struct {
	Description string                               `json:"description,omitempty"`
	Mode        string                               `json:"mode"`
	Tables      map[string][]pushdownGoldenPredicate `json:"tables,omitempty"`
	Error       string                               `json:"error,omitempty"`
}

type pushdownGoldenPredicate struct {
	Logical  string `json:"logical"`
	Physical string `json:"physical,omitempty"`
	// Pushed is whether it's forwarded to the delta (whose streamer applies
	// it if it can represent it); Kept says why not.
	Pushed bool   `json:"pushed"`
	Kept   string `json:"kept,omitempty"`
}

// TestPushdownGolden runs each testdata/pushdown/*.sql query through the
// transpiler and records its pushdown. Leading "-- " lines are the case's
// description; "-- mode: raw" transpiles it in raw mode.
func TestPushdownGolden(t *testing.T) {
	dir := "testdata/pushdown"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	g := goldie.New(t,
		goldie.WithFixtureDir(filepath.Join(dir, "golden")),
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
			out := pushdownGoldenCase{Mode: "product"}
			var desc, body []string
			for _, l := range strings.Split(string(raw), "\n") {
				if c, ok := strings.CutPrefix(l, "-- "); ok && len(body) == 0 {
					if c == "mode: raw" {
						out.Mode = "raw"
					} else {
						desc = append(desc, c)
					}
					continue
				}
				body = append(body, l)
			}
			out.Description = strings.Join(desc, " ")
			sql := strings.TrimSpace(strings.Join(body, "\n"))

			var tr *TranspileResult
			if out.Mode == "raw" {
				tr, err = TranspileRaw(sql)
			} else {
				tr, err = Transpile(sql, testAccountID, testEnvID)
			}
			if err != nil {
				out.Error = err.Error()
			} else {
				out.Tables = pushdownGolden(tr)
			}

			data, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			g.Assert(t, name, data)
		})
	}
}

func pushdownGolden(tr *TranspileResult) map[string][]pushdownGoldenPredicate {
	tables := make([]string, 0, len(tr.Pushdown))
	for lt := range tr.Pushdown {
		tables = append(tables, string(lt))
	}
	sort.Strings(tables)
	out := map[string][]pushdownGoldenPredicate{}
	for _, lt := range tables {
		for _, p := range tr.Pushdown[federate.Table(lt)] {
			pt, phys, kept := physicalPredicate(federate.Table(lt), p, tr.cat)
			gp := pushdownGoldenPredicate{Logical: formatPredicate("", p), Pushed: kept == "", Kept: kept}
			if phys.Column != "" || phys.IsBool() {
				gp.Physical = formatPredicate(string(pt), phys)
			}
			out[lt] = append(out[lt], gp)
		}
	}
	return out
}

// formatPredicate renders p as the SQL it stands for.
func formatPredicate(table string, p federate.Predicate) string {
	switch p.Op {
	case federate.OpAnd, federate.OpOr:
		parts := make([]string, len(p.Args))
		for i, a := range p.Args {
			parts[i] = formatPredicate(table, a)
		}
		return "(" + strings.Join(parts, " "+string(p.Op)+" ") + ")"
	case federate.OpNot:
		return "NOT " + formatPredicate(table, p.Args[0])
	}
	col := p.Column
	if table != "" {
		col = table + "." + col
	}
	if p.Path != "" {
		col = fmt.Sprintf("%s->>%s", col, sqlLiteral(p.Path))
	}
	if p.Op == federate.OpIn {
		vals, _ := p.Value.([]any)
		lits := make([]string, len(vals))
		for i, v := range vals {
			lits[i] = sqlLiteral(v)
		}
		return fmt.Sprintf("%s IN (%s)", col, strings.Join(lits, ", "))
	}
	return fmt.Sprintf("%s %s %s", col, p.Op, sqlLiteral(p.Value))
}

func sqlLiteral(v any) string {
	if s, ok := v.(string); ok {
		return "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
	return fmt.Sprint(v)
}
