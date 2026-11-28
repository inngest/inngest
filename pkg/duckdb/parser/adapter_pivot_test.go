// pkg/duckdb/parser/adapter_pivot_test.go
package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdaptTablePivotClause(t *testing.T) {
	stmt, err := ParseString("SELECT * FROM sales PIVOT (sum(amount) FOR quarter IN (q1, q2, q3, q4))")
	require.NoError(t, err)
	ref, ok := stmt.From.Refs[0].(*PivotRef)
	require.True(t, ok)
	_, ok = ref.Source.(*BaseTableRef)
	require.True(t, ok)
	require.Len(t, ref.Columns, 1)
	require.Len(t, ref.For, 1)
	require.Equal(t, []string{"q1", "q2", "q3", "q4"}, ref.For[0].In)
}

func TestAdaptTablePivotClauseWithGroupBy(t *testing.T) {
	stmt, err := ParseString("SELECT * FROM sales PIVOT (sum(amount) FOR quarter IN (q1, q2) GROUP BY region)")
	require.NoError(t, err)
	ref := stmt.From.Refs[0].(*PivotRef)
	require.Equal(t, []string{"region"}, ref.GroupBy)
}

func TestAdaptTableUnpivotClause(t *testing.T) {
	// UNPIVOT (value_col FOR name_col IN (source_cols...)): "amount" is the
	// new VALUE column name (Header), "quarter" is the new NAME column
	// name (Names), and (q1..q4) are the source columns (Columns).
	stmt, err := ParseString("SELECT * FROM wide UNPIVOT (amount FOR quarter IN (q1, q2, q3, q4))")
	require.NoError(t, err)
	ref, ok := stmt.From.Refs[0].(*UnpivotRef)
	require.True(t, ok)
	require.Equal(t, []string{"amount"}, ref.Header)
	require.Equal(t, []string{"quarter"}, ref.Names)
	require.Len(t, ref.Columns, 4)
}

func TestAdaptStandalonePivotStatement(t *testing.T) {
	stmt, err := ParseString("PIVOT sales ON quarter USING sum(amount)")
	require.NoError(t, err)
	require.Nil(t, stmt.Columns)
	ref, ok := stmt.From.Refs[0].(*PivotRef)
	require.True(t, ok)
	require.Len(t, ref.Columns, 1)
	require.Len(t, ref.For, 1)
}

func TestAdaptStandaloneUnpivotStatement(t *testing.T) {
	stmt, err := ParseString("UNPIVOT wide ON q1, q2, q3, q4")
	require.NoError(t, err)
	ref, ok := stmt.From.Refs[0].(*UnpivotRef)
	require.True(t, ok)
	require.Len(t, ref.Columns, 4)
}
