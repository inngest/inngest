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
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"log"
	"maps"
	"runtime"
	"slices"
	"sync/atomic"
	"time"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/config"
	"github.com/apache/iceberg-go/internal"
	iceio "github.com/apache/iceberg-go/io"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type producerImpl interface {
	// to perform any post-processing on the manifests before writing them
	// to the new snapshot. This will be called as the last step
	// before writing a manifest list file, using the result of this function
	// as the final list of manifests to write.
	processManifests(manifests []iceberg.ManifestFile) ([]iceberg.ManifestFile, error)
	// parentDependentManifests builds the manifests inherited from parent that
	// this producer needs to carry into the new snapshot. It is re-evaluated
	// against the fresh parent on every OCC retry, so it must read parent rather
	// than the transaction's (stale) base snapshot.
	parentDependentManifests(ctx context.Context, parent *Snapshot) ([]iceberg.ManifestFile, error)
	// validate runs producer-specific conflict checks against the
	// current catalog state. Implementations should return a wrapped
	// ErrCommit* sentinel on conflict, ErrCommitDiverged on terminal
	// divergence, or nil on success. A no-op default is fine for
	// producers that are safe against concurrent appends (fast-append
	// and merge-append).
	validate(cc *conflictContext) error
	// needsValidation reports whether this producer's validate method
	// performs real conflict checks. Return false only if validate is
	// unconditionally a no-op; commit() skips validator registration
	// entirely when this returns false, so validate will never run.
	needsValidation() bool
}

// supersededAccumulator marks a producer that writes merged manifests OCC
// retries orphan. Only rewrite implements it.
type supersededAccumulator interface {
	// supersededManifests returns paths of merged manifests to delete once the
	// commit resolves. When committed is false the commit never landed, so the
	// final attempt's manifests are orphaned too and are included.
	supersededManifests(committed bool) []string
}

// retryManifestRebuilder lets a producer reuse parent-independent manifest
// output when a retry's fresh parent still contains every manifest it rewrote.
// Producers that do not implement it fall back to assembling and processing
// the complete manifest set again.
type retryManifestRebuilder interface {
	rebuildManifests(ctx context.Context, parent *Snapshot, addedContent []iceberg.ManifestFile) ([]iceberg.ManifestFile, error)
}

func newManifestFileName(num int, commit uuid.UUID) string {
	return fmt.Sprintf("%s-m%d.avro", commit, num)
}

func newManifestListFileName(snapshotID int64, attempt int, commit uuid.UUID) string {
	// mimics behavior of java
	// https://github.com/apache/iceberg/blob/c862b9177af8e2d83122220764a056f3b96fd00c/core/src/main/java/org/apache/iceberg/SnapshotProducer.java#L491
	return fmt.Sprintf("snap-%d-%d-%s.avro", snapshotID, attempt, commit)
}

func newFastAppendFilesProducer(op Operation, txn *Transaction, fs iceio.WriteFileIO, commitUUID *uuid.UUID, snapshotProps iceberg.Properties) *snapshotProducer {
	prod := createSnapshotProducer(op, txn, fs, commitUUID, snapshotProps)
	prod.producerImpl = &fastAppendFiles{base: prod}

	return prod
}

type fastAppendFiles struct {
	base *snapshotProducer
}

func (fa *fastAppendFiles) processManifests(manifests []iceberg.ManifestFile) ([]iceberg.ManifestFile, error) {
	return manifests, nil
}

func (fa *fastAppendFiles) parentDependentManifests(_ context.Context, parent *Snapshot) ([]iceberg.ManifestFile, error) {
	if parent == nil {
		return nil, nil
	}

	return parent.Manifests(fa.base.io)
}

// validate is a no-op for fastAppendFiles: appends are commutative
// per the Iceberg spec, and Java's BaseFastAppend / SnapshotProducer
// likewise skip pre-commit validation for the append path. Two
// concurrent fast-appends of distinct files merge cleanly into the
// resulting manifest list. Two concurrent fast-appends adding the
// same file path is a writer-side error (paths are expected to be
// unique, normally via UUID-stamped filenames); detecting it here
// is out of Java parity scope.
func (fa *fastAppendFiles) validate(_ *conflictContext) error {
	return nil
}

func (fa *fastAppendFiles) needsValidation() bool { return false }

type overwriteFiles struct {
	base *snapshotProducer

	// manifestConcurrency bounds concurrent filtering and rewriting of
	// inherited manifests. Zero uses the process-wide worker configuration.
	manifestConcurrency int

	// filter is the row-level predicate the caller declared for this
	// overwrite (e.g. WithOverwriteFilter). Nil or AlwaysTrue means
	// "overwrite everything" — any concurrent added data file is a
	// conflict under serializable isolation. validate() reads this to
	// decide whether to run filter-bounded or full-table checks.
	filter iceberg.BooleanExpression

	// skipDefaultValidator disables the overwrite's pre-commit
	// conflict check when the producer is driven by a higher-level
	// operation that registers its own validator (e.g. compaction via
	// RewriteDataFiles). Without this flag a compaction would falsely
	// reject a concurrent append into the same partition, which is
	// semantically compatible with a rewrite.
	skipDefaultValidator bool
}

type overwriteManifestScan struct {
	notDeleted []iceberg.ManifestEntry
	deleted    []iceberg.ManifestEntry
}

type overwriteParentManifests struct {
	existing  []iceberg.ManifestFile
	deleted   []iceberg.ManifestEntry
	rewritten []iceberg.ManifestFile
}

func newOverwriteFilesProducer(op Operation, txn *Transaction, fs iceio.WriteFileIO, commitUUID *uuid.UUID, snapshotProps iceberg.Properties) *snapshotProducer {
	prod := createSnapshotProducer(op, txn, fs, commitUUID, snapshotProps)
	prod.producerImpl = &overwriteFiles{base: prod}

	return prod
}

// The overwrite configuration setters keep overwrite-only details out of
// transaction call sites. They are intentionally no-ops for another producer
// implementation so callers do not depend on its concrete representation.
func (sp *snapshotProducer) setOverwriteFilter(filter iceberg.BooleanExpression) {
	if filter == nil {
		return
	}

	if of, ok := sp.producerImpl.(*overwriteFiles); ok {
		of.filter = filter
	}
}

func (sp *snapshotProducer) setManifestConcurrency(concurrency int) {
	if of, ok := sp.producerImpl.(*overwriteFiles); ok {
		of.manifestConcurrency = concurrency
	}
}

func (sp *snapshotProducer) setSkipDefaultValidator(skip bool) {
	if of, ok := sp.producerImpl.(*overwriteFiles); ok {
		of.skipDefaultValidator = skip
	}
}

func (of *overwriteFiles) processManifests(manifests []iceberg.ManifestFile) ([]iceberg.ManifestFile, error) {
	// no post processing
	return manifests, nil
}

func (of *overwriteFiles) parentDependentManifests(
	ctx context.Context, parent *Snapshot,
) ([]iceberg.ManifestFile, error) {
	parentManifests, err := of.filterAndRewriteParentManifests(ctx, parent)
	if err != nil {
		return nil, err
	}

	// Keep tombstone writes after the scan/rewrite pass so a failure here can
	// remove every replacement manifest produced by that pass in one place.
	deletedManifests, err := of.base.writeDeletedEntries(ctx, parentManifests.deleted)
	if err != nil {
		if cleanupErr := of.base.cleanupGeneratedManifests(parentManifests.rewritten); cleanupErr != nil {
			return nil, errors.Join(err, cleanupErr)
		}

		return nil, err
	}

	return slices.Concat(deletedManifests, parentManifests.existing), nil
}

func (of *overwriteFiles) manifestWorkerCount() int {
	concurrency := of.manifestConcurrency
	if concurrency <= 0 {
		concurrency = config.EnvConfig.MaxWorkers
	}

	return manifestMergeConcurrencyLimit(concurrency)
}

