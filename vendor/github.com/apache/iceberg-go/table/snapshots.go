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
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/internal"
	iceio "github.com/apache/iceberg-go/io"
)

type Operation string

const (
	OpAppend    Operation = "append"
	OpReplace   Operation = "replace"
	OpOverwrite Operation = "overwrite"
	OpDelete    Operation = "delete"
)

var (
	ErrInvalidOperation  = errors.New("invalid operation value")
	ErrMissingOperation  = errors.New("missing operation key")
	ErrInvalidRowLineage = errors.New("invalid row lineage")
)

// ValidOperation ensures that a given string is one of the valid operation
// types: append,replace,overwrite,delete
func ValidOperation(s string) (Operation, error) {
	switch s {
	case "append", "replace", "overwrite", "delete":
		return Operation(s), nil
	}

	return "", fmt.Errorf("%w: found '%s'", ErrInvalidOperation, s)
}

const (
	operationKey = "operation"

	addedDataFilesKey         = "added-data-files"
	addedDeleteFilesKey       = "added-delete-files"
	addedEqDeletesKey         = "added-equality-deletes"
	addedFileSizeKey          = "added-files-size"
	addedPosDeletesKey        = "added-position-deletes"
	addedPosDeleteFilesKey    = "added-position-delete-files"
	addedRecordsKey           = "added-records"
	addedEqDeleteFilesKey     = "added-equality-delete-files"
	deletedDataFilesKey       = "deleted-data-files"
	deletedRecordsKey         = "deleted-records"
	removedDeleteFilesKey     = "removed-delete-files"
	removedEqDeletesKey       = "removed-equality-deletes"
	removedEqDeleteFilesKey   = "removed-equality-delete-files"
	removedFileSizeKey        = "removed-files-size"
	removedPosDeletesKey      = "removed-position-deletes"
	removedPosDeleteFilesKey  = "removed-position-delete-files"
	totalEqDeletesKey         = "total-equality-deletes"
	totalPosDeletesKey        = "total-position-deletes"
	totalDataFilesKey         = "total-data-files"
	totalDeleteFilesKey       = "total-delete-files"
	totalRecordsKey           = "total-records"
	totalFileSizeKey          = "total-files-size"
	changedPartitionCountProp = "changed-partition-count"
	changedPartitionPrefix    = "partitions."
)

type updateMetrics struct {
	addedFileSize         int64
	removedFileSize       int64
	addedDataFiles        int64
	removedDataFiles      int64
	addedEqDeleteFiles    int64
	removedEqDeleteFiles  int64
	addedPosDeleteFiles   int64
	removedPosDeleteFiles int64
	addedDeleteFiles      int64
	removedDeleteFiles    int64
	addedRecords          int64
	deletedRecords        int64
	addedPosDeletes       int64
	removedPosDeletes     int64
	addedEqDeletes        int64
	removedEqDeletes      int64
}

func (m *updateMetrics) addDataFile(df iceberg.DataFile) error {
	m.addedFileSize += df.FileSizeBytes()
	switch df.ContentType() {
	case iceberg.EntryContentData:
		m.addedDataFiles++
		m.addedRecords += df.Count()
	case iceberg.EntryContentPosDeletes:
		m.addedDeleteFiles++
		m.addedPosDeleteFiles++
		m.addedPosDeletes += df.Count()
	case iceberg.EntryContentEqDeletes:
		m.addedDeleteFiles++
		m.addedEqDeleteFiles++
		m.addedEqDeletes += df.Count()
	default:
		return fmt.Errorf("unknown data file content: %s", df.ContentType())
	}

	return nil
}

