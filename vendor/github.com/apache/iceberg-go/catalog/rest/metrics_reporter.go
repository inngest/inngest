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

package rest

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	iceberg "github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/metrics"
)

const (
	// keyReportMetricsEnabled opts a REST catalog into POSTing scan/commit
	// reports to the catalog's metrics endpoint. It is disabled by default so
	// existing users see no new network traffic unless they turn it on. This is
	// the canonical, cross-implementation spelling used by Iceberg Java and the
	// Iceberg docs; keyReportMetricsEnabledLegacy is accepted as an alias.
	keyReportMetricsEnabled = "rest-metrics-reporting-enabled"
	// keyReportMetricsEnabledLegacy is the historical dotted spelling, accepted
	// as an alias so existing configs keep working.
	keyReportMetricsEnabledLegacy = "rest.metrics-reporting-enabled"
	// keyReportMetricsTimeoutMs bounds a single report's request and response
	// cycle, in milliseconds. Read from the client-supplied properties only. Auth
	// (token refresh) is bounded separately by the OAuth refresh client Timeout.
	keyReportMetricsTimeoutMs = "rest-metrics-reporting-timeout-ms"

	// defaultReportMetricsTimeout bounds a single report's request and response
	// cycle when keyReportMetricsTimeoutMs is unset. Telemetry is safe to drop, so
	// the bound is deliberately short.
	defaultReportMetricsTimeout = 10 * time.Second
	// metricsDispatchWorkers is the fixed number of goroutines draining the
	// dispatch queue, capping the reporter's concurrent connection use.
	metricsDispatchWorkers = 4
	// metricsDispatchQueueSize bounds how many reports may await dispatch.
	// Reports offered while the queue is full are dropped (and counted, then
	// logged in aggregate) rather than queued without limit, so a stalled
	// endpoint cannot make reporting grow without bound.
	metricsDispatchQueueSize = 128
	// metricsStatsInterval is how often the dispatcher emits an aggregated
	// warning summarizing dropped and failed reports. Aggregating keeps a stalled
	// or unavailable endpoint from amplifying into one log line per report while
	// still surfacing back-pressure.
	metricsStatsInterval = 30 * time.Second
)

// reportMetricsEnabled reports whether the client opted into REST metrics
// reporting, accepting both the canonical and the legacy dotted key. It must be
// given the client-supplied properties only (never the server-merged config) so
// a server cannot flip the default and turn on outbound telemetry the client
// never asked for.
//
// The canonical key wins whenever it is present, so a client that has migrated
// to it and explicitly set it to false is honored even if a stale legacy dotted
// key lingers in the config. The legacy key is consulted only when the canonical
// key is absent.
func reportMetricsEnabled(props iceberg.Properties) bool {
	if v, ok := props[keyReportMetricsEnabled]; ok {
		enabled, err := strconv.ParseBool(v)

		return err == nil && enabled
	}

	return props.GetBool(keyReportMetricsEnabledLegacy, false)
}

// reportMetricsTimeout resolves the per-report deadline from the client-supplied
// properties, falling back to defaultReportMetricsTimeout for a missing or
// non-positive value.
func reportMetricsTimeout(props iceberg.Properties) time.Duration {
	ms := props.GetInt(keyReportMetricsTimeoutMs, int(defaultReportMetricsTimeout/time.Millisecond))
	if ms <= 0 {
		return defaultReportMetricsTimeout
	}

	return time.Duration(ms) * time.Millisecond
}

// metricsJob is a single report awaiting dispatch to a table's metrics endpoint.
type metricsJob struct {
	// ctx carries the observed scan/commit's context values (trace spans,
	// request-scoped attributes) with its cancellation detached — the scan/commit
	// is already done, so its cancellation must not abort the report, but its
	// values should still propagate to the outbound request.
	ctx     context.Context
	baseURI *url.URL
	cl      *http.Client
	path    []string
	req     metrics.ReportMetricsRequest
}

