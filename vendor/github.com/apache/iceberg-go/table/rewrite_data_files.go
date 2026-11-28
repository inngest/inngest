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
	"context"
	"errors"
	"fmt"
	iofs "io/fs"
	"log/slog"
	"maps"

	"github.com/apache/iceberg-go"
	iceberginternal "github.com/apache/iceberg-go/internal"
	iceio "github.com/apache/iceberg-go/io"
)

// RewriteResult summarizes a completed compaction.
type RewriteResult struct {
	// Table is the latest committed table when PartialProgress is enabled.
	// It is nil for an atomic rewrite until the caller commits the transaction.
	Table *Table

	// RewrittenGroups is the number of compaction groups committed.
	RewrittenGroups int

	// AddedDataFiles is the total number of new data files written.
	AddedDataFiles int

	// RemovedDataFiles is the total number of old data files replaced.
	RemovedDataFiles int

	// RemovedPositionDeleteFiles is the count of position delete files
	// removed because their referenced data file was rewritten or
	// because they were passed via
	// [RewriteDataFilesOptions.ExtraDeleteFilesToRemove].
	RemovedPositionDeleteFiles int

	// RemovedEqualityDeleteFiles is the count of equality delete files
	// removed via [RewriteDataFilesOptions.ExtraDeleteFilesToRemove].
	// The caller computes which eq-deletes are dead — typically via
	// [compaction.CollectDeadEqualityDeletesWithSpecs] — and passes the list in.
	RemovedEqualityDeleteFiles int

	// RemovedDeletionVectorFiles is the count of deletion vectors removed
	// because their referenced data file was rewritten.
	RemovedDeletionVectorFiles int

	// BytesBefore is the total size of input data files (from the compaction plan).
	BytesBefore int64

	// BytesAfter is the total size of output data files (measured from written files).
	BytesAfter int64

	// CompletedGroups contains the groups whose snapshots were committed in
	// partial-progress mode.
	CompletedGroups []CompactionGroupResult

	// FailedGroups contains groups whose catalog commits failed. A partial
	// rewrite can return both completed and failed groups.
	FailedGroups []CompactionGroupFailure
}

// CompactionGroupFailure records a group that could not be committed during a
// partial-progress rewrite.
type CompactionGroupFailure struct {
	PartitionKey string
	Err          error
}

// RewriteDataFiles runs a compaction rewrite as a table-level action. Atomic
// rewrites are committed before this method returns. Partial-progress rewrites
// commit each group as they run and return the latest table through
// [RewriteResult.Table].
//
// Use [Transaction.RewriteDataFiles] when the rewrite must be staged alongside
// other updates in one transaction. Partial progress is terminal and should
// normally be invoked through this action so callers do not accidentally try
// to commit the parent transaction again.
func (t Table) RewriteDataFiles(ctx context.Context, groups []CompactionTaskGroup, opts RewriteDataFilesOptions) (*RewriteResult, error) {
	txn, err := t.NewTransactionOnBranchWithError(MainBranch)
	if err != nil {
		return nil, err
	}

	result, err := txn.RewriteDataFiles(ctx, groups, opts)
	if err != nil || opts.PartialProgress {
		return result, err
	}

	committed, err := txn.Commit(ctx)
	if result != nil && err == nil {
		result.Table = committed
	}

	return result, err
}

// CompactionTaskGroup is a set of scan tasks in the same partition that
// should be compacted together. This bridges the compaction planner
// (table/compaction package) and the executor, avoiding a circular
// import between table and table/compaction.
//
// Use [compaction.Config.PlanCompaction] to produce groups, then convert
// [compaction.Group] → [CompactionTaskGroup] to call
// [Transaction.RewriteDataFiles] or [ExecuteCompactionGroup].
type CompactionTaskGroup struct {
	// PartitionKey is an opaque grouping key for display/logging.
	PartitionKey string

	// Tasks are the FileScanTasks to compact.
	Tasks []FileScanTask

	// TotalSizeBytes is the sum of data file sizes in this group.
	TotalSizeBytes int64
}

