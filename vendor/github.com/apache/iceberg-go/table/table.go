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

package table

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"log"
	"maps"
	"math"
	"math/rand/v2"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/internal"
	icebergio "github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/metrics"
	tblutils "github.com/apache/iceberg-go/table/internal"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"golang.org/x/sync/errgroup"
)

// ErrCommitFailed is the sentinel error returned by catalogs when a
// commit fails due to a concurrent modification (e.g. HTTP 409 Conflict
// from the REST catalog). Catalog implementations should wrap this
// error so that callers using errors.Is(err, table.ErrCommitFailed)
// can detect retryable commit conflicts.
//
// Currently only catalog/rest wraps this sentinel; Glue, SQL, and Hive
// catalogs return their conflict errors raw and will not trigger
// retries until follow-up work wires them through (tracked under
// issue #830).
var ErrCommitFailed = errors.New("commit failed, refresh and try again")

// ErrWriteIORequired is returned by write paths when the table's file system
// does not implement io.WriteFileIO. Commit retries also fail fast on this
// condition because manifest-list rebuilds need write access; skipping that
// rebuild can reintroduce stale-parent data loss. Callers should use
// errors.Is(err, ErrWriteIORequired) to detect the precise condition, or
// errors.Is(err, iceberg.ErrNotImplemented) for compatibility with older
// WriteRecords behavior.
var ErrWriteIORequired = fmt.Errorf("%w: file system does not implement WriteFileIO", iceberg.ErrNotImplemented)

const allManifestsMaxWorkers = 16

func allManifestsWorkerCount(snapshotCount int) int {
	return max(1, min(snapshotCount, allManifestsMaxWorkers))
}

// requireWriteFileIO should run immediately after resolving the table FS and
// before mutating transaction state such as automatic name mapping.
func requireWriteFileIO(fs icebergio.IO) (icebergio.WriteFileIO, error) {
	wfs, ok := fs.(icebergio.WriteFileIO)
	if !ok {
		return nil, ErrWriteIORequired
	}

	return wfs, nil
}

// ErrSnapshotNotFound is returned (wrapped) by metadata lookups when a
// snapshot ID does not exist in the table's snapshot list. Tests pin meaning
// via errors.Is(err, ErrSnapshotNotFound).
var ErrSnapshotNotFound = errors.New("snapshot not found")

type FSysF func(ctx context.Context) (icebergio.IO, error)

type Identifier = []string

type CatalogIO interface {
	LoadTable(context.Context, Identifier) (*Table, error)
	CommitTable(context.Context, Identifier, []Requirement, []Update) (Metadata, string, error)
}

type Table struct {
	identifier       Identifier
	metadata         Metadata
	metadataLocation string
	cat              CatalogIO
	fsF              FSysF
	manifestCache    *snapshotManifestCache
	planner          ScanPlanner
	// scanPlanningIOProps are the table-scoped FileIO properties supplied by a
	// catalog load response. They are separate from metadata properties because
	// REST catalogs can return FileIO configuration in the response's config
	// block.
	scanPlanningIOProps iceberg.Properties
	reporter            metrics.Reporter
	// reporterSet records whether a caller injected a reporter via
	// WithMetricsReporter. It distinguishes an explicit reporter (including an
	// explicit NopReporter opt-out) from the construction-time default, so
	// Refresh knows whether it may overwrite reporter with the catalog default.
	reporterSet bool
}

func (t Table) Equals(other Table) bool {
	return slices.Equal(t.identifier, other.identifier) &&
		t.metadataLocation == other.metadataLocation &&
		t.metadata.Equals(other.metadata)
}

func (t Table) Identifier() Identifier                       { return slices.Clone(t.identifier) }
func (t Table) Metadata() Metadata                           { return t.metadata }
func (t Table) MetadataLocation() string                     { return t.metadataLocation }
func (t Table) FS(ctx context.Context) (icebergio.IO, error) { return t.fsF(ctx) }
func (t Table) Schema() *iceberg.Schema                      { return t.metadata.CurrentSchema() }
func (t Table) Spec() iceberg.PartitionSpec                  { return t.metadata.PartitionSpec() }
func (t Table) SortOrder() SortOrder                         { return t.metadata.SortOrder() }
func (t Table) Properties() iceberg.Properties               { return t.metadata.Properties() }

// MetricsReporter returns the table's metrics reporter, never nil.
func (t Table) MetricsReporter() metrics.Reporter {
	if t.reporter == nil {
		return metrics.NopReporter{}
	}

	return t.reporter
}
func (t Table) NameMapping() iceberg.NameMapping     { return t.metadata.NameMapping() }
func (t Table) Location() string                     { return t.metadata.Location() }
func (t Table) CurrentSnapshot() *Snapshot           { return t.metadata.CurrentSnapshot() }
func (t Table) SnapshotByID(id int64) *Snapshot      { return t.metadata.SnapshotByID(id) }
func (t Table) SnapshotByName(name string) *Snapshot { return t.metadata.SnapshotByName(name) }
func (t Table) Schemas() map[int]*iceberg.Schema {
	m := make(map[int]*iceberg.Schema)
	for _, s := range t.metadata.Schemas() {
		m[s.ID] = s
	}

	return m
}

func (t Table) LocationProvider() (LocationProvider, error) {
	return LoadLocationProvider(t.metadata.Location(), t.metadata.Properties())
}

func (t Table) NewTransaction() *Transaction {
	txn, err := t.NewTransactionOnBranchWithError(MainBranch)
	if err != nil {
		return t.newBrokenTransaction(MainBranch, err)
	}

	return txn
}

// NewTransactionOnBranch creates a new transaction that commits to the named
// branch. Use [NewTransaction] to commit to the default "main" branch.
func (t Table) NewTransactionOnBranch(branch string) *Transaction {
	txn, err := t.NewTransactionOnBranchWithError(branch)
	if err != nil {
		return t.newBrokenTransaction(branch, err)
	}

	return txn
}

func (t Table) newBrokenTransaction(branch string, err error) *Transaction {
	return &Transaction{
		tbl:     &t,
		initErr: err,
		branch:  branch,
		reqs:    []Requirement{},
	}
}

// NewTransactionOnBranchWithError creates a new transaction and returns any metadata
// initialization error that prevents builder construction.
//
// This preserves the old non-failing constructor contract while allowing
// callers to receive the precise initialization error instead of hitting
// panic/undefined behavior later.
func (t Table) NewTransactionOnBranchWithError(branch string) (*Transaction, error) {
	for name, ref := range t.metadata.Refs() {
		if name != branch {
			continue
		}
		if ref.SnapshotRefType != BranchRef {
			return nil, fmt.Errorf("%w: ref %q is a %s; tags cannot be transaction targets",
				iceberg.ErrInvalidArgument, branch, ref.SnapshotRefType)
		}

		break
	}

	meta, err := MetadataBuilderFromBase(t.metadata, t.metadataLocation)
	if err != nil {
		return nil, err
	}

	return &Transaction{
		tbl:     &t,
		meta:    meta,
		branch:  branch,
		reqs:    []Requirement{},
		initErr: nil,
	}, nil
}

