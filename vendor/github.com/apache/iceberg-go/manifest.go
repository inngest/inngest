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

package iceberg

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"maps"
	"math"
	"math/big"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/apache/arrow-go/v18/arrow/decimal128"
	"github.com/apache/iceberg-go/internal"
	iceio "github.com/apache/iceberg-go/io"
	"github.com/google/uuid"
	"github.com/twmb/avro"
	"github.com/twmb/avro/atype"
	"github.com/twmb/avro/ocf"
)

// ManifestContent indicates the type of data inside of the files
// described by a manifest. This will indicate whether the data files
// contain active data or deleted rows.
type ManifestContent int32

const (
	ManifestContentData    ManifestContent = 0
	ManifestContentDeletes ManifestContent = 1
)

func (m ManifestContent) String() string {
	switch m {
	case ManifestContentData:
		return "data"
	case ManifestContentDeletes:
		return "deletes"
	default:
		return "UNKNOWN"
	}
}

const initialSequenceNumber = 0

type FieldSummary struct {
	ContainsNull bool    `avro:"contains_null"`
	ContainsNaN  *bool   `avro:"contains_nan"`
	LowerBound   *[]byte `avro:"lower_bound"`
	UpperBound   *[]byte `avro:"upper_bound"`
}

type ManifestBuilder struct {
	m *manifestFile
}

func NewManifestFile(version int, path string, length int64, partitionSpecID int32, addedSnapshotID int64) *ManifestBuilder {
	var seqNum int64
	if version != 1 {
		seqNum = -1
	}

	return &ManifestBuilder{
		m: &manifestFile{
			version:         version,
			Path:            path,
			Len:             length,
			SpecID:          partitionSpecID,
			AddedSnapshotID: addedSnapshotID,
			Content:         ManifestContentData,
			SeqNumber:       seqNum,
			MinSeqNumber:    seqNum,
		},
	}
}

func (b *ManifestBuilder) SequenceNum(num, minSeqNum int64) *ManifestBuilder {
	b.m.SeqNumber, b.m.MinSeqNumber = num, minSeqNum

	return b
}

func (b *ManifestBuilder) Content(content ManifestContent) *ManifestBuilder {
	b.m.Content = content

	return b
}

func (b *ManifestBuilder) AddedFiles(cnt int32) *ManifestBuilder {
	b.m.AddedFilesCount = cnt

	return b
}

func (b *ManifestBuilder) ExistingFiles(cnt int32) *ManifestBuilder {
	b.m.ExistingFilesCount = cnt

	return b
}

func (b *ManifestBuilder) DeletedFiles(cnt int32) *ManifestBuilder {
	b.m.DeletedFilesCount = cnt

	return b
}

func (b *ManifestBuilder) AddedRows(cnt int64) *ManifestBuilder {
	b.m.AddedRowsCount = cnt

	return b
}

func (b *ManifestBuilder) ExistingRows(cnt int64) *ManifestBuilder {
	b.m.ExistingRowsCount = cnt

	return b
}

func (b *ManifestBuilder) DeletedRows(cnt int64) *ManifestBuilder {
	b.m.DeletedRowsCount = cnt

	return b
}

func (b *ManifestBuilder) Partitions(p []FieldSummary) *ManifestBuilder {
	if p == nil {
		// Preserve the semantic "no partition summaries" state by clearing the
		// union pointer entirely instead of storing a pointer to a nil slice.
		b.m.PartitionList = nil

		return b
	}

	copied := cloneFieldSummaries(p)
	b.m.PartitionList = &copied

	return b
}

func cloneFieldSummaries(partitions []FieldSummary) []FieldSummary {
	if partitions == nil {
		return nil
	}

	copied := make([]FieldSummary, len(partitions))
	for i, partition := range partitions {
		copiedPartition := partition
		if partition.LowerBound != nil {
			lowerBound := slices.Clone(*partition.LowerBound)
			copiedPartition.LowerBound = &lowerBound
		}
		if partition.UpperBound != nil {
			upperBound := slices.Clone(*partition.UpperBound)
			copiedPartition.UpperBound = &upperBound
		}
		if partition.ContainsNaN != nil {
			containsNaN := *partition.ContainsNaN
			copiedPartition.ContainsNaN = &containsNaN
		}

		copied[i] = copiedPartition
	}

	return copied
}

func (b *ManifestBuilder) KeyMetadata(km []byte) *ManifestBuilder {
	if km == nil {
		// Preserve the semantic "no key metadata" state by clearing the union
		// pointer entirely instead of storing a pointer to a nil slice.
		b.m.Key = nil

		return b
	}

	keyMetadata := slices.Clone(km)
	b.m.Key = &keyMetadata

	return b
}

func (b *ManifestBuilder) Build() ManifestFile {
	return b.m
}

// manifestFileV1 is a read/write struct for V1 manifest list entries.
// Pointer fields handle nullable Avro types (["null", T] unions).
// The three count fields carry aliases for pre-1.4 Java Iceberg legacy names
// (apache/iceberg#5338) so that avro.Resolve maps them transparently.
type manifestFileV1 struct {
	Path               string          `avro:"manifest_path"`
	Len                int64           `avro:"manifest_length"`
	SpecID             int32           `avro:"partition_spec_id"`
	AddedSnapshotID    *int64          `avro:"added_snapshot_id"`
	AddedFilesCount    *int32          `avro:"added_files_count,alias=added_data_files_count"`
	ExistingFilesCount *int32          `avro:"existing_files_count,alias=existing_data_files_count"`
	DeletedFilesCount  *int32          `avro:"deleted_files_count,alias=deleted_data_files_count"`
	AddedRowsCount     *int64          `avro:"added_rows_count"`
	ExistingRowsCount  *int64          `avro:"existing_rows_count"`
	DeletedRowsCount   *int64          `avro:"deleted_rows_count"`
	PartitionList      *[]FieldSummary `avro:"partitions,type-alias=r508"`
	Key                *[]byte         `avro:"key_metadata"`
}

func (m *manifestFileV1) toFile() *manifestFile {
	snapshotID := int64(-1)
	if m.AddedSnapshotID != nil {
		snapshotID = *m.AddedSnapshotID
	}

	f := &manifestFile{
		version:         1,
		Path:            m.Path,
		Len:             m.Len,
		SpecID:          m.SpecID,
		Content:         ManifestContentData,
		SeqNumber:       initialSequenceNumber,
		MinSeqNumber:    initialSequenceNumber,
		AddedSnapshotID: snapshotID,
		PartitionList:   m.PartitionList,
		Key:             m.Key,
	}

	if m.AddedFilesCount != nil {
		f.AddedFilesCount = *m.AddedFilesCount
	} else {
		f.AddedFilesCount = -1
	}

	if m.ExistingFilesCount != nil {
		f.ExistingFilesCount = *m.ExistingFilesCount
	} else {
		f.ExistingFilesCount = -1
	}

	if m.DeletedFilesCount != nil {
		f.DeletedFilesCount = *m.DeletedFilesCount
	} else {
		f.DeletedFilesCount = -1
	}

	if m.AddedRowsCount != nil {
		f.AddedRowsCount = *m.AddedRowsCount
	} else {
		f.AddedRowsCount = -1
	}

	if m.ExistingRowsCount != nil {
		f.ExistingRowsCount = *m.ExistingRowsCount
	} else {
		f.ExistingRowsCount = -1
	}

	if m.DeletedRowsCount != nil {
		f.DeletedRowsCount = *m.DeletedRowsCount
	} else {
		f.DeletedRowsCount = -1
	}

	return f
}

type manifestFile struct {
	Path               string          `avro:"manifest_path"`
	Len                int64           `avro:"manifest_length"`
	SpecID             int32           `avro:"partition_spec_id"`
	Content            ManifestContent `avro:"content"`
	SeqNumber          int64           `avro:"sequence_number"`
	MinSeqNumber       int64           `avro:"min_sequence_number"`
	AddedSnapshotID    int64           `avro:"added_snapshot_id"`
	AddedFilesCount    int32           `avro:"added_files_count,alias=added_data_files_count"`
	ExistingFilesCount int32           `avro:"existing_files_count,alias=existing_data_files_count"`
	DeletedFilesCount  int32           `avro:"deleted_files_count,alias=deleted_data_files_count"`
	AddedRowsCount     int64           `avro:"added_rows_count"`
	ExistingRowsCount  int64           `avro:"existing_rows_count"`
	DeletedRowsCount   int64           `avro:"deleted_rows_count"`
	PartitionList      *[]FieldSummary `avro:"partitions,type-alias=r508"`
	Key                *[]byte         `avro:"key_metadata"`
	FirstRowIDValue    *int64          `avro:"first_row_id"`

	version int `avro:"-"`
}

func (m *manifestFile) setVersion(v int) {
	m.version = v
}

func (m *manifestFile) toV1(v1file *manifestFileV1) {
	v1file.Path = m.Path
	v1file.Len = m.Len
	v1file.SpecID = m.SpecID
	v1file.AddedSnapshotID = &m.AddedSnapshotID
	v1file.PartitionList = m.PartitionList
	v1file.Key = m.Key

	if m.AddedFilesCount >= 0 {
		v1file.AddedFilesCount = &m.AddedFilesCount
	} else {
		v1file.AddedFilesCount = nil
	}

	if m.ExistingFilesCount >= 0 {
		v1file.ExistingFilesCount = &m.ExistingFilesCount
	} else {
		v1file.ExistingFilesCount = nil
	}

	if m.DeletedFilesCount >= 0 {
		v1file.DeletedFilesCount = &m.DeletedFilesCount
	} else {
		v1file.DeletedFilesCount = nil
	}

	if m.AddedRowsCount >= 0 {
		v1file.AddedRowsCount = &m.AddedRowsCount
	} else {
		v1file.AddedRowsCount = nil
	}

	if m.ExistingRowsCount >= 0 {
		v1file.ExistingRowsCount = &m.ExistingRowsCount
	} else {
		v1file.ExistingRowsCount = nil
	}

	if m.DeletedRowsCount >= 0 {
		v1file.DeletedRowsCount = &m.DeletedRowsCount
	} else {
		v1file.DeletedRowsCount = nil
	}
}

// ensurePartitionList normalizes the narrow case behind #1309: an inherited
// manifest whose partition-field summary list is present but empty. Only an
// empty-spec manifest has zero summaries, so in practice this is an
// unpartitioned table. On decode that present-empty array becomes a non-nil
// pointer over a nil slice, which the manifest-list writer would otherwise
// re-encode as Avro null for the optional partitions union (field 507) — a
// value strict readers such as Redshift Spectrum reject. Only that
// present-but-nil slice is normalized back to an empty array. A genuinely
// absent value (nil pointer) is left null, which is spec-legal because
// partitions is optional; populated (partitioned) summaries are non-empty and
// pass through untouched.
func ensurePartitionList(partitions **[]FieldSummary) {
	if *partitions == nil {
		return
	}
	if **partitions == nil {
		empty := []FieldSummary{}
		*partitions = &empty
	}
}

func (m *manifestFile) Version() int                     { return m.version }
func (m *manifestFile) FilePath() string                 { return m.Path }
func (m *manifestFile) Length() int64                    { return m.Len }
func (m *manifestFile) PartitionSpecID() int32           { return m.SpecID }
func (m *manifestFile) ManifestContent() ManifestContent { return m.Content }
func (m *manifestFile) SnapshotID() int64                { return m.AddedSnapshotID }
func (m *manifestFile) AddedDataFiles() int32            { return m.AddedFilesCount }
func (m *manifestFile) ExistingDataFiles() int32         { return m.ExistingFilesCount }
func (m *manifestFile) DeletedDataFiles() int32          { return m.DeletedFilesCount }
func (m *manifestFile) AddedRows() int64                 { return m.AddedRowsCount }
func (m *manifestFile) ExistingRows() int64              { return m.ExistingRowsCount }
func (m *manifestFile) DeletedRows() int64               { return m.DeletedRowsCount }
func (m *manifestFile) SequenceNum() int64               { return m.SeqNumber }
func (m *manifestFile) MinSequenceNum() int64            { return m.MinSeqNumber }
func (m *manifestFile) KeyMetadata() []byte {
	if m.Key == nil {
		return nil
	}

	return slices.Clone(*m.Key)
}

func (m *manifestFile) Partitions() []FieldSummary {
	if m.PartitionList == nil {
		return nil
	}

	return cloneFieldSummaries(*m.PartitionList)
}

func (m *manifestFile) FirstRowID() *int64 {
	if m.FirstRowIDValue == nil {
		return nil
	}

	firstRowID := *m.FirstRowIDValue

	return &firstRowID
}

func (m *manifestFile) HasAddedFiles() bool    { return m.AddedFilesCount != 0 }
func (m *manifestFile) HasExistingFiles() bool { return m.ExistingFilesCount != 0 }

func (m *manifestFile) Entries(fs iceio.IO, discardDeleted bool) iter.Seq2[ManifestEntry, error] {
	return manifestEntries(fs, m, discardDeleted, nil)
}

func (m *manifestFile) FetchEntries(fs iceio.IO, discardDeleted bool) (_ []ManifestEntry, err error) {
	f, openErr := fs.Open(m.FilePath())
	if openErr != nil {
		return nil, openErr
	}
	defer internal.CheckedClose(f, &err)

	return ReadManifest(m, f, discardDeleted)
}

