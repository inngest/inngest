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
	stdfs "io/fs"
	"log/slog"
	"maps"
	"net/url"
	"os"
	pathpkg "path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/apache/iceberg-go"
	iceberginternal "github.com/apache/iceberg-go/internal"
	"github.com/apache/iceberg-go/internal/fileuri"
	iceio "github.com/apache/iceberg-go/io"
	"golang.org/x/sync/errgroup"
)

// PrefixMismatchMode defines how to handle cases where candidate files have different
// URI schemes or authorities compared to table location during orphan cleanup.
// This is useful when files may be referenced using different but equivalent schemes
// (e.g., s3:// vs s3a:// vs s3n://) or when cleaning up files across different locations.
type PrefixMismatchMode int

const (
	// PrefixMismatchError causes cleanup to fail with an error when candidate files
	// have URI schemes/authorities that don't match the table location and are not
	// covered by configured equivalences. This is the safest default behavior.
	PrefixMismatchError PrefixMismatchMode = iota // default

	// PrefixMismatchIgnore skips candidate files that have mismatched URI schemes/authorities
	// without treating it as an error. Files are silently ignored and not considered for deletion.
	PrefixMismatchIgnore

	// PrefixMismatchDelete treats candidate files with mismatched URI schemes/authorities
	// as orphans and includes them for deletion. Use with caution as this may delete
	// files from unexpected locations.
	PrefixMismatchDelete
)

func (p PrefixMismatchMode) String() string {
	switch p {
	case PrefixMismatchError:
		return "ERROR"
	case PrefixMismatchIgnore:
		return "IGNORE"
	case PrefixMismatchDelete:
		return "DELETE"
	default:
		return "UNKNOWN"
	}
}

// OrphanCleanupConfig holds configuration for orphan file cleanup operations.
type orphanCleanupConfig struct {
	location           string
	olderThan          time.Duration
	dryRun             bool
	deleteFunc         func(string) error
	maxConcurrency     int
	prefixMismatchMode PrefixMismatchMode
	equalSchemes       map[string]string
	equalAuthorities   map[string]string
	executingPlan      bool
	invalidPlanOption  error
	validationErr      error
}

const defaultPurgeMaxConcurrency = 32

type OrphanCleanupOption func(*orphanCleanupConfig)

func WithLocation(location string) OrphanCleanupOption {
	return func(cfg *orphanCleanupConfig) {
		rejectPlanOption(cfg, "WithLocation")
		cfg.location = location
	}
}

func WithFilesOlderThan(duration time.Duration) OrphanCleanupOption {
	return func(cfg *orphanCleanupConfig) {
		rejectPlanOption(cfg, "WithFilesOlderThan")
		cfg.olderThan = duration
		if duration < 0 && cfg.validationErr == nil {
			cfg.validationErr = errors.New("orphan cleanup age must be non-negative")
		}
	}
}

func WithDryRun(enabled bool) OrphanCleanupOption {
	return func(cfg *orphanCleanupConfig) {
		cfg.dryRun = enabled
	}
}

// WithDeleteFunc sets a custom delete function. If not provided, the table's FileIO
// delete method will be used.
func WithDeleteFunc(deleteFunc func(string) error) OrphanCleanupOption {
	return func(cfg *orphanCleanupConfig) {
		cfg.deleteFunc = deleteFunc
	}
}

// WithCleanupMaxConcurrency sets the maximum number of goroutines for parallel deletion.
// Defaults to a reasonable number based on the system. Only used when deleteFunc is nil or when
// the FileIO doesn't support bulk operations.
func WithCleanupMaxConcurrency(maxWorkers int) OrphanCleanupOption {
	return func(cfg *orphanCleanupConfig) {
		if maxWorkers > 0 {
			cfg.maxConcurrency = maxWorkers
		}
	}
}

// WithPrefixMismatchMode sets how to handle situations when metadata references files
// that match listed files except for authority/scheme differences.
func WithPrefixMismatchMode(mode PrefixMismatchMode) OrphanCleanupOption {
	return func(cfg *orphanCleanupConfig) {
		rejectPlanOption(cfg, "WithPrefixMismatchMode")
		cfg.prefixMismatchMode = mode
	}
}

// WithEqualSchemes specifies schemes that should be considered equivalent.
// For example, map["s3,s3a,s3n"] = "s3" treats all S3 scheme variants as equivalent.
// The key can be a comma-separated list of schemes that map to the value scheme.
func WithEqualSchemes(schemes map[string]string) OrphanCleanupOption {
	return func(cfg *orphanCleanupConfig) {
		rejectPlanOption(cfg, "WithEqualSchemes")
		if cfg.equalSchemes == nil {
			cfg.equalSchemes = make(map[string]string)
		}
		maps.Copy(cfg.equalSchemes, schemes)
	}
}

// WithEqualAuthorities specifies authorities that should be considered equivalent.
// For example, map["endpoint1.s3.amazonaws.com,endpoint2.s3.amazonaws.com"] = "s3.amazonaws.com"
// treats different S3 endpoints as equivalent. The key can be a comma-separated list.
func WithEqualAuthorities(authorities map[string]string) OrphanCleanupOption {
	return func(cfg *orphanCleanupConfig) {
		rejectPlanOption(cfg, "WithEqualAuthorities")
		if cfg.equalAuthorities == nil {
			cfg.equalAuthorities = make(map[string]string)
		}
		maps.Copy(cfg.equalAuthorities, authorities)
	}
}