func (t *Table) Refresh(ctx context.Context) error {
	fresh, err := t.cat.LoadTable(ctx, slices.Clone(t.identifier))
	if err != nil {
		return err
	}
	if t.metadata != nil && fresh.metadata != nil {
		if err := validateTableUUID(t.identifier, t.metadata.TableUUID(), fresh.metadata.TableUUID()); err != nil {
			return err
		}
	}

	t.metadata = fresh.metadata
	t.fsF = fresh.fsF
	t.metadataLocation = fresh.metadataLocation
	t.manifestCache = newSnapshotManifestCacheForMetadata(fresh.metadata)
	t.planner = fresh.planner
	t.scanPlanningIOProps = maps.Clone(fresh.scanPlanningIOProps)
	// Only inherit the catalog-derived reporter when the caller hasn't set one
	// of their own. Refresh runs inside commit retry loops, so unconditionally
	// copying fresh.reporter would silently revert a WithMetricsReporter-injected
	// reporter to the catalog default mid-operation. reporterSet distinguishes an
	// explicit reporter — including an explicit NopReporter opt-out — from the
	// construction-time default.
	if !t.reporterSet {
		t.reporter = fresh.reporter
	}

	return nil
}

func validateTableUUID(identifier Identifier, expected, actual uuid.UUID) error {
	if expected == uuid.Nil || actual == uuid.Nil || expected == actual {
		return nil
	}

	return fmt.Errorf("%w: table %s UUID changed during refresh or commit: expected %s, got %s; load a new table handle",
		ErrInvalidMetadata, strings.Join(identifier, "."), expected, actual)
}

// AppendTable is a shortcut for NewTransaction().AppendTable() and then committing the transaction
func (t Table) AppendTable(ctx context.Context, tbl arrow.Table, batchSize int64, snapshotProps iceberg.Properties) (*Table, error) {
	txn := t.NewTransaction()
	if err := txn.AppendTable(ctx, tbl, batchSize, snapshotProps); err != nil {
		return nil, err
	}

	return txn.Commit(ctx)
}

// Append is a shortcut for NewTransaction().Append() and then committing the transaction
func (t Table) Append(ctx context.Context, rdr array.RecordReader, snapshotProps iceberg.Properties) (*Table, error) {
	txn := t.NewTransaction()
	if err := txn.Append(ctx, rdr, snapshotProps); err != nil {
		return nil, err
	}

	return txn.Commit(ctx)
}

// OverwriteTable is a shortcut for NewTransaction().OverwriteTable() and then committing the transaction.
//
// An optional filter (see WithOverwriteFilter) determines which existing data to delete or rewrite:
//   - If filter is nil or AlwaysTrue, all existing data files are deleted and replaced with new data.
//   - If a filter is provided, it acts as a row-level predicate on existing data:
//   - Files where all rows match the filter (strict match) are completely deleted
//   - Files where some rows match and others don't (partial match) are rewritten to keep only non-matching rows
//   - Files where no rows match the filter are kept unchanged
//
// The filter uses both inclusive and strict metrics evaluators on file statistics to classify files:
//   - Inclusive evaluator identifies candidate files that may contain matching rows
//   - Strict evaluator determines if all rows in a file must match the filter
//   - Files that pass inclusive but not strict evaluation are rewritten with filtered data
//
// New data from the provided table is written to the table regardless of the filter.
//
// The batchSize parameter refers to the batch size for reading the input data, not the batch size for writes.
// The concurrency parameter controls the level of parallelism for manifest processing and file rewriting and
// can be overridden using the WithOverwriteConcurrency option. Defaults to runtime.GOMAXPROCS(0).
func (t Table) OverwriteTable(ctx context.Context, tbl arrow.Table, batchSize int64, snapshotProps iceberg.Properties, opts ...OverwriteOption) (*Table, error) {
	txn := t.NewTransaction()
	if err := txn.OverwriteTable(ctx, tbl, batchSize, snapshotProps, opts...); err != nil {
		return nil, err
	}

	return txn.Commit(ctx)
}

// Overwrite is a shortcut for NewTransaction().Overwrite() and then committing the transaction.
//
// An optional filter (see WithOverwriteFilter) determines which existing data to delete or rewrite:
//   - If filter is nil or AlwaysTrue, all existing data files are deleted and replaced with new data.
//   - If a filter is provided, it acts as a row-level predicate on existing data:
//   - Files where all rows match the filter (strict match) are completely deleted
//   - Files where some rows match and others don't (partial match) are rewritten to keep only non-matching rows
//   - Files where no rows match the filter are kept unchanged
//
// The filter uses both inclusive and strict metrics evaluators on file statistics to classify files:
//   - Inclusive evaluator identifies candidate files that may contain matching rows
//   - Strict evaluator determines if all rows in a file must match the filter
//   - Files that pass inclusive but not strict evaluation are rewritten with filtered data
//
// New data from the provided RecordReader is written to the table regardless of the filter.
//
// The concurrency parameter controls the level of parallelism for manifest processing and file rewriting and
// can be overridden using the WithOverwriteConcurrency option. Defaults to runtime.GOMAXPROCS(0).
func (t Table) Overwrite(ctx context.Context, rdr array.RecordReader, snapshotProps iceberg.Properties, opts ...OverwriteOption) (*Table, error) {
	txn := t.NewTransaction()
	if err := txn.Overwrite(ctx, rdr, snapshotProps, opts...); err != nil {
		return nil, err
	}

	return txn.Commit(ctx)
}

// Delete is a shortcut for NewTransaction().Delete() and then committing the transaction.
//
// The provided filter acts as a row-level predicate on existing data:
//   - Files where all rows match the filter (strict match) are completely deleted
//   - Files where some rows match and others don't (partial match) are rewritten to keep only non-matching rows
//   - Files where no rows match the filter are kept unchanged
//
// The filter uses both inclusive and strict metrics evaluators on file statistics to classify files:
//   - Inclusive evaluator identifies candidate files that may contain matching rows
//   - Strict evaluator determines if all rows in a file must match the filter
//   - Files that pass inclusive but not strict evaluation are rewritten with filtered data
//
// The concurrency parameter controls the level of parallelism for manifest processing and file rewriting and
// can be overridden using the WithOverwriteConcurrency option. Defaults to runtime.GOMAXPROCS(0).
func (t Table) Delete(ctx context.Context, filter iceberg.BooleanExpression, snapshotProps iceberg.Properties, opts ...DeleteOption) (*Table, error) {
	txn := t.NewTransaction()
	if err := txn.Delete(ctx, filter, snapshotProps, opts...); err != nil {
		return nil, err
	}

	return txn.Commit(ctx)
}

