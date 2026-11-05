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

// This file is the client-side conflict-validation framework for
// concurrent commits. It lives alongside the catalog-side Requirement
// machinery in requirements.go, and the two serve different roles:
//
//   - A Requirement (requirements.go) is asserted by the catalog on
//     commit. It can only check invariants visible to the catalog
//     (branch heads, schema ids, spec ids, table UUID) and has no
//     access to manifest or data-file content.
//
//   - A validator in this file runs client-side against a freshly
//     loaded Metadata, walks the concurrent snapshots added on the
//     committer's branch since the committer's base, and rejects
//     commits whose semantics would be violated by those concurrent
//     snapshots (e.g. a concurrent append into a partition the
//     committer is overwriting, or a concurrent pos-delete against a
//     file the committer is rewriting).
//
// Both systems cooperate: Requirement ensures the committer's base
// assumption about the branch head still holds, and if it does not
// the retry loop re-fetches metadata; these validators then check
// whether the semantic invariants still hold against the new head.
//
// The validators themselves are unexported: they are producer-facing
// plumbing, not a library API. The only user-visible surface is
// IsolationLevel, the isolation property keys, and the conflict
// error sentinels — mirroring Java's split where validators are
// `protected` on MergingSnapshotProducer and only the inputs and
// outputs are public.

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/apache/iceberg-go"
	iceberginternal "github.com/apache/iceberg-go/internal"
	iceio "github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/table/internal"
	"github.com/google/uuid"
)

// IsolationLevel controls how strictly a commit rejects concurrent
// writes on the same branch. It mirrors
// org.apache.iceberg.IsolationLevel from the Java reference.
type IsolationLevel string

const (
	// IsolationSerializable rejects any concurrent commit that added
	// data files matching the committer's filter since the base
	// snapshot. This is the strongest guarantee: a serializable commit
	// behaves as if it ran in isolation against the base snapshot.
	IsolationSerializable IsolationLevel = "serializable"

	// IsolationSnapshot only rejects concurrent commits that touched
	// files the committer explicitly references (e.g. a position
	// delete whose referenced data file was removed). Concurrent
	// appends into the committer's filter region are allowed.
	IsolationSnapshot IsolationLevel = "snapshot"
)

// Property keys for configuring isolation per write operation. Names
// match the Java TableProperties constants so configs can be ported
// directly between engines.
const (
	// WriteDeleteIsolationLevelKey controls isolation for DeleteFiles
	// and RowDelta eq-delete commits. Default: serializable (matches
	// Java trunk). Older Java releases and some pyiceberg codepaths
	// used "snapshot"; set this property explicitly to snapshot if
	// you are migrating from a pipeline that relied on that behavior.
	WriteDeleteIsolationLevelKey     = "write.delete.isolation-level"
	WriteDeleteIsolationLevelDefault = IsolationSerializable

	// WriteUpdateIsolationLevelKey controls isolation for overwrite /
	// merge commits. Default: serializable (matches Java trunk).
	// Older Java releases and some pyiceberg codepaths used
	// "snapshot"; set this property explicitly to snapshot if you are
	// migrating from a pipeline that relied on that behavior.
	// write.merge.isolation-level is not modeled separately here yet;
	// merge-style producers continue to use write.update.isolation-level.
	WriteUpdateIsolationLevelKey     = "write.update.isolation-level"
	WriteUpdateIsolationLevelDefault = IsolationSerializable
)

// ErrInvalidIsolationLevel is returned when a configured isolation level
// value is invalid. It is terminal for the current commit attempt and does
// not wrap ErrCommitFailed, so config errors are not retried as conflicts.
var ErrInvalidIsolationLevel = errors.New("invalid isolation level")

// readIsolationLevel returns the isolation level for the given
// property key, falling back to defVal when the key is absent.
func readIsolationLevel(props iceberg.Properties, key string, defVal IsolationLevel) (IsolationLevel, error) {
	v, ok := props[key]
	if !ok || v == "" {
		return defVal, nil
	}

	lower := IsolationLevel(strings.ToLower(v))
	switch lower {
	case IsolationSerializable, IsolationSnapshot:
		return lower, nil
	default:
		return defVal, fmt.Errorf("%w: %q for property %q", ErrInvalidIsolationLevel, v, key)
	}
}

// ErrCommitDiverged is returned when the committer's base snapshot is
// no longer on the branch, so conflict validation cannot enumerate
// concurrent snapshots. The committer must refresh and rebuild their
// commit from the new base — naive retry would fail identically.
//
// Unlike the other conflict sentinels in this file, ErrCommitDiverged
// does NOT wrap ErrCommitFailed: it is terminal for the current
// attempt. This mirrors Java's ValidationException, which the
// SnapshotProducer retry machinery does not catch.
var ErrCommitDiverged = errors.New("commit diverged: base snapshot is no longer on the branch")

