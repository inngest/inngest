// pkg/duckdb/parser/adapter_window_test.go
package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdaptNamedWindowClause(t *testing.T) {
	stmt, err := ParseString("SELECT row_number() OVER win FROM t WINDOW win AS (PARTITION BY a ORDER BY b)")
	require.NoError(t, err)
	require.Len(t, stmt.Windows, 1)
	require.Equal(t, "win", stmt.Windows[0].Name)
	require.Len(t, stmt.Windows[0].Spec.PartitionBy, 1)
	require.NotNil(t, stmt.Windows[0].Spec.OrderBy)
}

func TestAdaptMultipleNamedWindows(t *testing.T) {
	stmt, err := ParseString("SELECT a FROM t WINDOW w1 AS (PARTITION BY a), w2 AS (PARTITION BY b)")
	require.NoError(t, err)
	require.Len(t, stmt.Windows, 2)
	require.Equal(t, "w1", stmt.Windows[0].Name)
	require.Equal(t, "w2", stmt.Windows[1].Name)
}

func TestAdaptQualify(t *testing.T) {
	stmt, err := ParseString("SELECT a, row_number() OVER (ORDER BY a) AS rn FROM t QUALIFY rn = 1")
	require.NoError(t, err)
	require.NotNil(t, stmt.Qualify)
}
