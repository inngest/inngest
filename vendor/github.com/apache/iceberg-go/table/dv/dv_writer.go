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

package dv

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"strconv"

	"github.com/apache/iceberg-go"
	iceio "github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/puffin"
)

// SpecResolver looks up a PartitionSpec by its id, mirroring
// Metadata.PartitionSpecByID. Returns nil if the id is not known.
type SpecResolver func(specID int32) *iceberg.PartitionSpec

// dvEntry holds the per-data-file state a DV manifest entry needs. specID and
// partitionData come from the data file's own manifest entry; the spec itself
// is resolved at Flush time via the writer's SpecResolver, matching how the
// partitioned Parquet writer consults metadata.PartitionSpecByID.
type dvEntry struct {
	bitmap        *RoaringPositionBitmap
	specID        int32
	partitionData map[int]any
}

// DVWriter accumulates deletion positions per data file and flushes them as
// a single Puffin file containing one deletion-vector-v1 blob per data file.
// The returned DataFile entries are ready for RowDelta.AddDeletes().
type DVWriter struct {
	fs       iceio.WriteFileIO
	specByID SpecResolver
	entries  map[string]*dvEntry
	order    []string
}

// NewDVWriter creates a DVWriter backed by the given writable filesystem.
// specByID resolves PartitionSpec values at Flush time; typically the
// caller passes Metadata.PartitionSpecByID directly (wrapped to convert
// int → int32). The unpartitioned path can pass a resolver that returns
// iceberg.UnpartitionedSpec for id 0.
func NewDVWriter(fs iceio.WriteFileIO, specByID SpecResolver) *DVWriter {
	return &DVWriter{
		fs:       fs,
		specByID: specByID,
		entries:  make(map[string]*dvEntry),
	}
}

func (w *DVWriter) getOrCreateEntry(dataFilePath string, specID int32, partitionData map[int]any) *dvEntry {
	entry, ok := w.entries[dataFilePath]
	if ok {
		return entry
	}

	// Defensive copy of partitionData on capture so a caller that
	// mutates or reuses the map between calls and Flush can't silently
	// corrupt the entry. Mirrors Java's StructLikeUtil.copy(partition)
	// in BaseDVFileWriter.Deletes. partitionData=nil round-trips as
	// nil (maps.Clone returns nil on nil input). specID is a value
	// type so no copy is needed.
	entry = &dvEntry{
		bitmap:        NewRoaringPositionBitmap(),
		specID:        specID,
		partitionData: maps.Clone(partitionData),
	}
	w.entries[dataFilePath] = entry
	w.order = append(w.order, dataFilePath)

	return entry
}

// Add accumulates positions to delete for a given data file. specID and
// partitionData come from the data file's own manifest entry (typically
// partitionContext.specID + partitionContext.partitionData on the caller
// side) and are propagated to the output DV manifest entry, so partitioned
// tables produce DV entries with the correct spec id and partition record.
// Positions are deduplicated via the underlying roaring bitmap.
//
// partitionData keys must be partition field IDs (PartitionField.FieldID),
// not source column IDs. NewDataFileBuilder iterates spec.Fields() and
// re-keys by field name using these IDs; wrong keys silently produce an
// empty partition record on the output DataFile.
//
// First Add for a given dataFilePath captures specID and partitionData on
// the entry; later Adds for the same path append positions only and ignore
// the new specID/partitionData args. This mirrors Java's BaseDVFileWriter,
// which stores partition metadata via computeIfAbsent on the same key.
// Callers must not pass conflicting partition values across Adds for the
// same data file — the writer trusts the first-Add values for the rest of
// the writer's life.
// Add validates that every position is non-negative; negative positions are
// rejected and returned as errors to avoid silently writing malformed
// deletion-vector metadata.
func (w *DVWriter) Add(dataFilePath string, positions []int64, specID int32, partitionData map[int]any) error {
	if len(positions) == 0 {
		return nil
	}

	for _, pos := range positions {
		if pos < 0 {
			return fmt.Errorf("%w: invalid deletion position %d for %q: positions must be >= 0",
				iceberg.ErrInvalidArgument, pos, dataFilePath)
		}
	}

	entry := w.getOrCreateEntry(dataFilePath, specID, partitionData)

	for _, pos := range positions {
		entry.bitmap.Set(uint64(pos))
	}

	return nil
}

// AddPosition accumulates one position to delete for a given data file.
// It has the same first-write partition metadata and per-call validation
// semantics as Add, while avoiding Add's batch loop for callers that process
// one deleted row at a time. This mirrors Java's per-position delete operation.
// AddPosition calls are independent rather than transactional as a group. If
// a later call fails, positions added by earlier successful calls remain in
// the writer.
func (w *DVWriter) AddPosition(dataFilePath string, position int64, specID int32, partitionData map[int]any) error {
	if position < 0 {
		return fmt.Errorf("%w: invalid deletion position %d for %q: positions must be >= 0",
			iceberg.ErrInvalidArgument, position, dataFilePath)
	}

	entry := w.getOrCreateEntry(dataFilePath, specID, partitionData)

	entry.bitmap.Set(uint64(position))

	return nil
}