// CompactionGroupResult is the per-group output of a compaction
// worker: the new files written, the old files being replaced, and
// the position delete files safe to expunge in the rewrite snapshot.
//
// A distributed coordinator aggregates results from N workers and
// applies them to a [RewriteFiles] builder via [RewriteFiles.ApplyResult]
// to commit a single atomic snapshot. Each field is plain data
// ([]iceberg.DataFile values plus scalars) — callers serialize the
// contained DataFiles across process boundaries themselves; the
// typical pattern is to have the worker write a manifest containing
// the new files and ship the manifest path to the coordinator, which
// re-reads it.
type CompactionGroupResult struct {
	// PartitionKey mirrors [CompactionTaskGroup.PartitionKey] for
	// display/logging on the coordinator.
	PartitionKey string

	// OldDataFiles are the data files this group replaces.
	OldDataFiles []iceberg.DataFile

	// NewDataFiles are the consolidated outputs the worker wrote.
	NewDataFiles []iceberg.DataFile

	// SafePosDeletes are position-delete files referenced by tasks in
	// this group whose target data file is being rewritten, computed
	// via [CollectSafePositionDeletes]. Partial-progress callers must
	// re-check them against the complete commit batch before expunging
	// them because one file can reference data files in multiple groups.
	SafePosDeletes []iceberg.DataFile

	// SafeDeletionVectors are deletion vectors attached to tasks in this
	// group, computed via [CollectSafeDeletionVectors]. Each is bound to
	// a data file being rewritten, so all are safe to expunge.
	SafeDeletionVectors []iceberg.DataFile

	// BytesBefore is [CompactionTaskGroup.TotalSizeBytes] passed
	// through, recorded so the coordinator can roll up metrics
	// without re-reading the plan.
	BytesBefore int64

	// BytesAfter is the sum of [iceberg.DataFile.FileSizeBytes] across
	// NewDataFiles.
	BytesAfter int64
}

// RewriteDataFilesOptions bundles the per-rewrite knobs for
// [Transaction.RewriteDataFiles].
type RewriteDataFilesOptions struct {
	// PartialProgress, when true, commits non-empty groups in durable catalog
	// batches. When false (the default), every group is staged in a single
	// atomic rewrite snapshot.
	PartialProgress bool

	// MaxCommits bounds the number of catalog snapshots produced in
	// partial-progress mode. Groups are batched so all groups are processed
	// while staying within this bound. Zero uses the default of 10. It has no
	// effect for atomic rewrites.
	MaxCommits int

	// MaxFailedCommits bounds catalog commit failures in partial-progress mode.
	// Zero (the default) and negative values allow unlimited failures. Failures
	// within the bound are returned in RewriteResult.FailedGroups without making
	// the operation return an error.
	MaxFailedCommits int

	// SnapshotProps are added to the rewrite snapshot's summary.
	// In partial-progress mode the same properties land on every batch
	// snapshot rather than being summed or split.
	SnapshotProps iceberg.Properties

	// ExtraDeleteFilesToRemove are delete files that are dead after
	// the rewrite and that the caller wants expunged in the same
	// snapshot. Honored only when PartialProgress is false.
	//
	// Use [compaction.CollectDeadEqualityDeletesWithSpecs] and
	// [compaction.CollectDeadPositionDeletes] to compute this list
	// from the current snapshot. Position deletes attached to
	// rewritten tasks are already removed by the per-group staging;
	// listing them again here is harmless (each file is removed and
	// counted once).
	ExtraDeleteFilesToRemove []iceberg.DataFile

	// GroupOptions are forwarded to every [ExecuteCompactionGroup]
	// call to tune the per-group read+write pipeline (target file
	// size, scan concurrency). See the With* helpers returning
	// [CompactionGroupOption].
	GroupOptions []CompactionGroupOption
}

// CompactionGroupOption configures a single [ExecuteCompactionGroup]
// call. Use the With* helpers to construct values.
type CompactionGroupOption func(*compactionGroupConfig)

type compactionGroupConfig struct {
	targetFileSize        int64
	scanConcurrency       int
	arrowBatchSize        int
	recordBatchBufferSize int
	parquetRowGroupLimit  int
}

// WithCompactionTargetFileSize sets the size target for output files
// written by [ExecuteCompactionGroup]. Forwarded to [WriteRecords] as
// [WithTargetFileSize]. A non-positive value (including the zero
// default) means inherit the table's `write.target-file-size-bytes`
// property.
func WithCompactionTargetFileSize(size int64) CompactionGroupOption {
	if size <= 0 {
		return func(*compactionGroupConfig) {}
	}

	return func(c *compactionGroupConfig) {
		c.targetFileSize = size
	}
}