// Retryable conflict sentinels. All wrap ErrCommitFailed so the retry
// loop in doCommit treats them as retryable. Callers that need to
// distinguish which kind of conflict occurred can match on the
// specific sentinel via errors.Is.
var (
	// ErrConflictingDataFiles is returned when a concurrent commit
	// added data files that satisfy the committer's filter.
	ErrConflictingDataFiles = fmt.Errorf("%w: concurrent data files added", ErrCommitFailed)

	// ErrConflictingDeleteFiles is returned when a concurrent commit
	// added delete files that could mask rows the committer is
	// writing or replacing.
	ErrConflictingDeleteFiles = fmt.Errorf("%w: concurrent delete files added", ErrCommitFailed)

	// ErrDataFilesMissing is returned when a concurrent commit deleted
	// data files the committer explicitly references (e.g. a position
	// delete pointing at a file that is no longer reachable).
	ErrDataFilesMissing = fmt.Errorf("%w: referenced data files missing", ErrCommitFailed)
)

// conflictContext carries the inputs every validator needs: the
// latest catalog state (current) and the filesystem used to read
// manifest content on demand. Build it with newConflictContext.
//
// A validator inspects snapshots committed on the committer's branch
// between the writer's base and the current head. When both metadata
// values refer to the same snapshot on the branch, there are no
// concurrent commits and validators return nil without reading
// manifests.
//
// The metadata and concurrent-snapshot walk are fixed once constructed — do
// NOT reuse the same context across retry attempts that re-fetch catalog
// state, because both the walk and its read caches would become stale. There
// is deliberately no Refresh or mutator method; a refresh must flow as a
// fresh call to newConflictContext(newBase, newCurrent, ...).
type conflictContext struct {
	current       Metadata
	branch        string
	fs            iceio.IO
	caseSensitive bool

	// Resolved once; subsequent validator calls reuse the walk.
	concurrent []Snapshot

	// Lazily caches raw manifest bytes for this validation attempt. The
	// cache is intentionally scoped to the context because all validators
	// inspect the same immutable metadata snapshot, while ManifestFile.Entries
	// still performs descriptor-specific inheritance for each logical read.
	manifestIO *conflictManifestIO

	// Parsed manifest-list descriptors are also shared across validators.
	manifestLists map[int64]conflictManifestList
}

type conflictManifestList struct {
	manifests []iceberg.ManifestFile
	err       error
}

const maxConflictManifestCacheBytes = 64 << 20

var errConflictManifestIOReadOnly = errors.New("conflict manifest IO is read-only")

// conflictManifestIO is a read-through cache used only while conflict
// validators inspect one conflictContext. Manifest files are immutable after
// they are committed, so the same path can be safely served from the cached
// bytes to later validators without reopening the backing object store.
//
// The cache stores raw bytes instead of decoded entries. The first read still
// streams from the backing IO and is recorded only when it has consumed the
// size reported by Stat, so an early validator exit does not cache a partial
// manifest. The cache is bounded per validation attempt; reads after the
// bound fall through to the backing IO. Later reads stream decoded entries
// from completed bytes and let each ManifestFile descriptor apply its own
// inheritance metadata.
type conflictManifestIO struct {
	base        iceio.IO
	files       map[string][]byte
	cachedBytes int64
}

func newConflictManifestIO(base iceio.IO) *conflictManifestIO {
	return &conflictManifestIO{
		base:  base,
		files: make(map[string][]byte),
	}
}

func (c *conflictManifestIO) Open(name string) (iceio.File, error) {
	if data, ok := c.files[name]; ok {
		return newConflictManifestFile(name, data), nil
	}

	f, err := c.base.Open(name)
	if err != nil {
		return nil, err
	}
	if c.cachedBytes >= maxConflictManifestCacheBytes {
		return f, nil
	}

	info, statErr := f.Stat()
	if statErr != nil || info == nil || info.Size() < 0 {
		if statIO, ok := c.base.(iceio.StatIO); ok {
			info, statErr = statIO.Stat(name)
		}
	}
	if statErr != nil || info == nil || info.Size() < 0 {
		return f, nil
	}

	return &conflictManifestRecordingFile{
		base:         f,
		cache:        c,
		name:         name,
		expectedSize: info.Size(),
		cacheLimit:   maxConflictManifestCacheBytes - c.cachedBytes,
		cacheable:    true,
		complete:     info.Size() == 0,
	}, nil
}

func (c *conflictManifestIO) Remove(string) error {
	return errConflictManifestIOReadOnly
}

func (c *conflictManifestIO) Stat(name string) (fs.FileInfo, error) {
	if data, ok := c.files[name]; ok {
		return conflictManifestFileInfo{name: name, size: int64(len(data))}, nil
	}

	if statIO, ok := c.base.(iceio.StatIO); ok {
		return statIO.Stat(name)
	}

	f, err := c.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return f.Stat()
}

type conflictManifestFile struct {
	*bytes.Reader
	name string
	size int64
}

func newConflictManifestFile(name string, data []byte) *conflictManifestFile {
	return &conflictManifestFile{
		Reader: bytes.NewReader(data),
		name:   name,
		size:   int64(len(data)),
	}
}

func (f *conflictManifestFile) Close() error { return nil }

func (f *conflictManifestFile) Stat() (fs.FileInfo, error) {
	return conflictManifestFileInfo{name: f.name, size: f.size}, nil
}

type conflictManifestRecordingFile struct {
	base         iceio.File
	cache        *conflictManifestIO
	name         string
	expectedSize int64
	cacheLimit   int64
	data         []byte
	cacheable    bool
	complete     bool
}

