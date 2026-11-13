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
	"log"
	"slices"
	"strconv"

	"github.com/apache/iceberg-go"
	iceio "github.com/apache/iceberg-go/io"
)

// Snapshot summary keys for a manifest rewrite.
const (
	manifestsCreatedKey  = "manifests-created"  // new manifests the merge wrote
	manifestsReplacedKey = "manifests-replaced" // input manifests the merge consumed
	manifestsKeptKey     = "manifests-kept"     // manifests left untouched, delete manifests included
	entriesProcessedKey  = "entries-processed"  // data-file entries, not rows
)

// NoOpReason explains why a rewrite changed nothing. It lets callers tell a
// table with no current snapshot apart from one whose manifests are already
// optimal, which would otherwise both surface as an empty result.
type NoOpReason int

const (
	// NoOpNone means the rewrite produced changes (not a no-op).
	NoOpNone NoOpReason = iota
	// NoOpNoSnapshot means the table had no current snapshot to rewrite.
	NoOpNoSnapshot
	// NoOpAlreadyOptimal means the eligible manifests were already optimal.
	NoOpAlreadyOptimal
)

func (r NoOpReason) String() string {
	switch r {
	case NoOpNoSnapshot:
		return "no current snapshot"
	case NoOpAlreadyOptimal:
		return "manifests already optimal"
	default:
		return "none"
	}
}

// RewriteManifestsResult reports the manifests changed by a rewrite.
type RewriteManifestsResult struct {
	// RewrittenManifests are the old manifests that were replaced.
	RewrittenManifests []iceberg.ManifestFile
	// AddedManifests are the new manifests written in their place.
	AddedManifests []iceberg.ManifestFile
	// NoOpReason is set when the rewrite changed nothing, distinguishing a
	// missing snapshot from an already-optimal layout. NoOpNone otherwise.
	NoOpReason NoOpReason
}

// IsNoOp reports whether the rewrite changed nothing. Callers should skip the
// commit in that case rather than staging an empty REPLACE snapshot. NoOpReason
// is the single source of truth.
func (r *RewriteManifestsResult) IsNoOp() bool {
	return r.NoOpReason != NoOpNone
}

type rewriteManifestsCfg struct {
	targetSizeBytes int64
	specID          *int
	predicate       func(iceberg.ManifestFile) bool
	clusterBy       func(iceberg.DataFile) any
}

// RewriteManifestsOpt configures [Transaction.RewriteManifests].
type RewriteManifestsOpt func(*rewriteManifestsCfg)

// WithManifestTargetSize overrides the target manifest size in bytes. The
// clustered writer observes the size after Avro blocks flush, so an output can
// exceed the target by nearly one block. A non-positive size is ignored,
// leaving the default from the commit.manifest.target-size-bytes property.
func WithManifestTargetSize(size int64) RewriteManifestsOpt {
	return func(c *rewriteManifestsCfg) {
		if size > 0 {
			c.targetSizeBytes = size
		}
	}
}

// WithRewriteSpecID restricts the rewrite to manifests of one partition spec.
func WithRewriteSpecID(id int) RewriteManifestsOpt {
	return func(c *rewriteManifestsCfg) { c.specID = &id }
}

// WithRewriteManifestPredicate only rewrites manifests for which pred is true.
// Manifests that don't match are left untouched.
func WithRewriteManifestPredicate(pred func(iceberg.ManifestFile) bool) RewriteManifestsOpt {
	return func(c *rewriteManifestsCfg) { c.predicate = pred }
}

// WithRewriteManifestClusterBy groups rewritten data files by the key returned
// by clusterBy. Files with the same key and partition spec are written to the
// same manifest until the target size is reached, after which another manifest
// is started for that key. Clustering replaces normal size-only bin-packing,
// including minCountToMerge. One writer remains open for each distinct
// (partition spec, key) pair, so clusterBy should have reasonable cardinality
// to avoid exhausting object-store handles. The key must be non-nil,
// comparable, and equal to itself.
// The callback should be deterministic because an OCC retry may need to
// re-cluster manifests displaced by a concurrent writer. When all rewritten
// inputs remain present, the existing clustered output is reused instead.
func WithRewriteManifestClusterBy(clusterBy func(iceberg.DataFile) any) RewriteManifestsOpt {
	return func(c *rewriteManifestsCfg) { c.clusterBy = clusterBy }
}

