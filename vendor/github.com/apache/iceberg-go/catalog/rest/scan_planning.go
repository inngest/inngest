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

// This file contains the REST server-side scan planning client surface for
// apache/iceberg-go#1178. It includes the low-level endpoint methods and the
// end-to-end planner implementation used by table.Scan.

package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math/rand/v2"
	"net/http"
	"net/url"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/internal/scanmetrics"
	iceio "github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/table"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// Compile-time proof that the REST catalog satisfies the table planner seam.
var _ table.ScanPlanner = (*Catalog)(nil)

// Compile-time proof that the REST catalog exposes the optional full-capability
// extension used by table.Scan's auto planning mode.
var _ table.FullRemoteScanPlanner = (*Catalog)(nil)

// ErrPlanExpired is returned when polling a plan that the server no longer
// knows about: a fetchPlanningResult 404 whose error.type is exactly
// NoSuchPlanIdException. It is distinct from a table/namespace-gone 404
// (catalog.ErrNoSuchTable / catalog.ErrNoSuchNamespace) so the polling layer can
// tell retry-with-a-new-plan from abort. A bare or unrecognized 404 stays an
// ambiguous ErrRESTError rather than being guessed as an expiry.
var ErrPlanExpired = fmt.Errorf("%w: scan plan expired", ErrRESTError)

// ErrNoSuchPlanTask is returned when fetchScanTasks is called with a plan-task
// handle the server no longer knows about: a 404 whose error.type is
// NoSuchPlanTaskException. It is distinct from a table/namespace-gone 404 so a
// caller fanning out over plan-task handles can tell an expired handle from the
// table having vanished.
var ErrNoSuchPlanTask = fmt.Errorf("%w: scan plan task not found", ErrRESTError)

// ErrPlanFailed is returned by PlanTableScan and FetchPlanningResult when the
// server reports a failed plan. The returned error is a *PlanFailedError, so the
// structured PlanningError detail is reachable via errors.As.
var ErrPlanFailed = fmt.Errorf("%w: scan plan failed", ErrRESTError)

// ErrPlanCancelled is returned by FetchPlanningResult when polling a plan that
// was cancelled (by this client or another). Like a failed plan it is terminal,
// so it surfaces as an error rather than a (resp, nil) the if-err idiom skips.
var ErrPlanCancelled = fmt.Errorf("%w: scan plan cancelled", ErrRESTError)

// ErrPlanPollExhausted is returned by WaitForPlan when polling reaches
// WaitForPlanOptions.MaxRetries without the plan completing while the context is
// still live. It bounds a plan that stays submitted forever when the caller
// passes no context deadline, and is the Go analogue of Java's
// RemotePlanTimeoutException.
var ErrPlanPollExhausted = fmt.Errorf("%w: scan plan polling exhausted retries", ErrRESTError)

// REST error.type values the scan-planning 404 responses carry. The spec models
// a 404 on these endpoints as one of several distinct not-found conditions;
// each maps to its own sentinel so the polling/fanout layer can tell a
// retry-able expiry from a table/namespace-gone abort. Only these recognized
// types get a specific sentinel; a bare or unrecognized 404 provides no basis to
// choose one over another and stays an ambiguous ErrRESTError (the status-code
// fallback). The strings are the ErrorModel `type` values from the REST OpenAPI
// spec (the exception class simple names Java's RESTCatalog raises).
const (
	errTypeNoSuchPlanID    = "NoSuchPlanIdException"
	errTypeNoSuchPlanTask  = "NoSuchPlanTaskException"
	errTypeNoSuchTable     = "NoSuchTableException"
	errTypeNoSuchNamespace = "NoSuchNamespaceException"
)

// planTableScanErrorTypes splits a POST .../plan 404 by error.type: the plan
// endpoint takes an identifier but no plan-id, so a recognized 404 is a missing
// table or namespace. An unrecognized 404 stays ErrRESTError.
var planTableScanErrorTypes = map[string]error{
	errTypeNoSuchTable:     catalog.ErrNoSuchTable,
	errTypeNoSuchNamespace: catalog.ErrNoSuchNamespace,
}

// fetchPlanningResultErrorTypes splits a GET .../plan/{plan-id} 404: a forgotten
// plan-id is retry-with-a-new-plan (ErrPlanExpired), while a gone table or
// namespace is an abort. An unrecognized 404 stays ErrRESTError.
var fetchPlanningResultErrorTypes = map[string]error{
	errTypeNoSuchPlanID:    ErrPlanExpired,
	errTypeNoSuchTable:     catalog.ErrNoSuchTable,
	errTypeNoSuchNamespace: catalog.ErrNoSuchNamespace,
}

// fetchScanTasksErrorTypes splits a POST .../tasks 404: an expired plan-task
// handle (ErrNoSuchPlanTask) is distinct from a gone table or namespace. An
// unrecognized 404 stays ErrRESTError.
var fetchScanTasksErrorTypes = map[string]error{
	errTypeNoSuchPlanTask:  ErrNoSuchPlanTask,
	errTypeNoSuchTable:     catalog.ErrNoSuchTable,
	errTypeNoSuchNamespace: catalog.ErrNoSuchNamespace,
}

// PlanFailedError carries the server's structured PlanningError detail for a
// failed plan. It satisfies errors.Is(err, ErrPlanFailed) so callers can branch
// on the failure with the if-err idiom while still reaching the detail via
// errors.As.
type PlanFailedError struct {
	Detail *PlanningError
}

func (e *PlanFailedError) Error() string {
	if e.Detail != nil && e.Detail.Message != "" {
		return ErrPlanFailed.Error() + ": " + e.Detail.Message
	}

	return ErrPlanFailed.Error()
}

func (e *PlanFailedError) Unwrap() error { return ErrPlanFailed }

// headerIdempotencyKey is the per-request idempotency key on the plan and tasks
// POSTs. The access-delegation header name lives in rest.go because it is also
// a session default.
const headerIdempotencyKey = "Idempotency-Key"

// --- Capability gating -------------------------------------------------------
//
// Capability is split into two predicates. SupportsPlanTableScan is the narrow
// "server can plan" check (plan endpoint only). SupportsFullRemoteScanPlanning
// reports whether every continuation endpoint is also advertised. A plan-only
// server can still complete a remote scan synchronously with inline file tasks,
// so table.Scan routes on the narrow predicate and PlanFiles checks continuation
// endpoints only when the response requires polling or task expansion. The
// cancel endpoint is best-effort cleanup rather than an execution dependency.

// SupportsPlanTableScan reports whether the server advertised the plan
// submission endpoint.
func (r *Catalog) SupportsPlanTableScan() bool {
	return r.endpoints.contains(endpointPlanTableScan)
}

// SupportsFullRemoteScanPlanning reports whether the server advertised the
// execution endpoints (plan, fetch-result, fetch-tasks), i.e. it can drive the
// async/fanout path, not just sync inline planning. Cancellation is optional
// cleanup and does not prevent a plan from producing tasks.
func (r *Catalog) SupportsFullRemoteScanPlanning() bool {
	return r.SupportsPlanTableScan() &&
		r.endpoints.contains(endpointFetchPlanResult) &&
		r.endpoints.contains(endpointFetchScanTasks)
}