func (f *conflictManifestRecordingFile) Read(p []byte) (int, error) {
	n, err := f.base.Read(p)
	if f.cacheable && n > 0 {
		if int64(len(f.data))+int64(n) > f.cacheLimit {
			f.cacheable = false
			f.data = nil
		} else {
			f.data = append(f.data, p[:n]...)
			f.complete = int64(len(f.data)) == f.expectedSize
		}
	}

	return n, err
}

func (f *conflictManifestRecordingFile) ReadAt(p []byte, offset int64) (int, error) {
	f.disableCaching()

	return f.base.ReadAt(p, offset)
}

func (f *conflictManifestRecordingFile) Seek(offset int64, whence int) (int64, error) {
	f.disableCaching()

	return f.base.Seek(offset, whence)
}

func (f *conflictManifestRecordingFile) Stat() (fs.FileInfo, error) {
	return f.base.Stat()
}

func (f *conflictManifestRecordingFile) Close() error {
	err := f.base.Close()
	if err == nil && f.cacheable && f.complete && int64(len(f.data)) == f.expectedSize {
		f.cache.files[f.name] = f.data
		f.cache.cachedBytes += int64(len(f.data))
	}

	return err
}

func (f *conflictManifestRecordingFile) disableCaching() {
	f.cacheable = false
	f.complete = false
	f.data = nil
}

var (
	_ iceio.IO   = (*conflictManifestIO)(nil)
	_ iceio.File = (*conflictManifestFile)(nil)
	_ iceio.File = (*conflictManifestRecordingFile)(nil)
)

type conflictManifestFileInfo struct {
	name string
	size int64
}

func (f conflictManifestFileInfo) Name() string       { return f.name }
func (f conflictManifestFileInfo) Size() int64        { return f.size }
func (f conflictManifestFileInfo) Mode() fs.FileMode  { return 0 }
func (f conflictManifestFileInfo) ModTime() time.Time { return time.Time{} }
func (f conflictManifestFileInfo) IsDir() bool        { return false }
func (f conflictManifestFileInfo) Sys() any           { return nil }

func (c *conflictContext) manifestReadIO() *conflictManifestIO {
	if c.manifestIO == nil {
		c.manifestIO = newConflictManifestIO(c.fs)
	}

	return c.manifestIO
}

func (c *conflictContext) manifestsFor(snap Snapshot) ([]iceberg.ManifestFile, error) {
	if c.manifestLists == nil {
		c.manifestLists = make(map[int64]conflictManifestList)
	}
	if cached, ok := c.manifestLists[snap.SnapshotID]; ok {
		return cached.manifests, cached.err
	}

	manifests, err := snap.Manifests(c.fs)
	c.manifestLists[snap.SnapshotID] = conflictManifestList{manifests: manifests, err: err}

	return manifests, err
}

// newConflictContext builds a validation context for the given
// branch. It walks the ancestry of the branch's current head back to
// the writer's base snapshot to enumerate concurrent commits.
//
// caseSensitive must match the value the committer used when it bound
// its filter (typically the scan's case-sensitivity) so
// partition-spec projection resolves the same column identifiers.
//
// When base and current point at the same snapshot on the branch, the
// returned context has no concurrent snapshots and validators
// short-circuit to nil. When the base snapshot is no longer on the
// branch (divergent commit, expired base), newConflictContext returns
// ErrCommitDiverged; the commit cannot be safely revalidated without
// a refresh-and-rebuild.
func newConflictContext(base, current Metadata, branch string, fs iceio.IO, caseSensitive bool) (*conflictContext, error) {
	currentHead := current.SnapshotByName(branch)
	if currentHead == nil {
		// Branch does not exist on the current side — either it was
		// deleted concurrently or the writer is creating it. Treat as
		// divergent; refresh-and-replay will decide.
		return nil, fmt.Errorf("%w: branch %q missing on current metadata", ErrCommitDiverged, branch)
	}

	baseHead := base.SnapshotByName(branch)
	if baseHead == nil {
		// Writer's base has no view of this branch (empty table or
		// branch created concurrently). Every snapshot now reachable
		// from the branch head is concurrent relative to base.
		// Truncation (missing intermediate, cycle) is treated as
		// divergent for the same reason AncestorsBetween's
		// baseFound=false is: an under-counted concurrent list silently
		// hides conflicts.
		concurrent, complete := AncestorsOfChecked(currentHead.SnapshotID, current.SnapshotByID)
		if !complete {
			return nil, fmt.Errorf("%w: ancestry walk from %d on %s truncated; no base snapshot to bound against",
				ErrCommitDiverged, currentHead.SnapshotID, branch)
		}

		return &conflictContext{
			current:       current,
			branch:        branch,
			fs:            fs,
			caseSensitive: caseSensitive,
			concurrent:    concurrent,
		}, nil
	}

	concurrent, baseFound := AncestorsBetween(currentHead.SnapshotID, baseHead.SnapshotID, current.SnapshotByID)
	if !baseFound {
		return nil, fmt.Errorf("%w: base snapshot %d not found in %s ancestry from %d",
			ErrCommitDiverged, baseHead.SnapshotID, branch, currentHead.SnapshotID)
	}

	return &conflictContext{
		current:       current,
		branch:        branch,
		fs:            fs,
		caseSensitive: caseSensitive,
		concurrent:    concurrent,
	}, nil
}

