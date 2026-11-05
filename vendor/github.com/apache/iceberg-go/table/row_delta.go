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
	"maps"

	"github.com/apache/iceberg-go"
	iceberginternal "github.com/apache/iceberg-go/internal"
	iceio "github.com/apache/iceberg-go/io"
)

// RowDelta encodes a set of row-level changes to a table: new data files
// (inserts) and delete files (equality or position deletes). A delta
// that replaces a data file's deletion vector can remove the superseded
// DV via [RowDelta.RemoveDeletes]. All changes are committed atomically
// in a single snapshot.
//
// The operation type of the produced snapshot is determined automatically:
//   - Data files only → OpAppend
//   - Delete files only → OpDelete
//   - Both data and delete files → OpOverwrite
//
// This matches the semantics of Java's BaseRowDelta. It is the primary
// API for CDC/streaming workloads where INSERTs, UPDATEs, and DELETEs
// must be committed together.
//
// Client-side conflict validation runs before the commit is sent to
// the catalog:
//   - Position deletes: referenced data files must still be reachable
//     from the current branch head (validateDataFilesExist).
//   - Equality deletes under write.delete.isolation-level=serializable
//     (the default): concurrent data files in the same partition(s) as
//     the equality deletes are rejected. For partitioned tables an
//     OR-of-equalities filter is built from the eq-delete files'
//     partition tuples and routed through validateAddedDataFilesMatchingFilter
//     (spec-evolution safe, manifest-summary pruning, type-aware evaluation).
//     For unpartitioned tables the check is conservative (AlwaysTrue —
//     any concurrent append is a conflict). Opt out by setting
//     write.delete.isolation-level=snapshot.
//
// The pre-flight runs before cat.CommitTable on every commit attempt.
// On the first attempt the writer's view and the catalog state are the
// same, so there are no concurrent snapshots to inspect and the checks
// short-circuit; on retries doCommit refreshes the catalog state and
// the checks run against the freshly loaded branch head.
//
// Usage:
//
//	rd := tx.NewRowDelta(snapshotProps)
//	rd.AddRows(dataFile1, dataFile2)
//	rd.AddDeletes(equalityDeleteFile1)
//	err := rd.Commit(ctx)
type RowDelta struct {
	txn         *Transaction
	dataFiles   []iceberg.DataFile
	delFiles    []iceberg.DataFile
	removedDels []iceberg.DataFile
	props       iceberg.Properties
}

// NewRowDelta creates a new RowDelta for committing row-level changes
// within this transaction. The provided properties are included in the
// snapshot summary.
func (t *Transaction) NewRowDelta(snapshotProps iceberg.Properties) *RowDelta {
	return &RowDelta{
		txn:   t,
		props: maps.Clone(snapshotProps),
	}
}

// AddRows adds data files containing new rows (inserts) to this RowDelta.
func (rd *RowDelta) AddRows(files ...iceberg.DataFile) *RowDelta {
	rd.dataFiles = append(rd.dataFiles, files...)

	return rd
}

// AddDeletes adds delete files (equality or position) to this RowDelta.
// Equality delete files must have ContentType == EntryContentEqDeletes
// and non-empty EqualityFieldIDs referencing valid schema columns.
// Position delete files must have ContentType == EntryContentPosDeletes.
// DVs are rejected by Commit unless the table is v3 or later.
func (rd *RowDelta) AddDeletes(files ...iceberg.DataFile) *RowDelta {
	rd.delFiles = append(rd.delFiles, files...)

	return rd
}