// rewriteManifests is a producer that merges small data manifests into
// fewer, target-sized ones, committed as a metadata-only REPLACE snapshot.
type rewriteManifests struct {
	base *snapshotProducer
	cfg  rewriteManifestsCfg

	rewritten []iceberg.ManifestFile
	added     []iceberg.ManifestFile

	// superseded accumulates merged manifest files written by a rebuild on a
	// prior OCC attempt that a later attempt replaced. doCommit removes them
	// after a successful commit so retries don't leak orphaned manifests.
	superseded []string

	// result is the value returned to the caller. record() rewrites its
	// fields on every pass, including OCC retries, so the pointer the caller
	// holds reflects the manifests actually committed, not attempt 0's.
	result *RewriteManifestsResult
}

func newRewriteManifestsProducer(txn *Transaction, fs iceio.WriteFileIO, props iceberg.Properties, cfg rewriteManifestsCfg) *snapshotProducer {
	prod := createSnapshotProducer(OpReplace, txn, fs, nil, props)
	prod.producerImpl = &rewriteManifests{base: prod, cfg: cfg, result: &RewriteManifestsResult{}}

	return prod
}

// supersededManifests returns the merged manifests orphaned across OCC retries,
// safe to delete once the commit resolves. When the commit never landed
// (committed is false) the final attempt's manifests are orphaned too — a
// snapshot that never committed references nothing — so they are appended.
func (r *rewriteManifests) supersededManifests(committed bool) []string {
	if committed {
		return r.superseded
	}

	paths := slices.Clone(r.superseded)
	for _, m := range r.added {
		paths = append(paths, m.FilePath())
	}

	return paths
}

func (r *rewriteManifests) parentDependentManifests(_ context.Context, parent *Snapshot) ([]iceberg.ManifestFile, error) {
	if parent == nil {
		return nil, nil
	}

	return parent.Manifests(r.base.io)
}

// rebuildManifests reuses the clustered output when a concurrent commit only
// added manifests and left every input manifest this rewrite replaced in the
// fresh parent. This is the same requiresRewrite optimization used by Java's
// rewrite-manifest producer: the old output remains valid, and only the fresh
// parent's additional manifests need to be inherited.
func (r *rewriteManifests) rebuildManifests(ctx context.Context, parent *Snapshot, addedContent []iceberg.ManifestFile) ([]iceberg.ManifestFile, error) {
	if r.cfg.clusterBy != nil {
		if reused, ok, err := r.reuseManifests(parent); err != nil {
			return nil, err
		} else if ok {
			return reused, nil
		}
	}

	return r.base.assembleManifests(ctx, parent, addedContent)
}

func (r *rewriteManifests) reuseManifests(parent *Snapshot) ([]iceberg.ManifestFile, bool, error) {
	if parent == nil || len(r.rewritten) == 0 || len(r.added) == 0 {
		return nil, false, nil
	}

	current, err := parent.Manifests(r.base.io)
	if err != nil {
		return nil, false, err
	}

	required := make(map[string]struct{}, len(r.rewritten))
	for _, manifest := range r.rewritten {
		required[manifest.FilePath()] = struct{}{}
	}
	for _, manifest := range current {
		delete(required, manifest.FilePath())
	}
	if len(required) > 0 {
		return nil, false, nil
	}
	// The fresh parent may contain manifests added by a concurrent writer.
	// They remain untouched in the reused output and must be included in the
	// summary's kept count even though no re-merge was needed.
	r.base.snapshotProps[manifestsKeptKey] = strconv.Itoa(len(current) - len(r.rewritten))

	result := make([]iceberg.ManifestFile, 0, len(current)-len(r.rewritten)+len(r.added))
	rewritten := make(map[string]struct{}, len(r.rewritten))
	for _, manifest := range r.rewritten {
		rewritten[manifest.FilePath()] = struct{}{}
	}
	inserted := false
	for _, manifest := range current {
		if _, ok := rewritten[manifest.FilePath()]; ok {
			if !inserted {
				result = append(result, r.added...)
				inserted = true
			}

			continue
		}
		result = append(result, manifest)
	}

	return result, inserted, nil
}

func (r *rewriteManifests) processManifests(manifests []iceberg.ManifestFile) ([]iceberg.ManifestFile, error) {
	var toRewrite, kept []iceberg.ManifestFile
	for _, m := range manifests {
		if r.eligible(m) {
			toRewrite = append(toRewrite, m)
		} else {
			kept = append(kept, m)
		}
	}

	mgr := manifestMergeManager{
		targetSizeBytes: r.cfg.targetSizeBytes,
		minCountToMerge: 1,    // force a merge regardless of count
		mergeEnabled:    true, // explicit op ignores commit.manifest-merge.enabled
		clusterBy:       r.cfg.clusterBy,
		snap:            r.base,
	}
	merged, err := mgr.mergeManifests(toRewrite)
	if err != nil {
		return nil, err
	}

	// mergeManifests already wrote the new .avro files. If this pass fails
	// before record() takes ownership, nothing will reference them, so delete
	// them here rather than leak orphans that no commit/retry cleanup can reach.
	owned := false
	defer func() {
		if !owned {
			r.deleteWritten(toRewrite, merged)
		}
	}()

	// Record on every pass that actually re-merges, not just the first. An OCC
	// retry that finds a displaced input writes a different merged set; the last
	// pass is the one that commits, so its counts are the ones that must win.
	// record also runs the file-count guard, so an error here leaves the
	// merged files to the defer above for cleanup.
	if err := r.record(toRewrite, merged, kept); err != nil {
		return nil, err
	}
	owned = true

	return slices.Concat(merged, kept), nil
}

