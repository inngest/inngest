package grpc

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

var metricReader = sdkmetric.NewManualReader()

func TestMain(m *testing.M) {
	// Instruments are cached globally; install the reader before any test emits.
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(metricReader)))
	os.Exit(m.Run())
}

// counterTotal returns the sum of a counter across all attribute sets.
func counterTotal(t *testing.T, name string) int64 {
	t.Helper()
	var data metricdata.ResourceMetrics
	require.NoError(t, metricReader.Collect(t.Context(), &data))
	var total int64
	for _, scope := range data.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name != name {
				continue
			}
			for _, p := range m.Data.(metricdata.Sum[int64]).DataPoints {
				total += p.Value
			}
		}
	}
	return total
}