func getFieldIDMap(sc *avro.Schema) dataFileFieldMaps {
	root := sc.Root()
	getField := func(node avro.SchemaNode, name string) *avro.SchemaField {
		for i := range node.Fields {
			if node.Fields[i].Name == name {
				return &node.Fields[i]
			}
		}

		return nil
	}

	result := make(map[string]int)
	logicalTypes := make(map[int]string)
	fixedSizes := make(map[int]int)
	decimalScales := make(map[int]int)
	unknownFieldIDs := make(map[int]struct{})

	entryField := getField(*root, "data_file")
	partitionField := getField(entryField.Type, "partition")

	for _, field := range partitionField.Type.Fields {
		fid, ok := schemaFieldID(field)
		if !ok {
			continue
		}

		result[field.Name] = fid
		typ := field.Type
		if typ.Type == atype.Union {
			typ = typ.Branches[len(typ.Branches)-1]
		}
		if typ.Type == atype.Null {
			unknownFieldIDs[fid] = struct{}{}
		}
		if typ.LogicalType != "" {
			logicalTypes[fid] = typ.LogicalType
			if typ.LogicalType == atype.Decimal {
				fixedSizes[fid] = typ.Size
				decimalScales[fid] = typ.Scale
			}
		}
	}

	return dataFileFieldMaps{
		nameToID:         result,
		idToType:         logicalTypes,
		idToFixedSize:    fixedSizes,
		idToDecimalScale: decimalScales,
		unknownFieldIDs:  unknownFieldIDs,
	}
}

// applyDayTransformDates marks every day(...) partition field described by the
// manifest's partition-spec metadata as a "date" logical type. This normalizes
// day-transform partition values to iceberg.Date even when they were written as
// a plain Avro int with no logical type, so callers never observe the Avro
// encoding difference. Only day transforms are affected; other plain integer
// partition values (including hour/month/year transforms) are left untouched.
//
// The Iceberg spec gives the day transform result type "date" and requires
// readers to also accept the plain-int encoding, interpreting each integer as
// the number of days since 1970-01-01 (see the transform table and note [1] in
// apache/iceberg#16446). Accepting both encodings is therefore spec-mandated,
// not merely cross-engine convention.
//
// Absent or malformed partition-spec metadata is ignored: the reader simply
// falls back to the logical types declared in the Avro schema.
func applyDayTransformDates(specJSON []byte, fieldIDToType map[int]string) {
	if len(specJSON) == 0 {
		return
	}

	var fields []PartitionField
	if err := json.Unmarshal(specJSON, &fields); err != nil {
		return
	}

	for _, f := range fields {
		if _, ok := f.Transform.(DayTransform); ok {
			fieldIDToType[f.FieldID] = atype.Date
		}
	}
}

type hasFieldToIDMap interface {
	setFieldNameToIDMap(map[string]int)
	setFieldIDToLogicalTypeMap(map[int]string)
	setFieldIDToDecimalScaleMap(map[int]int)
}

// ManifestFile is the interface for version 1, 2, and 3 manifest files.
type ManifestFile interface {
	// Version returns the version number of this manifest file.
	// It must be 1, 2, or 3.
	Version() int
	// FilePath is the location URI of this manifest file.
	FilePath() string
	// Length is the length in bytes of the manifest file.
	Length() int64
	// PartitionSpecID is the ID of the partition spec used to write
	// this manifest. It must be listed in the table metadata
	// partition-specs.
	PartitionSpecID() int32
	// ManifestContent is the type of files tracked by this manifest,
	// either data or delete files. All v1 manifests track data files.
	ManifestContent() ManifestContent
	// SnapshotID is the ID of the snapshot where this manifest file
	// was added.
	SnapshotID() int64
	// AddedDataFiles returns the number of entries in the manifest that
	// have the status of EntryStatusADDED.
	AddedDataFiles() int32
	// ExistingDataFiles returns the number of entries in the manifest
	// which have the status of EntryStatusEXISTING.
	ExistingDataFiles() int32
	// DeletedDataFiles returns the number of entries in the manifest
	// which have the status of EntryStatusDELETED.
	DeletedDataFiles() int32
	// AddedRows returns the number of rows in all files of the manifest
	// that have status EntryStatusADDED.
	AddedRows() int64
	// ExistingRows returns the number of rows in all files of the manifest
	// which have status EntryStatusEXISTING.
	ExistingRows() int64
	// DeletedRows returns the number of rows in all files of the manifest
	// which have status EntryStatusDELETED.
	DeletedRows() int64
	// SequenceNum returns the sequence number when this manifest was
	// added to the table. Will be 0 for v1 manifest lists.
	SequenceNum() int64
	// MinSequenceNum is the minimum data sequence number of all live data
	// or delete files in the manifest. Will be 0 for v1 manifest lists.
	MinSequenceNum() int64
	// KeyMetadata returns implementation-specific key metadata for encryption
	// if it exists in the manifest list.
	KeyMetadata() []byte
	// Partitions returns a list of field summaries for each partition
	// field in the spec. Each field in the list corresponds to a field in
	// the manifest file's partition spec.
	Partitions() []FieldSummary
	// FirstRowID returns the first _row_id assigned to rows in this manifest (v3+ data manifests only).
	// Returns nil for v1/v2 or for delete manifests.
	FirstRowID() *int64

	// HasAddedFiles returns true if AddedDataFiles > 0 or if it was null.
	HasAddedFiles() bool
	// HasExistingFiles returns true if ExistingDataFiles > 0 or if it was null.
	HasExistingFiles() bool
	// Entries streams the manifest entries from the manifest file using
	// the provided file system IO interface. Entries that have been
	// marked as deleted are skipped if discardDeleted is true.
	//
	// Prefer Entries over FetchEntries when walking large manifests
	// since it avoids loading every entry into memory at once.
	//
	// Iteration contract:
	//
	//   - On the first error encountered while opening the manifest, decoding
	//     a record, or applying inheritance, the iterator yields (nil, err)
	//     and then stops. Callers must treat any non-nil error as terminal and
	//     break or return without consuming further values.
	//   - When iteration ends without an error from the read path, the
	//     iterator may yield a final (nil, closeErr) pair if closing the
	//     underlying file or manifest reader returns an error. This terminal
	//     close error is reported only when the consumer ranged through every
	//     value; an early break suppresses it (see below).
	//   - Breaking out of the range loop (or any other early termination of
	//     the yield function) is safe: the iterator releases the underlying
	//     file handle and reader before returning, and no close error from
	//     that path is yielded — the caller has already signalled it is no
	//     longer interested in further values, so an extra synthetic
	//     (nil, closeErr) tail would be discarded anyway.
	Entries(fs iceio.IO, discardDeleted bool) iter.Seq2[ManifestEntry, error]
	// FetchEntries reads the manifest list file to fetch the list of
	// manifest entries using the provided file system IO interface.
	// If discardDeleted is true, entries for files containing deleted rows
	// will be skipped.
	//
	// Deprecated: Use Entries instead, which streams manifest entries via an
	// iterator and avoids loading every entry into memory at once.
	FetchEntries(fs iceio.IO, discardDeleted bool) ([]ManifestEntry, error)
	// // WriteEntries writes a list of manifest entries to a provided
	// // io.Writer. The version of the manifest file is used to determine the
	// // schema to use for writing the entries.
	// WriteEntries(out io.Writer, entries []ManifestEntry) error

	setVersion(int)
}

// manifestFileV1Reader is the Avro reader schema for V1 manifest list entries.
// It handles both spec-correct field names and pre-1.4 Java Iceberg legacy
// names (added_data_files_count etc.) via the alias tags on manifestFileV1,
// and both nullable and non-nullable added_snapshot_id via *int64.
var manifestFileV1Reader = avro.MustSchemaFor[manifestFileV1](
	avro.WithName("manifest_file"),
)

// manifestFileReader is the Avro reader schema for V2+ manifest list entries.
// Alias tags on manifestFile handle pre-1.4 Java Iceberg legacy field names.
var manifestFileReader = avro.MustSchemaFor[manifestFile](
	avro.WithName("manifest_file"),
)

func decodeV1Manifests(rd *ocf.Reader) ([]ManifestFile, error) {
	results := make([]ManifestFile, 0)
	for {
		tmp := new(manifestFileV1)
		if err := rd.Decode(tmp); err != nil {
			if errors.Is(err, io.EOF) {
				return results, nil
			}

			return nil, err
		}

		results = append(results, tmp.toFile())
	}
}

func decodeManifests[I interface {
	ManifestFile
	*T
}, T any](rd *ocf.Reader, version int) ([]ManifestFile, error) {
	results := make([]ManifestFile, 0)
	for {
		tmp := I(new(T))
		if err := rd.Decode(tmp); err != nil {
			if errors.Is(err, io.EOF) {
				return results, nil
			}

			return nil, err
		}

		tmp.setVersion(version)
		results = append(results, tmp)
	}
}

// ManifestReader reads the metadata and data from an avro manifest file.
// This type is not thread-safe; its methods should not be called from
// multiple goroutines.
type ManifestReader struct {
	rd             *ocf.Reader
	file           ManifestFile
	formatVersion  int
	isFallback     bool
	projected      bool
	content        ManifestContent
	fieldNameToID  map[string]int
	fieldIDToType  map[int]string
	fieldIDToScale map[int]int

	// The rest are lazily populated, on demand. Most readers
	// will likely only try to load the entries.
	schema              Schema
	schemaLoaded        bool
	partitionSpec       PartitionSpec
	partitionSpecLoaded bool

	// inheritRowIDs controls whether this reader should apply First Row ID inheritance
	// for v3 data manifests (spec: First Row ID Inheritance).
	inheritRowIDs bool
	// nextFirstRowID tracks the next first_row_id to assign when reading v3 data
	// manifests; used for First Row ID inheritance (null data file first_row_id
	// gets manifest's first_row_id + sum of preceding files' record_count).
	nextFirstRowID int64
}

// NewManifestReader returns a value that can read the contents of an avro manifest
// file. If the caller is interested in the manifest entries in the file, it must call
// [ManifestReader.Entries] before closing the provided reader.
func NewManifestReader(file ManifestFile, in io.Reader) (*ManifestReader, error) {
	return newManifestReader(file, in, nil)
}

// NewManifestReaderWithProjection returns a manifest reader that decodes the
// standard scan fields and optionally the column statistics selected by
// projection. Manifest metadata validation and entry inheritance are the same
// as in NewManifestReader; fields omitted by the projection retain their zero
// values in the returned DataFile. Entries returned by this reader are
// read-only planning values and must not be passed to a ManifestWriter, because
// omitted statistics cannot be recovered on rewrite. ManifestWriter and the
// DataFile Avro codec reject these values with ErrInvalidArgument.
func NewManifestReaderWithProjection(
	file ManifestFile,
	in io.Reader,
	projection ManifestEntryProjection,
) (*ManifestReader, error) {
	return newManifestReader(file, in, &projection)
}

func newManifestReader(
	file ManifestFile,
	in io.Reader,
	projection *ManifestEntryProjection,
) (*ManifestReader, error) {
	var writerSchema *avro.Schema
	var rd *ocf.Reader
	var err error
	if projection == nil {
		rd, err = ocf.NewReader(in)
	} else {
		rd, err = ocf.NewReader(in, ocf.WithReaderSchemaFunc(func(reader *ocf.Reader) (*avro.Schema, error) {
			writerSchema = reader.Schema()

			return projectedManifestEntrySchema(writerSchema, *projection)
		}))
	}
	if err != nil {
		return nil, err
	}

	metadata := rd.Metadata()
	if writerSchema == nil {
		writerSchema = rd.Schema()
	}
	sc := writerSchema

	formatVersion := 1
	// format-version is optional for v1 manifest files, so default to v1.
	if raw := metadata["format-version"]; len(raw) > 0 {
		formatVersion, err = strconv.Atoi(string(raw))
		if err != nil {
			return nil, fmt.Errorf("manifest file's 'format-version' metadata is invalid: %w", err)
		}
	}
	if err := validateManifestFormatVersion(formatVersion); err != nil {
		return nil, err
	}
	// The manifest's own metadata is authoritative for its version. A v2/v3
	// manifest list may reference older manifests so a table can be upgraded
	// in place without rewriting history; only a manifest newer than the list
	// referencing it indicates corruption.
	if formatVersion > file.Version() {
		return nil, fmt.Errorf("manifest file's 'format-version' metadata indicates version %d, which is newer than the v%d manifest list referencing it",
			formatVersion, file.Version())
	}

	var content ManifestContent
	switch contentStr := string(metadata["content"]); contentStr {
	case "":
		if formatVersion != 1 {
			return nil, fmt.Errorf("manifest file's 'content' metadata is missing and 'format-version' is %d, but the field is required for 'format-version' 2 and beyond",
				formatVersion)
		}
		// V1 manifests do not contain the 'content' field, but should be interpretted as 'data' files
		fallthrough
	case "data":
		content = ManifestContentData
	case "deletes":
		content = ManifestContentDeletes
	default:
		return nil, fmt.Errorf("manifest file's 'content' metadata is invalid, should be \"data\" or \"deletes\" but instead is %q",
			contentStr)
	}
	if content != file.ManifestContent() {
		return nil, fmt.Errorf("manifest file's 'content' metadata indicates %q, but entry from manifest list indicates %q",
			content.String(), file.ManifestContent().String())
	}

	isFallback := false
	if formatVersion == 1 {
		root := sc.Root()
		for _, f := range root.Fields {
			if f.Name == "snapshot_id" {
				if f.Type.Type != atype.Union {
					isFallback = true
				}

				break
			}
		}
	}
	fieldMaps := getFieldIDMap(sc)
	// day(...) partition values are days since the Unix epoch, but a manifest
	// may encode them either as an Avro int carrying the "date" logical type or
	// as a legacy plain Avro int. Overlay the manifest's partition spec so
	// day-transform fields are always exposed as iceberg.Date, regardless of the
	// Avro encoding used to write them.
	applyDayTransformDates(metadata["partition-spec"], fieldMaps.idToType)

	// A non-nil manifest-list first_row_id is the spec's inheritance signal and
	// is only assigned by a v3+ manifest-list writer, so it — not the manifest
	// file's own internal format-version — gates inheritance. A v1- or v2-era
	// manifest carried into an upgraded v3 table keeps its older format-version
	// internally but is assigned a first_row_id when written into the v3 list; its
	// data files must still inherit row IDs (spec: First Row ID Inheritance, "even
	// if the data file is existing"), otherwise compaction drops their lineage.
	inheritRowIDs := content == ManifestContentData &&
		file.FirstRowID() != nil
	var nextFirstRowID int64
	if inheritRowIDs {
		nextFirstRowID = *file.FirstRowID()
		if nextFirstRowID < 0 {
			return nil, fmt.Errorf("%w: first row ID must be non-negative: %d",
				ErrInvalidArgument, nextFirstRowID)
		}
	}

	return &ManifestReader{
		rd:             rd,
		file:           file,
		formatVersion:  formatVersion,
		isFallback:     isFallback,
		projected:      projection != nil,
		content:        content,
		fieldNameToID:  fieldMaps.nameToID,
		fieldIDToType:  fieldMaps.idToType,
		fieldIDToScale: fieldMaps.idToDecimalScale,
		inheritRowIDs:  inheritRowIDs,
		nextFirstRowID: nextFirstRowID,
	}, nil
}