// WithCompactionScanConcurrency sets the scan concurrency used when
// reading the group's tasks. Forwarded to [Table.Scan] as
// [WithMaxConcurrency]. Zero (the default) means runtime.GOMAXPROCS.
// The scan runs min(n, number of tasks) workers, and each worker holds
// the decoded batches of at most one task until the writer has taken
// them, so the worker count multiplies the read-side term of the
// memory bound stated on [WithCompactionArrowBatchSize].
func WithCompactionScanConcurrency(n int) CompactionGroupOption {
	return func(c *compactionGroupConfig) {
		c.scanConcurrency = n
	}
}

// WithCompactionArrowBatchSize caps the number of rows decoded per
// Arrow record batch while reading the group's tasks, forwarded to the
// scan as [WithArrowBatchSize]. The record pipeline holds at most
//
//	workers x (rows in the largest task + n) + (recordBatchBufferSize + 2) x n
//
// rows, where workers is the scan's worker count (see
// [WithCompactionScanConcurrency]) and recordBatchBufferSize is the
// [WithCompactionRecordBatchBufferSize] value. Each scan worker holds
// the batches of at most one task at a time, in its hands, in the
// records channel, in the reorder heap or in the sequenced channel,
// and does not start its next task until the last batch of the
// previous one has been consumed; a task's batches total at most its
// row count rounded up to a multiple of n. The clustered writer holds
// one batch while routing it, recordBatchBufferSize batches in its
// input channel and one batch while encoding. Not covered: the Parquet
// reader's own buffers, including the raw batch it last returned, and
// delete-side memory, since positional deletes and deletion-vector
// bitmaps for the group's tasks are materialized up front and sized by
// delete volume, not by these knobs. A non-positive value keeps the
// table's read.parquet.batch-size property.
func WithCompactionArrowBatchSize(n int) CompactionGroupOption {
	return func(c *compactionGroupConfig) {
		if n > 0 {
			c.arrowBatchSize = n
		}
	}
}

// WithCompactionRecordBatchBufferSize sets the capacity, in record
// batches, of the write pipeline's per-writer input buffer, forwarded
// to [WriteRecords] as [WithRecordBatchBufferSize]. The default is 64
// batches. A non-positive value is ignored.
func WithCompactionRecordBatchBufferSize(n int) CompactionGroupOption {
	return func(c *compactionGroupConfig) {
		if n > 0 {
			c.recordBatchBufferSize = n
		}
	}
}

// WithCompactionParquetRowGroupLimit caps the rows per Parquet row
// group in the compacted output files, forwarded to [WriteRecords] as
// [WithParquetRowGroupLimit]. A non-positive value keeps the table's
// write.parquet.row-group-limit property.
func WithCompactionParquetRowGroupLimit(n int) CompactionGroupOption {
	return func(c *compactionGroupConfig) {
		if n > 0 {
			c.parquetRowGroupLimit = n
		}
	}
}