// RemoveDeletes marks superseded deletion vectors as removed in the
// snapshot produced by this RowDelta, mirroring Java's
// RowDelta#removeDeletes.
//
// The v3 spec allows at most one live deletion vector per data file: a
// writer adding a DV for a data file that already carries one must
// write a DV containing all of the previous DV's positions and remove
// the previous DV in the same snapshot. RemoveDeletes is the removal
// half of that contract; Commit validates that every removed DV's
// referenced data file gets a replacement DV among the files passed to
// AddDeletes, so the delta cannot silently resurrect deleted rows.
//
// Only deletion vectors (Puffin position deletes with a referenced
// data file) can be removed here. Expunging fully-applied position or
// equality delete files belongs to the rewrite path
// ([Transaction.ReplaceFiles] / [RewriteFiles]), which owns the
// data-file replacement validation that makes such removals safe.
//
// Removal identity is (file path, referenced data file): a multi-blob
// Puffin file may carry deletion vectors for several data files under
// one path, and removing one of its entries leaves the others live.
// When exactly one live entry carries the removed file's path, it is
// selected regardless of the caller copy's (possibly stale) reference;
// when several live entries share the path, the removed file's
// ReferencedDataFile selects which entry to remove, and Commit rejects
// the removal as ambiguous if it does not identify exactly one.
//
// Removal resolution is snapshot-relative, so a commit carrying
// removals is never replayed on an optimistic-concurrency conflict: if
// a concurrent writer advances the branch first, Commit's transaction
// fails with ErrCommitFailed instead of retrying, and the caller must
// reload the table and rebuild the delta against the fresh snapshot.
func (rd *RowDelta) RemoveDeletes(files ...iceberg.DataFile) *RowDelta {
	rd.removedDels = append(rd.removedDels, files...)

	return rd
}

// Commit validates and commits all accumulated row-level changes as a
// single atomic snapshot. Returns an error if there are no files to
// commit, if any file has an unexpected content type, if a removed
// deletion vector lacks a replacement or is not referenced by the
// current snapshot, if the delta would leave two live deletion vectors
// on one data file (a replacement added while the superseded live DV
// is not removed, or two added replacements for one data file), or if
// the table format version does not support delete files.
// Deletes require v2 or later, DVs require v3 or later.
//
// With removals present the staged commit is non-replayable: an
// optimistic-concurrency conflict fails the transaction's Commit with
// ErrCommitFailed rather than refreshing and retrying (see
// RemoveDeletes).
func (rd *RowDelta) Commit(ctx context.Context) error {
	meta, err := rd.txn.txnMeta()
	if err != nil {
		return err
	}

	// A removal-only delta deliberately passes this guard: it fails
	// later in validateRemovedDeletes with the replacement-required
	// error, which tells the caller what is actually missing.
	if len(rd.dataFiles) == 0 && len(rd.delFiles) == 0 && len(rd.removedDels) == 0 {
		return errors.New("row delta must have at least one data file or delete file")
	}

	// Delete files require format version >= 2.
	if len(rd.delFiles) > 0 && meta.formatVersion < 2 {
		return fmt.Errorf("delete files require table format version >= 2, got v%d",
			meta.formatVersion)
	}

	// Deletion vectors — the only removable delete files — exist only
	// in format version >= 3.
	if len(rd.removedDels) > 0 && meta.formatVersion < 3 {
		return fmt.Errorf("removing deletion vectors requires table format version >= 3, got v%d",
			meta.formatVersion)
	}

	for _, f := range rd.dataFiles {
		if f.ContentType() != iceberg.EntryContentData {
			return fmt.Errorf("expected data file, got content type %s: %s",
				f.ContentType(), f.FilePath())
		}
	}

	schema := meta.CurrentSchema()
	for _, f := range rd.delFiles {
		ct := f.ContentType()
		if ct != iceberg.EntryContentPosDeletes && ct != iceberg.EntryContentEqDeletes {
			return fmt.Errorf("expected delete file, got content type %s: %s",
				ct, f.FilePath())
		}

		if IsDeletionVector(f) {
			if err := validateDeletionVectorToAdd(f, meta.formatVersion, "row delta"); err != nil {
				return err
			}
		} else if ct == iceberg.EntryContentPosDeletes && meta.formatVersion >= 3 {
			return fmt.Errorf("position delete file %s must be a deletion vector for v%d table for row delta",
				f.FilePath(), meta.formatVersion)
		}

		// Equality delete files must declare which columns form the delete key,
		// and those columns must exist in the current schema.
		if ct == iceberg.EntryContentEqDeletes {
			eqIDs := f.EqualityFieldIDs()
			if len(eqIDs) == 0 {
				return fmt.Errorf("equality delete file must have non-empty EqualityFieldIDs: %s",
					f.FilePath())
			}

			if _, err := validateEqualityFieldIDs(schema, eqIDs); err != nil {
				return fmt.Errorf("invalid equality delete file %s: %w", f.FilePath(), err)
			}
		}
	}

	fs, err := rd.txn.tbl.fsF(ctx)
	if err != nil {
		return err
	}

	wfs, err := requireWriteFileIO(fs)
	if err != nil {
		return err
	}

	op := rd.Operation()

	var producer *snapshotProducer
	if len(rd.removedDels) > 0 {
		// Resolve the removed files against the current snapshot's
		// delete manifests so both validation and the produced DELETED
		// entries work from the manifest's own DataFile (correct
		// content type, referenced data file, spec ID, partition, and
		// sequence numbers) rather than the caller's copy, which may
		// carry stale or forged metadata. replacedLive collects the
		// live DVs whose referenced data file this delta adds a
		// replacement for; validateRemovedDeletes checks each is
		// actually removed.
		resolvedRemovals, replacedLive, err := rd.resolveRemovedDeletes(fs, meta)
		if err != nil {
			return err
		}

		if err := rd.validateRemovedDeletes(resolvedRemovals, replacedLive); err != nil {
			return err
		}

		// The overwrite producer knows how to drop the removed entries
		// from inherited delete manifests and to record them with
		// status DELETED in the produced snapshot. Its default
		// conflict validator (any concurrent append conflicts under
		// serializable isolation) implements overwrite semantics, not
		// row-delta semantics; RowDelta registers its own validator
		// below, so suppress the default the same way rewrites do.
		producer = newOverwriteFilesProducer(op, rd.txn, wfs, nil, rd.props)
		producer.setSkipDefaultValidator(true)

		for _, live := range resolvedRemovals {
			producer.removeDeletionVector(live)
		}
	} else {
		producer = newFastAppendFilesProducer(op, rd.txn, wfs, nil, rd.props)
	}

	for _, f := range rd.dataFiles {
		producer.appendDataFile(f)
	}

	for _, f := range rd.delFiles {
		producer.appendDeleteFile(f)
	}

	updates, reqs, err := producer.commit(ctx)
	if err != nil {
		return err
	}

	// Register RowDelta's pre-commit conflict validator. The underlying
	// producer contributes no conflict check of its own — fast-append's
	// validator is a no-op, and the overwrite producer's default is
	// suppressed above; RowDelta semantics (pos-delete references,
	// eq-delete predicate) require a dedicated check that
	// snapshot_producers does not know about.
	rd.txn.addValidator(rd.validate)

	return rd.txn.apply(updates, reqs)
}