// --- table.ScanPlanner implementation ---------------------------------------

// SupportsRemoteScanPlanning reports whether this catalog can submit a remote
// plan. Any continuation capability is validated against the response returned
// by the server; SupportsFullRemoteScanPlanning reports whether auto mode can
// safely use the complete continuation flow without a local fallback.
func (r *Catalog) SupportsRemoteScanPlanning() bool {
	return r.SupportsPlanTableScan()
}

// PlanFiles plans a scan server-side and returns tasks (and, optionally, a
// plan-scoped FileIO) for the table to read. It submits the plan, polls a
// submitted plan to completion, and expands any plan-task handles into their
// tasks before returning.
//
// Each response is decoded as its own ScanTasks envelope before the results are
// combined. Delete-file references are scoped to the envelope that returned
// them, so flattening all responses first would attach deletes to the wrong
// data files when the server uses plan-task fanout.
func (r *Catalog) PlanFiles(ctx context.Context, req table.ScanPlanningRequest) (table.ScanPlanningResult, error) {
	wire, err := planTableScanRequestFrom(req)
	if err != nil {
		return table.ScanPlanningResult{}, err
	}

	resp, err := r.PlanTableScan(ctx, req.Identifier, wire)
	if err != nil {
		return table.ScanPlanningResult{}, err
	}

	planID := ""
	if resp.PlanID != nil {
		planID = *resp.PlanID
	}
	cleanup := func() {
		if planID != "" {
			r.abandonPlan(ctx, req.Identifier, planID, DefaultWaitForPlanOptions.CancelGracePeriod)
		}
	}

	// Resolve to a completed plan: use an inline result as-is, or poll a
	// submitted one to completion. PlanTableScan maps failed to an error and
	// rejects other statuses, so only completed/submitted reach here.
	var completed CompletedPlanningResult
	switch resp.Status {
	case PlanStatusCompleted:
		completed = CompletedPlanningResult{
			ScanTasks:          resp.ScanTasks,
			StorageCredentials: resp.StorageCredentials,
		}
	case PlanStatusSubmitted:
		completed, err = r.WaitForPlan(ctx, req.Identifier, *resp.PlanID, WaitForPlanOptions{})
		if err != nil {
			// WaitForPlan already abandons plans when its retry budget or the
			// caller's context is exhausted. Other terminal client/transport
			// errors can leave a submitted plan active, so release it here.
			if !errors.Is(err, context.Canceled) &&
				!errors.Is(err, context.DeadlineExceeded) &&
				!errors.Is(err, ErrPlanPollExhausted) &&
				!errors.Is(err, ErrPlanFailed) &&
				!errors.Is(err, ErrPlanCancelled) &&
				!errors.Is(err, ErrPlanExpired) &&
				!errors.Is(err, catalog.ErrNoSuchTable) &&
				!errors.Is(err, catalog.ErrNoSuchNamespace) {
				cleanup()
			}

			return table.ScanPlanningResult{}, err
		}
	default:
		return table.ScanPlanningResult{}, fmt.Errorf(
			"%w: unexpected plan status %q from planTableScan", ErrRESTError, resp.Status)
	}

	envelopes, err := r.collectScanTasksWithConcurrency(
		ctx, req.Identifier, completed.ScanTasks, req.MaxConcurrency)
	if err != nil {
		cleanup()

		return table.ScanPlanningResult{}, err
	}

	tasks, err := remoteScanTasks(envelopes, req)
	if err != nil {
		cleanup()

		return table.ScanPlanningResult{}, err
	}

	result := table.ScanPlanningResult{
		Tasks: tasks,
		IO:    planIOFromCredentials(completed.StorageCredentials, req.MetadataLocation, r.planIOBaseProps(req)),
	}
	if result.IO == nil {
		// Without plan-scoped credentials, all task materialization is complete
		// and the plan can be released before the caller reads the files.
		cleanup()
	} else {
		// Keep the plan alive while the lazy plan-scoped IO may still be used.
		// table.Scan closes this IO after its active readers release their lease.
		result.IO = &planIOWithCleanup{io: result.IO, cleanup: cleanup}
	}

	return result, nil
}

// planIOBaseProps rebuilds the props the table's own FileIO was built with, so
// a plan-scoped IO keeps settings like a custom S3 endpoint or region. The
// table-scoped properties take precedence over catalog and metadata defaults;
// this mirrors the merge order used by LoadTable.
func (r *Catalog) planIOBaseProps(req table.ScanPlanningRequest) iceberg.Properties {
	props := make(iceberg.Properties, len(r.props))
	maps.Copy(props, r.props)
	if req.Metadata != nil {
		maps.Copy(props, req.Metadata.Properties())
	}
	maps.Copy(props, req.FileIOProperties)

	return props
}

// collectScanTasksWithConcurrency expands plan-task handles into their task
// envelopes, walking the fanout: a response can itself return more plan-tasks. Each
// frontier is fetched concurrently, but its responses are appended in handle
// order so completion timing cannot change the result order. A handle is
// fetched at most once; a server that re-issues one would otherwise loop
// forever. The envelope boundaries are retained because delete-file references
// are local to each response.
//
// maxConcurrency bounds the number of concurrent fetches. A non-positive limit
// uses runtime.GOMAXPROCS. On a fetch error, at most the configured number of
// requests can already be in flight when cancellation reaches their contexts.
func (r *Catalog) collectScanTasksWithConcurrency(
	ctx context.Context,
	ident table.Identifier,
	tasks ScanTasks,
	maxConcurrency int,
) ([]ScanTasks, error) {
	if maxConcurrency <= 0 {
		maxConcurrency = runtime.GOMAXPROCS(0)
	}

	envelopes := []ScanTasks{tasks}

	frontier := append([]string(nil), tasks.PlanTasks...)
	seen := make(map[string]bool, len(frontier))
	for len(frontier) > 0 {
		handles := make([]string, 0, len(frontier))
		for _, handle := range frontier {
			if seen[handle] {
				continue
			}
			seen[handle] = true
			handles = append(handles, handle)
		}
		if len(handles) == 0 {
			break
		}

		responses, err := r.fetchScanTaskFrontier(ctx, ident, handles, maxConcurrency)
		if err != nil {
			return nil, err
		}

		var nextFrontier []string
		for _, response := range responses {
			envelopes = append(envelopes, response.ScanTasks)
			nextFrontier = append(nextFrontier, response.PlanTasks...)
		}
		frontier = nextFrontier
	}

	return envelopes, nil
}