func (of *overwriteFiles) scanManifest(ctx context.Context, m iceberg.ManifestFile) (overwriteManifestScan, error) {
	if err := context.Cause(ctx); err != nil {
		return overwriteManifestScan{}, err
	}

	// Counts may be -1 (unset) on V1 manifests, so clamp before allocating.
	capacity := int(m.AddedDataFiles()) + int(m.ExistingDataFiles())
	scan := overwriteManifestScan{
		notDeleted: make([]iceberg.ManifestEntry, 0, max(0, capacity)),
	}
	for entry, err := range of.base.iterManifestEntries(m, true) {
		if err != nil {
			return overwriteManifestScan{}, err
		}
		if err := context.Cause(ctx); err != nil {
			return overwriteManifestScan{}, err
		}

		path := entry.DataFile().FilePath()
		content := entry.DataFile().ContentType()
		_, isDeletedData := of.base.deletedFiles[path]
		isDeletedDelete := of.base.deleteFileRemoved(entry.DataFile())

		isData := content == iceberg.EntryContentData
		matched := (isDeletedData && isData) || (isDeletedDelete && !isData)
		if matched {
			seqNum := entry.SequenceNum()
			scan.deleted = append(scan.deleted,
				iceberg.NewManifestEntry(iceberg.EntryStatusDELETED,
					&of.base.snapshotID, &seqNum, entry.FileSequenceNum(),
					entry.DataFile()))

			continue
		}

		scan.notDeleted = append(scan.notDeleted, entry)
	}

	return scan, nil
}

// rewriteManifest writes the surviving entries from an inherited manifest.
// The output is removed on any failure so a cancelled or failed worker cannot
// leave a partial object behind.
func (of *overwriteFiles) rewriteManifest(
	ctx context.Context,
	m iceberg.ManifestFile,
	notDeleted []iceberg.ManifestEntry,
) (_ iceberg.ManifestFile, retErr error) {
	spec, err := of.base.txn.meta.GetSpecByID(int(m.PartitionSpecID()))
	if err != nil {
		return nil, err
	}

	wr, path, counter, fileCloser, err := of.base.newManifestWriter(*spec, iceberg.WithManifestWriterContent(m.ManifestContent()))
	if err != nil {
		return nil, err
	}
	defer func() {
		if retErr == nil {
			return
		}
		if cleanupErr := of.base.io.Remove(path); cleanupErr != nil {
			retErr = errors.Join(retErr, fmt.Errorf("remove failed manifest %s: %w", path, cleanupErr))
		}
	}()
	defer internal.CheckedClose(fileCloser, &retErr)
	writerClosed := false
	defer func() {
		if !writerClosed {
			internal.CheckedClose(wr, &retErr)
		}
	}()

	for _, entry := range notDeleted {
		if err := context.Cause(ctx); err != nil {
			return nil, err
		}
		if err := wr.Existing(entry); err != nil {
			return nil, err
		}
	}
	if err := context.Cause(ctx); err != nil {
		return nil, err
	}

	// close the writer to force a flush and ensure counter.Count is accurate
	writerClosed = true
	if err := wr.Close(); err != nil {
		return nil, err
	}

	return wr.ToManifestFile(path, counter.Count, iceberg.WithManifestFileContent(m.ManifestContent()))
}

func (of *overwriteFiles) cleanupFilteredManifests(input []iceberg.ManifestFile, results [][]iceberg.ManifestFile) error {
	inputPaths := make(map[string]struct{}, len(input))
	for _, manifest := range input {
		inputPaths[manifest.FilePath()] = struct{}{}
	}

	seen := make(map[string]struct{})
	var cleanupErr error
	for _, manifests := range results {
		for _, manifest := range manifests {
			path := manifest.FilePath()
			if _, ok := inputPaths[path]; ok {
				continue
			}
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			if err := of.base.io.Remove(path); err != nil {
				cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove filtered manifest %s: %w", path, err))
			}
		}
	}

	return cleanupErr
}