// Close releases decoder resources associated with this manifest reader.
func (c *ManifestReader) Close() error {
	return c.rd.Close()
}

// Version returns the file's format version.
func (c *ManifestReader) Version() int {
	return c.formatVersion
}

// ManifestContent returns the type of content in the manifest file.
func (c *ManifestReader) ManifestContent() ManifestContent {
	return c.content
}

// SchemaID returns the schema ID encoded in the avro file's metadata.
func (c *ManifestReader) SchemaID() (int, error) {
	id, err := strconv.Atoi(string(c.rd.Metadata()["schema-id"]))
	if err != nil {
		return 0, fmt.Errorf("manifest file's 'schema-id' metadata is invalid: %w", err)
	}

	return id, nil
}

// Schema returns the schema encoded in the avro file's metadata.
func (c *ManifestReader) Schema() (*Schema, error) {
	if !c.schemaLoaded {
		schemaID, err := c.SchemaID()
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(c.rd.Metadata()["schema"], &c.schema); err != nil {
			return nil, fmt.Errorf("manifest file's 'schema' metadata is invalid: %w", err)
		}
		c.schema.ID = schemaID
		c.schemaLoaded = true
	}

	return &c.schema, nil
}

// PartitionSpecID returns the partition spec ID encoded in the avro file's metadata.
func (c *ManifestReader) PartitionSpecID() (int, error) {
	id, err := strconv.Atoi(string(c.rd.Metadata()["partition-spec-id"]))
	if err != nil {
		return 0, fmt.Errorf("manifest file's 'partition-spec-id' metadata is invalid: %w", err)
	}
	if id != int(c.file.PartitionSpecID()) {
		return 0, fmt.Errorf("manifest file's 'partition-spec-id' metadata indicates %d, but entry from manifest list indicates %d",
			id, c.file.PartitionSpecID())
	}

	return id, nil
}

// PartitionSpec returns the partition spec encoded in the avro file's metadata.
func (c *ManifestReader) PartitionSpec() (*PartitionSpec, error) {
	if !c.partitionSpecLoaded {
		partitionSpecID, err := c.PartitionSpecID()
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(c.rd.Metadata()["partition-spec"], &c.partitionSpec.fields); err != nil {
			return nil, fmt.Errorf("manifest file's 'partition-spec' metadata is invalid: %w", err)
		}
		c.partitionSpec.id = partitionSpecID
		c.partitionSpec.initialize()
		c.partitionSpecLoaded = true
	}

	return &c.partitionSpec, nil
}

// ReadEntry reads the next manifest entry in the avro file's data.
func (c *ManifestReader) ReadEntry() (ManifestEntry, error) {
	var tmp ManifestEntry
	if c.isFallback {
		tmp = &fallbackManifestEntry{
			manifestEntry: manifestEntry{Data: &dataFile{}},
		}
	} else {
		tmp = &manifestEntry{Data: &dataFile{}}
	}

	if err := c.rd.Decode(tmp); err != nil {
		return nil, err
	}
	if c.isFallback {
		tmp = tmp.(*fallbackManifestEntry).toEntry()
	}
	tmp.DataFile().(*dataFile).projected = c.projected
	switch tmp.Status() {
	case EntryStatusEXISTING, EntryStatusADDED, EntryStatusDELETED:
	default:
		return nil, fmt.Errorf("manifest entry has invalid status: %d", tmp.Status())
	}

	entryContent := tmp.DataFile().ContentType()
	switch entryContent {
	case EntryContentData:
		if c.content != ManifestContentData {
			return nil, errors.New("data file found in delete manifest")
		}
	case EntryContentPosDeletes, EntryContentEqDeletes:
		if c.content != ManifestContentDeletes {
			return nil, errors.New("delete file found in data manifest")
		}
	default:
		return nil, fmt.Errorf("manifest entry has invalid content: %d", entryContent)
	}
	tmp.inherit(c.file)
	// First Row ID Inheritance (spec): assign the manifest's first_row_id to data
	// files that lack one, advancing by record_count only on the files actually
	// assigned. Deleted entries consume no row IDs and are skipped — the
	// manifest-list writer reserves a manifest's id range as added+existing rows
	// (excludes deleted), so the read side must match or a live file following a
	// deleted one over-advances into the next manifest's range. Mirrors Java
	// ManifestReader.idAssigner (status != DELETED, increment inside the null check).
	if c.inheritRowIDs && tmp.Status() != EntryStatusDELETED {
		if df, ok := tmp.DataFile().(*dataFile); ok && df.FirstRowIDField == nil {
			id := c.nextFirstRowID
			count := df.Count()
			if count < 0 {
				return nil, fmt.Errorf("cannot inherit first row ID for data file %q: %w: record count must be non-negative: %d",
					df.FilePath(), ErrInvalidArgument, count)
			}
			nextFirstRowID, err := advanceRowID(id, 0, count)
			if err != nil {
				return nil, fmt.Errorf("cannot inherit first row ID for data file %q: %w", df.FilePath(), err)
			}
			df.FirstRowIDField = &id
			c.nextFirstRowID = nextFirstRowID
		}
	}
	if fieldToIDMap, ok := tmp.DataFile().(hasFieldToIDMap); ok {
		fieldToIDMap.setFieldNameToIDMap(c.fieldNameToID)
		fieldToIDMap.setFieldIDToLogicalTypeMap(c.fieldIDToType)
		fieldToIDMap.setFieldIDToDecimalScaleMap(c.fieldIDToScale)
	}

	return tmp, nil
}

// iterManifest returns an iterator that streams manifest entries from
// the provided reader without buffering them. If discardDeleted is true,
// entries whose status is "deleted" are skipped.
func iterManifest(
	m ManifestFile,
	f io.Reader,
	discardDeleted bool,
	projection *ManifestEntryProjection,
) iter.Seq2[ManifestEntry, error] {
	return func(yield func(ManifestEntry, error) bool) {
		var manifestReader *ManifestReader
		var err error
		if projection == nil {
			manifestReader, err = NewManifestReader(m, f)
		} else {
			manifestReader, err = NewManifestReaderWithProjection(m, f, *projection)
		}
		if err != nil {
			yield(nil, err)

			return
		}
		aborted := false
		defer func() {
			if cerr := manifestReader.Close(); cerr != nil && !aborted {
				yield(nil, cerr)
			}
		}()

		for {
			entry, err := manifestReader.ReadEntry()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				if !yield(nil, err) {
					aborted = true
				}

				return
			}
			if discardDeleted && entry.Status() == EntryStatusDELETED {
				continue
			}
			if !yield(entry, nil) {
				aborted = true

				return
			}
		}
	}
}

// ReadManifest reads in an avro list file and returns a slice
// of manifest entries or an error if one is encountered. If discardDeleted
// is true, the returned slice omits entries whose status is "deleted".
func ReadManifest(m ManifestFile, f io.Reader, discardDeleted bool) ([]ManifestEntry, error) {
	var results []ManifestEntry
	for entry, err := range iterManifest(m, f, discardDeleted, nil) {
		if err != nil {
			return results, err
		}
		results = append(results, entry)
	}

	return results, nil
}

// Manifest-list entry field IDs used to infer the format version from an
// embedded writer schema. Per the Iceberg spec, content and sequence_number
// are required fields for v2+ manifest lists and first_row_id is a v3 field;
// none of them exist in v1.
const (
	fieldIDManifestSequenceNumber = 515
	fieldIDManifestContent        = 517
	fieldIDManifestFirstRowID     = 520
)

// schemaFieldID returns the Iceberg "field-id" property of an Avro schema
// field, if present. Schemas parsed from JSON (e.g. an OCF header's embedded
// writer schema) carry numbers as float64, while schemas built in memory
// carry int.
func schemaFieldID(f avro.SchemaField) (int, bool) {
	switch v := f.Props["field-id"].(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	}

	return 0, false
}

// inferManifestListVersion infers the format version of a manifest list from
// the fields of its embedded writer schema: first_row_id (field-id 520)
// implies v3, content (517) or sequence_number (515) imply v2, and a schema
// with none of them is v1. Fields are identified by their field-id property —
// the stable contract per the spec — with the spec field names as a fallback
// for writers that don't annotate IDs.
func inferManifestListVersion(sc *avro.Schema) int {
	version := 1
	for _, f := range sc.Root().Fields {
		id, ok := schemaFieldID(f)
		switch {
		case ok && id == fieldIDManifestFirstRowID,
			!ok && f.Name == "first_row_id":
			return 3
		case ok && (id == fieldIDManifestContent || id == fieldIDManifestSequenceNumber),
			!ok && (f.Name == "content" || f.Name == "sequence_number"):
			version = 2
		}
	}

	return version
}

// ReadManifestList reads in an avro manifest list file and returns a slice
// of manifest files or an error if one is encountered.
//
// Per the Iceberg spec, manifest list files are not required to carry a
// "format-version" metadata key (only manifest files are). When the key is
// absent, the version is inferred from the embedded writer schema, so lists
// from writers that omit the key (e.g. DuckDB's iceberg extension) decode
// with their real content types and sequence numbers instead of falling back
// to v1. When the key is present but claims a lower version than the schema
// carries fields for, an error is returned rather than silently dropping
// those fields on decode (or on a later rewrite through the older writer
// schema).
func ReadManifestList(in io.Reader) ([]ManifestFile, error) {
	var version int

	rd, err := ocf.NewReader(in, ocf.WithReaderSchemaFunc(func(rd *ocf.Reader) (*avro.Schema, error) {
		inferred := inferManifestListVersion(rd.Schema())

		version = inferred
		if raw := rd.Metadata()["format-version"]; len(raw) > 0 {
			v, err := strconv.Atoi(string(raw))
			if err != nil {
				return nil, fmt.Errorf("invalid format-version: %w", err)
			}

			version = v
		}
		if err := validateManifestFormatVersion(version); err != nil {
			return nil, err
		}

		if version < inferred {
			return nil, fmt.Errorf("manifest list's 'format-version' metadata says %d, but the embedded writer schema carries v%d fields; reading it as v%d would silently drop them", version, inferred, version)
		}

		if version == 1 {
			return manifestFileV1Reader, nil
		}

		return manifestFileReader, nil
	}))
	if err != nil {
		return nil, err
	}

	if version == 1 {
		return decodeV1Manifests(rd)
	}

	return decodeManifests[*manifestFile](rd, version)
}

func validateManifestFormatVersion(version int) error {
	if version < 1 || version > 3 {
		return fmt.Errorf("unsupported manifest format version: %d", version)
	}

	return nil
}

type writerImpl interface {
	prepareEntry(*manifestEntry, int64) (ManifestEntry, error)
}

type v1writerImpl struct{}

func (v1writerImpl) prepareEntry(entry *manifestEntry, sn int64) (ManifestEntry, error) {
	if entry.Snapshot != nil && *entry.Snapshot != sn {
		if entry.EntryStatus != EntryStatusEXISTING {
			return nil, fmt.Errorf("mismatched snapshot id for entry: %d vs %d", *entry.Snapshot, sn)
		}
		sn = *entry.Snapshot
	}

	return &fallbackManifestEntry{
		manifestEntry: *entry,
		Snapshot:      sn,
	}, nil
}