// fetchScanTaskFrontier fetches one breadth-first frontier concurrently while
// placing responses back into handle order. When a handle fails, unfinished
// later handles are canceled, but earlier handles are allowed to finish so the
// first error in the old serial handle order remains deterministic.
func (r *Catalog) fetchScanTaskFrontier(
	ctx context.Context,
	ident table.Identifier,
	handles []string,
	maxConcurrency int,
) ([]FetchScanTasksResponse, error) {
	responses := make([]FetchScanTasksResponse, len(handles))
	errs := make([]error, len(handles))
	if len(handles) == 0 {
		return responses, nil
	}
	if maxConcurrency <= 0 {
		maxConcurrency = runtime.GOMAXPROCS(0)
	}
	maxConcurrency = min(maxConcurrency, len(handles))

	requestCtxs := make([]context.Context, len(handles))
	requestCancels := make([]context.CancelFunc, len(handles))
	for i := range handles {
		requestCtxs[i], requestCancels[i] = context.WithCancel(ctx)
	}
	defer func() {
		for _, cancel := range requestCancels {
			cancel()
		}
	}()

	var (
		mu               sync.Mutex
		completed        = make([]bool, len(handles))
		siblingCanceled  = make([]bool, len(handles))
		lowestFailureIdx = -1
	)

	var group errgroup.Group
	group.SetLimit(maxConcurrency)

	for i, handle := range handles {
		group.Go(func() error {
			response, fetchErr := r.FetchScanTasks(requestCtxs[i], ident, FetchScanTasksRequest{PlanTask: handle})

			mu.Lock()
			defer mu.Unlock()
			completed[i] = true
			if fetchErr == nil {
				responses[i] = response

				return nil
			}

			errs[i] = fmt.Errorf("fetching scan tasks for handle %q: %w", handle, fetchErr)
			if siblingCanceled[i] || (lowestFailureIdx >= 0 && i > lowestFailureIdx) {
				return fetchErr
			}

			if lowestFailureIdx < 0 || i < lowestFailureIdx {
				lowestFailureIdx = i
				for j := i + 1; j < len(handles); j++ {
					if completed[j] || siblingCanceled[j] {
						continue
					}
					siblingCanceled[j] = true
					requestCancels[j]()
				}
			}

			return fetchErr
		})
	}

	if waitErr := group.Wait(); waitErr != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		mu.Lock()
		defer mu.Unlock()
		for i, err := range errs {
			if err != nil && !siblingCanceled[i] {
				return nil, err
			}
		}

		return nil, waitErr
	}

	return responses, nil
}

// remoteScanTasks decodes each server task envelope into domain FileScanTasks.
// The decoder must run before envelopes are combined because delete-file
// references are indexes into the envelope-local delete-files array.
func remoteScanTasks(envelopes []ScanTasks, req table.ScanPlanningRequest) ([]table.FileScanTask, error) {
	var result []table.FileScanTask
	plans := newPartitionDecodePlanCache()
	for i, envelope := range envelopes {
		// Keep empty completed plans cheap and allow the low-level planner method
		// to return no tasks even when a caller did not provide metadata.
		if len(envelope.FileScanTasks) == 0 && len(envelope.DeleteFiles) == 0 {
			continue
		}

		tasks, err := decodeScanTasks(envelope, req.Metadata, req.Schema, req.RowFilter, plans)
		if err != nil {
			return nil, fmt.Errorf("decoding remote scan task envelope %d: %w", i, err)
		}
		result = append(result, tasks...)
	}

	return result, nil
}

// planIOFromCredentials wraps a plan's prefix-scoped storage credentials in a
// lazy PlanIO. The returned IO resolves the longest matching credential prefix
// for each data or delete file it opens, overlaid on the table's base IO props
// so vended keys win on collision. With none vended it returns a nil PlanIO,
// and the scan falls back to the table's own FileIO.
func planIOFromCredentials(creds []StorageCredential, location string, baseProps iceberg.Properties) table.PlanIO {
	if len(creds) == 0 {
		return nil
	}

	return &planScopedIO{refresher: &vendedCredentialRefresher{
		mu:          semaphore.NewWeighted(1),
		location:    location,
		props:       maps.Clone(baseProps),
		credentials: slices.Clone(creds),
		// fetchCreds stays nil: a plan's creds can't be renewed via the
		// table-credentials endpoint, so an expiry is fatal, not a re-fetch.
	}}
}

// planScopedIO lazily builds and caches a FileIO from a plan's vended creds,
// reusing vendedCredentialRefresher for expiry handling. Satisfies table.PlanIO.
type planScopedIO struct {
	refresher *vendedCredentialRefresher
}

func (p *planScopedIO) Load(ctx context.Context) (iceio.IO, error) {
	return p.refresher.loadFS(ctx)
}

func (p *planScopedIO) Close() error {
	return p.refresher.close()
}

// planIOWithCleanup keeps the remote plan alive for the lifetime of the
// plan-scoped IO. Scan owns the IO through its PlanIO state and closes it only
// after replacing the plan and releasing active readers, so cancellation cannot
// invalidate credentials before ReadTasks opens the files.
type planIOWithCleanup struct {
	io      table.PlanIO
	cleanup func()
	once    sync.Once
}

func (p *planIOWithCleanup) Load(ctx context.Context) (iceio.IO, error) {
	return p.io.Load(ctx)
}

func (p *planIOWithCleanup) Close() error {
	defer p.once.Do(p.cleanup)

	return p.io.Close()
}

// planTableScanRequestFrom builds the planTableScan request body from the
// planner-level request, serializing the row filter to ExpressionParser JSON. A
// nil or always-true filter is left off so the server plans without one.
func planTableScanRequestFrom(req table.ScanPlanningRequest) (PlanTableScanRequest, error) {
	out := PlanTableScanRequest{
		SnapshotID:        req.SnapshotID,
		MinRowsRequested:  req.MinRowsRequested,
		CaseSensitive:     req.CaseSensitive,
		UseSnapshotSchema: req.UseSnapshotSchema,
		StatsFields:       req.StatsFields,
	}
	// `*` is table.Scan's local sentinel, not a REST FieldName. A planner
	// normally expands it to schema field names before reaching this function;
	// omitting it here is the safe fallback for direct callers without a schema.
	if len(req.SelectedFields) > 0 && !slices.Contains(req.SelectedFields, "*") {
		out.Select = slices.Clone(req.SelectedFields)
	}

	if req.RowFilter != nil && !req.RowFilter.Equals(iceberg.AlwaysTrue{}) {
		filter, err := marshalScanFilter(req)
		if err != nil {
			return PlanTableScanRequest{}, err
		}
		out.Filter = filter
	}

	return out, nil
}

// marshalScanFilter binds the row filter before encoding it: a bound term gives
// the encoder the field type it needs for ambiguous literals (a bare timestamp
// can't pick timestamp vs timestamptz and won't serialize unbound), and binding
// errors loudly on an expression type it doesn't know.
func marshalScanFilter(req table.ScanPlanningRequest) ([]byte, error) {
	if req.Metadata == nil {
		return nil, fmt.Errorf("%w: cannot encode scan filter without table metadata", iceberg.ErrInvalidArgument)
	}

	caseSensitive := true
	if req.CaseSensitive != nil {
		caseSensitive = *req.CaseSensitive
	}

	schema := req.Schema
	if schema == nil {
		schema = req.Metadata.CurrentSchema()
	}
	if schema == nil {
		return nil, fmt.Errorf("%w: cannot encode scan filter without a scan schema", iceberg.ErrInvalidArgument)
	}

	bound, err := iceberg.BindExpr(schema, req.RowFilter, caseSensitive)
	if err != nil {
		return nil, fmt.Errorf("%w: binding scan filter: %s", iceberg.ErrInvalidArgument, err)
	}

	filter, err := json.Marshal(bound)
	if err != nil {
		return nil, fmt.Errorf("%w: encoding scan filter: %s", iceberg.ErrInvalidArgument, err)
	}

	return filter, nil
}