// RewriteDataFiles compacts the given groups by reading data with
// deletes applied, writing new consolidated files, and atomically
// replacing the old files. Position delete files that are fully
// applied (every referenced data file is in the rewrite set) are
// removed automatically.
//
// Cleanup beyond that per-group staging is the caller's
// responsibility: compute the dead sets with
// [compaction.CollectDeadEqualityDeletesWithSpecs] and
// [compaction.CollectDeadPositionDeletes] (against the same snapshot
// the rewrite is staged on) and pass them via
// [RewriteDataFilesOptions.ExtraDeleteFilesToRemove]. The executor
// only orchestrates the commit; it does not impose a cleanup policy.
// This split keeps the pure spec predicate in table/compaction and
// the unexported snapshot machinery in table.
//
// Use [compaction.Config.PlanCompaction] to produce the groups, then
// convert [compaction.Group] → [CompactionTaskGroup] and pass them
// here. Distributed coordinators stage worker results via
// [ExecuteCompactionGroup] and commit them via [Transaction.NewRewrite]
// + [RewriteFiles.ApplyResult] + [RewriteFiles.Commit] instead.
func (t *Transaction) RewriteDataFiles(ctx context.Context, groups []CompactionTaskGroup, opts RewriteDataFilesOptions) (*RewriteResult, error) {
	if _, err := t.txnMeta(); err != nil {
		return nil, err
	}
	if opts.PartialProgress {
		return t.rewriteDataFilesPartial(ctx, groups, opts)
	}
	if len(groups) == 0 {
		return &RewriteResult{}, nil
	}

	result := &RewriteResult{}
	rewrite := t.NewRewrite(opts.SnapshotProps)
	stagedDeleteFiles := make(map[string]struct{})

	for _, group := range groups {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		if len(group.Tasks) == 0 {
			continue
		}

		gr, err := ExecuteCompactionGroup(ctx, t.tbl, group, opts.GroupOptions...)
		if err != nil {
			return result, err
		}

		if len(gr.OldDataFiles) == 0 && len(gr.NewDataFiles) == 0 {
			continue
		}

		rewrite.ApplyResult(gr)
		accumulateGroupMetrics(result, gr)
		for _, df := range gr.SafePosDeletes {
			stagedDeleteFiles[df.FilePath()] = struct{}{}
		}
		for _, df := range gr.SafeDeletionVectors {
			stagedDeleteFiles[df.FilePath()] = struct{}{}
		}
	}

	if result.RewrittenGroups == 0 {
		return result, nil
	}

	// Extra delete files may overlap what the groups already staged
	// (e.g. [compaction.CollectDeadPositionDeletes] output includes
	// deletes attached to rewritten tasks); ReplaceFiles rejects
	// duplicate removals, so stage each file once.
	for _, df := range opts.ExtraDeleteFilesToRemove {
		if _, ok := stagedDeleteFiles[df.FilePath()]; ok {
			continue
		}
		stagedDeleteFiles[df.FilePath()] = struct{}{}
		rewrite.DeleteFile(df)
		switch {
		case df.ContentType() == iceberg.EntryContentEqDeletes:
			result.RemovedEqualityDeleteFiles++
		case IsDeletionVector(df):
			result.RemovedDeletionVectorFiles++
		default:
			result.RemovedPositionDeleteFiles++
		}
	}

	if err := rewrite.Commit(ctx); err != nil {
		return result, fmt.Errorf("commit compaction: %w", err)
	}

	return result, nil
}

