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
	"fmt"
	"maps"

	"github.com/apache/iceberg-go"
)

// RewriteFiles is the snapshot-operation builder for rewrite
// (compaction) commits. It is the snapshot-level sibling of [RowDelta]
// and mirrors Java's org.apache.iceberg.RewriteFiles interface
// (returned by Table.newRewrite() in Java).
//
// Compared to a raw [Transaction.ReplaceFiles] call, the builder
// owns the rewrite-specific isolation contract internally:
//
//   - The overwrite producer's default isolation validator is suppressed
//     (concurrent appends into rewritten partitions are allowed; this
//     is the defining behavior of a rewrite).
//   - A rewrite-specific conflict validator is registered so concurrent
//     pos/eq-delete files targeting any rewritten data file are
//     rejected pre-flight at [Transaction.Commit] time. A pos-delete
//     is matched to a rewritten file via its referenced_data_file
//     column, via equal file_path lower/upper bounds when that column
//     is unset, or — when neither resolves a single path — via
//     partition overlap with a rewritten file. Eq-deletes are matched
//     conservatively: any concurrent eq-delete during the rewrite
//     conflicts.
//
// Distributed compaction coordinators construct one [RewriteFiles] on
// the leader transaction, feed worker outputs in via [RewriteFiles.ApplyResult],
// and commit one snapshot. In-process callers can use
// [Transaction.RewriteDataFiles] which drives this builder internally.
//
// The builder follows the same fail-fast pattern as
// [view.MetadataBuilder]: a method that hits an invalid input stages
// the error and short-circuits all subsequent calls until
// [RewriteFiles.Commit] drains it. The builder is single-use; once
// Commit has been called, a second call returns an error regardless
// of whether the first call succeeded.
type RewriteFiles struct {
	txn                     *Transaction
	dataFilesToDelete       []iceberg.DataFile
	dataFilesToAdd          []iceberg.DataFile
	deleteFilesToAdd        []rewriteDeleteFileAddition
	deleteFilesToRemove     []iceberg.DataFile
	autoDeleteFilesToRemove []iceberg.DataFile
	dataSequenceNumber      *int64
	snapshotProps           iceberg.Properties
	err                     error
	committed               bool
}

type rewriteDeleteFileAddition struct {
	file               iceberg.DataFile
	dataSequenceNumber *int64
}

// NewRewrite returns a [RewriteFiles] builder bound to this transaction.
// Mirrors Java's org.apache.iceberg.Table#newRewrite. snapshotProps is
// cloned and the clone is added to the rewrite snapshot's summary;
// pass nil for none.
//
// Usage:
//
//	rewrite := tx.NewRewrite(nil)
//	rewrite.DeleteFile(oldDataFile)
//	rewrite.AddDataFile(newDataFile)
//	if err := rewrite.Commit(ctx); err != nil { ... }
//	committed, err := tx.Commit(ctx)
func (t *Transaction) NewRewrite(snapshotProps iceberg.Properties) *RewriteFiles {
	return &RewriteFiles{txn: t, snapshotProps: maps.Clone(snapshotProps)}
}

// DeleteFile marks a file for removal in this rewrite. Routes by
// content type: data files are queued as data-file replacements;
// pos/eq-delete files are queued for delete-file removal alongside
// the data rewrite (typical when a delete is fully applied to data
// files being rewritten and is therefore safe to expunge).
//
// Any other content type stages an error that is returned from the
// next [RewriteFiles.Commit] call.
func (r *RewriteFiles) DeleteFile(df iceberg.DataFile) *RewriteFiles {
	if r.err != nil {
		return r
	}
	if df == nil {
		r.err = fmt.Errorf("%w: DeleteFile got nil data file", ErrInvalidOperation)

		return r
	}

	switch df.ContentType() {
	case iceberg.EntryContentData:
		r.dataFilesToDelete = append(r.dataFilesToDelete, df)
	case iceberg.EntryContentPosDeletes, iceberg.EntryContentEqDeletes:
		r.deleteFilesToRemove = append(r.deleteFilesToRemove, df)
	default:
		r.err = fmt.Errorf("%w: DeleteFile got unsupported content type %s (%s)",
			ErrInvalidOperation, df.ContentType(), df.FilePath())
	}

	return r
}