// flattenURIEquivalences expands comma-separated URI equivalence groups into
// direct lookups. This intentionally differs from Java's flattenMap (lines
// 392-403), which uses input iteration order for conflicts. Go map iteration is
// unordered, so groups are processed in sorted order and the lexicographically
// last overlapping group wins. Exact mappings are overlaid afterward so they
// retain precedence over groups.
// https://github.com/apache/iceberg/blob/07c088fce9c54369864dcb6da16006e78206048b/spark/v3.5/spark/src/main/java/org/apache/iceberg/spark/actions/DeleteOrphanFilesSparkAction.java#L392-L403
func flattenURIEquivalences(equivalences map[string]string) map[string]string {
	if len(equivalences) == 0 {
		return nil
	}

	groups := make([]string, 0, len(equivalences))
	for group := range equivalences {
		groups = append(groups, group)
	}
	slices.Sort(groups)

	flattened := make(map[string]string, len(equivalences))
	for _, group := range groups {
		if !strings.Contains(group, ",") {
			continue
		}

		for _, value := range strings.Split(group, ",") {
			flattened[strings.TrimSpace(value)] = equivalences[group]
		}
	}

	for _, group := range groups {
		// Group declarations are configuration syntax, not URI lookup keys.
		if strings.Contains(group, ",") {
			continue
		}
		flattened[group] = equivalences[group]
	}

	return flattened
}

type OrphanCleanupResult struct {
	// OrphanFileLocations is retained for backward compatibility with callers
	// that consume only orphan paths. Prefer OrphanFiles for canonical path+size data.
	OrphanFileLocations []string
	// OrphanFiles is the canonical richer orphan result, carrying both path and size.
	OrphanFiles  []OrphanFile
	DeletedFiles []string
	// TotalSizeBytes is the combined size of orphan files only, not all scanned files.
	TotalSizeBytes int64
}

type OrphanFile struct {
	Path      string
	SizeBytes int64
}

// OrphanCleanupPlan contains the exact orphan files identified during one scan.
// The file list is private so callers can only obtain copies, keeping the plan
// stable between confirmation and execution.
type OrphanCleanupPlan struct {
	orphanFileLocations []string
	orphanFiles         []OrphanFile
	totalSizeBytes      int64
	cutoff              time.Time
}

// Files returns a copy of the files in the cleanup plan.
func (p OrphanCleanupPlan) Files() []string {
	return slices.Clone(p.orphanFileLocations)
}

// OrphanFiles returns copies of the path and size entries in the cleanup plan.
func (p OrphanCleanupPlan) OrphanFiles() []OrphanFile {
	return slices.Clone(p.orphanFiles)
}

// TotalSizeBytes returns the combined size of files in the cleanup plan.
func (p OrphanCleanupPlan) TotalSizeBytes() int64 {
	return p.totalSizeBytes
}

// Cutoff returns the age cutoff used to create the cleanup plan. It is
// informational only; executing a plan does not re-evaluate file ages.
func (p OrphanCleanupPlan) Cutoff() time.Time {
	return p.cutoff
}

func (p OrphanCleanupPlan) result() OrphanCleanupResult {
	return OrphanCleanupResult{
		OrphanFileLocations: p.Files(),
		OrphanFiles:         p.OrphanFiles(),
		TotalSizeBytes:      p.totalSizeBytes,
	}
}

func newOrphanCleanupConfig(opts ...OrphanCleanupOption) *orphanCleanupConfig {
	return newOrphanCleanupConfigWithMode(false, opts...)
}

func newExecutionOrphanCleanupConfig(opts ...OrphanCleanupOption) *orphanCleanupConfig {
	return newOrphanCleanupConfigWithMode(true, opts...)
}

func newOrphanCleanupConfigWithMode(executingPlan bool, opts ...OrphanCleanupOption) *orphanCleanupConfig {
	cfg := &orphanCleanupConfig{
		location:           "",             // empty means use table's data location
		olderThan:          72 * time.Hour, // 3 days ago
		dryRun:             false,
		deleteFunc:         nil,
		maxConcurrency:     runtime.GOMAXPROCS(0), // default to number of CPUs
		prefixMismatchMode: PrefixMismatchError,   // default to safest mode
		equalSchemes:       nil,                   // no scheme equivalence by default
		equalAuthorities:   nil,                   // no authority equivalence by default
		executingPlan:      executingPlan,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	cfg.equalSchemes = flattenURIEquivalences(cfg.equalSchemes)
	cfg.equalAuthorities = flattenURIEquivalences(cfg.equalAuthorities)

	return cfg
}

func rejectPlanOption(cfg *orphanCleanupConfig, name string) {
	if cfg.executingPlan && cfg.invalidPlanOption == nil {
		cfg.invalidPlanOption = fmt.Errorf("%s is only valid while planning orphan cleanup", name)
	}
}

// DeleteOrphanFiles identifies files under a table location that are no longer
// referenced by table metadata and deletes them unless dry-run is enabled.
//
// The table filesystem must implement iceio.ListableIO so orphan cleanup can
// fully enumerate candidate files before deciding what is safe to delete.
func (t Table) DeleteOrphanFiles(ctx context.Context, opts ...OrphanCleanupOption) (OrphanCleanupResult, error) {
	cfg := newOrphanCleanupConfig(opts...)
	if cfg.validationErr != nil {
		return OrphanCleanupResult{}, cfg.validationErr
	}
	plan, err := t.planOrphanFiles(ctx, cfg)
	if err != nil {
		return OrphanCleanupResult{}, err
	}
	if cfg.dryRun {
		return plan.result(), nil
	}

	return t.executeOrphanCleanup(ctx, plan, cfg)
}

// PlanOrphanFiles identifies orphan files without deleting anything. The
// returned plan can be shown to a user and passed to ExecuteOrphanCleanup to
// delete exactly that set.
func (t Table) PlanOrphanFiles(ctx context.Context, opts ...OrphanCleanupOption) (OrphanCleanupPlan, error) {
	cfg := newOrphanCleanupConfig(opts...)
	if cfg.validationErr != nil {
		return OrphanCleanupPlan{}, cfg.validationErr
	}

	return t.planOrphanFiles(ctx, cfg)
}

// ExecuteOrphanCleanup deletes exactly the files in plan. It does not perform
// another orphan scan, so files appearing after planning are not included.
func (t Table) ExecuteOrphanCleanup(ctx context.Context, plan OrphanCleanupPlan, opts ...OrphanCleanupOption) (OrphanCleanupResult, error) {
	cfg := newExecutionOrphanCleanupConfig(opts...)
	if cfg.validationErr != nil {
		return OrphanCleanupResult{}, cfg.validationErr
	}
	if cfg.invalidPlanOption != nil {
		return OrphanCleanupResult{}, cfg.invalidPlanOption
	}
	if cfg.dryRun {
		return plan.result(), nil
	}

	return t.executeOrphanCleanup(ctx, plan, cfg)
}

type scannedFile struct {
	path string
	size int64
}

type referencedFileIndex struct {
	normalized map[string]struct{}
	byPath     map[string][]string
}

func (t Table) planOrphanFiles(ctx context.Context, cfg *orphanCleanupConfig) (OrphanCleanupPlan, error) {
	fs, err := t.fsF(ctx)
	if err != nil {
		return OrphanCleanupPlan{}, fmt.Errorf("failed to get filesystem: %w", err)
	}

	scanLocation := cfg.location
	if scanLocation == "" {
		scanLocation = t.metadata.Location()
	}

	// Run the S3 walk and referenced-file collection concurrently.
	// Each goroutine owns its variable exclusively — no shared writes.
	var referencedFiles map[string]bool
	var scannedFiles []scannedFile
	cutoff := time.Now().Add(-cfg.olderThan)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		referencedFiles, err = t.getReferencedFiles(gctx, fs, cfg.maxConcurrency, true)

		return err
	})

	g.Go(func() error {
		return walkDirectory(fs, scanLocation, func(path string, info stdfs.FileInfo) error {
			if gctx.Err() != nil {
				return gctx.Err()
			}
			if info.IsDir() || !info.ModTime().Before(cutoff) {
				return nil
			}
			scannedFiles = append(scannedFiles, scannedFile{path: path, size: info.Size()})

			return nil
		})
	})

	if err = g.Wait(); err != nil {
		return OrphanCleanupPlan{}, err
	}

	// Identify orphans.
	referencedIndex := newReferencedFileIndex(referencedFiles, cfg)

	var orphanFiles []string
	orphanFileEntries := make([]OrphanFile, 0)
	var totalOrphanSize int64
	for _, f := range scannedFiles {
		isOrphan, err := isFileOrphan(f.path, referencedFiles, referencedIndex, cfg)
		if err != nil {
			return OrphanCleanupPlan{}, fmt.Errorf("failed to identify orphan %s: %w", f.path, err)
		}
		if isOrphan {
			orphanFiles = append(orphanFiles, f.path)
			orphanFileEntries = append(orphanFileEntries, OrphanFile{
				Path:      f.path,
				SizeBytes: f.size,
			})
			totalOrphanSize += f.size
		}
	}

	return OrphanCleanupPlan{
		orphanFileLocations: orphanFiles,
		orphanFiles:         orphanFileEntries,
		totalSizeBytes:      totalOrphanSize,
		cutoff:              cutoff,
	}, nil
}