func (m *updateMetrics) removeFile(df iceberg.DataFile) error {
	m.removedFileSize += df.FileSizeBytes()
	switch df.ContentType() {
	case iceberg.EntryContentData:
		m.removedDataFiles++
		m.deletedRecords += df.Count()
	case iceberg.EntryContentPosDeletes:
		m.removedDeleteFiles++
		m.removedPosDeleteFiles++
		m.removedPosDeletes += df.Count()
	case iceberg.EntryContentEqDeletes:
		m.removedDeleteFiles++
		m.removedEqDeleteFiles++
		m.removedEqDeletes += df.Count()
	default:
		return fmt.Errorf("unknown data file content: %s", df.ContentType())
	}

	return nil
}

func setWhenPositive(props iceberg.Properties, key string, value int64) {
	if value > 0 {
		props[key] = strconv.FormatInt(value, 10)
	}
}

func (m *updateMetrics) toProps() iceberg.Properties {
	props := iceberg.Properties{}
	setWhenPositive(props, addedFileSizeKey, m.addedFileSize)
	setWhenPositive(props, removedFileSizeKey, m.removedFileSize)
	setWhenPositive(props, addedDataFilesKey, m.addedDataFiles)
	setWhenPositive(props, deletedDataFilesKey, m.removedDataFiles)
	setWhenPositive(props, addedEqDeleteFilesKey, m.addedEqDeleteFiles)
	setWhenPositive(props, removedEqDeleteFilesKey, m.removedEqDeleteFiles)
	setWhenPositive(props, addedPosDeleteFilesKey, m.addedPosDeleteFiles)
	setWhenPositive(props, removedPosDeleteFilesKey, m.removedPosDeleteFiles)
	setWhenPositive(props, addedDeleteFilesKey, m.addedDeleteFiles)
	setWhenPositive(props, removedDeleteFilesKey, m.removedDeleteFiles)
	setWhenPositive(props, addedRecordsKey, m.addedRecords)
	setWhenPositive(props, deletedRecordsKey, m.deletedRecords)
	setWhenPositive(props, addedPosDeletesKey, m.addedPosDeletes)
	setWhenPositive(props, removedPosDeletesKey, m.removedPosDeletes)
	setWhenPositive(props, addedEqDeletesKey, m.addedEqDeletes)
	setWhenPositive(props, removedEqDeletesKey, m.removedEqDeletes)

	return props
}

// Summary stores the summary information for a snapshot indicating
// the operation that created the snapshot, and various properties
// which might exist in the summary.
type Summary struct {
	Operation  Operation
	Properties iceberg.Properties
}

func (s *Summary) String() string {
	out := string(s.Operation)
	if s.Properties != nil {
		data, _ := json.Marshal(s.Properties)
		out += ", " + string(data)
	}

	return out
}

func (s *Summary) Equals(other *Summary) bool {
	if s == other {
		return true
	}

	if s == nil || other == nil {
		return false
	}

	if s.Operation != other.Operation {
		return false
	}

	if len(s.Properties) == 0 && len(other.Properties) == 0 {
		return true
	}

	return maps.Equal(s.Properties, other.Properties)
}

func (s *Summary) UnmarshalJSON(b []byte) (err error) {
	alias := map[string]string{}
	if err = json.Unmarshal(b, &alias); err != nil {
		return err
	}

	op, ok := alias[operationKey]
	if !ok {
		// Match Java's compatibility behavior for older snapshots that contain
		// summary properties but omit the required operation.
		s.Operation = ""
		if len(alias) > 0 {
			s.Operation = OpOverwrite
		}
		s.Properties = alias

		return nil
	}

	if op == "" {
		return fmt.Errorf("%w: found empty operation", ErrInvalidOperation)
	}
	s.Operation = Operation(op)

	delete(alias, operationKey)
	s.Properties = alias

	return nil
}

func (s *Summary) MarshalJSON() ([]byte, error) {
	props := maps.Clone(s.Properties)
	if s.Operation != "" {
		if props == nil {
			props = make(map[string]string)
		}
		props[operationKey] = string(s.Operation)
	}

	return json.Marshal(props)
}

