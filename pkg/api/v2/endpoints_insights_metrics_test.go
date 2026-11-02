package apiv2

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/duckdb/driver"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// TestQueryInsights_NonFiniteFloatIsCountedAsOK: a NaN (over quack, since
// jsonlines doesn't surface a real float NaN) used to fail row conversion;
// it now encodes as "NaN", so the query succeeds and its metrics say so.
func TestQueryInsights_NonFiniteFloatIsCountedAsOK(t *testing.T) {
	binPath, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping")
	}
	quackAddr := driver.EphemeralQuackAddr
	db, err := driver.Open(t.Context(), driver.Options{BinaryPath: binPath, DBFile: ":memory:", QuackAddr: &quackAddr})
	if err != nil {
		t.Skipf("opening quack duckdb: %v", err)
	}
	defer func() { _ = db.Close() }()
	require.NoError(t, duckdb.Migrate(t.Context(), db, false))

	reader := sdkmetric.NewManualReader()
	orig := otel.GetMeterProvider()
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	t.Cleanup(func() { otel.SetMeterProvider(orig) })

	service := NewService(ServiceOptions{DuckDB: db})
	resp, err := service.QueryInsights(context.Background(), &apiv2.QueryInsightsRequest{Query: "SELECT 'NaN'::DOUBLE AS x FROM runs UNION ALL SELECT 'NaN'::DOUBLE AS x"})
	require.NoError(t, err)
	require.Len(t, resp.Data.Rows, 1)
	require.Equal(t, "NaN", resp.Data.Rows[0].Values[0].GetStringValue())

	var data metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &data))
	statuses := map[string]int64{}
	for _, sm := range data.ScopeMetrics {
		for _, m := range sm.Metrics {
			if !strings.HasSuffix(m.Name, "insights_query_total") {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, "%s is %T", m.Name, m.Data)
			for _, dp := range sum.DataPoints {
				status, _ := dp.Attributes.Value("status")
				statuses[status.AsString()] += dp.Value
			}
		}
	}
	require.Equal(t, map[string]int64{"ok": 1}, statuses)
}
