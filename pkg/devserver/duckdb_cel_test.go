package devserver

import (
	"context"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/query"
	"github.com/stretchr/testify/require"
)

// The duckdb read path's CEL search hooks default to "unimplemented" stubs;
// devserver's init must swap in insights' real conversion.
func TestDuckDBCELHooksAreWired(t *testing.T) {
	ctx := context.Background()

	filters, err := query.CELEventTableFilters(ctx, []string{`event.data.a == "x"`})
	require.NoError(t, err)
	require.NotEmpty(t, filters)

	_, err = query.CELEventFilters(ctx, []string{`event.data.a == "x"`})
	require.NoError(t, err)

	_, err = query.CELOutputFilters(ctx, []string{`output.a == "x"`})
	require.NoError(t, err)

	sql, _, err := query.RenderWhereSQL(filters)
	require.NoError(t, err)
	require.NotEmpty(t, sql)
}
