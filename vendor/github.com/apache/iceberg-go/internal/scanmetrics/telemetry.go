// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Package scanmetrics instruments client-side scan planning. Attribute values
// are bounded operation/outcome names; filters, credentials, table names, plan
// tokens, paths, and server error messages are never exported.
package scanmetrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const scope = "github.com/apache/iceberg-go/scan-planning"

// Start traces a logical operation, including its retries. Instruments come
// from the application's global provider and are no-ops without an SDK.
func Start(ctx context.Context, operation string) (context.Context, func(error)) {
	ctx, span := otel.Tracer(scope).Start(ctx, "iceberg.scan.planning."+operation,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("iceberg.scan.planning.operation", operation)))
	start := time.Now()

	return ctx, func(err error) {
		outcome := "success"
		if err != nil {
			outcome = "error"
			// Error strings can include filters, paths, or opaque plan tokens.
			span.SetStatus(codes.Error, "scan planning failed")
		}
		attrs := metric.WithAttributes(
			attribute.String("iceberg.scan.planning.operation", operation),
			attribute.String("iceberg.scan.planning.outcome", outcome))
		meter := otel.Meter(scope)
		if count, e := meter.Int64Counter("iceberg.scan.planning.requests"); e == nil {
			count.Add(ctx, 1, attrs)
		}
		if duration, e := meter.Float64Histogram("iceberg.scan.planning.request.duration", metric.WithUnit("ms")); e == nil {
			duration.Record(ctx, float64(time.Since(start))/float64(time.Millisecond), attrs)
		}
		span.End()
	}
}

// Retry records an additional attempt after a transient HTTP failure.
func Retry(ctx context.Context, operation string) {
	count(ctx, "retries", "operation", operation)
}

// Expired records a recognized expired plan or plan-task response.
func Expired(ctx context.Context, operation string) {
	count(ctx, "expirations", "operation", operation)
}

// Fallback records auto mode choosing local planning before submitting a plan.
func Fallback(ctx context.Context, reason string) {
	count(ctx, "fallbacks", "reason", reason)
}

func count(ctx context.Context, name, key, value string) {
	if counter, err := otel.Meter(scope).Int64Counter("iceberg.scan.planning." + name); err == nil {
		counter.Add(ctx, 1, metric.WithAttributes(attribute.String("iceberg.scan.planning."+key, value)))
	}
}