// ExecuteCompactionGroup reads a compaction group's tasks (with
// deletes applied), writes consolidated output files via
// [WriteRecords], and computes the position-delete files safe to
// expunge in the rewrite snapshot. It does not commit — the caller
// hands the result to a coordinator that uses [Transaction.NewRewrite]
// + [RewriteFiles.ApplyResult] + [RewriteFiles.Commit] to stage the
// atomic commit.
//
// Empty groups return a zero [CompactionGroupResult] without doing
// any I/O.
//
// In-process callers should prefer [Transaction.RewriteDataFiles],
// which drives this and the commit step in one call.
//
// Tunables are exposed via [CompactionGroupOption]. The clustered
// write path is always used (a compaction group is single-partition
// by construction so its read stream is trivially clustered).
func ExecuteCompactionGroup(ctx context.Context, tbl *Table, group CompactionTaskGroup, opts ...CompactionGroupOption) (CompactionGroupResult, error) {
	if len(group.Tasks) == 0 {
		return CompactionGroupResult{PartitionKey: group.PartitionKey}, nil
	}

	cfg := compactionGroupConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	var scanOpts []ScanOption
	if cfg.scanConcurrency > 0 {
		scanOpts = append(scanOpts, WithMaxConcurrency(cfg.scanConcurrency))
	}
	if cfg.arrowBatchSize > 0 {
		scanOpts = append(scanOpts, WithArrowBatchSize(cfg.arrowBatchSize))
	}

	// Preserve row lineage only when every source file in the group carries
	// it. A mixed group (some files with FirstRowID, some without — e.g.
	// legacy files on a v3 table) would otherwise produce one output where
	// post-lineage rows have explicit _row_id values and pre-lineage rows
	// have nulls, which violates the per-file uniqueness/coverage
	// invariant the v3 spec requires. Row IDs are assigned lazily during
	// the first v3 manifest-list write after a v1/v2->v3 upgrade, so mixed
	// groups are expected during migration; for now we degrade gracefully
	// and do not preserve lineage for the surviving rows.
	preserveLineage := tbl.metadata.Version() >= 3 && allTasksHaveRowLineage(group.Tasks)
	if preserveLineage {
		scanOpts = append(scanOpts, WithRowLineage())
	} else if tbl.metadata.Version() >= 3 {
		// Drop lineage for the whole mixed group. Warn only when at least one
		// source file already carried lineage; all-legacy groups fall through
		// silently because there is no lineage to lose.
		var lineageFiles, legacyFiles int
		for _, t := range group.Tasks {
			if t.FirstRowID != nil {
				lineageFiles++
			} else {
				legacyFiles++
			}
		}
		if lineageFiles > 0 {
			slog.Warn("compaction group has mixed row lineage; dropping _row_id on output",
				"partition_key", group.PartitionKey,
				"lineage_files", lineageFiles,
				"legacy_files", legacyFiles)
		}
	}

	arrowSchema, records, err := tbl.Scan(scanOpts...).ReadTasks(ctx, group.Tasks)
	if err != nil {
		return CompactionGroupResult{}, fmt.Errorf("read tasks for compaction group %q: %w", group.PartitionKey, err)
	}

	// Each compaction group is single-partition by construction, so the
	// read stream is trivially clustered and we can use the clustered writer.
	writeOpts := []WriteRecordOption{WithClusteredWrite()}
	if cfg.targetFileSize > 0 {
		writeOpts = append(writeOpts, WithTargetFileSize(cfg.targetFileSize))
	}
	if cfg.recordBatchBufferSize > 0 {
		writeOpts = append(writeOpts, WithRecordBatchBufferSize(cfg.recordBatchBufferSize))
	}
	if cfg.parquetRowGroupLimit > 0 {
		writeOpts = append(writeOpts, WithParquetRowGroupLimit(cfg.parquetRowGroupLimit))
	}
	if preserveLineage {
		// Rebuild the arrow schema from the projected iceberg schema so the
		// reserved row-lineage field IDs (_row_id, _last_updated_sequence_number)
		// are attached as Arrow field metadata. ArrowSchemaToIceberg prefers
		// embedded field IDs when present and otherwise falls back to the
		// table's name mapping — which doesn't (and cannot) contain the
		// reserved metadata column names, so the fallback path panics.
		projectedSchema := iceberg.SchemaWithRowLineage(tbl.Schema())
		arrowSchema, err = SchemaToArrowSchemaWithOptions(projectedSchema, ArrowSchemaOptions{
			IncludeFieldIDs: true,
			TableProperties: tbl.Metadata().Properties(),
		})
		if err != nil {
			return CompactionGroupResult{}, fmt.Errorf("build arrow schema for lineage write in group %q: %w", group.PartitionKey, err)
		}
		writeOpts = append(writeOpts, WithPreserveRowLineage(projectedSchema))
	}

	var (
		newFiles   []iceberg.DataFile
		bytesAfter int64
	)
	for df, err := range WriteRecords(ctx, tbl, arrowSchema, records, writeOpts...) {
		if err != nil {
			return CompactionGroupResult{
				PartitionKey: group.PartitionKey,
				NewDataFiles: newFiles,
				BytesAfter:   bytesAfter,
			}, fmt.Errorf("write compacted files for group %q: %w", group.PartitionKey, err)
		}
		newFiles = append(newFiles, df)
		bytesAfter += df.FileSizeBytes()
	}

	oldFiles := make([]iceberg.DataFile, 0, len(group.Tasks))
	for _, task := range group.Tasks {
		oldFiles = append(oldFiles, task.File)
	}

	return CompactionGroupResult{
		PartitionKey:        group.PartitionKey,
		OldDataFiles:        oldFiles,
		NewDataFiles:        newFiles,
		SafePosDeletes:      CollectSafePositionDeletes(group.Tasks),
		SafeDeletionVectors: CollectSafeDeletionVectors(group.Tasks),
		BytesBefore:         group.TotalSizeBytes,
		BytesAfter:          bytesAfter,
	}, nil
}

// allTasksHaveRowLineage returns true iff every task in the group has a
// non-nil FirstRowID — i.e. every source file already carries v3 row lineage.
// It returns false for an empty task slice.
func allTasksHaveRowLineage(tasks []FileScanTask) bool {
	if len(tasks) == 0 {
		return false
	}
	for _, t := range tasks {
		if t.FirstRowID == nil {
			return false
		}
	}

	return true
}

