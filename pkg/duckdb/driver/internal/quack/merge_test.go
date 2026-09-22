package quack

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBuildSendDataMergeSQLDedupKeysRejectsUnknownColumn guards
// buildSendDataMergeSQL's validation: a DedupKeys entry that doesn't
// name one of the columns passed to driver.NewQuackMergeAppender must fail clearly
// up front, rather than reaching the server as a "column not found" SQL
// error against a QUALIFY clause the caller never wrote themselves.
func TestBuildSendDataMergeSQLDedupKeysRejectsUnknownColumn(t *testing.T) {
	_, err := buildSendDataMergeSQL("main", "t", MergeConfig{
		On:           "t.k = s.k",
		DedupKeys:    []string{"nope"},
		DedupOrderBy: "seq",
	}, []MergeColumn{{Name: "k", Kind: ColumnVarchar}}, "stream-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "nope")
}

// TestBuildSendDataMergeSQLDedupKeysRequiresDedupOrderBy guards the other
// buildSendDataMergeSQL validation: DedupKeys with no DedupOrderBy has
// no way to choose which same-key row survives, so it must fail clearly up
// front rather than building a QUALIFY with an empty ORDER BY.
func TestBuildSendDataMergeSQLDedupKeysRequiresDedupOrderBy(t *testing.T) {
	_, err := buildSendDataMergeSQL("main", "t", MergeConfig{
		On:        "t.k = s.k",
		DedupKeys: []string{"k"},
	}, []MergeColumn{{Name: "k", Kind: ColumnVarchar}}, "stream-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "DedupOrderBy")
}
