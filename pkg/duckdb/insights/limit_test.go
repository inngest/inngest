package insights

import (
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/parser"
	"github.com/stretchr/testify/require"
)

func TestAddDefaultLimitAddsWhenAbsent(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs")
	outcome, err := addDefaultLimit(stmt)
	require.NoError(t, err)
	require.Equal(t, limitDefaulted, outcome)
	require.Equal(t, "SELECT run_id FROM runs LIMIT 1000", parser.String(stmt))
}

func TestAddDefaultLimitLeavesExistingLimit(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs LIMIT 10")
	outcome, err := addDefaultLimit(stmt)
	require.NoError(t, err)
	require.Equal(t, limitUnchanged, outcome)
	require.Equal(t, "SELECT run_id FROM runs LIMIT 10", parser.String(stmt))
}

func TestAddDefaultLimitCapsExcessiveLimit(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs LIMIT 5000000")
	outcome, err := addDefaultLimit(stmt)
	require.NoError(t, err)
	require.Equal(t, limitCapped, outcome)
	require.Equal(t, "SELECT run_id FROM runs LIMIT 1000", parser.String(stmt))
}

func TestAddDefaultLimitLeavesLimitAtCeiling(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs LIMIT 1000")
	outcome, err := addDefaultLimit(stmt)
	require.NoError(t, err)
	require.Equal(t, limitUnchanged, outcome)
	require.Equal(t, "SELECT run_id FROM runs LIMIT 1000", parser.String(stmt))
}

func TestAddDefaultLimitCapsLimitAll(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs LIMIT ALL")
	outcome, err := addDefaultLimit(stmt)
	require.NoError(t, err)
	require.Equal(t, limitCapped, outcome)
	require.Equal(t, "SELECT run_id FROM runs LIMIT 1000", parser.String(stmt))
}

func TestAddDefaultLimitRejectsPercent(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs LIMIT 50 PERCENT")
	_, err := addDefaultLimit(stmt)
	require.ErrorContains(t, err, "PERCENT-based LIMIT is not supported")
}

func TestAddDefaultLimitRejectsComputedExpression(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs LIMIT 1 + 1")
	_, err := addDefaultLimit(stmt)
	require.ErrorContains(t, err, "LIMIT must be a literal, non-negative integer")
}

func TestAddDefaultLimitCapsFetch(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs FETCH FIRST 5000000 ROWS ONLY")
	outcome, err := addDefaultLimit(stmt)
	require.NoError(t, err)
	require.Equal(t, limitCapped, outcome)
	require.Equal(t, "SELECT run_id FROM runs LIMIT 1000", parser.String(stmt))
}
