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

// Package metrics implements Iceberg's Metrics Reporting API for iceberg-go.
//
// A [Reporter] is a pluggable sink that receives a [MetricsReport] — a
// ScanReport after scan planning or a CommitReport after a commit — describing
// what the client did (files and manifests considered, scanned and skipped;
// bytes read; commit attempts and durations). Reporting gives operators a
// standard way to aggregate these otherwise-invisible client-side metrics
// across many clients.
//
// Reporting is strictly opt-in: with no reporter configured the instrumented
// code paths do no work. Reporters must never block or fail the scan/commit
// they observe — see [Reporter] for the contract.
//
// This package provides the contract and the built-in reporters
// ([NopReporter], [LoggingReporter], [InMemoryReporter], [Combine], and
// [CombineWithLogger]). The concrete report types and the scan/commit
// instrumentation are layered on top in later work.
package metrics

import (
	"context"
	"io"
)

// MetricsReport is the marker interface implemented by the concrete report
// types (ScanReport and CommitReport). It is intentionally empty, mirroring the
// open MetricsReport interface in the Java and Python Iceberg implementations,
// so downstream operators can implement it for their own report wrappers.
//
// The set of report types is controlled by this package for now, but that is a
// convention rather than a structural guarantee: the interface is deliberately
// not sealed. That openness is the irreversible choice — sealing it later would
// be the breaking change — so it is worth being precise about what it does and
// does not buy. Custom report types are honored only by the in-process reporters
// ([LoggingReporter], [InMemoryReporter], and the [Combine] fan-out). The REST
// reporter (once it lands) can serialize only the discriminators it knows
// (scan-report and commit-report), so a third-party MetricsReport carries no
// discriminator and is silently dropped by that reporter — a spec-compliant
// catalog would reject it outright. Do not build a custom report expecting it to
// reach a catalog endpoint.
//
// The name mirrors the MetricsReport type in the Java and Python Iceberg
// implementations, easing ports across the ecosystem and disambiguating the
// type from the [Reporter.Report] method that consumes it.
type MetricsReport any

// Reporter is a pluggable sink for metrics reports.
//
// Report is invoked inline at the scan/commit completion point, so
// implementations must return promptly and must never block or fail the
// operation being observed: network-backed reporters should dispatch the
// actual send on a background worker, and any error must be handled internally
// (logged and swallowed), never propagated back to the caller. Report must be
// safe for concurrent use by multiple goroutines.
//
// Reporter embeds [io.Closer] to mirror Java's Closeable MetricsReporter: a
// stateful reporter (e.g. an HTTP-backed one holding a shared client or a
// background dispatch worker) releases those resources in Close. Close is
// called once, by the owner of the reporter, when the reporter is no longer
// needed; Report must not be called after Close returns. Stateless reporters
// return nil. Close lives on the interface deliberately — adding it after this
// surface stabilizes would be a breaking change for external implementers.
type Reporter interface {
	Report(ctx context.Context, report MetricsReport)
	io.Closer
}
