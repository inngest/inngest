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

func TestAddDefaultLimitDefaultsOffsetOnly(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs OFFSET 5")
	outcome, err := addDefaultLimit(stmt)
	require.NoError(t, err)
	require.Equal(t, limitDefaulted, outcome)
	require.Equal(t, "SELECT run_id FROM runs LIMIT 1000 OFFSET 5", parser.String(stmt))
}

func TestAddDefaultLimitCapsOverflowingLimit(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs LIMIT 99999999999999999999")
	outcome, err := addDefaultLimit(stmt)
	require.NoError(t, err)
	require.Equal(t, limitCapped, outcome)
	require.Equal(t, "SELECT run_id FROM runs LIMIT 1000", parser.String(stmt))
}

func TestAddDefaultLimitRejectsNonIntegerLiteral(t *testing.T) {
	for _, sql := range []string{
		"SELECT run_id FROM runs LIMIT 1e9",
		"SELECT run_id FROM runs LIMIT 10.5",
	} {
		t.Run(sql, func(t *testing.T) {
			stmt := mustParse(t, sql)
			_, err := addDefaultLimit(stmt)
			require.ErrorContains(t, err, "LIMIT must be a literal, non-negative integer")
		})
	}
}

// TestAddDefaultLimitDigitSeparators checks a `_`-separated integer
// (`1_000`, which DuckDB lexes as 1000) is bounded like its plain spelling,
// and rewritten without separators when capped.
func TestAddDefaultLimitDigitSeparators(t *testing.T) {
	for _, c := range []struct {
		sql, want string
		outcome   limitOutcome
	}{
		{"SELECT run_id FROM runs LIMIT 1_000", "SELECT run_id FROM runs LIMIT 1_000", limitUnchanged},
		{"SELECT run_id FROM runs LIMIT 1_0", "SELECT run_id FROM runs LIMIT 1_0", limitUnchanged},
		{"SELECT run_id FROM runs LIMIT 1_001", "SELECT run_id FROM runs LIMIT 1000", limitCapped},
		{"SELECT run_id FROM runs LIMIT 5_000_000", "SELECT run_id FROM runs LIMIT 1000", limitCapped},
		{"SELECT run_id FROM runs LIMIT 99_999_999_999_999_999_999", "SELECT run_id FROM runs LIMIT 1000", limitCapped},
	} {
		t.Run(c.sql, func(t *testing.T) {
			stmt := mustParse(t, c.sql)
			outcome, err := addDefaultLimit(stmt)
			require.NoError(t, err)
			require.Equal(t, c.outcome, outcome)
			require.Equal(t, c.want, parser.String(stmt))
		})
	}
}

func TestLimitDigits(t *testing.T) {
	for _, c := range []struct {
		in, want string
		ok       bool
	}{
		{"1000", "1000", true},
		{"1_000", "1000", true},
		{"1_0_0", "100", true},
		{"01_0", "010", true},
		{"1__0", "", false},
		{"_1", "", false},
		{"1_", "", false},
		{"1_.5", "", false},
		{"1e1_0", "", false},
		{"", "", false},
	} {
		got, ok := limitDigits(c.in)
		require.Equal(t, c.ok, ok, c.in)
		if c.ok {
			require.Equal(t, c.want, got, c.in)
		}
	}
}

func TestAddDefaultLimitCapsFetch(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs FETCH FIRST 5000000 ROWS ONLY")
	outcome, err := addDefaultLimit(stmt)
	require.NoError(t, err)
	require.Equal(t, limitCapped, outcome)
	require.Equal(t, "SELECT run_id FROM runs LIMIT 1000", parser.String(stmt))
}