type v2writerImpl struct{}

func (v2writerImpl) prepareEntry(entry *manifestEntry, snapshotID int64) (ManifestEntry, error) {
	if entry.SeqNum == nil {
		if entry.Snapshot != nil && *entry.Snapshot != snapshotID {
			return nil, fmt.Errorf("found unassigned sequence number for entry from snapshot: %d", entry.SnapshotID())
		}

		if entry.EntryStatus != EntryStatusADDED {
			return nil, errors.New("only entries with status ADDED can be missing a sequence number")
		}
	}

	return entry, nil
}

type v3writerImpl struct{}

func (v3writerImpl) prepareEntry(entry *manifestEntry, snapshotID int64) (ManifestEntry, error) {
	if entry.SeqNum == nil {
		if entry.Snapshot != nil && *entry.Snapshot != snapshotID {
			return nil, fmt.Errorf("found unassigned sequence number for entry from snapshot: %d", entry.SnapshotID())
		}
		if entry.EntryStatus != EntryStatusADDED {
			return nil, errors.New("only entries with status ADDED can be missing a sequence number")
		}
	}

	return entry, nil
}

type fieldStats interface {
	toSummary() FieldSummary
	update(value any) error
}

type partitionFieldStats[T LiteralType] struct {
	containsNull bool
	containsNan  bool
	min          *T
	max          *T

	cmp     Comparator[T]
	convert partitionValueConverter[T]
}

func clonePartitionStatValue[T LiteralType](value T) T {
	if data, ok := any(value).([]byte); ok {
		return any(slices.Clone(data)).(T)
	}

	return value
}

type partitionValueConverter[T LiteralType] func(any) (T, bool)

func partitionLiteralValue(value any) (any, bool) {
	switch value.(type) {
	case AboveMaxLiteral, BelowMinLiteral:
		return nil, false
	}

	literal, ok := value.(Literal)
	if !ok {
		return nil, false
	}

	return literal.Any(), true
}

// unknownPartitionFieldStats records null presence but omits bounds because
// the dropped source type needed to encode and compare non-null values is lost.
type unknownPartitionFieldStats struct {
	containsNull bool
}

func (p *unknownPartitionFieldStats) toSummary() FieldSummary {
	containsNaN := false

	return FieldSummary{ContainsNull: p.containsNull, ContainsNaN: &containsNaN}
}

func (p *unknownPartitionFieldStats) update(any) error {
	// Dropped-source values are normalized to Avro null when the entry is
	// encoded, even if an in-memory data file still carries a historical value.
	p.containsNull = true

	return nil
}

func makePartitionFieldStats[T LiteralType](convert partitionValueConverter[T]) fieldStats {
	return &partitionFieldStats[T]{
		cmp:     getComparator[T](),
		convert: convert,
	}
}

func convertPartitionBool(value any) (bool, bool) {
	switch value := value.(type) {
	case bool:
		return value, true
	default:
		if literal, ok := partitionLiteralValue(value); ok {
			return convertPartitionBool(literal)
		}

		return false, false
	}
}

func convertPartitionInt32(value any) (int32, bool) {
	switch value := value.(type) {
	case int:
		return int32(value), true
	case int8:
		return int32(value), true
	case int16:
		return int32(value), true
	case int32:
		return value, true
	case int64:
		return int32(value), true
	case uint:
		return int32(value), true
	case uint8:
		return int32(value), true
	case uint16:
		return int32(value), true
	case uint32:
		return int32(value), true
	case uint64:
		return int32(value), true
	case uintptr:
		return int32(value), true
	case float32:
		return int32(value), true
	case float64:
		return int32(value), true
	case Date:
		return int32(value), true
	case Time:
		return int32(value), true
	case Timestamp:
		return int32(value), true
	case TimestampNano:
		return int32(value), true
	case time.Duration:
		return int32(value), true
	default:
		if literal, ok := partitionLiteralValue(value); ok {
			return convertPartitionInt32(literal)
		}

		return 0, false
	}
}

func convertPartitionInt64(value any) (int64, bool) {
	switch value := value.(type) {
	case int:
		return int64(value), true
	case int8:
		return int64(value), true
	case int16:
		return int64(value), true
	case int32:
		return int64(value), true
	case int64:
		return value, true
	case uint:
		return int64(value), true
	case uint8:
		return int64(value), true
	case uint16:
		return int64(value), true
	case uint32:
		return int64(value), true
	case uint64:
		return int64(value), true
	case uintptr:
		return int64(value), true
	case float32:
		return int64(value), true
	case float64:
		return int64(value), true
	case Date:
		return int64(value), true
	case Time:
		return int64(value), true
	case Timestamp:
		return int64(value), true
	case TimestampNano:
		return int64(value), true
	case time.Duration:
		return int64(value), true
	default:
		if literal, ok := partitionLiteralValue(value); ok {
			return convertPartitionInt64(literal)
		}

		return 0, false
	}
}

func convertPartitionFloat32(value any) (float32, bool) {
	switch value := value.(type) {
	case int:
		return float32(value), true
	case int8:
		return float32(value), true
	case int16:
		return float32(value), true
	case int32:
		return float32(value), true
	case int64:
		return float32(value), true
	case uint:
		return float32(value), true
	case uint8:
		return float32(value), true
	case uint16:
		return float32(value), true
	case uint32:
		return float32(value), true
	case uint64:
		return float32(value), true
	case uintptr:
		return float32(value), true
	case float32:
		return value, true
	case float64:
		return float32(value), true
	case Date:
		return float32(value), true
	case Time:
		return float32(value), true
	case Timestamp:
		return float32(value), true
	case TimestampNano:
		return float32(value), true
	case time.Duration:
		return float32(value), true
	default:
		if literal, ok := partitionLiteralValue(value); ok {
			return convertPartitionFloat32(literal)
		}

		return 0, false
	}
}

func convertPartitionFloat64(value any) (float64, bool) {
	switch value := value.(type) {
	case int:
		return float64(value), true
	case int8:
		return float64(value), true
	case int16:
		return float64(value), true
	case int32:
		return float64(value), true
	case int64:
		return float64(value), true
	case uint:
		return float64(value), true
	case uint8:
		return float64(value), true
	case uint16:
		return float64(value), true
	case uint32:
		return float64(value), true
	case uint64:
		return float64(value), true
	case uintptr:
		return float64(value), true
	case float32:
		return float64(value), true
	case float64:
		return value, true
	case Date:
		return float64(value), true
	case Time:
		return float64(value), true
	case Timestamp:
		return float64(value), true
	case TimestampNano:
		return float64(value), true
	case time.Duration:
		return float64(value), true
	default:
		if literal, ok := partitionLiteralValue(value); ok {
			return convertPartitionFloat64(literal)
		}

		return 0, false
	}
}

func convertPartitionString(value any) (string, bool) {
	switch value := value.(type) {
	case string:
		return value, true
	case []byte:
		return string(value), true
	case []rune:
		return string(value), true
	default:
		if literal, ok := partitionLiteralValue(value); ok {
			return convertPartitionString(literal)
		}

		return "", false
	}
}

func convertPartitionBytes(value any) ([]byte, bool) {
	switch value := value.(type) {
	case []byte:
		return value, true
	case string:
		return []byte(value), true
	default:
		if literal, ok := partitionLiteralValue(value); ok {
			return convertPartitionBytes(literal)
		}

		return nil, false
	}
}

func convertPartitionUUID(value any) (uuid.UUID, bool) {
	switch value := value.(type) {
	case uuid.UUID:
		return value, true
	case [16]byte:
		return uuid.UUID(value), true
	case []byte:
		if len(value) != len(uuid.UUID{}) {
			return uuid.UUID{}, false
		}

		var converted uuid.UUID
		copy(converted[:], value)

		return converted, true
	default:
		if literal, ok := partitionLiteralValue(value); ok {
			return convertPartitionUUID(literal)
		}

		return uuid.UUID{}, false
	}
}

func convertPartitionDecimal(value any) (Decimal, bool) {
	switch value := value.(type) {
	case Decimal:
		return value, true
	default:
		if literal, ok := partitionLiteralValue(value); ok {
			return convertPartitionDecimal(literal)
		}

		return Decimal{}, false
	}
}

func convertPartitionDate(value any) (Date, bool) {
	converted, ok := convertPartitionInt32(value)

	return Date(converted), ok
}

func convertPartitionTime(value any) (Time, bool) {
	converted, ok := convertPartitionInt64(value)

	return Time(converted), ok
}

func convertPartitionTimestamp(value any) (Timestamp, bool) {
	converted, ok := convertPartitionInt64(value)

	return Timestamp(converted), ok
}

func convertPartitionTimestampNano(value any) (TimestampNano, bool) {
	converted, ok := convertPartitionInt64(value)

	return TimestampNano(converted), ok
}

func newPartitionFieldStat(typ PrimitiveType) (fieldStats, error) {
	switch typ.(type) {
	case UnknownType:
		return &unknownPartitionFieldStats{}, nil
	case BooleanType:
		return makePartitionFieldStats[bool](convertPartitionBool), nil
	case Int32Type:
		return makePartitionFieldStats[int32](convertPartitionInt32), nil
	case Int64Type:
		return makePartitionFieldStats[int64](convertPartitionInt64), nil
	case Float32Type:
		return makePartitionFieldStats[float32](convertPartitionFloat32), nil
	case Float64Type:
		return makePartitionFieldStats[float64](convertPartitionFloat64), nil
	case StringType:
		return makePartitionFieldStats[string](convertPartitionString), nil
	case DateType:
		return makePartitionFieldStats[Date](convertPartitionDate), nil
	case TimeType:
		return makePartitionFieldStats[Time](convertPartitionTime), nil
	case TimestampType:
		return makePartitionFieldStats[Timestamp](convertPartitionTimestamp), nil
	case TimestampTzType:
		return makePartitionFieldStats[Timestamp](convertPartitionTimestamp), nil
	case TimestampNsType, TimestampTzNsType:
		return makePartitionFieldStats[TimestampNano](convertPartitionTimestampNano), nil
	case UUIDType:
		return makePartitionFieldStats[uuid.UUID](convertPartitionUUID), nil
	case BinaryType:
		return makePartitionFieldStats[[]byte](convertPartitionBytes), nil
	case FixedType:
		return makePartitionFieldStats[[]byte](convertPartitionBytes), nil
	case DecimalType:
		return makePartitionFieldStats[Decimal](convertPartitionDecimal), nil
	default:
		return nil, fmt.Errorf("expected primitive type for partition type: %s", typ)
	}
}

func (p *partitionFieldStats[T]) toSummary() FieldSummary {
	var (
		lowerBound *[]byte
		upperBound *[]byte
		lit        Literal
	)

	if p.min != nil {
		lit = NewLiteral(*p.min)
		lb, _ := lit.MarshalBinary()
		lowerBound = &lb
	}

	if p.max != nil {
		lit = NewLiteral(*p.max)
		ub, _ := lit.MarshalBinary()
		upperBound = &ub
	}

	return FieldSummary{
		ContainsNull: p.containsNull,
		ContainsNaN:  &p.containsNan,
		LowerBound:   lowerBound,
		UpperBound:   upperBound,
	}
}

func (p *partitionFieldStats[T]) update(value any) (err error) {
	if value == nil {
		p.containsNull = true

		return err
	}

	actualVal, ok := p.convert(value)
	if !ok {
		return fmt.Errorf("expected type %T, got %T", actualVal, value)
	}

	switch f := any(actualVal).(type) {
	case float32:
		if math.IsNaN(float64(f)) {
			p.containsNan = true

			return err
		}
	case float64:
		if math.IsNaN(f) {
			p.containsNan = true

			return err
		}
	}

	if p.min == nil {
		actualVal = clonePartitionStatValue(actualVal)
		p.min = &actualVal
		p.max = &actualVal
	} else {
		isMin := p.cmp(actualVal, *p.min) < 0
		isMax := p.cmp(actualVal, *p.max) > 0
		if isMin || isMax {
			actualVal = clonePartitionStatValue(actualVal)
		}
		if isMin {
			p.min = &actualVal
		}
		if isMax {
			p.max = &actualVal
		}
	}

	return nil
}

type partitionSummaryStats struct {
	fields []NestedField
	stats  []fieldStats
}

func newPartitionSummaryStats(spec PartitionSpec, schema *Schema) (*partitionSummaryStats, error) {
	fields := spec.PartitionType(schema).FieldList
	stats := make([]fieldStats, len(fields))
	var err error
	for i, field := range fields {
		pt, ok := field.Type.(PrimitiveType)
		if !ok {
			return nil, fmt.Errorf("expected primitive type for partition field, got %s", field.Type)
		}

		stats[i], err = newPartitionFieldStat(pt)
		if err != nil {
			return nil, fmt.Errorf("error constructing field stats for partition %d: %s: %s", i, field.Name, err)
		}
	}

	return &partitionSummaryStats{fields: fields, stats: stats}, nil
}

func (p *partitionSummaryStats) update(partition map[int]any) error {
	for i, field := range p.fields {
		if err := p.stats[i].update(partition[field.ID]); err != nil {
			return fmt.Errorf("error updating field stats for partition %d: %s: %s", i, field.Name, err)
		}
	}

	return nil
}