func (t Table) executeOrphanCleanup(ctx context.Context, plan OrphanCleanupPlan, cfg *orphanCleanupConfig) (OrphanCleanupResult, error) {
	fs, err := t.fsF(ctx)
	if err != nil {
		return OrphanCleanupResult{}, fmt.Errorf("failed to get filesystem: %w", err)
	}
	orphanFiles := plan.Files()
	deletedFiles, err := deleteFiles(ctx, fs, orphanFiles, cfg)
	if err != nil {
		return OrphanCleanupResult{}, fmt.Errorf("failed to delete orphan files: %w", err)
	}

	return OrphanCleanupResult{
		OrphanFileLocations: orphanFiles,
		OrphanFiles:         plan.OrphanFiles(),
		DeletedFiles:        deletedFiles,
		TotalSizeBytes:      plan.totalSizeBytes,
	}, nil
}

// getReferencedFiles collects all files referenced by table metadata: previous metadata
// files, statistics and partition-statistics paths (Puffin, etc.), and all paths reachable
// from current snapshots (manifest lists, manifests, data files).
//
// The collection uses a two-pass approach: first it reads every snapshot's manifest list
// (small files) to discover the set of unique manifest file paths, then it reads each
// unique manifest's entries in parallel. Manifests are immutable and shared across
// snapshots via copy-on-write, so deduplicating avoids redundant I/O that would
// otherwise grow as O(snapshots × manifests-per-snapshot).
//
// If the table has snapshots, fs must not be nil, otherwise an error is returned.
// Paths retain their original spelling so consumers can derive every applicable
// comparison identity before normalization discards information. The bool value
// distinguishes data files (true) from metadata files (false), which is used by
// PurgeFiles to respect gc.enabled.
func (t Table) getReferencedFiles(ctx context.Context, fs iceio.IO, maxConcurrency int, discardDeleted bool) (map[string]bool, error) {
	referenced := make(map[string]bool)
	metadata := t.metadata

	for entry := range metadata.PreviousFiles() {
		referenced[entry.MetadataFile] = false
	}
	referenced[t.metadataLocation] = false

	// Add version hint file (for Hadoop-style tables)
	// Following Java's ReachableFileUtil.versionHintLocation() logic:
	versionHintPath, err := versionHintLocation(metadata.Location())
	if err != nil {
		return nil, fmt.Errorf("failed to build version hint path: %w", err)
	}
	referenced[versionHintPath] = false

	for sf := range metadata.Statistics() {
		// Guard against malformed metadata; statistics-path is required per spec.
		if sf.StatisticsPath != "" {
			referenced[sf.StatisticsPath] = false
		}
	}
	for psf := range metadata.PartitionStatistics() {
		// Guard against malformed metadata; statistics-path is required per spec.
		if psf.StatisticsPath != "" {
			referenced[psf.StatisticsPath] = false
		}
	}

	snapshots := metadata.Snapshots()
	if len(snapshots) > 0 && fs == nil {
		return nil, errors.New("fs cannot be nil when table has snapshots")
	}

	// Pass 1: Read manifest lists (lightweight) to collect unique manifests.
	// Each snapshot writes only to its own result slot, so the reads can run in
	// parallel without locking or changing the order used for deduplication.
	manifestLists := make([][]iceberg.ManifestFile, len(snapshots))
	listGroup, listCtx := errgroup.WithContext(ctx)
	listGroup.SetLimit(max(1, min(maxConcurrency, len(snapshots))))
	for i := range snapshots {
		listGroup.Go(func() error {
			if err := listCtx.Err(); err != nil {
				return err
			}

			manifestFiles, err := snapshots[i].Manifests(fs)
			if err != nil {
				return fmt.Errorf("failed to read manifests for snapshot %d: %w", snapshots[i].SnapshotID, err)
			}
			if err := listCtx.Err(); err != nil {
				return err
			}

			manifestLists[i] = manifestFiles

			return nil
		})
	}
	if err := listGroup.Wait(); err != nil {
		return nil, err
	}

	uniqueManifests := make(map[string]iceberg.ManifestFile)
	for i, snapshot := range snapshots {
		if snapshot.ManifestList != "" {
			referenced[snapshot.ManifestList] = false
		}

		for _, manifest := range manifestLists[i] {
			path := manifest.FilePath()
			if _, ok := uniqueManifests[path]; !ok {
				uniqueManifests[path] = manifest
				referenced[path] = false
			}
		}
	}

	if len(uniqueManifests) == 0 {
		return referenced, nil
	}

	// Pass 2: Read entries from each unique manifest in parallel.
	type refEntry struct {
		path   string
		isData bool
	}
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(max(min(maxConcurrency, len(uniqueManifests)), 1))
	for _, m := range uniqueManifests {
		g.Go(func() error {
			var entries []refEntry
			// discardDeleted=true: skip DELETED-status entries when
			// computing the reachable file set. A DELETED entry is
			// not live in this snapshot and should not pin the file
			// against orphan cleanup once the snapshot that
			// originally held it live has been expired.
			// This matches iceberg-java and pyiceberg behavior.
			for entry, err := range m.Entries(fs, discardDeleted) {
				if err != nil {
					return fmt.Errorf("manifest %s: %w", m.FilePath(), err)
				}
				if gctx.Err() != nil {
					return gctx.Err()
				}
				// All files tracked within a manifest (data files, equality deletes, position deletes)
				// are considered "data files" for the purposes of gc.enabled.
				entries = append(entries, refEntry{
					path:   entry.DataFile().FilePath(),
					isData: true,
				})
				if ref := iceberginternal.BorrowedDataFileReferencedDataFile(entry.DataFile()); ref != nil {
					// This is a deletion vector entry referencing a data file.
					// Its FilePath() is the deletion vector (.dv) file itself (added above).
					// We must also mark the referenced data file as referenced.
					entries = append(entries, refEntry{
						path:   *ref,
						isData: true,
					})
				}
			}

			mu.Lock()
			for _, e := range entries {
				referenced[e.path] = e.isData
			}
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("failed to read manifest entries: %w", err)
	}

	return referenced, nil
}

func walkDirectory(fsys iceio.IO, root string, fn func(path string, info stdfs.FileInfo) error) error {
	if listable, ok := fsys.(iceio.ListableIO); ok {
		return listable.WalkDir(root, func(path string, d stdfs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			return fn(path, info)
		})
	}

	return fmt.Errorf("filesystem %T does not implement iceio.ListableIO", fsys)
}

func isFileOrphan(
	file string,
	referencedFiles map[string]bool,
	referencedIndex referencedFileIndex,
	cfg *orphanCleanupConfig,
) (bool, error) {
	normalizedFiles := normalizedFilePathAliases(file, cfg)
	normalizedFile := normalizedFiles[0]

	// Any presence in referencedFiles means referenced;
	// the bool distinguishes data vs metadata for gc.enabled, not membership"
	if _, ok := referencedFiles[file]; ok {
		return false, nil
	}
	if _, ok := referencedFiles[normalizedFile]; ok {
		return false, nil
	}

	for _, candidate := range normalizedFiles {
		if _, exists := referencedIndex.normalized[candidate]; exists {
			return false, nil
		}
	}

	references := referencedIndex.byPath[filePathKey(normalizedFile)]
	if len(references) == 0 {
		return true, nil
	}

	for _, referencedPath := range references {
		decision, err := checkPrefixMismatch(referencedPath, file, cfg)
		if err != nil {
			return false, err
		}
		if decision == prefixMatch {
			// Matching prefixes with different normalized URLs are distinct
			// object keys, for example key%2Fpart and key/part.
			continue
		}
		if decision == prefixMismatchKeep {
			return false, nil
		}
	}

	return true, nil
}

func newReferencedFileIndex(referencedFiles map[string]bool, cfg *orphanCleanupConfig) referencedFileIndex {
	index := referencedFileIndex{
		normalized: make(map[string]struct{}, len(referencedFiles)*2),
		byPath:     make(map[string][]string, len(referencedFiles)),
	}
	for referencedPath := range referencedFiles {
		normalizedPaths := normalizedFilePathAliases(referencedPath, cfg)
		for _, normalizedPath := range normalizedPaths {
			index.normalized[normalizedPath] = struct{}{}
		}
		index.normalized[referencedPath] = struct{}{}
		pathKey := filePathKey(normalizedPaths[0])
		index.byPath[pathKey] = append(index.byPath[pathKey], referencedPath)
	}
	for pathKey := range index.byPath {
		slices.Sort(index.byPath[pathKey])
	}

	return index
}

func deleteFiles(ctx context.Context, fs iceio.IO, orphanFiles []string, cfg *orphanCleanupConfig) ([]string, error) {
	if len(orphanFiles) == 0 {
		return nil, nil
	}

	// Use bulk delete when available and no custom deleteFunc is set.
	if cfg.deleteFunc == nil {
		if bulk, ok := fs.(iceio.BulkRemovableIO); ok {
			return bulk.DeleteFiles(ctx, orphanFiles)
		}
	}

	if cfg.maxConcurrency <= 1 {
		return deleteFilesSequential(ctx, fs, orphanFiles, cfg)
	}

	deleteFunc := fs.Remove
	if cfg.deleteFunc != nil {
		deleteFunc = cfg.deleteFunc
	}

	return deleteFilesParallel(
		ctx,
		orphanFiles,
		cfg.maxConcurrency,
		deleteFunc,
		func(file string, err error) error {
			return fmt.Errorf("failed to delete orphan file %s: %w", file, err)
		},
	)
}

func deleteFilesSequential(ctx context.Context, fs iceio.IO, orphanFiles []string, cfg *orphanCleanupConfig) ([]string, error) {
	var deletedFiles []string

	deleteFunc := fs.Remove
	if cfg.deleteFunc != nil {
		deleteFunc = cfg.deleteFunc
	}

	var (
		result          error
		cancellationErr error
	)
	for _, file := range orphanFiles {
		if err := ctx.Err(); err != nil {
			cancellationErr = err

			break
		}
		if err := deleteFunc(file); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to delete orphan file %s: %w", file, err))

			continue
		}
		deletedFiles = append(deletedFiles, file)
	}

	return deletedFiles, errors.Join(result, cancellationErr)
}

// If wrapError returns nil for a failed deletion, the original error is retained.
func deleteFilesParallel(
	ctx context.Context,
	files []string,
	maxConcurrency int,
	deleteFunc func(string) error,
	wrapError func(string, error) error,
) ([]string, error) {
	workers := min(max(maxConcurrency, 1), len(files))
	jobs := make(chan int)
	deleted := make([]bool, len(files))
	deleteErrors := make([]error, len(files))

	var cancellationErr error
	var cancellationOnce sync.Once
	recordCancellation := func(err error) {
		cancellationOnce.Do(func() {
			cancellationErr = err
		})
	}

	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					recordCancellation(ctx.Err())

					return
				case index, ok := <-jobs:
					if !ok {
						return
					}
					// Both cases can be ready after cancellation, so check the context
					// again before starting the deletion.
					if err := ctx.Err(); err != nil {
						recordCancellation(err)

						return
					}

					if err := deleteFunc(files[index]); err != nil {
						wrappedErr := wrapError(files[index], err)
						if wrappedErr == nil {
							// Keep failed deletions observable if the wrapper suppresses the error.
							wrappedErr = err
						}
						deleteErrors[index] = wrappedErr
					} else {
						deleted[index] = true
					}
				}
			}
		}()
	}