func (t Table) manifestSet(ctx context.Context, snapshot Snapshot) (snapshotManifestSet, error) {
	return t.manifestSetWithFSF(ctx, snapshot, t.fsF)
}

func (t Table) manifestSetWithFSF(
	ctx context.Context,
	snapshot Snapshot,
	fsF FSysF,
) (snapshotManifestSet, error) {
	return t.manifestCache.get(ctx, snapshot, func(loadCtx context.Context) (snapshotManifestSet, error) {
		return readSnapshotManifestSet(loadCtx, snapshot, fsF)
	})
}

// AllManifests returns the manifests referenced by the table's snapshots.
// Tables without snapshots return an empty sequence without resolving file IO.
func (t Table) AllManifests(ctx context.Context) iter.Seq2[iceberg.ManifestFile, error] {
	type list = tblutils.Enumerated[[]iceberg.ManifestFile]
	snapshots := t.metadata.Snapshots()
	n := len(snapshots)
	workCtx, cancel := context.WithCancel(ctx)
	jobs := make(chan int)
	// This buffer lets workers finish after an early consumer stop. The result
	// channel created below still retains all snapshot results, so this is not a
	// memory bound; all remote reads are bounded by allManifestsMaxWorkers.
	ch := make(chan list, allManifestsWorkerCount(n))
	workers := allManifestsWorkerCount(n)
	g, groupCtx := errgroup.WithContext(workCtx)
	manifestFS := sharedSnapshotManifestFSF(t.fsF)

	for range workers {
		g.Go(func() error {
			for {
				select {
				case <-groupCtx.Done():
					return groupCtx.Err()
				case i, ok := <-jobs:
					if !ok {
						return nil
					}

					manifestSet, err := t.manifestSetWithFSF(groupCtx, snapshots[i], manifestFS)
					if err != nil {
						return err
					}
					manifests := manifestSet.allManifests()

					select {
					case ch <- list{Index: i, Value: manifests, Last: i == n-1}:
					case <-groupCtx.Done():
						return groupCtx.Err()
					}
				}
			}
		})
	}

	go func() {
		defer close(jobs)
		for i := range snapshots {
			select {
			case jobs <- i:
			case <-groupCtx.Done():
				return
			}
		}
	}()

	errch := make(chan error, 1)
	go func() {
		defer close(errch)
		defer close(ch)
		if err := g.Wait(); err != nil {
			errch <- err
		}
	}()

	results := tblutils.MakeSequencedChan(uint(n), ch,
		func(left, right *list) bool {
			switch {
			case left.Index < 0:
				return true
			case right.Index < 0:
				return false
			default:
				return left.Index < right.Index
			}
		}, func(prev, next *list) bool {
			if prev.Index < 0 {
				return next.Index == 0
			}

			return next.Index == prev.Index+1
		}, list{Index: -1})

	return func(yield func(iceberg.ManifestFile, error) bool) {
		defer cancel()
		defer func() {
			// drain channels if we exited early
			go func() {
				for range results {
				}
				if errch != nil {
					for range errch {
					}
				}
			}()
		}()

		yieldAllManifests(results, errch, yield)
	}
}

func yieldAllManifests(
	results <-chan tblutils.Enumerated[[]iceberg.ManifestFile],
	errch <-chan error,
	yield func(iceberg.ManifestFile, error) bool,
) {
	for results != nil || errch != nil {
		select {
		case err, ok := <-errch:
			if !ok {
				errch = nil

				continue
			}
			if err != nil {
				yield(nil, err)

				return
			}
		case next, ok := <-results:
			if !ok {
				results = nil

				continue
			}
			for _, mf := range next.Value {
				if !yield(mf, nil) {
					return
				}
			}

			if next.Last {
				return
			}
		}
	}
}

// conflictValidatorFunc runs a single producer's client-side conflict
// check against a pre-built conflictContext. Validators return a wrapped
// ErrCommit* sentinel on retryable conflict, ErrCommitDiverged on
// terminal divergence, or nil on success.
type conflictValidatorFunc func(cc *conflictContext) error

// commitOpts controls optional behavior of doCommit beyond the core
// updates/requirements loop. All fields are zero-valued by default and
// callers opt in via the commitOption functional options passed to
// doCommit.
type commitOpts struct {
	// branch is the ref the commit targets. When empty, pre-flight
	// conflict validation is skipped because no conflictContext can
	// be built. Direct doCommit callers (unit tests, low-level utils)
	// may leave this empty; Transaction.Commit always sets it.
	branch string

	// validators run before cat.CommitTable on every attempt of the
	// retry loop. On attempt 0 the writer's metadata and the catalog
	// state coincide, so the conflict context has no concurrent
	// snapshots and validators short-circuit; on retries they run
	// against the freshly refreshed catalog state (refresh-and-replay).
	validators []conflictValidatorFunc

	// noReplay makes a CAS conflict terminal: doCommit returns the
	// ErrCommitFailed error instead of entering refresh-and-replay.
	// Set for commits carrying delete-file removals; see the flag site
	// in snapshotProducer.commitManifests for the rationale.
	noReplay bool

	// pinnedRefs names branches with an explicit requirement from
	// Transaction.AssertRefSnapshotID, whose assertions must not be
	// rewritten to the fresh branch head between retries (see
	// Transaction.pinnedRefs).
	pinnedRefs map[string]struct{}
}

type commitOption func(*commitOpts)

// withCommitBranch sets the target branch for the pre-flight
// conflict-validation walk. An empty branch is treated as the main
// branch — Transaction.branch is empty when the caller never picked
// one explicitly, and the implicit default is main.
func withCommitBranch(branch string) commitOption {
	if branch == "" {
		branch = MainBranch
	}

	return func(o *commitOpts) { o.branch = branch }
}

func withCommitValidators(vs ...conflictValidatorFunc) commitOption {
	return func(o *commitOpts) { o.validators = append(o.validators, vs...) }
}

func withCommitNoReplay(noReplay bool) commitOption {
	return func(o *commitOpts) { o.noReplay = noReplay }
}

func withCommitPinnedRefs(refs map[string]struct{}) commitOption {
	return func(o *commitOpts) { o.pinnedRefs = refs }
}