// removeAutomaticDeleteFile queues a delete file that became removable as a
// consequence of rewriting data files. These files are removed in the same
// snapshot, but they are not considered source files for an explicit delete
// file rewrite.
func (r *RewriteFiles) removeAutomaticDeleteFile(df iceberg.DataFile) *RewriteFiles {
	if r.err != nil {
		return r
	}
	if df == nil {
		r.err = fmt.Errorf("%w: automatic delete removal got nil data file", ErrInvalidOperation)

		return r
	}

	switch df.ContentType() {
	case iceberg.EntryContentPosDeletes, iceberg.EntryContentEqDeletes:
		r.autoDeleteFilesToRemove = append(r.autoDeleteFilesToRemove, df)
	default:
		r.err = fmt.Errorf("%w: automatic delete removal got unsupported content type %s (%s)",
			ErrInvalidOperation, df.ContentType(), df.FilePath())
	}

	return r
}

// AddDataFile queues a new data file. Use [RewriteFiles.AddDeleteFile] for
// positional deletes, equality deletes, or deletion vectors.
func (r *RewriteFiles) AddDataFile(df iceberg.DataFile) *RewriteFiles {
	if r.err != nil {
		return r
	}
	if df == nil {
		r.err = fmt.Errorf("%w: AddDataFile got nil data file", ErrInvalidOperation)

		return r
	}

	if df.ContentType() != iceberg.EntryContentData {
		r.err = fmt.Errorf("%w: AddDataFile only supports data files; got content type %s (%s)",
			ErrInvalidOperation, df.ContentType(), df.FilePath())

		return r
	}
	r.dataFilesToAdd = append(r.dataFilesToAdd, df)

	return r
}

// DataSequenceNumber configures the data sequence number used for every data
// file added by this rewrite. This is useful when replacement data files must
// retain the applicability of delete files committed before the rewrite
// snapshot.
func (r *RewriteFiles) DataSequenceNumber(seq int64) *RewriteFiles {
	if r.err != nil {
		return r
	}
	if seq < 0 {
		r.err = fmt.Errorf("%w: invalid rewrite data sequence number %d", ErrInvalidOperation, seq)

		return r
	}

	seqCopy := seq
	r.dataSequenceNumber = &seqCopy

	return r
}

// AddDeleteFile queues a new positional delete, equality delete, or deletion
// vector for this rewrite. When existing delete files are replaced, the new
// file inherits the maximum data sequence number of those files.
func (r *RewriteFiles) AddDeleteFile(df iceberg.DataFile) *RewriteFiles {
	if r.err != nil {
		return r
	}
	if df == nil {
		r.err = fmt.Errorf("%w: AddDeleteFile got nil data file", ErrInvalidOperation)

		return r
	}

	switch df.ContentType() {
	case iceberg.EntryContentPosDeletes, iceberg.EntryContentEqDeletes:
		r.deleteFilesToAdd = append(r.deleteFilesToAdd, rewriteDeleteFileAddition{file: df})
	default:
		r.err = fmt.Errorf("%w: AddDeleteFile only supports delete files; got content type %s (%s)",
			ErrInvalidOperation, df.ContentType(), df.FilePath())
	}

	return r
}

// AddDeleteFileWithDataSequenceNumber is a convenience form of
// [RewriteFiles.AddDeleteFile]. The caller must assign the maximum data
// sequence number of the exact source delete files represented by df.
func (r *RewriteFiles) AddDeleteFileWithDataSequenceNumber(df iceberg.DataFile, seq int64) *RewriteFiles {
	if r.err != nil {
		return r
	}
	if df == nil {
		return r.AddDeleteFile(df)
	}
	if seq < 0 {
		r.err = fmt.Errorf("%w: AddDeleteFile got invalid data sequence number %d for %s",
			ErrInvalidOperation, seq, df.FilePath())

		return r
	}

	switch df.ContentType() {
	case iceberg.EntryContentPosDeletes, iceberg.EntryContentEqDeletes:
		seqCopy := seq
		r.deleteFilesToAdd = append(r.deleteFilesToAdd, rewriteDeleteFileAddition{
			file:               df,
			dataSequenceNumber: &seqCopy,
		})
	default:
		r.err = fmt.Errorf("%w: AddDeleteFile only supports delete files; got content type %s (%s)",
			ErrInvalidOperation, df.ContentType(), df.FilePath())
	}

	return r
}

// AddFile queues a file according to its content type. It is useful for
// coordinators that carry data and delete files in one result slice.
func (r *RewriteFiles) AddFile(df iceberg.DataFile) *RewriteFiles {
	if r.err != nil {
		return r
	}
	if df == nil {
		r.err = fmt.Errorf("%w: AddFile got nil data file", ErrInvalidOperation)

		return r
	}

	switch df.ContentType() {
	case iceberg.EntryContentData:
		return r.AddDataFile(df)
	case iceberg.EntryContentPosDeletes, iceberg.EntryContentEqDeletes:
		return r.AddDeleteFile(df)
	default:
		r.err = fmt.Errorf("%w: AddFile got unsupported content type %s (%s)",
			ErrInvalidOperation, df.ContentType(), df.FilePath())

		return r
	}
}

