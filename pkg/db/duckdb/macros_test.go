package duckdb

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTableMacrosParse checks every table macro views.sql creates parses,
// so nothing that sees through them (federate's column lineage) silently
// misses one.
func TestTableMacrosParse(t *testing.T) {
	macros, err := TableMacros()
	require.NoError(t, err)
	require.Equal(t, strings.Count(viewsSQL, ") AS TABLE"), len(macros))

	runs := macros["insights_runs"]
	require.NotNil(t, runs)
	require.Equal(t, []string{"p_account_id", "p_env_id", "p_runs", "p_metadata"}, runs.Params)
	require.Equal(t, map[string]string{"p_runs": "inngest.runs", "p_metadata": "inngest.run_metadata"}, runs.Defaults)
	require.NotNil(t, runs.Body)
}