// rewriteDataFilesPartial executes groups and commits them in durable batches.
// A batch is the atomic unit for both the MaxCommits bound and delete cleanup:
// classic position deletes are rechecked against the union of every old data
// file in the batch before they are removed. A later batch can fail without
// rolling back snapshots already committed for earlier batches.
func (t *Transaction) rewriteDataFilesPartial(ctx context.Context, groups []CompactionTaskGroup, opts RewriteDataFilesOptions) (*RewriteResult, error) {
	if err := t.checkNotNil(); err != nil {
		return nil, err
	}
	t.mx.Lock()
	defer t.mx.Unlock()

	meta, err := t.txnMeta()
	if err != nil {
		return nil, err
	}
	if t.committed {
		return nil, errors.New("transaction has already been committed")
	}
	result := &RewriteResult{Table: t.tbl}
	if len(meta.updates) > 0 || len(t.reqs) > 0 || len(t.validators) > 0 {
		return nil, fmt.Errorf("%w: partial progress requires a fresh transaction",
			ErrInvalidOperation)
	}
	maxCommits := opts.MaxCommits
	if maxCommits == 0 {
		maxCommits = 10
	}
	if maxCommits < 0 {
		return nil, fmt.Errorf("%w: MaxCommits must be non-negative", ErrInvalidOperation)
	}
	maxFailedCommits := opts.MaxFailedCommits
	if len(groups) == 0 {
		return result, nil
	}

	pendingGroups := make([]CompactionTaskGroup, 0, len(groups))
	for _, group := range groups {
		if len(group.Tasks) > 0 {
			pendingGroups = append(pendingGroups, group)
		}
	}
	if len(pendingGroups) == 0 {
		return result, nil
	}

	// Match Iceberg's action semantics: MaxCommits is a bound on snapshots,
	// not on the number of groups processed. Distribute all groups across at
	// most MaxCommits batches.
	groupsPerCommit := (len(pendingGroups)-1)/maxCommits + 1
	props := maps.Clone(opts.SnapshotProps)
	current := t.tbl
	failedCommits := 0

	for batchStart := 0; batchStart < len(pendingGroups); batchStart += groupsPerCommit {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		batchEnd := min(batchStart+groupsPerCommit, len(pendingGroups))
		batchGroups := pendingGroups[batchStart:batchEnd]
		batchResults := make([]CompactionGroupResult, 0, len(batchGroups))
		rewrittenPaths := make(map[string]struct{})
		rewrittenFiles := make([]iceberg.DataFile, 0)
		fs, err := current.fsF(ctx)
		if err != nil {
			return result, fmt.Errorf("open table IO for partial rewrite batch: %w", err)
		}
		cleanupBatch := func(cause error, extras ...CompactionGroupResult) error {
			results := batchResults
			if len(extras) > 0 {
				results = make([]CompactionGroupResult, 0, len(batchResults)+len(extras))
				results = append(results, batchResults...)
				results = append(results, extras...)
			}
			if cleanupErr := cleanupCompactionOutputs(fs, results); cleanupErr != nil {
				return errors.Join(cause, fmt.Errorf("clean up partial rewrite batch outputs: %w", cleanupErr))
			}

			return cause
		}

		for _, group := range batchGroups {
			if err := ctx.Err(); err != nil {
				return result, cleanupBatch(err)
			}

			gr, err := ExecuteCompactionGroup(ctx, current, group, opts.GroupOptions...)
			if err != nil {
				return result, cleanupBatch(err, gr)
			}

			if len(gr.OldDataFiles) == 0 && len(gr.NewDataFiles) == 0 {
				continue
			}
			batchResults = append(batchResults, gr)
			for _, df := range gr.OldDataFiles {
				if _, ok := rewrittenPaths[df.FilePath()]; ok {
					continue
				}
				rewrittenPaths[df.FilePath()] = struct{}{}
				rewrittenFiles = append(rewrittenFiles, df)
			}
		}

		if len(batchResults) == 0 {
			continue
		}

		deadPositionDeletes, err := CollectDeadPositionDeletes(
			ctx, fs, latestSnapshotForBranch(current.Metadata(), t.branch), rewrittenPaths)
		if err != nil {
			return result, cleanupBatch(fmt.Errorf("collect dead position deletes for partial rewrite batch: %w", err))
		}

		// Deletion vectors are one-to-one with their referenced data file, so
		// task-level results are sufficient. Deduplicate by reference because
		// ReplaceFiles rejects multiple DVs for one data file.
		safeDVs := make([]iceberg.DataFile, 0)
		seenDVRefs := make(map[string]struct{})
		for _, gr := range batchResults {
			for _, dv := range gr.SafeDeletionVectors {
				ref := dv.ReferencedDataFile()
				if ref == nil {
					continue
				}
				if _, ok := seenDVRefs[*ref]; ok {
					continue
				}
				seenDVRefs[*ref] = struct{}{}
				safeDVs = append(safeDVs, dv)
			}
		}
		deletesToRemove := append(deadPositionDeletes, safeDVs...)

		groupTxn, err := current.NewTransactionOnBranchWithError(t.branch)
		if err != nil {
			return result, cleanupBatch(fmt.Errorf("create transaction for partial rewrite batch: %w", err))
		}
		newDataFiles := make([]iceberg.DataFile, 0)
		oldDataFiles := make([]iceberg.DataFile, 0, len(rewrittenFiles))
		for _, gr := range batchResults {
			oldDataFiles = append(oldDataFiles, gr.OldDataFiles...)
			newDataFiles = append(newDataFiles, gr.NewDataFiles...)
		}
		if err := groupTxn.ReplaceFiles(ctx, oldDataFiles, newDataFiles, deletesToRemove,
			props, withRewriteSemantics()); err != nil {
			return result, cleanupBatch(fmt.Errorf("stage partial rewrite batch: %w", err))
		}
		groupTxn.addValidator(rewriteValidator(rewrittenFiles))

		next, err := groupTxn.Commit(ctx)
		if err != nil {
			if next != nil {
				// A non-nil table means the catalog commit succeeded even if a
				// post-commit hook returned an error. Keep the committed state and
				// stop before planning another batch from an error-bearing result.
				recordCommittedRewriteBatch(result, next, batchResults, deadPositionDeletes, safeDVs, &current, t)

				return result, err
			}

			// ErrCommitFailed is the only error that proves the catalog did not
			// commit. Every other error leaves commit state unknown, so continuing
			// with the old table could apply later batches on stale state.
			if !errors.Is(err, ErrCommitFailed) {
				t.committed = true

				return result, err
			}

			failedCommits++
			for _, gr := range batchResults {
				result.FailedGroups = append(result.FailedGroups, CompactionGroupFailure{
					PartitionKey: gr.PartitionKey,
					Err:          err,
				})
			}
			if cleanupErr := cleanupCompactionOutputs(fs, batchResults); cleanupErr != nil {
				return result, errors.Join(err, fmt.Errorf("clean up failed partial rewrite batch outputs: %w", cleanupErr))
			}
			if maxFailedCommits > 0 && failedCommits > maxFailedCommits {
				return result, fmt.Errorf("commit partial rewrite batch %d: %w (maximum failed commits reached)",
					batchStart/groupsPerCommit+1, err)
			}

			continue
		}

		recordCommittedRewriteBatch(result, next, batchResults, deadPositionDeletes, safeDVs, &current, t)
	}

	return result, nil
}