// --- Low-level client methods -----------------------------------------------

// PlanTableScan submits a scan plan. A completed or submitted plan returns
// (resp, nil); callers branch on resp.Status for the plan-id (completed) vs
// poll (submitted) distinction. A failed plan returns a zero response and a
// non-nil *PlanFailedError (errors.Is(err, ErrPlanFailed)) so the if-err idiom
// does not mistake a failure for success; the server detail rides on the error.
// Any other status — including the empty status of a 200 with no body, which
// bypasses the response UnmarshalJSON validation — returns an ErrRESTError so a
// malformed response cannot masquerade as an empty completed plan.
//
// Transient HTTP responses (408, 429, 500, 502, 503, 504) are retried up to
// three times with jittered backoff and the same idempotency key. Retry-After
// is honored, capped at five seconds when ctx has no deadline. Transport
// failures and malformed successful responses are not retried.
func (r *Catalog) PlanTableScan(ctx context.Context, ident table.Identifier, req PlanTableScanRequest) (result PlanTableScanResponse, err error) {
	ctx, finish := scanmetrics.Start(ctx, "plan")
	defer func() {
		finish(err)
	}()

	if err := r.endpoints.check(endpointPlanTableScan); err != nil {
		return PlanTableScanResponse{}, err
	}

	path, err := r.scanPlanningPath(endpointPlanTableScan, ident)
	if err != nil {
		return PlanTableScanResponse{}, err
	}

	headers, err := scanPlanningHeaders(req.IdempotencyKey, req.AccessDelegation, true)
	if err != nil {
		return PlanTableScanResponse{}, err
	}

	resp, err := retryScanPlanning(ctx, "plan", func() (PlanTableScanResponse, error) {
		return doPost[PlanTableScanRequest, PlanTableScanResponse](
			ctx, r.baseURI, path, req, r.cl,
			map[int]error{http.StatusNotFound: ErrRESTError},
			withHeaders(headers), withErrorTypeOverride(planTableScanErrorTypes))
	})
	if err != nil {
		return PlanTableScanResponse{}, err
	}
	switch resp.Status {
	case PlanStatusCompleted, PlanStatusSubmitted:
		return resp, nil
	case PlanStatusFailed:
		return PlanTableScanResponse{}, &PlanFailedError{Detail: resp.Error}
	default:
		return PlanTableScanResponse{}, fmt.Errorf(
			"%w: planTableScan response has invalid status %q", ErrRESTError, resp.Status)
	}
}

// FetchPlanningResult polls a previously submitted plan. opts.AccessDelegation
// is sent as the X-Iceberg-Access-Delegation header so an async poll can still
// receive plan-scoped storage credentials: the spec defines data-access on this
// endpoint, and the completed-async result is where those credentials are vended.
//
// completed and submitted return (resp, nil) and callers branch on resp.Status
// (done vs poll again). The terminal failure states surface as errors so an
// if-err poll loop cannot mistake them for an empty scan: failed returns a
// *PlanFailedError (errors.Is(err, ErrPlanFailed)) and cancelled returns
// ErrPlanCancelled. A 404 is split by the response error.type: a forgotten
// plan-id (NoSuchPlanIdException) is ErrPlanExpired, while a gone table or
// namespace is catalog.ErrNoSuchTable / catalog.ErrNoSuchNamespace, so the poller
// can tell retry-with-a-new-plan from abort. A bare or unrecognized 404 stays an
// ambiguous ErrRESTError rather than being guessed as an expiry.
func (r *Catalog) FetchPlanningResult(ctx context.Context, ident table.Identifier, planID string, opts FetchPlanningResultOptions) (result FetchPlanningResultResponse, err error) {
	ctx, finish := scanmetrics.Start(ctx, "fetch-result")
	defer func() {
		if errors.Is(err, ErrPlanExpired) {
			scanmetrics.Expired(ctx, "fetch-result")
		}
		finish(err)
	}()

	if err := r.endpoints.check(endpointFetchPlanResult); err != nil {
		return FetchPlanningResultResponse{}, err
	}

	path, err := r.scanPlanningPath(endpointFetchPlanResult, ident, planID)
	if err != nil {
		return FetchPlanningResultResponse{}, err
	}

	headers, err := scanPlanningHeaders(nil, opts.AccessDelegation, false)
	if err != nil {
		return FetchPlanningResultResponse{}, err
	}

	resp, err := doGet[FetchPlanningResultResponse](
		ctx, r.baseURI, path, r.cl,
		map[int]error{http.StatusNotFound: ErrRESTError},
		withHeaders(headers), withErrorTypeOverride(fetchPlanningResultErrorTypes))
	if err != nil {
		return FetchPlanningResultResponse{}, err
	}
	switch resp.Status {
	case PlanStatusFailed:
		return FetchPlanningResultResponse{}, &PlanFailedError{Detail: resp.Error}
	case PlanStatusCancelled:
		return FetchPlanningResultResponse{}, ErrPlanCancelled
	}

	return resp, nil
}

// CancelPlanning cancels a server-side plan. Callers should cancel on context
// cancellation using a detached context with a short timeout. The spec supports
// idempotency and access-delegation headers on cancel; this low-level method
// deliberately defers those until a cancel options type is added, and suppresses
// the session-default access-delegation header (cancel vends no credentials). A
// 404 (already-expired or unknown plan) is not special-cased: cancel is
// best-effort, so the generic REST error is acceptable.
func (r *Catalog) CancelPlanning(ctx context.Context, ident table.Identifier, planID string) (err error) {
	ctx, finish := scanmetrics.Start(ctx, "cancel")
	defer func() { finish(err) }()
	if err := r.endpoints.check(endpointCancelPlanning); err != nil {
		return err
	}

	path, err := r.scanPlanningPath(endpointCancelPlanning, ident, planID)
	if err != nil {
		return err
	}

	_, err = doDelete[struct{}](
		ctx, r.baseURI, path, r.cl, nil,
		withSuppressedHeaders(headerIcebergAccessDelegation))

	return err
}