// metricsDispatcher POSTs metrics reports to REST metrics endpoints on a fixed
// pool of workers draining a bounded queue. It is owned by the catalog and
// shared across that catalog's table reporters, so concurrent report volume
// stays bounded no matter how many tables are loaded or how often they are
// scanned. A stalled endpoint sheds load — reports are dropped and counted —
// rather than accumulating goroutines and connections. Close cancels in-flight
// reports and drains the workers.
type metricsDispatcher struct {
	jobs    chan metricsJob
	timeout time.Duration
	// ctx/cancel are the dispatcher's own lifecycle context, deliberately stored
	// on the struct (the documented exception to "don't store a context in a
	// struct"): Close cancels it to abort in-flight reports and stop the workers.
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	logger    *slog.Logger // nil means resolve slog.Default at call time
	dropped   atomic.Uint64
	failed    atomic.Uint64
	closeOnce sync.Once
	closeDone chan struct{}
}

func newMetricsDispatcher(workers, queueSize int, timeout time.Duration, logger *slog.Logger) *metricsDispatcher {
	ctx, cancel := context.WithCancel(context.Background())
	d := &metricsDispatcher{
		jobs:      make(chan metricsJob, queueSize),
		timeout:   timeout,
		ctx:       ctx,
		cancel:    cancel,
		logger:    logger,
		closeDone: make(chan struct{}),
	}

	d.wg.Add(workers + 1)
	for range workers {
		go d.worker()
	}
	go d.reportStats()

	return d
}

func (d *metricsDispatcher) log() *slog.Logger {
	if d.logger != nil {
		return d.logger
	}

	return slog.Default()
}

func (d *metricsDispatcher) worker() {
	defer d.wg.Done()
	for {
		select {
		case <-d.ctx.Done():
			return
		case job := <-d.jobs:
			d.send(job)
		}
	}
}

func (d *metricsDispatcher) send(job metricsJob) {
	// If the dispatcher is already shutting down, drop the report before doing
	// any work: the derived context would be cancelled immediately anyway.
	if d.ctx.Err() != nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			d.log().Warn("iceberg: panic while reporting metrics to REST catalog", "recovered", r)
		}
	}()

	// Bound the report by the per-report deadline, starting from the job's
	// context so the caller's values (trace spans, request-scoped attributes)
	// still reach the transport, and tie the derived context to dispatcher
	// shutdown so Close cancels in-flight requests. The deadline covers the
	// request and response cycle; a token refresh triggered by auth is bounded
	// separately by the OAuth refresh client's Timeout.
	ctx, cancel := context.WithTimeout(job.ctx, d.timeout)
	defer cancel()
	stop := context.AfterFunc(d.ctx, cancel)
	defer stop()

	if _, err := doPost[metrics.ReportMetricsRequest, struct{}](
		ctx, job.baseURI, job.path, job.req, job.cl, nil, allowNoContent()); err != nil {
		// A report interrupted by Close (dispatcher context cancelled) is expected
		// shutdown behavior, not a failure worth counting. A per-report timeout
		// leaves the dispatcher context live, so genuine failures still surface.
		if d.ctx.Err() != nil {
			return
		}
		d.failed.Add(1)
		// The transport error embeds the request URL, which can carry a sensitive
		// host, prefix, namespace or table name, so keep the detail at Debug and
		// let reportStats surface the aggregate count at Warn.
		d.log().Debug("iceberg: failed to report metrics to REST catalog", "error", err)
	}
}

// submit offers a job to the queue without blocking. It drops the report (and
// counts the drop, aggregated and logged by reportStats so back-pressure is
// visible without amplifying into one log line per drop) when the queue is
// full, and ignores reports once the dispatcher is closed. Every path here is
// non-blocking so Report never blocks the observed scan/commit.
func (d *metricsDispatcher) submit(job metricsJob) {
	select {
	case <-d.ctx.Done():
		return
	default:
	}

	select {
	case d.jobs <- job:
	case <-d.ctx.Done():
	default:
		d.dropped.Add(1)
	}
}