send:
	for index := range files {
		select {
		case jobs <- index:
		case <-ctx.Done():
			recordCancellation(ctx.Err())

			break send
		}
	}
	close(jobs)
	wg.Wait()

	deletedFiles := make([]string, 0, len(files))
	var allErrors []error
	for index, file := range files {
		if deleted[index] {
			deletedFiles = append(deletedFiles, file)
		}
		if deleteErrors[index] != nil {
			allErrors = append(allErrors, deleteErrors[index])
		}
	}
	if cancellationErr != nil {
		allErrors = append(allErrors, cancellationErr)
	}

	return deletedFiles, errors.Join(allErrors...)
}

// normalizeFilePath normalizes file paths for comparison by handling different
// path representations that might refer to the same file, with support for
// scheme/authority equivalence as specified in the configuration.
//
// This implementation is based on Apache Iceberg's Java DeleteOrphanFiles action:
// https://github.com/apache/iceberg/blob/07c088fce9c54369864dcb6da16006e78206048b/spark/v3.5/spark/src/main/java/org/apache/iceberg/spark/actions/DeleteOrphanFilesSparkAction.java#L1
//
// The normalization logic specifically follows the ToFileURI.toFileURI() method (lines 542-548):
// - Line 545: scheme = equalSchemes.getOrDefault(uri.getScheme(), uri.getScheme())
// - Line 546: authority = equalAuthorities.getOrDefault(uri.getAuthority(), uri.getAuthority())
//
// See also: https://iceberg.apache.org/docs/latest/maintenance/#remove-orphan-files
//
// Path normalization is essential for orphan cleanup because:
//  1. Files may be referenced using different but equivalent URI schemes (e.g., s3:// vs s3a:// vs s3n://)
//  2. Different authorities/endpoints may refer to the same storage (e.g., region-specific S3 endpoints)
//  3. Path separators and casing may vary across different systems and configurations
//  4. Without normalization, semantically identical paths would be treated as different,
//     leading to false positives in orphan detection
//
// normalizeFilePath normalizes a file path for comparison, handling schemes, authorities, and separators.
// It also aligns file:// URIs and bare local file system paths so they normalize to the same format.
func normalizeFilePath(path string) string {
	return normalizeFilePathWithConfig(path, nil)
}