// FetchScanTasks fetches the scan tasks for a plan-task handle returned by a
// completed plan. A 404 is split by the response error.type: an expired
// plan-task handle (NoSuchPlanTaskException) is ErrNoSuchPlanTask, while a gone
// table or namespace is catalog.ErrNoSuchTable / catalog.ErrNoSuchNamespace, so
// a caller fanning out over handles can tell a handle expiry from the table
// vanishing. A bare or unrecognized 404 stays an ambiguous ErrRESTError. An
// empty 200 body is rejected (requireBody) so a truncated response is not read
// as a successfully completed empty task set.
// Transient HTTP responses use the same retry policy as [Catalog.PlanTableScan],
// retaining the idempotency key and plan-task handle across attempts.
func (r *Catalog) FetchScanTasks(ctx context.Context, ident table.Identifier, req FetchScanTasksRequest) (result FetchScanTasksResponse, err error) {
	ctx, finish := scanmetrics.Start(ctx, "fetch-tasks")
	defer func() {
		if errors.Is(err, ErrNoSuchPlanTask) {
			scanmetrics.Expired(ctx, "fetch-tasks")
		}
		finish(err)
	}()

	if err := r.endpoints.check(endpointFetchScanTasks); err != nil {
		return FetchScanTasksResponse{}, err
	}

	path, err := r.scanPlanningPath(endpointFetchScanTasks, ident)
	if err != nil {
		return FetchScanTasksResponse{}, err
	}

	headers, err := scanPlanningHeaders(req.IdempotencyKey, nil, true)
	if err != nil {
		return FetchScanTasksResponse{}, err
	}

	return retryScanPlanning(ctx, "fetch-tasks", func() (FetchScanTasksResponse, error) {
		return doPost[FetchScanTasksRequest, FetchScanTasksResponse](
			ctx, r.baseURI, path, req, r.cl,
			map[int]error{http.StatusNotFound: ErrRESTError},
			withHeaders(headers), withSuppressedHeaders(headerIcebergAccessDelegation),
			withErrorTypeOverride(fetchScanTasksErrorTypes), requireBody())
	})
}

// WaitForPlan polls a submitted plan to completion using jittered backoff. It
// retries the idempotent-GET status set used by Java's REST transport
// (408/429/500/502/503/504); every other HTTP error is terminal. A positive
// Retry-After hint on a retried error response overrides the backoff, floored at
// MinDelay and — only when the caller set no context deadline — capped at
// MaxDelay; with a deadline the hint is honoured in full, bounded by that
// deadline (so a rate-limiting 429/503 is respected rather than clamped). The
// hint is read only from error responses, not from a 200 "submitted" body.
//
// It returns the completed planning result verbatim: a completed plan may carry
// file-scan-tasks that are ready to read and/or plan-tasks (opaque handles that
// still need FetchScanTasks). WaitForPlan does not expand plan-tasks — driving
// that fanout is the caller's (PlanFiles') job — so "completed" does not imply
// every task is a file-scan-task.
//
// Polling time is bounded by the caller's context deadline (the Go equivalent of
// Java's REST_SCAN_PLANNING_POLL_TIMEOUT_MS). opts.MaxRetries is a separate
// safety net for a caller that set no deadline: it caps the poll count and, when
// exhausted, cancels the plan server-side and returns ErrPlanPollExhausted. The
// default cap is disabled when the context has a deadline, so a longer deadline
// is never silently cut short by the retry count.
//
// On context cancellation — or on retry exhaustion — WaitForPlan cancels the plan
// server-side and then returns (the wrapped context error, errors.Is
// context.Canceled / context.DeadlineExceeded; or ErrPlanPollExhausted). The
// cancel is synchronous so a short-lived caller cannot exit before the plan is
// released; it is best-effort and bounded by opts.CancelGracePeriod, so a stalled
// cancel endpoint delays the return by at most that grace (see abandonPlan).
//
// It requires the fetchPlanningResult endpoint (returning ErrEndpointNotSupported
// otherwise) and a non-empty planID (ErrInvalidArgument otherwise); the cancel is
// best-effort, so cancelPlanning is not required. Other terminal errors from the
// poll — *PlanFailedError (errors.Is ErrPlanFailed), ErrPlanCancelled,
// ErrPlanExpired, or a gone table/namespace — are returned unchanged.
func (r *Catalog) WaitForPlan(ctx context.Context, ident table.Identifier, planID string, opts WaitForPlanOptions) (CompletedPlanningResult, error) {
	if err := r.endpoints.check(endpointFetchPlanResult); err != nil {
		return CompletedPlanningResult{}, err
	}
	if planID == "" {
		return CompletedPlanningResult{}, fmt.Errorf("%w: empty plan-id", iceberg.ErrInvalidArgument)
	}

	_, hasDeadline := ctx.Deadline()
	minDelay, maxDelay, grace, maxRetries := resolveWaitOptions(opts, hasDeadline)
	unlimited := maxRetries < 0
	fetchOpts := FetchPlanningResultOptions{AccessDelegation: opts.AccessDelegation}

	// Decorrelated jitter (AWS "Exponential Backoff And Jitter"): each sleep is a
	// random duration in [minDelay, prevSleep*3] capped at maxDelay, so every wait
	// stays within [minDelay, maxDelay] — minDelay a true floor, maxDelay a true
	// cap — while spreading concurrent pollers to avoid a thundering herd. (This is
	// deliberately not table.go's capped-exponential commit-retry backoff: many
	// clients poll one plan endpoint, so decorrelated jitter de-synchronises them.)
	sleep := minDelay
	retries := 0
	for {
		retryAfter := time.Duration(0)
		resp, err := r.FetchPlanningResult(ctx, ident, planID, fetchOpts)
		switch {
		case err != nil:
			// A context cancellation mid-request surfaces as a transport error;
			// funnel it through the same abandon-and-report path as a cancellation
			// observed during the backoff sleep so the plan is cancelled server-side.
			if ctxErr := ctx.Err(); ctxErr != nil {
				r.abandonPlan(ctx, ident, planID, grace)

				return CompletedPlanningResult{}, fmt.Errorf("waiting for plan %q on %v: %w", planID, ident, ctxErr)
			}

			// Retry the same idempotent GET statuses as Java's REST transport. Every
			// other error — failed (*PlanFailedError), cancelled (ErrPlanCancelled),
			// expired (ErrPlanExpired), a gone table/namespace, or a non-retryable
			// HTTP status such as 501 — is terminal and propagates unchanged.
			var retryable bool
			retryAfter, retryable = scanPlanPollRetry(err)
			if !retryable {
				return CompletedPlanningResult{}, err
			}
		case resp.Status == PlanStatusCompleted:
			return CompletedPlanningResult{
				Status:             resp.Status,
				ScanTasks:          resp.ScanTasks,
				StorageCredentials: resp.StorageCredentials,
			}, nil
		case resp.Status == PlanStatusSubmitted:
			// keep polling
		default:
			// FetchPlanningResult maps failed/cancelled to errors and its decoder
			// rejects unknown statuses, so only completed/submitted reach here;
			// guard the invariant rather than silently spin.
			return CompletedPlanningResult{}, fmt.Errorf(
				"%w: unexpected plan status %q while waiting for plan", ErrRESTError, resp.Status)
		}

		// The plan is still in flight (submitted or a retried HTTP response). Stop if the
		// retry budget is spent — the safety net for a caller that set no deadline.
		// Cancel the still-active plan server-side first, like the cancellation path
		// (the OpenAPI contract and Java's cleanupPlanResources free abandoned plans).
		if !unlimited && retries >= maxRetries {
			r.abandonPlan(ctx, ident, planID, grace)

			return CompletedPlanningResult{}, fmt.Errorf(
				"waiting for plan %q on %v: %w after %d retries", planID, ident, ErrPlanPollExhausted, retries)
		}
		retries++

		// Back off before the next poll. On cancellation, cancel the plan
		// server-side (bounded, best-effort) before returning the context error.
		sleep = nextScanPlanBackoff(sleep, minDelay, maxDelay)
		sleep = applyRetryAfter(sleep, retryAfter, minDelay, maxDelay, hasDeadline)
		timer := time.NewTimer(sleep)
		select {
		case <-ctx.Done():
			timer.Stop()
			r.abandonPlan(ctx, ident, planID, grace)

			return CompletedPlanningResult{}, fmt.Errorf("waiting for plan %q on %v: %w", planID, ident, ctx.Err())
		case <-timer.C:
			if err != nil {
				scanmetrics.Retry(ctx, "fetch-result")
			}
		}
	}
}