// Load seeds the writer with an already-written deletion vector for a data
// file so subsequent Add or AddPosition calls merge new positions into it.
// The spec permits at most one DV per data file per snapshot. When a data file
// that already has a DV receives more deletes, the old positions must be
// carried into the replacement DV (and the old DV superseded) rather than dropped.
//
// Load is intended to be called before any Add or AddPosition call for the same
// path: on the first call for a path it captures specID and partitionData
// exactly as Add would and unions the supplied bitmap into a fresh entry. If an
// entry already exists (because Add, AddPosition, or Load ran first for this
// path), Load only unions the bitmap in
// and the earlier specID/partitionData are kept — the later values are ignored,
// mirroring Add's first-write-wins behavior. Callers must not pass conflicting
// partition values across calls for the same path.
//
// bitmap must be non-nil; its set positions are copied and the caller retains
// ownership. (A nil bitmap would register a cardinality-0 entry, which is never
// useful — collectExistingDVs always supplies a non-nil bitmap read from the
// existing DV.)
func (w *DVWriter) Load(dataFilePath string, bitmap *RoaringPositionBitmap, specID int32, partitionData map[int]any) {
	entry := w.getOrCreateEntry(dataFilePath, specID, partitionData)

	entry.bitmap.Or(bitmap)
}

// Flush writes one Puffin file containing one blob per data file, and returns
// manifest entries ready for RowDelta.AddDeletes(). Each output DataFile
// carries the partition spec and partition record of the data file it
// references, with the spec resolved via the writer's SpecResolver at Flush
// time. A specID with no corresponding spec is a programming error and is
// surfaced as an error here rather than producing a malformed DataFile.
//
// The location parameter is the full path (including filename) for the
// Puffin file to create. The caller is responsible for generating a unique
// path within the table's metadata directory.
func (w *DVWriter) Flush(_ context.Context, location string) ([]iceberg.DataFile, error) {
	if len(w.entries) == 0 {
		return nil, nil
	}

	var buf bytes.Buffer

	pw, err := puffin.NewWriter(&buf)
	if err != nil {
		return nil, fmt.Errorf("create puffin writer: %w", err)
	}

	type blobResult struct {
		dataFilePath string
		entry        *dvEntry
		meta         puffin.BlobMetadata
		cardinality  int64
	}

	results := make([]blobResult, 0, len(w.order))

	for _, dataFilePath := range w.order {
		entry := w.entries[dataFilePath]
		dvBytes, err := SerializeDV(entry.bitmap)
		if err != nil {
			return nil, fmt.Errorf("serialize DV for %s: %w", dataFilePath, err)
		}

		cardinality := entry.bitmap.Cardinality()
		meta, err := pw.AddBlob(puffin.BlobMetadataInput{
			Type:           puffin.BlobTypeDeletionVector,
			SnapshotID:     -1,
			SequenceNumber: -1,
			Fields:         []int32{},
			Properties: map[string]string{
				dvCardinalityProperty:        strconv.FormatInt(cardinality, 10),
				dvReferencedDataFileProperty: dataFilePath,
			},
		}, dvBytes)
		if err != nil {
			return nil, fmt.Errorf("add DV blob for %s: %w", dataFilePath, err)
		}

		results = append(results, blobResult{
			dataFilePath: dataFilePath,
			entry:        entry,
			meta:         meta,
			cardinality:  cardinality,
		})
	}

	if err := pw.Finish(); err != nil {
		return nil, fmt.Errorf("finish puffin file: %w", err)
	}

	fileBytes := buf.Bytes()
	fileSize := int64(len(fileBytes))

	dataFiles := make([]iceberg.DataFile, 0, len(results))
	for _, r := range results {
		spec := w.specByID(r.entry.specID)
		if spec == nil {
			return nil, fmt.Errorf("unknown partition spec id %d for data file %s",
				r.entry.specID, r.dataFilePath)
		}

		builder, err := iceberg.NewDataFileBuilder(
			*spec,
			iceberg.EntryContentPosDeletes,
			location,
			iceberg.PuffinFile,
			r.entry.partitionData,
			nil, nil,
			r.cardinality,
			fileSize,
		)
		if err != nil {
			return nil, fmt.Errorf("build DataFile for %s: %w", r.dataFilePath, err)
		}

		df := builder.
			ReferencedDataFile(r.dataFilePath).
			ContentOffset(r.meta.Offset).
			ContentSizeInBytes(r.meta.Length).
			Build()

		dataFiles = append(dataFiles, df)
	}

	if err := w.fs.WriteFile(location, fileBytes); err != nil {
		return nil, fmt.Errorf("write DV puffin file: %w", err)
	}

	w.entries = make(map[string]*dvEntry)
	w.order = nil

	return dataFiles, nil
}
