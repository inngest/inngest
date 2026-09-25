package grammar

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// wantLines guards against a silently truncated or failed re-fetch — see
// Task 1/Task 15 of docs/plans/011-duckdb-select-parser-plan.md.
var wantLines = map[string]int{
	"vendor/statements/select.gram":            215,
	"vendor/statements/expression.gram":        342,
	"vendor/statements/common.gram":            196,
	"vendor/statements/pivot.gram":             24,
	"vendor/statements/describe.gram":          39,
	"vendor/statements/base.gram":              19,
	"vendor/keywords/reserved_keyword.list":    75,
	"vendor/keywords/unreserved_keyword.list":  339,
	"vendor/keywords/column_name_keyword.list": 54,
	"vendor/keywords/func_name_keyword.list":   29,
	"vendor/keywords/type_name_keyword.list":   31,
}

func TestVendoredGrammarShape(t *testing.T) {
	version, err := os.ReadFile("vendor/VERSION")
	require.NoError(t, err)
	require.Equal(t, "f931913e6b03beaf75df0f56695900f765dab426", strings.TrimSpace(string(version)))

	for path, want := range wantLines {
		data, err := os.ReadFile(path)
		require.NoErrorf(t, err, "reading %s", path)
		got := strings.Count(string(data), "\n")
		require.Equalf(t, want, got, "%s: line count drifted — re-vendor was incomplete or upstream changed", path)
	}
}
