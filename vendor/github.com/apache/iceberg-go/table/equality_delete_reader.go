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
	"cmp"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"sync"
	"unsafe"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/bitutil"
	"github.com/apache/arrow-go/v18/arrow/compute"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/iceberg-go"
	iceinternal "github.com/apache/iceberg-go/internal"
	iceio "github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/table/internal"
	"golang.org/x/sync/errgroup"
)

var ErrAmbiguousEqualityColumn = errors.New("equality delete column is ambiguous")

// equalityDeleteSet holds the set of delete keys and the column names
// used to look them up in data records. Each set corresponds to one
// group of equality field IDs — delete files with different field IDs
// produce separate sets. The set is immutable after construction so it
// can be shared by tasks with the same delete files.
type equalityDeleteSet struct {
	keys     set[string]
	fieldIDs []int
	colNames []string
}

type arrowFieldRef struct {
	path []int
}

type arrowFieldRefsByID map[int][]arrowFieldRef

func equalityFieldLocation(filePath string) string {
	location := filePath
	if location == "" {
		location = "data record"
	}

	return location
}

// indexArrowFields derives Arrow child paths from the structurally aligned,
// ID-resolved Iceberg file schema. It deliberately ignores names: dots in an
// Iceberg name are literal and must not be interpreted as a path.
func indexArrowFields(schema *iceberg.Schema) arrowFieldRefsByID {
	refs := make(arrowFieldRefsByID)
	if schema == nil {
		return refs
	}

	var visit func([]iceberg.NestedField, []int)
	visit = func(fields []iceberg.NestedField, parentPath []int) {
		for i, field := range fields {
			path := append(append([]int(nil), parentPath...), i)
			refs[field.ID] = append(refs[field.ID], arrowFieldRef{path: path})

			if nested, ok := field.Type.(*iceberg.StructType); ok {
				visit(nested.Fields(), path)
			}
		}
	}
	visit(schema.Fields(), nil)

	return refs
}

type requestedArrowFieldResolver struct {
	requested map[int]struct{}
	refs      arrowFieldRefsByID
	path      []int
	pathBuf   [8]int
}

func (r *requestedArrowFieldResolver) visit(fields []iceberg.NestedField) {
	for i, field := range fields {
		parentLen := len(r.path)
		r.path = append(r.path, i)

		if _, ok := r.requested[field.ID]; ok {
			r.refs[field.ID] = append(r.refs[field.ID], arrowFieldRef{path: slices.Clone(r.path)})
		}

		if nested, ok := field.Type.(*iceberg.StructType); ok {
			r.visit(nested.FieldList)
		}

		r.path = r.path[:parentLen]
	}
}

// resolveArrowFieldsByID resolves paths only for the requested field IDs.
// It walks the schema using borrowed fields and keeps one reusable path stack,
// copying a path only when it matches an equality field.
func resolveArrowFieldsByID(schema *iceberg.Schema, fieldIDs []int) arrowFieldRefsByID {
	refs := make(arrowFieldRefsByID, len(fieldIDs))
	if schema == nil || len(fieldIDs) == 0 {
		return refs
	}

	requested := make(map[int]struct{}, len(fieldIDs))
	for _, fieldID := range fieldIDs {
		requested[fieldID] = struct{}{}
	}

	resolver := requestedArrowFieldResolver{
		requested: requested,
		refs:      refs,
	}
	resolver.path = resolver.pathBuf[:0]
	resolver.visit(schema.FieldsRef(iceinternal.SchemaRef{}))

	return refs
}

// indexArrowFieldsByMetadata is used for delete files whose Arrow fields carry
// IDs directly, before any name mapping is needed.
func indexArrowFieldsByMetadata(schema *arrow.Schema) arrowFieldRefsByID {
	refs := make(arrowFieldRefsByID)
	var visit func([]arrow.Field, []int)
	visit = func(fields []arrow.Field, parentPath []int) {
		for i, field := range fields {
			path := append(append([]int(nil), parentPath...), i)
			if id := getFieldID(field); id != nil {
				refs[*id] = append(refs[*id], arrowFieldRef{path: path})
			}

			if nested, ok := field.Type.(*arrow.StructType); ok {
				visit(nested.Fields(), path)
			}
		}
	}
	visit(schema.Fields(), nil)

	return refs
}

type requestedArrowMetadataFieldResolver struct {
	requested map[int]struct{}
	refs      arrowFieldRefsByID
	path      []int
	pathBuf   [8]int
}

func (r *requestedArrowMetadataFieldResolver) visitField(field arrow.Field, index int) {
	parentLen := len(r.path)
	r.path = append(r.path, index)

	if id := getFieldID(field); id != nil {
		if _, ok := r.requested[*id]; ok {
			r.refs[*id] = append(r.refs[*id], arrowFieldRef{path: slices.Clone(r.path)})
		}
	}

	if nested, ok := field.Type.(*arrow.StructType); ok {
		r.visitStruct(nested)
	}

	r.path = r.path[:parentLen]
}