func normalizeFilePathWithConfig(path string, cfg *orphanCleanupConfig) string {
	normalizedSeparators := strings.ReplaceAll(path, "\\", "/")
	// Native Windows volumes take precedence over URI parsing, matching LocalFS.
	// A path such as C://warehouse is drive-shaped even though it contains ://.
	if fileuri.HasWindowsDrivePrefix(normalizedSeparators) {
		return normalizeNonURLPath(path)
	}

	if strings.HasPrefix(strings.ToLower(path), "file:") {
		if fileURI, err := fileuri.Parse(path); err == nil {
			host := strings.ToLower(fileURI.Host())
			if host == "" || host == "localhost" {
				return normalizeNonURLPath(fileURI.LocalPathForOS())
			}
			if fileuri.IsWindowsDriveHost(fileURI.Host()) {
				return normalizeNonURLPath(fileURI.LocalPath(true))
			}
			// Remote authority – keep it as //host/path
			return normalizeNonURLPath("//" + fileURI.Host() + fileURI.LocalPath(false))
		}
	}

	// Handle URL-based paths (s3://, gs://, etc.)
	if strings.Contains(path, "://") {
		return normalizeURLPath(path, cfg)
	}

	return normalizeNonURLPath(path)
}

func normalizedFilePathAliases(path string, cfg *orphanCleanupConfig) []string {
	normalized := normalizeFilePathWithConfig(path, cfg)
	aliases := []string{normalized}
	appendAlias := func(alias string) {
		if !slices.Contains(aliases, alias) {
			aliases = append(aliases, alias)
		}
	}

	appendWindowsAliases := func(windowsPath string, collapseUNC bool) {
		appendAlias(windowsPath)
		appendAlias(strings.ToLower(windowsPath))
		if collapseUNC {
			collapsed := pathpkg.Clean(windowsPath)
			appendAlias(collapsed)
			appendAlias(strings.ToLower(collapsed))
		}
	}

	// Local file URIs use the same host interpretation as LocalFS. A portable
	// Windows interpretation is retained only as a conservative comparison
	// alias when it differs from the native path.
	if strings.HasPrefix(strings.ToLower(path), "file:") {
		if fileURI, err := fileuri.Parse(path); err == nil {
			host := strings.ToLower(fileURI.Host())
			if host == "" || host == "localhost" {
				appendWindowsAliases(normalizeNonURLPath(fileURI.LocalPath(true)), false)

				return aliases
			}
		}
	}

	// A path originally written as //server/share is ambiguous on non-Windows:
	// derive its POSIX and UNC identities independently before either cleaner
	// can discard dot segments or separator evidence. filepath.WalkDir may also
	// collapse the UNC identity's leading // for child paths.
	if runtime.GOOS != "windows" && isForwardSlashUNCPath(path) {
		posixPath := pathpkg.Clean(path)
		aliases = []string{posixPath}
		appendWindowsAliases(normalizeNonURLPath(path), true)

		return aliases
	}

	if isWindowsLocalPath(normalized) {
		appendWindowsAliases(normalized, false)
	}

	return aliases
}

