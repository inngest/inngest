package apiv2

import (
	"context"
	"database/sql"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/inngest/inngest/pkg/db/duckdb"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// newTestDuckDB duplicates pkg/coreapi/graph/resolvers/insights_integration_test.go's
// unexported helper of the same name/shape (itself a duplicate of
// pkg/cqrs/duckdbquery/testutil_test.go's, see that file's own comment) --
// redefined here because it's a different package.
func newTestDuckDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	dir := t.TempDir()
	db, err := duckdb.Open(t.Context(), duckdb.Options{
		BinaryPath: binPath,
		DBFile:     ":memory:",
		DuckLake: &duckdb.DuckLakeOptions{
			CatalogPath: filepath.Join(dir, "catalog.duckdb"),
			DataPath:    filepath.Join(dir, "data"),
		},
	})
	if err != nil {
		t.Fatalf("opening duckdb: %v", err)
	}
	if err := duckdb.Migrate(t.Context(), db, true); err != nil {
		t.Fatalf("migrating duckdb: %v", err)
	}
	return db, func() { _ = db.Close() }
}

func TestQueryInsights_RequiresDuckDB(t *testing.T) {
	service := NewService(ServiceOptions{})

	resp, err := service.QueryInsights(context.Background(), &apiv2.QueryInsightsRequest{Query: "SELECT run_id FROM runs"})

	require.Nil(t, resp)
	require.Equal(t, codes.Unimplemented, status.Code(err))
	require.ErrorContains(t, err, "Insights requires dual-write")
}

func TestQueryInsights_ShowTablesShortCircuit(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	service := NewService(ServiceOptions{DuckDB: db})

	resp, err := service.QueryInsights(context.Background(), &apiv2.QueryInsightsRequest{Query: "SHOW TABLES"})

	require.NoError(t, err)
	require.Empty(t, resp.Data.Diagnostics)
	require.Len(t, resp.Data.Columns, 2)
	require.Equal(t, "name", resp.Data.Columns[0].Name)
	require.Len(t, resp.Data.Rows, 6)
	require.NotNil(t, resp.Metadata.FetchedAt)
}

func TestQueryInsights_ValidationRejection(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	service := NewService(ServiceOptions{DuckDB: db})

	resp, err := service.QueryInsights(context.Background(), &apiv2.QueryInsightsRequest{Query: "SELECT * FROM nonexistent_table"})

	require.NoError(t, err)
	require.Empty(t, resp.Data.Columns)
	require.Empty(t, resp.Data.Rows)
	require.Len(t, resp.Data.Diagnostics, 1)
	require.Equal(t, apiv2.InsightsDiagnosticSeverity_ERROR, resp.Data.Diagnostics[0].Severity)
	require.Equal(t, "validation-error", resp.Data.Diagnostics[0].Code)
	require.NotEmpty(t, resp.Data.Diagnostics[0].Message)
	require.NotNil(t, resp.Data.Diagnostics[0].Position)
	require.Equal(t, "nonexistent_table", resp.Data.Diagnostics[0].Position.Context)
}

func TestQueryInsights_ParseErrorReturnsError(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	service := NewService(ServiceOptions{DuckDB: db})

	resp, err := service.QueryInsights(context.Background(), &apiv2.QueryInsightsRequest{Query: "SELECT run_id FROM runs; SELECT run_id FROM runs;"})

	require.Nil(t, resp)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestQueryInsights_ExecutesRealQuery(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	service := NewService(ServiceOptions{DuckDB: db})

	resp, err := service.QueryInsights(context.Background(), &apiv2.QueryInsightsRequest{Query: "SELECT run_id FROM runs"})

	require.NoError(t, err)
	// addDefaultLimit's own non-fatal diagnostic (no LIMIT given, capped at
	// 1000 rows) rides along on a successful query too, via tr.Diagnostics --
	// confirms QueryInsights carries those through, not just fatal ones.
	require.Len(t, resp.Data.Diagnostics, 1)
	require.Equal(t, apiv2.InsightsDiagnosticSeverity_INFO, resp.Data.Diagnostics[0].Severity)
	require.Equal(t, "default-limit-applied", resp.Data.Diagnostics[0].Code)
	require.Len(t, resp.Data.Columns, 1)
	require.Equal(t, "run_id", resp.Data.Columns[0].Name)
	require.Equal(t, apiv2.InsightsOutputColumnType_STRING, resp.Data.Columns[0].Type)
	require.Empty(t, resp.Data.Rows)
}

func TestListInsightsTables(t *testing.T) {
	// No DuckDB dependency -- insights.AllTableSchemas() is a static
	// registry, so this works without --duckdb dual-write enabled.
	service := NewService(ServiceOptions{})

	resp, err := service.ListInsightsTables(context.Background(), &apiv2.ListInsightsTablesRequest{})

	require.NoError(t, err)
	require.NotNil(t, resp.Metadata.FetchedAt)
	require.Len(t, resp.Data, 6)

	names := make([]string, len(resp.Data))
	for i, tbl := range resp.Data {
		names[i] = tbl.Name
	}
	require.ElementsMatch(t, []string{"runs", "events", "metadata", "extended_trace_spans", "steps", "step_attempts"}, names)
}

func TestListInsightsTables_RunsTableColumns(t *testing.T) {
	service := NewService(ServiceOptions{})

	resp, err := service.ListInsightsTables(context.Background(), &apiv2.ListInsightsTablesRequest{})
	require.NoError(t, err)

	var runs *apiv2.InsightsTable
	for _, tbl := range resp.Data {
		if tbl.Name == "runs" {
			runs = tbl
		}
	}
	require.NotNil(t, runs)
	require.NotEmpty(t, runs.Description)
	require.NotEmpty(t, runs.Columns)
	require.Equal(t, "run_id", runs.Columns[0].Name)
	require.Equal(t, "STRING", runs.Columns[0].Type)
	require.NotEmpty(t, runs.Columns[0].Description)
}

// QueryInsightsPrompt and ListInsightsEventSchemas have no backing
// implementation anywhere in this codebase (no NL->SQL capability, no
// event-schema-catalog concept in pkg/duckdb/insights) -- they stay 501
// stubs regardless of DuckDB availability.
func TestInsightsPromptAndEventSchemasStillStubbed(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	service := NewService(ServiceOptions{DuckDB: db})

	t.Run("QueryInsightsPrompt", func(t *testing.T) {
		_, err := service.QueryInsightsPrompt(context.Background(), &apiv2.QueryInsightsPromptRequest{Prompt: "top functions by failure rate"})
		require.Equal(t, codes.Unimplemented, status.Code(err))
		require.ErrorContains(t, err, "Insights not implemented in OSS")
	})

	t.Run("ListInsightsEventSchemas", func(t *testing.T) {
		_, err := service.ListInsightsEventSchemas(context.Background(), &apiv2.ListInsightsEventSchemasRequest{})
		require.Equal(t, codes.Unimplemented, status.Code(err))
		require.ErrorContains(t, err, "Insights not implemented in OSS")
	})
}
