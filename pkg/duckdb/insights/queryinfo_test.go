package insights

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractQueryInfoSingleTable(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs")
	info := extractQueryInfo(stmt)
	require.Equal(t, "runs", info.PrimaryTable)
	require.Equal(t, []string{"runs"}, info.Tables)
}

func TestExtractQueryInfoJoin(t *testing.T) {
	stmt := mustParse(t, "SELECT * FROM runs JOIN events ON runs.run_id = events.id")
	info := extractQueryInfo(stmt)
	require.Equal(t, "runs", info.PrimaryTable)
	require.Equal(t, []string{"runs", "events"}, info.Tables)
}

func TestExtractQueryInfoDedupes(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs UNION SELECT run_id FROM runs")
	info := extractQueryInfo(stmt)
	require.Equal(t, []string{"runs"}, info.Tables)
}

func TestExtractQueryInfoUnion(t *testing.T) {
	stmt := mustParse(t, "SELECT run_id FROM runs UNION SELECT run_id FROM extended_trace_spans")
	info := extractQueryInfo(stmt)
	require.Equal(t, "runs", info.PrimaryTable)
	require.Equal(t, []string{"runs", "extended_trace_spans"}, info.Tables)
}