func (p *partitionSummaryStats) summaries() []FieldSummary {
	summaries := make([]FieldSummary, len(p.stats))
	for i, stat := range p.stats {
		summaries[i] = stat.toSummary()
	}

	return summaries
}

type ManifestWriter struct {
	closed   bool
	closeErr error
	version  int
	impl     writerImpl

	output io.Writer
	writer *ocf.Writer

	spec    PartitionSpec
	schema  *Schema
	content ManifestContent

	partFieldMaps dataFileFieldMaps

	snapshotID    int64
	addedFiles    int32
	addedRows     int64
	existingFiles int32
	existingRows  int64
	deletedFiles  int32
	deletedRows   int64

	partitionStats    *partitionSummaryStats
	partitionStatsErr error
	minSeqNum         int64
	reusedEntry       manifestEntry
}

type ManifestWriterOption func(w *ManifestWriter)

func WithManifestWriterContent(content ManifestContent) ManifestWriterOption {
	return func(w *ManifestWriter) {
		w.content = content
	}
}

func NewManifestWriter(version int, out io.Writer, spec PartitionSpec, schema *Schema, snapshotID int64, opts ...ManifestWriterOption) (*ManifestWriter, error) {
	var impl writerImpl

	switch version {
	case 1:
		impl = v1writerImpl{}
	case 2:
		impl = v2writerImpl{}
	case 3:
		impl = v3writerImpl{}
	default:
		return nil, fmt.Errorf("unsupported manifest version: %d", version)
	}

	sc, err := partitionTypeToAvroSchema(spec.PartitionType(schema))
	if err != nil {
		return nil, err
	}

	fileSchema, err := internal.NewManifestEntrySchema(sc, version)
	if err != nil {
		return nil, err
	}

	fieldMaps := getFieldIDMap(fileSchema)
	partitionStats, partitionStatsErr := newPartitionSummaryStats(spec, schema)

	w := &ManifestWriter{
		impl:              impl,
		version:           version,
		output:            out,
		spec:              spec,
		content:           ManifestContentData,
		schema:            schema,
		partFieldMaps:     fieldMaps,
		snapshotID:        snapshotID,
		partitionStats:    partitionStats,
		partitionStatsErr: partitionStatsErr,
		minSeqNum:         -1,
	}

	for _, apply := range opts {
		apply(w)
	}
	if w.content != ManifestContentData && w.content != ManifestContentDeletes {
		return nil, fmt.Errorf("%w: invalid manifest content: %d", ErrInvalidArgument, w.content)
	}
	if version < 2 && w.content != ManifestContentData {
		return nil, fmt.Errorf("unsupported content '%s' for format version '%d'", w.content, version)
	}

	md, err := w.meta()
	if err != nil {
		return nil, err
	}

	wr, err := ocf.NewWriter(out, fileSchema,
		ocf.WithSchema(fileSchema.String()),
		ocf.WithMetadata(md),
		ocf.WithCodec(ocf.DeflateCodec(-1)))

	w.writer = wr

	return w, err
}

func (w *ManifestWriter) Close() error {
	if w.closed {
		return w.closeErr
	}
	w.closed = true

	var emptyErr error
	if w.addedFiles+w.existingFiles+w.deletedFiles == 0 {
		emptyErr = ErrEmptyManifest
	}

	var writerErr error
	if w.writer != nil {
		writerErr = w.writer.Close()
	}

	switch {
	case emptyErr == nil:
		w.closeErr = writerErr
	case writerErr == nil:
		w.closeErr = emptyErr
	default:
		w.closeErr = errors.Join(emptyErr, writerErr)
	}

	return w.closeErr
}

type ManifestFileOption func(mf *manifestFile)

// WithManifestFileContent sets the ManifestContent of a new manifest file.
// The default is the content the ManifestWriter was created with, so this is
// only needed to state that content explicitly. Passing a value that disagrees
// with the writer makes ToManifestFile fail.
func WithManifestFileContent(content ManifestContent) ManifestFileOption {
	return func(mf *manifestFile) {
		mf.Content = content
	}
}

func (w *ManifestWriter) ToManifestFile(location string, length int64, opts ...ManifestFileOption) (ManifestFile, error) {
	if err := w.Close(); err != nil {
		return nil, err
	}

	if w.partitionStatsErr != nil {
		return nil, w.partitionStatsErr
	}
	partitions := w.partitionStats.summaries()

	mf := manifestFile{
		version:            w.version,
		Path:               location,
		Len:                length,
		SpecID:             int32(w.spec.id),
		Content:            w.content,
		SeqNumber:          -1,
		MinSeqNumber:       w.minSeqNum,
		AddedSnapshotID:    w.snapshotID,
		AddedFilesCount:    w.addedFiles,
		ExistingFilesCount: w.existingFiles,
		DeletedFilesCount:  w.deletedFiles,
		AddedRowsCount:     w.addedRows,
		ExistingRowsCount:  w.existingRows,
		DeletedRowsCount:   w.deletedRows,
		PartitionList:      &partitions,
		Key:                nil,
	}
	for _, apply := range opts {
		apply(&mf)
	}
	// The writer already stamped w.content into the manifest's own avro metadata,
	// so a different content here would contradict the file it describes.
	if mf.Content != w.content {
		return nil, fmt.Errorf("%w: manifest file content '%s' does not match writer content '%s'",
			ErrInvalidArgument, mf.Content, w.content)
	}

	if mf.Content == ManifestContentDeletes && mf.FirstRowIDValue != nil {
		return nil, errors.New("first_row_id must not be set on delete manifests")
	}

	return &mf, nil
}

func (w *ManifestWriter) meta() (map[string][]byte, error) {
	schemaJson, err := json.Marshal(w.schema)
	if err != nil {
		return nil, err
	}

	specFields := w.spec.fields

	if specFields == nil {
		specFields = []PartitionField{}
	}

	specFieldsJson, err := json.Marshal(specFields)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		"schema":            schemaJson,
		"schema-id":         []byte(strconv.Itoa(w.schema.ID)),
		"partition-spec":    specFieldsJson,
		"partition-spec-id": []byte(strconv.Itoa(w.spec.ID())),
		"format-version":    []byte(strconv.Itoa(w.version)),
		"content":           []byte(w.content.String()),
	}, nil
}

func (w *ManifestWriter) addEntry(entry *manifestEntry) error {
	if w.closed {
		return errors.New("cannot add entry to closed manifest writer")
	}

	status := entry.Status()
	switch status {
	case EntryStatusADDED, EntryStatusEXISTING, EntryStatusDELETED:
	default:
		return fmt.Errorf("unknown entry status: %v", status)
	}
	if w.version > 1 {
		entryContent := entry.DataFile().ContentType()
		switch entryContent {
		case EntryContentData:
			if w.content != ManifestContentData {
				return fmt.Errorf("%w: data file cannot be written to a delete manifest", ErrInvalidArgument)
			}
		case EntryContentPosDeletes, EntryContentEqDeletes:
			if w.content != ManifestContentDeletes {
				return fmt.Errorf("%w: delete file cannot be written to a data manifest", ErrInvalidArgument)
			}
		default:
			return fmt.Errorf("%w: invalid manifest entry content: %d", ErrInvalidArgument, entryContent)
		}
	}
	count := entry.DataFile().Count()

	var partition map[int]any
	entryToEncode := *entry
	if dataFile, ok := entry.DataFile().(*dataFile); ok {
		if err := dataFile.validateForWrite(); err != nil {
			return err
		}
		partition = dataFile.DataFilePartitionRef(internal.DataFileRef{})
		encodeDataFile := cloneDataFileAvroFields(dataFile)
		encodePartition, err := avroEncodePartitionData(
			partition, w.partFieldMaps)
		if err != nil {
			return err
		}
		encodeDataFile.PartitionData = encodePartition
		entryToEncode.Data = encodeDataFile
	} else {
		partition = entry.Data.Partition()
	}

	toEncode, err := w.impl.prepareEntry(&entryToEncode, w.snapshotID)
	if err != nil {
		return err
	}

	if err := w.writer.Encode(toEncode); err != nil {
		return err
	}

	switch status {
	case EntryStatusADDED:
		w.addedFiles++
		w.addedRows += count
	case EntryStatusEXISTING:
		w.existingFiles++
		w.existingRows += count
	case EntryStatusDELETED:
		w.deletedFiles++
		w.deletedRows += count
	}

	if w.partitionStatsErr == nil {
		w.partitionStatsErr = w.partitionStats.update(partition)
	}

	if status == EntryStatusADDED || status == EntryStatusEXISTING {
		if seq := entry.SequenceNum(); seq >= 0 && (w.minSeqNum < 0 || seq < w.minSeqNum) {
			w.minSeqNum = seq
		}
	}

	return nil
}

func (w *ManifestWriter) Add(entry ManifestEntry) error {
	w.reusedEntry.wrap(EntryStatusADDED, &w.snapshotID, entry.(*manifestEntry).SeqNum, nil, entry.DataFile())

	return w.addEntry(&w.reusedEntry)
}

func (w *ManifestWriter) Delete(entry ManifestEntry) error {
	w.reusedEntry.wrap(EntryStatusDELETED, &w.snapshotID, entry.(*manifestEntry).SeqNum, entry.FileSequenceNum(), entry.DataFile())

	return w.addEntry(&w.reusedEntry)
}

func (w *ManifestWriter) Existing(entry ManifestEntry) error {
	snapshotID := entry.SnapshotID()
	w.reusedEntry.wrap(EntryStatusEXISTING, &snapshotID, entry.(*manifestEntry).SeqNum, entry.FileSequenceNum(), entry.DataFile())

	return w.addEntry(&w.reusedEntry)
}

type ManifestListWriter struct {
	version          int
	out              io.Writer
	commitSnapshotID int64
	sequenceNumber   int64
	writer           *ocf.Writer
	nextRowID        *int64
	failure          error
}

func NewManifestListWriterV1(out io.Writer, snapshotID int64, parentSnapshot *int64) (*ManifestListWriter, error) {
	m := &ManifestListWriter{
		version:          1,
		out:              out,
		commitSnapshotID: snapshotID,
		sequenceNumber:   -1,
	}

	parentSnapshotStr := "null"
	if parentSnapshot != nil {
		parentSnapshotStr = strconv.FormatInt(*parentSnapshot, 10)
	}

	return m, m.init(map[string][]byte{
		"format-version":     []byte(strconv.Itoa(m.version)),
		"snapshot-id":        []byte(strconv.FormatInt(snapshotID, 10)),
		"parent-snapshot-id": []byte(parentSnapshotStr),
	})
}

func NewManifestListWriterV2(out io.Writer, snapshotID, sequenceNumber int64, parentSnapshot *int64) (*ManifestListWriter, error) {
	m := &ManifestListWriter{
		version:          2,
		out:              out,
		commitSnapshotID: snapshotID,
		sequenceNumber:   sequenceNumber,
	}

	parentSnapshotStr := "null"
	if parentSnapshot != nil {
		parentSnapshotStr = strconv.FormatInt(*parentSnapshot, 10)
	}

	return m, m.init(map[string][]byte{
		"format-version":     []byte(strconv.Itoa(m.version)),
		"snapshot-id":        []byte(strconv.FormatInt(snapshotID, 10)),
		"sequence-number":    []byte(strconv.FormatInt(sequenceNumber, 10)),
		"parent-snapshot-id": []byte(parentSnapshotStr),
	})
}

func NewManifestListWriterV3(out io.Writer, snapshotId, sequenceNumber, firstRowID int64, parentSnapshot *int64) (*ManifestListWriter, error) {
	if firstRowID < 0 {
		return nil, fmt.Errorf("%w: first row ID must be non-negative: %d", ErrInvalidArgument, firstRowID)
	}

	m := &ManifestListWriter{
		version:          3,
		out:              out,
		commitSnapshotID: snapshotId,
		sequenceNumber:   sequenceNumber,
		nextRowID:        &firstRowID,
	}
	parentSnapshotStr := "null"
	if parentSnapshot != nil {
		parentSnapshotStr = strconv.FormatInt(*parentSnapshot, 10)
	}

	return m, m.init(map[string][]byte{
		"format-version":     []byte(strconv.Itoa(m.version)),
		"snapshot-id":        []byte(strconv.FormatInt(snapshotId, 10)),
		"sequence-number":    []byte(strconv.FormatInt(sequenceNumber, 10)),
		"first-row-id":       []byte(strconv.FormatInt(firstRowID, 10)),
		"parent-snapshot-id": []byte(parentSnapshotStr),
	})
}

func advanceRowID(firstRowID, existingRows, addedRows int64) (int64, error) {
	if firstRowID < 0 {
		return 0, fmt.Errorf("%w: first row ID must be non-negative: %d", ErrInvalidArgument, firstRowID)
	}
	if existingRows == -1 || addedRows == -1 {
		return 0, fmt.Errorf("%w: cannot assign row-lineage IDs because at least one row count is unknown (existing=%d added=%d); rewrite or compact the legacy manifest before upgrading to v3",
			ErrInvalidArgument, existingRows, addedRows)
	}
	if existingRows < 0 || addedRows < 0 {
		return 0, fmt.Errorf("%w: row counts must be non-negative: existing=%d added=%d",
			ErrInvalidArgument, existingRows, addedRows)
	}
	if existingRows > math.MaxInt64-firstRowID || addedRows > math.MaxInt64-firstRowID-existingRows {
		return 0, fmt.Errorf("%w: assigning %d existing and %d added rows from first row ID %d overflows int64",
			ErrInvalidArgument, existingRows, addedRows, firstRowID)
	}

	return firstRowID + existingRows + addedRows, nil
}