func (t Table) doCommit(ctx context.Context, updates []Update, reqs []Requirement, opts ...commitOption) (*Table, error) {
	var co commitOpts
	for _, apply := range opts {
		apply(&co)
	}

	cfg, err := readRetryConfig(t.metadata.Properties())
	if err != nil {
		return nil, err
	}

	// Bound total retry time with a derived context so both the wait loop
	// and the CommitTable call itself respect the deadline uniformly.
	retryCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.totalTimeoutMs)*time.Millisecond)
	defer cancel()

	fs, err := t.fsF(ctx)
	if err != nil {
		return nil, err
	}

	// Every real commit-path FS implements WriteFileIO. Failing here is
	// preferable to silently skipping the manifest-list rebuild inside the
	// retry loop — a skip reintroduces the original stale-parent data loss.
	wfs, err := requireWriteFileIO(fs)
	if err != nil {
		return nil, fmt.Errorf("%w: manifest list rebuild requires write access", err)
	}

	var (
		newMeta           Metadata
		newLoc            string
		timer             *time.Timer
		orphanedManifests []string // manifest-list files orphaned by rebuilds
		commitDuration    time.Duration
	)

	// attemptsUsed initializes to 1: the emit block below is reached only via the
	// success break, which always runs at least one attempt. Deriving it from a
	// 0-based counter risks emitting 0 if a second success exit is ever added.
	var attemptsUsed int64 = 1

	// cleanupOrphans controls whether the defer below removes orphaned manifest-list
	// files on exit. It defaults to true (clean on all safe exits) and is set to
	// false only for the one unsafe case: a non-ErrCommitFailed error from
	// CommitTable, where the catalog may have silently accepted the commit and one
	// of the "orphaned" files may actually be the live snapshot.
	cleanupOrphans := true
	defer func() {
		if !cleanupOrphans || len(orphanedManifests) == 0 {
			return
		}
		for _, path := range orphanedManifests {
			if removeErr := wfs.Remove(path); removeErr != nil {
				log.Printf("Warning: failed to delete orphaned manifest list %s: %v", path, removeErr)
			}
		}
	}()

	// current tracks the catalog state between retries. On attempt 0 it
	// equals t.metadata (so the conflict context's concurrent-snapshot
	// walk is empty and validators short-circuit). On subsequent
	// attempts it is the freshly-loaded post-conflict state.
	current := t.metadata

	// numRetries counts retries; total attempts = 1 initial + numRetries.
	totalAttempts := cfg.numRetries + 1

	// commitStart brackets the commit loop itself (not FS resolution above or
	// metadata cleanup below) so TotalDuration times only the CommitTable
	// submission loop, matching Java's CommitReport.
	commitStart := time.Now()

	for attempt := range totalAttempts {
		if attempt != 0 {
			wait := backoffDuration(attempt-1, cfg.minWaitMs, cfg.maxWaitMs)
			if timer == nil {
				timer = time.NewTimer(wait)
			} else {
				timer.Reset(wait)
			}
			select {
			case <-retryCtx.Done():
				timer.Stop()

				return nil, context.Cause(retryCtx)
			case <-timer.C:
			}

			// Refresh-and-replay: reload the catalog's current state,
			// run the producers' validators against the fresh
			// (base=t.metadata, current=fresh) conflict context, and
			// rewrite any AssertRefSnapshotID requirements to target
			// the new branch head so re-submission is not rejected
			// just because a peer advanced the head with a
			// non-conflicting commit.
			fresh, refreshErr := t.cat.LoadTable(retryCtx, slices.Clone(t.identifier))
			if refreshErr != nil {
				return nil, fmt.Errorf("refresh table for retry: %w", refreshErr)
			}
			if t.metadata != nil && fresh.metadata != nil {
				if err := validateTableUUID(t.identifier, t.metadata.TableUUID(), fresh.metadata.TableUUID()); err != nil {
					return nil, err
				}
			}
			current = fresh.metadata
			reqs = rewriteRefSnapshotRequirements(reqs, co.branch, current, co.pinnedRefs)

			// A pinned assertion the fresh catalog state violates can
			// never succeed — fail now instead of burning the remaining
			// retries on it.
			if err := validatePinnedRefRequirements(reqs, co.pinnedRefs, current); err != nil {
				return nil, fmt.Errorf("%w: explicit ref requirement failed: %w", ErrCommitFailed, err)
			}
			if err := validateBranchRequirement(reqs, co.branch, current); err != nil {
				return nil, err
			}

			// Rebuild snapshot manifest lists to inherit all files committed
			// by concurrent writers since the snapshot was originally built.
			// Without this, the new snapshot's manifest list would only
			// contain its own files and callers scanning the current snapshot
			// would miss every concurrent writer's data.
			rebuiltUpdates, orphaned, rebuildErr := rebuildSnapshotUpdates(retryCtx, updates, current, co.branch, wfs, int(attempt))
			if rebuildErr != nil {
				return nil, fmt.Errorf("rebuild manifest list for retry attempt %d: %w", attempt, rebuildErr)
			}
			orphanedManifests = append(orphanedManifests, orphaned...)
			updates = rebuiltUpdates
		}

		if err := validateBranchRequirement(reqs, co.branch, current); err != nil {
			return nil, err
		}

		// Pre-flight client-side conflict validation. Producers can
		// reject commits whose semantics are violated by concurrent
		// peers (partition-filter overlap, referenced-file removal)
		// even when the catalog-side AssertRefSnapshotID would accept
		// them. On attempt 0 base == current → no concurrent
		// snapshots → validators short-circuit. Real divergence
		// detection fires on attempts > 0 once `current` is the
		// post-conflict state.
		//
		// Skipped when the branch does not exist on `current` — that
		// always means "the committer is creating this branch" (e.g.
		// first commit on a fresh table). There are no concurrent
		// snapshots on a branch that does not yet exist, and
		// newConflictContext would otherwise return ErrCommitDiverged.
		if co.branch != "" && len(co.validators) > 0 && current.SnapshotByName(co.branch) != nil {
			// caseSensitive is hardcoded to true here: transaction-
			// level case-sensitivity is not yet threaded through the
			// Commit path, and true is the scan default throughout the
			// codebase.
			cc, ccErr := newConflictContext(t.metadata, current, co.branch, fs, true)
			if ccErr != nil {
				// ErrCommitDiverged — terminal, do not retry. The
				// sentinel deliberately does not wrap ErrCommitFailed.
				return nil, ccErr
			}
			for _, v := range co.validators {
				if vErr := v(cc); vErr != nil {
					return nil, vErr
				}
			}
		}

		if retryCtx.Err() != nil {
			return nil, context.Cause(retryCtx)
		}

		newMeta, newLoc, err = t.cat.CommitTable(retryCtx, slices.Clone(t.identifier), reqs, updates)
		if err == nil {
			attemptsUsed = int64(attempt) + 1
			// Capture elapsed time at the commit boundary, before orphan
			// cleanup and deleteOldMetadata below (both can do I/O).
			commitDuration = time.Since(commitStart)

			break
		}

		// Only retry on retryable commit conflicts. Unknown-state errors
		// (5xx, gateway timeouts) must NOT be retried because the commit
		// may have actually succeeded — retrying could duplicate work.
		// Suppress orphan cleanup for the same reason: one of the orphaned
		// manifest-list files may actually be the snapshot the catalog accepted.
		if !errors.Is(err, ErrCommitFailed) {
			cleanupOrphans = false

			return nil, err
		}

		// Non-replayable commits fail on the first CAS conflict instead
		// of replaying (see commitOpts.noReplay). The annotation
		// distinguishes this abort from an exhausted retry budget while
		// preserving errors.Is(err, ErrCommitFailed).
		if co.noReplay {
			return nil, fmt.Errorf("%w (commit carries snapshot-relative delete-file removals and cannot be replayed; reload the table and rebuild the removals)", err)
		}
	}

	// Inner data manifests written by superseded retry attempts (a rewrite
	// re-merges everything on each retry) are orphaned objects. On a safe
	// exhausted-ErrCommitFailed failure (err != nil here) nothing committed, so
	// the accumulator also folds in the final attempt's manifests. On success
	// the committed snapshot references those, so they are excluded. The defer
	// skips cleanup only on the unsafe non-ErrCommitFailed path, which returned
	// above with cleanupOrphans = false.
	for _, u := range updates {
		su, ok := u.(*addSnapshotUpdate)
		if !ok || su.supersededSource == nil {
			continue
		}
		orphanedManifests = append(orphanedManifests, su.supersededSource.supersededManifests(err == nil)...)
	}

	if err != nil {
		return nil, err
	}

	deleteOldMetadata(fs, t.metadata, newMeta)

	// Emit a commit report on success. Prefer the just-committed branch head
	// over the table's current snapshot so commits to a non-default branch
	// report the snapshot they actually created.
	//
	// Mirrors the scan path: building the report is skipped for a no-op
	// reporter (the opt-in default), since a nop discards it and assembling one
	// would be pure overhead. A metadata-only commit produces no snapshot and
	// must be skipped too — its branch head is unchanged, so reporting it would
	// attribute a prior snapshot's metrics to this commit.
	if rep := t.MetricsReporter(); !metrics.IsNop(rep) && commitAddedSnapshot(updates) {
		committed := newMeta.CurrentSnapshot()
		if co.branch != "" {
			// A nil lookup means the branch head could not be resolved (e.g. a
			// fresh non-default branch); attributing CurrentSnapshot() would
			// carry the wrong snapshot, so skip emission entirely.
			committed = newMeta.SnapshotByName(co.branch)
		}
		if committed != nil {
			safeReport(ctx, rep,
				buildCommitReport(strings.Join(t.identifier, "."), committed, attemptsUsed, commitDuration))
		}
	}

	return New(
		t.identifier,
		newMeta,
		newLoc,
		t.fsF,
		t.cat,
		withReporterState(t.reporter, t.reporterSet),
		WithScanPlanningIOProperties(t.scanPlanningIOProps),
	), nil
}