// scanPlanPollRetry reports whether err came from an HTTP status that Java's
// REST transport retries for an idempotent GET. It also returns a positive
// Retry-After delay, capped later by the caller's MaxDelay. The fallback for
// ErrServiceUnavailable preserves compatibility with callers or transports that
// return the sentinel without the package's concrete errorResponse wrapper.
func scanPlanPollRetry(err error) (time.Duration, bool) {
	var restErr errorResponse
	if errors.As(err, &restErr) {
		switch restErr.statusCode {
		case http.StatusRequestTimeout,
			http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return parseRetryAfter(restErr.retryAfter), true
		default:
			return 0, false
		}
	}

	return 0, errors.Is(err, ErrServiceUnavailable)
}

// applyRetryAfter overrides the jittered backoff with a positive server
// Retry-After hint, floored at minDelay. The hint is capped at maxDelay only when
// the caller set no context deadline: with a deadline the poll loop's ctx.Done()
// select already bounds an over-long hint, so genuine 429/503 backpressure is
// honoured in full; without one, maxDelay stays the safety cap so a hostile or
// accidental hint cannot become a single unbounded sleep. A non-positive hint
// leaves the backoff unchanged.
func applyRetryAfter(backoff, retryAfter, minDelay, maxDelay time.Duration, hasDeadline bool) time.Duration {
	if retryAfter <= 0 {
		return backoff
	}

	sleep := max(retryAfter, minDelay)
	if !hasDeadline {
		sleep = min(sleep, maxDelay)
	}

	return sleep
}

// parseRetryAfter accepts both Retry-After forms from RFC 9110: integer seconds
// or an HTTP date. Invalid, expired, and non-positive values are ignored.
func parseRetryAfter(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		if seconds <= 0 {
			return 0
		}
		const maxDuration = time.Duration(1<<63 - 1)
		if seconds > int64(maxDuration/time.Second) {
			return maxDuration
		}

		return time.Duration(seconds) * time.Second
	}

	retryAt, err := http.ParseTime(value)
	if err != nil {
		return 0
	}
	delay := time.Until(retryAt)
	if delay <= 0 {
		return 0
	}

	return delay
}

// resolveWaitOptions normalizes user options into the effective backoff floor,
// cap, cancel grace, and retry budget. A zero (or negative, for the durations)
// field takes its DefaultWaitForPlanOptions value; an inverted floor/cap is
// clamped so maxDelay >= minDelay (the invariant nextScanPlanBackoff relies on).
//
// A negative maxRetries is preserved as the "unlimited, bound only by context"
// sentinel. A zero maxRetries resolves to unlimited when the context already has
// a deadline (that deadline is the real bound, so the default cap must not cut it
// short) and to the default cap otherwise (the safety net for a no-deadline
// caller). An explicit positive maxRetries is always honoured.
func resolveWaitOptions(opts WaitForPlanOptions, hasDeadline bool) (minDelay, maxDelay, grace time.Duration, maxRetries int) {
	minDelay, maxDelay, grace, maxRetries = opts.MinDelay, opts.MaxDelay, opts.CancelGracePeriod, opts.MaxRetries
	if minDelay <= 0 {
		minDelay = DefaultWaitForPlanOptions.MinDelay
	}
	if maxDelay <= 0 {
		maxDelay = DefaultWaitForPlanOptions.MaxDelay
	}
	if maxDelay < minDelay {
		maxDelay = minDelay
	}
	if grace <= 0 {
		grace = DefaultWaitForPlanOptions.CancelGracePeriod
	}
	if maxRetries == 0 {
		if hasDeadline {
			maxRetries = -1
		} else {
			maxRetries = DefaultWaitForPlanOptions.MaxRetries
		}
	}

	return minDelay, maxDelay, grace, maxRetries
}

// nextScanPlanBackoff returns the next decorrelated-jitter sleep: a random
// duration in [minDelay, min(prev*3, maxDelay)]. Callers guarantee
// 0 < minDelay <= maxDelay and prev >= minDelay, so the ceiling is >= minDelay
// and the sampled span is non-negative.
func nextScanPlanBackoff(prev, minDelay, maxDelay time.Duration) time.Duration {
	// Grow the ceiling geometrically but overflow-safe: prev*3 is only formed when
	// it is known to stay <= maxDelay (hence <= MaxInt64); otherwise the ceiling is
	// maxDelay. Without this guard a large prev (e.g. maxDelay == MaxInt64) would
	// overflow prev*3 into a non-positive rand.Int64N argument and panic.
	ceiling := maxDelay
	if prev <= maxDelay/3 {
		ceiling = prev * 3
	}

	// The +1 makes the upper bound inclusive; minDelay > 0 keeps the argument
	// within (0, MaxInt64], so rand.Int64N never sees a non-positive n.
	//nolint:gosec // non-security randomness, jitter for poll backoff spread
	return minDelay + time.Duration(rand.Int64N(int64(ceiling-minDelay)+1))
}

// abandonPlan cancels a plan server-side after the caller's context is done. It
// runs synchronously — WaitForPlan calls CancelPlanning and only then returns —
// so a short-lived caller cannot exit before the plan is released, matching the
// #1178 "call CancelPlanning, then return" contract. The caller's context is
// already cancelled, so the DELETE runs on one detached from cancellation
// (values, e.g. auth, preserved) bounded by grace, which caps how long a stalled
// endpoint can delay the return. It is best-effort: the plan expires server-side
// regardless, so any error — including a server that never advertised the cancel
// endpoint — is dropped.
func (r *Catalog) abandonPlan(ctx context.Context, ident table.Identifier, planID string, grace time.Duration) {
	cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), grace)
	defer cancel()

	_ = r.CancelPlanning(cctx, ident, planID)
}

func (r *Catalog) scanPlanningPath(ep endpoint, ident table.Identifier, extra ...string) ([]string, error) {
	ns, tbl, err := r.splitIdentForPath(ident)
	if err != nil {
		return nil, err
	}

	// extra carries opaque, server-issued path segments (the plan-id). The spec
	// puts no character restriction on them, and baseURI.JoinPath path.Cleans its
	// arguments — so an unescaped id containing '/' would split into segments and
	// a '..'/'.' segment would resolve to a different endpoint (e.g. ".." on the
	// plan path lands on .../tasks). Escape each as a single literal path segment.
	params := make([]string, 0, 2+len(extra))
	params = append(params, ns, tbl)
	for _, seg := range extra {
		params = append(params, escapeOpaquePathSegment(seg))
	}

	return ep.reqPath(params...)
}

