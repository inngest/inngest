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
	"io"
	"iter"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/compute"
	"github.com/apache/arrow-go/v18/arrow/compute/exprs"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/parquet/metadata"
	"github.com/apache/iceberg-go"
	iceinternal "github.com/apache/iceberg-go/internal"
	iceio "github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/table/dv"
	tblutils "github.com/apache/iceberg-go/table/internal"
	"github.com/apache/iceberg-go/table/substrait"
	"github.com/substrait-io/substrait-go/v8/expr"
	"golang.org/x/sync/errgroup"
)

const (
	ScanOptionArrowUseLargeTypes = "arrow.use_large_types"
	ScanOptionRowLineageEnabled  = "row_lineage.enabled"
	// This must stay a power of two because appendFilePathChunk uses a bit mask
	// for periodic cancellation checks.
	positionalDeleteCancellationCheckInterval = 16 * 1024
)

var PositionalDeleteArrowSchema, _ = SchemaToArrowSchema(iceberg.PositionalDeleteSchema, nil, true, false)

type (
	positionDeletes      = []*arrow.Chunked
	perFilePosDeletes    = map[string]positionDeletes
	perDeleteFileTargets = map[string]map[string]struct{}
)

// releasePerFilePosDeletes releases every Arrow chunk in a positional-delete
// map. Required on every error return between readAllDeleteFiles and the
// iterator returned by createIterator — Arrow allocations are not freed by
// GC, so dropping the map on the floor leaks the chunks. Safe to call on a
// nil map; the nil-chunk guard is defensive — position-delete readers never insert a
// nil *arrow.Chunked, but the guard keeps callers safe if that invariant
// ever changes.
func releasePerFilePosDeletes(deletesPerFile perFilePosDeletes) {
	for _, chunks := range deletesPerFile {
		for _, chunk := range chunks {
			if chunk != nil {
				chunk.Release()
			}
		}
	}
}

// collectPositionDeleteFilesAndTargets indexes each positional-delete file and
// the data-file paths of tasks that reference it. A nil target set preserves
// the whole-file fallback when any referencing task lacks a usable data path.
func collectPositionDeleteFilesAndTargets(tasks []FileScanTask) (map[string]iceberg.DataFile, perDeleteFileTargets) {
	uniqueDeletes := make(map[string]iceberg.DataFile)
	targetsByDelete := make(perDeleteFileTargets)

	for _, task := range tasks {
		for _, deleteFile := range task.DeleteFiles {
			if deleteFile.ContentType() != iceberg.EntryContentPosDeletes {
				continue
			}

			deletePath := deleteFile.FilePath()
			if _, ok := uniqueDeletes[deletePath]; !ok {
				uniqueDeletes[deletePath] = deleteFile
			}

			targets, ok := targetsByDelete[deletePath]
			if !ok {
				targets = make(map[string]struct{})
				targetsByDelete[deletePath] = targets
			}
			if targets == nil {
				continue
			}
			if task.File == nil || task.File.FilePath() == "" {
				targetsByDelete[deletePath] = nil

				continue
			}
			targets[task.File.FilePath()] = struct{}{}
		}
	}

	return uniqueDeletes, targetsByDelete
}

