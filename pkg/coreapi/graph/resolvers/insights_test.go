package resolvers

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/duckdb/insights"
	"github.com/stretchr/testify/require"
)

func TestInsightsErrorsWhenDuckDBNil(t *testing.T) {
	r := &Resolver{DuckDB: nil}
	qr := r.Query().(*queryResolver)

	_, err := qr.Insights(context.Background(), "SELECT run_id FROM runs")
	require.Error(t, err)
	require.Contains(t, err.Error(), "--duckdb")
}

// TestInsightsRecoversPanics pins that a panic anywhere in the insights
// pipeline becomes an ordinary error for this one query instead of
// propagating out of the resolver.
func TestInsightsRecoversPanics(t *testing.T) {
	r := &Resolver{DuckDB: &sql.DB{}}
	qr := r.Query().(*queryResolver)

	t.Run("transpile", func(t *testing.T) {
		orig := transpileInsights
		t.Cleanup(func() { transpileInsights = orig })
		transpileInsights = func(string, uuid.UUID, uuid.UUID) (*insights.TranspileResult, error) {
			panic("boom")
		}

		var (
			res *models.InsightsQueryResult
			err error
		)
		require.NotPanics(t, func() { res, err = qr.Insights(context.Background(), "SELECT run_id FROM runs") })
		require.Nil(t, res)
		require.ErrorIs(t, err, errInsightsInternal)
	})

	t.Run("execute", func(t *testing.T) {
		orig := executeInsights
		t.Cleanup(func() { executeInsights = orig })
		executeInsights = func(context.Context, *sql.DB, *insights.TranspileResult) (*insights.Result, error) {
			var m map[string]int
			m["x"]++ //nolint:staticcheck // a real runtime panic, not just panic(...)
			return nil, nil
		}

		var (
			res *models.InsightsQueryResult
			err error
		)
		require.NotPanics(t, func() { res, err = qr.Insights(context.Background(), "SELECT run_id FROM runs") })
		require.Nil(t, res)
		require.ErrorIs(t, err, errInsightsInternal)
	})
}