// rewriteRefSnapshotRequirements returns a copy of reqs with every
// AssertRefSnapshotID targeting `branch` rewritten to point at the
// branch head on `fresh`. Other requirements pass through untouched.
//
// Producers register AssertRefSnapshotID at commit-build time with the
// committer's base snapshot id. After a peer advances the branch head
// with a non-conflicting commit, that assertion no longer matches the
// catalog. Without rewriting the retry would burn the budget on the
// same stale requirement; with it, validators get to decide if the
// commit is still safe to replay against the new head.
//
// Java's SnapshotProducer rewrites the same way between retries. If
// the branch is empty or the new head cannot be resolved (branch
// deleted underneath us), reqs is returned unchanged — newConflict-
// Context will surface the divergence on the next pre-flight pass.
//
// Assertions on branches in pinned were registered explicitly by the
// committer (Transaction.AssertRefSnapshotID) for compare-and-swap
// semantics and are never rewritten: a branch that has changed must
// fail the commit, not be replayed against the new head.
func rewriteRefSnapshotRequirements(reqs []Requirement, branch string, fresh Metadata, pinned map[string]struct{}) []Requirement {
	if branch == "" || fresh == nil {
		return reqs
	}
	head := fresh.SnapshotByName(branch)
	if head == nil {
		return reqs
	}

	out := make([]Requirement, len(reqs))
	for i, r := range reqs {
		if a, ok := r.(*assertRefSnapshotID); ok && a.Ref == branch {
			if _, isPinned := pinned[a.Ref]; !isPinned {
				newID := head.SnapshotID
				if a.requireBranch {
					out[i] = assertBranchRefSnapshotID(branch, &newID)
				} else {
					out[i] = AssertRefSnapshotID(branch, &newID)
				}

				continue
			}
		}
		out[i] = r
	}

	return out
}

// validatePinnedRefRequirements validates every assert-ref-snapshot-id
// requirement on a pinned branch against the freshly refreshed catalog
// metadata. A failure means the branch has changed from the required
// snapshot: the assertion is never rewritten, so it can never hold and
// the commit must fail instead of retrying.
func validatePinnedRefRequirements(reqs []Requirement, pinned map[string]struct{}, fresh Metadata) error {
	if len(pinned) == 0 {
		return nil
	}
	for _, r := range reqs {
		a, ok := r.(*assertRefSnapshotID)
		if !ok {
			continue
		}
		if _, isPinned := pinned[a.Ref]; !isPinned {
			continue
		}
		if err := a.Validate(fresh); err != nil {
			return err
		}
	}

	return nil
}

func validateBranchRequirement(reqs []Requirement, branch string, meta Metadata) error {
	for _, req := range reqs {
		if ref, ok := req.(*assertRefSnapshotID); ok && ref.requireBranch && ref.Ref == branch {
			return ref.Validate(meta)
		}
	}

	return nil
}

// latestSnapshotForBranch returns the head a write targeting branch is parented
// on: a branch that does not exist yet resolves to nil here and falls back to
// main's head so the first write to a new branch forks from main. It is the
// Metadata-side twin of currentSnapshotForRef, which the retry path cannot use
// because it has no builder; the two must resolve a ref identically, or a
// retried snapshot is reparented somewhere its first attempt never pointed.
func latestSnapshotForBranch(meta Metadata, branch string) *Snapshot {
	if branch == "" || branch == MainBranch {
		return meta.CurrentSnapshot()
	}

	if s := meta.SnapshotByName(branch); s != nil {
		return s
	}

	return meta.CurrentSnapshot()
}

