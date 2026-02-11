package api

import (
	"net/http"

	"github.com/felixge/httpsnoop"
	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

const (
	metricsPkgName = "api.inngest"
)

type MetricsMiddleware interface {
	Middleware(next http.Handler) http.Handler
}

type metricsMiddleware struct{}

func NewMetricsMiddleware() metricsMiddleware {
	return metricsMiddleware{}
}

func (m metricsMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		m := httpsnoop.CaptureMetrics(next, w, r)

		tags := map[string]any{
			"method": r.Method,
			"route":  chi.RouteContext(ctx).RoutePattern(),
			"status": m.Code,
		}

		metrics.IncrHTTPAPIRequestsCounter(ctx, metrics.CounterOpt{
			PkgName: metricsPkgName,
			Tags:    tags,
		})

		metrics.HistogramHTTPAPIDuration(ctx, m.Duration.Milliseconds(), metrics.HistogramOpt{
			PkgName: metricsPkgName,
			Tags:    tags,
		})

		metrics.HistogramHTTPAPIBytesWritten(ctx, m.Written, metrics.HistogramOpt{
			PkgName: metricsPkgName,
			Tags:    tags,
		})
	})
}