// Apply is a bulk shortcut that routes three slices onto this builder:
// every entry in deletes is queued via
// [RewriteFiles.DeleteFile] (which routes data vs. delete files by
// content type), every entry in adds via [RewriteFiles.AddFile], and every
// entry in safeDeletes as an automatic delete-file removal.
//
// Deprecated: use [RewriteFiles.ApplyResult], which also carries
// SafeDeletionVectors. Apply has no slot for them, so a coordinator wiring
// worker output through Apply leaves deletion vectors for the rewritten files
// orphaned.
func (r *RewriteFiles) Apply(deletes, adds, safeDeletes []iceberg.DataFile) *RewriteFiles {
	if r.err != nil {
		return r
	}

	for _, df := range deletes {
		r.DeleteFile(df)
	}
	for _, df := range adds {
		r.AddFile(df)
	}
	for _, df := range safeDeletes {
		r.removeAutomaticDeleteFile(df)
	}

	return r
}

// ApplyResult queues a worker's [CompactionGroupResult] onto this builder,
// routing OldDataFiles (DeleteFile), NewDataFiles (AddFile), SafePosDeletes
// and SafeDeletionVectors as automatic delete-file removals. Prefer it over
// [RewriteFiles.Apply], which cannot carry SafeDeletionVectors.
//
// Typical distributed-coordinator pattern:
//
//	rewrite := leaderTxn.NewRewrite(snapshotProps)
//	for _, gr := range workerResults {
//	    rewrite.ApplyResult(gr)
//	}
//	if err := rewrite.Commit(ctx); err != nil { ... }
func (r *RewriteFiles) ApplyResult(gr CompactionGroupResult) *RewriteFiles {
	for _, df := range gr.OldDataFiles {
		r.DeleteFile(df)
	}
	for _, df := range gr.NewDataFiles {
		r.AddFile(df)
	}
	// DVs are automatic removals keyed by their referenced data file.
	for _, df := range gr.SafePosDeletes {
		r.removeAutomaticDeleteFile(df)
	}
	for _, df := range gr.SafeDeletionVectors {
		r.removeAutomaticDeleteFile(df)
	}

	return r
}

// Commit stages the rewrite snapshot on the underlying transaction.
// The catalog commit happens once, later, at [Transaction.Commit] time.
//
// Commit is single-shot: any second call returns an error regardless
// of whether the first call succeeded, and neither re-stages the
// rewrite nor re-registers the conflict validator. Returns an error
// if any file passed to [RewriteFiles.AddFile], [RewriteFiles.AddDataFile], or
// [RewriteFiles.DeleteFile] had an unsupported content type, if the
// builder has no file changes, or if the underlying
// [Transaction.ReplaceFiles] call fails.
func (r *RewriteFiles) Commit(ctx context.Context) error {
	if r.committed {
		return fmt.Errorf("%w: RewriteFiles.Commit already called on this builder", ErrInvalidOperation)
	}
	r.committed = true

	if r.err != nil {
		return r.err
	}
	if len(r.dataFilesToDelete) == 0 && len(r.dataFilesToAdd) == 0 &&
		len(r.deleteFilesToAdd) == 0 && len(r.deleteFilesToRemove) == 0 &&
		len(r.autoDeleteFilesToRemove) == 0 {
		return fmt.Errorf("%w: rewrite must have at least one file change", ErrInvalidOperation)
	}
	// Adds-without-deletes would route through ReplaceFiles →
	// ReplaceDataFilesWithDataFiles → AddDataFiles, an OpAppend
	// producer that never reads cfg.rewriteSemantics. The snapshot
	// would be tagged append with no rewrite validator — silently
	// wrong for a rewrite. A pure delete-file expunge (only
	// deleteFilesToRemove non-empty) is still legitimate.
	if len(r.dataFilesToDelete) == 0 && len(r.dataFilesToAdd) > 0 {
		return fmt.Errorf("%w: rewrite must delete at least one data file when adding data files", ErrInvalidOperation)
	}

	opts := []WriteOption{withRewriteSemantics()}
	if r.dataSequenceNumber != nil {
		opts = append(opts, withDataSequenceNumber(*r.dataSequenceNumber))
	}
	if err := r.txn.replaceFiles(ctx, r.dataFilesToDelete, r.dataFilesToAdd,
		r.deleteFilesToRemove, r.autoDeleteFilesToRemove, r.deleteFilesToAdd,
		r.snapshotProps, opts...); err != nil {
		return err
	}

	if len(r.dataFilesToDelete) > 0 && len(r.deleteFilesToAdd) == 0 {
		r.txn.addValidator(rewriteValidator(r.dataFilesToDelete))
	}

	return nil
}