func (m *ManifestListWriter) init(meta map[string][]byte) error {
	fileSchema, err := internal.NewManifestFileSchema(m.version)
	if err != nil {
		return err
	}

	wr, err := ocf.NewWriter(m.out, fileSchema,
		ocf.WithSchema(fileSchema.String()),
		ocf.WithMetadata(meta),
		ocf.WithCodec(ocf.DeflateCodec(-1)))
	if err != nil {
		return err
	}

	m.writer = wr

	return nil
}

func (m *ManifestListWriter) Close() error {
	if m.writer == nil {
		return nil
	}

	return m.writer.Close()
}

func (m *ManifestListWriter) NextRowID() *int64 {
	return m.nextRowID
}

// AddManifests appends manifest files to the list. If it returns an error, the
// writer is poisoned: callers must close and discard it, and subsequent calls
// return the original error.
func (m *ManifestListWriter) AddManifests(files []ManifestFile) (err error) {
	if m.failure != nil {
		return m.failure
	}
	if len(files) == 0 {
		return nil
	}
	hasRowID := m.version == 3 && m.nextRowID != nil
	var batchStartRowID int64
	if hasRowID {
		batchStartRowID = *m.nextRowID
	}
	defer func() {
		if err != nil {
			if hasRowID {
				*m.nextRowID = batchStartRowID
			}
			m.failure = err
		}
	}()

	switch m.version {
	case 1:
		if slices.ContainsFunc(files, func(f ManifestFile) bool {
			return f.Version() != 1
		}) {
			return fmt.Errorf("%w: ManifestListWriter only supports version 1 manifest files", ErrInvalidArgument)
		}

		var tmp manifestFileV1
		for _, file := range files {
			file.(*manifestFile).toV1(&tmp)
			ensurePartitionList(&tmp.PartitionList)
			if err := m.writer.Encode(&tmp); err != nil {
				return err
			}
		}

	case 2, 3:
		for _, file := range files {
			var assignedNextRowID *int64

			// Per the Iceberg spec a v2 manifest list may reference v1 manifest
			// files (and a v3 list may reference v1 or v2 manifests) so that a
			// table can be upgraded without rewriting historical manifests. The
			// in-memory ManifestFile produced for v1 inputs already carries the
			// inheritance values mandated by the spec — Content=data and
			// SeqNumber/MinSeqNumber=0 — so it can be encoded directly against
			// the v2/v3 entry schema. Newer-than-writer inputs are rejected
			// because the v2 schema cannot represent v3 fields such as
			// first_row_id.
			if file.Version() > m.version {
				return fmt.Errorf("%w: manifest list v%d cannot reference v%d manifest files",
					ErrInvalidArgument, m.version, file.Version())
			}

			wrapped := *(file.(*manifestFile))
			ensurePartitionList(&wrapped.PartitionList)
			if m.version == 3 {
				// Ref: https://github.com/apache/iceberg/blob/ea2071568dc66148b483a82eefedcd2992b435f7/core/src/main/java/org/apache/iceberg/ManifestListWriter.java#L157-L168
				if wrapped.Content == ManifestContentData {
					if wrapped.FirstRowIDValue == nil {
						if m.nextRowID != nil {
							firstRowID := *m.nextRowID
							nextRowID, err := advanceRowID(firstRowID, wrapped.ExistingRowsCount, wrapped.AddedRowsCount)
							if err != nil {
								return fmt.Errorf("manifest %q: %w", wrapped.Path, err)
							}
							wrapped.FirstRowIDValue = &firstRowID
							assignedNextRowID = &nextRowID
						}
					}
				}
			}
			if wrapped.SeqNumber == -1 {
				// if the sequence number is being assigned here,
				// then the manifest must be created by the current
				// operation.
				// to validate this, check the snapshot id matches the current commit
				if m.commitSnapshotID != wrapped.AddedSnapshotID {
					return fmt.Errorf("found unassigned sequence number for a manifest from snapshot %d != %d",
						m.commitSnapshotID, wrapped.AddedSnapshotID)
				}
				wrapped.SeqNumber = m.sequenceNumber
			}

			if wrapped.MinSeqNumber == -1 {
				if m.commitSnapshotID != wrapped.AddedSnapshotID {
					return fmt.Errorf("found unassigned sequence number for a manifest from snapshot: %d", wrapped.AddedSnapshotID)
				}
				// if the min sequence number is not determined, then there was no assigned sequence number
				// for any file written to the wrapped manifest. replace the unassigned sequence number with
				// the one for this commit
				wrapped.MinSeqNumber = m.sequenceNumber
			}
			if err := m.writer.Encode(wrapped); err != nil {
				return err
			}
			if assignedNextRowID != nil {
				*m.nextRowID = *assignedNextRowID
			}
		}
	default:
		return fmt.Errorf("unsupported manifest version: %d", m.version)
	}

	return nil
}

// WriteManifestList writes a list of manifest files to an avro file.
func WriteManifestList(version int, out io.Writer, snapshotID int64, parentSnapshotID, sequenceNumber *int64, firstRowId int64, files []ManifestFile) (err error) {
	var writer *ManifestListWriter

	switch version {
	case 1:
		writer, err = NewManifestListWriterV1(out, snapshotID, parentSnapshotID)
	case 2:
		if sequenceNumber == nil {
			return errors.New("sequence number is required for V2 tables")
		}
		writer, err = NewManifestListWriterV2(out, snapshotID, *sequenceNumber, parentSnapshotID)
	case 3:
		if sequenceNumber == nil {
			return errors.New("sequence number is required for V3 tables")
		}
		writer, err = NewManifestListWriterV3(out, snapshotID, *sequenceNumber, firstRowId, parentSnapshotID)
	default:
		return fmt.Errorf("unsupported manifest version: %d", version)
	}

	if err != nil {
		return err
	}
	defer internal.CheckedClose(writer, &err)

	return writer.AddManifests(files)
}

func WriteManifest(
	filename string,
	out io.Writer,
	version int,
	spec PartitionSpec,
	schema *Schema,
	snapshotID int64,
	entries []ManifestEntry,
) (mf ManifestFile, err error) {
	cnt := &internal.CountingWriter{W: out}

	w, err := NewManifestWriter(version, cnt, spec, schema, snapshotID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if !w.closed {
			internal.CheckedClose(w, &err)
		}
	}()

	for _, entry := range entries {
		if err := w.addEntry(entry.(*manifestEntry)); err != nil {
			return nil, err
		}
	}

	// flush the writer to ensure cnt.Count is accurate
	if err := w.Close(); err != nil {
		return nil, err
	}

	return w.ToManifestFile(filename, cnt.Count)
}

// WriteManifestV3 writes a v3 data manifest and assigns first_row_id.
// The returned ManifestFile has FirstRowID() set to firstRowID.
// nextFirstRowID is firstRowID + AddedRowsCount + ExistingRowsCount
// for the caller to chain to the next manifest. Use this only for
// reconstruction — manifests committed through a v3 list writer get
// first_row_id from AddManifests; pre-setting collides with that allocation.
func WriteManifestV3(
	filename string,
	out io.Writer,
	firstRowID int64,
	spec PartitionSpec,
	schema *Schema,
	snapshotID int64,
	entries []ManifestEntry,
) (mf ManifestFile, nextFirstRowID int64, err error) {
	if firstRowID < 0 {
		return nil, 0, fmt.Errorf("%w: first row ID must be non-negative: %d", ErrInvalidArgument, firstRowID)
	}

	cnt := &internal.CountingWriter{W: out}

	w, err := NewManifestWriter(3, cnt, spec, schema, snapshotID)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if !w.closed {
			internal.CheckedClose(w, &err)
		}
	}()

	for _, entry := range entries {
		if err := w.addEntry(entry.(*manifestEntry)); err != nil {
			return nil, 0, err
		}
	}

	if err := w.Close(); err != nil {
		return nil, 0, err
	}

	mf, err = w.ToManifestFile(filename, cnt.Count)
	if err != nil {
		return nil, 0, err
	}

	raw := mf.(*manifestFile)
	v := firstRowID
	raw.FirstRowIDValue = &v
	nextFirstRowID, err = advanceRowID(firstRowID, raw.ExistingRowsCount, raw.AddedRowsCount)
	if err != nil {
		return nil, 0, fmt.Errorf("manifest %q: %w", filename, err)
	}

	return raw, nextFirstRowID, nil
}

// ManifestEntryStatus defines constants for the entry status of
// existing, added or deleted.
type ManifestEntryStatus int8

const (
	EntryStatusEXISTING ManifestEntryStatus = 0
	EntryStatusADDED    ManifestEntryStatus = 1
	EntryStatusDELETED  ManifestEntryStatus = 2
)

// ManifestEntryContent defines constants for the type of file contents
// in the file entries. Data, Position based deletes and equality based
// deletes.
type ManifestEntryContent int8

const (
	EntryContentData       ManifestEntryContent = 0
	EntryContentPosDeletes ManifestEntryContent = 1
	EntryContentEqDeletes  ManifestEntryContent = 2
)

func (m ManifestEntryContent) String() string {
	switch m {
	case EntryContentData:
		return "Data"
	case EntryContentPosDeletes:
		return "Positional_Deletes"
	case EntryContentEqDeletes:
		return "Equality_Deletes"
	default:
		return "UNKNOWN"
	}
}

// FileFormat defines constants for the format of data files.
type FileFormat string

const (
	AvroFile    FileFormat = "AVRO"
	OrcFile     FileFormat = "ORC"
	ParquetFile FileFormat = "PARQUET"
	PuffinFile  FileFormat = "PUFFIN"
)

// FileFormatFromString parses a file format string (case-insensitive).
func FileFormatFromString(s string) (FileFormat, error) {
	switch strings.ToUpper(s) {
	case string(ParquetFile):
		return ParquetFile, nil
	case string(OrcFile):
		return OrcFile, nil
	case string(AvroFile):
		return AvroFile, nil
	case string(PuffinFile):
		return PuffinFile, nil
	default:
		return "", fmt.Errorf("unknown file format: %s", s)
	}
}

type colMap[K, V any] struct {
	Key   K `avro:"key"`
	Value V `avro:"value"`
}

func avroColMapToMap[K comparable, V any](c *[]colMap[K, V]) map[K]V {
	if c == nil {
		return nil
	}

	out := make(map[K]V, len(*c))
	for _, data := range *c {
		out[data.Key] = data.Value
	}

	return out
}

func mapToAvroColMap[K comparable, V any](m map[K]V) *[]colMap[K, V] {
	if m == nil {
		return nil
	}

	out := make([]colMap[K, V], 0, len(m))
	for k, v := range m {
		out = append(out, colMap[K, V]{Key: k, Value: v})
	}

	return &out
}

func convertLogicalTypeValue(v any, logicalType string, fixedSize int) (any, error) {
	switch logicalType {
	case atype.Date:
		return convertDateValue(v), nil
	case atype.TimeMicros:
		return convertTimeMicrosValue(v), nil
	case atype.TimestampMicros:
		return convertTimestampMicrosValue(v), nil
	case atype.Decimal:
		return convertDecimalValue(v, fixedSize)
	case atype.UUID:
		return convertUUIDValue(v), nil
	default:
		return v, nil
	}
}

func convertDateValue(v any) any {
	if d, ok := v.(Date); ok {
		return int32(d)
	}

	return v
}

func convertTimeMicrosValue(v any) any {
	if t, ok := v.(Time); ok {
		return int64(t)
	}

	return v
}

func convertTimestampMicrosValue(v any) any {
	if ts, ok := v.(Timestamp); ok {
		return int64(ts)
	}

	return v
}

func convertDecimalValue(v any, fixedSize int) (any, error) {
	switch dec := v.(type) {
	case Decimal:
		return encodeDecimalBytes(DecimalLiteral(dec), fixedSize)
	case DecimalLiteral:
		return encodeDecimalBytes(dec, fixedSize)
	default:
		return v, nil
	}
}

func encodeDecimalBytes(dec DecimalLiteral, fixedSize int) ([]byte, error) {
	if fixedSize <= 0 {
		return nil, errors.New("missing decimal fixed size")
	}

	bytes, err := dec.MarshalBinary()
	if err != nil {
		return nil, err
	}

	return fitDecimalBytes(bytes, fixedSize)
}

func convertUUIDValue(v any) any {
	if uuidVal, ok := v.(uuid.UUID); ok {
		return [16]byte(uuidVal)
	}

	return v
}

func fitDecimalBytes(bytes []byte, size int) ([]byte, error) {
	if size <= 0 {
		return nil, fmt.Errorf("decimal fixed size must be positive: %d", size)
	}

	if len(bytes) == size {
		return bytes, nil
	}

	if len(bytes) < size {
		signByte := byte(0x00)
		if bytes[0]&0x80 != 0 {
			signByte = 0xff
		}
		padded := make([]byte, size)
		for i := range padded[:size-len(bytes)] {
			padded[i] = signByte
		}
		copy(padded[size-len(bytes):], bytes)

		return padded, nil
	}

	retained := bytes[len(bytes)-size:]
	signByte := byte(0x00)
	if retained[0]&0x80 != 0 {
		signByte = 0xff
	}
	for _, b := range bytes[:len(bytes)-size] {
		if b != signByte {
			return nil, fmt.Errorf("decimal value of %d bytes does not fit fixed size %d", len(bytes), size)
		}
	}

	return retained, nil
}