type Snapshot struct {
	SnapshotID        int64    `json:"snapshot-id"`
	ParentSnapshotID  *int64   `json:"parent-snapshot-id,omitempty"`
	SequenceNumber    int64    `json:"sequence-number"`
	TimestampMs       int64    `json:"timestamp-ms"`
	ManifestList      string   `json:"manifest-list,omitempty"`
	ManifestLocations []string `json:"manifests,omitempty"` // V1: Embedded manifest locations
	Summary           *Summary `json:"summary,omitempty"`
	SchemaID          *int     `json:"schema-id,omitempty"`
	FirstRowID        *int64   `json:"first-row-id,omitempty"` // V3: Starting row ID for this snapshot
	AddedRows         *int64   `json:"added-rows,omitempty"`   // V3: Number of rows added by this snapshot
}

func (s *Snapshot) UnmarshalJSON(data []byte) error {
	type Alias Snapshot
	var next Alias
	// snapshot-id and timestamp-ms are required by the spec for every format
	// version. Decode them through pointers so an absent or null field stays
	// distinguishable from an explicit zero, which would otherwise be accepted
	// as a valid identity or as the Unix epoch.
	aux := struct {
		SnapshotID  *int64 `json:"snapshot-id"`
		TimestampMs *int64 `json:"timestamp-ms"`
		*Alias
	}{Alias: &next}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.SnapshotID == nil {
		return fmt.Errorf("%w: snapshot-id is absent or null", ErrInvalidMetadata)
	}
	if aux.TimestampMs == nil {
		return fmt.Errorf("%w: timestamp-ms is absent or null", ErrInvalidMetadata)
	}
	// The pointer fields shadow Alias, so json.Unmarshal does not populate next.
	next.SnapshotID, next.TimestampMs = *aux.SnapshotID, *aux.TimestampMs

	if next.ManifestList != "" {
		next.ManifestLocations = nil
	}

	*s = Snapshot(next)

	return nil
}

func (s Snapshot) MarshalJSON() ([]byte, error) {
	type Alias Snapshot
	if s.ManifestList != "" {
		// The manifest list is authoritative when both legacy and current
		// representations are present. Do not write the V1 fallback alongside it.
		s.ManifestLocations = nil
	}

	data, err := json.Marshal((*Alias)(&s))
	if err != nil {
		return nil, err
	}
	if s.ManifestLocations == nil {
		return data, nil
	}

	// A non-nil slice represents the V1 embedded-manifest form, including an
	// explicitly empty manifests array. Match Java by omitting the default
	// sequence number while preserving any positive value accepted on read.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	locations, err := json.Marshal(s.ManifestLocations)
	if err != nil {
		return nil, err
	}
	fields["manifests"] = locations
	if s.SequenceNumber <= 0 {
		delete(fields, "sequence-number")
	}

	return json.Marshal(fields)
}

func (s Snapshot) String() string {
	var op, parent, schema string

	if s.Summary != nil {
		op = s.Summary.String() + ": "
	}
	if s.ParentSnapshotID != nil {
		parent = ", parent_id=" + strconv.FormatInt(*s.ParentSnapshotID, 10)
	}
	if s.SchemaID != nil {
		schema = ", schema_id=" + strconv.Itoa(*s.SchemaID)
	}

	return fmt.Sprintf("%sid=%d%s%s, sequence_number=%d, timestamp_ms=%d, manifest_list=%s",
		op, s.SnapshotID, parent, schema, s.SequenceNumber, s.TimestampMs, s.ManifestList)
}