// eligible reports whether m is a data manifest selected for rewrite.
func (r *rewriteManifests) eligible(m iceberg.ManifestFile) bool {
	if m.ManifestContent() != iceberg.ManifestContentData {
		return false
	}
	if r.cfg.specID != nil && int(m.PartitionSpecID()) != *r.cfg.specID {
		return false
	}
	if r.cfg.predicate != nil && !r.cfg.predicate(m) {
		return false
	}

	return true
}

// deleteWritten removes the manifests mergeManifests newly wrote for this pass.
// Single-manifest bins pass through with their input path and are left alone;
// only the merged files (in the output but not the input) are deleted.
func (r *rewriteManifests) deleteWritten(input, merged []iceberg.ManifestFile) {
	inPaths := make(map[string]struct{}, len(input))
	for _, m := range input {
		inPaths[m.FilePath()] = struct{}{}
	}
	for _, m := range merged {
		if _, ok := inPaths[m.FilePath()]; ok {
			continue
		}
		if err := r.base.io.Remove(m.FilePath()); err != nil {
			log.Printf("Warning: failed to delete orphaned merged manifest %s: %v", m.FilePath(), err)
		}
	}
}

func (r *rewriteManifests) record(toRewrite, merged, kept []iceberg.ManifestFile) error {
	// A prior pass (a superseded OCC attempt) wrote its own merged manifests
	// that the catalog never referenced. Carry their paths into the
	// producer's superseded set so commit() can delete them on success;
	// otherwise every retry leaks a full set of merged .avro files.
	for _, m := range r.added {
		r.superseded = append(r.superseded, m.FilePath())
	}
	r.added, r.rewritten = nil, nil

	inPaths := make(map[string]struct{}, len(toRewrite))
	for _, m := range toRewrite {
		inPaths[m.FilePath()] = struct{}{}
	}
	outPaths := make(map[string]struct{}, len(merged))
	for _, m := range merged {
		outPaths[m.FilePath()] = struct{}{}
	}

	// Bins of a single manifest pass through unchanged; only the manifests
	// that actually appear on one side and not the other are added/replaced.
	// This drives the returned RewriteManifestsResult.
	for _, m := range merged {
		if _, ok := inPaths[m.FilePath()]; !ok {
			r.added = append(r.added, m)
		}
	}
	for _, m := range toRewrite {
		if _, ok := outPaths[m.FilePath()]; !ok {
			r.rewritten = append(r.rewritten, m)
		}
	}

	// Guard against a merge dropping or duplicating data files: the rewritten
	// manifests must hold the same active files as the ones written in their
	// place. Pass-through bins appear on both sides and cancel, so comparing the
	// rewritten/added diff is equivalent to comparing the full input/output and
	// avoids re-walking the pass-throughs. The processed count doubles as the
	// entries-processed summary value.
	processed, err := manifestActiveFiles(r.base.io, r.rewritten)
	if err != nil {
		return err
	}
	written, err := manifestActiveFiles(r.base.io, r.added)
	if err != nil {
		return err
	}
	if processed != written {
		return fmt.Errorf("rewrite manifests changed active file count: %d before, %d after", processed, written)
	}

	// Publish into the shared result the caller holds so a retry's counts win.
	r.result.RewrittenManifests = r.rewritten
	r.result.AddedManifests = r.added

	// The summary follows Java's Appendix-F accounting, which counts only the
	// manifests a merge actually wrote and consumed — not single-manifest bins
	// that pass through untouched — so a Go-written snapshot reads the same as a
	// Java one.
	r.base.snapshotProps[manifestsCreatedKey] = strconv.Itoa(len(r.added))
	r.base.snapshotProps[manifestsReplacedKey] = strconv.Itoa(len(r.rewritten))
	r.base.snapshotProps[manifestsKeptKey] = strconv.Itoa(len(kept))
	r.base.snapshotProps[entriesProcessedKey] = strconv.FormatInt(processed, 10)

	return nil
}