type dataFile struct {
	Content                 ManifestEntryContent `avro:"content"`
	projected               bool
	Path                    string                 `avro:"file_path"`
	Format                  FileFormat             `avro:"file_format"`
	PartitionData           map[string]any         `avro:"partition"`
	RecordCount             int64                  `avro:"record_count"`
	FileSize                int64                  `avro:"file_size_in_bytes"`
	BlockSizeInBytes        int64                  `avro:"block_size_in_bytes"`
	ColSizes                *[]colMap[int, int64]  `avro:"column_sizes"`
	ValCounts               *[]colMap[int, int64]  `avro:"value_counts"`
	NullCounts              *[]colMap[int, int64]  `avro:"null_value_counts"`
	NaNCounts               *[]colMap[int, int64]  `avro:"nan_value_counts"`
	DistinctCounts          *[]colMap[int, int64]  `avro:"distinct_counts"`
	LowerBounds             *[]colMap[int, []byte] `avro:"lower_bounds"`
	UpperBounds             *[]colMap[int, []byte] `avro:"upper_bounds"`
	Key                     *[]byte                `avro:"key_metadata"`
	Splits                  *[]int64               `avro:"split_offsets"`
	EqualityIDs             *[]int                 `avro:"equality_ids"`
	SortOrder               *int                   `avro:"sort_order_id"`
	FirstRowIDField         *int64                 `avro:"first_row_id"`
	ReferencedDataFileField *string                `avro:"referenced_data_file"`
	ContentOffsetField      *int64                 `avro:"content_offset"`
	ContentSizeInBytesField *int64                 `avro:"content_size_in_bytes"`

	colSizeMap     map[int]int64
	valCntMap      map[int]int64
	nullCntMap     map[int]int64
	nanCntMap      map[int]int64
	distinctCntMap map[int]int64
	lowerBoundMap  map[int][]byte
	upperBoundMap  map[int][]byte

	// used for partition retrieval
	fieldNameToID          map[string]int
	fieldIDToLogicalType   map[int]string
	fieldIDToPartitionData map[int]any
	fieldIDToDecimalScale  map[int]int

	specID          int32
	initPartition   sync.Once
	initColumnStats sync.Once
}

func (d *dataFile) initPartitionData() {
	d.initPartition.Do(func() {
		if len(d.fieldIDToPartitionData) < len(d.PartitionData) {
			d.fieldIDToPartitionData = make(map[int]any, len(d.PartitionData))
			for k, v := range d.PartitionData {
				if id, ok := d.fieldNameToID[k]; ok {
					convertedValue := d.convertAvroValueToIcebergType(v, id)
					d.fieldIDToPartitionData[id] = convertedValue
				}
			}
		}
	})
}

func (d *dataFile) initColumnStatsData() {
	d.initColumnStats.Do(func() {
		d.colSizeMap = avroColMapToMap(d.ColSizes)
		d.valCntMap = avroColMapToMap(d.ValCounts)
		d.nullCntMap = avroColMapToMap(d.NullCounts)
		d.nanCntMap = avroColMapToMap(d.NaNCounts)
		d.distinctCntMap = avroColMapToMap(d.DistinctCounts)
		d.lowerBoundMap = avroColMapToMap(d.LowerBounds)
		d.upperBoundMap = avroColMapToMap(d.UpperBounds)
	})
}

func (d *dataFile) convertAvroValueToIcebergType(v any, fieldID int) any {
	if v == nil {
		return nil
	}

	if logicalType, ok := d.fieldIDToLogicalType[fieldID]; ok {
		// twmb/avro returns rich Go types (time.Time, time.Duration,
		// *big.Rat, [16]byte) when the file schema includes a logicalType,
		// but raw primitives (int32, int64, []byte, string) when it does
		// not. Each case handles both possibilities.
		switch logicalType {
		case atype.Date:
			if val, ok := v.(time.Time); ok {
				return Date(val.Truncate(24*time.Hour).Unix() / int64((time.Hour * 24).Seconds()))
			}
			if val, ok := v.(int32); ok {
				return Date(val)
			}

			// Unreachable with twmb/avro: an int+date logical type decodes
			// to time.Time and a plain int to int32, so v is always one of
			// the two cases above. Returning v rather than panicking keeps a
			// future decoder change from crashing here, but callers that type
			// assert iceberg.Date would then fail; do not add a guard that
			// silently coerces other types, which reintroduces #1200.
			return v
		case atype.TimeMillis:
			// iceberg.Time counts microseconds; a time-millis Duration must be
			// read as microseconds (val.Milliseconds() undercounts 1000x). The
			// raw arm converts the millisecond wire value to microseconds too.
			if val, ok := v.(time.Duration); ok {
				return Time(val.Microseconds())
			}
			if ms, ok := avroIntAsInt64(v); ok {
				return Time(ms * 1000)
			}

			return v
		case atype.TimeMicros:
			if val, ok := v.(time.Duration); ok {
				return Time(val.Microseconds())
			}
			if us, ok := avroIntAsInt64(v); ok {
				return Time(us)
			}

			return v
		case atype.TimestampMillis:
			// iceberg.Timestamp counts microseconds; UnixMilli() undercounts
			// 1000x. Read both the time.Time and the raw millisecond wire value
			// as microseconds.
			if val, ok := v.(time.Time); ok {
				return Timestamp(val.UTC().UnixMicro())
			}
			if ms, ok := avroIntAsInt64(v); ok {
				return Timestamp(ms * 1000)
			}

			return v
		case atype.TimestampMicros:
			if val, ok := v.(time.Time); ok {
				return Timestamp(val.UTC().UnixMicro())
			}
			if us, ok := avroIntAsInt64(v); ok {
				return Timestamp(us)
			}

			return v
		case atype.TimestampNanos:
			if val, ok := v.(time.Time); ok {
				return TimestampNano(val.UTC().UnixNano())
			}
			if ns, ok := avroIntAsInt64(v); ok {
				return TimestampNano(ns)
			}

			return v
		case atype.Decimal:
			if r, ok := v.(*big.Rat); ok {
				scale := d.fieldIDToDecimalScale[fieldID]
				scaleFactor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
				unscaled := new(big.Int).Mul(r.Num(), scaleFactor)
				unscaled = unscaled.Div(unscaled, r.Denom())

				return DecimalLiteral{
					Scale: scale,
					Val:   decimal128.FromBigInt(unscaled),
				}
			}

			return v
		case atype.UUID:
			if uuidArr, ok := v.([16]byte); ok {
				return uuid.UUID(uuidArr)
			}

			return v
		}
	}

	return v
}

// avroIntAsInt64 extracts an Avro int (int32) or long (int64) partition value
// as an int64. A manifest from a foreign writer may declare a time/timestamp
// logicalType on an underlying Avro type the spec does not permit for it (e.g.
// timestamp-millis on an int). twmb/avro drops such an invalid logicalType when
// decoding and yields the raw primitive, yet Schema.Root still reports the
// logicalType, so the conversion arms must accept the raw integer instead of an
// unchecked assertion that would panic (apache/iceberg-go#1847).
func avroIntAsInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int32:
		return int64(n), true
	case int:
		return int64(n), true
	}

	return 0, false
}

func (d *dataFile) setFieldNameToIDMap(m map[string]int) { d.fieldNameToID = m }
func (d *dataFile) setFieldIDToLogicalTypeMap(m map[int]string) {
	d.fieldIDToLogicalType = m
}

func (d *dataFile) setFieldIDToDecimalScaleMap(m map[int]int) {
	d.fieldIDToDecimalScale = m
}

func (d *dataFile) ContentType() ManifestEntryContent { return d.Content }
func (d *dataFile) FilePath() string                  { return d.Path }
func (d *dataFile) FileFormat() FileFormat            { return d.Format }

// Partition returns the partition data as a map of partition field ID to value.
func (d *dataFile) Partition() map[int]any {
	d.initPartitionData()

	return clonePartitionMap(d.fieldIDToPartitionData)
}

func clonePartitionMap(src map[int]any) map[int]any {
	if src == nil {
		return nil
	}

	// Manifest decoding produces immutable scalar/literal values and []byte
	// for binary and fixed values. Clone the latter explicitly; arbitrary
	// caller-defined mutable values are not recursively copied.
	out := maps.Clone(src)
	for id, value := range out {
		if bytes, ok := value.([]byte); ok {
			out[id] = slices.Clone(bytes)
		}
	}

	return out
}

func cloneByteMap(src map[int][]byte) map[int][]byte {
	if src == nil {
		return nil
	}

	out := make(map[int][]byte, len(src))
	for id, value := range src {
		out[id] = slices.Clone(value)
	}

	return out
}

func mapToAvroColMapClonedBytes(m map[int][]byte) *[]colMap[int, []byte] {
	if m == nil {
		return nil
	}

	out := make([]colMap[int, []byte], 0, len(m))
	for k, v := range m {
		out = append(out, colMap[int, []byte]{Key: k, Value: slices.Clone(v)})
	}

	return &out
}

func clonePointer[T any](value *T) *T {
	if value == nil {
		return nil
	}

	cloned := *value

	return &cloned
}

func (d *dataFile) Count() int64         { return d.RecordCount }
func (d *dataFile) FileSizeBytes() int64 { return d.FileSize }
func (d *dataFile) SpecID() int32        { return d.specID }

func (d *dataFile) ColumnSizes() map[int]int64 {
	d.initColumnStatsData()

	return maps.Clone(d.colSizeMap)
}

func (d *dataFile) ValueCounts() map[int]int64 {
	d.initColumnStatsData()

	return maps.Clone(d.valCntMap)
}

func (d *dataFile) NullValueCounts() map[int]int64 {
	d.initColumnStatsData()

	return maps.Clone(d.nullCntMap)
}

func (d *dataFile) NaNValueCounts() map[int]int64 {
	d.initColumnStatsData()

	return maps.Clone(d.nanCntMap)
}

func (d *dataFile) DistinctValueCounts() map[int]int64 {
	d.initColumnStatsData()

	return maps.Clone(d.distinctCntMap)
}

func (d *dataFile) LowerBoundValues() map[int][]byte {
	d.initColumnStatsData()

	return cloneByteMap(d.lowerBoundMap)
}

func (d *dataFile) UpperBoundValues() map[int][]byte {
	d.initColumnStatsData()

	return cloneByteMap(d.upperBoundMap)
}

func (d *dataFile) KeyMetadata() []byte {
	if d.Key == nil {
		return nil
	}

	return slices.Clone(*d.Key)
}

func (d *dataFile) SplitOffsets() []int64 {
	if d.Splits == nil {
		return nil
	}

	return slices.Clone(*d.Splits)
}

func (d *dataFile) EqualityFieldIDs() []int {
	if d.EqualityIDs == nil {
		return nil
	}

	return slices.Clone(*d.EqualityIDs)
}

func (d *dataFile) SortOrderID() *int { return clonePointer(d.SortOrder) }

func (d *dataFile) FirstRowID() *int64          { return clonePointer(d.FirstRowIDField) }
func (d *dataFile) ReferencedDataFile() *string { return clonePointer(d.ReferencedDataFileField) }
func (d *dataFile) ContentSizeInBytes() *int64  { return clonePointer(d.ContentSizeInBytesField) }
func (d *dataFile) ContentOffset() *int64       { return clonePointer(d.ContentOffsetField) }

type ManifestEntryBuilder struct {
	m *manifestEntry
}

func NewManifestEntryBuilder(status ManifestEntryStatus, snapshotID *int64, data DataFile) *ManifestEntryBuilder {
	return &ManifestEntryBuilder{
		m: &manifestEntry{
			EntryStatus: status,
			Snapshot:    snapshotID,
			Data:        data,
		},
	}
}

func (b *ManifestEntryBuilder) SequenceNum(num int64) *ManifestEntryBuilder {
	b.m.SeqNum = &num

	return b
}

func (b *ManifestEntryBuilder) FileSequenceNum(num int64) *ManifestEntryBuilder {
	b.m.FileSeqNum = &num

	return b
}

func (b *ManifestEntryBuilder) Build() ManifestEntry {
	return b.m
}

type manifestEntry struct {
	EntryStatus ManifestEntryStatus `avro:"status"`
	Snapshot    *int64              `avro:"snapshot_id"`
	SeqNum      *int64              `avro:"sequence_number"`
	FileSeqNum  *int64              `avro:"file_sequence_number"`
	Data        DataFile            `avro:"data_file"`
}

func (m *manifestEntry) Status() ManifestEntryStatus { return m.EntryStatus }
func (m *manifestEntry) SnapshotID() int64 {
	if m.Snapshot == nil {
		return -1
	}

	return *m.Snapshot
}

func (m *manifestEntry) SequenceNum() int64 {
	if m.SeqNum == nil {
		return -1
	}

	return *m.SeqNum
}

func (m *manifestEntry) FileSequenceNum() *int64 {
	if m.FileSeqNum == nil {
		return nil
	}

	fileSeqNum := *m.FileSeqNum

	return &fileSeqNum
}

func (m *manifestEntry) DataFile() DataFile { return m.Data }