func (r *requestedArrowMetadataFieldResolver) visitStruct(schema *arrow.StructType) {
	for i := range schema.NumFields() {
		r.visitField(schema.Field(i), i)
	}
}

// resolveArrowFieldsByMetadata resolves paths only for requested field IDs in
// an Arrow schema whose fields carry Iceberg IDs in metadata.
func resolveArrowFieldsByMetadata(schema *arrow.Schema, fieldIDs []int) arrowFieldRefsByID {
	refs := make(arrowFieldRefsByID, len(fieldIDs))
	if schema == nil || len(fieldIDs) == 0 {
		return refs
	}

	requested := make(map[int]struct{}, len(fieldIDs))
	for _, fieldID := range fieldIDs {
		requested[fieldID] = struct{}{}
	}

	resolver := requestedArrowMetadataFieldResolver{
		requested: requested,
		refs:      refs,
	}
	resolver.path = resolver.pathBuf[:0]
	for i := range schema.NumFields() {
		resolver.visitField(schema.Field(i), i)
	}

	return refs
}

func resolveArrowField(refs arrowFieldRefsByID, fieldID int, fieldName, filePath string) (arrowFieldRef, error) {
	matches := refs[fieldID]
	location := equalityFieldLocation(filePath)
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return arrowFieldRef{}, fmt.Errorf("%w: equality field ID %d (%s) in %s: found %d fields",
			ErrAmbiguousEqualityColumn, fieldID, fieldName, location, len(matches))
	}

	return arrowFieldRef{}, fmt.Errorf("equality field ID %d (%s) not found in %s", fieldID, fieldName, location)
}

func arrowArrayAtField(record arrow.RecordBatch, ref arrowFieldRef, fieldID int, fieldName, filePath string) (arrow.Array, error) {
	result, _, err := arrowArraysAtField(record, ref, fieldID, fieldName, filePath)

	return result, err
}

func arrowArraysAtField(record arrow.RecordBatch, ref arrowFieldRef, fieldID int, fieldName, filePath string) (arrow.Array, []arrow.Array, error) {
	location := filePath
	if location == "" {
		location = "data record"
	}
	if len(ref.path) == 0 || ref.path[0] >= int(record.NumCols()) {
		return nil, nil, fmt.Errorf("equality field ID %d (%s) not found in %s", fieldID, fieldName, location)
	}

	result := record.Column(ref.path[0])
	parents := make([]arrow.Array, 0, len(ref.path)-1)
	for _, index := range ref.path[1:] {
		structArray, ok := result.(*array.Struct)
		if !ok || index >= structArray.NumField() {
			return nil, nil, fmt.Errorf("equality field ID %d (%s) has unsupported nested path in %s", fieldID, fieldName, location)
		}
		parents = append(parents, result)
		result = structArray.Field(index)
	}

	return result, parents, nil
}

func makeArrowFieldEncoder(record arrow.RecordBatch, ref arrowFieldRef, fieldID int, fieldName, filePath string) (colEncoder, error) {
	column, parents, err := arrowArraysAtField(record, ref, fieldID, fieldName, filePath)
	if err != nil {
		return nil, err
	}

	encoder := makeColEncoder(column)
	if len(parents) == 0 {
		return encoder, nil
	}

	return func(buf *bytes.Buffer, row int) {
		for _, parent := range parents {
			if parent.IsNull(row) {
				buf.WriteByte(0)

				return
			}
		}

		encoder(buf, row)
	}, nil
}

type equalityDeleteFileSet struct {
	id       int
	groupKey string
	*equalityDeleteSet
}

func newEqualityDeleteFileSet(id int, deleteSet *equalityDeleteSet) *equalityDeleteFileSet {
	return &equalityDeleteFileSet{
		id: id,
		// Keys are encoded in fieldIDs order, so this key must remain
		// order-sensitive unless the encoded keys are canonicalized too.
		groupKey:          fmt.Sprint(deleteSet.fieldIDs),
		equalityDeleteSet: deleteSet,
	}
}

func schemaForEqualityFields(current *iceberg.Schema, schemas []*iceberg.Schema, fieldIDs []int) *iceberg.Schema {
	hasAllFields := func(schema *iceberg.Schema) bool {
		for _, fieldID := range fieldIDs {
			if _, ok := schema.FindFieldByID(fieldID); !ok {
				return false
			}
		}

		return true
	}

	if hasAllFields(current) {
		return current
	}
	// Scan tasks do not retain the equality delete's sequence number, so use
	// the newest schema that can resolve the file's complete equality key.
	for i := len(schemas) - 1; i >= 0; i-- {
		if hasAllFields(schemas[i]) {
			return schemas[i]
		}
	}

	return current
}