// stagedSnapshotStillValid reports whether a snapshot built on an earlier
// attempt would still be accepted by AddSnapshot against fresh. Both values it
// checks are table-wide, so a peer committing on ANOTHER branch invalidates them
// without moving this branch's head — the case the caller's parent check misses,
// and one AddSnapshot rejects terminally rather than retryably.
func stagedSnapshotStillValid(snap *Snapshot, fresh Metadata) bool {
	if snap == nil || fresh == nil {
		return true
	}

	if fresh.Version() >= 2 && snap.SequenceNumber <= fresh.LastSequenceNumber() {
		return false
	}

	if fresh.Version() >= 3 && (snap.FirstRowID == nil || *snap.FirstRowID < fresh.NextRowID()) {
		return false
	}

	return true
}

// rebuildSnapshotUpdates returns a new slice of updates where any
// addSnapshotUpdate that carries a rebuildManifestList closure has its
// snapshot regenerated to inherit all data files committed to the branch
// since the original snapshot was built. Updates without a rebuild closure
// pass through unchanged.
//
// It also returns the manifest-list file paths that were superseded by
// the rebuild (i.e., the paths from the input updates that were replaced).
// These become orphaned objects in object storage and should be removed
// by the caller after a successful commit.
//
// This is the manifest-layer "refresh-and-replay" step: the data files
// (already written to object storage) are reused as-is; only the manifest
// list is rewritten to include the fresh parent's manifests so that the
// rebuilt snapshot contains every committed file.
func rebuildSnapshotUpdates(ctx context.Context, updates []Update, freshMeta Metadata, branch string, fs icebergio.WriteFileIO, attempt int) (rebuilt []Update, orphanedPaths []string, err error) {
	// Must mirror createSnapshotProducer's attempt-0 fallback, or a retried new
	// branch rebuilds with a nil parent and silently drops main's data.
	var freshHead *Snapshot
	if freshMeta != nil {
		freshHead = latestSnapshotForBranch(freshMeta, branch)
	}

	result := make([]Update, len(updates))
	copy(result, updates)

	// A single transaction can stage several snapshots on the same branch (an
	// append followed by a delete, say). They were chained at build time so
	// snapshot[n].parent == snapshot[n-1]; on retry they must be replayed as a
	// chain, each rebuilt snapshot becoming the parent of the next. Feeding
	// every staged snapshot the same fresh branch head would turn the chain
	// into siblings — the branch ref advances only to the last one, silently
	// dropping the earlier snapshots' data. chainRebuilt forces every snapshot
	// after the first rebuild to rebuild too: its recorded parent is an earlier
	// staged snapshot that has itself just been reparented onto the fresh head.
	chainRebuilt := false
	for i, u := range result {
		su, ok := u.(*addSnapshotUpdate)
		if !ok || su.rebuildManifestList == nil {
			continue
		}

		// Skip only when nothing upstream changed: the running head still
		// matches this snapshot's recorded parent, no earlier snapshot in this
		// chain was rebuilt, and the table-wide values the rebuild recomputes
		// are still acceptable. Saves an unnecessary S3 write.
		if !chainRebuilt && freshHead != nil && su.Snapshot.ParentSnapshotID != nil &&
			*su.Snapshot.ParentSnapshotID == freshHead.SnapshotID &&
			stagedSnapshotStillValid(su.Snapshot, freshMeta) {
			// Already parented on the running head; it becomes the parent of
			// the next staged snapshot in the chain.
			freshHead = su.Snapshot

			continue
		}

		oldManifestList := su.Snapshot.ManifestList

		newSnap, rebuildErr := su.rebuildManifestList(ctx, freshMeta, freshHead, fs, attempt)
		if rebuildErr != nil {
			return nil, nil, rebuildErr
		}

		result[i] = &addSnapshotUpdate{
			baseUpdate:          su.baseUpdate,
			Snapshot:            newSnap,
			ownManifests:        su.ownManifests,
			rebuildManifestList: su.rebuildManifestList,
			supersededSource:    su.supersededSource,
		}

		// The old manifest list is now an orphaned object in object storage.
		orphanedPaths = append(orphanedPaths, oldManifestList)

		// Chain the next staged snapshot onto the one just rebuilt.
		freshHead = newSnap
		chainRebuilt = true
	}

	return result, orphanedPaths, nil
}

const (
	maxRetryDurationMs = uint64(math.MaxInt64 / int64(time.Millisecond))
	maxRetryCount      = uint64(math.MaxUint32)
)

type retryConfig struct {
	numRetries     uint
	minWaitMs      uint64
	maxWaitMs      uint64
	totalTimeoutMs uint64
}

func readRetryConfig(props iceberg.Properties) (retryConfig, error) {
	numRetries := props.GetUInt64(CommitNumRetriesKey, CommitNumRetriesDefault)
	if numRetries >= uint64(^uint(0)) || numRetries > maxRetryCount {
		return retryConfig{}, fmt.Errorf(
			"invalid retry property %q=%d: retry count exceeds the maximum of %d",
			CommitNumRetriesKey, numRetries, maxRetryCount)
	}

	cfg := retryConfig{
		numRetries:     uint(numRetries),
		minWaitMs:      props.GetUInt64(CommitMinRetryWaitMsKey, CommitMinRetryWaitMsDefault),
		maxWaitMs:      props.GetUInt64(CommitMaxRetryWaitMsKey, CommitMaxRetryWaitMsDefault),
		totalTimeoutMs: props.GetUInt64(CommitTotalRetryTimeoutMsKey, CommitTotalRetryTimeoutMsDefault),
	}

	if cfg.minWaitMs == 0 {
		cfg.minWaitMs = CommitMinRetryWaitMsDefault
	}
	if cfg.maxWaitMs == 0 {
		cfg.maxWaitMs = CommitMaxRetryWaitMsDefault
	}
	if cfg.totalTimeoutMs == 0 {
		cfg.totalTimeoutMs = CommitTotalRetryTimeoutMsDefault
	}

	for _, property := range []struct {
		key   string
		value uint64
	}{
		{CommitMinRetryWaitMsKey, cfg.minWaitMs},
		{CommitMaxRetryWaitMsKey, cfg.maxWaitMs},
		{CommitTotalRetryTimeoutMsKey, cfg.totalTimeoutMs},
	} {
		key, value := property.key, property.value
		if value > maxRetryDurationMs {
			return retryConfig{}, fmt.Errorf(
				"invalid retry property %q=%d: exceeds maximum duration of %d milliseconds",
				key, value, maxRetryDurationMs)
		}
	}

	if cfg.minWaitMs > cfg.maxWaitMs {
		return retryConfig{}, fmt.Errorf(
			"invalid retry properties %q=%d and %q=%d: minimum wait exceeds maximum wait",
			CommitMinRetryWaitMsKey, cfg.minWaitMs,
			CommitMaxRetryWaitMsKey, cfg.maxWaitMs)
	}

	return cfg, nil
}