func (m *manifestEntry) inherit(manifest ManifestFile) {
	if m.Snapshot == nil {
		snap := manifest.SnapshotID()
		m.Snapshot = &snap
	}

	manifestSequenceNum := manifest.SequenceNum()
	if manifestSequenceNum != -1 {
		if m.SeqNum == nil && (manifestSequenceNum == initialSequenceNumber || m.EntryStatus == EntryStatusADDED) {
			m.SeqNum = &manifestSequenceNum
		}

		if m.FileSeqNum == nil && (manifestSequenceNum == initialSequenceNumber || m.EntryStatus == EntryStatusADDED) {
			m.FileSeqNum = &manifestSequenceNum
		}
	}

	m.Data.(*dataFile).specID = manifest.PartitionSpecID()
}

func (m *manifestEntry) wrap(status ManifestEntryStatus, newSnapID, newSeq, newFileSeq *int64, data DataFile) ManifestEntry {
	if newSeq != nil && *newSeq == -1 {
		newSeq = nil
	}

	m.EntryStatus = status
	m.Snapshot = newSnapID
	m.SeqNum = newSeq
	m.FileSeqNum = newFileSeq
	m.Data = data

	return m
}

type fallbackManifestEntry struct {
	manifestEntry
	Snapshot int64 `avro:"snapshot_id"`
}

func (f *fallbackManifestEntry) toEntry() *manifestEntry {
	f.manifestEntry.Snapshot = &f.Snapshot

	return &f.manifestEntry
}

func NewManifestEntry(status ManifestEntryStatus, snapshotID *int64, seqNum, fileSeqNum *int64, df DataFile) ManifestEntry {
	return &manifestEntry{
		EntryStatus: status,
		Snapshot:    snapshotID,
		SeqNum:      seqNum,
		FileSeqNum:  fileSeqNum,
		Data:        df,
	}
}

// DataFileBuilder is a helper for building a data file struct which will
// conform to the DataFile interface.
type DataFileBuilder struct {
	d *dataFile
}

// NewDataFileBuilder is passed all of the required fields and then allows
// all of the optional fields to be set by calling the corresponding methods
// before calling [DataFileBuilder.Build] to construct the object.
// The fieldIDToFixedSize argument is retained for source compatibility and is
// ignored; manifest schemas determine decimal fixed widths during encoding.
// Byte-slice partition values are cloned. Other partition values must be the
// immutable scalar or literal values produced by Iceberg's manifest decoder;
// mutable caller-defined values are retained by reference.
func NewDataFileBuilder(
	spec PartitionSpec,
	content ManifestEntryContent,
	path string,
	format FileFormat,
	fieldIDToPartitionData map[int]any,
	fieldIDToLogicalType map[int]string,
	fieldIDToFixedSize map[int]int,
	recordCount int64,
	fileSize int64,
) (*DataFileBuilder, error) {
	_ = fieldIDToFixedSize
	if content != EntryContentData && content != EntryContentPosDeletes && content != EntryContentEqDeletes {
		return nil, fmt.Errorf(
			"%w: content must be one of %s, %s, or %s",
			ErrInvalidArgument, EntryContentData, EntryContentPosDeletes, EntryContentEqDeletes,
		)
	}

	if path == "" {
		return nil, fmt.Errorf("%w: path cannot be empty", ErrInvalidArgument)
	}

	if format != AvroFile && format != OrcFile && format != ParquetFile && format != PuffinFile {
		return nil, fmt.Errorf(
			"%w: format must be one of %s, %s, %s, or %s",
			ErrInvalidArgument, AvroFile, OrcFile, ParquetFile, PuffinFile,
		)
	}

	if format == PuffinFile && content != EntryContentPosDeletes {
		return nil, fmt.Errorf(
			"%w: %s format is only valid for %s content",
			ErrInvalidArgument, PuffinFile, EntryContentPosDeletes,
		)
	}

	if recordCount < 0 {
		return nil, fmt.Errorf("%w: record count must be non-negative", ErrInvalidArgument)
	}

	if fileSize < 0 {
		return nil, fmt.Errorf("%w: file size must be non-negative", ErrInvalidArgument)
	}
	partitionData := make(map[string]any)
	fieldNameToID := make(map[string]int)
	for _, p := range spec.fields {
		if pData, ok := fieldIDToPartitionData[p.FieldID]; ok {
			if bytes, ok := pData.([]byte); ok {
				pData = slices.Clone(bytes)
			}
			partitionData[p.Name] = pData
			fieldNameToID[p.Name] = p.FieldID
		}
	}

	return &DataFileBuilder{
		d: &dataFile{
			Content:                content,
			Path:                   path,
			Format:                 format,
			PartitionData:          partitionData,
			RecordCount:            recordCount,
			FileSize:               fileSize,
			specID:                 int32(spec.id),
			fieldIDToPartitionData: clonePartitionMap(fieldIDToPartitionData),
			fieldNameToID:          fieldNameToID,
			fieldIDToLogicalType:   maps.Clone(fieldIDToLogicalType),
		},
	}, nil
}

// BlockSizeInBytes sets the block size in bytes for the data file. Deprecated in v2.
func (b *DataFileBuilder) BlockSizeInBytes(size int64) *DataFileBuilder {
	b.d.BlockSizeInBytes = size

	return b
}

// ColumnSizes sets the column sizes for the data file.
func (b *DataFileBuilder) ColumnSizes(sizes map[int]int64) *DataFileBuilder {
	b.d.ColSizes = mapToAvroColMap(sizes)

	return b
}

// ValueCounts sets the value counts for the data file.
func (b *DataFileBuilder) ValueCounts(counts map[int]int64) *DataFileBuilder {
	b.d.ValCounts = mapToAvroColMap(counts)

	return b
}

// NullValueCounts sets the null value counts for the data file.
func (b *DataFileBuilder) NullValueCounts(counts map[int]int64) *DataFileBuilder {
	b.d.NullCounts = mapToAvroColMap(counts)

	return b
}

// NaNValueCounts sets the NaN value counts for the data file.
func (b *DataFileBuilder) NaNValueCounts(counts map[int]int64) *DataFileBuilder {
	b.d.NaNCounts = mapToAvroColMap(counts)

	return b
}

// DistinctValueCounts sets the distinct value counts for the data file.
//
// Deprecated: distinct_counts (field 111) is deprecated in every
// version of the Iceberg spec (apache/iceberg#12182). The Avro
// manifest-entry schemas omit the field for v1, v2, and v3, so values
// set here are not transported in manifests written by this library.
// The setter is retained for round-tripping legacy DataFiles read from
// older manifests; new code should not call it.
func (b *DataFileBuilder) DistinctValueCounts(counts map[int]int64) *DataFileBuilder {
	b.d.DistinctCounts = mapToAvroColMap(counts)

	return b
}

// LowerBoundValues sets the lower bound values for the data file.
func (b *DataFileBuilder) LowerBoundValues(bounds map[int][]byte) *DataFileBuilder {
	b.d.LowerBounds = mapToAvroColMapClonedBytes(bounds)

	return b
}

// UpperBoundValues sets the upper bound values for the data file.
func (b *DataFileBuilder) UpperBoundValues(bounds map[int][]byte) *DataFileBuilder {
	b.d.UpperBounds = mapToAvroColMapClonedBytes(bounds)

	return b
}

// KeyMetadata sets the key metadata for the data file.
func (b *DataFileBuilder) KeyMetadata(key []byte) *DataFileBuilder {
	cloned := slices.Clone(key)
	b.d.Key = &cloned

	return b
}

// SplitOffsets sets the split offsets for the data file.
func (b *DataFileBuilder) SplitOffsets(offsets []int64) *DataFileBuilder {
	cloned := slices.Clone(offsets)
	b.d.Splits = &cloned

	return b
}

// EqualityFieldIDs sets the equality field ids for the data file.
func (b *DataFileBuilder) EqualityFieldIDs(ids []int) *DataFileBuilder {
	cloned := slices.Clone(ids)
	b.d.EqualityIDs = &cloned

	return b
}

// SortOrderID sets the sort order id for the data file.
func (b *DataFileBuilder) SortOrderID(id int) *DataFileBuilder {
	b.d.SortOrder = &id

	return b
}

func (b *DataFileBuilder) FirstRowID(id int64) *DataFileBuilder {
	b.d.FirstRowIDField = &id

	return b
}

func (b *DataFileBuilder) ReferencedDataFile(path string) *DataFileBuilder {
	b.d.ReferencedDataFileField = &path

	return b
}

func (b *DataFileBuilder) ContentOffset(offset int64) *DataFileBuilder {
	b.d.ContentOffsetField = &offset

	return b
}

func (b *DataFileBuilder) ContentSizeInBytes(size int64) *DataFileBuilder {
	b.d.ContentSizeInBytesField = &size

	return b
}

func (b *DataFileBuilder) Build() DataFile {
	return b.d
}

// DataFile is the interface for reading the information about a
// given data file indicated by an entry in a manifest list.
type DataFile interface {
	// ContentType is the type of the content stored by the data file,
	// either Data, Equality deletes, or Position deletes. All v1 files
	// are Data files.
	ContentType() ManifestEntryContent
	// FilePath is the full URI for the file, complete with FS scheme.
	FilePath() string
	// FileFormat is the format of the data file, AVRO, Orc, or Parquet.
	FileFormat() FileFormat
	// Partition returns a mapping of field id to partition value for
	// each of the partition spec's fields.
	Partition() map[int]any
	// Count returns the number of records in this file.
	Count() int64
	// FileSizeBytes is the total file size in bytes.
	FileSizeBytes() int64
	// ColumnSizes is a mapping from column id to the total size on disk
	// of all regions that store the column. Does not include bytes
	// necessary to read other columns, like footers. Map will be nil for
	// row-oriented formats (avro).
	ColumnSizes() map[int]int64
	// ValueCounts is a mapping from column id to the number of values
	// in the column, including null and NaN values.
	ValueCounts() map[int]int64
	// NullValueCounts is a mapping from column id to the number of
	// null values in the column.
	NullValueCounts() map[int]int64
	// NaNValueCounts is a mapping from column id to the number of NaN
	// values in the column.
	NaNValueCounts() map[int]int64
	// DistictValueCounts is a mapping from column id to the number of
	// distinct values in the column. Distinct counts must be derived
	// using values in the file by counting or using sketches, but not
	// using methods like merging existing distinct counts.
	DistinctValueCounts() map[int]int64
	// LowerBoundValues is a mapping from column id to the lower bounded
	// value of the column, serialized as binary. Each value in the column
	// must be less than or requal to all non-null, non-NaN values in the
	// column for the file.
	LowerBoundValues() map[int][]byte
	// UpperBoundValues is a mapping from column id to the upper bounded
	// value of the column, serialized as binary. Each value in the column
	// must be greater than or equal to all non-null, non-NaN values in
	// the column for the file.
	UpperBoundValues() map[int][]byte
	// KeyMetadata is implementation-specific key metadata for encryption.
	KeyMetadata() []byte
	// SplitOffsets are the split offsets for the data file. For example,
	// all row group offsets in a Parquet file. Must be sorted ascending.
	SplitOffsets() []int64
	// EqualityFieldIDs are used to determine row equality in equality
	// delete files. It is required when the content type is
	// EntryContentEqDeletes.
	EqualityFieldIDs() []int
	// SortOrderID returns the id representing the sort order for this
	// file, or nil if there is no sort order.
	SortOrderID() *int
	// SpecID returns the partition spec id for this data file, inherited
	// from the manifest that the data file was read from
	SpecID() int32
	// FirstRowID returns the first row ID for this data file ( v3+ only )
	FirstRowID() *int64
	// ReferencedDataFile returns the location of the data file that deletion vector reference
	ReferencedDataFile() *string
	// ContentOffset returns the offset in the file where the content starts ( v3+ only )
	ContentOffset() *int64
	// ContentSizeInBytes returns the length of referenced contented stored in the file (v3+ only)
	ContentSizeInBytes() *int64
}

// ManifestEntry is an interface for both v1 and v2 manifest entries.
type ManifestEntry interface {
	// Status returns the type of the file tracked by this entry.
	// Whether entries with EntryStatusDELETED are returned depends on the
	// caller's discardDeleted option.
	Status() ManifestEntryStatus
	// SnapshotID is the id where the file was added, or deleted,
	// if null it is inherited from the manifest list.
	SnapshotID() int64
	// SequenceNum returns the data sequence number of the file.
	// If it was null and the status is EntryStatusADDED then it
	// is inherited from the manifest list.
	SequenceNum() int64
	// FileSequenceNum returns the file sequence number indicating
	// when the file was added. If it was null and the status is
	// EntryStatusADDED then it is inherited from the manifest list.
	FileSequenceNum() *int64
	// DataFile provides the information about the data file indicated
	// by this manifest entry.
	DataFile() DataFile

	inherit(manifest ManifestFile)

	wrap(status ManifestEntryStatus, snapshotID, seqNum, fileSeqNum *int64, datafile DataFile) ManifestEntry
}

var PositionalDeleteSchema = NewSchema(0,
	NestedField{ID: 2147483546, Type: PrimitiveTypes.String, Name: "file_path", Required: true},
	NestedField{ID: 2147483545, Type: PrimitiveTypes.Int64, Name: "pos", Required: true},
)