type lazyEqualityDeleteLoader struct {
	fs           iceio.IO
	tableSchema  *iceberg.Schema
	tableSchemas []*iceberg.Schema
	nameMapping  iceberg.NameMapping
	files        map[string]*lazyEqualityDeleteFile
	combinations sync.Map
}

type lazyEqualityDeleteFile struct {
	id       int
	dataFile iceberg.DataFile
	fieldIDs []int

	once sync.Once
	set  *equalityDeleteFileSet
	err  error
}

type lazyEqualityDeleteCombination struct {
	once sync.Once
	set  *equalityDeleteSet
}

func newLazyEqualityDeleteLoader(
	fs iceio.IO,
	tableSchema *iceberg.Schema,
	tableSchemas []*iceberg.Schema,
	nameMapping iceberg.NameMapping,
	tasks []FileScanTask,
) (*lazyEqualityDeleteLoader, error) {
	loader := &lazyEqualityDeleteLoader{
		fs:           fs,
		tableSchema:  tableSchema,
		tableSchemas: tableSchemas,
		nameMapping:  nameMapping,
		files:        make(map[string]*lazyEqualityDeleteFile),
	}

	for _, task := range tasks {
		for _, dataFile := range task.EqualityDeleteFiles {
			if dataFile.ContentType() != iceberg.EntryContentEqDeletes {
				continue
			}

			fieldIDs := dataFile.EqualityFieldIDs()
			if len(fieldIDs) == 0 {
				return nil, fmt.Errorf("%w: equality delete file %s", ErrEmptyEqualityFieldIDs, dataFile.FilePath())
			}

			path := dataFile.FilePath()
			if _, ok := loader.files[path]; ok {
				continue
			}

			loader.files[path] = &lazyEqualityDeleteFile{
				id:       len(loader.files),
				dataFile: dataFile,
				fieldIDs: fieldIDs,
			}
		}
	}

	if len(loader.files) == 0 {
		return nil, nil
	}

	return loader, nil
}

func (l *lazyEqualityDeleteLoader) addFieldIDs(idset set[int]) {
	if l == nil {
		return
	}

	for _, file := range l.files {
		for _, fieldID := range file.fieldIDs {
			idset[fieldID] = struct{}{}
		}
	}
}

func (l *lazyEqualityDeleteLoader) loadFile(ctx context.Context, file *lazyEqualityDeleteFile) (*equalityDeleteFileSet, error) {
	file.once.Do(func() {
		deleteSchema := schemaForEqualityFields(l.tableSchema, l.tableSchemas, file.fieldIDs)
		keys, colNames, err := readEqualityDeleteFile(
			ctx, l.fs, deleteSchema, l.nameMapping, file.dataFile, file.fieldIDs)
		if err != nil {
			file.err = err

			return
		}

		file.set = newEqualityDeleteFileSet(file.id, &equalityDeleteSet{
			fieldIDs: file.fieldIDs,
			colNames: colNames,
			keys:     keys,
		})
	})

	return file.set, file.err
}

func (l *lazyEqualityDeleteLoader) combine(files []*equalityDeleteFileSet) *equalityDeleteSet {
	files = normalizeEqualityDeleteFiles(files)
	if len(files) == 1 {
		return files[0].equalityDeleteSet
	}

	key := equalityDeleteSetCombinationKey(files)
	entryValue, _ := l.combinations.LoadOrStore(key, &lazyEqualityDeleteCombination{})
	entry := entryValue.(*lazyEqualityDeleteCombination)
	entry.once.Do(func() {
		entry.set = mergeEqualityDeleteSets(files)
	})

	return entry.set
}

func (l *lazyEqualityDeleteLoader) load(ctx context.Context, task FileScanTask) ([]*equalityDeleteSet, error) {
	if l == nil || len(task.EqualityDeleteFiles) == 0 {
		return nil, nil
	}
	if len(task.EqualityDeleteFiles) == 1 {
		dataFile := task.EqualityDeleteFiles[0]
		if dataFile.ContentType() != iceberg.EntryContentEqDeletes {
			return nil, nil
		}

		file, ok := l.files[dataFile.FilePath()]
		if !ok {
			return nil, nil
		}

		fileSet, err := l.loadFile(ctx, file)
		if err != nil {
			return nil, err
		}
		if len(fileSet.keys) == 0 {
			return nil, nil
		}

		return []*equalityDeleteSet{fileSet.equalityDeleteSet}, nil
	}

	perFile := make(map[string]*equalityDeleteFileSet, len(task.EqualityDeleteFiles))
	for _, dataFile := range task.EqualityDeleteFiles {
		if dataFile.ContentType() != iceberg.EntryContentEqDeletes {
			continue
		}

		path := dataFile.FilePath()
		if _, seen := perFile[path]; seen {
			continue
		}

		file, ok := l.files[path]
		if !ok {
			continue
		}

		fileSet, err := l.loadFile(ctx, file)
		if err != nil {
			return nil, err
		}
		perFile[path] = fileSet
	}

	if len(perFile) == 0 {
		return nil, nil
	}

	return buildEqualityDeleteSetsForTask(task, perFile, l.combine), nil
}