// readAllDeleteFiles reads every referenced positional-delete file up front. It
// remains the path used by Transaction.makePositionDeleteRecordsForFilter and by
// the eager-vs-lazy benchmark; arrowScan.GetRecords uses lazyPositionDeleteLoader.
func readAllDeleteFiles(ctx context.Context, fs iceio.IO, tasks []FileScanTask, concurrency int) (perFilePosDeletes, error) {
	deletesPerFile := make(perFilePosDeletes)
	uniqueDeletes, targetsByDelete := collectPositionDeleteFilesAndTargets(tasks)

	if len(uniqueDeletes) == 0 {
		return deletesPerFile, nil
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	perFileChan := make(chan map[string]*arrow.Chunked, concurrency)
	go func() {
		// Inner g.Wait() gates the deferred channel close so a late worker
		// can't send on a closed channel. Outer g.Wait() below collects the
		// error — no cross-goroutine shared err variable.
		defer close(perFileChan)
		for _, v := range uniqueDeletes {
			g.Go(func() error {
				deletes, err := readDeletesForPaths(gctx, fs, v, targetsByDelete[v.FilePath()])
				if err != nil {
					return fmt.Errorf("read position deletes from %s: %w", v.FilePath(), err)
				}
				if deletes == nil {
					return nil
				}
				// Safe even after gctx cancellation: the channel buffer equals
				// g's concurrency limit, so every in-flight worker can send
				// without blocking while the for-range below drains to close.
				perFileChan <- deletes

				return nil
			})
		}
		_ = g.Wait()
	}()

	for deletes := range perFileChan {
		for file, arr := range deletes {
			deletesPerFile[file] = append(deletesPerFile[file], arr)
		}
	}

	if err := g.Wait(); err != nil {
		return deletesPerFile, err
	}

	return deletesPerFile, nil
}

// lazyPositionDeleteLoader indexes positional-delete metadata for a scan, but
// waits to open each delete file until a worker reaches a task that references
// it. A delete file can apply to more than one data file, so the cache keeps
// the complete grouped result for the delete file rather than caching only one
// task's positions.
//
// The grouped Arrow chunks are owned by the loader until release. The iterator
// calls release after all workers have stopped, which keeps shared chunks alive
// while multiple tasks use them and also covers early iterator termination.
// The loader has the lifetime of exactly one scan. Each file's first load also
// locks in its result, including context errors, for every caller; a loader
// must not be reused for a retry with a different context.
type lazyPositionDeleteLoader struct {
	fs    iceio.IO
	files map[string]*lazyPositionDeleteFile

	releaseOnce sync.Once
	released    atomic.Bool
}

var errPositionDeleteLoaderReleased = errors.New("position delete loader already released")

type lazyPositionDeleteFile struct {
	dataFile iceberg.DataFile
	targets  map[string]struct{}

	once    sync.Once
	deletes map[string]*arrow.Chunked
	err     error
}

func newLazyPositionDeleteLoader(fs iceio.IO, tasks []FileScanTask) *lazyPositionDeleteLoader {
	uniqueDeletes, targetsByDelete := collectPositionDeleteFilesAndTargets(tasks)
	loader := &lazyPositionDeleteLoader{
		fs:    fs,
		files: make(map[string]*lazyPositionDeleteFile, len(uniqueDeletes)),
	}

	for path, deleteFile := range uniqueDeletes {
		loader.files[path] = &lazyPositionDeleteFile{
			dataFile: deleteFile,
			targets:  targetsByDelete[path],
		}
	}

	return loader
}

func (l *lazyPositionDeleteLoader) load(ctx context.Context, task FileScanTask) (positionDeletes, error) {
	if l.released.Load() {
		return nil, errPositionDeleteLoaderReleased
	}

	if len(task.DeleteFiles) == 0 {
		return nil, nil
	}

	targetPath := task.File.FilePath()
	deletes := make(positionDeletes, 0, len(task.DeleteFiles))
	// Most scan tasks carry one positional delete file. Avoid allocating a
	// deduplication map unless there can actually be duplicate entries.
	var seen map[string]struct{}
	if len(task.DeleteFiles) > 1 {
		seen = make(map[string]struct{}, len(task.DeleteFiles))
	}
	for _, deleteFile := range task.DeleteFiles {
		if deleteFile.ContentType() != iceberg.EntryContentPosDeletes {
			continue
		}

		path := deleteFile.FilePath()
		if seen != nil {
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
		}

		cached, ok := l.files[path]
		if !ok {
			// The loader is normally built from the same task slice supplied to
			// this method. Keep this guard so a malformed caller cannot panic a
			// scan if it changes a task after loader construction.
			continue
		}

		cached.once.Do(func() {
			cached.deletes, cached.err = readDeletesForPaths(ctx, l.fs, cached.dataFile, cached.targets)
			if cached.err != nil {
				// readDeletesForPaths currently returns nil on errors. Release defensively
				// in case a future reader returns partial Arrow ownership.
				releasePosDeletes(cached.deletes)
				cached.deletes = nil
				cached.err = fmt.Errorf("read position deletes from %s: %w",
					cached.dataFile.FilePath(), cached.err)
			}
		})
		if cached.err != nil {
			return nil, cached.err
		}

		if chunk := cached.deletes[targetPath]; chunk != nil {
			deletes = append(deletes, chunk)
		}
	}

	return deletes, nil
}

func (l *lazyPositionDeleteLoader) release() {
	if l == nil {
		return
	}

	l.releaseOnce.Do(func() {
		l.released.Store(true)
		for _, cached := range l.files {
			releasePosDeletes(cached.deletes)
			cached.deletes = nil
		}
	})
}

// perFileDVBitmaps maps each data-file path to the deletion-vector bitmap
// that applies to it. Kept separate from perFilePosDeletes so the row-filter
// pipeline can use compute.Filter on a Boolean mask built from Contains()
// directly, instead of materializing positions into a set[int64] + Take.
type perFileDVBitmaps = map[string]*dv.RoaringPositionBitmap

// lazyDeletionVectorLoader indexes deletion-vector metadata for a scan, but
// waits to read a Puffin file until a task for one of its data files is
// processed. Each Puffin group is loaded once and all of its bitmaps are kept
// in the scan-scoped cache so tasks sharing a file do not repeat the read.
type lazyDeletionVectorLoader struct {
	fs iceio.IO

	byDataFile map[string]*lazyDeletionVectorGroup
}

type lazyDeletionVectorGroup struct {
	puffinPath          string
	referencedDataFiles []string
	files               []iceberg.DataFile

	once    sync.Once
	bitmaps perFileDVBitmaps
	err     error
}

func newLazyDeletionVectorLoader(fs iceio.IO, tasks []FileScanTask) (*lazyDeletionVectorLoader, error) {
	uniqueDVs, err := collectUniqueDeletionVectors(tasks)
	if err != nil {
		return nil, err
	}

	groups := groupDeletionVectors(uniqueDVs)
	loader := &lazyDeletionVectorLoader{
		fs:         fs,
		byDataFile: make(map[string]*lazyDeletionVectorGroup, len(uniqueDVs)),
	}
	for _, group := range groups {
		for _, ref := range group.referencedDataFiles {
			loader.byDataFile[ref] = group
		}
	}

	return loader, nil
}

func (l *lazyDeletionVectorLoader) load(ctx context.Context, dataFilePath string) (*dv.RoaringPositionBitmap, error) {
	if l == nil {
		return nil, nil
	}

	group := l.byDataFile[dataFilePath]
	if group == nil {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	group.once.Do(func() {
		bitmaps, err := dv.ReadDVs(l.fs, group.files)
		if err != nil {
			group.err = fmt.Errorf("read deletion vectors from %s: %w", group.puffinPath, err)

			return
		}
		group.bitmaps = make(perFileDVBitmaps, len(bitmaps))
		for i, ref := range group.referencedDataFiles {
			group.bitmaps[ref] = bitmaps[i]
		}
	})

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if group.err != nil {
		return nil, group.err
	}

	return group.bitmaps[dataFilePath], nil
}

func collectUniqueDeletionVectors(tasks []FileScanTask) (map[string]iceberg.DataFile, error) {
	uniqueDVs := make(map[string]iceberg.DataFile)

	for _, t := range tasks {
		for _, d := range t.DeletionVectorFiles {
			_, _, ref, contentOffset, contentSize := iceinternal.BorrowedDataFilePointers(d)
			if ref == nil {
				return nil, fmt.Errorf("%w: deletion vector %s missing referenced_data_file",
					dv.ErrInvalidDeletionVector, d.FilePath())
			}
			if *ref == "" {
				return nil, fmt.Errorf("%w: deletion vector %s missing or empty referenced_data_file",
					dv.ErrInvalidDeletionVector, d.FilePath())
			}
			if contentOffset == nil || contentSize == nil {
				// Spec §Manifest Files: content_offset and content_size_in_
				// bytes are required for DV entries. Surface the missing-
				// field cause directly here — otherwise the dedup check
				// below would produce the misleading "multiple deletion
				// vectors" error when two equally-broken entries collide.
				return nil, fmt.Errorf("%w: deletion vector %s missing content_offset/content_size_in_bytes",
					dv.ErrInvalidDeletionVector, d.FilePath())
			}
			if existing, seen := uniqueDVs[*ref]; seen {
				if !sameDVBlob(existing, d) {
					return nil, fmt.Errorf(
						"multiple deletion vectors for data file %s: %s and %s",
						*ref, existing.FilePath(), d.FilePath())
				}

				continue
			}
			uniqueDVs[*ref] = d
		}
	}

	return uniqueDVs, nil
}

func groupDeletionVectors(uniqueDVs map[string]iceberg.DataFile) map[string]*lazyDeletionVectorGroup {
	groups := make(map[string]*lazyDeletionVectorGroup)
	for ref, dvFile := range uniqueDVs {
		group := groups[dvFile.FilePath()]
		if group == nil {
			group = &lazyDeletionVectorGroup{puffinPath: dvFile.FilePath()}
			groups[dvFile.FilePath()] = group
		}

		group.referencedDataFiles = append(group.referencedDataFiles, ref)
		group.files = append(group.files, dvFile)
	}

	return groups
}

// sameDVBlob reports whether two DV manifest entries point at the same puffin
// blob — identical puffin file path and content offset. Different of either
// means two distinct DVs for the same data file, the over-deletion case
// collectUniqueDeletionVectors rejects.
//
// Java's DeleteFileIndex is stricter: any second DV for the same data file is
// rejected with ValidationException("Can't index multiple DVs for %s"), even
// when both entries reference the same underlying blob. Same-blob dedup here
// is a deliberate divergence — reading the same blob twice is wasteful, not
// incorrect. ContentOffset is required to be non-nil by the spec and by the
// pre-pass in collectUniqueDeletionVectors, so the comparison below assumes
// both.
func sameDVBlob(a, b iceberg.DataFile) bool {
	if a.FilePath() != b.FilePath() {
		return false
	}

	_, _, _, aOffset, _ := iceinternal.BorrowedDataFilePointers(a)
	_, _, _, bOffset, _ := iceinternal.BorrowedDataFilePointers(b)

	return *aOffset == *bOffset
}

func filePathValueType(dt arrow.DataType) arrow.DataType {
	if dictType, ok := dt.(*arrow.DictionaryType); ok {
		return dictType.ValueType
	}

	return dt
}

func filePathValues(values arrow.Array) (arrow.TypedArray[string], error) {
	switch arr := values.(type) {
	case *array.Dictionary:
		wrapped, err := array.NewDictWrapper[string](arr)
		if err != nil {
			return nil, fmt.Errorf("%w: file_path column is not string: %w",
				iceberg.ErrInvalidSchema, err)
		}

		return wrapped, nil
	case arrow.TypedArray[string]:
		return arr, nil
	default:
		return nil, fmt.Errorf("%w: unsupported file_path column type %s in position delete file",
			iceberg.ErrInvalidSchema, values.DataType())
	}
}

type posDeleteCursor struct {
	chunks     []*array.Int64
	chunkIndex int
	offset     int
}

func validatePositionDeletePositions(positions *array.Int64) error {
	for row := range positions.Len() {
		if position := positions.Value(row); position < 0 {
			return fmt.Errorf("%w: negative pos %d in position delete file",
				iceberg.ErrInvalidSchema, position)
		}
	}

	return nil
}

func newPosDeleteCursor(posCol *arrow.Chunked) (posDeleteCursor, error) {
	chunks := posCol.Chunks()
	arrays := make([]*array.Int64, len(chunks))
	for i, chunk := range chunks {
		posArr, ok := chunk.(*array.Int64)
		if !ok {
			return posDeleteCursor{}, fmt.Errorf("%w: unsupported pos chunk array type %T in position delete file",
				iceberg.ErrInvalidSchema, chunk)
		}
		arrays[i] = posArr
	}

	return posDeleteCursor{chunks: arrays}, nil
}

func (c *posDeleteCursor) next() (int64, bool) {
	for c.chunkIndex < len(c.chunks) && c.offset == c.chunks[c.chunkIndex].Len() {
		c.chunkIndex++
		c.offset = 0
	}
	if c.chunkIndex == len(c.chunks) {
		return 0, false
	}

	pos := c.chunks[c.chunkIndex].Value(c.offset)
	c.offset++

	return pos, true
}

type posDeleteAccumulator struct {
	mem      memory.Allocator
	targets  map[string]struct{}
	builders map[string]*array.Int64Builder
}

func newPosDeleteAccumulator(ctx context.Context, targets map[string]struct{}) *posDeleteAccumulator {
	return &posDeleteAccumulator{
		mem:      compute.GetAllocator(ctx),
		targets:  targets,
		builders: make(map[string]*array.Int64Builder),
	}
}

func (a *posDeleteAccumulator) release() {
	for _, builder := range a.builders {
		builder.Release()
	}
	a.builders = nil
}

func (a *posDeleteAccumulator) finish() map[string]*arrow.Chunked {
	if a.builders == nil {
		panic("position delete accumulator is already finished or released")
	}

	results := make(map[string]*arrow.Chunked, len(a.builders))
	for path, builder := range a.builders {
		positions := builder.NewInt64Array()
		builder.Release()

		results[path] = arrow.NewChunked(arrow.PrimitiveTypes.Int64, []arrow.Array{positions})
		positions.Release()
	}
	a.builders = nil

	return results
}

func validatePosDeleteColumns(filePathType arrow.DataType, filePathNulls int,
	posType arrow.DataType, posNulls int,
) error {
	if filePathNulls > 0 {
		return fmt.Errorf("%w: null file_path in position delete file", iceberg.ErrInvalidSchema)
	}
	if filePathValueType(filePathType).ID() == arrow.STRING_VIEW {
		return fmt.Errorf("%w: unsupported file_path column type %s in position delete file",
			iceberg.ErrInvalidSchema, filePathType)
	}
	if posNulls > 0 {
		return fmt.Errorf("%w: null pos in position delete file", iceberg.ErrInvalidSchema)
	}
	if posType.ID() != arrow.INT64 {
		return fmt.Errorf("%w: unsupported pos column type %s in position delete file",
			iceberg.ErrInvalidSchema, posType)
	}

	return nil
}

func validatePosDeleteColumnLengths(filePathLen, posLen int) error {
	if filePathLen != posLen {
		return fmt.Errorf("%w: file_path and pos columns have different lengths: %d and %d",
			iceberg.ErrInvalidSchema, filePathLen, posLen)
	}

	return nil
}

func (a *posDeleteAccumulator) appendFilePathChunk(ctx context.Context, filePathChunk arrow.Array,
	posCursor *posDeleteCursor,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	paths, err := filePathValues(filePathChunk)
	if err != nil {
		return err
	}

	var dictionary arrow.Array
	var indices *array.Dictionary
	if dict, ok := filePathChunk.(*array.Dictionary); ok && dict.Dictionary().NullN() > 0 {
		dictionary = dict.Dictionary()
		indices = dict
	}

	for i := range filePathChunk.Len() {
		if i&(positionalDeleteCancellationCheckInterval-1) == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}

		pos, ok := posCursor.next()
		if !ok {
			return fmt.Errorf("%w: position delete columns ended before file_path column",
				iceberg.ErrInvalidSchema)
		}
		if pos < 0 {
			return fmt.Errorf("%w: negative pos %d in position delete file",
				iceberg.ErrInvalidSchema, pos)
		}
		if dictionary != nil && dictionary.IsNull(indices.GetValueIndex(i)) {
			return fmt.Errorf("%w: null file_path dictionary value in position delete file",
				iceberg.ErrInvalidSchema)
		}

		path := paths.Value(i)
		if len(a.targets) > 0 {
			if _, ok := a.targets[path]; !ok {
				continue
			}
		}

		builder, ok := a.builders[path]
		if !ok {
			path = strings.Clone(path)
			builder = array.NewInt64Builder(a.mem)
			a.builders[path] = builder
		}
		builder.Append(pos)
	}

	return nil
}

func (a *posDeleteAccumulator) appendChunked(ctx context.Context, filePathCol, posCol *arrow.Chunked) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validatePosDeleteColumns(filePathCol.DataType(), filePathCol.NullN(),
		posCol.DataType(), posCol.NullN()); err != nil {
		return err
	}
	if err := validatePosDeleteColumnLengths(filePathCol.Len(), posCol.Len()); err != nil {
		return err
	}

	posCursor, err := newPosDeleteCursor(posCol)
	if err != nil {
		return err
	}

	for _, filePathChunk := range filePathCol.Chunks() {
		if err := a.appendFilePathChunk(ctx, filePathChunk, &posCursor); err != nil {
			return err
		}
	}

	return ctx.Err()
}

// appendRecord requires columns projected in file_path, pos order.
func (a *posDeleteAccumulator) appendRecord(ctx context.Context, record arrow.RecordBatch) error {
	if record.NumCols() != 2 {
		return fmt.Errorf("%w: projected position delete record has %d columns, expected 2",
			iceberg.ErrInvalidSchema, record.NumCols())
	}

	filePathCol := record.Column(0)
	posCol := record.Column(1)
	if err := validatePosDeleteColumns(filePathCol.DataType(), filePathCol.NullN(),
		posCol.DataType(), posCol.NullN()); err != nil {
		return err
	}
	if err := validatePosDeleteColumnLengths(filePathCol.Len(), posCol.Len()); err != nil {
		return err
	}

	posArr, ok := posCol.(*array.Int64)
	if !ok {
		return fmt.Errorf("%w: unsupported pos record array type %T in position delete file",
			iceberg.ErrInvalidSchema, posCol)
	}
	posCursor := posDeleteCursor{chunks: []*array.Int64{posArr}}
	if err := a.appendFilePathChunk(ctx, filePathCol, &posCursor); err != nil {
		return err
	}

	return ctx.Err()
}