// escapeOpaquePathSegment percent-encodes an opaque string so it survives
// url.URL.JoinPath as a single literal path segment. url.PathEscape handles '/',
// '%', spaces, etc., but leaves a pure-dot segment ("." or "..") unescaped, which
// JoinPath's path.Clean would then resolve as a dot-segment; encode those dots so
// the segment is preserved verbatim.
func escapeOpaquePathSegment(s string) string {
	switch escaped := url.PathEscape(s); escaped {
	case ".":
		return "%2E"
	case "..":
		return "%2E%2E"
	default:
		return escaped
	}
}

func scanPlanningHeaders(idempotencyKey, accessDelegation *string, includeIdempotency bool) (map[string]string, error) {
	headers := make(map[string]string, 2)
	if includeIdempotency {
		key, err := idempotencyHeaderValue(idempotencyKey)
		if err != nil {
			return nil, err
		}
		headers[headerIdempotencyKey] = key
	}
	if accessDelegation != nil {
		headers[headerIcebergAccessDelegation] = *accessDelegation
	}
	if len(headers) == 0 {
		return nil, nil
	}

	return headers, nil
}

func idempotencyHeaderValue(idempotencyKey *string) (string, error) {
	if idempotencyKey == nil {
		// Plan and task POSTs always send an idempotency key. The spec pins it
		// to a UUIDv7 string (RFC 9562) so a server can key its dedup window off
		// the embedded timestamp, matching Java's UUIDUtil.generateUuidV7(). A
		// v4 key would be treated as undefined by such servers. All automatic
		// retries within this call reuse the generated key.
		key, err := uuid.NewV7()
		if err != nil {
			return "", fmt.Errorf("generating idempotency key: %w", err)
		}

		return key.String(), nil
	}

	parsed, err := uuid.Parse(*idempotencyKey)
	if err != nil {
		return "", fmt.Errorf("%w: invalid idempotency key %q", iceberg.ErrInvalidArgument, *idempotencyKey)
	}
	// uuid.Parse is lenient (its docs say it should not be used for validation):
	// it accepts the URN, brace, and un-hyphenated 32-char forms. The spec's
	// header schema pins this to the 36-char canonical hyphenated form, so reject
	// anything whose canonical rendering differs (case-insensitively) from the
	// input rather than forward a nonconforming encoding.
	if !strings.EqualFold(parsed.String(), *idempotencyKey) {
		return "", fmt.Errorf("%w: idempotency key %q must be a canonical hyphenated UUID",
			iceberg.ErrInvalidArgument, *idempotencyKey)
	}
	// The spec pins the header to UUIDv7, so reject other versions rather than
	// forward a key a timestamp-keyed server would treat as undefined. RFC 9562
	// UUIDv7 also uses the RFC 4122 variant, so require that too: a canonical
	// string can carry the version-7 nibble while setting non-RFC variant bits.
	if parsed.Version() != 7 || parsed.Variant() != uuid.RFC4122 {
		return "", fmt.Errorf("%w: idempotency key %q must be an RFC 4122 UUIDv7 (got v%d, variant %v)",
			iceberg.ErrInvalidArgument, *idempotencyKey, int(parsed.Version()), parsed.Variant())
	}

	return *idempotencyKey, nil
}

// --- Wire types -------------------------------------------------------------

// PlanStatus is the status of a server-side plan.
type PlanStatus string

const (
	PlanStatusCompleted PlanStatus = "completed"
	PlanStatusSubmitted PlanStatus = "submitted"
	// PlanStatusCancelled is valid when polling a submitted plan, but invalid
	// as a planTableScan response. PlanTableScan and WaitForPlan should treat a
	// cancelled initial planning response as an error.
	PlanStatusCancelled PlanStatus = "cancelled"
	PlanStatusFailed    PlanStatus = "failed"
)

// PlanningError is the REST ErrorModel payload carried by the error arm of a
// failed planning result. It mirrors the package's internal error wire shape
// but is a dedicated exported struct so it renders cleanly in godoc and does
// not leak an unexported type or its unexported fields into the public API.
type PlanningError struct {
	Message string   `json:"message"`
	Type    string   `json:"type"`
	Code    int      `json:"code"`
	Stack   []string `json:"stack,omitempty"`
}

// ScanTasks carries the task payload shared by completed planning responses and
// fetchScanTasks responses.
type ScanTasks struct {
	PlanTasks     []string           `json:"plan-tasks,omitempty"`
	FileScanTasks []RESTFileScanTask `json:"file-scan-tasks,omitempty"`
	DeleteFiles   []RESTDeleteFile   `json:"delete-files,omitempty"`
}

// CompletedPlanningResult is the completed arm of the planning-result union.
// planTableScan carries the plan-id on PlanTableScanResponse; fetchPlanningResult
// omits it.
type CompletedPlanningResult struct {
	Status PlanStatus `json:"status"`
	ScanTasks
	StorageCredentials []StorageCredential `json:"storage-credentials,omitempty"`
}

// PlanTableScanRequest is the POST .../plan request body. Filter is the
// ExpressionParser-format JSON of the scan's row filter, produced by
// json.Marshal over the BooleanExpression (see planTableScanRequestFrom).
//
// Point-in-time only for now: the spec's incremental start-snapshot-id /
// end-snapshot-id fields are deliberately omitted and land with the incremental
// phase, together with the matching fields on table.ScanPlanningRequest, so the
// wire type and the seam stay in agreement.
type PlanTableScanRequest struct {
	// IdempotencyKey is sent as the Idempotency-Key header, not in the JSON body.
	// If set, it must be a UUIDv7 string (RFC 9562). If nil, a fresh UUIDv7 is
	// generated per call and reused for automatic retries. Supply an explicit
	// key to preserve idempotency across separate calls.
	IdempotencyKey *string `json:"-"`
	// AccessDelegation is sent as the X-Iceberg-Access-Delegation header, not
	// in the JSON body. Nil uses the catalog default.
	AccessDelegation *string `json:"-"`

	SnapshotID        *int64          `json:"snapshot-id,omitempty"`
	Select            []string        `json:"select,omitempty"`
	Filter            json.RawMessage `json:"filter,omitempty"`
	MinRowsRequested  *int64          `json:"min-rows-requested,omitempty"`
	CaseSensitive     *bool           `json:"case-sensitive,omitempty"`
	UseSnapshotSchema *bool           `json:"use-snapshot-schema,omitempty"`
	StatsFields       []string        `json:"stats-fields,omitempty"`
}

// PlanTableScanResponse is the POST .../plan response. The spec models this as
// a `status`-discriminated union; the flat struct carries every arm's fields
// with omitempty so none are discarded. Per the spec, plan-id is required for
// both completed (CompletedPlanningWithIDResult) and submitted
// (AsyncPlanningResult) responses here; the wire decoder must validate
// PlanID != nil at unmarshal rather than rely on the omitempty pointer. A
// cancelled status is invalid for this endpoint and must be treated as an
// error. A failed status decodes even when a non-compliant server omits or
// malforms Error; callers must branch on Status before dereferencing PlanID.
type PlanTableScanResponse struct {
	Status PlanStatus     `json:"status"`
	PlanID *string        `json:"plan-id,omitempty"`
	Error  *PlanningError `json:"error,omitempty"`
	ScanTasks
	StorageCredentials []StorageCredential `json:"storage-credentials,omitempty"`
}