// readAllEqualityDeleteFiles reads all unique equality delete files from
// the tasks and builds per-task delete key sets. Returns nil if there are
// no equality deletes. Delete files with different equality field IDs are
// kept as separate sets (not merged).
func readAllEqualityDeleteFiles(ctx context.Context, fs iceio.IO, schema *iceberg.Schema, nameMapping iceberg.NameMapping, tasks []FileScanTask, concurrency int) (map[int][]*equalityDeleteSet, error) {
	type deleteFileInfo struct {
		id       int
		file     iceberg.DataFile
		fieldIDs []int
	}

	uniqueDeletes := make(map[string]deleteFileInfo)
	hasAny := false

	for _, t := range tasks {
		for _, d := range t.EqualityDeleteFiles {
			if d.ContentType() != iceberg.EntryContentEqDeletes {
				continue
			}

			if len(d.EqualityFieldIDs()) == 0 {
				return nil, fmt.Errorf("%w: equality delete file %s", ErrEmptyEqualityFieldIDs, d.FilePath())
			}

			hasAny = true
			if _, ok := uniqueDeletes[d.FilePath()]; !ok {
				uniqueDeletes[d.FilePath()] = deleteFileInfo{
					id:       len(uniqueDeletes),
					file:     d,
					fieldIDs: d.EqualityFieldIDs(),
				}
			}
		}
	}

	if !hasAny {
		return nil, nil
	}

	type deleteFileResult struct {
		id       int
		path     string
		fieldIDs []int
		colNames []string
		keys     set[string]
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	resultCh := make(chan deleteFileResult, len(uniqueDeletes))

	go func() {
		defer close(resultCh)

		for _, info := range uniqueDeletes {
			g.Go(func() error {
				keys, colNames, err := readEqualityDeleteFile(ctx, fs, schema, nameMapping, info.file, info.fieldIDs)
				if err != nil {
					return err
				}

				resultCh <- deleteFileResult{
					id:       info.id,
					path:     info.file.FilePath(),
					fieldIDs: info.fieldIDs,
					colNames: colNames,
					keys:     keys,
				}

				return nil
			})
		}

		_ = g.Wait()
	}()

	perFile := make(map[string]*equalityDeleteFileSet)
	for result := range resultCh {
		perFile[result.path] = newEqualityDeleteFileSet(result.id, &equalityDeleteSet{
			fieldIDs: result.fieldIDs,
			colNames: result.colNames,
			keys:     result.keys,
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return buildEqualityDeleteSetsPerTask(tasks, perFile), nil
}

// buildEqualityDeleteSetsPerTask groups delete files by field IDs and merges
// their keys into the sets used by each scan task.
func buildEqualityDeleteSetsPerTask(
	tasks []FileScanTask,
	perFile map[string]*equalityDeleteFileSet,
) map[int][]*equalityDeleteSet {
	perTask := make(map[int][]*equalityDeleteSet)
	// File IDs are sufficient as the cache key because each ID identifies one
	// immutable delete set with a fixed equality-field group for this call.
	sharedSets := make(map[string]*equalityDeleteSet)
	combine := func(files []*equalityDeleteFileSet) *equalityDeleteSet {
		return equalityDeleteSetForFiles(files, sharedSets)
	}

	for i, t := range tasks {
		sets := buildEqualityDeleteSetsForTask(t, perFile, combine)
		if len(sets) > 0 {
			perTask[i] = sets
		}
	}

	return perTask
}

func buildEqualityDeleteSetsForTask(
	task FileScanTask,
	perFile map[string]*equalityDeleteFileSet,
	combine func([]*equalityDeleteFileSet) *equalityDeleteSet,
) []*equalityDeleteSet {
	if len(task.EqualityDeleteFiles) == 0 {
		return nil
	}
	if len(task.EqualityDeleteFiles) == 1 {
		fileSet, ok := perFile[task.EqualityDeleteFiles[0].FilePath()]
		if !ok || len(fileSet.keys) == 0 {
			return nil
		}

		return []*equalityDeleteSet{fileSet.equalityDeleteSet}
	}

	var (
		groupKey string
		groups   map[string][]*equalityDeleteFileSet
	)
	groupFiles := make([]*equalityDeleteFileSet, 0, len(task.EqualityDeleteFiles))

	for _, dataFile := range task.EqualityDeleteFiles {
		fileSet, ok := perFile[dataFile.FilePath()]
		if !ok {
			continue
		}

		if groups != nil {
			groups[fileSet.groupKey] = append(groups[fileSet.groupKey], fileSet)
		} else if len(groupFiles) == 0 {
			groupKey = fileSet.groupKey
			groupFiles = append(groupFiles, fileSet)
		} else if fileSet.groupKey != groupKey {
			groups = make(map[string][]*equalityDeleteFileSet, 2)
			groups[groupKey] = groupFiles
			groupFiles = nil
			groups[fileSet.groupKey] = append(groups[fileSet.groupKey], fileSet)
		} else {
			groupFiles = append(groupFiles, fileSet)
		}
	}

	if groups == nil {
		if len(groupFiles) == 0 {
			return nil
		}

		deleteSet := combine(groupFiles)
		if len(deleteSet.keys) == 0 {
			return nil
		}

		return []*equalityDeleteSet{deleteSet}
	}

	sets := make([]*equalityDeleteSet, 0, len(groups))
	for _, files := range groups {
		deleteSet := combine(files)
		if len(deleteSet.keys) > 0 {
			sets = append(sets, deleteSet)
		}
	}

	return sets
}

func equalityDeleteSetForFiles(
	files []*equalityDeleteFileSet,
	sharedSets map[string]*equalityDeleteSet,
) *equalityDeleteSet {
	files = normalizeEqualityDeleteFiles(files)
	if len(files) == 1 {
		return files[0].equalityDeleteSet
	}

	key := equalityDeleteSetCombinationKey(files)
	if deleteSet, ok := sharedSets[key]; ok {
		return deleteSet
	}

	deleteSet := mergeEqualityDeleteSets(files)
	sharedSets[key] = deleteSet

	return deleteSet
}

func normalizeEqualityDeleteFiles(files []*equalityDeleteFileSet) []*equalityDeleteFileSet {
	switch len(files) {
	case 0, 1:
		return files
	case 2:
		if files[0].id == files[1].id {
			return files[:1]
		}
		if files[0].id > files[1].id {
			files[0], files[1] = files[1], files[0]
		}

		return files
	}

	slices.SortFunc(files, func(a, b *equalityDeleteFileSet) int {
		return cmp.Compare(a.id, b.id)
	})

	return slices.CompactFunc(files, func(a, b *equalityDeleteFileSet) bool {
		return a.id == b.id
	})
}

func equalityDeleteSetCombinationKey(files []*equalityDeleteFileSet) string {
	combinationKey := make([]byte, 0, len(files)*8)
	for _, file := range files {
		combinationKey = binary.LittleEndian.AppendUint64(combinationKey, uint64(file.id))
	}

	return string(combinationKey)
}

func mergeEqualityDeleteSets(files []*equalityDeleteFileSet) *equalityDeleteSet {
	largest := 0
	for i := 1; i < len(files); i++ {
		if len(files[i].keys) > len(files[largest].keys) {
			largest = i
		}
	}

	deleteSet := &equalityDeleteSet{
		keys:     maps.Clone(files[largest].keys),
		fieldIDs: files[0].fieldIDs,
		colNames: files[0].colNames,
	}
	for i, file := range files {
		if i == largest {
			continue
		}

		for key := range file.keys {
			deleteSet.keys[key] = struct{}{}
		}
	}

	return deleteSet
}

// readEqualityDeleteFile reads a single equality delete file and returns
// the set of encoded delete keys and the column names used.
func readEqualityDeleteFile(ctx context.Context, fs iceio.IO, tableSchema *iceberg.Schema, nameMapping iceberg.NameMapping, dataFile iceberg.DataFile, fieldIDs []int) (keys set[string], colNames []string, err error) {
	src, err := internal.GetFile(ctx, fs, dataFile, true)
	if err != nil {
		return nil, nil, err
	}

	rdr, err := src.GetReader(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer iceinternal.CheckedClose(rdr, &err)

	if nameMapping == nil {
		nameMapping = tableSchema.NameMapping()
	}

	projectedIDs := make(map[int]struct{}, len(fieldIDs))
	for _, fieldID := range fieldIDs {
		projectedIDs[fieldID] = struct{}{}
	}

	projectedSchema, colIndices, err := rdr.PrunedSchema(projectedIDs, nameMapping)
	if err != nil {
		return nil, nil, err
	}

	hasFieldIDs, err := VisitArrowSchema(projectedSchema, hasIDs{})
	if err != nil {
		return nil, nil, err
	}

	var fileSchema *iceberg.Schema
	if !hasFieldIDs {
		fileSchema, err = ArrowSchemaToIcebergWithOptions(projectedSchema, ArrowToIcebergOptions{
			NameMapping: nameMapping,
			TableSchema: tableSchema,
		})
		if err != nil {
			return nil, nil, err
		}
	}

	var fieldRefsByID arrowFieldRefsByID
	if hasFieldIDs {
		fieldRefsByID = resolveArrowFieldsByMetadata(projectedSchema, fieldIDs)
	} else {
		fieldRefsByID = resolveArrowFieldsByID(fileSchema, fieldIDs)
	}

	// Resolve projected field paths from field IDs.
	colNames = make([]string, len(fieldIDs))
	fieldRefs := make([]arrowFieldRef, len(fieldIDs))

	for i, fid := range fieldIDs {
		name, ok := tableSchema.FindColumnName(fid)
		if !ok {
			return nil, nil, fmt.Errorf("equality delete field ID %d not found in table schema for %s", fid, dataFile.FilePath())
		}

		ref, err := resolveArrowField(fieldRefsByID, fid, name, dataFile.FilePath())
		if err != nil {
			return nil, nil, err
		}

		colNames[i] = name
		fieldRefs[i] = ref
	}

	// Stream projected batches directly into the encoded key set. The reader
	// owns the current record and releases it before producing the next one.
	recRdr, err := rdr.GetRecords(ctx, colIndices, nil)
	if err != nil {
		return nil, nil, err
	}
	defer recRdr.Release()

	keys = make(set[string])
	var keyBuf bytes.Buffer

	for recRdr.Next() {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		rec := recRdr.RecordBatch()
		encoders := make([]colEncoder, len(fieldRefs))
		for i, ref := range fieldRefs {
			encoders[i], err = makeArrowFieldEncoder(rec, ref, fieldIDs[i], colNames[i], dataFile.FilePath())
			if err != nil {
				return nil, nil, err
			}
		}

		numRows := int(rec.NumRows())
		for row := range numRows {
			if row&1023 == 0 {
				if err := ctx.Err(); err != nil {
					return nil, nil, err
				}
			}

			keyBuf.Reset()
			for _, enc := range encoders {
				enc(&keyBuf, row)
			}

			keys[keyBuf.String()] = struct{}{}
		}
	}
	if err := recRdr.Err(); err != nil {
		return nil, nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	return keys, colNames, nil
}

// bufPutUint16 writes a uint16 in big-endian without allocating.
func bufPutUint16(buf *bytes.Buffer, v uint16) {
	buf.Write([]byte{byte(v >> 8), byte(v)})
}

// bufPutUint32 writes a uint32 in big-endian without allocating.
func bufPutUint32(buf *bytes.Buffer, v uint32) {
	buf.Write([]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)})
}

// bufPutUint64 writes a uint64 in big-endian without allocating.
func bufPutUint64(buf *bytes.Buffer, v uint64) {
	buf.Write([]byte{
		byte(v >> 56), byte(v >> 48), byte(v >> 40), byte(v >> 32),
		byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v),
	})
}

// bufString returns the buffer contents as a string without copying.
// The returned string is only valid until the next buffer modification.
func bufString(buf *bytes.Buffer) string {
	b := buf.Bytes()

	return unsafe.String(unsafe.SliceData(b), len(b))
}

// encodeArrowValue writes a single Arrow value to the buffer for key
// encoding. Values are type-tagged and length-prefixed for variable-length
// types to avoid hash collisions.
func encodeArrowValue(buf *bytes.Buffer, arr arrow.Array, idx int) {
	if arr.IsNull(idx) {
		buf.WriteByte(0) // null tag

		return
	}

	buf.WriteByte(1) // non-null tag

	switch a := arr.(type) {
	case *array.Int8:
		buf.WriteByte(byte(a.Value(idx)))
	case *array.Int16:
		bufPutUint16(buf, uint16(a.Value(idx)))
	case *array.Int32:
		bufPutUint32(buf, uint32(a.Value(idx)))
	case *array.Int64:
		bufPutUint64(buf, uint64(a.Value(idx)))
	case *array.Float32:
		bufPutUint32(buf, math.Float32bits(a.Value(idx)))
	case *array.Float64:
		bufPutUint64(buf, math.Float64bits(a.Value(idx)))
	case *array.String:
		s := a.Value(idx)
		bufPutUint32(buf, uint32(len(s)))
		buf.WriteString(s)
	case *array.LargeString:
		s := a.Value(idx)
		bufPutUint32(buf, uint32(len(s)))
		buf.WriteString(s)
	case *array.Binary:
		b := a.Value(idx)
		bufPutUint32(buf, uint32(len(b)))
		buf.Write(b)
	case *array.LargeBinary:
		b := a.Value(idx)
		bufPutUint32(buf, uint32(len(b)))
		buf.Write(b)
	case *array.FixedSizeBinary:
		buf.Write(a.Value(idx))
	case *array.Boolean:
		if a.Value(idx) {
			buf.WriteByte(1)
		} else {
			buf.WriteByte(0)
		}
	case *array.Date32:
		bufPutUint32(buf, uint32(a.Value(idx)))
	case *array.Date64:
		bufPutUint64(buf, uint64(a.Value(idx)))
	case *array.Time32:
		bufPutUint32(buf, uint32(a.Value(idx)))
	case *array.Time64:
		bufPutUint64(buf, uint64(a.Value(idx)))
	case *array.Timestamp:
		bufPutUint64(buf, uint64(a.Value(idx)))
	default:
		// Fallback: length-prefixed string representation.
		s := a.ValueStr(idx)
		bufPutUint32(buf, uint32(len(s)))
		buf.WriteString(s)
	}
}

// colEncoder writes the value at row idx to buf. Resolved once per column
// to avoid per-row type switches.
type colEncoder func(buf *bytes.Buffer, row int)

func writeNullTagIfNull(buf *bytes.Buffer, arr arrow.Array, row int) bool {
	if arr.IsNull(row) {
		buf.WriteByte(0)

		return true
	}

	return false
}

func withNullGuard(arr arrow.Array, enc colEncoder) colEncoder {
	if arr.NullN() == 0 {
		return enc
	}

	return func(buf *bytes.Buffer, row int) {
		if writeNullTagIfNull(buf, arr, row) {
			return
		}

		enc(buf, row)
	}
}

// makeColEncoder returns a colEncoder for the given Arrow array that writes
// values directly from the raw typed backing slice when possible.
func makeColEncoder(arr arrow.Array) colEncoder {
	switch a := arr.(type) {
	case *array.Int8:
		vals := a.Int8Values()

		return withNullGuard(a, func(buf *bytes.Buffer, row int) {
			buf.WriteByte(1)
			buf.WriteByte(byte(vals[row]))
		})
	case *array.Int16:
		vals := a.Int16Values()

		return withNullGuard(a, func(buf *bytes.Buffer, row int) {
			buf.WriteByte(1)
			bufPutUint16(buf, uint16(vals[row]))
		})
	case *array.Int32:
		vals := a.Int32Values()

		return withNullGuard(a, func(buf *bytes.Buffer, row int) {
			buf.WriteByte(1)
			bufPutUint32(buf, uint32(vals[row]))
		})
	case *array.Int64:
		vals := a.Int64Values()

		return withNullGuard(a, func(buf *bytes.Buffer, row int) {
			buf.WriteByte(1)
			bufPutUint64(buf, uint64(vals[row]))
		})
	case *array.Float32:
		vals := a.Float32Values()

		return withNullGuard(a, func(buf *bytes.Buffer, row int) {
			buf.WriteByte(1)
			bufPutUint32(buf, math.Float32bits(vals[row]))
		})
	case *array.Float64:
		vals := a.Float64Values()

		return withNullGuard(a, func(buf *bytes.Buffer, row int) {
			buf.WriteByte(1)
			bufPutUint64(buf, math.Float64bits(vals[row]))
		})
	case *array.Date32:
		vals := a.Date32Values()

		return withNullGuard(a, func(buf *bytes.Buffer, row int) {
			buf.WriteByte(1)
			bufPutUint32(buf, uint32(vals[row]))
		})
	case *array.Date64:
		vals := a.Date64Values()

		return withNullGuard(a, func(buf *bytes.Buffer, row int) {
			buf.WriteByte(1)
			bufPutUint64(buf, uint64(vals[row]))
		})
	case *array.Time32:
		vals := a.Time32Values()

		return withNullGuard(a, func(buf *bytes.Buffer, row int) {
			buf.WriteByte(1)
			bufPutUint32(buf, uint32(vals[row]))
		})
	case *array.Time64:
		vals := a.Time64Values()

		return withNullGuard(a, func(buf *bytes.Buffer, row int) {
			buf.WriteByte(1)
			bufPutUint64(buf, uint64(vals[row]))
		})
	case *array.Timestamp:
		vals := a.TimestampValues()

		return withNullGuard(a, func(buf *bytes.Buffer, row int) {
			buf.WriteByte(1)
			bufPutUint64(buf, uint64(vals[row]))
		})
	case *array.String:
		offsets := a.ValueOffsets()

		rawBytes := a.ValueBytes()

		return withNullGuard(a, func(buf *bytes.Buffer, row int) {
			buf.WriteByte(1)
			start, end := offsets[row], offsets[row+1]
			bufPutUint32(buf, uint32(end-start))
			buf.Write(rawBytes[start:end])
		})
	case *array.LargeString:
		offsets := a.ValueOffsets()
		rawBytes := a.ValueBytes()

		return withNullGuard(a, func(buf *bytes.Buffer, row int) {
			buf.WriteByte(1)
			start, end := offsets[row], offsets[row+1]
			bufPutUint32(buf, uint32(end-start))
			buf.Write(rawBytes[start:end])
		})
	case *array.Binary:
		offsets := a.ValueOffsets()
		rawBytes := a.ValueBytes()

		return withNullGuard(a, func(buf *bytes.Buffer, row int) {
			buf.WriteByte(1)
			start, end := offsets[row], offsets[row+1]
			bufPutUint32(buf, uint32(end-start))
			buf.Write(rawBytes[start:end])
		})
	case *array.LargeBinary:
		offsets := a.ValueOffsets()
		rawBytes := a.ValueBytes()

		return withNullGuard(a, func(buf *bytes.Buffer, row int) {
			buf.WriteByte(1)
			start, end := offsets[row], offsets[row+1]
			bufPutUint32(buf, uint32(end-start))
			buf.Write(rawBytes[start:end])
		})
	case *array.Boolean:
		return withNullGuard(a, func(buf *bytes.Buffer, row int) {
			buf.WriteByte(1)
			if a.Value(row) {
				buf.WriteByte(1)
			} else {
				buf.WriteByte(0)
			}
		})
	case *array.FixedSizeBinary:
		return withNullGuard(a, func(buf *bytes.Buffer, row int) {
			buf.WriteByte(1)
			buf.Write(a.Value(row))
		})
	default:
		return withNullGuard(arr, func(buf *bytes.Buffer, row int) {
			encodeArrowValue(buf, arr, row)
		})
	}
}

// processEqualityDeletesColumnarForFile resolves field paths once per file and
// typed column encoders once per batch, then iterates rows without per-row type
// switches. Each delete set is applied independently because sets may have
// different field IDs.
func processEqualityDeletesColumnarForFile(ctx context.Context, eqDeleteSets []*equalityDeleteSet, fileSchema *iceberg.Schema, dataFilePath string) (recProcessFn, error) {
	requestedFieldIDs := make([]int, 0)
	requestedFieldIDSet := make(map[int]struct{})
	for _, eqDel := range eqDeleteSets {
		if len(eqDel.fieldIDs) != len(eqDel.colNames) {
			return nil, fmt.Errorf("%w: equality delete set has %d field IDs and %d column names",
				iceberg.ErrInvalidArgument, len(eqDel.fieldIDs), len(eqDel.colNames))
		}

		for _, fieldID := range eqDel.fieldIDs {
			if _, ok := requestedFieldIDSet[fieldID]; !ok {
				requestedFieldIDSet[fieldID] = struct{}{}
				requestedFieldIDs = append(requestedFieldIDs, fieldID)
			}
		}
	}

	fieldRefsByID := resolveArrowFieldsByID(fileSchema, requestedFieldIDs)
	fieldRefs := make([][]arrowFieldRef, len(eqDeleteSets))
	for i, eqDel := range eqDeleteSets {
		fieldRefs[i] = make([]arrowFieldRef, len(eqDel.fieldIDs))
		for fieldIdx, fieldID := range eqDel.fieldIDs {
			ref, err := resolveArrowField(fieldRefsByID, fieldID, eqDel.colNames[fieldIdx], dataFilePath)
			if err != nil {
				return nil, err
			}

			fieldRefs[i][fieldIdx] = ref
		}
	}

	return func(r arrow.RecordBatch) (arrow.RecordBatch, error) {
		defer r.Release()

		mem := compute.GetAllocator(ctx)
		numRows := int(r.NumRows())

		var maskBuf *memory.Buffer
		var maskBytes []byte

		var keyBuf bytes.Buffer

		for setIdx, eqDel := range eqDeleteSets {
			encoders := make([]colEncoder, len(eqDel.colNames))
			for i, name := range eqDel.colNames {
				var err error
				encoders[i], err = makeArrowFieldEncoder(r, fieldRefs[setIdx][i], eqDel.fieldIDs[i], name, dataFilePath)
				if err != nil {
					return nil, err
				}
			}

			for row := range numRows {
				if maskBytes != nil && !bitutil.BitIsSet(maskBytes, row) {
					continue
				}

				keyBuf.Reset()

				for _, enc := range encoders {
					enc(&keyBuf, row)
				}

				if _, deleted := eqDel.keys[bufString(&keyBuf)]; !deleted {
					continue
				}

				if maskBuf == nil {
					maskBuf = memory.NewResizableBuffer(mem)
					defer maskBuf.Release()
					maskBuf.Resize(int(bitutil.BytesForBits(int64(numRows))))
					maskBytes = maskBuf.Bytes()

					for i := range maskBytes {
						maskBytes[i] = 0xFF
					}
				}

				bitutil.ClearBit(maskBytes, row)
			}
		}

		if maskBuf == nil {
			r.Retain()

			return r, nil
		}

		maskData := array.NewData(
			arrow.FixedWidthTypes.Boolean, numRows,
			[]*memory.Buffer{nil, maskBuf}, nil, 0, 0)
		mask := array.NewBooleanData(maskData)
		maskData.Release()
		defer mask.Release()

		filtered, err := compute.Filter(ctx,
			compute.NewDatumWithoutOwning(r),
			compute.NewDatumWithoutOwning(mask),
			*compute.DefaultFilterOptions())
		if err != nil {
			return nil, err
		}

		return filtered.(*compute.RecordDatum).Value, nil
	}, nil
}