func recordCommittedRewriteBatch(
	result *RewriteResult,
	next *Table,
	batchResults []CompactionGroupResult,
	deadPositionDeletes []iceberg.DataFile,
	safeDVs []iceberg.DataFile,
	current **Table,
	txn *Transaction,
) {
	*current = next
	txn.tbl = next
	txn.committed = true
	for _, gr := range batchResults {
		accumulateGroupMetricsWithDeletes(result, gr, 0, 0)
		result.CompletedGroups = append(result.CompletedGroups, gr)
	}
	result.RemovedPositionDeleteFiles += len(deadPositionDeletes)
	result.RemovedDeletionVectorFiles += len(safeDVs)
	result.Table = next
}

func cleanupCompactionOutputs(fs iceio.IO, batchResults []CompactionGroupResult) error {
	paths := make(map[string]struct{})
	for _, result := range batchResults {
		for _, file := range result.NewDataFiles {
			paths[file.FilePath()] = struct{}{}
		}
	}

	var cleanupErr error
	for path := range paths {
		if err := fs.Remove(path); err != nil && !errors.Is(err, iofs.ErrNotExist) {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove %s: %w", path, err))
		}
	}

	return cleanupErr
}

func accumulateGroupMetrics(r *RewriteResult, gr CompactionGroupResult) {
	accumulateGroupMetricsWithDeletes(r, gr, len(gr.SafePosDeletes), len(gr.SafeDeletionVectors))
}

func accumulateGroupMetricsWithDeletes(r *RewriteResult, gr CompactionGroupResult, positionDeletes, deletionVectors int) {
	r.RewrittenGroups++
	r.AddedDataFiles += len(gr.NewDataFiles)
	r.RemovedDataFiles += len(gr.OldDataFiles)
	r.RemovedPositionDeleteFiles += positionDeletes
	r.RemovedDeletionVectorFiles += deletionVectors
	r.BytesBefore += gr.BytesBefore
	r.BytesAfter += gr.BytesAfter
}