// backoffDuration computes wait time for the given 0-based retry attempt
// using exponential backoff (minMs << attempt) clamped to maxMs, with
// jitter in [minMs, ceiling] to avoid retry stampedes while keeping a
// non-zero floor between attempts. Java Iceberg uses a deterministic
// exponential backoff here; we add jitter to reduce stampede risk on
// concurrent Go writers. Backoff is client-local, so this does not
// affect cross-client interop.
//
// Inputs from readRetryConfig are validated. The defensive bounds below also
// keep direct callers from reaching an invalid random bound or duration.
func backoffDuration(attempt uint, minMs, maxMs uint64) time.Duration {
	if minMs == 0 {
		minMs = CommitMinRetryWaitMsDefault
	}
	if maxMs == 0 {
		maxMs = CommitMaxRetryWaitMsDefault
	}
	if minMs > maxRetryDurationMs {
		minMs = maxRetryDurationMs
	}
	if maxMs > maxRetryDurationMs {
		maxMs = maxRetryDurationMs
	}
	if minMs > maxMs {
		minMs = maxMs
	}
	// Cap the shift count so the exponential calculation stays within the
	// bounded duration range.
	if attempt > 62 {
		attempt = 62
	}

	var ceiling uint64
	if minMs > maxRetryDurationMs>>attempt {
		ceiling = maxMs
	} else {
		ceiling = min(minMs<<attempt, maxMs)
	}

	// Jitter in [minMs, ceiling]: keeps a non-zero floor so concurrent
	// writers don't all sample 0 and retry in lockstep.
	// Both conversions are safe because minMs and ceiling-minMs+1 are bounded
	// by maxRetryDurationMs, which fits in int64.
	//nolint:gosec // non-security randomness; bounded int64 conversions are safe
	wait := int64(minMs) + rand.Int64N(int64(ceiling-minMs+1))

	return time.Duration(wait) * time.Millisecond
}

// SnapshotAsOf finds the snapshot that was current as of or right before the given timestamp.
func (t Table) SnapshotAsOf(timestampMs int64, inclusive bool) *Snapshot {
	entry, ok := snapshotLogEntryAsOf(t.metadata.SnapshotLogs(), timestampMs, inclusive)
	if ok {
		return t.metadata.SnapshotByID(entry.SnapshotID)
	}

	return nil
}

func getFiles(it iter.Seq[MetadataLogEntry]) iter.Seq[string] {
	return func(yield func(string) bool) {
		next, stop := iter.Pull(it)
		defer stop()
		for {
			entry, ok := next()
			if !ok {
				return
			}
			if !yield(entry.MetadataFile) {
				return
			}
		}
	}
}

func deleteOldMetadata(fs icebergio.IO, baseMeta, newMeta Metadata) {
	deleteAfterCommit := newMeta.Properties().GetBool(MetadataDeleteAfterCommitEnabledKey,
		MetadataDeleteAfterCommitEnabledDefault)

	if deleteAfterCommit {
		removedPrevious := slices.Collect(getFiles(baseMeta.PreviousFiles()))
		currentMetadata := slices.Collect(getFiles(newMeta.PreviousFiles()))
		toRemove := internal.Difference(removedPrevious, currentMetadata)

		for _, file := range toRemove {
			if err := fs.Remove(file); err != nil {
				// Log the error instead of raising it when deleting old metadata files, as an external entity like a compactor may have already deleted them
				log.Printf("Warning: Failed to delete old metadata file: %s error: %v", file, err)
			}
		}
	}
}

type ScanOption func(*Scan)

func noopOption(*Scan) {}

func WithSelectedFields(fields ...string) ScanOption {
	if len(fields) == 0 || slices.Contains(fields, "*") {
		return noopOption
	}

	return func(scan *Scan) {
		scan.selectedFields = slices.Clone(fields)
	}
}

func WithRowFilter(e iceberg.BooleanExpression) ScanOption {
	if e == nil || e.Equals(iceberg.AlwaysTrue{}) {
		return noopOption
	}

	return func(scan *Scan) {
		scan.rowFilter = e
	}
}

func WithSnapshotID(n int64) ScanOption {
	if n == 0 {
		return noopOption
	}

	return func(scan *Scan) {
		if scan.asOfTimestamp != nil {
			scan.selectorErr = fmt.Errorf("%w: cannot select snapshot ID %d when as-of timestamp %d is already selected",
				iceberg.ErrInvalidArgument, n, *scan.asOfTimestamp)

			return
		}
		scan.snapshotID = &n
		scan.asOfTimestamp = nil
		useSnapshotSchema := true
		scan.useSnapshotSchema = &useSnapshotSchema
	}
}

func WithSnapshotAsOf(timeStampMs int64) ScanOption {
	return func(scan *Scan) {
		if scan.snapshotID != nil {
			scan.selectorErr = fmt.Errorf("%w: cannot select as-of timestamp %d when snapshot ID %d is already selected",
				iceberg.ErrInvalidArgument, timeStampMs, *scan.snapshotID)

			return
		}
		scan.asOfTimestamp = &timeStampMs
		scan.snapshotID = nil
		useSnapshotSchema := true
		scan.useSnapshotSchema = &useSnapshotSchema
	}
}

func WithCaseSensitive(b bool) ScanOption {
	return func(scan *Scan) {
		scan.caseSensitive = b
	}
}

func WithLimit(n int64) ScanOption {
	if n < 0 {
		return noopOption
	}

	return func(scan *Scan) {
		scan.limit = n
	}
}

// WithMaxConcurrency sets the maximum concurrency for table scan and plan
// operations. When unset it defaults to runtime.GOMAXPROCS. Above one, a
// scan holds the decoded batches of up to n tasks at once while it returns
// them in task order; [Scan.ReadTasks] states the bound.
func WithMaxConcurrency(n int) ScanOption {
	if n <= 0 {
		return noopOption
	}

	return func(scan *Scan) {
		scan.concurrency = n
	}
}

// WitMaxConcurrency is a deprecated alias for [WithMaxConcurrency], kept for
// backward compatibility with the pre-existing typo'd name.
//
// Deprecated: use [WithMaxConcurrency].
func WitMaxConcurrency(n int) ScanOption {
	return WithMaxConcurrency(n)
}

func WithOptions(opts iceberg.Properties) ScanOption {
	if opts == nil {
		return noopOption
	}

	return func(scan *Scan) {
		scan.options = maps.Clone(opts)
	}
}

// WithReporter overrides the metrics reporter for a single scan, taking
// precedence over the reporter inherited from the table. A nil reporter is
// ignored.
func WithReporter(r metrics.Reporter) ScanOption {
	if r == nil {
		return noopOption
	}

	return func(scan *Scan) {
		scan.reporter = r
	}
}

