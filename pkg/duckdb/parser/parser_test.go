// pkg/duckdb/parser/parser_test.go
package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	files, err := filepath.Glob("testdata/queries/*.sql")
	require.NoError(t, err)
	require.NotEmpty(t, files, "no fixtures discovered; check the glob")

	g := goldie.New(t,
		goldie.WithFixtureDir("testdata/queries/parser"),
		goldie.WithNameSuffix(".sql.out"),
		goldie.WithDiffEngine(goldie.ColoredDiff),
	)

	for _, f := range files {
		f := f
		name := strings.TrimSuffix(filepath.Base(f), ".sql")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(f)
			require.NoError(t, err)

			var out strings.Builder
			for _, stmt := range strings.Split(string(src), ";\n\n") {
				stmt = strings.TrimSpace(stmt)
				if stmt == "" {
					continue
				}
				out.WriteString("-- " + stmt + "\n")
				parsed, err := ParseString(stmt)
				if err != nil {
					out.WriteString("ERROR: " + err.Error() + "\n\n")
					continue
				}
				out.WriteString(Dump(parsed))
				out.WriteString("\n")
			}
			g.Assert(t, name, []byte(out.String()))
		})
	}
}

func TestParseStringRecoversPanicAsError(t *testing.T) {
	// DESCRIBE as a standalone statement stays out of scope permanently
	// (see Task 11's plan entry) — the adapter panics internally;
	// ParseString must turn that into a plain error, never let it cross
	// out. (This example has been swapped three times now, from GROUP BY
	// to JOIN to WITH to PIVOT, as each gained real support in a later
	// task — this one shouldn't need to move again.)
	_, err := ParseString("DESCRIBE t")
	require.Error(t, err)
}

func TestParseStringRejectsMalformedSQL(t *testing.T) {
	_, err := ParseString("SELEC * FROM")
	require.Error(t, err)
}