func isForwardSlashUNCPath(path string) bool {
	if strings.Contains(path, `\`) {
		return false
	}

	volume, _, rooted := splitPortableVolume(path)

	return rooted && strings.HasPrefix(volume, "//")
}

func versionHintLocation(tableLocation string) (string, error) {
	if strings.HasPrefix(strings.ToLower(tableLocation), "file:") {
		fileURI, err := fileuri.Parse(tableLocation)
		if err != nil {
			return "", err
		}

		return fileURI.JoinPath("metadata", "version-hint.text"), nil
	}

	if strings.Contains(tableLocation, "://") {
		if _, ok := splitURLPath(tableLocation); !ok {
			return "", fmt.Errorf("invalid table location: %s", tableLocation)
		}

		// Remote object keys are opaque. Append the suffix without URL or path
		// joining, which would escape characters or clean duplicate slashes and
		// dot segments that may be meaningful parts of the key.
		separator := "/"
		if strings.HasSuffix(tableLocation, separator) {
			separator = ""
		}

		return tableLocation + separator + "metadata/version-hint.text", nil
	}

	return filepath.Join(tableLocation, "metadata", "version-hint.text"), nil
}

// normalizeURLPath normalizes URL-based file paths with scheme/authority equivalence.
//
// This function handles the complexities of cloud storage URIs where the same file
// can be referenced using different but semantically equivalent schemes and authorities.
//
// Examples of equivalent schemes (configured via equalSchemes):
//   - s3://bucket/path, s3a://bucket/path, s3n://bucket/path (all refer to S3)
//   - abfs://container@account.dfs.core.windows.net/path, abfss://container@account.dfs.core.windows.net/path (Azure)
//
// Examples of equivalent authorities (configured via equalAuthorities):
//   - s3://mybucket.s3.us-west-2.amazonaws.com/path vs s3://s3.us-west-2.amazonaws.com/mybucket/path
//   - Different regional endpoints that serve the same data
//
// Based on Apache Iceberg Java's DeleteOrphanFilesSparkAction.toFileURI() normalization (lines 542-548).
// https://github.com/apache/iceberg/blob/07c088fce9c54369864dcb6da16006e78206048b/spark/v3.5/spark/src/main/java/org/apache/iceberg/spark/actions/DeleteOrphanFilesSparkAction.java#L1
func normalizeURLPath(path string, cfg *orphanCleanupConfig) string {
	parts, ok := splitURLPath(path)
	if !ok {
		return normalizeNonURLPath(path)
	}

	var equalSchemes map[string]string
	var equalAuthorities map[string]string
	if cfg != nil {
		equalSchemes = cfg.equalSchemes
		equalAuthorities = cfg.equalAuthorities
	}

	normalizedScheme := applySchemeEquivalence(parts.scheme, equalSchemes)
	normalizedAuthority := applyAuthorityEquivalence(parts.rawAuthority, equalAuthorities)

	// Object-store paths are opaque keys. Keep their spelling exactly as
	// supplied: escaped separators, duplicate slashes, dot segments, queries,
	// and fragments can all be meaningful parts of a key. Only the explicitly
	// configured scheme and authority equivalences are normalized here.
	//
	// This intentionally differs from iceberg-java's Hadoop Path-based
	// normalization, which resolves dot segments. The Go object-store FileIO
	// preserves raw keys, so resolving them here would conflate distinct files.
	return normalizedScheme + "://" + normalizedAuthority + parts.rawSuffix
}

type urlPathParts struct {
	scheme       string
	rawAuthority string
	rawSuffix    string
}

// splitURLPath parses only a URL's scheme and authority. The remaining suffix
// is returned unchanged because object-store FileIO implementations treat it as
// an opaque object key, including invalid URL escapes such as %zz.
func splitURLPath(path string) (urlPathParts, bool) {
	schemeEnd := strings.Index(path, "://")
	if schemeEnd <= 0 {
		return urlPathParts{}, false
	}

	remainder := path[schemeEnd+3:]
	authorityEnd := strings.IndexAny(remainder, "/?#")
	if authorityEnd < 0 {
		authorityEnd = len(remainder)
	}
	rawAuthority := remainder[:authorityEnd]

	parsedPrefix, err := url.Parse(path[:schemeEnd+3+authorityEnd])
	if err != nil || parsedPrefix.Scheme == "" {
		return urlPathParts{}, false
	}

	return urlPathParts{
		scheme:       parsedPrefix.Scheme,
		rawAuthority: rawAuthority,
		rawSuffix:    remainder[authorityEnd:],
	}, true
}

// normalizeNonURLPath provides basic path normalization for non-URL paths.
//
// Handles file system paths by:
// 1. Converting Windows-style backslashes to forward slashes for consistency
// 2. Applying slash-based path cleaning to resolve "..", ".", and redundant separators
//
// This ensures that paths like "dir/./file", "dir//file", and "dir\file" (on Windows)
// all normalize to "dir/file" for consistent comparison.
func normalizeNonURLPath(path string) string {
	normalized := strings.ReplaceAll(path, "\\", "/")
	volume, remainder, rooted := splitPortableVolume(normalized)
	if volume == "" {
		return normalizeWindowsLocalPathCase(pathpkg.Clean(normalized))
	}
	if remainder == "" {
		if rooted && isWindowsDriveVolume(volume) {
			return normalizeWindowsLocalPathCase(volume + "/")
		}

		return normalizeWindowsLocalPathCase(volume)
	}

	if rooted {
		cleaned := pathpkg.Clean("/" + remainder)
		cleaned = strings.TrimPrefix(cleaned, "/")
		if cleaned == "" {
			if isWindowsDriveVolume(volume) {
				return normalizeWindowsLocalPathCase(volume + "/")
			}

			return normalizeWindowsLocalPathCase(volume)
		}

		return normalizeWindowsLocalPathCase(volume + "/" + cleaned)
	}

	return normalizeWindowsLocalPathCase(volume + pathpkg.Clean(remainder))
}

func normalizeWindowsLocalPathCase(path string) string {
	if !isWindowsLocalPath(path) {
		return path
	}

	// A slash-normalized UNC path can also name a case-sensitive POSIX path.
	// Keep its exact spelling on non-Windows hosts; comparison aliases provide
	// conservative Windows case folding without losing the POSIX identity.
	volume, _, rooted := splitPortableVolume(path)
	if runtime.GOOS != "windows" && rooted && strings.HasPrefix(volume, "//") {
		return path
	}

	return strings.ToLower(path)
}

func isWindowsLocalPath(path string) bool {
	if len(path) >= 2 && isDriveLetter(path[0]) && path[1] == ':' {
		return true
	}

	volume, _, rooted := splitPortableVolume(path)

	return rooted && strings.HasPrefix(volume, "//")
}

func splitPortableVolume(path string) (volume, remainder string, rooted bool) {
	if len(path) >= 2 && isDriveLetter(path[0]) && path[1] == ':' {
		if len(path) >= 3 && path[2] == '/' {
			return path[:2], path[3:], true
		}

		return path[:2], path[2:], false
	}

	if !strings.HasPrefix(path, "//") {
		return "", path, false
	}

	unc := strings.TrimPrefix(path, "//")
	serverEnd := strings.IndexByte(unc, '/')
	if serverEnd <= 0 || serverEnd+1 >= len(unc) {
		return "", path, false
	}

	shareStart := serverEnd + 1
	shareEnd := strings.IndexByte(unc[shareStart:], '/')
	if shareEnd < 0 {
		return "//" + unc, "", true
	}
	if shareEnd == 0 {
		return "", path, false
	}

	shareEnd += shareStart

	return "//" + unc[:shareEnd], unc[shareEnd+1:], true
}

func isDriveLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isWindowsDriveVolume(volume string) bool {
	return len(volume) == 2 && isDriveLetter(volume[0]) && volume[1] == ':'
}

// filePathKey returns the path component used to compare listed files with
// references before applying scheme and authority mismatch policy.
func filePathKey(file string) string {
	// Local file URIs use decoded path semantics. Handle them separately so
	// remote object-store suffixes can retain their raw spelling below.
	if strings.HasPrefix(strings.ToLower(file), "file:") {
		if _, err := fileuri.Parse(file); err == nil {
			return normalizeFilePath(file)
		}
	}

	if strings.Contains(file, "://") {
		if parts, ok := splitURLPath(file); ok {
			// Prefix-mismatch policy applies only when the object key itself is
			// identical. Keep the raw suffix so escaped separators, duplicate
			// slashes, and dot segments remain distinct object keys.
			return parts.rawSuffix
		}
	}

	return normalizeNonURLPath(file)
}

// applySchemeEquivalence maps schemes to their equivalent canonical form.
//
// Common equivalences include:
//   - S3: s3://, s3a://, s3n:// → s3://
//   - Azure: abfs://, abfss:// → abfs://
//   - HDFS: hdfs://, hdfs+webhdfs:// → hdfs://
//
// Based on Apache Iceberg Java's flattenMap() (lines 392-403) and EQUAL_SCHEMES_DEFAULT (line 102).
// https://github.com/apache/iceberg/blob/07c088fce9c54369864dcb6da16006e78206048b/spark/v3.5/spark/src/main/java/org/apache/iceberg/spark/actions/DeleteOrphanFilesSparkAction.java#L1
func applySchemeEquivalence(scheme string, equalSchemes map[string]string) string {
	if canonical, exists := equalSchemes[scheme]; exists {
		return canonical
	}

	return scheme
}

// applyAuthorityEquivalence maps authorities to their equivalent canonical form.
//
// Different cloud storage endpoints and authorities may serve the same logical storage,
// but appear different in URIs. This function normalizes these equivalent authorities
// to enable proper file matching during orphan cleanup.
//
// Common authority equivalences include:
//   - Regional S3 endpoints: s3.us-west-2.amazonaws.com, s3-us-west-2.amazonaws.com
//   - S3 path vs virtual-hosted style: bucket.s3.amazonaws.com vs s3.amazonaws.com/bucket
//   - Azure storage endpoints: account.dfs.core.windows.net, account.blob.core.windows.net
//   - Custom endpoints: minio.company.com, s3.company.local
//
// Based on Apache Iceberg Java's equalAuthorities logic (lines 546, 161-165, 392-403).
// https://github.com/apache/iceberg/blob/07c088fce9c54369864dcb6da16006e78206048b/spark/v3.5/spark/src/main/java/org/apache/iceberg/spark/actions/DeleteOrphanFilesSparkAction.java#L1
func applyAuthorityEquivalence(authority string, equalAuthorities map[string]string) string {
	if canonical, exists := equalAuthorities[authority]; exists {
		return canonical
	}

	// ADLS authorities use container@host. Preserve the container while still
	// allowing endpoint-only equivalence mappings to normalize the host.
	if userInfoEnd := strings.LastIndexByte(authority, '@'); userInfoEnd >= 0 {
		host := authority[userInfoEnd+1:]
		normalizedHost := applyAuthorityEquivalence(host, equalAuthorities)
		if normalizedHost != host {
			return authority[:userInfoEnd+1] + normalizedHost
		}
	}

	return authority
}

type prefixMismatchDecision int

const (
	prefixMatch prefixMismatchDecision = iota
	prefixMismatchKeep
	prefixMismatchDeleteCandidate
)

// checkPrefixMismatch decides how to handle prefix mismatches between referenced files and filesystem files.
func checkPrefixMismatch(referencedPath, filesystemPath string, cfg *orphanCleanupConfig) (prefixMismatchDecision, error) {
	refScheme, refAuth, refOK := pathPrefix(referencedPath)
	fsScheme, fsAuth, fsOK := pathPrefix(filesystemPath)
	if !refOK || !fsOK {
		return prefixMismatchKeep, nil
	}

	refScheme = applySchemeEquivalence(refScheme, cfg.equalSchemes)
	fsScheme = applySchemeEquivalence(fsScheme, cfg.equalSchemes)
	refAuth = applyAuthorityEquivalence(refAuth, cfg.equalAuthorities)
	fsAuth = applyAuthorityEquivalence(fsAuth, cfg.equalAuthorities)

	// Check for mismatches
	schemeMismatch := refScheme != fsScheme
	authMismatch := refAuth != fsAuth

	if !schemeMismatch && !authMismatch {
		return prefixMatch, nil
	}

	switch cfg.prefixMismatchMode {
	case PrefixMismatchError:
		return prefixMismatchKeep, fmt.Errorf("prefix mismatch detected: referenced=%s (scheme=%s, auth=%s) vs filesystem=%s (scheme=%s, auth=%s)",
			referencedPath, refScheme, refAuth, filesystemPath, fsScheme, fsAuth)
	case PrefixMismatchIgnore:
		return prefixMismatchKeep, nil
	case PrefixMismatchDelete:
		return prefixMismatchDeleteCandidate, nil
	default:
		return prefixMismatchKeep, fmt.Errorf("unknown prefix mismatch mode: %d", cfg.prefixMismatchMode)
	}
}

func pathPrefix(path string) (scheme, authority string, ok bool) {
	if strings.Contains(path, "://") {
		parts, ok := splitURLPath(path)
		if !ok {
			return "", "", false
		}

		return parts.scheme, parts.rawAuthority, true
	}

	parsed, err := url.Parse(path)
	if err != nil {
		return "", "", false
	}

	return parsed.Scheme, parsed.Host, true
}

// PurgeFiles physically deletes all files under the table's warehouse location
// and any referenced files written outside the location root (e.g., via write.data.path
// or write.metadata.path properties).
//
// It operates on a best-effort basis. Errors from individual file deletions are
// collected and returned together as a joined error. If files cannot be deleted
// (e.g. due to permission errors or missing paths), the errors are logged but
// the overall catalog drop operation should typically proceed so the catalog
// does not get out of sync with storage. Non-bulk deletion invokes Remove
// concurrently and is bounded to defaultPurgeMaxConcurrency operations because
// it is typically I/O-bound.
func (t Table) PurgeFiles(ctx context.Context) error {
	gcEnabled := isGCEnabled(t.Metadata().Properties())

	fs, err := t.FS(ctx)
	if err != nil {
		return fmt.Errorf("failed to load filesystem for table purge: %w", err)
	}

	var errs []error
	fileSet := make(map[string]string)
	location := t.metadata.Location()

	// 1. Walk the table location directory tree to capture all local files
	// Only walk the directory if gc.enabled=true to prevent accidental deletion
	// of unreferenced branched data files.
	if gcEnabled {
		if listable, ok := fs.(iceio.ListableIO); ok {
			walkErr := listable.WalkDir(location, func(path string, d stdfs.DirEntry, err error) error {
				if err := ctx.Err(); err != nil {
					return err
				}
				if err != nil {
					if os.IsNotExist(err) || errors.Is(err, stdfs.ErrNotExist) {
						return nil
					}

					return err
				}
				if !d.IsDir() {
					fileSet[normalizedFilePathAliases(path, nil)[0]] = path
				}

				return nil
			})
			if walkErr != nil && !os.IsNotExist(walkErr) && !errors.Is(walkErr, stdfs.ErrNotExist) {
				errs = append(errs, fmt.Errorf("failed walking directory %s: %w", location, walkErr))
			}
		}
	}

	// 2. Union in manifest-referenced and metadata files (which might be outside the table location)
	referencedFiles, refErr := t.getReferencedFiles(ctx, fs, runtime.GOMAXPROCS(0), false)
	if refErr != nil {
		return fmt.Errorf("failed to get referenced files: %w", refErr)
	}

	for path, isData := range referencedFiles {
		if !gcEnabled && isData {
			slog.WarnContext(ctx, "purge: skipping data file, gc.enabled=false", "path", path)

			continue
		}

		norm := normalizedFilePathAliases(path, nil)[0]
		if _, ok := fileSet[norm]; !ok {
			fileSet[norm] = path
		}
	}

	// Convert to slice and sort for deterministic behavior
	files := make([]string, 0, len(fileSet))
	for _, orig := range fileSet {
		files = append(files, orig)
	}
	slices.Sort(files)

	if len(files) > 0 {
		if bulk, ok := fs.(iceio.BulkRemovableIO); ok {
			_, bulkErr := bulk.DeleteFiles(ctx, files)
			if bulkErr != nil {
				errs = append(errs, fmt.Errorf("bulk deletion failed: %w", bulkErr))
			}
		} else {
			_, removeErr := deleteFilesParallel(
				ctx,
				files,
				defaultPurgeMaxConcurrency,
				func(file string) error {
					if err := fs.Remove(file); err != nil && !os.IsNotExist(err) {
						return err
					}

					return nil
				},
				func(file string, err error) error {
					return fmt.Errorf("failed to remove %s: %w", file, err)
				},
			)
			if removeErr != nil {
				errs = append(errs, removeErr)
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
