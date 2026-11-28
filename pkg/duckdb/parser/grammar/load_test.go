package grammar

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	g, kl, err := Load()
	require.NoError(t, err)

	require.Greater(t, len(g.Rules), 200)
	require.Contains(t, g.Rules, "SelectStatement")
	require.Contains(t, g.Rules, "ColId")
	require.Contains(t, g.Rules, "Expression")

	// These counts are the true number of keyword entries, not each file's
	// wc -l (vendor_test.go's line-count drift check uses wc -l on
	// purpose — a different, valid metric): reserved_keyword.list and
	// unreserved_keyword.list happen to end with a trailing newline, but
	// column_name/func_name/type_name_keyword.list don't, so their last
	// entry has no trailing "\n" and wc -l undercounts them by one.
	require.Len(t, kl.Reserved, 75)
	require.Len(t, kl.Unreserved, 339)
	require.Len(t, kl.ColumnName, 55)
	require.Len(t, kl.FuncName, 30)
	require.Len(t, kl.TypeName, 32)
	require.Contains(t, kl.Reserved, "SELECT")
}