// pathRefKey is the removal identity of a delete-manifest entry: its
// file path plus the data file its deletion vector references (""
// for delete files that record none).
type pathRefKey struct{ path, ref string }

// explicitReferencedDataFile returns df's referenced_data_file field,
// or "" when unset. Unlike referencedDataFilePath it does not fall back
// to file_path column bounds: it is used for removal identity, where a
// bounds-derived guess must not stand in for the recorded reference.
func explicitReferencedDataFile(df iceberg.DataFile) string {
	if ref := df.ReferencedDataFile(); ref != nil {
		return *ref
	}

	return ""
}

// validateRemovedDeletes enforces the DV-supersession contract against
// the RESOLVED live entries returned by resolveRemovedDeletes. The
// caller's copies are deliberately not consulted: a stale or forged
// DataFile with a real path but a wrong referenced_data_file must not
// be able to pair a removal with an unrelated replacement.
//
//   - at most one added DV may reference a given data file; two
//     replacements for one data file would commit two live DVs, which
//     the v3 spec forbids and scan planning rejects.
//   - every removed entry must be a deletion vector (a Puffin
//     pos-delete file with referenced_data_file set); other delete
//     files are removed through the rewrite path, which validates them
//     differently.
//   - every removed DV's referenced data file — as recorded by the
//     live manifest entry — must get a replacement DV among the added
//     delete files. The v3 spec requires the superseding DV to be
//     committed in the same snapshot as the removal; without a
//     replacement the delta would resurrect the rows the removed DV
//     was hiding. The replacement must contain the removed DV's
//     positions — that is the writer's responsibility, as in Java;
//     only the metadata pairing is validated here.
//   - conversely, every live DV whose referenced data file gets a
//     replacement in this delta (replacedLive) must be among the
//     removals — otherwise the commit would leave both the old and the
//     new DV live on one data file.
func (rd *RowDelta) validateRemovedDeletes(resolved, replacedLive []iceberg.DataFile) error {
	addedDVs := make(map[string]struct{}, len(rd.delFiles))
	for _, f := range rd.delFiles {
		if ref := explicitReferencedDataFile(f); IsDeletionVector(f) && ref != "" {
			if _, ok := addedDVs[ref]; ok {
				return fmt.Errorf("multiple added deletion vectors reference data file %s; at most one live deletion vector may exist per data file",
					ref)
			}
			addedDVs[ref] = struct{}{}
		}
	}

	removedKeys := make(map[pathRefKey]struct{}, len(resolved))
	for _, live := range resolved {
		ref := explicitReferencedDataFile(live)
		if !IsDeletionVector(live) {
			return fmt.Errorf("only deletion vectors can be removed by a row delta, got %s file with content type %s: %s",
				live.FileFormat(), live.ContentType(), live.FilePath())
		}
		if ref == "" {
			return fmt.Errorf("cannot remove %s: the live deletion vector entry does not record a referenced data file",
				live.FilePath())
		}

		if _, ok := addedDVs[ref]; !ok {
			return fmt.Errorf("cannot remove deletion vector %s: no replacement deletion vector for data file %s in this row delta",
				live.FilePath(), ref)
		}
		removedKeys[pathRefKey{path: live.FilePath(), ref: ref}] = struct{}{}
	}

	for _, live := range replacedLive {
		key := pathRefKey{path: live.FilePath(), ref: explicitReferencedDataFile(live)}
		if _, ok := removedKeys[key]; !ok {
			return fmt.Errorf("cannot add a replacement deletion vector for data file %s: live deletion vector %s is not removed by this row delta; the superseded entry must be removed in the same snapshot",
				key.ref, key.path)
		}
	}

	return nil
}

