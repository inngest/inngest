package resolvers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInsightsErrorsWhenDuckDBNil(t *testing.T) {
	r := &Resolver{DuckDB: nil}
	qr := r.Query().(*queryResolver)

	_, err := qr.Insights(context.Background(), "SELECT run_id FROM runs")
	require.Error(t, err)
	require.Contains(t, err.Error(), "--duckdb")
}