func (r *PlanTableScanResponse) UnmarshalJSON(data []byte) error {
	var resp struct {
		Status PlanStatus      `json:"status"`
		PlanID *string         `json:"plan-id,omitempty"`
		Error  json.RawMessage `json:"error,omitempty"`
		ScanTasks
		StorageCredentials []StorageCredential `json:"storage-credentials,omitempty"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}

	switch resp.Status {
	case PlanStatusCompleted, PlanStatusSubmitted:
		if resp.PlanID == nil {
			return fmt.Errorf("%w: planTableScan response with status %q missing plan-id", ErrRESTError, resp.Status)
		}
	case PlanStatusFailed:
		if resp.PlanID != nil {
			return fmt.Errorf("%w: planTableScan response with status %q must not include plan-id", ErrRESTError, resp.Status)
		}
	case PlanStatusCancelled:
		return fmt.Errorf("%w: planTableScan response has invalid status %q", ErrRESTError, resp.Status)
	default:
		return fmt.Errorf("%w: planTableScan response has unknown status %q", ErrRESTError, resp.Status)
	}
	if resp.Status != PlanStatusFailed && len(resp.Error) > 0 && !isJSONNull(resp.Error) {
		return fmt.Errorf("%w: planTableScan response includes error for status %q", ErrRESTError, resp.Status)
	}
	if err := validatePlanningTaskEnvelope(data, resp.Status, resp.ScanTasks, "planTableScan"); err != nil {
		return err
	}

	*r = PlanTableScanResponse{
		Status:             resp.Status,
		PlanID:             resp.PlanID,
		Error:              decodePlanningError(resp.Error),
		ScanTasks:          resp.ScanTasks,
		StorageCredentials: resp.StorageCredentials,
	}

	return nil
}

// FetchPlanningResultResponse is the GET .../plan/{plan-id} poll response. Same
// `status`-discriminated union (completed / submitted / cancelled / failed).
type FetchPlanningResultResponse struct {
	Status PlanStatus     `json:"status"`
	Error  *PlanningError `json:"error,omitempty"`
	ScanTasks
	StorageCredentials []StorageCredential `json:"storage-credentials,omitempty"`
}

func (r *FetchPlanningResultResponse) UnmarshalJSON(data []byte) error {
	var resp struct {
		Status PlanStatus      `json:"status"`
		Error  json.RawMessage `json:"error,omitempty"`
		ScanTasks
		StorageCredentials []StorageCredential `json:"storage-credentials,omitempty"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}

	switch resp.Status {
	case PlanStatusCompleted, PlanStatusSubmitted, PlanStatusCancelled:
	case PlanStatusFailed:
	default:
		return fmt.Errorf("%w: fetchPlanningResult response has unknown status %q", ErrRESTError, resp.Status)
	}
	if resp.Status != PlanStatusFailed && len(resp.Error) > 0 && !isJSONNull(resp.Error) {
		return fmt.Errorf("%w: fetchPlanningResult response includes error for status %q", ErrRESTError, resp.Status)
	}
	if err := validatePlanningTaskEnvelope(data, resp.Status, resp.ScanTasks, "fetchPlanningResult"); err != nil {
		return err
	}

	*r = FetchPlanningResultResponse{
		Status:             resp.Status,
		Error:              decodePlanningError(resp.Error),
		ScanTasks:          resp.ScanTasks,
		StorageCredentials: resp.StorageCredentials,
	}

	return nil
}

// decodePlanningError parses the ErrorModel carried by a failed planning arm.
// The spec requires an object, but Java deliberately treats a missing, null, or
// primitive value as absent so the failed status itself still reaches callers.
func decodePlanningError(raw json.RawMessage) *PlanningError {
	if len(raw) == 0 {
		return nil
	}

	var detail *PlanningError
	if err := json.Unmarshal(raw, &detail); err != nil {
		return nil
	}

	return detail
}

// FetchScanTasksRequest is the POST .../tasks request body.
type FetchScanTasksRequest struct {
	// IdempotencyKey is sent as the Idempotency-Key header, not in the JSON body.
	// If set, it must be a UUIDv7 string (RFC 9562). If nil, a fresh UUIDv7 is
	// generated per call and reused for automatic retries. Supply an explicit
	// key to preserve idempotency across separate calls.
	IdempotencyKey *string `json:"-"`

	PlanTask string `json:"plan-task"`
}

// FetchScanTasksResponse is the POST .../tasks response. May itself return more
// plan-tasks for further fanout.
type FetchScanTasksResponse struct {
	ScanTasks
}

// DefaultWaitForPlanOptions is the conservative polling backoff, cancel grace,
// and retry budget used when callers pass the zero-value WaitForPlanOptions.
// MaxRetries mirrors Java's maxRetries=10, but it applies only when the caller
// set no context deadline — with a deadline, the default cap is disabled so it
// cannot cut a long deadline short (see resolveWaitOptions). A caller expecting a
// long plan should therefore set a context deadline, not rely on this default.
var DefaultWaitForPlanOptions = WaitForPlanOptions{
	MinDelay:          100 * time.Millisecond,
	MaxDelay:          5 * time.Second,
	CancelGracePeriod: 5 * time.Second,
	MaxRetries:        10,
}

// WaitForPlanOptions tunes the polling backoff and bounds. The total polling wait
// is bounded by whichever comes first: the caller's context deadline
// (context.WithTimeout) or MaxRetries. There is deliberately no Timeout field —
// the context is the time bound, avoiding a duplicated deadline and its
// zero-value footgun. Zero MinDelay/MaxDelay/CancelGracePeriod values use
// DefaultWaitForPlanOptions. Zero MaxRetries uses the default only without a
// context deadline; with a deadline it is unlimited so the deadline remains the
// authoritative bound.
type WaitForPlanOptions struct {
	MinDelay time.Duration
	MaxDelay time.Duration
	// CancelGracePeriod bounds the synchronous best-effort server-side cancel
	// issued once the caller's context is done: it is the most WaitForPlan's
	// return can be delayed past the deadline when the cancel endpoint stalls. A
	// caller in a hurry can shrink it; a zero value uses DefaultWaitForPlanOptions.
	CancelGracePeriod time.Duration
	// MaxRetries caps the number of poll retries (attempts after the first) before
	// WaitForPlan gives up with ErrPlanPollExhausted. It is the safety net for a
	// caller that set no context deadline. Zero uses DefaultWaitForPlanOptions when
	// the context has no deadline, and unlimited when it does (the deadline is the
	// real bound). A negative value means unlimited; an explicit positive value is
	// always honoured, even alongside a deadline.
	MaxRetries int
	// AccessDelegation is sent as the X-Iceberg-Access-Delegation header on each
	// poll; nil uses the catalog default. Needed for async plans so the
	// completed poll can return vended storage credentials.
	AccessDelegation *string
}

// FetchPlanningResultOptions carries per-call headers for fetchPlanningResult.
type FetchPlanningResultOptions struct {
	// AccessDelegation is sent as the X-Iceberg-Access-Delegation header; nil
	// uses the catalog default. The spec defines data-access on this endpoint,
	// so an async poll needs it to receive vended storage credentials.
	AccessDelegation *string
}