func (r *rewriteManifests) validate(*conflictContext) error { return nil }
func (r *rewriteManifests) needsValidation() bool           { return false }

// manifestActiveFiles sums the active (added + existing) data files across
// manifests. It uses the manifest header counts when present and falls back to
// reading and counting the live entries when a count is absent — V1 manifests
// don't populate the count fields, so without the fallback the count guard
// would silently not run on exactly the tables that most need it.
func manifestActiveFiles(fs iceio.IO, manifests []iceberg.ManifestFile) (int64, error) {
	var total int64
	for _, m := range manifests {
		added, existing := m.AddedDataFiles(), m.ExistingDataFiles()
		if added < 0 || existing < 0 {
			// discardDeleted drops DELETED entries, leaving added + existing.
			for _, err := range m.Entries(fs, true) {
				if err != nil {
					return 0, fmt.Errorf("count active files in manifest %s: %w", m.FilePath(), err)
				}
				total++
			}

			continue
		}
		total += int64(added) + int64(existing)
	}

	return total, nil
}

// RewriteManifests merges small data manifests in the current snapshot into
// fewer, target-sized ones and stages the result as a REPLACE snapshot. It
// rewrites metadata only; no data files are read or written. Delete manifests
// are left untouched.
//
// By default, manifests are clustered by size only (bin-packed toward the
// target size). With WithRewriteManifestClusterBy, entries are grouped by the
// returned key and partition spec before the same target-size rolling is
// applied. This can reduce planning time when the key matches common query
// filters, because each manifest contains fewer unrelated partition values.
//
// On a V3 table each rewrite advances next-row-id by the eligible manifests'
// row count even though no rows are written, so running it on a cadence
// steadily consumes row-ID space.
//
// A no-op result (IsNoOp) means there was nothing to do — either the target
// branch has no head (NoOpNoSnapshot) or the eligible manifests are already
// optimal (NoOpAlreadyOptimal). A no-op writes no manifest list and stages
// nothing on the transaction, so a following Commit is a true no-op; callers
// can skip it either way.
func (t *Transaction) RewriteManifests(ctx context.Context, opts ...RewriteManifestsOpt) (*RewriteManifestsResult, error) {
	meta, err := t.txnMeta()
	if err != nil {
		return nil, err
	}

	// Ask the target branch for its head, not main: the manifests actually
	// rewritten below come from prod.parentSnapshot(), which resolves the
	// branch head. A main-only check reports NoOpNoSnapshot for a branch that
	// has manifests to merge whenever main has no head of its own.
	if t.planningSnapshot(meta) == nil {
		return &RewriteManifestsResult{NoOpReason: NoOpNoSnapshot}, nil
	}

	cfg := rewriteManifestsCfg{
		targetSizeBytes: int64(meta.props.GetInt(ManifestTargetSizeBytesKey, ManifestTargetSizeBytesDefault)),
	}
	for _, o := range opts {
		o(&cfg)
	}

	if cfg.specID != nil {
		if _, err := meta.GetSpecByID(*cfg.specID); err != nil {
			return nil, fmt.Errorf("cannot rewrite manifests: unknown partition spec id %d: %w", *cfg.specID, err)
		}
	}

	fs, err := t.tbl.fsF(ctx)
	if err != nil {
		return nil, err
	}
	wfs, ok := fs.(iceio.WriteFileIO)
	if !ok {
		return nil, ErrWriteIORequired
	}

	prod := newRewriteManifestsProducer(t, wfs, iceberg.Properties{}, cfg)
	impl, ok := prod.producerImpl.(*rewriteManifests)
	if !ok {
		return nil, fmt.Errorf("internal error: unexpected producer type %T", prod.producerImpl)
	}

	parent, err := prod.parentSnapshot()
	if err != nil {
		return nil, err
	}

	// Plan the merge first (this writes merged manifests only when a real merge
	// happens) so a no-op is detected before commit writes the manifest list or
	// apply stages an empty REPLACE. A single-manifest bin passes through and
	// writes nothing, so an empty diff means nothing was written.
	newManifests, addedContent, err := prod.buildManifests(ctx, parent)
	if err != nil {
		return nil, err
	}
	if len(impl.result.AddedManifests) == 0 && len(impl.result.RewrittenManifests) == 0 {
		impl.result.NoOpReason = NoOpAlreadyOptimal

		return impl.result, nil
	}

	updates, reqs, err := prod.commitManifests(newManifests, addedContent)
	if err != nil {
		return nil, err
	}
	if err := t.apply(updates, reqs); err != nil {
		return nil, err
	}

	// impl.result aliases the struct record() updates on every pass, so it
	// reflects the winning attempt after the caller commits the transaction.
	return impl.result, nil
}