func (s Snapshot) Equals(other Snapshot) bool {
	manifestLocationsEqual := (s.ManifestLocations == nil) == (other.ManifestLocations == nil) &&
		slices.Equal(s.ManifestLocations, other.ManifestLocations)

	switch {
	case s.ParentSnapshotID == nil && other.ParentSnapshotID != nil:
		fallthrough
	case s.ParentSnapshotID != nil && other.ParentSnapshotID == nil:
		fallthrough
	case s.SchemaID == nil && other.SchemaID != nil:
		fallthrough
	case s.SchemaID != nil && other.SchemaID == nil:
		fallthrough
	case s.FirstRowID == nil && other.FirstRowID != nil:
		fallthrough
	case s.FirstRowID != nil && other.FirstRowID == nil:
		fallthrough
	case s.AddedRows == nil && other.AddedRows != nil:
		fallthrough
	case s.AddedRows != nil && other.AddedRows == nil:
		return false
	}

	return s.SnapshotID == other.SnapshotID &&
		((s.ParentSnapshotID == other.ParentSnapshotID) || (*s.ParentSnapshotID == *other.ParentSnapshotID)) &&
		((s.SchemaID == other.SchemaID) || (*s.SchemaID == *other.SchemaID)) &&
		((s.FirstRowID == other.FirstRowID) || (*s.FirstRowID == *other.FirstRowID)) &&
		((s.AddedRows == other.AddedRows) || (*s.AddedRows == *other.AddedRows)) &&
		s.SequenceNumber == other.SequenceNumber &&
		s.TimestampMs == other.TimestampMs &&
		s.ManifestList == other.ManifestList &&
		manifestLocationsEqual &&
		s.Summary.Equals(other.Summary)
}

func (s Snapshot) ValidateRowLineage() error {
	if s.FirstRowID != nil && s.AddedRows == nil {
		return fmt.Errorf("%w: added-rows is required when first-row-id is set", ErrInvalidRowLineage)
	}
	if s.AddedRows != nil && *s.AddedRows < 0 {
		return fmt.Errorf("%w: added-rows cannot be negative: %d", ErrInvalidRowLineage, *s.AddedRows)
	}
	if s.FirstRowID != nil && *s.FirstRowID < 0 {
		return fmt.Errorf("%w: first-row-id cannot be negative: %d", ErrInvalidRowLineage, *s.FirstRowID)
	}

	return nil
}

func (s Snapshot) Manifests(fio iceio.IO) (_ []iceberg.ManifestFile, err error) {
	if s.ManifestList != "" {
		f, err := fio.Open(s.ManifestList)
		if err != nil {
			return nil, fmt.Errorf("could not open manifest file: %w", err)
		}
		defer internal.CheckedClose(f, &err)

		return iceberg.ReadManifestList(f)
	}
	if s.ManifestLocations != nil {
		manifests := make([]iceberg.ManifestFile, len(s.ManifestLocations))
		for i, path := range s.ManifestLocations {
			length, err := embeddedManifestLength(fio, path)
			if err != nil {
				return nil, err
			}

			manifests[i] = iceberg.NewManifestFile(1, path, length, 0, s.SnapshotID).
				AddedFiles(-1).
				ExistingFiles(-1).
				DeletedFiles(-1).
				AddedRows(-1).
				ExistingRows(-1).
				DeletedRows(-1).
				Build()
		}

		return manifests, nil
	}

	return nil, nil
}

func embeddedManifestLength(fio iceio.IO, path string) (_ int64, err error) {
	if fio == nil {
		return 0, fmt.Errorf("cannot stat embedded manifest %q without IO", path)
	}

	if statIO, ok := fio.(iceio.StatIO); ok {
		info, err := statIO.Stat(path)
		if err != nil {
			return 0, fmt.Errorf("could not stat embedded manifest %q: %w", path, err)
		}

		return info.Size(), nil
	}

	f, err := fio.Open(path)
	if err != nil {
		return 0, fmt.Errorf("could not open embedded manifest %q: %w", path, err)
	}
	defer internal.CheckedClose(f, &err)

	info, err := f.Stat()
	if err != nil {
		return 0, fmt.Errorf("could not stat embedded manifest %q: %w", path, err)
	}

	return info.Size(), nil
}