// rewriteValidator builds a conflictValidatorFunc that rejects the
// commit if a concurrent snapshot added delete files targeting any of
// the rewritten data files — by referenced-data-file path, by
// file_path bounds, by partition overlap for partition-scoped
// pos-deletes, or (conservatively) any eq-delete during the rewrite.
// Always runs — no isolation gating, because rewrite is a structural
// operation, not a user-facing isolation choice.
func rewriteValidator(rewrittenFiles []iceberg.DataFile) conflictValidatorFunc {
	return rewriteValidatorWithReferencedDataFiles(rewrittenFiles, nil)
}

// rewriteValidatorWithReferencedDataFiles extends rewrite validation to data
// files targeted by deletion vectors added by the rewrite. Existing targets
// must remain live, and no concurrent delete may be added for them before the
// rewrite commits; otherwise the rewrite could create a duplicate DV or leave
// a position delete surviving beside the new DV.
func rewriteValidatorWithReferencedDataFiles(
	rewrittenFiles []iceberg.DataFile,
	referencedDataFilePaths []string,
) conflictValidatorFunc {
	return func(cc *conflictContext) error {
		if cc == nil {
			return nil
		}
		if err := validateDataFilesExist(cc, referencedDataFilePaths); err != nil {
			return err
		}

		return validateNoNewDeletesForRewrittenFiles(cc, rewrittenFiles)
	}
}

// CollectSafePositionDeletes returns position delete files from the
// given tasks that are safe to remove during compaction.
//
// A position delete file is safe to remove when it was matched to a
// data file (via scan planning) and that data file is being rewritten
// in this compaction group. Since ReadTasks applies the deletes during
// reading, the new output files will not contain the deleted rows.
//
// Only position deletes (EntryContentPosDeletes) are considered.
// Equality deletes are decided by [compaction.DecideDeadEqualityDeletesWithSpecs]
// (which needs partition-wide visibility, not just the task scope).
// Deletion vectors will be handled when DV read support lands.
//
// Caller contract: every data file referenced by a returned pos-delete
// must be in the caller's rewrite set across the entire commit.
// This function only sees one group's tasks, but a pos-delete file
// can reference data files across multiple groups (the planner
// bin-packs within a partition via [compaction.Config.PlanCompaction]
// and skips files via MinInputFiles). If a pos-delete is reported safe
// by one group but references a still-live data file in another group
// — or a file the planner skipped — committing only this group's
// rewrite would orphan the still-live data file's deletes. Coordinators
// that aggregate multiple groups into one rewrite snapshot are
// responsible for re-checking against the full set of rewritten paths,
// or for moving this computation leader-side once worker outputs have
// aggregated.
//
// [ExecuteCompactionGroup] calls this internally to populate
// [CompactionGroupResult.SafePosDeletes]. It is kept exported for
// custom workers that want the spec-shaped predicate without taking
// the rest of [ExecuteCompactionGroup]'s read+write pipeline.
func CollectSafePositionDeletes(tasks []FileScanTask) []iceberg.DataFile {
	seen := make(map[string]bool)
	var safe []iceberg.DataFile

	for _, task := range tasks {
		for _, df := range task.DeleteFiles {
			if df.ContentType() != iceberg.EntryContentPosDeletes {
				continue
			}

			path := df.FilePath()
			if seen[path] {
				continue
			}
			seen[path] = true
			safe = append(safe, df)
		}
	}

	return safe
}

// CollectSafeDeletionVectors returns the tasks' deletion vectors, deduplicated
// by referenced data file.
//
// Scan planning attaches to a task only the DV referencing its own data file,
// so every returned DV references a file in the rewrite set. A hand-built
// [FileScanTask] carrying a DV for some other live data file would have that DV
// expunged here — populate DeletionVectorFiles only from scan planning.
func CollectSafeDeletionVectors(tasks []FileScanTask) []iceberg.DataFile {
	seen := make(map[string]struct{})
	var safe []iceberg.DataFile

	for _, task := range tasks {
		for _, dv := range task.DeletionVectorFiles {
			ref := iceberginternal.BorrowedDataFileReferencedDataFile(dv)
			if ref == nil {
				continue
			}
			if _, ok := seen[*ref]; ok {
				continue
			}
			seen[*ref] = struct{}{}
			safe = append(safe, dv)
		}
	}

	return safe
}