// reportStats periodically emits an aggregated warning summarizing reports
// dropped (queue full) and failed (endpoint errors) since the last tick, so a
// stalled or unavailable endpoint surfaces as bounded, rate-limited log volume
// rather than one line per report. A final summary is emitted on shutdown.
func (d *metricsDispatcher) reportStats() {
	defer d.wg.Done()

	ticker := time.NewTicker(metricsStatsInterval)
	defer ticker.Stop()

	var lastDropped, lastFailed uint64
	flush := func() {
		dropped, failed := d.dropped.Load(), d.failed.Load()
		if dropped == lastDropped && failed == lastFailed {
			return
		}
		d.log().Warn("iceberg: REST metrics reports dropped or failed",
			"dropped", dropped-lastDropped, "dropped_total", dropped,
			"failed", failed-lastFailed, "failed_total", failed)
		lastDropped, lastFailed = dropped, failed
	}

	for {
		select {
		case <-d.ctx.Done():
			flush()

			return
		case <-ticker.C:
			flush()
		}
	}
}

// close cancels in-flight reports and waits for the workers to return, bounded
// by the report timeout so shutdown cannot hang on a stalled endpoint. It is
// one-shot: repeated calls block until the first completes, then return, so no
// extra waiter goroutine is spawned per call.
func (d *metricsDispatcher) close() {
	d.closeOnce.Do(func() {
		d.cancel()

		done := make(chan struct{})
		go func() {
			d.wg.Wait()
			close(done)
		}()

		// Explicit timer with a deferred Stop rather than time.After, which would
		// leak a live timer (up to the full timeout) on the common path where the
		// workers finish first — it adds up in a process that cycles through many
		// catalogs.
		t := time.NewTimer(d.timeout)
		defer t.Stop()
		select {
		case <-done:
		case <-t.C:
			// Shutdown budget expired before the workers returned (a report wedged in
			// auth, say). d.ctx is already cancelled, so each worker self-exits once
			// its attempt unblocks — nothing leaks — but Close can return here before
			// the workers have fully stopped; do not tear down a shared transport on
			// the assumption that they have.
		}

		close(d.closeDone)
	})

	<-d.closeDone
}

// restMetricsReporter POSTs metrics reports for a single table to its REST
// metrics endpoint (POST .../tables/{table}/metrics) via the catalog-owned
// dispatcher. It satisfies metrics.Reporter.
type restMetricsReporter struct {
	baseURI    *url.URL
	cl         *http.Client
	path       []string // table metrics path, relative to baseURI
	dispatcher *metricsDispatcher
}

var _ metrics.Reporter = (*restMetricsReporter)(nil)

// Report wraps the report in a ReportMetricsRequest and hands it to the
// catalog's dispatcher, returning immediately. Per the Reporter contract it
// never blocks or fails the observed scan/commit: dispatch is asynchronous and
// bounded, and any error is counted and swallowed by the dispatcher. The
// caller's context is detached from cancellation (the scan/commit is already
// done) but its values are preserved so trace spans and request-scoped
// attributes still propagate to the outbound report.
func (rep *restMetricsReporter) Report(ctx context.Context, report metrics.MetricsReport) {
	if report == nil {
		return
	}

	if ctx == nil {
		ctx = context.Background()
	}

	rep.dispatcher.submit(metricsJob{
		ctx:     context.WithoutCancel(ctx),
		baseURI: rep.baseURI,
		cl:      rep.cl,
		path:    rep.path,
		req:     metrics.NewReportMetricsRequest(report),
	})
}

// Close satisfies [metrics.Reporter]. A per-table reporter holds no resources of
// its own: the goroutines that dispatch its reports live in the catalog-owned
// dispatcher (closed by Catalog.Close), and it only borrows the catalog's shared
// HTTP client. So there is nothing to release here.
func (rep *restMetricsReporter) Close() error { return nil }