func (s Snapshot) dataFiles(fio iceio.IO, fileFilter set[iceberg.ManifestEntryContent]) iter.Seq2[iceberg.DataFile, error] {
	return func(yield func(iceberg.DataFile, error) bool) {
		manifests, err := s.Manifests(fio)
		if err != nil {
			yield(nil, err)

			return
		}

		for _, m := range manifests {
			// Discard DELETED entries: they are tombstones recording a
			// removal, not files reachable from this snapshot. Yielding
			// them would make existence and duplicate checks treat a
			// file deleted by this snapshot as still live.
			for entry, err := range m.Entries(fio, true) {
				if err != nil {
					yield(nil, err)

					return
				}

				if fileFilter != nil {
					if _, ok := fileFilter[entry.DataFile().ContentType()]; !ok {
						continue
					}
				}
				if !yield(entry.DataFile(), nil) {
					return
				}
			}
		}
	}
}

// entries iterates every manifest entry in the snapshot, filtered by
// manifest content (data or deletes). It is the entry-level analog
// of dataFiles and exposes the underlying ManifestEntry so callers
// can inspect Status / SnapshotID / DataFile — needed by conflict
// validation where attribution of an entry to a specific snapshot
// matters.
//
// manifestContent < 0 yields entries across both data and delete
// manifests. Set discardDeleted to skip DELETED entries: they are files
// already removed from the table, not files still in it.
func (s Snapshot) entries(fio iceio.IO, manifestContent iceberg.ManifestContent, discardDeleted bool) iter.Seq2[iceberg.ManifestEntry, error] {
	return func(yield func(iceberg.ManifestEntry, error) bool) {
		manifests, err := s.Manifests(fio)
		if err != nil {
			yield(nil, err)

			return
		}

		for _, m := range manifests {
			if manifestContent >= 0 && m.ManifestContent() != manifestContent {
				continue
			}

			for entry, err := range m.Entries(fio, discardDeleted) {
				if err != nil {
					yield(nil, err)

					return
				}
				if !yield(entry, nil) {
					return
				}
			}
		}
	}
}

type MetadataLogEntry struct {
	MetadataFile string `json:"metadata-file"`
	TimestampMs  int64  `json:"timestamp-ms"`
}

type SnapshotLogEntry struct {
	SnapshotID  int64 `json:"snapshot-id"`
	TimestampMs int64 `json:"timestamp-ms"`
}

// snapshotLogEntryAsOf returns the log entry with the greatest eligible
// timestamp. Snapshot-log entries can be out of chronological order when
// commits have small clock skews, so the result must not depend on iteration
// order.
func snapshotLogEntryAsOf(entries iter.Seq[SnapshotLogEntry], timestampMs int64, inclusive bool) (SnapshotLogEntry, bool) {
	var (
		best  SnapshotLogEntry
		found bool
	)

	for entry := range entries {
		eligible := entry.TimestampMs < timestampMs
		if inclusive {
			eligible = entry.TimestampMs <= timestampMs
		}

		if eligible && (!found || entry.TimestampMs > best.TimestampMs) {
			best = entry
			found = true
		}
	}

	return best, found
}

type SnapshotSummaryCollector struct {
	metrics                          updateMetrics
	partitionMetrics                 map[string]updateMetrics
	maxChangedPartitionsForSummaries int
}

func (s *SnapshotSummaryCollector) setPartitionSummaryLimit(limit int) {
	s.maxChangedPartitionsForSummaries = limit
}

func (s *SnapshotSummaryCollector) updatePartitionMetrics(partitionPath string, df iceberg.DataFile, isAddFile bool) error {
	if s.partitionMetrics == nil {
		s.partitionMetrics = make(map[string]updateMetrics)
	}

	metrics := s.partitionMetrics[partitionPath]
	if isAddFile {
		if err := metrics.addDataFile(df); err != nil {
			return err
		}
	} else {
		if err := metrics.removeFile(df); err != nil {
			return err
		}
	}
	s.partitionMetrics[partitionPath] = metrics

	return nil
}