// filterAndRewriteParentManifests scans each inherited manifest once, using the
// same pass to collect tombstones and rewrite surviving entries. The old
// producer path needed one full manifest read for each of those operations.
func (of *overwriteFiles) filterAndRewriteParentManifests(
	ctx context.Context,
	parent *Snapshot,
) (overwriteParentManifests, error) {
	result := overwriteParentManifests{}
	if parent == nil {
		return result, nil
	}

	manifestList, err := parent.Manifests(of.base.io)
	if err != nil {
		return result, err
	}

	manifestResults := make([][]iceberg.ManifestFile, len(manifestList))
	scanResults := make([]overwriteManifestScan, len(manifestList))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(min(of.manifestWorkerCount(), len(manifestList)))
	for index, manifest := range manifestList {
		g.Go(func() error {
			scan, err := of.scanManifest(gctx, manifest)
			if err != nil {
				return err
			}
			scanResults[index] = scan

			if len(scan.notDeleted) == 0 {
				return nil
			}
			if len(scan.deleted) == 0 {
				manifestResults[index] = []iceberg.ManifestFile{manifest}

				return nil
			}

			filtered, err := of.rewriteManifest(gctx, manifest, scan.notDeleted)
			if err != nil {
				return err
			}
			manifestResults[index] = []iceberg.ManifestFile{filtered}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		if cleanupErr := of.cleanupFilteredManifests(manifestList, manifestResults); cleanupErr != nil {
			return result, errors.Join(err, cleanupErr)
		}

		return result, err
	}

	result.existing = make([]iceberg.ManifestFile, 0, len(manifestList))
	result.deleted = make([]iceberg.ManifestEntry, 0, len(manifestList))
	for index, manifests := range manifestResults {
		result.existing = append(result.existing, manifests...)
		result.deleted = append(result.deleted, scanResults[index].deleted...)
		if len(scanResults[index].deleted) > 0 && len(scanResults[index].notDeleted) > 0 {
			result.rewritten = append(result.rewritten, manifests...)
		}
	}

	return result, nil
}

func (sp *snapshotProducer) cleanupGeneratedManifests(manifests []iceberg.ManifestFile) error {
	var cleanupErr error
	for _, manifest := range manifests {
		if err := sp.io.Remove(manifest.FilePath()); err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove generated manifest %s: %w", manifest.FilePath(), err))
		}
	}

	return cleanupErr
}

// validate rejects the overwrite if a concurrent commit added data
// files that fall inside the committer's filter region.
//
// Isolation level is read per-operation: write.delete.isolation-level
// for OpDelete (mirroring Java's BaseDeleteFiles), write.update.iso-
// lation-level otherwise. SNAPSHOT returns nil — concurrent appends
// are allowed. SERIALIZABLE runs validateAddedDataFilesMatchingFilter
// against the committer's filter (AlwaysTrue when no filter is set).
func (of *overwriteFiles) validate(cc *conflictContext) error {
	// Delete operations (copy-on-write / merge-on-read deletes that
	// run through overwriteFiles) must read write.delete.isolation-
	// level, not write.update.isolation-level. Java's BaseDeleteFiles
	// makes the same split. Any other op (Overwrite, Replace) reads
	// the update key.
	key, defVal := WriteUpdateIsolationLevelKey, WriteUpdateIsolationLevelDefault
	if of.base.op == OpDelete {
		key, defVal = WriteDeleteIsolationLevelKey, WriteDeleteIsolationLevelDefault
	}
	level, err := readIsolationLevel(of.base.txn.meta.props, key, defVal)
	if err != nil {
		return err
	}
	if cc == nil || of.skipDefaultValidator {
		return nil
	}
	if level != IsolationSerializable {
		// SNAPSHOT isolation allows concurrent appends into the
		// filter region. No further checks on this path.
		return nil
	}

	filter := of.filter
	if filter == nil {
		filter = iceberg.AlwaysTrue{}
	}

	return validateAddedDataFilesMatchingFilter(cc, filter)
}

func (of *overwriteFiles) needsValidation() bool { return true }

type manifestMergeManager struct {
	targetSizeBytes  int64
	minCountToMerge  int
	mergeEnabled     bool
	mergeConcurrency int
	clusterBy        func(iceberg.DataFile) any
	snap             *snapshotProducer
}

func manifestMergeConcurrencyLimit(configured int) int {
	if configured <= 0 {
		return runtime.GOMAXPROCS(0)
	}

	return configured
}

func (m *manifestMergeManager) groupBySpec(manifests []iceberg.ManifestFile) map[int][]iceberg.ManifestFile {
	groups := make(map[int][]iceberg.ManifestFile)
	for _, m := range manifests {
		specid := int(m.PartitionSpecID())
		group := groups[specid]
		groups[specid] = append(group, m)
	}

	return groups
}

func (m *manifestMergeManager) createManifest(specID int, bin []iceberg.ManifestFile) (mf iceberg.ManifestFile, err error) {
	spec, err := m.snap.spec(specID)
	if err != nil {
		return nil, err
	}

	wr, path, counter, fileCloser, err := m.snap.newManifestWriter(spec)
	if err != nil {
		return nil, err
	}
	defer internal.CheckedClose(fileCloser, &err)
	writerClosed := false
	defer func() {
		if !writerClosed {
			internal.CheckedClose(wr, &err)
		}
	}()

	for _, manifest := range bin {
		for entry, err := range m.snap.iterManifestEntries(manifest, false) {
			if err != nil {
				return nil, err
			}

			switch {
			case entry.Status() == iceberg.EntryStatusDELETED && entry.SnapshotID() == m.snap.snapshotID:
				// only files deleted by this snapshot should be added to the new manifest
				err = wr.Delete(entry)
			case entry.Status() == iceberg.EntryStatusADDED && entry.SnapshotID() == m.snap.snapshotID:
				// added entries from this snapshot are still added, otherwise they should be existing
				err = wr.Add(entry)
			case entry.Status() != iceberg.EntryStatusDELETED:
				// add all non-deleted files from the old manifest as existing files
				err = wr.Existing(entry)
			}

			if err != nil {
				return nil, err
			}
		}
	}

	// close the writer to force a flush and ensure counter.Count is accurate
	writerClosed = true
	if err := wr.Close(); err != nil {
		return nil, err
	}

	return wr.ToManifestFile(path, counter.Count)
}

func (m *manifestMergeManager) mergeGroup(firstManifest iceberg.ManifestFile, specID int, manifests []iceberg.ManifestFile) ([]iceberg.ManifestFile, error) {
	packer := internal.SlicePacker[iceberg.ManifestFile]{
		TargetWeight:    m.targetSizeBytes,
		Lookback:        1,
		LargestBinFirst: false,
	}
	bins := packer.PackEnd(manifests, func(m iceberg.ManifestFile) int64 {
		return m.Length()
	})

	mergeBin := func(bin []iceberg.ManifestFile) ([]iceberg.ManifestFile, error) {
		output := make([]iceberg.ManifestFile, 0, 1)
		if len(bin) == 1 {
			output = append(output, bin[0])
		} else if len(bin) < m.minCountToMerge && slices.ContainsFunc(bin, func(m iceberg.ManifestFile) bool { return m == firstManifest }) {
			// if the bin has the first manifest (the new data files or an appended
			// manifest file) then only merge it if the number of manifests is above
			// the minimum count. this is applied only to bins with an in-memory manifest
			// so that large manifests don't prevent merging older groups
			output = append(output, bin...)
		} else {
			created, err := m.createManifest(specID, bin)
			if err != nil {
				return nil, err
			}
			output = append(output, created)
		}

		return output, nil
	}

	binResults := make([][]iceberg.ManifestFile, len(bins))
	g := errgroup.Group{}
	g.SetLimit(manifestMergeConcurrencyLimit(m.mergeConcurrency))
	for i, bin := range bins {
		g.Go(func() error {
			var err error
			binResults[i], err = mergeBin(bin)

			return err
		})
	}

	if err := g.Wait(); err != nil {
		// Delete bins that finished before this one failed.
		m.removeOrphans(manifests, slices.Concat(binResults...))

		return nil, err
	}

	return slices.Concat(binResults...), nil
}

// removeOrphans deletes merged manifests a failed merge left behind. Pass-through
// bins reuse an input path and are kept; only outputs not in input are removed.
func (m *manifestMergeManager) removeOrphans(input, output []iceberg.ManifestFile) {
	inPaths := make(map[string]struct{}, len(input))
	for _, mf := range input {
		inPaths[mf.FilePath()] = struct{}{}
	}
	for _, mf := range output {
		if _, ok := inPaths[mf.FilePath()]; ok {
			continue
		}
		if err := m.snap.io.Remove(mf.FilePath()); err != nil {
			log.Printf("Warning: failed to delete orphaned merged manifest %s: %v", mf.FilePath(), err)
		}
	}
}

func (m *manifestMergeManager) mergeManifests(manifests []iceberg.ManifestFile) ([]iceberg.ManifestFile, error) {
	if !m.mergeEnabled || len(manifests) == 0 {
		return manifests, nil
	}
	if m.clusterBy != nil {
		// Clustering replaces size-only bin-packing, including minCountToMerge.
		return m.clusterManifests(manifests)
	}

	first := manifests[0]
	groups := m.groupBySpec(manifests)

	merged := make([]iceberg.ManifestFile, 0, len(groups))
	for _, specID := range slices.Backward(slices.Sorted(maps.Keys(groups))) {
		groupMerged, err := m.mergeGroup(first, specID, groups[specID])
		if err != nil {
			// This group cleaned up its own writes; drop earlier groups' too.
			m.removeOrphans(manifests, merged)

			return nil, err
		}

		merged = append(merged, groupMerged...)
	}

	return merged, nil
}

type mergeAppendFiles struct {
	fastAppendFiles

	targetSizeBytes  int64
	minCountToMerge  int
	mergeEnabled     bool
	mergeConcurrency int
}

func newMergeAppendFilesProducer(op Operation, txn *Transaction, fs iceio.WriteFileIO, commitUUID *uuid.UUID, snapshotProps iceberg.Properties) *snapshotProducer {
	prod := createSnapshotProducer(op, txn, fs, commitUUID, snapshotProps)
	prod.producerImpl = &mergeAppendFiles{
		fastAppendFiles: fastAppendFiles{base: prod},
		targetSizeBytes: int64(txn.meta.props.GetInt(ManifestTargetSizeBytesKey, ManifestTargetSizeBytesDefault)),
		minCountToMerge: txn.meta.props.GetInt(ManifestMinMergeCountKey, ManifestMinMergeCountDefault),
		mergeEnabled:    txn.meta.props.GetBool(ManifestMergeEnabledKey, ManifestMergeEnabledDefault),
		mergeConcurrency: manifestMergeConcurrencyLimit(
			txn.meta.props.GetInt(ManifestMergeMaxConcurrencyKey, ManifestMergeMaxConcurrencyDefault)),
	}

	return prod
}

func (m *mergeAppendFiles) processManifests(manifests []iceberg.ManifestFile) ([]iceberg.ManifestFile, error) {
	unmergedDataManifests, unmergedDeleteManifests := []iceberg.ManifestFile{}, []iceberg.ManifestFile{}
	for _, m := range manifests {
		if m.ManifestContent() == iceberg.ManifestContentData {
			unmergedDataManifests = append(unmergedDataManifests, m)
		} else if m.ManifestContent() == iceberg.ManifestContentDeletes {
			unmergedDeleteManifests = append(unmergedDeleteManifests, m)
		}
	}

	dataManifestMergeMgr := manifestMergeManager{
		targetSizeBytes:  m.targetSizeBytes,
		minCountToMerge:  m.minCountToMerge,
		mergeEnabled:     m.mergeEnabled,
		mergeConcurrency: m.mergeConcurrency,
		snap:             m.base,
	}

	result, err := dataManifestMergeMgr.mergeManifests(unmergedDataManifests)
	if err != nil {
		return nil, err
	}

	return append(result, unmergedDeleteManifests...), nil
}

func (m *mergeAppendFiles) needsValidation() bool { return false }

type snapshotProducer struct {
	producerImpl

	commitUuid              uuid.UUID
	io                      iceio.WriteFileIO
	txn                     *Transaction
	op                      Operation
	snapshotID              int64
	parentSnapshotID        int64
	addedFiles              []iceberg.DataFile
	addedDataSequenceNumber *int64
	addedDeleteFiles        []deleteFileAddition
	manifestCount           atomic.Int32
	deletedFiles            map[string]iceberg.DataFile
	deletedDeleteFiles      map[string]iceberg.DataFile
	deletedDVsByRef         map[string]iceberg.DataFile
	snapshotProps           iceberg.Properties
}

// deleteFileAddition carries the data sequence number of a rewritten delete
// file. A nil sequence means the file is a normal delete added by the current
// snapshot and should inherit that snapshot's sequence number. Rewrite paths
// use an explicit sequence so replacing delete files cannot change which data
// files the delete applies to.
type deleteFileAddition struct {
	file               iceberg.DataFile
	dataSequenceNumber *int64
}

func createSnapshotProducer(op Operation, txn *Transaction, fs iceio.WriteFileIO, commitUUID *uuid.UUID, snapshotProps iceberg.Properties) *snapshotProducer {
	var (
		commit         uuid.UUID
		parentSnapshot int64 = -1
	)

	if commitUUID == nil {
		commit = uuid.New()
	} else {
		commit = *commitUUID
	}

	if snap := txn.meta.currentSnapshotForRef(txn.branch); snap != nil {
		parentSnapshot = snap.SnapshotID
	}

	return &snapshotProducer{
		commitUuid:         commit,
		io:                 fs,
		txn:                txn,
		op:                 op,
		snapshotID:         txn.meta.newSnapshotID(),
		parentSnapshotID:   parentSnapshot,
		addedFiles:         []iceberg.DataFile{},
		deletedFiles:       make(map[string]iceberg.DataFile),
		deletedDeleteFiles: make(map[string]iceberg.DataFile),
		deletedDVsByRef:    make(map[string]iceberg.DataFile),
		snapshotProps:      snapshotProps,
	}
}

func (sp *snapshotProducer) spec(id int) (iceberg.PartitionSpec, error) {
	spec, err := sp.txn.meta.GetSpecByID(id)
	if err != nil {
		return iceberg.PartitionSpec{}, err
	}

	return *spec, nil
}

func (sp *snapshotProducer) appendDataFile(df iceberg.DataFile) *snapshotProducer {
	sp.addedFiles = append(sp.addedFiles, df)

	return sp
}

func (sp *snapshotProducer) setNewDataFilesDataSequenceNumber(seq int64) *snapshotProducer {
	seqCopy := seq
	sp.addedDataSequenceNumber = &seqCopy

	return sp
}

func (sp *snapshotProducer) appendDeleteFile(df iceberg.DataFile) *snapshotProducer {
	sp.addedDeleteFiles = append(sp.addedDeleteFiles, deleteFileAddition{file: df})

	return sp
}

func (sp *snapshotProducer) appendDeleteFileWithDataSequenceNumber(df iceberg.DataFile, seq int64) *snapshotProducer {
	sp.addedDeleteFiles = append(sp.addedDeleteFiles, deleteFileAddition{
		file:               df,
		dataSequenceNumber: &seq,
	})

	return sp
}

func (sp *snapshotProducer) deleteDataFile(df iceberg.DataFile) *snapshotProducer {
	sp.deletedFiles[df.FilePath()] = df

	return sp
}

func (sp *snapshotProducer) removeDeleteFile(df iceberg.DataFile) *snapshotProducer {
	sp.deletedDeleteFiles[df.FilePath()] = df

	return sp
}

func (sp *snapshotProducer) removeDeletionVector(df iceberg.DataFile) *snapshotProducer {
	ref := internal.BorrowedDataFileReferencedDataFile(df)
	if ref == nil {
		return sp
	}
	sp.deletedDVsByRef[*ref] = df

	return sp
}

// deleteFileRemoved reports whether a delete-file entry is being expunged in
// this snapshot: position/equality deletes match by path, deletion vectors by
// (referenced data file, path) pair. Path alone would over-remove live
// siblings (one Puffin holds blobs for several data files); ref alone would
// expunge a peer's replacement DV for the same data file on an OCC retry,
// silently discarding the peer's newer deletes.
func (sp *snapshotProducer) deleteFileRemoved(df iceberg.DataFile) bool {
	if _, ok := sp.deletedDeleteFiles[df.FilePath()]; ok {
		return true
	}
	if ref := internal.BorrowedDataFileReferencedDataFile(df); IsDeletionVector(df) && ref != nil {
		want, ok := sp.deletedDVsByRef[*ref]

		return ok && want.FilePath() == df.FilePath()
	}

	return false
}

func (sp *snapshotProducer) newManifestWriter(spec iceberg.PartitionSpec, opts ...iceberg.ManifestWriterOption) (_ *iceberg.ManifestWriter, _ string, _ *internal.CountingWriter, _ io.Closer, err error) {
	out, path, err := sp.newManifestOutput()
	if err != nil {
		return nil, "", nil, nil, err
	}

	counter := &internal.CountingWriter{W: out}
	wr, err := iceberg.NewManifestWriter(sp.txn.meta.formatVersion, counter, spec,
		sp.txn.meta.CurrentSchema(), sp.snapshotID, opts...)
	if err != nil {
		return nil, "", nil, nil, errors.Join(err, out.Close(), sp.io.Remove(path))
	}

	return wr, path, counter, out, nil
}

func (sp *snapshotProducer) newManifestOutput() (io.WriteCloser, string, error) {
	provider, err := sp.txn.tbl.LocationProvider()
	if err != nil {
		return nil, "", err
	}
	fname := newManifestFileName(int(sp.manifestCount.Add(1)), sp.commitUuid)
	filepath := provider.NewMetadataLocation(fname)
	f, err := sp.io.Create(filepath)
	if err != nil {
		return nil, "", fmt.Errorf("could not create manifest file: %w", err)
	}

	return f, filepath, nil
}

func (sp *snapshotProducer) iterManifestEntries(m iceberg.ManifestFile, discardDeleted bool) iter.Seq2[iceberg.ManifestEntry, error] {
	return m.Entries(sp.io, discardDeleted)
}

// parentSnapshot resolves the snapshot this producer is layered on (the branch
// head captured when the producer was built), or nil for the first snapshot.
func (sp *snapshotProducer) parentSnapshot() (*Snapshot, error) {
	if sp.parentSnapshotID <= 0 {
		return nil, nil
	}

	return sp.txn.meta.SnapshotByID(sp.parentSnapshotID)
}

func (sp *snapshotProducer) manifests(ctx context.Context) ([]iceberg.ManifestFile, error) {
	parent, err := sp.parentSnapshot()
	if err != nil {
		return nil, err
	}

	all, _, err := sp.buildManifests(ctx, parent)

	return all, err
}

// buildManifests assembles the new snapshot's full manifest set layered on
// parent and separately returns the parent-independent added-content manifests
// so the caller can reuse them across OCC retries. The parent-dependent portion
// (which reads the parent and evaluates overwrite removals) is built before
// any added-content writer is created, so a delete-side failure cannot orphan
// added-content writers.
func (sp *snapshotProducer) buildManifests(ctx context.Context, parent *Snapshot) (all, addedContent []iceberg.ManifestFile, err error) {
	if err := sp.validateSpecs(); err != nil {
		return nil, nil, err
	}

	dependent, err := sp.parentDependentManifests(ctx, parent)
	if err != nil {
		return nil, nil, err
	}

	addedContent, err = sp.addedContentManifests()
	if err != nil {
		return nil, nil, err
	}

	all, err = sp.processManifests(slices.Concat(addedContent, dependent))
	if err != nil {
		return nil, nil, err
	}

	return all, addedContent, nil
}

// addedContentManifests writes the manifests for this producer's newly added
// data and delete files. They do not depend on the parent snapshot, so they are
// written once and reused verbatim across OCC retries (rewriting them would
// churn manifest paths and orphan object-store files on every attempt).
func (sp *snapshotProducer) addedContentManifests() ([]iceberg.ManifestFile, error) {
	var g errgroup.Group

	addedManifests := make([]iceberg.ManifestFile, 0)
	positionDeleteManifests := make([]iceberg.ManifestFile, 0)

	if len(sp.addedFiles) > 0 {
		g.Go(sp.manifestProducer(iceberg.ManifestContentData, sp.addedFiles, &addedManifests))
	}

	if len(sp.addedDeleteFiles) > 0 {
		g.Go(sp.deleteManifestProducer(&positionDeleteManifests))
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return slices.Concat(addedManifests, positionDeleteManifests), nil
}

// validateSpecs resolves the spec ids of this producer's added and removed files up front.
// Manifests are written phase by phase, so rejecting an id mid-way would leave the earlier phase's manifests orphaned.
func (sp *snapshotProducer) validateSpecs() error {
	check := func(df iceberg.DataFile) error {
		if _, err := sp.spec(int(df.SpecID())); err != nil {
			return fmt.Errorf("file %s: %w", df.FilePath(), err)
		}

		return nil
	}

	for _, df := range sp.addedFiles {
		if err := check(df); err != nil {
			return err
		}
	}

	for _, addition := range sp.addedDeleteFiles {
		if err := check(addition.file); err != nil {
			return err
		}
	}

	for _, removed := range []map[string]iceberg.DataFile{sp.deletedFiles, sp.deletedDeleteFiles, sp.deletedDVsByRef} {
		for _, df := range removed {
			if err := check(df); err != nil {
				return err
			}
		}
	}

	return nil
}

func (sp *snapshotProducer) deleteManifestProducer(output *[]iceberg.ManifestFile) func() error {
	return func() error {
		groups := make(map[int][]deleteFileAddition)
		for _, addition := range sp.addedDeleteFiles {
			specID := int(addition.file.SpecID())
			groups[specID] = append(groups[specID], addition)
		}

		for _, specID := range slices.Sorted(maps.Keys(groups)) {
			additions := groups[specID]
			mf, err := sp.writeAddedDeleteManifest(specID, additions)
			if err != nil {
				return err
			}
			*output = append(*output, mf)
		}

		return nil
	}
}

// assembleManifests recomputes the parent-dependent manifests against parent and
// combines them with the already-written added-content manifests, then applies
// producer-specific post-processing (manifest merging for merge-append, a no-op
// otherwise). Used on OCC retries, where addedContent was written at attempt 0
// and is reused verbatim.
func (sp *snapshotProducer) assembleManifests(ctx context.Context, parent *Snapshot, addedContent []iceberg.ManifestFile) ([]iceberg.ManifestFile, error) {
	if err := sp.validateSpecs(); err != nil {
		return nil, err
	}

	dependent, err := sp.parentDependentManifests(ctx, parent)
	if err != nil {
		return nil, err
	}

	return sp.processManifests(slices.Concat(addedContent, dependent))
}

// parentDependentManifests builds the manifests whose contents depend on the
// parent the new snapshot is layered on: DELETE tombstone manifests for removed
// files, followed by the inherited/filtered existing manifests. It is recomputed
// against the fresh parent on every OCC retry so that a concurrent writer's
// files are inherited and the removed files are actually dropped — grafting the
// attempt-0 result onto a fresh parent would resurrect the removed files.
func (sp *snapshotProducer) parentDependentManifests(ctx context.Context, parent *Snapshot) (_ []iceberg.ManifestFile, err error) {
	return sp.producerImpl.parentDependentManifests(ctx, parent)
}

func (sp *snapshotProducer) writeDeletedEntries(ctx context.Context, deleted []iceberg.ManifestEntry) (_ []iceberg.ManifestFile, retErr error) {
	if len(deleted) == 0 {
		return nil, nil
	}

	// Group deleted entries by (specID, contentType) to ensure data and delete
	// file entries are written to separate manifests with the correct content.
	type groupKey struct {
		specID  int
		content iceberg.ManifestContent
	}
	groups := map[groupKey][]iceberg.ManifestEntry{}
	for _, entry := range deleted {
		content := iceberg.ManifestContentData
		if entry.DataFile().ContentType() != iceberg.EntryContentData {
			content = iceberg.ManifestContentDeletes
		}
		key := groupKey{specID: int(entry.DataFile().SpecID()), content: content}
		groups[key] = append(groups[key], entry)
	}

	writeGroup := func(key groupKey, entries []iceberg.ManifestEntry) (_ iceberg.ManifestFile, retErr error) {
		if err := context.Cause(ctx); err != nil {
			return nil, err
		}

		spec, err := sp.spec(key.specID)
		if err != nil {
			return nil, err
		}

		wr, path, counter, out, err := sp.newManifestWriter(spec,
			iceberg.WithManifestWriterContent(key.content))
		if err != nil {
			return nil, err
		}
		defer func() {
			if retErr == nil {
				return
			}
			if cleanupErr := sp.io.Remove(path); cleanupErr != nil {
				retErr = errors.Join(retErr, fmt.Errorf("remove failed delete manifest %s: %w", path, cleanupErr))
			}
		}()
		defer internal.CheckedClose(out, &retErr)

		writerClosed := false
		defer func() {
			if !writerClosed {
				internal.CheckedClose(wr, &retErr)
			}
		}()

		for _, entry := range entries {
			if err := context.Cause(ctx); err != nil {
				return nil, err
			}
			if err := wr.Delete(entry); err != nil {
				return nil, err
			}
		}

		writerClosed = true
		if err := wr.Close(); err != nil {
			return nil, err
		}

		return wr.ToManifestFile(path, counter.Count, iceberg.WithManifestFileContent(key.content))
	}

	keys := slices.SortedFunc(maps.Keys(groups), func(a, b groupKey) int {
		if c := cmp.Compare(a.specID, b.specID); c != 0 {
			return c
		}

		return cmp.Compare(a.content, b.content)
	})

	result := make([]iceberg.ManifestFile, 0, len(keys))
	for _, key := range keys {
		mf, err := writeGroup(key, groups[key])
		if err != nil {
			if cleanupErr := sp.cleanupGeneratedManifests(result); cleanupErr != nil {
				return nil, errors.Join(err, cleanupErr)
			}

			return nil, err
		}
		result = append(result, mf)
	}

	return result, nil
}

func (sp *snapshotProducer) manifestProducer(content iceberg.ManifestContent, files []iceberg.DataFile, output *[]iceberg.ManifestFile) func() (err error) {
	return func() (err error) {
		groups := make(map[int][]iceberg.DataFile)
		for _, df := range files {
			specID := int(df.SpecID())
			groups[specID] = append(groups[specID], df)
		}

		for _, specID := range slices.Sorted(maps.Keys(groups)) {
			mf, err := sp.writeAddedManifest(content, specID, groups[specID])
			if err != nil {
				return err
			}
			*output = append(*output, mf)
		}

		return nil
	}
}

func (sp *snapshotProducer) writeAddedManifest(content iceberg.ManifestContent, specID int, files []iceberg.DataFile) (_ iceberg.ManifestFile, retErr error) {
	spec, err := sp.spec(specID)
	if err != nil {
		return nil, err
	}

	wr, path, counter, out, err := sp.newManifestWriter(spec, iceberg.WithManifestWriterContent(content))
	if err != nil {
		return nil, err
	}
	defer internal.CheckedClose(out, &retErr)
	writerClosed := false
	defer func() {
		if !writerClosed {
			internal.CheckedClose(wr, &retErr)
		}
	}()

	for _, df := range files {
		err := wr.Add(iceberg.NewManifestEntry(iceberg.EntryStatusADDED, &sp.snapshotID,
			sp.addedDataSequenceNumber, nil, df))
		if err != nil {
			return nil, err
		}
	}

	// close the writer to force a flush and ensure counter.Count is accurate
	writerClosed = true
	if err := wr.Close(); err != nil {
		return nil, err
	}

	return wr.ToManifestFile(path, counter.Count, iceberg.WithManifestFileContent(content))
}

func (sp *snapshotProducer) writeAddedDeleteManifest(specID int, additions []deleteFileAddition) (_ iceberg.ManifestFile, retErr error) {
	spec, err := sp.spec(specID)
	if err != nil {
		return nil, err
	}

	wr, path, counter, out, err := sp.newManifestWriter(spec, iceberg.WithManifestWriterContent(iceberg.ManifestContentDeletes))
	if err != nil {
		return nil, err
	}
	defer internal.CheckedClose(out, &retErr)
	writerClosed := false
	defer func() {
		if !writerClosed {
			internal.CheckedClose(wr, &retErr)
		}
	}()

	for _, addition := range additions {
		err := wr.Add(iceberg.NewManifestEntry(iceberg.EntryStatusADDED, &sp.snapshotID,
			addition.dataSequenceNumber, nil, addition.file))
		if err != nil {
			return nil, err
		}
	}

	writerClosed = true
	if err := wr.Close(); err != nil {
		return nil, err
	}

	return wr.ToManifestFile(path, counter.Count, iceberg.WithManifestFileContent(iceberg.ManifestContentDeletes))
}

func (sp *snapshotProducer) summary(props iceberg.Properties) (Summary, error) {
	delta, err := sp.accumulateSummaryDelta(nil)
	if err != nil {
		return Summary{}, err
	}

	var previousSnapshot *Snapshot
	if sp.parentSnapshotID > 0 {
		previousSnapshot, err = sp.txn.meta.SnapshotByID(sp.parentSnapshotID)
		if err != nil {
			return Summary{}, fmt.Errorf("summary: lookup parent snapshot %d: %w", sp.parentSnapshotID, err)
		}
	}

	var previousSummary iceberg.Properties
	if previousSnapshot != nil && previousSnapshot.Summary != nil {
		previousSummary = previousSnapshot.Summary.Properties
	}

	return sp.rebaseSummary(delta, previousSummary, props)
}

// summaryOnRetry recomputes the snapshot summary against the fresh parent for an
// OCC retry. Additions are always counted; a removal of a delete file / deletion
// vector is counted only when it is still present on parent (present.counts),
// because a peer may have already dropped it — replaying the captured removal
// delta on top of a fresh parent that no longer counts the file undercounts
// total-delete-files. Removed data files are always counted: a successful retry
// has already asserted they are present (checkRemovedFiles).
func (sp *snapshotProducer) summaryOnRetry(props iceberg.Properties, parent *Snapshot, present *removedFilePresence) (Summary, error) {
	delta, err := sp.accumulateSummaryDelta(present.counts)
	if err != nil {
		return Summary{}, err
	}

	var previousSummary iceberg.Properties
	if parent != nil && parent.Summary != nil {
		previousSummary = parent.Summary.Properties
	}

	return sp.rebaseSummary(delta, previousSummary, props)
}

// accumulateSummaryDelta builds the added/removed file delta for this producer.
// countDeleteRemoval, when non-nil, gates removal of delete files / deletion
// vectors (returning false skips that removal); data-file removals are always
// counted. nil counts every removal (the initial commit).
func (sp *snapshotProducer) accumulateSummaryDelta(countDeleteRemoval func(iceberg.DataFile) bool) (iceberg.Properties, error) {
	var ssc SnapshotSummaryCollector
	partitionSummaryLimit := sp.txn.meta.props.
		GetInt(WritePartitionSummaryLimitKey, WritePartitionSummaryLimitDefault)
	if _, ok := sp.producerImpl.(*rewriteManifests); ok {
		partitionSummaryLimit = 0
	}
	ssc.setPartitionSummaryLimit(partitionSummaryLimit)

	currentSchema := sp.txn.meta.CurrentSchema()
	partitionSpec, err := sp.txn.meta.CurrentSpec()
	if err != nil || partitionSpec == nil {
		return nil, fmt.Errorf("could not get current partition spec: %w", err)
	}

	addFile := func(df iceberg.DataFile) error {
		spec, err := sp.spec(int(df.SpecID()))
		if err != nil {
			return err
		}

		return ssc.addFile(df, currentSchema, spec)
	}
	removeFile := func(df iceberg.DataFile) error {
		spec, err := sp.spec(int(df.SpecID()))
		if err != nil {
			return err
		}

		return ssc.removeFile(df, currentSchema, spec)
	}

	for _, df := range sp.addedFiles {
		if err = addFile(df); err != nil {
			return nil, err
		}
	}
	for _, addition := range sp.addedDeleteFiles {
		if err = addFile(addition.file); err != nil {
			return nil, err
		}
	}

	for _, df := range sp.deletedFiles {
		if err = removeFile(df); err != nil {
			return nil, err
		}
	}
	for _, df := range sp.deletedDeleteFiles {
		if countDeleteRemoval != nil && !countDeleteRemoval(df) {
			continue
		}
		if err = removeFile(df); err != nil {
			return nil, err
		}
	}
	for _, df := range sp.deletedDVsByRef {
		if countDeleteRemoval != nil && !countDeleteRemoval(df) {
			continue
		}
		if err = removeFile(df); err != nil {
			return nil, err
		}
	}

	return ssc.build(), nil
}

func (sp *snapshotProducer) rebaseSummary(delta, previousSummary, props iceberg.Properties) (Summary, error) {
	maps.Copy(delta, props)

	summary, err := updateSnapshotSummaries(Summary{
		Operation:  sp.op,
		Properties: delta,
	}, previousSummary)
	if err != nil {
		return Summary{}, err
	}

	// Match Java's SnapshotProducer.summary precedence: environment context is
	// applied after user properties and computed totals.
	maps.Copy(summary.Properties, iceberg.EnvironmentContext())

	return summary, nil
}

// removedFilePresence records which of a producer's to-be-removed delete files
// and deletion vectors are still live on the retry's fresh parent. counts is the
// gate passed to accumulateSummaryDelta so the retry summary only subtracts
// removals the parent still counts.
type removedFilePresence struct {
	deleteFiles map[string]struct{} // pos/eq delete FilePath()
	dvRefs      map[string]struct{} // ReferencedDataFile() of DVs whose exact captured path is still on parent
}

func (p *removedFilePresence) counts(df iceberg.DataFile) bool {
	if IsDeletionVector(df) {
		ref := internal.BorrowedDataFileReferencedDataFile(df)
		if ref == nil {
			return false
		}
		_, ok := p.dvRefs[*ref]

		return ok
	}
	_, ok := p.deleteFiles[df.FilePath()]

	return ok
}

// checkRemovedFiles scans parent once and (1) fails terminally if any data file
// this producer intends to rewrite is no longer live — a concurrent writer
// already removed it, so proceeding would leave our replacement beside the
// concurrent result and duplicate rows (Java's failMissingDeletePaths) — and
// (2) returns which removed delete files / deletion vectors are still present so
// the retry summary does not double-subtract a peer's prior removal. A DV
// superseded by a peer (same referenced data file, new path) counts as absent:
// the replacement is inherited untouched and our removal is a no-op.
func (sp *snapshotProducer) checkRemovedFiles(parent *Snapshot) (*removedFilePresence, error) {
	present := &removedFilePresence{
		deleteFiles: map[string]struct{}{},
		dvRefs:      map[string]struct{}{},
	}

	// Producers that remove nothing (fast/merge append) skip the parent scan
	// entirely — otherwise every retry re-reads the full manifest list only to
	// find there is nothing to check.
	if len(sp.deletedFiles) == 0 && len(sp.deletedDeleteFiles) == 0 && len(sp.deletedDVsByRef) == 0 {
		return present, nil
	}

	missingData := make(map[string]struct{}, len(sp.deletedFiles))
	for path := range sp.deletedFiles {
		missingData[path] = struct{}{}
	}

	if parent == nil {
		return sp.missingDataError(present, missingData)
	}

	manifests, err := parent.Manifests(sp.io)
	if err != nil {
		return nil, err
	}
	for _, m := range manifests {
		for entry, err := range sp.iterManifestEntries(m, true) {
			if err != nil {
				return nil, err
			}
			df := entry.DataFile()
			if m.ManifestContent() == iceberg.ManifestContentData {
				delete(missingData, df.FilePath())

				continue
			}
			if IsDeletionVector(df) {
				// A DV without a referenced data file cannot be matched by ref;
				// skip it rather than misclassifying it as a path-keyed delete file.
				// The path must match too: a peer may have superseded our DV with
				// a replacement for the same data file (same ref, new path), and
				// that replacement must not read as "our DV is still present".
				if ref := internal.BorrowedDataFileReferencedDataFile(df); ref != nil {
					if want, ok := sp.deletedDVsByRef[*ref]; ok && want.FilePath() == df.FilePath() {
						present.dvRefs[*ref] = struct{}{}
					}
				}

				continue
			}
			if _, want := sp.deletedDeleteFiles[df.FilePath()]; want {
				present.deleteFiles[df.FilePath()] = struct{}{}
			}
		}
	}

	return sp.missingDataError(present, missingData)
}

func (sp *snapshotProducer) missingDataError(present *removedFilePresence, missingData map[string]struct{}) (*removedFilePresence, error) {
	if len(missingData) == 0 {
		return present, nil
	}

	missing := make([]string, 0, len(missingData))
	for path := range missingData {
		missing = append(missing, path)
	}
	slices.Sort(missing)

	return nil, fmt.Errorf("%w: %d data file(s) to rewrite are no longer on the branch head, e.g. %q",
		ErrCommitDiverged, len(missing), missing[:min(len(missing), 5)])
}

func (sp *snapshotProducer) commit(ctx context.Context) (_ []Update, _ []Requirement, err error) {
	parent, err := sp.parentSnapshot()
	if err != nil {
		return nil, nil, err
	}

	// addedContent is parent-independent: written once here and reused verbatim
	// on every OCC retry (rebuildManifestList recomputes only the parent-
	// dependent portion against the fresh parent).
	newManifests, addedContent, err := sp.buildManifests(ctx, parent)
	if err != nil {
		return nil, nil, err
	}

	return sp.commitManifests(newManifests, addedContent)
}

func writeManifestListFile(
	fs iceio.WriteFileIO,
	path string,
	version int,
	snapshotID int64,
	sequenceNumber int64,
	firstRowID int64,
	parentSnapshotID *int64,
	manifests []iceberg.ManifestFile,
) (addedRows int64, err error) {
	out, err := fs.Create(path)
	if err != nil {
		return 0, err
	}

	defer func() {
		if err != nil {
			err = errors.Join(err, fs.Remove(path))
		}
	}()
	defer internal.CheckedClose(out, &err)

	var writer *iceberg.ManifestListWriter
	switch version {
	case 1:
		writer, err = iceberg.NewManifestListWriterV1(out, snapshotID, parentSnapshotID)
	case 2:
		writer, err = iceberg.NewManifestListWriterV2(out, snapshotID, sequenceNumber, parentSnapshotID)
	case 3:
		writer, err = iceberg.NewManifestListWriterV3(out, snapshotID, sequenceNumber, firstRowID, parentSnapshotID)
	default:
		return 0, fmt.Errorf("unsupported manifest version: %d", version)
	}

	if writer != nil {
		defer internal.CheckedClose(writer, &err)
	}
	if err != nil {
		return 0, err
	}

	if err = writer.AddManifests(manifests); err != nil {
		return 0, err
	}
	if version == 3 && writer.NextRowID() != nil {
		// addedRows counts ALL rows in new manifests (existing + added), even
		// for rewrites where survivors preserve old _row_id values. This
		// "wastes" ID space but doesn't violate uniqueness: actual row IDs come
		// from the explicit Parquet column, not the global counter. Java's
		// ManifestListWriter.V3Writer uses the same accounting.
		addedRows = *writer.NextRowID() - firstRowID
	}

	return addedRows, nil
}

// commitManifests stages the snapshot for an already-built manifest set.
// It is split out of commit so callers that must inspect the planned manifests
// before anything is written (RewriteManifests detecting a no-op) can build the
// manifests themselves and only then decide to write the manifest list.
func (sp *snapshotProducer) commitManifests(newManifests, addedContent []iceberg.ManifestFile) (_ []Update, _ []Requirement, err error) {
	nextSequence := sp.txn.meta.nextSequenceNumber()
	summary, err := sp.summary(sp.snapshotProps)
	if err != nil {
		return nil, nil, err
	}

	fname := newManifestListFileName(sp.snapshotID, 0, sp.commitUuid)
	locProvider, err := sp.txn.tbl.LocationProvider()
	if err != nil {
		return nil, nil, err
	}

	manifestListFilePath := locProvider.NewMetadataLocation(fname)

	var parentSnapshot *int64
	if sp.parentSnapshotID > 0 {
		parentSnapshot = &sp.parentSnapshotID
	}

	firstRowID := int64(0)
	var addedRows int64
	if sp.txn.meta.formatVersion == 3 {
		firstRowID = sp.txn.meta.NextRowID()
	}
	addedRows, err = writeManifestListFile(
		sp.io,
		manifestListFilePath,
		sp.txn.meta.formatVersion,
		sp.snapshotID,
		nextSequence,
		firstRowID,
		parentSnapshot,
		newManifests,
	)
	if err != nil {
		return nil, nil, err
	}

	snapshot := Snapshot{
		SnapshotID:       sp.snapshotID,
		ParentSnapshotID: parentSnapshot,
		SequenceNumber:   nextSequence,
		ManifestList:     manifestListFilePath,
		Summary:          &summary,
		SchemaID:         &sp.txn.meta.currentSchemaID,
		TimestampMs:      time.Now().UnixMilli(),
	}
	if sp.txn.meta.formatVersion == 3 {
		snapshot.FirstRowID = &firstRowID
		snapshot.AddedRows = &addedRows
	}

	branch := sp.txn.branch
	if branch == "" {
		branch = MainBranch
	}

	// Register this producer's client-side conflict validator on the
	// transaction. doCommit runs it against the current catalog state
	// before cat.CommitTable so conflicts the catalog can't see
	// (partition-filter overlap, referenced-file removal) are caught
	// pre-flight. Producers that are commutative against concurrent
	// appends (fast-append, merge-append) opt out via needsValidation()
	// == false and skip registration entirely. The nil guard remains for
	// unit tests that drive commit() on a bare snapshotProducer.
	if impl := sp.producerImpl; impl != nil && impl.needsValidation() {
		sp.txn.validators = append(sp.txn.validators, func(cc *conflictContext) error {
			return impl.validate(cc)
		})
	}

	// Delete-file removals (path-keyed delete files and ref-keyed
	// deletion vectors) are resolved against the snapshot this producer
	// built on — removal identity is snapshot-relative. A
	// refresh-and-replay would inherit a concurrently committed
	// replacement from the fresh base while the stale removal replays
	// as a no-op (checkRemovedFiles deliberately treats a superseded
	// entry as absent), stranding two live deletion vectors on one data
	// file. Such commits must fail on a CAS conflict instead of
	// replaying; the caller re-resolves the removal against the current
	// snapshot and retries. Data-file removals (deletedFiles) stay
	// replayable: they are path-keyed and checkRemovedFiles fails the
	// rebuild terminally when the path is gone from the fresh base.
	if len(sp.deletedDeleteFiles) > 0 || len(sp.deletedDVsByRef) > 0 {
		sp.txn.noReplay = true
	}

	// Build the manifest-list rebuild closure. It is called by doCommit
	// on each OCC retry to regenerate the manifest list so it correctly
	// inherits all data files committed by concurrent writers since the
	// original snapshot was built.
	formatVersion := sp.txn.meta.formatVersion
	snapshotID := sp.snapshotID
	commitUUID := sp.commitUuid
	capturedSnapshot := snapshot // copy the value so the closure is self-contained

	rebuildFn := func(rebuildCtx context.Context, freshMeta Metadata, freshParent *Snapshot, fio iceio.WriteFileIO, attempt int) (*Snapshot, error) {
		// A concurrent writer may have already removed the files this producer
		// intends to rewrite. Grafting our replacement on top of the fresh
		// parent would duplicate rows, so abort terminally. The scan also
		// reports which removed delete files / DVs survive on the fresh parent
		// so the summary below does not double-subtract a peer's prior removal.
		present, err := sp.checkRemovedFiles(freshParent)
		if err != nil {
			return nil, err
		}

		// Rebuild or reuse the parent-dependent manifest set against the fresh
		// parent and combine it with the producer's own manifests. This drops
		// removed files from the fresh parent's manifests rather than
		// resurrecting them; rewrite clustering can reuse its prior output when
		// none of the rewritten inputs were displaced.
		var combined []iceberg.ManifestFile
		var procErr error
		if rebuilder, ok := sp.producerImpl.(retryManifestRebuilder); ok {
			combined, procErr = rebuilder.rebuildManifests(rebuildCtx, freshParent, addedContent)
		} else {
			combined, procErr = sp.assembleManifests(rebuildCtx, freshParent, addedContent)
		}
		if procErr != nil {
			return nil, fmt.Errorf("rebuild manifest list: assemble manifests: %w", procErr)
		}

		// Derive the sequence number from the fresh table-wide last-sequence-number.
		// Using freshParent.SequenceNumber + 1 alone would violate the spec when a
		// concurrent writer on a different branch bumps last-sequence-number
		// without advancing this branch's parent — MetadataBuilder.AddSnapshot
		// rejects SequenceNumber <= lastSequenceNumber.
		//
		// When this snapshot is chained behind a sibling staged in the same
		// transaction (e.g. an append followed by a delete), that sibling has
		// already been rebuilt and is passed here as freshParent with
		// SequenceNumber == lastSequenceNumber + 1. Taking the max keeps the
		// chain strictly increasing so the sibling's own AddSnapshot is not
		// rejected for a duplicate sequence number on apply.
		var newSeq int64
		if formatVersion >= 2 {
			newSeq = freshMeta.LastSequenceNumber() + 1
			if freshParent != nil && freshParent.SequenceNumber >= newSeq {
				newSeq = freshParent.SequenceNumber + 1
			}
		}

		var parentID *int64
		if freshParent != nil {
			id := freshParent.SnapshotID
			parentID = &id
		}

		rebuiltSummary := capturedSnapshot.Summary
		if freshParent != nil && freshParent.Summary != nil && capturedSnapshot.Summary != nil {
			s, sumErr := sp.summaryOnRetry(sp.snapshotProps, freshParent, present)
			if sumErr != nil {
				return nil, fmt.Errorf("rebuild manifest list: recompute summary: %w", sumErr)
			}
			rebuiltSummary = &s
		}

		// Write the rebuilt manifest list to a path unique to this retry
		// attempt. Each retry uses a different attempt counter in the filename
		// (snap-{id}-{attempt}-{uuid}.avro) so that S3 conditional-write
		// semantics (if-none-match) do not reject the overwrite. Orphaned files
		// from superseded retry attempts are removed by doCommit after the
		// commit succeeds.
		fname := newManifestListFileName(snapshotID, attempt, commitUUID)
		manifestListPath := locProvider.NewMetadataLocation(fname)

		firstRowID := int64(0)
		if formatVersion == 3 {
			// Derive firstRowID from the fresh metadata so the manifest-list
			// first-row-id field is consistent with the catalog's nextRowID
			// after concurrent writers have advanced it since attempt 0.
			//
			// A sibling snapshot chained ahead of this one in the same
			// transaction has already claimed the row-id range starting at
			// freshMeta.NextRowID(); begin after it (freshParent.FirstRowID +
			// freshParent.AddedRows) so the two ranges do not overlap. For an
			// external peer parent this term never exceeds freshMeta.NextRowID(),
			// so the max leaves the single-snapshot behavior unchanged.
			firstRowID = freshMeta.NextRowID()
			if freshParent != nil && freshParent.FirstRowID != nil && freshParent.AddedRows != nil {
				if parentNext := *freshParent.FirstRowID + *freshParent.AddedRows; parentNext > firstRowID {
					firstRowID = parentNext
				}
			}
		}
		addedRows, writeErr := writeManifestListFile(
			fio,
			manifestListPath,
			formatVersion,
			snapshotID,
			newSeq,
			firstRowID,
			parentID,
			combined,
		)
		if writeErr != nil {
			return nil, fmt.Errorf("rebuild manifest list: write: %w", writeErr)
		}

		rebuilt := capturedSnapshot
		rebuilt.ManifestList = manifestListPath
		rebuilt.ParentSnapshotID = parentID
		rebuilt.SequenceNumber = newSeq
		if formatVersion == 3 {
			rebuilt.FirstRowID = &firstRowID
			rebuilt.AddedRows = &addedRows
		}
		rebuilt.Summary = rebuiltSummary

		return &rebuilt, nil
	}

	addSnap := NewAddSnapshotUpdate(&snapshot)
	addSnap.ownManifests = addedContent
	addSnap.rebuildManifestList = rebuildFn
	// Hold the producer so doCommit can read the manifests orphaned by OCC
	// retries and clean them up. Only rewrite producers implement this.
	if acc, ok := sp.producerImpl.(supersededAccumulator); ok {
		addSnap.supersededSource = acc
	}

	// Build the assertion from the base table's branch head, not the
	// staged metadata's current snapshot: a staged intermediate snapshot
	// never exists on the catalog, so requiring it could never hold. A
	// nil id requires that the branch not exist yet (this commit
	// creates it).
	baseHeadID := sp.txn.baseRefSnapshotID(branch)

	return []Update{
			addSnap,
			// Carry over the branch's existing retention settings so advancing
			// the ref on commit does not silently discard them. The update
			// encodes exactly the current ref's retention (settings the branch
			// lacks stay 0 and are dropped by the `omitempty` tags); the catalog
			// applies a set-snapshot-ref as a pure replace, so this fully
			// determines the resulting ref rather than merging with the old one.
			sp.txn.meta.NewRetainingSnapshotRefUpdate(branch, sp.snapshotID, BranchRef),
		}, []Requirement{
			AssertRefSnapshotID(branch, baseHeadID),
		}, nil
}
