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

package metrics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"runtime/debug"
	"sync"
)

// isNilReport reports whether report carries no value. A plain report == nil
// catches only an untyped nil interface; once the concrete report types land, a
// typed nil pointer (e.g. (*ScanReport)(nil)) is a non-nil interface value that
// would slip past that check, so the reporters below would log or store a nil
// report — and nil-deref if the concrete type grows an [slog.LogValuer] on a nil
// receiver. This also treats such typed nils as "no report" so the guards honor
// the documented "nil reports are ignored" behavior.
func isNilReport(report MetricsReport) bool {
	if report == nil {
		return true
	}
	v := reflect.ValueOf(report)

	return v.Kind() == reflect.Pointer && v.IsNil()
}

// NopReporter is a [Reporter] that discards every report. It is the default
// when no reporter is configured, so that instrumentation is free unless a user
// opts in. The zero value is ready to use.
type NopReporter struct{}

var _ Reporter = NopReporter{}

// Report implements [Reporter] and does nothing.
func (NopReporter) Report(context.Context, MetricsReport) {}

// Close implements [Reporter]. NopReporter holds no resources, so it is a no-op.
func (NopReporter) Close() error { return nil }

// nopAware is implemented by reporters that can report whether they discard
// every report, letting callers skip assembling a report that would be thrown
// away. It is unexported: it is a hint for [IsNop], not part of the [Reporter]
// contract.
type nopAware interface {
	isNop() bool
}

func (NopReporter) isNop() bool { return true }

// IsNop reports whether r is known to discard every report — a bare
// [NopReporter] or a [Combine] of only nop reporters. A false result does not
// guarantee the report is consumed; it only means r is not a recognized no-op.
func IsNop(r Reporter) bool {
	n, ok := r.(nopAware)

	return ok && n.isNop()
}

// LoggingReporter is a [Reporter] that logs each report via an [slog.Logger]. It
// is a convenient default for development and debugging.
type LoggingReporter struct {
	logger *slog.Logger // nil means resolve slog.Default at call time
}

var _ Reporter = (*LoggingReporter)(nil)

// NewLoggingReporter returns a [LoggingReporter] that logs to logger. If logger
// is nil, [slog.Default] is resolved at each Report call, so a later
// [slog.SetDefault] is honored rather than snapshotted at construction.
func NewLoggingReporter(logger *slog.Logger) *LoggingReporter {
	return &LoggingReporter{logger: logger}
}

// Report logs report at info level.
func (r *LoggingReporter) Report(ctx context.Context, report MetricsReport) {
	if isNilReport(report) {
		return
	}
	logger := r.logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.InfoContext(ctx, "iceberg metrics report", "report", report)
}

// Close implements [Reporter]. A LoggingReporter does not own the logger it
// writes to, so there is nothing to release; it is a no-op.
func (r *LoggingReporter) Close() error { return nil }

// InMemoryReporter is a [Reporter] that retains every report it receives. It is
// primarily intended for tests and inspection. It is safe for concurrent use.
type InMemoryReporter struct {
	mu      sync.Mutex
	reports []MetricsReport
}

var _ Reporter = (*InMemoryReporter)(nil)

// Report appends report to the retained set.
func (r *InMemoryReporter) Report(_ context.Context, report MetricsReport) {
	if isNilReport(report) {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reports = append(r.reports, report)
}

// Reports returns a copy of the reports received so far, in arrival order.
func (r *InMemoryReporter) Reports() []MetricsReport {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]MetricsReport(nil), r.reports...)
}

// Reset discards all retained reports.
func (r *InMemoryReporter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reports = nil
}

// Close implements [Reporter]. InMemoryReporter retains reports in memory and
// holds no external resources, so Close is a no-op; retained reports remain
// readable via Reports.
func (r *InMemoryReporter) Close() error { return nil }

// Combine returns a [Reporter] that forwards each report to all of the given
// reporters in order. nil reporters are skipped. A panic in one reporter must
// not prevent the others from receiving the report, so each call is isolated;
// in keeping with the [Reporter] contract a misbehaving reporter never affects
// the observed operation. A recovered panic is logged (with the reporter type)
// at warn level via [slog.Default] so a broken reporter is not silently
// swallowed; use [CombineWithLogger] to direct that log elsewhere.
//
// As a convenience, Combine with no non-nil reporters returns [NopReporter].
// Otherwise every reporter — even a lone one — is wrapped so it receives the
// per-reporter panic isolation the contract advertises. Wrapping a single
// reporter also closes a typed-nil hole: a concrete nil pointer (e.g.
// (*LoggingReporter)(nil)) is a non-nil interface, so it passes the nil filter;
// returning it unwrapped would let its eventual Report nil-deref escape with no
// recover to catch it.
func Combine(reporters ...Reporter) Reporter {
	return CombineWithLogger(nil, reporters...)
}

// CombineWithLogger is [Combine] with an explicit logger for recovered reporter
// panics. A nil logger resolves [slog.Default] at each Report call, matching
// Combine, so a later [slog.SetDefault] is honored rather than snapshotted at
// construction. Like Combine, with no non-nil reporters it returns
// [NopReporter] and the logger is unused.
func CombineWithLogger(logger *slog.Logger, reporters ...Reporter) Reporter {
	nonNil := make([]Reporter, 0, len(reporters))
	for _, r := range reporters {
		if r != nil {
			nonNil = append(nonNil, r)
		}
	}

	if len(nonNil) == 0 {
		return NopReporter{}
	}

	return &compositeReporter{reporters: nonNil, logger: logger}
}

// compositeReporter fans a report out to several reporters, isolating each from
// the others' panics.
type compositeReporter struct {
	reporters []Reporter
	logger    *slog.Logger // nil means resolve slog.Default at call time
}

var _ Reporter = (*compositeReporter)(nil)

// isNop reports true only when every wrapped reporter is itself a nop, so
// Combine(NopReporter{}) is recognized as a no-op by [IsNop].
func (c *compositeReporter) isNop() bool {
	for _, r := range c.reporters {
		if n, ok := r.(nopAware); !ok || !n.isNop() {
			return false
		}
	}

	return true
}

func (c *compositeReporter) Report(ctx context.Context, report MetricsReport) {
	for _, r := range c.reporters {
		func() {
			defer func() {
				if v := recover(); v != nil {
					// Swallow per the Reporter contract, but surface the
					// failure (with the offending reporter type and the stack
					// at the panic) so a broken reporter is traceable rather
					// than showing up only as mysteriously-missing metrics.
					logger := c.logger
					if logger == nil {
						logger = slog.Default()
					}
					logger.WarnContext(ctx, "iceberg metrics reporter panicked; recovered",
						"reporter", fmt.Sprintf("%T", r), "panic", v,
						slog.String("stack", string(debug.Stack())))
				}
			}()
			r.Report(ctx, report)
		}()
	}
}

// Close closes every wrapped reporter, isolating each from the others so one
// failing or panicking Close still lets the rest run — mirroring the fan-out
// panic isolation in Report. It returns the joined non-nil errors (a recovered
// panic is converted to an error), or nil if all children closed cleanly.
func (c *compositeReporter) Close() error {
	var errs []error
	for _, r := range c.reporters {
		errs = append(errs, closeIsolated(r))
	}

	return errors.Join(errs...)
}

// closeIsolated calls r.Close, converting a panic into an error so a single
// misbehaving reporter cannot abort closing its peers.
func closeIsolated(r Reporter) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("metrics reporter %T panicked on Close: %v", r, v)
		}
	}()

	return r.Close()
}