func (s *SnapshotSummaryCollector) addFile(df iceberg.DataFile, sc *iceberg.Schema, spec iceberg.PartitionSpec) error {
	if err := s.metrics.addDataFile(df); err != nil {
		return err
	}

	partition := dataFilePartition(df)
	if len(partition) > 0 {
		partitionPath := spec.PartitionToPath(
			newPartitionRecord(partition, spec.PartitionType(sc)), sc)

		return s.updatePartitionMetrics(partitionPath, df, true)
	}

	return nil
}

func (s *SnapshotSummaryCollector) removeFile(df iceberg.DataFile, sc *iceberg.Schema, spec iceberg.PartitionSpec) error {
	if err := s.metrics.removeFile(df); err != nil {
		return err
	}

	partition := dataFilePartition(df)
	if len(partition) > 0 {
		partitionPath := spec.PartitionToPath(
			newPartitionRecord(partition, spec.PartitionType(sc)), sc)

		return s.updatePartitionMetrics(partitionPath, df, false)
	}

	return nil
}

func (s *SnapshotSummaryCollector) partitionSummary(metrics *updateMetrics) string {
	props := metrics.toProps()

	return strings.Join(slices.Sorted(func(yield func(s string) bool) {
		for k, v := range props {
			if !yield(fmt.Sprintf("%s=%s", k, v)) {
				return
			}
		}
	}), ",")
}

func (s *SnapshotSummaryCollector) build() iceberg.Properties {
	props := s.metrics.toProps()
	changedPartitionsSize := len(s.partitionMetrics)
	setWhenPositive(props, changedPartitionCountProp, int64(changedPartitionsSize))
	if changedPartitionsSize <= s.maxChangedPartitionsForSummaries {
		for partPath, updateMetricsPart := range s.partitionMetrics {
			if summary := s.partitionSummary(&updateMetricsPart); len(summary) > 0 {
				props[changedPartitionPrefix+partPath] = summary
			}
		}
	}

	return props
}

func updateSnapshotSummaries(sum Summary, previous iceberg.Properties) (Summary, error) {
	switch sum.Operation {
	case OpAppend, OpOverwrite, OpDelete, OpReplace:
	default:
		return sum, fmt.Errorf("%w: operation: %s", iceberg.ErrNotImplemented, sum.Operation)
	}

	if sum.Properties == nil {
		sum.Properties = make(iceberg.Properties)
	}

	if previous == nil {
		previous = iceberg.Properties{
			totalDataFilesKey:   "0",
			totalDeleteFilesKey: "0",
			totalRecordsKey:     "0",
			totalFileSizeKey:    "0",
			totalPosDeletesKey:  "0",
			totalEqDeletesKey:   "0",
		}
	}

	updateTotals := func(totalProp, addedProp, removedProp string) {
		newTotal := previous.GetInt64(totalProp, 0)
		newTotal += sum.Properties.GetInt64(addedProp, 0)
		newTotal -= sum.Properties.GetInt64(removedProp, 0)

		if newTotal >= 0 {
			sum.Properties[totalProp] = strconv.FormatInt(newTotal, 10)
		}
	}

	updateTotals(totalDataFilesKey, addedDataFilesKey, deletedDataFilesKey)
	updateTotals(totalDeleteFilesKey, addedDeleteFilesKey, removedDeleteFilesKey)
	updateTotals(totalRecordsKey, addedRecordsKey, deletedRecordsKey)
	updateTotals(totalFileSizeKey, addedFileSizeKey, removedFileSizeKey)
	updateTotals(totalPosDeletesKey, addedPosDeletesKey, removedPosDeletesKey)
	updateTotals(totalEqDeletesKey, addedEqDeletesKey, removedEqDeletesKey)

	return sum, nil
}