func groupPosDeletesByFilePath(ctx context.Context, filePathCol, posCol *arrow.Chunked) (results map[string]*arrow.Chunked, err error) {
	acc := newPosDeleteAccumulator(ctx, nil)
	defer func() {
		if err != nil {
			acc.release()
		}
	}()

	if err := acc.appendChunked(ctx, filePathCol, posCol); err != nil {
		return nil, err
	}

	return acc.finish(), nil
}

const maxPositionalDeletePreallocation = 64 * 1024

func collectPosDeletePositions(positionalDeletes positionDeletes) (set[int64], error) {
	totalPositions := 0
	for _, chunk := range positionalDeletes {
		if chunk == nil {
			continue
		}
		// The set only reserves a bounded hint. Stop summing once that
		// hint is reached so a large collection of chunks cannot overflow
		// int before the cap is applied.
		if chunk.Len() >= maxPositionalDeletePreallocation-totalPositions {
			totalPositions = maxPositionalDeletePreallocation

			break
		}
		totalPositions += chunk.Len()
	}

	deletes := make(set[int64], min(totalPositions, maxPositionalDeletePreallocation))
	for _, chunk := range positionalDeletes {
		if chunk == nil {
			return nil, fmt.Errorf("%w: nil pos column chunk in position delete file",
				iceberg.ErrInvalidSchema)
		}
		if chunk.DataType().ID() != arrow.INT64 {
			return nil, fmt.Errorf("%w: unsupported pos column type %s in position delete file",
				iceberg.ErrInvalidSchema, chunk.DataType())
		}
		for _, arr := range chunk.Chunks() {
			posArr, ok := arr.(*array.Int64)
			if !ok {
				return nil, fmt.Errorf("%w: unsupported pos chunk array type %T in position delete file",
					iceberg.ErrInvalidSchema, arr)
			}
			if posArr.NullN() > 0 {
				return nil, fmt.Errorf("%w: null pos in position delete file",
					iceberg.ErrInvalidSchema)
			}
			for _, v := range posArr.Int64Values() {
				if v < 0 {
					return nil, fmt.Errorf("%w: negative pos %d in position delete file",
						iceberg.ErrInvalidSchema, v)
				}
				deletes[v] = struct{}{}
			}
		}
	}

	return deletes, nil
}

func releasePosDeletes(deletes map[string]*arrow.Chunked) {
	for _, chunk := range deletes {
		if chunk != nil {
			chunk.Release()
		}
	}
}

func readDeletes(ctx context.Context, fs iceio.IO, dataFile iceberg.DataFile) (map[string]*arrow.Chunked, error) {
	return readDeletesForPaths(ctx, fs, dataFile, nil)
}