// resolveRemovedDeletes walks the current snapshot's delete manifests
// and resolves each removed file to its live manifest entry, returned
// in the order the removals were registered.
//
// Removal identity is (file path, referenced data file), not path
// alone: a multi-blob Puffin file legally carries deletion vectors for
// several data files under one path, one manifest entry each, and
// removing one blob's entry must not drop its siblings. When a single
// live entry carries the path, the caller's copy resolves to it
// regardless of its (possibly stale) metadata; when several live
// entries share the path, the caller's referenced data file selects
// among them and the removal is rejected as ambiguous if it does not
// identify exactly one. Blob offset and size are not part of the
// identity: the spec allows at most one live DV per data file, so
// (path, referenced data file) is unique among live entries, and every
// record this commit produces is built from the live entry's own
// DataFile, so stale caller-side offsets cannot corrupt the removal.
//
// A removed file that does not resolve to a live delete entry is an
// error: the producer would otherwise silently skip the removal and
// the commit would strand two live DVs on one data file. Two removals
// resolving to live entries that reference the same data file are also
// an error — the producer keys DV removals by referenced data file, so
// letting both through would silently drop one, and two live DVs on
// one data file is a spec violation to surface, not paper over. For
// the same reason a removal whose referenced data file matches more
// than one live entry at the path is rejected: the producer would
// tombstone every matching entry while the snapshot summary counts
// only one removal.
//
// The walk also collects replacedLive: every live deletion vector
// whose referenced data file this delta adds a replacement DV for.
// validateRemovedDeletes requires each to be among the removals, so a
// delta cannot commit a replacement while leaving the superseded DV
// live. This piggybacks on the manifest walk the removals already
// need; deltas without removals do not pay for it (nor get it — see
// AddDeletes).
func (rd *RowDelta) resolveRemovedDeletes(fs iceio.IO, meta *MetadataBuilder) (resolved, replacedLive []iceberg.DataFile, _ error) {
	snap := rd.txn.planningSnapshot(meta)
	if snap == nil {
		return nil, nil, errors.New("cannot remove delete files from a table without an existing snapshot")
	}

	want := make(map[string]struct{}, len(rd.removedDels))
	for _, f := range rd.removedDels {
		want[f.FilePath()] = struct{}{}
	}

	addedRefs := make(map[string]struct{}, len(rd.delFiles))
	for _, f := range rd.delFiles {
		if ref := explicitReferencedDataFile(f); IsDeletionVector(f) && ref != "" {
			addedRefs[ref] = struct{}{}
		}
	}

	liveByPath := make(map[string][]iceberg.DataFile, len(rd.removedDels))
	for entry, err := range snap.entries(fs, iceberg.ManifestContentDeletes, true) {
		if err != nil {
			return nil, nil, err
		}
		df := entry.DataFile()
		if _, ok := want[df.FilePath()]; ok {
			liveByPath[df.FilePath()] = append(liveByPath[df.FilePath()], df)
		}
		if ref := explicitReferencedDataFile(df); IsDeletionVector(df) && ref != "" {
			if _, ok := addedRefs[ref]; ok {
				replacedLive = append(replacedLive, df)
			}
		}
	}

	seenKeys := make(map[pathRefKey]struct{}, len(rd.removedDels))
	seenRefs := make(map[string]string, len(rd.removedDels)) // live ref → live path
	resolved = make([]iceberg.DataFile, len(rd.removedDels))
	for i, f := range rd.removedDels {
		candidates := liveByPath[f.FilePath()]

		var live iceberg.DataFile
		switch {
		case len(candidates) == 0:
			return nil, nil, fmt.Errorf("cannot remove delete files that do not belong to the table: %s", f.FilePath())
		case len(candidates) == 1:
			live = candidates[0]
		default:
			ref := explicitReferencedDataFile(f)
			if ref == "" {
				return nil, nil, fmt.Errorf("ambiguous removal: %d live delete entries share path %s; the removed file must declare a referenced data file to identify one",
					len(candidates), f.FilePath())
			}
			matches := 0
			for _, c := range candidates {
				if explicitReferencedDataFile(c) == ref {
					if live == nil {
						live = c
					}
					matches++
				}
			}
			switch {
			case matches == 0:
				return nil, nil, fmt.Errorf("cannot remove delete file %s: no live delete entry references data file %s",
					f.FilePath(), ref)
			case matches > 1:
				return nil, nil, fmt.Errorf("found %d live delete entries at %s referencing data file %s; the table has duplicate live deletion vectors for one data file",
					matches, f.FilePath(), ref)
			}
		}

		liveRef := explicitReferencedDataFile(live)
		key := pathRefKey{path: live.FilePath(), ref: liveRef}
		if _, ok := seenKeys[key]; ok {
			return nil, nil, fmt.Errorf("removed delete files must be unique: %s (referenced data file %q)",
				live.FilePath(), liveRef)
		}
		seenKeys[key] = struct{}{}
		// The producer keys DV removals by referenced data file, so two
		// distinct live entries sharing a ref (a spec violation: two
		// live DVs on one data file) would silently collapse to one
		// removal. Surface the corruption instead. Non-DV entries (ref
		// "") are rejected by validateRemovedDeletes with a clearer
		// error, so they are exempt here.
		if liveRef != "" {
			if prevPath, ok := seenRefs[liveRef]; ok {
				return nil, nil, fmt.Errorf("removed delete files %s and %s both reference data file %q; the table has two live deletion vectors for one data file",
					prevPath, live.FilePath(), liveRef)
			}
			seenRefs[liveRef] = live.FilePath()
		}
		resolved[i] = live
	}

	return resolved, replacedLive, nil
}

