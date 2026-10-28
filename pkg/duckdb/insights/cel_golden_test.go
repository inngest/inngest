package insights

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sq "github.com/doug-martin/goqu/v9"
	"github.com/sebdah/goldie/v2"
)

// celGoldenSection appends a labeled section (mirroring the reference
// pkg/insights.cel_test.go's INPUT/OUTPUT/PARAMS/ERROR fixture shape) to
// buf: a header line, the body, then a blank separator line. Plain text
// rather than JSON — a fixture full of "->"/"->>",  if JSON-encoded, is
// unreadable noise (json.Marshal escapes "&"/"<"/">" as &/</
// > by default), and this reads far better in a diff besides.
func celGoldenSection(buf *strings.Builder, header, body string) {
	fmt.Fprintf(buf, "%s\n%s\n\n", header, body)
}

func celGoldenFiltersSection(buf *strings.Builder, label string, filters []sq.Expression, err error) {
	if err != nil {
		celGoldenSection(buf, label+" ERROR", err.Error())
		return
	}
	sqlText, args, err := RenderWhereSQL(filters)
	if err != nil {
		celGoldenSection(buf, label+" ERROR", err.Error())
		return
	}
	celGoldenSection(buf, label+" SQL", sqlText)
	celGoldenSection(buf, label+" ARGS", fmt.Sprintf("%v", args))
}

func TestCELGolden(t *testing.T) {
	ctx := context.Background()
	dir := "testdata/cel"
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
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".cel" {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			input := string(raw)

			var buf strings.Builder
			celGoldenSection(&buf, "INPUT", input)

			eventFilters, eventErr := CELEventFilters(ctx, []string{input})
			celGoldenFiltersSection(&buf, "EVENT", eventFilters, eventErr)

			outputFilters, outputErr := CELOutputFilters(ctx, []string{input})
			celGoldenFiltersSection(&buf, "OUTPUT", outputFilters, outputErr)

			g.Assert(t, filepath.Base(name), []byte(strings.TrimRight(buf.String(), "\n")+"\n"))
		})
	}
}

// TestCELEventsTableGolden covers CELEventTableFilters (events-table search
// via eventsTableCELScope) separately from TestCELGolden's runsInputsCELScope
// coverage — a different query shape (real typed columns for event.id/name/
// v/ts, no "x" lambda parameter), not just a different phase of the same
// query.
func TestCELEventsTableGolden(t *testing.T) {
	ctx := context.Background()
	dir := "testdata/cel_events_table"
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
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".cel" {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			input := string(raw)

			var buf strings.Builder
			celGoldenSection(&buf, "INPUT", input)

			filters, err := CELEventTableFilters(ctx, []string{input})
			celGoldenFiltersSection(&buf, "EVENTS", filters, err)

			g.Assert(t, filepath.Base(name), []byte(strings.TrimRight(buf.String(), "\n")+"\n"))
		})
	}
}