func readDeletesForPaths(ctx context.Context, fs iceio.IO, dataFile iceberg.DataFile,
	targets map[string]struct{},
) (_ map[string]*arrow.Chunked, err error) {
	src, err := tblutils.GetFile(ctx, fs, dataFile, true)
	if err != nil {
		return nil, err
	}

	rdr, err := src.GetReader(ctx)
	if err != nil {
		return nil, err
	}
	defer iceinternal.CheckedClose(rdr, &err)

	schema, err := rdr.Schema()
	if err != nil {
		return nil, err
	}

	columns, err := positionDeleteProjectionIndices(schema, rdr)
	if err != nil {
		return nil, err
	}

	tester, err := newPositionDeleteRowGroupTester(schema, targets)
	if err != nil {
		return nil, err
	}

	records, err := rdr.GetRecords(ctx, columns, tester)
	if err != nil {
		return nil, err
	}
	// Do not unify dictionaries here: appendFilePathChunk decodes each batch to
	// string values, so independent dictionaries across batches are safe.
	defer records.Release()

	acc := newPosDeleteAccumulator(ctx, targets)
	defer func() {
		// Returning an error assigns the named return value before deferred
		// functions run, which releases builders on every error path.
		if err != nil {
			acc.release()
		}
	}()

	for records.Next() {
		if err := acc.appendRecord(ctx, records.RecordBatch()); err != nil {
			return nil, err
		}
	}
	if err := records.Err(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return acc.finish(), nil
}

func newPositionDeleteRowGroupTester(schema *arrow.Schema, targets map[string]struct{}) (*tblutils.ParquetRowGroupTester, error) {
	if len(targets) == 0 || len(targets) > inPredicateLimit {
		return nil, nil
	}
	pruningEnabled, err := positionDeletePruningEnabled(schema)
	if err != nil {
		return nil, err
	}
	if !pruningEnabled {
		return nil, nil
	}

	paths := make([]string, 0, len(targets))
	for path := range targets {
		paths = append(paths, path)
	}

	var filter iceberg.BooleanExpression
	if len(paths) == 1 {
		// A single target is the common case. EqualTo avoids building the
		// set literal used by IsIn and gives the stats/bloom planners the
		// simpler predicate directly.
		filter = iceberg.EqualTo(iceberg.Reference("file_path"), paths[0])
	} else {
		slices.Sort(paths)
		filter = iceberg.IsIn(iceberg.Reference("file_path"), paths...)
	}
	filter, err = iceberg.BindExpr(iceberg.PositionalDeleteSchema, filter, true)
	if err != nil {
		return nil, err
	}

	statsFn, err := newParquetRowGroupStatsEvaluator(iceberg.PositionalDeleteSchema, filter, false)
	if err != nil {
		return nil, err
	}
	bloomPreds, err := newBloomFilterPredicates(filter)
	if err != nil {
		return nil, err
	}

	return &tblutils.ParquetRowGroupTester{
		StatsFn:    statsFn,
		BloomPreds: bloomPreds,
	}, nil
}

func positionDeletePruningEnabled(schema *arrow.Schema) (bool, error) {
	// Row-group stats and Bloom predicates are keyed by Parquet physical field
	// IDs, while projection resolves these columns by their spec-defined names.
	// pqarrow carries the Parquet IDs into Arrow metadata, so only enable
	// pushdown when those two views agree for the reserved delete columns.
	physicalIDs := make(map[int]int)
	var collectIDs func([]arrow.Field)
	collectIDs = func(fields []arrow.Field) {
		for _, field := range fields {
			if id := getFieldID(field); id != nil {
				physicalIDs[*id]++
			}
			if nested, ok := field.Type.(arrow.NestedType); ok {
				collectIDs(nested.Fields())
			}
		}
	}
	collectIDs(schema.Fields())
	if len(physicalIDs) == 0 {
		// External position-delete files are allowed to omit Iceberg field IDs.
		// The name-based projection and row-level target filter remain safe, but
		// stats and Bloom pruning cannot be trusted without the IDs.
		return false, nil
	}

	deleteFields := iceberg.PositionalDeleteSchema.Fields()
	for _, field := range deleteFields {
		if physicalIDs[field.ID] > 1 {
			return false, fmt.Errorf("%w: position delete field ID %d is not unique",
				iceberg.ErrInvalidSchema, field.ID)
		}
	}

	pruningEnabled := true
	for _, want := range deleteFields {
		indices := schema.FieldIndices(want.Name)
		if len(indices) != 1 {
			return false, fmt.Errorf("%w: position delete file must contain exactly one %q column, found %d",
				iceberg.ErrInvalidSchema, want.Name, len(indices))
		}

		fieldID := getFieldID(schema.Field(indices[0]))
		if fieldID == nil {
			if physicalIDs[want.ID] != 0 {
				return false, fmt.Errorf("%w: position delete field ID %d is assigned to another column instead of %q",
					iceberg.ErrInvalidSchema, want.ID, want.Name)
			}
			// Missing IDs only disable pruning. Keep checking the remaining
			// columns so this fallback cannot hide an invalid ID mapping.
			pruningEnabled = false

			continue
		}
		if *fieldID != want.ID {
			if physicalIDs[want.ID] != 0 {
				return false, fmt.Errorf("%w: position delete column %q has field ID %d, want %d; field ID %d is assigned to another column",
					iceberg.ErrInvalidSchema, want.Name, *fieldID, want.ID, want.ID)
			}
			// A non-canonical ID is safe for name-based reading, but not for
			// stats or Bloom pruning. Keep the pre-pruning read compatible with
			// external writers that renumber fields.
			pruningEnabled = false
		}
	}

	return pruningEnabled, nil
}

func positionDeleteProjectionIndices(schema *arrow.Schema, reader tblutils.FileReader) ([]int, error) {
	filePathIndex, posIndex, err := positionDeleteColumnIndices(schema)
	if err != nil {
		return nil, err
	}
	fileMetadata, ok := reader.Metadata().(*metadata.FileMetaData)
	if !ok {
		return nil, fmt.Errorf("%w: position delete reader does not expose Parquet metadata", iceberg.ErrInvalidSchema)
	}

	// Parquet projection uses leaf indices. A nested row column can shift
	// those indices relative to the top-level Arrow fields.
	columns := make([]int, 2)
	for i, fieldIndex := range []int{filePathIndex, posIndex} {
		name := schema.Field(fieldIndex).Name
		columns[i] = fileMetadata.Schema.ColumnIndexByName(name)
		if columns[i] < 0 {
			return nil, fmt.Errorf("%w: position delete column %q must be a primitive column", iceberg.ErrInvalidSchema, name)
		}
	}

	return columns, nil
}

func positionDeleteColumnIndices(schema *arrow.Schema) (int, int, error) {
	// Position-delete columns are resolved by their spec-defined names because
	// Arrow schemas read from external files do not always retain Iceberg IDs.
	requiredColumn := func(name string) (int, error) {
		indices := schema.FieldIndices(name)
		if len(indices) != 1 {
			return 0, fmt.Errorf("%w: position delete file must contain exactly one %q column, found %d",
				iceberg.ErrInvalidSchema, name, len(indices))
		}

		return indices[0], nil
	}

	filePathIndex, err := requiredColumn("file_path")
	if err != nil {
		return 0, 0, err
	}
	posIndex, err := requiredColumn("pos")
	if err != nil {
		return 0, 0, err
	}

	return filePathIndex, posIndex, nil
}

type set[T comparable] map[T]struct{}

// combinePositionalDeletes builds the surviving-row index list for a single record
// batch. The deletes set holds file-relative (global) positions; cursor maps each
// batch row to its original file position (jumping across pruned row groups), and
// the appended indices are batch-local (the loop counter): they index into the
// batch handed to compute.Take, not into the whole file. Without batch-local
// indices, the second and later batches of a file would pass indices >= the batch
// length and compute.Take would fail with "index error: N out of bounds".
//
// A nil result means that no row in the batch was deleted. The index builder is
// allocated lazily after the first deleted row so clean batches can pass through
// without reconstructing their columns.
func combinePositionalDeletes(mem memory.Allocator, deletes set[int64], cursor *rowPositionCursor, nrows int64) arrow.Array {
	var bldr *array.Int64Builder

	for i := range nrows {
		if _, deleted := deletes[cursor.next()]; deleted {
			if bldr == nil {
				bldr = array.NewInt64Builder(mem)
				bldr.Reserve(int(i))
				for j := range i {
					bldr.Append(j)
				}
			}

			continue
		}

		if bldr != nil {
			bldr.Append(i)
		}
	}

	if bldr == nil {
		return nil
	}
	defer bldr.Release()

	return bldr.NewArray()
}

type recProcessFn func(arrow.RecordBatch) (arrow.RecordBatch, error)

func processPositionalDeletes(ctx context.Context, deletes set[int64], cursor *rowPositionCursor) recProcessFn {
	mem := compute.GetAllocator(ctx)

	return func(r arrow.RecordBatch) (arrow.RecordBatch, error) {
		defer r.Release()

		indices := combinePositionalDeletes(mem, deletes, cursor, r.NumRows())
		if indices == nil {
			r.Retain()

			return r, nil
		}
		defer indices.Release()

		out, err := compute.Take(ctx, *compute.DefaultTakeOptions(),
			compute.NewDatumWithoutOwning(r), compute.NewDatumWithoutOwning(indices))
		if err != nil {
			return nil, err
		}

		return out.(*compute.RecordDatum).Value, nil
	}
}

func deletionVectorKeepsRow(keepBits []byte, rowCount, pos int64) bool {
	return pos >= rowCount || keepBits[pos>>3]&(1<<(uint(pos)&7)) != 0
}

// filterByDeletionVector returns a pipeline step that drops rows present in
// the bitmap by precomputing a bit-packed keep-mask covering the whole file
// once and slicing the relevant range per batch into compute.FilterRecordBatch.
//
// The mask layout matches Arrow's Boolean buffer convention (LSB-first per
// byte, little-endian word order), so dv.KeepMaskBytes -> memory.NewBufferBytes
// -> array.NewBoolean is a zero-copy wrap. The keepBits slice is Go-allocated
// (GC-friendly) and shared across every per-batch Boolean array; each
// array.NewBoolean / array.NewSlice pair is released after the batch.
//
// rowCount bounds the mask to the data file's row count. cursor maps each batch
// row to its original file position: when no row group was pruned the positions
// are contiguous and the mask is sliced zero-copy; when pruning skipped groups
// the emitted rows are non-contiguous, so the keep bits are gathered per row at
// the cursor's positions.
func filterByDeletionVector(ctx context.Context, bitmap *dv.RoaringPositionBitmap, rowCount int64, cursor *rowPositionCursor) recProcessFn {
	nextIdx := int64(0)
	keepBits := bitmap.KeepMaskBytes(rowCount)
	buf := memory.NewBufferBytes(keepBits)
	mem := compute.GetAllocator(ctx)

	return func(r arrow.RecordBatch) (arrow.RecordBatch, error) {
		defer r.Release()

		nrows := r.NumRows()

		if !cursor.src.pruned() {
			currentIdx := nextIdx
			nextIdx += nrows

			if nextIdx <= rowCount {
				// Wrap (and slice) the shared keep-mask buffer for this batch.
				// array.NewSlice on a Boolean array tracks the bit-level offset,
				// so we don't need byte-aligned slicing — currentIdx can land
				// anywhere within a byte.
				full := array.NewBoolean(int(rowCount), buf, nil, 0)
				defer full.Release()
				sliced := array.NewSlice(full, currentIdx, nextIdx).(*array.Boolean)
				defer sliced.Release()

				return compute.FilterRecordBatch(ctx, r, sliced, compute.DefaultFilterOptions())
			}
			if currentIdx >= rowCount {
				r.Retain()

				return r, nil
			}

			// A stale manifest count can be smaller than the rows emitted by
			// the file. Preserve rows beyond the mask rather than slicing past
			// its bounds and panicking.
			bldr := array.NewBooleanBuilder(mem)
			defer bldr.Release()
			bldr.Reserve(int(nrows))
			for pos := currentIdx; pos < nextIdx; pos++ {
				bldr.Append(deletionVectorKeepsRow(keepBits, rowCount, pos))
			}
			mask := bldr.NewBooleanArray()
			defer mask.Release()

			return compute.FilterRecordBatch(ctx, r, mask, compute.DefaultFilterOptions())
		}

		bldr := array.NewBooleanBuilder(mem)
		defer bldr.Release()
		bldr.Reserve(int(nrows))
		for range nrows {
			pos := cursor.next()
			// keepBits is sized for rowCount; a pos past it means row-group
			// metadata and the manifest's File.Count() disagree. Keep the row
			// rather than indexing out of bounds.
			bldr.Append(deletionVectorKeepsRow(keepBits, rowCount, pos))
		}
		mask := bldr.NewBooleanArray()
		defer mask.Release()

		return compute.FilterRecordBatch(ctx, r, mask, compute.DefaultFilterOptions())
	}
}

// enrichRecordsWithPosDeleteFields enriches a RecordBatch with the columns declared in the PositionalDeleteArrowSchema
// so that during the pipeline filtering stages that sheds filtered out records, we still have a way to
// preserve the original position of those records.
func enrichRecordsWithPosDeleteFields(ctx context.Context, filePath iceberg.DataFile, cursor *rowPositionCursor) recProcessFn {
	mem := compute.GetAllocator(ctx)

	return func(inData arrow.RecordBatch) (outData arrow.RecordBatch, err error) {
		defer inData.Release()

		schema := inData.Schema()
		fieldIdx := schema.NumFields()
		schema, err = schema.AddField(fieldIdx, PositionalDeleteArrowSchema.Field(0))
		if err != nil {
			return nil, err
		}
		schema, err = schema.AddField(fieldIdx+1, PositionalDeleteArrowSchema.Field(1))
		if err != nil {
			return nil, err
		}

		rb := array.NewRecordBuilder(mem, PositionalDeleteArrowSchema)
		defer rb.Release()

		filePathBldr, posBldr := rb.Field(0).(*array.StringBuilder), rb.Field(1).(*array.Int64Builder)

		for range inData.NumRows() {
			filePathBldr.Append(filePath.FilePath())
			posBldr.Append(cursor.next())
		}

		newCols := rb.NewRecordBatch()
		defer newCols.Release()

		columns := append(inData.Columns(), newCols.Column(0), newCols.Column(1))
		outData = array.NewRecordBatch(schema, columns, inData.NumRows())

		return outData, err
	}
}

func filterRecords(ctx context.Context, recordFilter expr.Expression) recProcessFn {
	return func(rec arrow.RecordBatch) (arrow.RecordBatch, error) {
		defer rec.Release()

		input := compute.NewDatumWithoutOwning(rec)
		mask, err := exprs.ExecuteScalarExpression(ctx, rec.Schema(), recordFilter, input)
		if err != nil {
			return nil, err
		}
		defer mask.Release()

		result, err := compute.Filter(ctx, input, mask, *compute.DefaultFilterOptions())
		if err != nil {
			return nil, err
		}

		return result.(*compute.RecordDatum).Value, nil
	}
}

type arrowScan struct {
	fs              iceio.IO
	metadata        Metadata
	scanSchema      *iceberg.Schema
	projectedSchema *iceberg.Schema
	boundRowFilter  iceberg.BooleanExpression
	// rowGroupFilter is used only for Parquet statistics and bloom-filter
	// pruning. It lets callers keep boundRowFilter as AlwaysTrue while they
	// must evaluate the real row filter after position-dependent enrichment.
	rowGroupFilter    iceberg.BooleanExpression
	filterSchema      *iceberg.Schema
	caseSensitive     bool
	rowLimit          int64
	options           iceberg.Properties
	filterPlanCache   compiledFileFilterPlanCache
	fileReadPlanCache preparedFileReadPlanCache
	cacheFileReadPlan bool

	useLargeTypes bool
	concurrency   int

	// arrowBatchSize, when positive, overrides the table's
	// read.parquet.batch-size property for this scan's reads.
	arrowBatchSize int
}

// preparedFileRead contains the physical schema projection shared by all
// tasks reading the same data file during one scan. The actual FileReader is
// intentionally not shared: split tasks may run concurrently and each reader
// owns its column readers and input handle.
type preparedFileRead struct {
	schema     *iceberg.Schema
	colIndices []int
}

type preparedFileReadEntry struct {
	once sync.Once
	plan *preparedFileRead
	err  error
}

type preparedFileReadPlanCache struct {
	mu      sync.Mutex
	entries map[string]*preparedFileReadEntry
}

func (c *preparedFileReadPlanCache) entry(path string) *preparedFileReadEntry {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.entries == nil {
		c.entries = make(map[string]*preparedFileReadEntry)
	}
	if entry, ok := c.entries[path]; ok {
		return entry
	}

	entry := &preparedFileReadEntry{}
	c.entries[path] = entry

	return entry
}

// arrowScanInvariants holds metadata-derived values that cannot change during
// a scan. The snapshots returned by Metadata are owned by the scan and are
// safe to share read-only across file workers.
type arrowScanInvariants struct {
	tableSchema     *iceberg.Schema
	tableProperties iceberg.Properties
	projectedIDs    set[int]
	nameMapping     iceberg.NameMapping
}

// collectLeafIDs recursively collects leaf field IDs from a type
func collectLeafIDs(typ iceberg.Type, fieldID int, idset set[int]) {
	switch t := typ.(type) {
	case *iceberg.MapType:
		// For maps, collect leaf IDs from both key and value
		collectLeafIDs(t.KeyType, t.KeyID, idset)
		collectLeafIDs(t.ValueType, t.ValueID, idset)
	case *iceberg.ListType:
		// For lists, collect leaf IDs from the element
		collectLeafIDs(t.Element, t.ElementID, idset)
	case *iceberg.StructType:
		// For structs, collect leaf IDs from all fields
		for _, field := range t.FieldList {
			collectLeafIDs(field.Type, field.ID, idset)
		}
	default:
		// Primitive type - this is a leaf
		idset[fieldID] = struct{}{}
	}
}

func addFilterFieldIDs(idset set[int], rowFilter iceberg.BooleanExpression) error {
	if rowFilter == nil {
		return nil
	}

	extracted, err := iceberg.ExtractFieldIDs(rowFilter)
	if err != nil {
		return err
	}

	for _, id := range extracted {
		idset[id] = struct{}{}
	}

	return nil
}

func (as *arrowScan) projectedFieldIDs(rowFilter iceberg.BooleanExpression, equalityDeletes []*equalityDeleteSet) (set[int], error) {
	idset := set[int]{}
	// Collect leaf field IDs for column pruning.
	// For nested types (map, list, struct), we recursively descend to find
	// the actual leaf primitive fields, not the intermediate container nodes.
	for _, field := range as.projectedSchema.Fields() {
		collectLeafIDs(field.Type, field.ID, idset)
	}

	if err := addFilterFieldIDs(idset, rowFilter); err != nil {
		return nil, err
	}
	for _, deletes := range equalityDeletes {
		for _, id := range deletes.fieldIDs {
			idset[id] = struct{}{}
		}
	}

	return idset, nil
}

func (as *arrowScan) scanInvariants(tableProperties iceberg.Properties) (*arrowScanInvariants, error) {
	// Filter columns are added per task below. A local task may have a
	// partition-elided residual that no longer references the original filter
	// fields.
	projectedIDs, err := as.projectedFieldIDs(nil, nil)
	if err != nil {
		return nil, err
	}

	return &arrowScanInvariants{
		tableSchema:     as.metadata.CurrentSchema(),
		tableProperties: tableProperties,
		projectedIDs:    projectedIDs,
		nameMapping:     as.metadata.NameMapping(),
	}, nil
}

func (as *arrowScan) addTaskProjectedFieldIDs(invariants *arrowScanInvariants, tasks []FileScanTask) error {
	// Tasks without a residual still use the scan's original filter. Add those
	// fields once, then add only the actual residual fields for other tasks.
	for _, task := range tasks {
		if task.Residual == nil {
			if err := addFilterFieldIDs(invariants.projectedIDs, as.boundRowFilter); err != nil {
				return err
			}

			break
		}
	}

	for _, task := range tasks {
		if task.Residual == nil {
			continue
		}

		rowFilter, err := as.rowFilterForTask(task)
		if err != nil {
			return err
		}

		if err := addFilterFieldIDs(invariants.projectedIDs, rowFilter); err != nil {
			return err
		}
	}

	return nil
}

const maxInFlightTasksPerWorker = 1

type taskCredits chan struct{}

func newTaskCredits() taskCredits {
	return make(taskCredits, maxInFlightTasksPerWorker)
}

func (c taskCredits) acquire(ctx context.Context) error {
	select {
	case c <- struct{}{}:
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

func (c taskCredits) release() {
	if c != nil {
		<-c
	}
}

type enumeratedRecord struct {
	Record  tblutils.Enumerated[arrow.RecordBatch]
	Task    tblutils.Enumerated[FileScanTask]
	Err     error
	credits taskCredits
}

type recordSink struct {
	out     chan<- enumeratedRecord
	credits taskCredits
	held    bool
}

func newRecordSink(out chan<- enumeratedRecord) *recordSink {
	return &recordSink{out: out, credits: newTaskCredits()}
}

func (s *recordSink) reserve(ctx context.Context) error {
	if err := s.credits.acquire(ctx); err != nil {
		return err
	}
	s.held = true

	return nil
}

func (s *recordSink) send(rec enumeratedRecord) {
	if rec.Record.Last {
		rec.credits = s.handOff()
	}
	s.out <- rec
}

func (s *recordSink) fail(task tblutils.Enumerated[FileScanTask], err error) {
	s.out <- enumeratedRecord{Task: task, Err: err, credits: s.handOff()}
}

func (s *recordSink) handOff() taskCredits {
	if !s.held {
		return nil
	}
	s.held = false

	return s.credits
}

func (as *arrowScan) prepareToRead(ctx context.Context, file iceberg.DataFile, invariants *arrowScanInvariants) (iceSchema *iceberg.Schema, colIndices []int, rdr tblutils.FileReader, err error) {
	src, err := tblutils.GetFile(ctx, as.fs, file, false)
	if err != nil {
		return nil, nil, nil, err
	}

	rdr, err = src.GetReader(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	keepReader := false
	defer func(reader tblutils.FileReader) {
		if !keepReader {
			_ = reader.Close()
		}
	}(rdr)

	if !as.cacheFileReadPlan {
		var fileSchema *arrow.Schema
		fileSchema, colIndices, err = rdr.PrunedSchema(
			invariants.projectedIDs, invariants.nameMapping)
		if err != nil {
			return nil, nil, nil, err
		}

		iceSchema, err = ArrowSchemaToIcebergWithOptions(fileSchema, ArrowToIcebergOptions{
			NameMapping:     invariants.nameMapping,
			TableSchema:     invariants.tableSchema,
			TableProperties: invariants.tableProperties,
		})
		if err != nil {
			return nil, nil, nil, err
		}

		keepReader = true

		return iceSchema, colIndices, rdr, nil
	}

	entry := as.fileReadPlanCache.entry(file.FilePath())
	entry.once.Do(func() {
		var fileSchema *arrow.Schema
		var colIndices []int
		fileSchema, colIndices, entry.err = rdr.PrunedSchema(
			invariants.projectedIDs, invariants.nameMapping)
		if entry.err != nil {
			return
		}

		var iceSchema *iceberg.Schema
		iceSchema, entry.err = ArrowSchemaToIcebergWithOptions(fileSchema, ArrowToIcebergOptions{
			NameMapping:     invariants.nameMapping,
			TableSchema:     invariants.tableSchema,
			TableProperties: invariants.tableProperties,
		})
		if entry.err == nil {
			entry.plan = &preparedFileRead{
				schema:     iceSchema,
				colIndices: colIndices,
			}
		}
	})
	if entry.err != nil {
		return nil, nil, nil, entry.err
	}

	keepReader = true

	return entry.plan.schema, entry.plan.colIndices, rdr, nil
}

func (as *arrowScan) getRecordFilter(ctx context.Context, fileSchema *iceberg.Schema, rowFilter iceberg.BooleanExpression) (recProcessFn, bool, error) {
	if rowFilter == nil || rowFilter.Equals(iceberg.AlwaysTrue{}) {
		return nil, false, nil
	}

	plan, err := compileFileFilterPlan(fileSchema, rowFilter, as.caseSensitive, true, false)
	if err != nil {
		return nil, false, err
	}

	return as.recordProcessorWithExtracts(ctx, plan), plan.dropFile, nil
}

// recordProcessorWithExtracts returns the plan's record filter, wrapped with the
// per-batch variant extract residual when the plan carries extract terms.
func (as *arrowScan) recordProcessorWithExtracts(ctx context.Context, plan *compiledFileFilterPlan) recProcessFn {
	base := plan.recordProcessor(ctx)
	if base == nil {
		// No record filter (nil plan or filter folded to AlwaysTrue): keep every row, no residual.
		return nil
	}
	if len(plan.extracts) == 0 {
		return base
	}

	return as.extractResidualFilter(ctx, plan.extracts, base)
}

func (as *arrowScan) getRecordFilterFromPlan(ctx context.Context, plan *compiledFileFilterPlan) (recProcessFn, bool, error) {
	if plan == nil {
		return nil, false, nil
	}

	return as.recordProcessorWithExtracts(ctx, plan), plan.dropFile, nil
}

type filterBindingState struct {
	hasBound   bool
	hasUnbound bool
}

type filterBindingVisitor struct{}

func (filterBindingVisitor) VisitTrue() filterBindingState  { return filterBindingState{} }
func (filterBindingVisitor) VisitFalse() filterBindingState { return filterBindingState{} }
func (filterBindingVisitor) VisitNot(child filterBindingState) filterBindingState {
	return child
}

func (filterBindingVisitor) VisitAnd(left, right filterBindingState) filterBindingState {
	return filterBindingState{
		hasBound:   left.hasBound || right.hasBound,
		hasUnbound: left.hasUnbound || right.hasUnbound,
	}
}

func (filterBindingVisitor) VisitOr(left, right filterBindingState) filterBindingState {
	return filterBindingVisitor{}.VisitAnd(left, right)
}

func (filterBindingVisitor) VisitUnbound(iceberg.UnboundPredicate) filterBindingState {
	return filterBindingState{hasUnbound: true}
}

func (filterBindingVisitor) VisitBound(iceberg.BoundPredicate) filterBindingState {
	return filterBindingState{hasBound: true}
}

type boundFilterSchemaVisitor struct {
	schema *iceberg.Schema
}

func (boundFilterSchemaVisitor) VisitTrue() error  { return nil }
func (boundFilterSchemaVisitor) VisitFalse() error { return nil }
func (boundFilterSchemaVisitor) VisitNot(child error) error {
	return child
}

func firstFilterValidationError(left, right error) error {
	if left != nil {
		return left
	}

	return right
}

func (boundFilterSchemaVisitor) VisitAnd(left, right error) error {
	return firstFilterValidationError(left, right)
}

func (boundFilterSchemaVisitor) VisitOr(left, right error) error {
	return firstFilterValidationError(left, right)
}

func (boundFilterSchemaVisitor) VisitUnbound(iceberg.UnboundPredicate) error {
	return fmt.Errorf("%w: expected a fully bound scan task residual", iceberg.ErrInvalidArgument)
}

func (v boundFilterSchemaVisitor) VisitBound(pred iceberg.BoundPredicate) error {
	boundRef := pred.Ref()
	boundField := boundRef.Field()
	effectiveField, ok := v.schema.FindFieldByID(boundField.ID)
	if !ok {
		return fmt.Errorf("%w: bound residual references field ID %d, which is not present in the scan schema",
			iceberg.ErrInvalidArgument, boundField.ID)
	}
	if !effectiveField.Type.Equals(boundField.Type) {
		return fmt.Errorf("%w: bound residual field ID %d has type %s, but the scan schema has type %s",
			iceberg.ErrInvalidArgument, boundField.ID, boundField.Type, effectiveField.Type)
	}

	columnName, ok := v.schema.FindColumnName(boundField.ID)
	if !ok {
		return fmt.Errorf("%w: scan schema has no name for field ID %d",
			iceberg.ErrInvalidArgument, boundField.ID)
	}
	rebound, err := iceberg.Reference(columnName).Bind(v.schema, true)
	if err != nil {
		return fmt.Errorf("%w: cannot validate field ID %d against the scan schema: %v",
			iceberg.ErrInvalidArgument, boundField.ID, err)
	}
	if !slices.Equal(boundRef.PosPath(), rebound.Ref().PosPath()) {
		return fmt.Errorf("%w: bound residual field ID %d has accessor path %v, but the scan schema uses %v",
			iceberg.ErrInvalidArgument, boundField.ID, boundRef.PosPath(), rebound.Ref().PosPath())
	}

	return nil
}

func validateBoundFilter(schema *iceberg.Schema, filter iceberg.BooleanExpression) error {
	if schema == nil {
		return fmt.Errorf("%w: cannot validate a bound scan task residual against a nil schema",
			iceberg.ErrInvalidArgument)
	}

	validationErr, visitErr := iceberg.VisitExpr(filter, boundFilterSchemaVisitor{schema: schema})
	if visitErr != nil {
		return visitErr
	}

	return validationErr
}

func bindTaskFilter(schema *iceberg.Schema, filter iceberg.BooleanExpression, caseSensitive bool) (iceberg.BooleanExpression, error) {
	if filter == nil {
		return nil, nil
	}

	state, err := iceberg.VisitExpr(filter, filterBindingVisitor{})
	if err != nil {
		return nil, err
	}
	if state.hasBound && state.hasUnbound {
		return nil, fmt.Errorf("%w: scan task residual mixes bound and unbound predicates", iceberg.ErrInvalidArgument)
	}
	if !state.hasUnbound {
		if state.hasBound {
			if err := validateBoundFilter(schema, filter); err != nil {
				return nil, err
			}
		}

		return filter, nil
	}

	return iceberg.BindExpr(schema, filter, caseSensitive)
}

func (as *arrowScan) rowFilterForTask(task FileScanTask) (iceberg.BooleanExpression, error) {
	if task.Residual == nil {
		return as.boundRowFilter, nil
	}

	filterSchema := as.scanSchema
	if as.filterSchema != nil {
		filterSchema = as.filterSchema
	}

	return bindTaskFilter(filterSchema, task.Residual, as.caseSensitive)
}

// fieldIndexByID returns the index of the field carrying fieldID in its Arrow
// metadata, or -1. Resolving by reserved id (not name) matches how
// SchemaWithRowLineageColumns and projection identify lineage columns.
func fieldIndexByID(schema *arrow.Schema, fieldID int) int {
	for i, f := range schema.Fields() {
		if v, ok := f.Metadata.GetValue(ArrowParquetFieldIDKey); ok {
			if id, err := strconv.Atoi(v); err == nil && id == fieldID {
				return i
			}
		}
	}

	return -1
}

// rowPositionSource holds the surviving row groups for a single file read.
// processRecords populates spans (via the row-group tester) before any record is
// read, then every position-keyed pipeline step takes its own cursor over the
// shared, immutable spans. One source, many cursors: each step advances
// independently but resolves the same original file positions.
//
// With no spans recorded (non-Parquet files, or a read with no pruning pass) the
// cursors fall back to a contiguous counter from zero.
type rowPositionSource struct {
	spans []tblutils.RowGroupSpan
}

func (s *rowPositionSource) cursor() *rowPositionCursor {
	return &rowPositionCursor{src: s}
}

// pruned reports whether any row group was skipped, i.e. whether emitted rows
// are non-contiguous. Read only after spans are populated.
func (s *rowPositionSource) pruned() bool {
	return len(s.spans) > 0
}

// rowPositionCursor maps each emitted row back to its original position in the
// data file. Surviving row groups are visited in file order, so positions jump
// across the gaps left by pruned groups, keeping _row_id = first_row_id +
// original position correct even while row-group pruning is active.
type rowPositionCursor struct {
	src      *rowPositionSource
	spanIdx  int
	consumed int64
}

// next returns the original file position of the next row. It is meant to be
// called once per physical row read, in order. spanIdx is bounded to the last
// span, so reading past the total row count degrades to positions beyond the
// final span rather than panicking here. Callers that index a buffer sized to
// the file's row count (e.g. filterByDeletionVector's keep-mask) must still
// guard against the returned position exceeding that count.
func (c *rowPositionCursor) next() int64 {
	spans := c.src.spans
	if len(spans) == 0 {
		pos := c.consumed
		c.consumed++

		return pos
	}

	for c.spanIdx < len(spans)-1 && c.consumed >= spans[c.spanIdx].NumRows {
		c.spanIdx++
		c.consumed = 0
	}

	span := spans[c.spanIdx]
	pos := span.FirstRowPos + c.consumed
	c.consumed++

	return pos
}

// synthesizeRowLineageColumns fills the requested row-lineage columns from task
// constants: per the v3 spec a null value inherits first_row_id + position /
// data_sequence_number, a non-null value is kept. A column absent from the batch
// is appended. The caller sets synthesizeRowID/synthesizeSeq only when the
// matching task constant is present.
//
// MUST run before any row-dropping step: _row_id is first_row_id + the row's
// ORIGINAL position, drawn from cursor (advanced once per row in the full batch),
// so it is only correct before rows are dropped. ToRequestedSchema then resolves
// the columns by reserved field id.
func synthesizeRowLineageColumns(
	ctx context.Context,
	cursor *rowPositionCursor,
	task FileScanTask,
	batch arrow.RecordBatch,
	synthesizeRowID, synthesizeSeq bool,
) (arrow.RecordBatch, error) {
	alloc := compute.GetAllocator(ctx)
	schema := batch.Schema()
	nrows := batch.NumRows()

	fields := append([]arrow.Field(nil), schema.Fields()...)
	newCols := append([]arrow.Array(nil), batch.Columns()...)
	var built []arrow.Array
	defer func() {
		for _, a := range built {
			a.Release()
		}
	}()

	// perRow, when set, runs once for every physical row in order — including
	// rows whose explicit value wins — so a cursor stays in lockstep with the
	// batch even when value is never called.
	synth := func(name string, fieldID int, perRow func(k int64), value func(k int64) int64) error {
		idx := fieldIndexByID(schema, fieldID)
		var existing *array.Int64
		if idx >= 0 {
			var ok bool
			if existing, ok = newCols[idx].(*array.Int64); !ok {
				return fmt.Errorf("row-lineage column %s is %s, want int64", name, newCols[idx].DataType())
			}
		}

		bldr := array.NewInt64Builder(alloc)
		defer bldr.Release()
		bldr.Reserve(int(nrows))
		for k := range nrows {
			if perRow != nil {
				perRow(k)
			}
			if existing != nil && !existing.IsNull(int(k)) {
				bldr.Append(existing.Value(int(k)))
			} else {
				bldr.Append(value(k))
			}
		}
		arr := bldr.NewArray()
		built = append(built, arr)

		if idx >= 0 {
			newCols[idx] = arr

			return nil
		}
		fields = append(fields, arrow.Field{
			Name:     name,
			Type:     arrow.PrimitiveTypes.Int64,
			Nullable: true,
			Metadata: arrow.NewMetadata([]string{ArrowParquetFieldIDKey}, []string{strconv.Itoa(fieldID)}),
		})
		newCols = append(newCols, arr)

		return nil
	}

	if synthesizeRowID {
		first := *task.FirstRowID
		var pos int64
		if err := synth(iceberg.RowIDColumnName, iceberg.RowIDFieldID,
			func(int64) { pos = cursor.next() },
			func(int64) int64 { return first + pos },
		); err != nil {
			return nil, err
		}
	}
	if synthesizeSeq {
		seq := *task.DataSequenceNumber
		if err := synth(iceberg.LastUpdatedSequenceNumberColumnName, iceberg.LastUpdatedSequenceNumberFieldID, nil,
			func(int64) int64 {
				return seq
			}); err != nil {
			return nil, err
		}
	}

	meta := schema.Metadata()

	return array.NewRecordBatch(arrow.NewSchema(fields, &meta), newCols, nrows), nil
}

func (as *arrowScan) processRecords(
	ctx context.Context,
	task tblutils.Enumerated[FileScanTask],
	fileSchema *iceberg.Schema,
	rowFilter iceberg.BooleanExpression,
	rdr tblutils.FileReader,
	columns []int,
	pipeline []recProcessFn,
	posSource *rowPositionSource,
	sink *recordSink,
) error {
	return as.processRecordsWithPlans(ctx, task, fileSchema, rowFilter, rdr, columns, pipeline, posSource, sink, nil)
}

// adjustParquetTaskRange keeps a planned range within the physical file when
// manifest and footer sizes differ. It only clamps ranges that run past the
// physical end; caller-owned task boundaries must never be extended.
func adjustParquetTaskRange(task FileScanTask, physicalFileSize int64) (start, length int64) {
	start, length = task.Start, task.Length
	if task.File == nil || physicalFileSize <= 0 || start < 0 || start > physicalFileSize {
		return start, length
	}

	maxLength := physicalFileSize - start
	if length > maxLength {
		length = maxLength
	}

	return start, length
}

func (as *arrowScan) processRecordsWithPlans(
	ctx context.Context,
	task tblutils.Enumerated[FileScanTask],
	fileSchema *iceberg.Schema,
	rowFilter iceberg.BooleanExpression,
	rdr tblutils.FileReader,
	columns []int,
	pipeline []recProcessFn,
	posSource *rowPositionSource,
	sink *recordSink,
	plans *compiledFileFilterPlans,
) (err error) {
	var (
		testRowGroups any
		recRdr        array.RecordReader
	)
	pruningFilter := rowFilter
	if as.rowGroupFilter != nil {
		pruningFilter = as.rowGroupFilter
	}
	if pruningFilter == nil {
		pruningFilter = iceberg.AlwaysTrue{}
	}

	// Row-group stats/dictionary/bloom pruning skips whole groups, so emitted batches no
	// longer cover contiguous file positions. Steps that key on the original
	// position (row-lineage _row_id, positional/DV deletes, generated position
	// deletes) recover it from posSource, which the tester seeds with each
	// surviving group's file position before any record is read, so pruning
	// stays enabled.
	switch {
	case task.Value.File.FileFormat() == iceberg.ParquetFile:
		var tester *tblutils.ParquetRowGroupTester
		if plans != nil && plans.pruning != nil {
			tester = &tblutils.ParquetRowGroupTester{
				StatsFn:         plans.pruning.statsEvaluator(),
				BloomPreds:      plans.pruning.bloomPreds,
				DictionaryPreds: plans.pruning.dictionaryPreds,
			}
		} else {
			logicalSchema := as.filterSchema
			if logicalSchema == nil {
				logicalSchema = as.projectedSchema
			}

			hasMissingDefault, err := pruningFilterHasMissingInitialDefault(
				pruningFilter, logicalSchema, fileSchema)
			if err != nil {
				return err
			}
			if hasMissingDefault {
				// TranslateColumnNames treats fields missing from the physical file
				// as null. That is unsafe for fields added with a non-null
				// initial-default because Iceberg materializes the default on read.
				// Keep the actual row filter, but conservatively skip early pruning.
				pruningFilter = iceberg.AlwaysTrue{}
			}

			pruningFilter, err = stripExtractPredicates(pruningFilter)
			if err != nil {
				return err
			}

			filePruningFilter, err := iceberg.TranslateColumnNames(pruningFilter, fileSchema)
			if err != nil {
				return err
			}

			filePruningFilter, err = iceberg.BindExpr(fileSchema, filePruningFilter, as.caseSensitive)
			if err != nil {
				return err
			}

			statsFn, err := newParquetRowGroupStatsEvaluator(fileSchema, filePruningFilter, false)
			if err != nil {
				return err
			}

			bloomPreds, err := newBloomFilterPredicates(filePruningFilter)
			if err != nil {
				return err
			}

			dictionaryPreds, err := newDictionaryPredicates(filePruningFilter)
			if err != nil {
				return err
			}

			tester = &tblutils.ParquetRowGroupTester{
				StatsFn:         statsFn,
				BloomPreds:      bloomPreds,
				DictionaryPreds: dictionaryPreds,
			}
		}
		// A complete task already reads every row group. Leave its byte range
		// unset so malformed or legacy row-group offsets cannot change the
		// historical full-file behavior. A zero-value task is also treated as
		// complete for callers that omit the range. Split and remote partial
		// tasks carry their range through to the reader.
		if task.Value.Start != 0 ||
			(task.Value.Length != 0 && task.Value.Length != task.Value.File.FileSizeBytes()) {
			tester.RangeSet = true
			tester.Start, tester.Length = adjustParquetTaskRange(task.Value, rdr.SourceFileSize())
			tester.PlanningSplitOffsets = task.Value.File.SplitOffsets()
		}
		if posSource != nil {
			tester.Survivors = &posSource.spans
		}
		testRowGroups = tester
	}

	recRdr, err = rdr.GetRecords(ctx, columns, testRowGroups)
	if err != nil {
		return err
	}
	defer recRdr.Release()

	var (
		idx  int
		prev arrow.RecordBatch
	)

	for recRdr.Next() {
		if prev != nil {
			sink.send(enumeratedRecord{Record: tblutils.Enumerated[arrow.RecordBatch]{
				Value: prev, Index: idx, Last: false,
			}, Task: task})
			idx++
		}

		prev = recRdr.RecordBatch()
		prev.Retain()

		for _, f := range pipeline {
			prev, err = f(prev)
			if err != nil {
				return err
			}
		}
	}

	if prev != nil {
		sink.send(enumeratedRecord{Record: tblutils.Enumerated[arrow.RecordBatch]{
			Value: prev, Index: idx, Last: true,
		}, Task: task})
	} else {
		// The reader produced no batches (e.g. every row group was pruned by
		// stats). The sequenced channel in createIterator still needs this
		// task's Last record to release the following tasks' records, so emit
		// an empty batch just like the dropFile path does; without it every
		// record queued behind this task is silently discarded on close.
		var emptySchema *arrow.Schema
		emptySchema, err = SchemaToArrowSchema(as.projectedSchema, nil, false, as.useLargeTypes)
		if err != nil {
			return err
		}
		sink.send(enumeratedRecord{Record: tblutils.Enumerated[arrow.RecordBatch]{
			Value: array.NewRecordBatch(emptySchema, nil, 0), Index: idx, Last: true,
		}, Task: task})
	}

	if recRdr.Err() != nil && recRdr.Err() != io.EOF {
		err = recRdr.Err()
	}

	return err
}

func pruningFilterHasMissingInitialDefault(
	filter iceberg.BooleanExpression,
	logicalSchema, fileSchema *iceberg.Schema,
) (bool, error) {
	if logicalSchema == nil || fileSchema == nil {
		return false, nil
	}

	fieldIDs, err := iceberg.ExtractFieldIDs(filter)
	if err != nil {
		return false, err
	}

	for _, fieldID := range fieldIDs {
		if _, found := fileSchema.FindFieldByID(fieldID); found {
			continue
		}

		field, found := logicalSchema.FindFieldByID(fieldID)
		if found && field.InitialDefault != nil {
			return true, nil
		}
	}

	return false, nil
}

func (as *arrowScan) recordsFromTask(ctx context.Context, task tblutils.Enumerated[FileScanTask], sink *recordSink, positionalDeletes positionDeletes, dvBitmap *dv.RoaringPositionBitmap, eqDeleteSets []*equalityDeleteSet, invariants *arrowScanInvariants) (err error) {
	defer func() {
		if err != nil {
			sink.fail(task, err)
		}
	}()

	var (
		rowFilter   iceberg.BooleanExpression
		rdr         tblutils.FileReader
		iceSchema   *iceberg.Schema
		colIndices  []int
		filterFunc  recProcessFn
		dropFile    bool
		filterPlans *compiledFileFilterPlans
	)

	rowFilter, err = as.rowFilterForTask(task.Value)
	if err != nil {
		return err
	}

	iceSchema, colIndices, rdr, err = as.prepareToRead(ctx, task.Value.File, invariants)
	if err != nil {
		return err
	}
	defer iceinternal.CheckedClose(rdr, &err)
	if task.Value.Residual == nil {
		filterPlans, err = as.cachedFileFilterPlans(
			iceSchema, task.Value.File.FileFormat() == iceberg.ParquetFile)
		if err != nil {
			return err
		}
	}

	pipeline := make([]recProcessFn, 0, 4)

	// Synthesize lineage before any row-dropping step so deletes/filters can't
	// renumber survivors' _row_id; readSchema carries the field ids for
	// ToRequestedSchema. Gated so non-lineage scans keep the allocation-free path.
	readSchema := iceSchema
	rowLineageEnabled, lerr := strconv.ParseBool(as.options.Get(ScanOptionRowLineageEnabled, "true"))
	if lerr != nil {
		rowLineageEnabled = true
	}
	_, wantRowID := as.projectedSchema.FindFieldByID(iceberg.RowIDFieldID)
	_, wantSeqNum := as.projectedSchema.FindFieldByID(iceberg.LastUpdatedSequenceNumberFieldID)
	synthesizeRowID := rowLineageEnabled && wantRowID && task.Value.FirstRowID != nil
	synthesizeSeq := rowLineageEnabled && wantSeqNum && task.Value.DataSequenceNumber != nil

	// Every position-keyed step (_row_id synthesis, positional/DV deletes) needs
	// each row's original file position; under row-group pruning the emitted rows
	// are non-contiguous. processRecords seeds one shared source with the
	// surviving groups and each step takes its own cursor over it.
	hasDV := dvBitmap != nil && !dvBitmap.IsEmpty()
	// Per the v3 spec a DV supersedes positional delete files for a data file, so
	// when both are attached (only possible on a hand-built task; PlanFiles never
	// produces it) the DV wins and positional deletes are ignored. This mirrors
	// the planner (scanner.go) and Java's DeleteFileIndex, and keeps at most one
	// row-dropping position step in the pipeline so cursors never run over a
	// sequence an earlier step already shortened.
	applyPosDeletes := len(positionalDeletes) > 0 && !hasDV
	var posSource *rowPositionSource
	if synthesizeRowID || applyPosDeletes || hasDV {
		posSource = &rowPositionSource{}
	}

	if synthesizeRowID || synthesizeSeq {
		// Mirror the columns synthesize will append so ToRequestedSchema never
		// resolves a lineage field missing from the batch.
		readSchema = iceberg.SchemaWithRowLineageColumns(iceSchema, synthesizeRowID, synthesizeSeq)
		taskVal := task.Value
		// Only _row_id depends on position, so a cursor is taken only then.
		var cursor *rowPositionCursor
		if synthesizeRowID {
			cursor = posSource.cursor()
		}
		pipeline = append(pipeline, func(r arrow.RecordBatch) (arrow.RecordBatch, error) {
			defer r.Release()

			return synthesizeRowLineageColumns(ctx, cursor, taskVal, r, synthesizeRowID, synthesizeSeq)
		})
	}

	if applyPosDeletes {
		deletes, err := collectPosDeletePositions(positionalDeletes)
		if err != nil {
			return err
		}

		pipeline = append(pipeline, processPositionalDeletes(ctx, deletes, posSource.cursor()))
	}

	if hasDV {
		pipeline = append(pipeline, filterByDeletionVector(ctx, dvBitmap, task.Value.File.Count(), posSource.cursor()))
	}

	if len(eqDeleteSets) > 0 {
		eqFn, eqErr := processEqualityDeletesColumnarForFile(ctx, eqDeleteSets, iceSchema, task.Value.File.FilePath())
		if eqErr != nil {
			return eqErr
		}

		pipeline = append(pipeline, eqFn)
	}

	if filterPlans != nil {
		filterFunc, dropFile, err = as.getRecordFilterFromPlan(ctx, filterPlans.record)
	} else {
		filterFunc, dropFile, err = as.getRecordFilter(ctx, iceSchema, rowFilter)
	}
	if err != nil {
		return err
	}

	if dropFile {
		var emptySchema *arrow.Schema
		emptySchema, err = SchemaToArrowSchema(as.projectedSchema, nil, false, as.useLargeTypes)
		if err != nil {
			return err
		}
		sink.send(enumeratedRecord{Task: task, Record: tblutils.Enumerated[arrow.RecordBatch]{
			Value: array.NewRecordBatch(emptySchema, nil, 0), Index: 0, Last: true,
		}})

		return err
	}

	if filterFunc != nil {
		pipeline = append(pipeline, filterFunc)
	}

	pipeline = append(pipeline, func(r arrow.RecordBatch) (arrow.RecordBatch, error) {
		defer r.Release()

		return ToRequestedSchema(ctx, as.projectedSchema, readSchema, r, SchemaOptions{
			UseLargeTypes:   as.useLargeTypes,
			TableProperties: invariants.tableProperties,
		})
	})

	err = as.processRecordsWithPlans(ctx, task, iceSchema, rowFilter, rdr, colIndices, pipeline, posSource, sink, filterPlans)

	return err
}

func (as *arrowScan) producePosDeletesFromTask(ctx context.Context, task tblutils.Enumerated[FileScanTask], positionalDeletes positionDeletes, sink *recordSink, invariants *arrowScanInvariants) (err error) {
	defer func() {
		if err != nil {
			sink.fail(task, err)
		}
	}()

	var (
		rdr         tblutils.FileReader
		iceSchema   *iceberg.Schema
		colIndices  []int
		filterFunc  recProcessFn
		dropFile    bool
		filterPlans *compiledFileFilterPlans
	)

	iceSchema, colIndices, rdr, err = as.prepareToRead(ctx, task.Value.File, invariants)
	if err != nil {
		return err
	}
	defer iceinternal.CheckedClose(rdr, &err)
	filterPlans, err = as.cachedFileFilterPlans(
		iceSchema, task.Value.File.FileFormat() == iceberg.ParquetFile)
	if err != nil {
		return err
	}

	fields := append(iceSchema.Fields(), iceberg.PositionalDeleteSchema.Fields()...)
	enrichedIcebergSchema := iceberg.NewSchema(iceSchema.ID+1, fields...)

	// Each row's original file position is stamped into the generated delete
	// records and matched against any pre-existing positional deletes. Under the
	// delete filter, pruned row groups hold no matching rows, so skipping them
	// drops no generated deletes; cursors over the shared source keep the
	// emitted positions correct across the gaps.
	posSource := &rowPositionSource{}
	pipeline := make([]recProcessFn, 0, 2)
	pipeline = append(pipeline, enrichRecordsWithPosDeleteFields(ctx, task.Value.File, posSource.cursor()))
	if len(positionalDeletes) > 0 {
		deletes, err := collectPosDeletePositions(positionalDeletes)
		if err != nil {
			return err
		}

		pipeline = append(pipeline, processPositionalDeletes(ctx, deletes, posSource.cursor()))
	}

	filterFunc, dropFile, err = as.getRecordFilterFromPlan(ctx, filterPlans.record)
	if err != nil {
		return err
	}

	// Nothing to delete in a dropped file
	if dropFile {
		var emptySchema *arrow.Schema
		emptySchema, err = SchemaToArrowSchema(iceberg.PositionalDeleteSchema, nil, false, as.useLargeTypes)
		if err != nil {
			return err
		}
		sink.send(enumeratedRecord{Task: task, Record: tblutils.Enumerated[arrow.RecordBatch]{
			Value: array.NewRecordBatch(emptySchema, nil, 0), Index: 0, Last: true,
		}})

		return err
	}

	if filterFunc != nil {
		pipeline = append(pipeline, filterFunc)
	}
	pipeline = append(pipeline, func(r arrow.RecordBatch) (arrow.RecordBatch, error) {
		defer r.Release()

		return ToRequestedSchema(ctx, iceberg.PositionalDeleteSchema, enrichedIcebergSchema, r, SchemaOptions{IncludeFieldIDs: true, UseLargeTypes: as.useLargeTypes})
	})

	err = as.processRecordsWithPlans(ctx, task, iceSchema, as.boundRowFilter, rdr, colIndices, pipeline, posSource, sink, filterPlans)

	return err
}

func createIterator(ctx context.Context, numWorkers uint, records <-chan enumeratedRecord, deletesPerFile perFilePosDeletes, cancel context.CancelCauseFunc, rowLimit int64) iter.Seq2[arrow.RecordBatch, error] {
	return createIteratorWithCleanup(ctx, numWorkers, records, deletesPerFile, cancel, rowLimit, nil)
}

func createIteratorWithCleanup(ctx context.Context, numWorkers uint, records <-chan enumeratedRecord, deletesPerFile perFilePosDeletes, cancel context.CancelCauseFunc, rowLimit int64, cleanup func()) iter.Seq2[arrow.RecordBatch, error] {
	isBeforeAny := func(batch enumeratedRecord) bool {
		return batch.Task.Index < 0
	}

	sequenced := tblutils.MakeSequencedChanWithDiscard(uint(numWorkers), records,
		func(left, right *enumeratedRecord) bool {
			switch {
			case isBeforeAny(*left):
				return true
			case isBeforeAny(*right):
				return false
			case left.Err != nil || right.Err != nil:
				return true
			case left.Task.Index == right.Task.Index:
				return left.Record.Index < right.Record.Index
			default:
				return left.Task.Index < right.Task.Index
			}
		}, func(prev, next *enumeratedRecord) bool {
			switch {
			case isBeforeAny(*prev):
				return next.Task.Index == 0 && next.Record.Index == 0
			case next.Err != nil:
				return true
			case prev.Task.Index == next.Task.Index:
				return next.Record.Index == prev.Record.Index+1
			default:
				return next.Task.Index == prev.Task.Index+1 &&
					prev.Record.Last && next.Record.Index == 0
			}
		}, enumeratedRecord{Task: tblutils.Enumerated[FileScanTask]{Index: -1}}, func(rec enumeratedRecord) {
			rec.credits.release()
			if rec.Record.Value != nil {
				rec.Record.Value.Release()
			}
		})

	totalRowCount := int64(0)

	return func(yield func(arrow.RecordBatch, error) bool) {
		defer func() {
			for rec := range sequenced {
				rec.credits.release()
				if rec.Record.Value != nil {
					rec.Record.Value.Release()
				}
			}

			releasePerFilePosDeletes(deletesPerFile)
			if cleanup != nil {
				cleanup()
			}
		}()

		defer cancel(nil)

		for {
			select {
			case <-ctx.Done():
				if err := context.Cause(ctx); err != nil {
					yield(nil, err)
				}

				return
			case enum, ok := <-sequenced:
				if !ok {
					if err := context.Cause(ctx); err != nil {
						yield(nil, err)
					}

					return
				}

				enum.credits.release()
				if enum.Err != nil {
					yield(nil, enum.Err)

					return
				}

				rec := enum.Record.Value
				if rowLimit > 0 {
					if totalRowCount >= rowLimit {
						rec.Release()

						return
					} else if totalRowCount+rec.NumRows() > rowLimit {
						defer rec.Release()
						rec = rec.NewSlice(0, rowLimit-totalRowCount)
					}
				}

				if rec.NumRows() == 0 {
					// skip empty records
					rec.Release()

					continue
				}

				if !yield(rec, nil) {
					return
				}
				totalRowCount += rec.NumRows()
				if rowLimit > 0 && totalRowCount >= rowLimit {
					return
				}
			}
		}
	}
}

func (as *arrowScan) recordBatchesFromTasksAndDeletes(ctx context.Context, tasks []FileScanTask, positionDeleteLoader *lazyPositionDeleteLoader, dvLoader *lazyDeletionVectorLoader, equalityDeleteLoader *lazyEqualityDeleteLoader, invariants *arrowScanInvariants) iter.Seq2[arrow.RecordBatch, error] {
	return func(yield func(arrow.RecordBatch, error) bool) {
		extSet := substrait.NewExtensionSet()
		scanCtx, cancel := context.WithCancelCause(exprs.WithExtensionIDSet(ctx, extSet))
		numWorkers := min(as.concurrency, len(tasks))
		taskChan := make(chan tblutils.Enumerated[FileScanTask], len(tasks))
		records := make(chan enumeratedRecord, numWorkers)

		var wg sync.WaitGroup
		wg.Add(numWorkers)
		for range numWorkers {
			go func() {
				defer wg.Done()
				sink := newRecordSink(records)
				for {
					if err := sink.reserve(scanCtx); err != nil {
						return
					}
					select {
					case <-scanCtx.Done():
						return
					case task, ok := <-taskChan:
						if !ok {
							return
						}
						if scanCtx.Err() != nil {
							return
						}

						filePath := task.Value.File.FilePath()
						var positionalDeletes positionDeletes
						if positionDeleteLoader != nil {
							var err error
							positionalDeletes, err = positionDeleteLoader.load(scanCtx, task.Value)
							if err != nil {
								sink.fail(task, err)
								cancel(err)

								return
							}
						}

						dvBitmap, err := dvLoader.load(scanCtx, filePath)
						if err != nil {
							sink.fail(task, err)
							cancel(err)

							return
						}
						eqDeleteSets, err := equalityDeleteLoader.load(scanCtx, task.Value)
						if err != nil {
							sink.fail(task, err)
							cancel(err)

							return
						}

						if err := as.recordsFromTask(scanCtx, task, sink,
							positionalDeletes,
							dvBitmap,
							eqDeleteSets,
							invariants); err != nil {
							cancel(err)

							return
						}
					}
				}
			}()
		}

		go func() {
			defer func() {
				close(taskChan)
				wg.Wait()
				close(records)
			}()

			for i, t := range tasks {
				select {
				case <-scanCtx.Done():
					return
				case taskChan <- tblutils.Enumerated[FileScanTask]{
					Value: t, Index: i, Last: i == len(tasks)-1,
				}:
				}
			}
		}()

		var cleanup func()
		if positionDeleteLoader != nil {
			cleanup = positionDeleteLoader.release
		}
		createIteratorWithCleanup(scanCtx, uint(numWorkers), records, nil,
			cancel, as.rowLimit, cleanup)(yield)
	}
}

// GetRecords prepares the projected Arrow schema and a single-use record
// iterator. Positional- and equality-delete files are opened and read during
// iteration. Errors from those files are delivered through the iterator only if
// iteration reaches the task that encounters the error; a row limit or early
// termination may finish the scan before the error is observed. Deletion-vector
// errors are returned before the iterator is created.
func (as *arrowScan) GetRecords(ctx context.Context, tasks []FileScanTask) (*arrow.Schema, iter.Seq2[arrow.RecordBatch, error], error) {
	var err error
	as.useLargeTypes, err = strconv.ParseBool(as.options.Get(ScanOptionArrowUseLargeTypes, "false"))
	if err != nil {
		as.useLargeTypes = false
	}

	tableProperties := as.metadata.Properties()
	batchSize := as.options.Get(ParquetBatchSizeKey, "")
	if as.arrowBatchSize > 0 {
		batchSize = strconv.Itoa(as.arrowBatchSize)
	}
	if batchSize != "" {
		tableProperties = maps.Clone(tableProperties)
		if tableProperties == nil {
			tableProperties = iceberg.Properties{}
		}
		tableProperties[ParquetBatchSizeKey] = batchSize
	}
	ctx = tblutils.WithTableProperties(ctx, tableProperties)

	resultSchema, err := SchemaToArrowSchemaWithOptions(as.projectedSchema, ArrowSchemaOptions{
		UseLargeTypes:   as.useLargeTypes,
		TableProperties: tableProperties,
	})
	if err != nil {
		return nil, nil, err
	}

	if as.rowLimit == 0 {
		return resultSchema, func(yield func(arrow.RecordBatch, error) bool) {}, nil
	}

	if len(tasks) > 1 {
		as.cacheFileReadPlan = true
		ctx = tblutils.WithParquetMetadataCache(ctx)
	} else {
		as.cacheFileReadPlan = false
	}

	invariants, err := as.scanInvariants(tableProperties)
	if err != nil {
		return nil, nil, err
	}
	if err := as.addTaskProjectedFieldIDs(invariants, tasks); err != nil {
		return nil, nil, err
	}

	// Index DV ownership up front, but defer Puffin reads until a task using a
	// referenced data file enters the iterator. The loader keeps each shared
	// Puffin group cached after its first read.
	dvLoader, err := newLazyDeletionVectorLoader(as.fs, tasks)
	if err != nil {
		return nil, nil, err
	}

	var tableSchemas []*iceberg.Schema

loadSchemaHistory:
	for _, task := range tasks {
		for _, deleteFile := range task.EqualityDeleteFiles {
			for _, fieldID := range deleteFile.EqualityFieldIDs() {
				if _, found := invariants.tableSchema.FindFieldByID(fieldID); !found {
					tableSchemas = as.metadata.Schemas()

					break loadSchemaHistory
				}
			}
		}
	}
	equalityDeleteLoader, err := newLazyEqualityDeleteLoader(
		as.fs, invariants.tableSchema, tableSchemas, invariants.nameMapping, tasks)
	if err != nil {
		return nil, nil, err
	}
	equalityDeleteLoader.addFieldIDs(invariants.projectedIDs)

	positionDeleteLoader := newLazyPositionDeleteLoader(as.fs, tasks)

	return resultSchema, as.recordBatchesFromTasksAndDeletes(ctx, tasks,
		positionDeleteLoader, dvLoader, equalityDeleteLoader, invariants), nil
}