// validate is the client-side conflict check for a RowDelta commit. It
// runs on every commit attempt: on attempt 0 cc's base and current
// state coincide (nothing concurrent to check), and on retries cc
// reflects the freshly refreshed branch state. Two invariants are
// enforced:
//
//   - Every data file referenced by a position-delete in this RowDelta
//     must still be reachable from the branch head. A concurrent
//     compaction or overwrite that rewrote a referenced file would
//     orphan this pos-delete and produce incorrect results — reject.
//     Always runs, no isolation gating.
//
//   - When any equality-delete is included and isolation is
//     SERIALIZABLE, reject the commit if a concurrent snapshot added
//     conflicting data files. For unpartitioned tables the check is
//     conservative (AlwaysTrue — any concurrent append is a conflict).
//     For partitioned tables, an OR-of-equalities filter is built from
//     the eq-delete files' partition tuples and routed through
//     validateAddedDataFilesMatchingFilter, which performs per-spec
//     projection (spec-evolution safe), manifest-summary pruning, and
//     type-aware partition evaluation — so only concurrent data files
//     in the same partitions as the equality deletes are rejected.
//
// Fast appends alongside a RowDelta see no validators from RowDelta:
// data-only commits are as safe as a fastAppend.
func (rd *RowDelta) validate(cc *conflictContext) error {
	meta, err := rd.txn.txnMeta()
	if err != nil {
		return err
	}

	level, err := readIsolationLevel(meta.props,
		WriteDeleteIsolationLevelKey, WriteDeleteIsolationLevelDefault)
	if err != nil {
		return err
	}

	// Collect every data-file path the pos-deletes in this delta
	// reference. A nil ReferencedDataFile means the pos-delete does
	// not record its target — we cannot check it here; the file is
	// still present in the per-row position_delete_file column and
	// would apply correctly regardless of concurrent removals,
	// matching Java's behavior when the referenced-file column is
	// unset.
	var referenced []string
	var eqDeleteFiles []iceberg.DataFile
	for _, f := range rd.delFiles {
		switch f.ContentType() {
		case iceberg.EntryContentPosDeletes:
			if ref := iceberginternal.BorrowedDataFileReferencedDataFile(f); ref != nil && *ref != "" {
				referenced = append(referenced, *ref)
			}
		case iceberg.EntryContentEqDeletes:
			eqDeleteFiles = append(eqDeleteFiles, f)
		}
	}

	if cc == nil {
		return nil
	}

	if len(referenced) > 0 {
		if err := validateDataFilesExist(cc, referenced); err != nil {
			return err
		}
	}

	if len(eqDeleteFiles) > 0 {
		// Route through the existing validateNoConflictingDataFiles path,
		// which calls validateAddedDataFilesMatchingFilter internally.
		// For unpartitioned tables, use AlwaysTrue conservatively — an
		// equality delete can affect any row. For partitioned tables,
		// build an OR-of-equalities filter from the eq-delete files'
		// partition tuples so that concurrent appends to different
		// partitions are not falsely rejected.
		currentSpec, specErr := meta.CurrentSpec()
		if specErr != nil {
			return fmt.Errorf("reading current partition spec: %w", specErr)
		}

		var conflictErr error
		if currentSpec == nil || currentSpec.NumFields() == 0 {
			conflictErr = validateNoConflictingDataFiles(cc, iceberg.AlwaysTrue{}, level)
		} else {
			conflictErr = validateNoConflictingDataFilesInPartitions(cc, eqDeleteFiles, level)
		}
		if conflictErr != nil {
			return conflictErr
		}
	}

	return nil
}

// Operation returns the snapshot operation type that will be used when
// this RowDelta is committed:
//   - data only → OpAppend
//   - deletes only → OpDelete
//   - both → OpOverwrite
func (rd *RowDelta) Operation() Operation {
	hasData := len(rd.dataFiles) > 0
	hasDeletes := len(rd.delFiles) > 0

	switch {
	case hasData && hasDeletes:
		return OpOverwrite
	case hasDeletes:
		return OpDelete
	default:
		return OpAppend
	}
}