// forEachAddedEntry visits every ManifestEntry with status ADDED
// whose snapshot id equals one of the concurrent snapshots. This
// matches Java's per-entry filter: an ADDED entry is attributed to
// the snapshot that produced it, independent of which manifest
// happens to carry it today — a manifest-rewrite (e.g. the merging
// append producer) can carry ADDED entries from a prior snapshot
// into a newer manifest whose added-snapshot-id is the committing
// one, so per-manifest filtering alone would miss or mis-attribute
// entries. Filtering on entry.SnapshotID() is the only attribution
// that survives manifest rewrites.
//
// The visitor returns early if the callback returns a non-nil error.
func (c *conflictContext) forEachAddedEntry(content iceberg.ManifestContent, visit func(Snapshot, iceberg.ManifestEntry) error) error {
	for _, snap := range c.concurrent {
		manifests, err := c.manifestsFor(snap)
		if err != nil {
			return fmt.Errorf("loading entries for concurrent snapshot %d: %w", snap.SnapshotID, err)
		}
		for _, mf := range manifests {
			if content >= 0 && mf.ManifestContent() != content {
				continue
			}

			for entry, err := range mf.Entries(c.manifestReadIO(), false) {
				if err != nil {
					return fmt.Errorf("loading entries for concurrent snapshot %d: %w", snap.SnapshotID, err)
				}
				if entry.Status() != iceberg.EntryStatusADDED {
					continue
				}
				if entry.SnapshotID() != snap.SnapshotID {
					// Entry was inherited from a prior snapshot and is
					// not attributable to this concurrent commit.
					continue
				}
				if err := visit(snap, entry); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// validateDataFilesExist verifies that every file path in
// referencedPaths is still a live data file on the current branch head.
// This is the check a position-delete commit performs to prove the
// files it references have not been removed by a concurrent commit
// (at which point the pos-delete would apply to rewritten data and
// produce incorrect results).
//
// A file counts as present only if the head carries a non-deleted entry
// (ADDED or EXISTING) for it. A DELETED entry — left behind by a
// concurrent compaction or overwrite that rewrote the file — does NOT
// satisfy existence: the rows moved to a new file, so the pos-delete is
// orphaned and the commit must be rejected. This mirrors Java's
// MergingSnapshotProducer, which treats a concurrently-deleted entry for
// a referenced path as a conflict.
//
// Returns ErrDataFilesMissing listing the first few missing paths.
// The committer should treat this as unrecoverable for the current
// commit attempt and rebuild the delete against the current
// snapshot's data files.
//
// Cost is O(all data manifests × all entries) regardless of
// len(referencedPaths); callers MUST batch referenced paths into a
// single call rather than calling once per path.
func validateDataFilesExist(ctx *conflictContext, referencedPaths []string) error {
	if len(referencedPaths) == 0 {
		return nil
	}
	needed := make(map[string]struct{}, len(referencedPaths))
	for _, p := range referencedPaths {
		needed[p] = struct{}{}
	}

	head := ctx.current.SnapshotByName(ctx.branch)
	if head == nil {
		return fmt.Errorf("%w: branch %q missing on current metadata", ErrCommitDiverged, ctx.branch)
	}

	manifests, err := ctx.manifestsFor(*head)
	if err != nil {
		return fmt.Errorf("iterating data files for current head %d: %w", head.SnapshotID, err)
	}
	for _, mf := range manifests {
		if mf.ManifestContent() != iceberg.ManifestContentData {
			continue
		}
		for entry, err := range mf.Entries(ctx.manifestReadIO(), false) {
			if err != nil {
				return fmt.Errorf("iterating data files for current head %d: %w", head.SnapshotID, err)
			}
			// A DELETED entry means the file was removed (e.g. rewritten by a
			// concurrent compaction); it is no longer live data a pos-delete can
			// apply to, so it does not satisfy existence.
			if entry.Status() == iceberg.EntryStatusDELETED {
				continue
			}
			path := entry.DataFile().FilePath()
			delete(needed, path)
		}
		if len(needed) == 0 {
			return nil
		}
	}

	if len(needed) == 0 {
		return nil
	}

	missing := make([]string, 0, len(needed))
	for p := range needed {
		missing = append(missing, p)
	}
	sort.Strings(missing)
	sample := missing
	if len(sample) > 5 {
		sample = sample[:5]
	}

	return fmt.Errorf("%w: %d files missing, e.g. %v", ErrDataFilesMissing, len(missing), sample)
}

// validateAddedDataFilesMatchingFilter returns an error if any
// concurrent snapshot added a data file whose partition satisfies the
// given filter. Overwrite and delete operations that declare a
// predicate call this so that a concurrent append into the same
// partition is rejected before the commit overwrites it.
//
// The check runs in three layers:
//  1. A manifest-level partition-summary evaluator prunes manifests
//     whose summaries cannot overlap the filter.
//  2. Inside surviving manifests, every ADDED entry attributed to the
//     concurrent snapshot is evaluated against a per-spec partition
//     evaluator on its partition tuple, so a manifest whose summary
//     straddles the filter only triggers a conflict when at least
//     one actual added file's partition value satisfies the filter.
//  3. The surviving files are evaluated against their column metrics so
//     files whose bounds cannot match the filter are ignored.
func validateAddedDataFilesMatchingFilter(ctx *conflictContext, filter iceberg.BooleanExpression) error {
	if len(ctx.concurrent) == 0 {
		return nil
	}
	if filter == nil {
		filter = iceberg.AlwaysTrue{}
	}

	// Per-spec projected filter, memoized per spec id. Concurrent
	// snapshots may have been written against different spec ids
	// after a spec evolution, so each needs its own projection.
	// Reuses the same projection/evaluator helpers the scanner uses
	// (buildPartitionProjection, buildManifestEvaluator,
	// buildPartitionEvaluator) so there is one code path that pruning
	// semantics flow through.
	partitionFilters := newKeyDefaultMapWrapErr(func(specID int) (iceberg.BooleanExpression, error) {
		return buildPartitionProjection(specID, ctx.current, ctx.current.CurrentSchema(), filter, ctx.caseSensitive)
	})
	manifestEvals := newKeyDefaultMapWrapErr(func(specID int) (func(iceberg.ManifestFile) (bool, error), error) {
		return buildManifestEvaluator(specID, ctx.current, ctx.current.CurrentSchema(), partitionFilters, ctx.caseSensitive)
	})
	partitionEvals := newKeyDefaultMapWrapErr(func(specID int) (func(iceberg.DataFile) (bool, error), error) {
		return buildPartitionEvaluator(specID, ctx.current, ctx.current.CurrentSchema(), partitionFilters, ctx.caseSensitive)
	})
	// Eval copies each file's metrics into fresh evaluator state, so this
	// evaluator can be reused across concurrent manifests.
	// includeEmptyFiles=false is intentional: an empty file has no rows that
	// can match the filter and must not create a conflict.
	metricsEval, err := newInclusiveMetricsEvaluator(ctx.current.CurrentSchema(), filter, ctx.caseSensitive, false)
	if err != nil {
		return fmt.Errorf("failed to build metrics evaluator: %w", err)
	}

	for _, snap := range ctx.concurrent {
		manifests, err := ctx.manifestsFor(snap)
		if err != nil {
			return fmt.Errorf("loading manifests for concurrent snapshot %d: %w", snap.SnapshotID, err)
		}
		for _, mf := range manifests {
			if mf.ManifestContent() != iceberg.ManifestContentData {
				continue
			}

			mEval, err := manifestEvals.Get(int(mf.PartitionSpecID()))
			if err != nil {
				return fmt.Errorf("failed to build manifest evaluator for spec %d: %w", mf.PartitionSpecID(), err)
			}
			keep, err := mEval(mf)
			if err != nil {
				return err
			}
			if !keep {
				continue
			}

			pEval, err := partitionEvals.Get(int(mf.PartitionSpecID()))
			if err != nil {
				return fmt.Errorf("failed to build partition evaluator for spec %d: %w", mf.PartitionSpecID(), err)
			}
			for e, err := range mf.Entries(ctx.manifestReadIO(), false) {
				if err != nil {
					return fmt.Errorf("reading entries from manifest %s: %w", mf.FilePath(), err)
				}
				if e.Status() != iceberg.EntryStatusADDED || e.SnapshotID() != snap.SnapshotID {
					continue
				}
				matches, err := pEval(e.DataFile())
				if err != nil {
					return err
				}
				if !matches {
					continue
				}
				matches, err = metricsEval(e.DataFile())
				if err != nil {
					return fmt.Errorf("evaluating metrics for data file %s: %w", e.DataFile().FilePath(), err)
				}
				if !matches {
					continue
				}

				return fmt.Errorf("%w: snapshot %d added data file %s matching filter %s",
					ErrConflictingDataFiles, snap.SnapshotID, e.DataFile().FilePath(), filter)
			}
		}
	}

	return nil
}

// validateNoConflictingDataFiles is the serializable-isolation check
// used by RowDelta and other delete operations. It rejects the commit
// if any concurrent snapshot added data files into the partition(s)
// the committer is modifying.
//
// Under IsolationSnapshot this validator is a no-op: concurrent
// appends are allowed and will simply land in the same partition
// alongside the new deletes.
func validateNoConflictingDataFiles(ctx *conflictContext, filter iceberg.BooleanExpression, level IsolationLevel) error {
	if level != IsolationSerializable {
		return nil
	}

	return validateAddedDataFilesMatchingFilter(ctx, filter)
}

// validateNoConflictingDataFilesInPartitions is like
// validateNoConflictingDataFiles but scoped to the partitions touched
// by equality-delete files. It builds an OR-of-equalities filter from
// the provided partition tuples and delegates to
// validateAddedDataFilesMatchingFilter, which performs per-spec
// projection, manifest-summary pruning, and type-aware evaluation via
// iceberg.Literal — making it safe for UUID, decimal, binary, fixed,
// and future partition types, and correct across partition-spec
// evolution because each concurrent manifest is projected against its
// own spec ID.
//
// Callers are responsible for ensuring the table is partitioned
// (i.e. at least one partition field exists) before calling this
// function. For unpartitioned tables, call
// validateNoConflictingDataFiles(ctx, iceberg.AlwaysTrue{}, level)
// directly.
//
// Under IsolationSnapshot this validator is a no-op.
func validateNoConflictingDataFilesInPartitions(ctx *conflictContext, eqDeleteFiles []iceberg.DataFile, level IsolationLevel) error {
	if level != IsolationSerializable {
		return nil
	}

	if len(ctx.concurrent) == 0 || len(eqDeleteFiles) == 0 {
		return nil
	}

	filter, err := eqDeletePartitionsToFilter(eqDeleteFiles, ctx.current)
	if err != nil {
		return fmt.Errorf("building partition conflict filter: %w", err)
	}

	return validateNoConflictingDataFiles(ctx, filter, level)
}

type equalityDeletePartitionSpecInfo struct {
	fields       map[int]iceberg.PartitionField
	sourceFields map[int]equalityDeleteSourceFieldInfo
}

type equalityDeleteSourceFieldInfo struct {
	name  string
	found bool
}

// eqDeletePartitionsToFilter converts equality-delete data files into an
// OR-of-ANDs BooleanExpression in row (source) space, suitable for passing
// to validateAddedDataFilesMatchingFilter.
//
// For each distinct (spec ID, partition) tuple it resolves each partition
// field ID to the source schema field name via the file's partition spec, then
// builds an EqualTo predicate using Reference(sourceFieldName). Multiple
// fields within one partition are AND-ed; multiple partitions are OR-ed.
//
// The resulting expression is projected per-concurrent-manifest's spec ID
// inside validateAddedDataFilesMatchingFilter (via buildPartitionProjection),
// ensuring correct conflict detection even after partition-spec evolution.
//
// An empty partition tuple (unpartitioned delete) returns AlwaysTrue so the
// caller falls back to the conservative full-table scan. Callers should
// normally guard against calling this function for unpartitioned tables (see
// RowDelta.validate).
func eqDeletePartitionsToFilter(files []iceberg.DataFile, meta Metadata) (iceberg.BooleanExpression, error) {
	terms := make([]iceberg.BooleanExpression, 0, len(files))
	specInfoByID := make(map[int]*equalityDeletePartitionSpecInfo)
	seen := make(map[string]struct{}, len(files))
	var currentSchema *iceberg.Schema

	for _, f := range files {
		p := iceberginternal.BorrowedDataFilePartition(f)
		if len(p) == 0 {
			return iceberg.AlwaysTrue{}, nil
		}

		specID := int(f.SpecID())
		specInfo, ok := specInfoByID[specID]
		if !ok {
			spec := meta.PartitionSpecByID(specID)
			if spec == nil {
				return nil, fmt.Errorf("partition spec ID %d not found in metadata", f.SpecID())
			}

			specInfo = &equalityDeletePartitionSpecInfo{
				fields:       make(map[int]iceberg.PartitionField, spec.NumFields()),
				sourceFields: make(map[int]equalityDeleteSourceFieldInfo, spec.NumFields()),
			}
			for _, pf := range spec.Fields() {
				specInfo.fields[pf.FieldID] = pf
			}
			specInfoByID[specID] = specInfo
		}

		// Sort partition field IDs for deterministic expression order.
		fieldIDs := make([]int, 0, len(p))
		for id := range p {
			fieldIDs = append(fieldIDs, id)
		}
		sort.Ints(fieldIDs)

		// Non-identity transforms (bucket, day, hour, truncate, year, month) store
		// post-transform values in DataFile.Partition(). Building a row-space predicate
		// with those values would cause validateAddedDataFilesMatchingFilter to
		// re-apply the transform, producing wrong matches (double-transformation).
		// Fall back to AlwaysTrue (conservative: treat the eq-delete as table-wide)
		// until a full PartitionSet-style partition-space approach is added.
		identityOnly := true
		for _, fid := range fieldIDs {
			if pf, ok := specInfo.fields[fid]; ok {
				if _, isIdentity := pf.Transform.(iceberg.IdentityTransform); !isIdentity {
					identityOnly = false

					break
				}
			}
		}
		if !identityOnly {
			return iceberg.AlwaysTrue{}, nil
		}

		// canonicalPartitionKey is injective for the supported partition values
		// and includes the spec ID. If a custom Literal implementation is not
		// part of that closed set, skip deduplication and let the existing literal
		// conversion below preserve its behavior.
		if key, err := canonicalPartitionKey(f.SpecID(), p); err == nil {
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			seen[key] = struct{}{}
		}

		conjuncts := make([]iceberg.BooleanExpression, 0, len(p))
		for _, partFieldID := range fieldIDs {
			pf, ok := specInfo.fields[partFieldID]
			if !ok {
				return nil, fmt.Errorf("partition field ID %d not found in spec %d", partFieldID, f.SpecID())
			}

			// Resolve each source field at most once per partition spec. The
			// current schema is cloned lazily, and only once for this filter.
			sourceField, resolved := specInfo.sourceFields[partFieldID]
			if !resolved {
				if currentSchema == nil {
					currentSchema = meta.CurrentSchema()
				}
				field, found := currentSchema.FindFieldByID(pf.SourceID())
				sourceField = equalityDeleteSourceFieldInfo{name: field.Name, found: found}
				specInfo.sourceFields[partFieldID] = sourceField
			}
			if !sourceField.found {
				return nil, fmt.Errorf("source field ID %d (partition field %q) not found in schema", pf.SourceID(), pf.Name)
			}

			value := p[partFieldID]
			if value == nil {
				conjuncts = append(conjuncts, iceberg.IsNull(iceberg.Reference(sourceField.name)))

				continue
			}

			lit, err := internal.LiteralForPartitionValue(value)
			if err != nil {
				return nil, fmt.Errorf("partition field %q: %w", sourceField.name, err)
			}

			conjuncts = append(conjuncts, iceberg.LiteralPredicate(iceberg.OpEQ, iceberg.Reference(sourceField.name), lit))
		}

		if len(conjuncts) == 1 {
			terms = append(terms, conjuncts[0])
		} else {
			terms = append(terms, iceberg.NewAnd(conjuncts[0], conjuncts[1], conjuncts[2:]...))
		}
	}

	if len(terms) == 0 {
		return iceberg.AlwaysTrue{}, nil
	}

	if len(terms) == 1 {
		return terms[0], nil
	}

	return iceberg.NewOr(terms[0], terms[1], terms[2:]...), nil
}

// validateNoNewDeletesForRewrittenFiles rejects the commit if any
// concurrent snapshot added delete files that would be lost when the
// committer's rewrite replaces one of rewrittenFiles.
//
// Two cases are flagged:
//
//   - A position-delete that resolves to one of the rewritten paths,
//     via referencedDataFilePath (explicit referenced_data_file, then
//     equal file_path lower/upper bounds), or, when no single path can
//     be resolved, whose (specID, partition) matches a rewritten file
//     — the delete applies to a file the rewrite is removing, so it
//     would be orphaned and its semantics lost. This mirrors Java's
//     DeleteFileIndex, which indexes path-scoped deletes by path and
//     falls back to indexing partition-scoped deletes by (specId,
//     partition).
//
//   - An equality-delete added by a concurrent snapshot — these apply
//     by predicate, not by path, so we cannot precisely check overlap
//     without the rewritten files' partition tuples. The conservative
//     behavior is to reject any concurrent eq-delete, which matches
//     Java's approach for RewriteFiles and avoids silently losing
//     deletes. A follow-up PR will accept optional partition-overlap
//     hints to narrow this check.
func validateNoNewDeletesForRewrittenFiles(ctx *conflictContext, rewrittenFiles []iceberg.DataFile) error {
	if len(rewrittenFiles) == 0 || len(ctx.concurrent) == 0 {
		return nil
	}

	rewrittenPaths := make(map[string]struct{}, len(rewrittenFiles))
	rewrittenPartitions := make(map[string]struct{}, len(rewrittenFiles))
	for _, df := range rewrittenFiles {
		rewrittenPaths[df.FilePath()] = struct{}{}
		key, err := canonicalPartitionKey(df.SpecID(), dataFilePartition(df))
		if err != nil {
			return fmt.Errorf("building partition conflict key for rewritten file %s (spec %d): %w",
				df.FilePath(), df.SpecID(), err)
		}
		rewrittenPartitions[key] = struct{}{}
	}

	return ctx.forEachAddedEntry(iceberg.ManifestContentDeletes, func(snap Snapshot, e iceberg.ManifestEntry) error {
		df := e.DataFile()
		switch df.ContentType() {
		case iceberg.EntryContentPosDeletes:
			if path := referencedDataFilePath(df); path != "" {
				if _, overlap := rewrittenPaths[path]; overlap {
					return fmt.Errorf("%w: snapshot %d added pos-delete %s referencing rewritten file %s",
						ErrConflictingDeleteFiles, snap.SnapshotID, df.FilePath(), path)
				}

				return nil
			}

			key, err := canonicalPartitionKey(df.SpecID(), dataFilePartition(df))
			if err != nil {
				return fmt.Errorf("building partition conflict key for concurrent pos-delete %s (spec %d): %w",
					df.FilePath(), df.SpecID(), err)
			}
			if _, overlap := rewrittenPartitions[key]; overlap {
				return fmt.Errorf("%w: snapshot %d added partition-scoped pos-delete %s overlapping rewritten partition (spec %d)",
					ErrConflictingDeleteFiles, snap.SnapshotID, df.FilePath(), df.SpecID())
			}

			return nil

		case iceberg.EntryContentEqDeletes:
			// Conservative: any concurrent eq-delete is a conflict
			// for a rewrite, because the predicate could cover a
			// rewritten file. Follow-up: accept partition-overlap
			// hints to narrow this.
			return fmt.Errorf("%w: snapshot %d added equality delete %s during rewrite",
				ErrConflictingDeleteFiles, snap.SnapshotID, df.FilePath())
		}

		return nil
	})
}

// referencedDataFilePath resolves the single data file a pos-delete
// targets, mirroring Java ContentFileUtil.referencedDataFile: explicit
// referenced_data_file first, then equal file_path lower/upper bounds.
// Returns "" when the delete is partition-scoped (no single data file
// can be determined), which callers must treat conservatively via a
// partition-tuple match rather than skipping the delete.
func referencedDataFilePath(df iceberg.DataFile) string {
	if ref := iceberginternal.BorrowedDataFileReferencedDataFile(df); ref != nil && *ref != "" {
		return *ref
	}

	lowerBounds, upperBounds := iceberginternal.BorrowedDataFileBounds(df)
	lower := lowerBounds[filePathFieldID]
	upper := upperBounds[filePathFieldID]
	if len(lower) == 0 || !bytes.Equal(lower, upper) {
		return ""
	}

	lit, err := iceberg.LiteralFromBytes(iceberg.PrimitiveTypes.String, lower)
	if err != nil {
		return ""
	}
	value, ok := lit.(iceberg.TypedLiteral[string])
	if !ok {
		return ""
	}

	return value.Value()
}

// filePathFieldID is the reserved field ID of the file_path column in a
// positional delete file, sourced from the canonical delete schema.
var filePathFieldID = func() int {
	f, _ := iceberg.PositionalDeleteSchema.FindFieldByName("file_path")

	return f.ID
}()

// canonicalPartitionKey returns a deterministic, injective string key for a
// (specID, partition) tuple. Partition values are compared post-transform
// (direct equality), matching Java's DeleteFileIndex — no transform-aware
// logic is needed because both sides store the same post-transform
// representation.
//
// Values are encoded type-normalized across the known Avro-decode and
// write-side representations (see appendCanonicalPartitionValue):
// integer-backed types are unified per field, so e.g. an int32 and an
// iceberg.Date carrying the same day match. An unknown value type fails the
// key build rather than falling back to lossy formatting that could silently
// mismatch.
//
// Different spec IDs never match, even with identical-looking tuples:
// Java's DeleteFileIndex keys its PartitionMap by specId, and field IDs
// are only meaningful within a single spec.
func canonicalPartitionKey(specID int32, partition map[int]any) (string, error) {
	var idStorage [8]int
	ids := idStorage[:0]
	if len(partition) > len(idStorage) {
		ids = make([]int, 0, len(partition))
	}
	for id := range partition {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	var bufStorage [256]byte
	buf := strconv.AppendInt(bufStorage[:0], int64(specID), 10)
	buf = append(buf, '|')
	for _, id := range ids {
		buf = strconv.AppendInt(buf, int64(id), 10)
		buf = append(buf, ':')
		var err error
		buf, err = appendCanonicalPartitionValue(buf, partition[id])
		if err != nil {
			return "", fmt.Errorf("partition field %d: %w", id, err)
		}
		buf = append(buf, ';')
	}

	return string(buf), nil
}

// appendCanonicalPartitionValue encodes one partition value into dst by
// semantic class, so different Go representations of the same logical
// value produce the same bytes:
//
//   - integer-backed types (int/int32/int64, Date, Time, Timestamp,
//     TimestampNano) collapse into one class — within a single
//     (specID, fieldID) the logical type is fixed, so this only
//     tolerates decoder representation drift, never conflates two
//     logical values of one field;
//   - uuid.UUID and []byte collapse into raw-bytes hex;
//   - Decimal and DecimalLiteral collapse into (scale, unscaled).
//
// String values are length-prefixed so keys stay injective for inputs
// containing the field separators. Unknown types return an error.
func appendCanonicalPartitionValue(dst []byte, v any) ([]byte, error) {
	switch v := v.(type) {
	case nil:
		return append(dst, 'z'), nil
	case bool:
		return strconv.AppendBool(append(dst, 'b', ':'), v), nil
	case int:
		return strconv.AppendInt(append(dst, 'i', ':'), int64(v), 10), nil
	case int32:
		return strconv.AppendInt(append(dst, 'i', ':'), int64(v), 10), nil
	case int64:
		return strconv.AppendInt(append(dst, 'i', ':'), v, 10), nil
	case iceberg.Date:
		return strconv.AppendInt(append(dst, 'i', ':'), int64(v), 10), nil
	case iceberg.Time:
		return strconv.AppendInt(append(dst, 'i', ':'), int64(v), 10), nil
	case iceberg.Timestamp:
		return strconv.AppendInt(append(dst, 'i', ':'), int64(v), 10), nil
	case iceberg.TimestampNano:
		return strconv.AppendInt(append(dst, 'i', ':'), int64(v), 10), nil
	case float32:
		return strconv.AppendFloat(append(dst, 'f', ':'), float64(v), 'x', -1, 64), nil
	case float64:
		return strconv.AppendFloat(append(dst, 'f', ':'), v, 'x', -1, 64), nil
	case string:
		dst = strconv.AppendInt(append(dst, 's', ':'), int64(len(v)), 10)

		return append(append(dst, ':'), v...), nil
	case []byte:
		return hex.AppendEncode(append(dst, 'B', ':'), v), nil
	case uuid.UUID:
		return hex.AppendEncode(append(dst, 'B', ':'), v[:]), nil
	case iceberg.Decimal:
		return appendCanonicalPartitionDecimal(dst, v), nil
	case iceberg.DecimalLiteral:
		return appendCanonicalPartitionDecimal(dst, iceberg.Decimal(v)), nil
	default:
		return nil, fmt.Errorf("unsupported partition value type %T in canonical key", v)
	}
}

func appendCanonicalPartitionDecimal(dst []byte, v iceberg.Decimal) []byte {
	dst = strconv.AppendInt(append(dst, 'd', ':'), int64(v.Scale), 10)

	return append(append(dst, ':'), v.Val.BigInt().String()...)
}