// WithRowLineage projects the row-lineage metadata columns (_row_id and
// _last_updated_sequence_number) so that row identity and per-row update
// sequence are preserved through rewrites and compactions. Requires a v3
// table — calling Scan.Projection on a v1/v2 table after applying this
// option returns an error.
func WithRowLineage() ScanOption {
	return func(scan *Scan) {
		scan.includeRowLineage = true
	}
}

// WithArrowBatchSize caps the number of rows decoded per Arrow record
// batch when reading data files, overriding the table's
// read.parquet.batch-size property for this scan. Smaller batches bound
// the memory a scan holds per decoded batch, which matters when the
// consumer buffers batches (e.g. a compaction's read+write pipeline).
// The cap is stored on the scan itself rather than in the options map,
// so it applies regardless of ordering relative to [WithOptions]. A
// non-positive value is ignored.
func WithArrowBatchSize(n int) ScanOption {
	if n <= 0 {
		return noopOption
	}

	return func(scan *Scan) {
		scan.arrowBatchSize = n
	}
}

func (t Table) Scan(opts ...ScanOption) *Scan {
	s := &Scan{
		identifier:          slices.Clone(t.identifier),
		metadata:            t.metadata,
		metadataLocation:    t.metadataLocation,
		ioF:                 t.fsF,
		manifestCache:       t.manifestCache,
		planner:             t.planner,
		scanPlanningIOProps: maps.Clone(t.scanPlanningIOProps),
		// TODO(#1178 Phase 6): resolve scan-planning-mode table properties here.
		planningMode:   ScanPlanningLocal,
		rowFilter:      iceberg.AlwaysTrue{},
		selectedFields: []string{"*"},
		caseSensitive:  true,
		limit:          ScanNoLimit,
		concurrency:    runtime.GOMAXPROCS(0),
		reporter:       t.MetricsReporter(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Option configures a [Table] at construction. Options are applied in order
// after the core fields are set.
type Option func(*Table)

// noopTableOption is the shared no-op [Option], returned when an option has
// nothing to apply (e.g. WithMetricsReporter(nil)). It mirrors noopOption on
// the ScanOption side.
func noopTableOption(*Table) {}

// WithMetricsReporter sets the metrics reporter for the table; scans created
// from the table inherit it. A nil reporter is ignored (the table keeps its
// default no-op reporter).
func WithMetricsReporter(r metrics.Reporter) Option {
	if r == nil {
		return noopTableOption
	}

	return func(t *Table) {
		t.reporter = r
		t.reporterSet = true
	}
}

// WithScanPlanningIOProperties supplies the table-scoped FileIO properties
// that a remote scan planner needs when it builds a plan-scoped FileIO. This
// is intended for catalog implementations whose table-load response carries
// FileIO configuration separately from table metadata properties.
func WithScanPlanningIOProperties(props iceberg.Properties) Option {
	if props == nil {
		return noopTableOption
	}

	return func(t *Table) {
		t.scanPlanningIOProps = maps.Clone(props)
	}
}

// withReporterState copies both the reporter and the reporterSet flag verbatim.
// Unlike WithMetricsReporter it does not force reporterSet true, so a table
// riding the catalog default (reporterSet == false) stays defaulted across a
// New(...) rebuild instead of being frozen to its current reporter. Used by
// doCommit and StagedTable to carry reporter state without breaking the Refresh
// inheritance invariant — WithMetricsReporter cannot be used there because
// MetricsReporter() is never nil, so it would always mark the table caller-set.
func withReporterState(r metrics.Reporter, set bool) Option {
	return func(t *Table) {
		t.reporter = r
		t.reporterSet = set
	}
}

// New constructs a Table. If cat implements ScanPlanner — as rest.Catalog does
// for servers that support remote scan planning — it is wired as the table's
// planner so (*Scan).PlanFiles can delegate to it; catalogs that do not
// implement ScanPlanner leave planner nil and planning stays local. This is the
// concrete Catalog -> Table -> Scan wiring for #1178: the catalog already
// satisfies ScanPlanner, so no catalog accessor is needed.
func New(ident Identifier, meta Metadata, metadataLocation string, fsF FSysF, cat CatalogIO, opts ...Option) *Table {
	// A catalog that supports server-side scan planning implements ScanPlanner.
	var planner ScanPlanner
	if p, ok := cat.(ScanPlanner); ok {
		planner = p
	}

	t := &Table{
		identifier:       slices.Clone(ident),
		metadata:         meta,
		metadataLocation: metadataLocation,
		fsF:              fsF,
		manifestCache:    newSnapshotManifestCacheForMetadata(meta),
		cat:              cat,
		planner:          planner,
		reporter:         metrics.NopReporter{},
	}
	for _, opt := range opts {
		opt(t)
	}

	return t
}

func NewFromLocation(
	ctx context.Context,
	ident Identifier,
	metalocation string,
	fsysF FSysF,
	cat CatalogIO,
	opts ...Option,
) (_ *Table, err error) {
	var meta Metadata

	fsys, err := fsysF(ctx)
	if err != nil {
		return nil, err
	}
	if rf, ok := fsys.(icebergio.ReadFileIO); ok {
		data, err := rf.ReadFile(metalocation)
		if err != nil {
			return nil, err
		}

		if codec := metadataCompressionCodec(metalocation); codec != "" {
			rc, err := newDecompressor(bytes.NewReader(data), codec)
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			data, err = io.ReadAll(rc)
			if err != nil {
				return nil, err
			}
		}

		if meta, err = ParseMetadataBytes(data); err != nil {
			return nil, err
		}
	} else {
		f, err := fsys.Open(metalocation)
		if err != nil {
			return nil, err
		}
		defer internal.CheckedClose(f, &err)

		var r io.Reader = f
		if codec := metadataCompressionCodec(metalocation); codec != "" {
			rc, err := newDecompressor(f, codec)
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			r = rc
		}

		if meta, err = ParseMetadata(r); err != nil {
			return nil, err
		}
	}

	return New(ident, meta, metalocation, fsysF, cat, opts...), nil
}

func metadataCompressionCodec(location string) string {
	switch {
	case strings.HasSuffix(location, ".gz.metadata.json") || strings.HasSuffix(location, "metadata.json.gz"):
		return MetadataCompressionCodecGzip
	case strings.HasSuffix(location, ".zstd.metadata.json") || strings.HasSuffix(location, "metadata.json.zstd"):
		return MetadataCompressionCodecZstd
	default:
		return ""
	}
}

func newDecompressor(r io.Reader, codec string) (io.ReadCloser, error) {
	switch codec {
	case MetadataCompressionCodecGzip:
		return gzip.NewReader(r)
	case MetadataCompressionCodecZstd:
		dec, err := zstd.NewReader(r)
		if err != nil {
			return nil, err
		}

		return dec.IOReadCloser(), nil
	default:
		return nil, fmt.Errorf("unsupported metadata decompression codec: %s", codec)
	}
}
