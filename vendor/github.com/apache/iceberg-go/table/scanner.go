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
	"iter"
	"maps"
	"math"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/iceberg-go"
	iceberginternal "github.com/apache/iceberg-go/internal"
	"github.com/apache/iceberg-go/internal/scanmetrics"
	"github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/metrics"
	"golang.org/x/sync/errgroup"
)

const ScanNoLimit = -1

type keyDefaultMap[K comparable, V any] struct {
	defaultFactory func(K) V
	data           map[K]V

	mx sync.RWMutex
}

type keyDefaultMapErr[K comparable, V any] struct {
	defaultFactory func(K) (V, error)
	data           map[K]keyDefaultValueErr[V]

	mx sync.RWMutex
}

type keyDefaultValueErr[V any] struct {
	value V
	err   error
}

func (k *keyDefaultMap[K, V]) Get(key K) V {
	k.mx.RLock()
	if v, ok := k.data[key]; ok {
		k.mx.RUnlock()

		return v
	}

	k.mx.RUnlock()
	k.mx.Lock()
	defer k.mx.Unlock()

	// race check between RLock and Lock
	if v, ok := k.data[key]; ok {
		return v
	}

	v := k.defaultFactory(key)
	k.data[key] = v

	return v
}

func (k *keyDefaultMapErr[K, V]) Get(key K) (V, error) {
	k.mx.RLock()
	if v, ok := k.data[key]; ok {
		k.mx.RUnlock()

		return v.value, v.err
	}

	k.mx.RUnlock()
	k.mx.Lock()
	defer k.mx.Unlock()

	// race check between RLock and Lock
	if v, ok := k.data[key]; ok {
		return v.value, v.err
	}

	value, err := k.defaultFactory(key)
	k.data[key] = keyDefaultValueErr[V]{value: value, err: err}

	return value, err
}

func newKeyDefaultMap[K comparable, V any](factory func(K) V) *keyDefaultMap[K, V] {
	return &keyDefaultMap[K, V]{
		data:           make(map[K]V),
		defaultFactory: factory,
	}
}

// newKeyDefaultMapWrapErr memoizes both successful values and deterministic
// factory errors, so the same failing key is not retried on subsequent reads.
func newKeyDefaultMapWrapErr[K comparable, V any](factory func(K) (V, error)) *keyDefaultMapErr[K, V] {
	return &keyDefaultMapErr[K, V]{
		data:           make(map[K]keyDefaultValueErr[V]),
		defaultFactory: factory,
	}
}

type partitionRecord []any

func (p partitionRecord) Size() int            { return len(p) }
func (p partitionRecord) Get(pos int) any      { return p[pos] }
func (p partitionRecord) Set(pos int, val any) { p[pos] = val }

// borrowedPartitionRecord exposes a DataFile's immutable partition map in
// partition-field order without materializing a positional record or cloning
// binary values. It is only valid for the current evaluator call.
type borrowedPartitionRecord struct {
	partition     map[int]any
	partitionType *iceberg.StructType
}

func (p borrowedPartitionRecord) Size() int {
	return len(p.partitionType.FieldList)
}

func (p borrowedPartitionRecord) Get(pos int) any {
	return p.partition[p.partitionType.FieldList[pos].ID]
}

func (borrowedPartitionRecord) Set(int, any) {
	panic("cannot set a borrowed partition record")
}

// manifestEntries holds the data, positional delete, and equality delete
// entries read from manifests.
type manifestEntries struct {
	dataEntries             []iceberg.ManifestEntry
	positionalDeleteEntries []iceberg.ManifestEntry
	equalityDeleteEntries   []iceberg.ManifestEntry
	dvEntries               []iceberg.ManifestEntry
	mu                      sync.Mutex
}

type classifiedManifestEntries struct {
	dataEntries             []iceberg.ManifestEntry
	positionalDeleteEntries []iceberg.ManifestEntry
	equalityDeleteEntries   []iceberg.ManifestEntry
	dvEntries               []iceberg.ManifestEntry
}

type manifestEntryKind uint8

const (
	manifestEntryData manifestEntryKind = iota
	manifestEntryPositionalDelete
	manifestEntryEqualityDelete
	manifestEntryDV
	manifestEntryKindCount
)

func newManifestEntries() *manifestEntries {
	return &manifestEntries{
		dataEntries:             make([]iceberg.ManifestEntry, 0),
		positionalDeleteEntries: make([]iceberg.ManifestEntry, 0),
		equalityDeleteEntries:   make([]iceberg.ManifestEntry, 0),
		dvEntries:               make([]iceberg.ManifestEntry, 0),
	}
}

func classifyManifestEntry(entry iceberg.ManifestEntry) (manifestEntryKind, error) {
	dataFile := entry.DataFile()
	switch dataFile.ContentType() {
	case iceberg.EntryContentData:
		return manifestEntryData, nil
	case iceberg.EntryContentPosDeletes:
		if IsDeletionVector(dataFile) {
			return manifestEntryDV, nil
		}

		return manifestEntryPositionalDelete, nil
	case iceberg.EntryContentEqDeletes:
		return manifestEntryEqualityDelete, nil
	default:
		return 0, fmt.Errorf("%w: unknown DataFileContent type (%s): %s",
			ErrInvalidMetadata, dataFile.ContentType(), entry)
	}
}

func newClassifiedManifestEntries(counts [manifestEntryKindCount]int) classifiedManifestEntries {
	return classifiedManifestEntries{
		dataEntries:             make([]iceberg.ManifestEntry, 0, counts[manifestEntryData]),
		positionalDeleteEntries: make([]iceberg.ManifestEntry, 0, counts[manifestEntryPositionalDelete]),
		equalityDeleteEntries:   make([]iceberg.ManifestEntry, 0, counts[manifestEntryEqualityDelete]),
		dvEntries:               make([]iceberg.ManifestEntry, 0, counts[manifestEntryDV]),
	}
}

func classifiedManifestEntriesForKind(kind manifestEntryKind, entries []iceberg.ManifestEntry) classifiedManifestEntries {
	classified := classifiedManifestEntries{}
	switch kind {
	case manifestEntryData:
		classified.dataEntries = entries
	case manifestEntryPositionalDelete:
		classified.positionalDeleteEntries = entries
	case manifestEntryEqualityDelete:
		classified.equalityDeleteEntries = entries
	case manifestEntryDV:
		classified.dvEntries = entries
	default:
		panic(fmt.Sprintf("unhandled manifest entry kind %d", kind))
	}

	return classified
}

func classifyManifestEntries(entries []iceberg.ManifestEntry) (classifiedManifestEntries, error) {
	if len(entries) == 0 {
		return classifiedManifestEntries{}, nil
	}

	firstKind, err := classifyManifestEntry(entries[0])
	if err != nil {
		return classifiedManifestEntries{}, err
	}

	var (
		kinds  []manifestEntryKind
		counts [manifestEntryKindCount]int
	)
	for i := 1; i < len(entries); i++ {
		kind, err := classifyManifestEntry(entries[i])
		if err != nil {
			if kinds == nil {
				return classifiedManifestEntriesForKind(firstKind, entries[:i]), err
			}

			classified := newClassifiedManifestEntries(counts)
			for j, validEntry := range entries[:i] {
				appendClassifiedManifestEntry(&classified, kinds[j], validEntry)
			}

			return classified, err
		}

		if kinds == nil && kind != firstKind {
			kinds = make([]manifestEntryKind, len(entries))
			for j := range i {
				kinds[j] = firstKind
			}
			counts[firstKind] = i
		}
		if kinds != nil {
			kinds[i] = kind
			counts[kind]++
		}
	}

	if kinds == nil {
		return classifiedManifestEntriesForKind(firstKind, entries), nil
	}

	classified := newClassifiedManifestEntries(counts)
	for i, entry := range entries {
		appendClassifiedManifestEntry(&classified, kinds[i], entry)
	}

	return classified, nil
}

func appendClassifiedManifestEntry(classified *classifiedManifestEntries, kind manifestEntryKind, entry iceberg.ManifestEntry) {
	switch kind {
	case manifestEntryData:
		classified.dataEntries = append(classified.dataEntries, entry)
	case manifestEntryPositionalDelete:
		classified.positionalDeleteEntries = append(classified.positionalDeleteEntries, entry)
	case manifestEntryEqualityDelete:
		classified.equalityDeleteEntries = append(classified.equalityDeleteEntries, entry)
	case manifestEntryDV:
		classified.dvEntries = append(classified.dvEntries, entry)
	default:
		panic(fmt.Sprintf("unhandled manifest entry kind %d", kind))
	}
}

func (m *manifestEntries) merge(entries []iceberg.ManifestEntry) error {
	classified, err := classifyManifestEntries(entries)

	// Preserve the existing partial-commit behavior on classification errors.
	// Callers discard the accumulator when merge returns an error.
	m.mu.Lock()
	defer m.mu.Unlock()

	m.dataEntries = append(m.dataEntries, classified.dataEntries...)
	m.positionalDeleteEntries = append(m.positionalDeleteEntries, classified.positionalDeleteEntries...)
	m.equalityDeleteEntries = append(m.equalityDeleteEntries, classified.equalityDeleteEntries...)
	m.dvEntries = append(m.dvEntries, classified.dvEntries...)

	return err
}

func flattenClassifiedManifestEntries(results []classifiedManifestEntries) *manifestEntries {
	var counts [manifestEntryKindCount]int
	for _, result := range results {
		counts[manifestEntryData] += len(result.dataEntries)
		counts[manifestEntryPositionalDelete] += len(result.positionalDeleteEntries)
		counts[manifestEntryEqualityDelete] += len(result.equalityDeleteEntries)
		counts[manifestEntryDV] += len(result.dvEntries)
	}

	flattened := &manifestEntries{
		dataEntries:             make([]iceberg.ManifestEntry, counts[manifestEntryData]),
		positionalDeleteEntries: make([]iceberg.ManifestEntry, counts[manifestEntryPositionalDelete]),
		equalityDeleteEntries:   make([]iceberg.ManifestEntry, counts[manifestEntryEqualityDelete]),
		dvEntries:               make([]iceberg.ManifestEntry, counts[manifestEntryDV]),
	}

	var offsets [manifestEntryKindCount]int
	for _, result := range results {
		offsets[manifestEntryData] += copy(
			flattened.dataEntries[offsets[manifestEntryData]:], result.dataEntries,
		)
		offsets[manifestEntryPositionalDelete] += copy(
			flattened.positionalDeleteEntries[offsets[manifestEntryPositionalDelete]:], result.positionalDeleteEntries,
		)
		offsets[manifestEntryEqualityDelete] += copy(
			flattened.equalityDeleteEntries[offsets[manifestEntryEqualityDelete]:], result.equalityDeleteEntries,
		)
		offsets[manifestEntryDV] += copy(
			flattened.dvEntries[offsets[manifestEntryDV]:], result.dvEntries,
		)
	}

	return flattened
}

func newPartitionRecord(partitionData map[int]any, partitionType *iceberg.StructType) partitionRecord {
	out := make(partitionRecord, len(partitionType.FieldList))
	for i, f := range partitionType.FieldList {
		value := partitionData[f.ID]
		if bytes, ok := value.([]byte); ok {
			out[i] = slices.Clone(bytes)
		} else {
			out[i] = value
		}
	}

	return out
}

// GetPartitionRecord converts a DataFile's partition map into a positional
// record ordered by the fields of the given partition struct type.
func GetPartitionRecord(dataFile iceberg.DataFile, partitionType *iceberg.StructType) iceberg.StructLike {
	return newPartitionRecord(dataFilePartition(dataFile), partitionType)
}

func openManifest(io io.IO, manifest iceberg.ManifestFile,
	partitionFilter, metricsEval func(iceberg.DataFile) (bool, error),
) ([]iceberg.ManifestEntry, error) {
	return openManifestWithProjection(io, manifest, partitionFilter, metricsEval, nil, false)
}

func openManifestWithProjection(
	io io.IO,
	manifest iceberg.ManifestFile,
	partitionFilter, metricsEval func(iceberg.DataFile) (bool, error),
	projection *iceberg.ManifestEntryProjection,
	dropColumnStats bool,
) ([]iceberg.ManifestEntry, error) {
	return openManifestWithReadOptions(
		io, manifest, partitionFilter, metricsEval, projection, dropColumnStats, true, false)
}

func openManifestWithOptions(io io.IO, manifest iceberg.ManifestFile,
	partitionFilter, metricsEval func(iceberg.DataFile) (bool, error), discardDeleted, discardExisting bool,
) ([]iceberg.ManifestEntry, error) {
	return openManifestWithReadOptions(
		io, manifest, partitionFilter, metricsEval, nil, false, discardDeleted, discardExisting)
}

func openManifestWithReadOptions(
	io io.IO,
	manifest iceberg.ManifestFile,
	partitionFilter, metricsEval func(iceberg.DataFile) (bool, error),
	projection *iceberg.ManifestEntryProjection,
	dropColumnStats, discardDeleted, discardExisting bool,
) ([]iceberg.ManifestEntry, error) {
	// Counts may be -1 (unset) on V1 manifests, so clamp before allocating.
	capacity := 0
	if added := manifest.AddedDataFiles(); added > 0 {
		capacity += int(added)
	}
	if !discardExisting {
		if existing := manifest.ExistingDataFiles(); existing > 0 {
			capacity += int(existing)
		}
	}
	if !discardDeleted {
		if deleted := manifest.DeletedDataFiles(); deleted > 0 {
			capacity += int(deleted)
		}
	}
	out := make([]iceberg.ManifestEntry, 0, capacity)
	if err := streamManifestWithReadOptions(
		io, manifest, partitionFilter, metricsEval, projection, dropColumnStats,
		discardDeleted, discardExisting, func(entry iceberg.ManifestEntry) error {
			out = append(out, entry)

			return nil
		}); err != nil {
		return nil, err
	}

	return out, nil
}

// streamManifest reads live entries from a manifest, applying partition and
// metrics filters before passing each match to visit. It deliberately does not
// retain the entries, so callers can choose the smallest representation needed
// for the next planning step.
func streamManifest(manifestIO io.IO, manifest iceberg.ManifestFile,
	partitionFilter, metricsEval func(iceberg.DataFile) (bool, error),
	projection *iceberg.ManifestEntryProjection,
	dropColumnStats bool,
	visit func(iceberg.ManifestEntry) error,
) error {
	return streamManifestWithReadOptions(
		manifestIO, manifest, partitionFilter, metricsEval, projection, dropColumnStats,
		true, false, visit)
}

func streamManifestWithOptions(manifestIO io.IO, manifest iceberg.ManifestFile,
	partitionFilter, metricsEval func(iceberg.DataFile) (bool, error), discardDeleted, discardExisting bool,
	visit func(iceberg.ManifestEntry) error,
) error {
	return streamManifestWithReadOptions(
		manifestIO, manifest, partitionFilter, metricsEval, nil, false,
		discardDeleted, discardExisting, visit)
}

func streamManifestWithReadOptions(manifestIO io.IO, manifest iceberg.ManifestFile,
	partitionFilter, metricsEval func(iceberg.DataFile) (bool, error),
	projection *iceberg.ManifestEntryProjection,
	dropColumnStats, discardDeleted, discardExisting bool,
	visit func(iceberg.ManifestEntry) error,
) error {
	entries := manifest.Entries(manifestIO, discardDeleted)
	if projection != nil {
		entries = iceberg.EntriesWithProjection(manifestIO, manifest, discardDeleted, *projection)
	}
	for entry, err := range entries {
		if err != nil {
			return err
		}
		if discardExisting && entry.Status() == iceberg.EntryStatusEXISTING {
			continue
		}

		dataFile := entry.DataFile()
		use, err := partitionFilter(dataFile)
		if err != nil {
			return err
		}
		if !use {
			continue
		}

		use, err = metricsEval(dataFile)
		if err != nil {
			return err
		}
		if !use {
			continue
		}

		if dropColumnStats {
			entry = iceberg.ManifestEntryWithoutColumnStats(entry)
		}
		if err := visit(entry); err != nil {
			return err
		}
	}

	return nil
}

// IsDeletionVector reports whether df is a deletion vector: a Puffin file with
// position-delete content. The content-type guard matters because df is an
// arbitrary DataFile, so a non-pos-delete Puffin from an external writer must
// not be misclassified. Keying on format rather than referenced_data_file
// avoids misclassifying a Parquet pos-delete that legally sets it.
func IsDeletionVector(df iceberg.DataFile) bool {
	return df.FileFormat() == iceberg.PuffinFile &&
		df.ContentType() == iceberg.EntryContentPosDeletes
}

type dataFileKind int

const (
	dataFileKindData dataFileKind = iota
	dataFileKindPosDeletes
	dataFileKindEqDeletes
	dataFileKindDeletionVector
)

// classifyDataFile buckets a file by content type. Deletion vectors are
// Puffin position-delete files and are split out from regular pos-deletes.
func classifyDataFile(f iceberg.DataFile) (dataFileKind, error) {
	switch f.ContentType() {
	case iceberg.EntryContentData:
		return dataFileKindData, nil
	case iceberg.EntryContentPosDeletes:
		if IsDeletionVector(f) {
			return dataFileKindDeletionVector, nil
		}

		return dataFileKindPosDeletes, nil
	case iceberg.EntryContentEqDeletes:
		return dataFileKindEqDeletes, nil
	default:
		return 0, fmt.Errorf("%w: unknown DataFileContent type (%s)",
			ErrInvalidMetadata, f.ContentType())
	}
}

// Scan represents a table scan. It implements [io.Closer]; callers should
// close it when they are done, including early exits after remote planning
// succeeds but before all records are consumed.
type Scan struct {
	identifier          Identifier
	metadata            Metadata
	metadataLocation    string
	ioF                 FSysF
	manifestCache       *snapshotManifestCache
	planner             ScanPlanner
	scanPlanningIOProps iceberg.Properties
	planningMode        ScanPlanningMode
	// planIO, when non-nil, is a plan-scoped FileIO loader set by remote scan
	// planning. ReadTasks leases it instead of falling back to ioF, and replacing
	// the plan retires it after all active readers finish. See PlanIO.
	planIO         *planIOState
	closed         uint32
	rowFilter      iceberg.BooleanExpression
	selectedFields []string
	caseSensitive  bool
	snapshotID     *int64
	asOfTimestamp  *int64
	// useSnapshotSchema is set for explicit snapshot/time-travel and tag
	// scans. A branch ref deliberately keeps the table's current schema. A nil
	// value preserves the historical behavior for scans assembled directly in
	// package tests with snapshotID/asOfTimestamp fields set.
	useSnapshotSchema *bool
	options           iceberg.Properties
	limit             int64
	selectorErr       error

	includeRowLineage bool

	concurrency int

	// arrowBatchSize, when positive, caps the rows decoded per Arrow
	// record batch, overriding the table's read.parquet.batch-size
	// property. Set via WithArrowBatchSize; kept as a dedicated field
	// (like limit and concurrency) so WithOptions replacing the options
	// map cannot silently drop it.
	arrowBatchSize int

	reporter metrics.Reporter
}

// clone copies the scan configuration and gives the copy its own ownership
// reference to any remote plan.
func (scan *Scan) clone() *Scan {
	out := *scan
	if out.planIO != nil {
		out.planIO.retain()
	}

	return &out
}

func (scan *Scan) UseRowLimit(n int64) *Scan {
	out := scan.clone()
	out.limit = n

	return out
}

// Reporter returns the metrics reporter for this scan, never nil. The
// scan-planning instrumentation emits its ScanReport through it.
func (scan *Scan) Reporter() metrics.Reporter {
	if scan.reporter == nil {
		return metrics.NopReporter{}
	}

	return scan.reporter
}

// UseRef selects a named snapshot reference. UseRef(MainBranch) is the one
// intentional exception to selector exclusivity: it returns a clone without
// changing an existing snapshot or as-of selector. Any conflicting selectors
// recorded by scan options are still surfaced by scan execution.
func (scan *Scan) UseRef(name string) (*Scan, error) {
	if name == MainBranch {
		return scan.clone(), nil
	}
	if scan.selectorErr != nil {
		return nil, scan.selectorErr
	}

	if scan.snapshotID != nil {
		return nil, fmt.Errorf("%w: cannot override ref, already set snapshot id %d",
			iceberg.ErrInvalidArgument, *scan.snapshotID)
	}
	if scan.asOfTimestamp != nil {
		return nil, fmt.Errorf("%w: cannot override ref, already set as-of timestamp %d",
			iceberg.ErrInvalidArgument, *scan.asOfTimestamp)
	}

	if snap := scan.metadata.SnapshotByName(name); snap != nil {
		out := scan.clone()
		out.snapshotID = &snap.SnapshotID
		out.asOfTimestamp = nil
		useSnapshotSchema := true
		for refName, ref := range scan.metadata.Refs() {
			if refName == name {
				useSnapshotSchema = ref.SnapshotRefType == TagRef

				break
			}
		}
		out.useSnapshotSchema = &useSnapshotSchema

		return out, nil
	}

	return nil, fmt.Errorf("%w: cannot scan unknown ref=%s", iceberg.ErrInvalidArgument, name)
}

// ResolveSnapshot resolves the snapshot selected by this scan. Live scans use
// the table's current snapshot; explicit snapshot IDs and as-of timestamps
// must resolve to an existing snapshot.
func (scan *Scan) ResolveSnapshot() (*Snapshot, error) {
	if scan.selectorErr != nil {
		return nil, scan.selectorErr
	}

	if scan.snapshotID != nil {
		snap := scan.metadata.SnapshotByID(*scan.snapshotID)
		if snap == nil {
			return nil, fmt.Errorf("%w: snapshot not found: %d", ErrInvalidOperation, *scan.snapshotID)
		}

		return snap, nil
	}

	if scan.asOfTimestamp != nil {
		entry, ok := snapshotLogEntryAsOf(scan.metadata.SnapshotLogs(), *scan.asOfTimestamp, true)
		if !ok {
			return nil, fmt.Errorf("no snapshot found for timestamp %d", *scan.asOfTimestamp)
		}

		snap := scan.metadata.SnapshotByID(entry.SnapshotID)
		if snap == nil {
			return nil, fmt.Errorf("%w: snapshot log references unknown snapshot %d", ErrInvalidMetadata, entry.SnapshotID)
		}

		return snap, nil
	}

	return scan.metadata.CurrentSnapshot(), nil
}

// Snapshot returns the snapshot selected by this scan. It returns nil when an
// explicit snapshot cannot be resolved; use ResolveSnapshot when the reason
// for that result must be distinguished from a table with no current snapshot.
func (scan *Scan) Snapshot() *Snapshot {
	snap, _ := scan.ResolveSnapshot()

	return snap
}

func (scan *Scan) Projection() (*iceberg.Schema, error) {
	if scan.selectorErr != nil {
		return nil, scan.selectorErr
	}

	curSchema, err := scan.effectiveSchema()
	if err != nil {
		return nil, err
	}
	curVersion := scan.metadata.Version()

	if scan.includeRowLineage && curVersion < minFormatVersionRowLineage {
		return nil, fmt.Errorf("%w: row lineage requires format version %d, table is v%d",
			ErrInvalidOperation, minFormatVersionRowLineage, curVersion)
	}

	var schema *iceberg.Schema
	if slices.Contains(scan.selectedFields, "*") {
		schema = curSchema
	} else {
		// Intercept row-lineage metadata column names (_row_id,
		// _last_updated_sequence_number) before calling Select: they are
		// reserved and never appear in the user schema's fields, so
		// Select would fail with "could not find column" on v3 tables
		// where they are otherwise legal to project. The scanner reads
		// them from file metadata (or synthesizes them) at scan time;
		// here we just need to ensure they survive into the projection.
		userFields, lineageFields := splitLineageMetadataFields(scan.selectedFields, scan.caseSensitive)
		if len(lineageFields) > 0 && curVersion < minFormatVersionRowLineage {
			// Reject explicitly so the contract lives in the code rather
			// than emerging from Select's "could not find column" path —
			// a future v2 schema field literally named _row_id should not
			// silently succeed here.
			return nil, fmt.Errorf("%w: row lineage column %q requires format version %d, table is v%d",
				ErrInvalidOperation, lineageFields[0].Name, minFormatVersionRowLineage, curVersion)
		}

		var err error
		schema, err = curSchema.Select(scan.caseSensitive, userFields...)
		if err != nil {
			return nil, err
		}
		// Skip the per-name append when scan.includeRowLineage is set: the
		// SchemaWithRowLineage call below adds both lineage columns
		// unconditionally, and appendMissingLineageFields would just be
		// redundant work whose result is overwritten.
		if len(lineageFields) > 0 && !scan.includeRowLineage {
			schema = appendMissingLineageFields(schema, lineageFields)
		}
	}

	if scan.includeRowLineage {
		schema = iceberg.SchemaWithRowLineage(schema)
	}

	return schema, nil
}

func (scan *Scan) effectiveSchema() (*iceberg.Schema, error) {
	if scan.selectorErr != nil {
		return nil, scan.selectorErr
	}

	if !scan.snapshotSchemaEnabled() {
		// Live scans intentionally use the table's current schema. A schema-only
		// metadata update can advance CurrentSchema without creating a snapshot,
		// and branch refs intentionally use the table schema even though they
		// resolve to a snapshot.
		return scan.metadata.CurrentSchema(), nil
	}

	snap, err := scan.ResolveSnapshot()
	if err != nil {
		return nil, err
	}

	if snap.SchemaID == nil {
		return scan.metadata.CurrentSchema(), nil
	}

	if schema := schemaFromMetadata(scan.metadata, *snap.SchemaID); schema != nil {
		return schema, nil
	}

	return nil, fmt.Errorf("%w: snapshot %d references unknown schema id %d",
		ErrInvalidMetadata, snap.SnapshotID, *snap.SchemaID)
}

type metadataSchemaByID interface {
	schemaByID(int) *iceberg.Schema
}

func schemaFromMetadata(metadata Metadata, id int) *iceberg.Schema {
	if lookup, ok := metadata.(metadataSchemaByID); ok {
		return lookup.schemaByID(id)
	}

	for _, schema := range metadata.Schemas() {
		if schema.ID == id {
			return schema
		}
	}

	return nil
}

func (scan *Scan) snapshotSchemaEnabled() bool {
	if scan.useSnapshotSchema != nil {
		return *scan.useSnapshotSchema
	}

	return scan.snapshotID != nil || scan.asOfTimestamp != nil
}

// splitLineageMetadataFields partitions selectedFields into user fields and
// row-lineage metadata fields (_row_id, _last_updated_sequence_number). The
// returned lineage slice contains the canonical NestedField for each
// metadata column name found, in the order encountered.
func splitLineageMetadataFields(selectedFields []string, caseSensitive bool) (userFields []string, lineageFields []iceberg.NestedField) {
	matches := func(field, target string) bool {
		if caseSensitive {
			return field == target
		}

		return strings.EqualFold(field, target)
	}

	userFields = make([]string, 0, len(selectedFields))
	for _, field := range selectedFields {
		switch {
		case matches(field, iceberg.RowIDColumnName):
			lineageFields = append(lineageFields, iceberg.RowID())
		case matches(field, iceberg.LastUpdatedSequenceNumberColumnName):
			lineageFields = append(lineageFields, iceberg.LastUpdatedSequenceNumber())
		default:
			userFields = append(userFields, field)
		}
	}

	return userFields, lineageFields
}

// appendMissingLineageFields returns a new schema with each lineage field
// appended only if no field with that ID is already present. Idempotent so
// callers can pass schemas that already declare the reserved fields.
func appendMissingLineageFields(s *iceberg.Schema, lineageFields []iceberg.NestedField) *iceberg.Schema {
	fields := s.Fields()
	existing := make(map[int]struct{}, len(fields))
	for _, f := range fields {
		existing[f.ID] = struct{}{}
	}

	for _, f := range lineageFields {
		if _, ok := existing[f.ID]; ok {
			continue
		}
		fields = append(fields, f)
		existing[f.ID] = struct{}{}
	}

	return iceberg.NewSchemaWithIdentifiers(s.ID, s.IdentifierFieldIDs, fields...)
}

func buildPartitionProjection(specID int, meta Metadata, schema *iceberg.Schema, rowFilter iceberg.BooleanExpression, caseSensitive bool) (iceberg.BooleanExpression, error) {
	spec := meta.PartitionSpecByID(specID)
	if spec == nil {
		return nil, fmt.Errorf("%w: id %d", ErrPartitionSpecNotFound, specID)
	}
	project := newInclusiveProjection(schema, *spec, caseSensitive)

	return project(rowFilter)
}

func buildManifestEvaluator(specID int, metadata Metadata, schema *iceberg.Schema, partitionFilters *keyDefaultMapErr[int, iceberg.BooleanExpression], caseSensitive bool) (func(iceberg.ManifestFile) (bool, error), error) {
	spec := metadata.PartitionSpecByID(specID)
	if spec == nil {
		return nil, fmt.Errorf("%w: id %d", ErrPartitionSpecNotFound, specID)
	}

	partitionFilter, err := partitionFilters.Get(specID)
	if err != nil {
		return nil, err
	}

	return newManifestEvaluator(*spec, schema,
		partitionFilter, caseSensitive)
}

func buildPartitionEvaluator(specID int, metadata Metadata, schema *iceberg.Schema, partitionFilters *keyDefaultMapErr[int, iceberg.BooleanExpression], caseSensitive bool) (func(iceberg.DataFile) (bool, error), error) {
	spec := metadata.PartitionSpecByID(specID)
	if spec == nil {
		return nil, fmt.Errorf("%w: id %d", ErrPartitionSpecNotFound, specID)
	}
	partType := spec.PartitionType(schema)
	partSchema := iceberg.NewSchema(0, partType.FieldList...)

	partitionFilter, err := partitionFilters.Get(specID)
	if err != nil {
		return nil, err
	}

	fn, err := iceberg.ExpressionEvaluator(partSchema, partitionFilter, caseSensitive)
	if err != nil {
		return nil, err
	}

	return func(d iceberg.DataFile) (bool, error) {
		return fn(borrowedPartitionRecord{
			partition:     dataFilePartition(d),
			partitionType: partType,
		})
	}, nil
}

func (scan *Scan) partitionFiltersForSchema(schema *iceberg.Schema) *keyDefaultMapErr[int, iceberg.BooleanExpression] {
	return newKeyDefaultMapWrapErr(func(specID int) (iceberg.BooleanExpression, error) {
		return buildPartitionProjection(specID, scan.metadata, schema, scan.rowFilter, scan.caseSensitive)
	})
}

func (scan *Scan) checkSequenceNumber(minSeqNum int64, manifest iceberg.ManifestFile) bool {
	return manifest.ManifestContent() == iceberg.ManifestContentData ||
		(manifest.ManifestContent() == iceberg.ManifestContentDeletes &&
			manifest.SequenceNum() >= minSeqNum)
}

func minSequenceNum(manifests []iceberg.ManifestFile) int64 {
	var n int64 = math.MaxInt64
	for _, m := range manifests {
		if m.ManifestContent() == iceberg.ManifestContentData {
			n = min(n, m.MinSequenceNum())
		}
	}
	if n == math.MaxInt64 {
		return 0
	}

	return n
}

// matchEqualityDeletesToData returns the equality delete files that apply to
// the given data entry. An equality delete applies when:
//   - it has a strictly greater sequence number than the data file
//   - it shares the same partition (for partitioned tables)
//
// The "strictly greater" rule ensures that data files committed in the same
// snapshot as the equality deletes are not affected — this is how RowDelta
// atomically adds new rows alongside deletes for old rows.
func matchEqualityDeletesToData(dataEntry iceberg.ManifestEntry, eqDeleteEntries []iceberg.ManifestEntry) []iceberg.DataFile {
	dataSeqNum := dataEntry.SequenceNum()
	dataPartition := dataEntry.DataFile().Partition()

	out := make([]iceberg.DataFile, 0)
	for _, del := range eqDeleteEntries {
		// Equality deletes only apply to data files with a strictly lower
		// sequence number.
		if del.SequenceNum() <= dataSeqNum {
			continue
		}

		// For partitioned tables, equality deletes must share the same
		// partition as the data file. Unpartitioned deletes (nil/empty
		// partition) apply globally.
		delPartition := del.DataFile().Partition()
		if len(delPartition) > 0 && len(dataPartition) > 0 {
			if !partitionsMatch(dataPartition, delPartition) {
				continue
			}
		}

		out = append(out, del.DataFile())
	}

	return out
}

func partitionsMatch(a, b map[int]any) bool {
	return maps.EqualFunc(a, b, partitionValuesEqual)
}

// buildDVIndex indexes deletion vectors by the data file path they reference.
// The spec requires at most one DV per data file; a second entry for the same
// path is rejected with an error.
func buildDVIndex(dvEntries []iceberg.ManifestEntry) (map[string]iceberg.ManifestEntry, error) {
	dvIndex := make(map[string]iceberg.ManifestEntry, len(dvEntries))
	for _, del := range dvEntries {
		if ref := iceberginternal.BorrowedDataFileReferencedDataFile(del.DataFile()); ref != nil {
			if _, exists := dvIndex[*ref]; exists {
				return nil, fmt.Errorf("can't index multiple deletion vectors for %s", *ref)
			}
			dvIndex[*ref] = del
		}
	}

	return dvIndex, nil
}

// matchDVToData returns the deletion vector that applies to the given data
// entry, if any. A DV applies when the data file's sequence number is less
// than or equal to the DV's sequence number.
//
// SequenceNum reports the -1 sentinel when an entry's sequence number is
// unset (see manifest.go). Entries arrive here already inherited, so a
// committed ADDED entry always carries a real (>= 0) sequence number; an
// unset value comes from an EXISTING/DELETED entry missing its required
// explicit sequence number, or a not-yet-committed manifest. Such an
// indeterminate sequence number — on either side — is treated as "applies":
// comparing a real data sequence number against an unset DV sequence (-1)
// would never satisfy dataSeq <= -1 and would silently drop the DV,
// resurfacing deleted rows; an unset data sequence likewise satisfies
// -1 <= dvSeq for any known DV sequence.
func matchDVToData(dataEntry iceberg.ManifestEntry, dvIndex map[string]iceberg.ManifestEntry) []iceberg.DataFile {
	dvEntry, ok := dvIndex[dataEntry.DataFile().FilePath()]
	if !ok {
		return nil
	}
	if dvSeq := dvEntry.SequenceNum(); dvSeq < 0 || dataEntry.SequenceNum() <= dvSeq {
		return []iceberg.DataFile{dvEntry.DataFile()}
	}

	return nil
}

// fetchPartitionSpecFilteredManifests retrieves the table's current snapshot,
// fetches its manifest files, and applies partition-spec filters to remove irrelevant manifests.
func (scan *Scan) fetchPartitionSpecFilteredManifests(ctx context.Context) ([]iceberg.ManifestFile, error) {
	schema, err := scan.effectiveSchema()
	if err != nil {
		return nil, err
	}
	snap, err := scan.ResolveSnapshot()
	if err != nil || snap == nil {
		return nil, err
	}

	// This path has no reporter behind it, so the manifest counts recorded into
	// this accumulator are intentionally discarded. A future caller that needs
	// those counts should use fetchPartitionSpecFilteredManifestsWithSchema and
	// pass in an accumulator it actually reads.
	return scan.fetchPartitionSpecFilteredManifestsWithSchema(
		ctx, snap, schema, &scanMetricsAccumulator{}, scan.partitionFiltersForSchema(schema))
}

// fetchPartitionSpecFilteredManifestsWithSchema loads the snapshot's manifests
// and filters them using the given schema. It records
// total/scanned/skipped manifest counts (split by data vs delete content) into acc.
func (scan *Scan) fetchPartitionSpecFilteredManifestsWithSchema(
	ctx context.Context,
	snap *Snapshot,
	schema *iceberg.Schema,
	acc *scanMetricsAccumulator,
	partitionFilters *keyDefaultMapErr[int, iceberg.BooleanExpression],
) ([]iceberg.ManifestFile, error) {
	// Fetch all manifests for the current snapshot.
	manifestSet, err := scan.manifestSet(ctx, *snap)
	if err != nil {
		return nil, err
	}

	return scan.filterManifestsWithSchema(manifestSet.allManifests(), schema, acc, partitionFilters)
}

func (scan *Scan) manifestSet(
	ctx context.Context,
	snapshot Snapshot,
) (snapshotManifestSet, error) {
	return scan.manifestSetWithFSF(ctx, snapshot, scan.ioF)
}

func (scan *Scan) manifestSetWithFSF(
	ctx context.Context,
	snapshot Snapshot,
	fsF FSysF,
) (snapshotManifestSet, error) {
	return scan.manifestCache.get(ctx, snapshot, func(loadCtx context.Context) (snapshotManifestSet, error) {
		return readSnapshotManifestSet(loadCtx, snapshot, fsF)
	})
}

// filterManifestsWithSchema applies partition-summary pruning to an existing
// list of manifests. Callers use this after resolving a snapshot's manifest
// list, or after collecting manifests across an incremental snapshot range.
func (scan *Scan) filterManifestsWithSchema(
	manifestList []iceberg.ManifestFile,
	schema *iceberg.Schema,
	acc *scanMetricsAccumulator,
	partitionFilters *keyDefaultMapErr[int, iceberg.BooleanExpression],
) ([]iceberg.ManifestFile, error) {
	return scan.filterManifestsWithSchemaOptions(
		manifestList, schema, acc, partitionFilters, false)
}

// filterManifestsWithSchemaOptions is filterManifestsWithSchema with an
// option for changelog scans, which must retain data manifests containing
// deleted entries even when they have no live entries.
func (scan *Scan) filterManifestsWithSchemaOptions(
	manifestList []iceberg.ManifestFile,
	schema *iceberg.Schema,
	acc *scanMetricsAccumulator,
	partitionFilters *keyDefaultMapErr[int, iceberg.BooleanExpression],
	includeDeleted bool,
) ([]iceberg.ManifestFile, error) {
	// Build per-spec manifest evaluators and filter out irrelevant manifests.
	manifestEvaluators := newKeyDefaultMapWrapErr(func(specID int) (func(iceberg.ManifestFile) (bool, error), error) {
		return buildManifestEvaluator(specID, scan.metadata, schema, partitionFilters, scan.caseSensitive)
	})
	filtered := make([]iceberg.ManifestFile, 0, len(manifestList))
	for _, mf := range manifestList {
		isDelete := mf.ManifestContent() == iceberg.ManifestContentDeletes
		if isDelete {
			acc.totalDeleteManifests++
		} else {
			acc.totalDataManifests++
		}

		eval, err := manifestEvaluators.Get(int(mf.PartitionSpecID()))
		if err != nil {
			return nil, fmt.Errorf("failed to build manifest evaluator for spec %d: %w", mf.PartitionSpecID(), err)
		}
		use, err := eval(mf)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate manifest %s: %w", mf.FilePath(), err)
		}
		// Has*Files returns true for unknown counts, so this only skips manifests
		// known to contain no added or existing (live) entries. The deleted-file
		// count follows the same rule: V1's -1 means unknown, while zero is the
		// only known empty value. Changelog scans also retain manifests known to
		// contain deleted entries.
		if use && !mf.HasAddedFiles() && !mf.HasExistingFiles() &&
			(!includeDeleted || mf.DeletedDataFiles() == 0) {
			if isDelete {
				acc.skippedDeleteManifests++
			} else {
				acc.skippedDataManifests++
			}

			continue
		}
		if use {
			if isDelete {
				acc.scannedDeleteManifests++
			} else {
				acc.scannedDataManifests++
			}
			filtered = append(filtered, mf)
		} else if isDelete {
			acc.skippedDeleteManifests++
		} else {
			acc.skippedDataManifests++
		}
	}

	return filtered, nil
}

// manifestIOBatch shares a FileIO within a concurrency-sized batch. A new
// batch loads through the factory again so factories that renew credentials
// still get regular checkpoints without rebuilding the backend for every
// manifest.
type manifestIOBatch struct {
	factory FSysF
	limit   int

	mu        sync.Mutex
	fs        io.IO
	remaining int
}

func newManifestIOBatch(factory FSysF, limit int) *manifestIOBatch {
	return &manifestIOBatch{factory: factory, limit: max(limit, 1)}
}

func (b *manifestIOBatch) acquire(ctx context.Context) (io.IO, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if b.remaining == 0 {
		fs, err := b.factory(ctx)
		if err != nil {
			return nil, err
		}

		b.fs = fs
		b.remaining = b.limit
	}

	b.remaining--

	return b.fs, nil
}

// collectManifestEntries concurrently opens manifests, applies partition and metrics
// filters, and accumulates both data entries and positional-delete entries.
func (scan *Scan) collectManifestEntries(
	ctx context.Context,
	manifestList []iceberg.ManifestFile,
) (*manifestEntries, error) {
	schema, err := scan.effectiveSchema()
	if err != nil {
		return nil, err
	}

	return scan.collectManifestEntriesWithSchema(
		ctx, manifestList, schema, scan.partitionFiltersForSchema(schema))
}

func (scan *Scan) collectManifestEntriesWithSchema(
	ctx context.Context,
	manifestList []iceberg.ManifestFile,
	schema *iceberg.Schema,
	partitionFilters *keyDefaultMapErr[int, iceberg.BooleanExpression],
) (*manifestEntries, error) {
	return scan.collectManifestEntriesWithSchemaMinSequenceNum(
		ctx, manifestList, schema, partitionFilters, minSequenceNum(manifestList), false)
}

func (scan *Scan) collectManifestEntriesWithSchemaMinSequenceNum(
	ctx context.Context,
	manifestList []iceberg.ManifestFile,
	schema *iceberg.Schema,
	partitionFilters *keyDefaultMapErr[int, iceberg.BooleanExpression],
	minSeqNum int64,
	projectScanColumns bool,
) (*manifestEntries, error) {
	return scan.collectManifestEntriesWithSchemaOptionsAndMinSequenceNum(
		ctx, manifestList, schema, partitionFilters, minSeqNum, projectScanColumns, true, false)
}

func (scan *Scan) collectManifestEntriesWithSchemaOptions(
	ctx context.Context,
	manifestList []iceberg.ManifestFile,
	schema *iceberg.Schema,
	partitionFilters *keyDefaultMapErr[int, iceberg.BooleanExpression],
	discardDeleted, discardExisting bool,
) (*manifestEntries, error) {
	return scan.collectManifestEntriesWithSchemaOptionsAndMinSequenceNum(
		ctx, manifestList, schema, partitionFilters,
		minSequenceNum(manifestList), false, discardDeleted, discardExisting)
}

func (scan *Scan) collectManifestEntriesWithSchemaOptionsAndMinSequenceNum(
	ctx context.Context,
	manifestList []iceberg.ManifestFile,
	schema *iceberg.Schema,
	partitionFilters *keyDefaultMapErr[int, iceberg.BooleanExpression],
	minSeqNum int64,
	projectScanColumns bool,
	discardDeleted, discardExisting bool,
) (*manifestEntries, error) {
	metricsEval, err := newInclusiveMetricsEvaluator(
		schema,
		scan.rowFilter,
		scan.caseSensitive,
		scan.options["include_empty_files"] == "true",
	)
	if err != nil {
		return nil, err
	}

	concurrencyLimit := min(scan.concurrency, len(manifestList))
	manifestIO := newManifestIOBatch(scan.ioF, concurrencyLimit)

	manifestResults := make([]classifiedManifestEntries, len(manifestList))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrencyLimit)

	partitionEvaluators := newKeyDefaultMapWrapErr(func(specID int) (func(iceberg.DataFile) (bool, error), error) {
		return buildPartitionEvaluator(specID, scan.metadata, schema, partitionFilters, scan.caseSensitive)
	})

	for manifestIndex, mf := range manifestList {
		if !scan.checkSequenceNumber(minSeqNum, mf) {
			continue
		}

		g.Go(func() error {
			fs, err := manifestIO.acquire(gctx)
			if err != nil {
				return err
			}
			partEval, err := partitionEvaluators.Get(int(mf.PartitionSpecID()))
			if err != nil {
				return fmt.Errorf("failed to build partition evaluator for spec %d: %w", mf.PartitionSpecID(), err)
			}
			var projection *iceberg.ManifestEntryProjection
			if projectScanColumns {
				// Projected collection reads delete manifests before classification.
				// Keep pruning stats until equality and positional deletes are indexed.
				projection = &iceberg.ManifestEntryProjection{IncludePruningStats: true}
			}
			manifestEntries, err := openManifestWithReadOptions(
				fs, mf, partEval, metricsEval, projection, false, discardDeleted, discardExisting)
			if err != nil {
				return err
			}
			classified, err := classifyManifestEntries(manifestEntries)
			if err != nil {
				return err
			}
			manifestResults[manifestIndex] = classified

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return flattenClassifiedManifestEntries(manifestResults), nil
}

func (scan *Scan) dataManifestProjection(retainDataFileStats bool) (iceberg.ManifestEntryProjection, bool) {
	includePruningStats := retainDataFileStats ||
		(scan.rowFilter != nil && !scan.rowFilter.Equals(iceberg.AlwaysTrue{}))
	dropColumnStats := includePruningStats && !retainDataFileStats

	return iceberg.ManifestEntryProjection{IncludePruningStats: includePruningStats}, dropColumnStats
}

func splitManifestList(manifestList []iceberg.ManifestFile) (dataManifests, deleteManifests []iceberg.ManifestFile) {
	dataManifests = make([]iceberg.ManifestFile, 0, len(manifestList))
	deleteManifests = make([]iceberg.ManifestFile, 0, len(manifestList))
	for _, manifest := range manifestList {
		if manifest.ManifestContent() == iceberg.ManifestContentDeletes {
			deleteManifests = append(deleteManifests, manifest)
		} else {
			dataManifests = append(dataManifests, manifest)
		}
	}

	return dataManifests, deleteManifests
}

func manifestTaskCapacity(manifest iceberg.ManifestFile) (int, bool) {
	addedFiles, existingFiles := manifest.AddedDataFiles(), manifest.ExistingDataFiles()
	if addedFiles < 0 || existingFiles < 0 {
		return 0, false
	}

	capacity := int64(addedFiles) + int64(existingFiles)
	if capacity > int64(^uint(0)>>1) {
		return 0, false
	}

	return int(capacity), true
}

// planDataManifestTasks streams filtered data entries into task batches. The
// batches are kept per manifest while reads run concurrently, then flattened
// in manifest-list order so task ordering does not depend on read completion.
// Manifests with reliable live-entry counts share one task buffer; manifests
// with unknown counts use independent batches and the same final flattening.
func (scan *Scan) planDataManifestTasks(
	ctx context.Context,
	manifestList []iceberg.ManifestFile,
	schema *iceberg.Schema,
	minSeqNum int64,
	posDeleteIndex *positionalDeleteIndex,
	dvIndex map[string]iceberg.ManifestEntry,
	eqDeleteIndex *equalityDeleteIndex,
) ([]FileScanTask, error) {
	return scan.planDataManifestTasksWithOptions(
		ctx, manifestList, schema, minSeqNum, posDeleteIndex, dvIndex, eqDeleteIndex, false, false)
}

func (scan *Scan) planDataManifestTasksWithOptions(
	ctx context.Context,
	manifestList []iceberg.ManifestFile,
	schema *iceberg.Schema,
	minSeqNum int64,
	posDeleteIndex *positionalDeleteIndex,
	dvIndex map[string]iceberg.ManifestEntry,
	eqDeleteIndex *equalityDeleteIndex,
	projectScanColumns bool,
	retainDataFileStats bool,
) ([]FileScanTask, error) {
	metricsEval, err := newInclusiveMetricsEvaluator(
		schema,
		scan.rowFilter,
		scan.caseSensitive,
		scan.options["include_empty_files"] == "true",
	)
	if err != nil {
		return nil, err
	}

	taskBatches := make([][]FileScanTask, len(manifestList))
	taskCapacities := make([]int, len(manifestList))
	taskOffsets := make([]int, len(manifestList))
	directBuffer := scan.rowFilter == nil || scan.rowFilter.Equals(iceberg.AlwaysTrue{})
	totalCapacity := 0
	maxInt := int(^uint(0) >> 1)
	if directBuffer {
		for index, manifest := range manifestList {
			if !scan.checkSequenceNumber(minSeqNum, manifest) {
				continue
			}

			capacity, ok := manifestTaskCapacity(manifest)
			if !ok || capacity > maxInt-totalCapacity {
				directBuffer = false

				break
			}
			taskCapacities[index] = capacity
			taskOffsets[index] = totalCapacity
			totalCapacity += capacity
		}
	}
	var taskBuffer []FileScanTask
	if directBuffer {
		taskBuffer = make([]FileScanTask, totalCapacity)
	}

	concurrencyLimit := min(scan.concurrency, len(manifestList))
	manifestIO := newManifestIOBatch(scan.ioF, concurrencyLimit)
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrencyLimit)

	partitionFilters := scan.partitionFiltersForSchema(schema)
	partitionEvaluators := newKeyDefaultMapWrapErr(func(specID int) (func(iceberg.DataFile) (bool, error), error) {
		return buildPartitionEvaluator(specID, scan.metadata, schema, partitionFilters, scan.caseSensitive)
	})

	for index, manifest := range manifestList {
		if !scan.checkSequenceNumber(minSeqNum, manifest) {
			continue
		}

		g.Go(func() error {
			fs, err := manifestIO.acquire(gctx)
			if err != nil {
				return err
			}
			partEval, err := partitionEvaluators.Get(int(manifest.PartitionSpecID()))
			if err != nil {
				return fmt.Errorf("failed to build partition evaluator for spec %d: %w", manifest.PartitionSpecID(), err)
			}

			var tasks []FileScanTask
			if directBuffer {
				offset := taskOffsets[index]
				capacity := taskCapacities[index]
				tasks = taskBuffer[offset : offset+capacity : offset+capacity]
				tasks = tasks[:0]
			} else {
				tasks = make([]FileScanTask, 0,
					max(0, int(manifest.AddedDataFiles())+int(manifest.ExistingDataFiles())))
			}
			var projection *iceberg.ManifestEntryProjection
			dropColumnStats := false
			stripTaskColumnStats := false
			if projectScanColumns {
				var p iceberg.ManifestEntryProjection
				p, dropColumnStats = scan.dataManifestProjection(retainDataFileStats)
				stripTaskColumnStats = p.IncludePruningStats && !dropColumnStats
				projection = &p
			}
			err = streamManifest(fs, manifest, partEval, metricsEval, projection, dropColumnStats, func(entry iceberg.ManifestEntry) error {
				dataFile := entry.DataFile()
				if dataFile.ContentType() != iceberg.EntryContentData {
					return fmt.Errorf("%w: data manifest contains %s file %q",
						ErrInvalidMetadata, dataFile.ContentType(), dataFile.FilePath())
				}

				task, err := fileScanTaskForDataEntry(entry, posDeleteIndex, dvIndex, eqDeleteIndex)
				if err != nil {
					return err
				}
				if stripTaskColumnStats {
					// Statistics are needed while filtering and matching deletes, but
					// are transient for the task returned to Arrow readers.
					task.File = iceberg.DataFileWithoutColumnStats(task.File)
				}
				tasks = append(tasks, task)

				return nil
			})
			if err != nil {
				return err
			}

			taskBatches[index] = tasks

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var results []FileScanTask
	if directBuffer {
		overflowed := false
		for index, tasks := range taskBatches {
			if len(tasks) > taskCapacities[index] {
				overflowed = true

				break
			}
		}
		if !overflowed {
			writeIndex := 0
			for index, tasks := range taskBatches {
				count := len(tasks)
				if count > 0 {
					sourceOffset := taskOffsets[index]
					copy(taskBuffer[writeIndex:writeIndex+count],
						taskBuffer[sourceOffset:sourceOffset+count])
					writeIndex += count
				}
				taskBatches[index] = nil
			}

			results = taskBuffer[:writeIndex:writeIndex]
		} else {
			directBuffer = false
		}
	}

	if !directBuffer {
		totalTasks := 0
		for _, tasks := range taskBatches {
			totalTasks += len(tasks)
		}
		results = make([]FileScanTask, 0, totalTasks)
		for i, tasks := range taskBatches {
			results = append(results, tasks...)
			// Release the per-manifest backing array as soon as it has been copied
			// into the final result slice.
			taskBatches[i] = nil
		}
	}

	if projectScanColumns {
		memo := make(map[iceberg.DataFile]iceberg.DataFile)
		for i := range results {
			results[i].DeleteFiles = dataFilesWithoutColumnStats(results[i].DeleteFiles, memo)
			results[i].EqualityDeleteFiles = dataFilesWithoutColumnStats(results[i].EqualityDeleteFiles, memo)
			results[i].DeletionVectorFiles = dataFilesWithoutColumnStats(results[i].DeletionVectorFiles, memo)
		}
	}

	return results, nil
}

func fileScanTaskForDataEntry(
	entry iceberg.ManifestEntry,
	posDeleteIndex *positionalDeleteIndex,
	dvIndex map[string]iceberg.ManifestEntry,
	eqDeleteIndex *equalityDeleteIndex,
) (FileScanTask, error) {
	// Spec §Scan Planning: when a deletion vector applies to a data file,
	// positional-delete files must NOT be applied. The DV is guaranteed to
	// encode all prior pos-delete positions; reading the pos-delete Parquet too
	// would be wasteful I/O, and on a buggy writer whose DV omits prior
	// positions, applying both would over-delete. Mirrors Java's
	// DeleteFileIndex.forDataFile.
	dvFiles := matchDVToData(entry, dvIndex)
	var deleteFiles []iceberg.DataFile
	if len(dvFiles) == 0 {
		var err error
		deleteFiles, err = posDeleteIndex.forDataFile(entry)
		if err != nil {
			return FileScanTask{}, err
		}
	}
	eqDeleteFiles, err := eqDeleteIndex.forDataFile(entry)
	if err != nil {
		return FileScanTask{}, err
	}

	dataFile := entry.DataFile()
	task := FileScanTask{
		File:                dataFile,
		DeleteFiles:         deleteFiles,
		EqualityDeleteFiles: eqDeleteFiles,
		DeletionVectorFiles: dvFiles,
		Start:               0,
		Length:              dataFile.FileSizeBytes(),
	}
	// Row lineage constants: readers use these to synthesize _row_id and
	// _last_updated_sequence_number when requested. Per spec the synthesized
	// _last_updated_sequence_number is the manifest entry's data sequence
	// number (field id 3), not file sequence number (field id 4).
	task.FirstRowID = dataFile.FirstRowID()
	if seq := entry.SequenceNum(); seq >= 0 {
		task.DataSequenceNumber = &seq
	}

	return task, nil
}

// PlanFiles orchestrates the fetching and filtering of manifests, building a
// list of FileScanTasks that match the current Scan criteria. When planning
// happens locally it times the whole operation and emits a ScanReport to the
// scan's reporter on success; remote (server-side) planning reports its own
// metrics and does not emit here.
func (scan *Scan) PlanFiles(ctx context.Context) ([]FileScanTask, error) {
	return scan.planFiles(ctx, false)
}

// planFiles performs scan planning. Projected local planning is used by
// ToArrowRecords because manifest column statistics are only needed while
// pruning; PlanFiles keeps the complete DataFile metadata for callers that
// inspect planned tasks.
func (scan *Scan) planFiles(ctx context.Context, projectScanColumns bool) ([]FileScanTask, error) {
	if atomic.LoadUint32(&scan.closed) != 0 {
		return nil, fmt.Errorf("%w: scan is closed", ErrInvalidOperation)
	}

	if scan.selectorErr != nil {
		return nil, scan.selectorErr
	}

	if scan.asOfTimestamp != nil {
		snapshot, err := scan.ResolveSnapshot()
		if err != nil {
			return nil, err
		}
		scan.snapshotID = &snapshot.SnapshotID
		scan.asOfTimestamp = nil
	}

	switch scan.planningMode {
	case ScanPlanningRemote:
		return scan.planFilesRemote(ctx)
	case ScanPlanningAuto:
		if !supportsAutomaticRemotePlanning(scan.planner) {
			scanmetrics.Fallback(ctx, "capability")
		} else if scan.requiresLastUpdatedSequenceNumber() {
			scanmetrics.Fallback(ctx, "row-lineage")
		} else {
			return scan.planFilesRemote(ctx)
		}
	case ScanPlanningLocal:
	default:
		return nil, fmt.Errorf("%w: unknown scan planning mode %q", iceberg.ErrInvalidArgument, scan.planningMode)
	}

	start := time.Now()
	var acc scanMetricsAccumulator

	// Resolve the planning schema once and thread it through both planning and
	// the report, so the report describes exactly the schema that was used.
	schema, err := scan.effectiveSchema()
	if err != nil {
		return nil, err
	}

	results, err := scan.planFilesLocal(ctx, &acc, schema, projectScanColumns)
	if err != nil {
		return nil, err
	}
	// Snap the elapsed time right after planning so total-planning-duration
	// reflects planning alone, not the report assembly below.
	planningDuration := time.Since(start)

	// Only emit a report when planning ran against a real snapshot. A table with
	// no snapshot plans zero files and has no real snapshot id to report; Java
	// skips the scan report entirely in that case, so we do too.
	//
	// Building the report is also skipped for a no-op reporter — the opt-in
	// default set by Table.Scan, the nil-reporter case for scans constructed
	// directly in tests (see Reporter, which maps nil to NopReporter), and a
	// Combine of only nop reporters. A nop discards the report, so assembling
	// one would be pure overhead.
	if scan.Snapshot() != nil {
		if rep := scan.Reporter(); !metrics.IsNop(rep) {
			// Resolve the projected schema best-effort for the report's projected
			// fields. Report assembly must never fail a scan, so a projection error
			// just yields a report that omits projected fields.
			projected, _ := scan.Projection()
			safeReport(ctx, rep, scan.buildScanReport(&acc, schema, projected, planningDuration))
		}
	}

	return results, nil
}

// planFilesLocal performs local scan planning: it reads and filters the
// snapshot's manifests using schema, builds the matching FileScanTasks, and
// records planning metrics into acc for the caller to report. A successful
// local plan retires any previous remote plan; a failed local plan leaves it
// usable. It returns a nil slice (not an empty one) when there is no snapshot
// or every manifest is pruned.
func (scan *Scan) planFilesLocal(
	ctx context.Context,
	acc *scanMetricsAccumulator,
	schema *iceberg.Schema,
	projectScanColumns bool,
) (results []FileScanTask, err error) {
	defer func() {
		if err == nil {
			err = scan.closePlanIO()
		}
	}()

	snap, err := scan.ResolveSnapshot()
	if err != nil || snap == nil {
		return nil, err
	}
	// Keep the projection cache alive across both local planning phases. The
	// manifest and data-file evaluators need the same per-spec projections.
	partitionFilters := scan.partitionFiltersForSchema(schema)

	// Step 1: Retrieve filtered manifests based on snapshot and partition specs.
	manifestList, err := scan.fetchPartitionSpecFilteredManifestsWithSchema(
		ctx, snap, schema, acc, partitionFilters)
	if err != nil || len(manifestList) == 0 {
		return nil, err
	}
	if scan.canLimitLocalPlanning(acc) {
		// The manifest counters above describe partition pruning. Keep them
		// unchanged when a row limit narrows the manifest list afterwards.
		if limitedManifests, limited := limitManifestListByRows(manifestList, scan.limit); limited {
			manifestList = limitedManifests
		}
	}

	// Step 2: Read delete manifests first so data entries can be turned into
	// tasks immediately after their delete indexes are ready.
	dataManifests, deleteManifests := splitManifestList(manifestList)
	minSeqNum := minSequenceNum(manifestList)
	deleteEntries := newManifestEntries()
	if len(deleteManifests) > 0 {
		deleteEntries, err = scan.collectManifestEntriesWithSchemaMinSequenceNum(
			ctx, deleteManifests, schema, partitionFilters, minSeqNum, projectScanColumns)
		if err != nil {
			return nil, err
		}
	}
	// Equality-delete pruning is the only data-manifest consumer of column
	// metrics after the delete entries have been indexed. Positional deletes and
	// deletion vectors use path/reference metadata instead, so they should not
	// force every data file to materialize metric maps.
	retainDataFileStats := len(deleteEntries.equalityDeleteEntries) > 0
	// Step 3: Index positional deletes and match them to data files.
	posDeleteIndex, err := buildPositionalDeleteIndex(deleteEntries.positionalDeleteEntries)
	if err != nil {
		return nil, err
	}

	dvIndex, err := buildDVIndex(deleteEntries.dvEntries)
	if err != nil {
		return nil, err
	}
	eqDeleteIndex, err := buildEqualityDeleteIndex(deleteEntries.equalityDeleteEntries, scan.metadata, schema)
	if err != nil {
		return nil, err
	}

	// Step 4: Stream data entries into per-manifest task batches, then flatten
	// them in manifest order.
	plannedTasks, err := scan.planDataManifestTasksWithOptions(
		ctx, dataManifests, schema, minSeqNum, posDeleteIndex, dvIndex, eqDeleteIndex,
		projectScanColumns, retainDataFileStats)
	if err != nil {
		return nil, err
	}
	if projectScanColumns {
		// Release the full delete indexes before residual and split planning
		// retain the compact task metadata.
		deleteEntries = nil
		posDeleteIndex = nil
		dvIndex = nil
		eqDeleteIndex = nil
	}

	var boundRowFilter iceberg.BooleanExpression
	if scan.rowFilter != nil && !scan.rowFilter.Equals(iceberg.AlwaysTrue{}) {
		boundRowFilter, err = iceberg.BindExpr(schema, scan.rowFilter, scan.caseSensitive)
		if err != nil {
			return nil, err
		}
	}
	var residualEvaluators map[int]*partitionResidualEvaluator
	if boundRowFilter != nil {
		residualEvaluators = make(map[int]*partitionResidualEvaluator)
	}

	// Apply residuals, metrics, and parquet range splitting in manifest order.
	// These operations stay outside the concurrent manifest workers so the
	// residual evaluator cache and scan metrics remain race-free.
	results = make([]FileScanTask, 0, len(plannedTasks))
	splitTargetSize := scan.metadata.Properties().GetInt64(
		ReadSplitTargetSizeKey, ReadSplitTargetSizeDefault)
	acc.resultDataFiles = int64(len(plannedTasks))
	for _, task := range plannedTasks {
		if boundRowFilter != nil {
			specID := int(task.File.SpecID())
			residualEvaluator, found := residualEvaluators[specID]
			if !found {
				residualEvaluator, err = newPartitionResidualEvaluator(
					schema, scan.metadata.PartitionSpecByID(specID), boundRowFilter, scan.caseSensitive)
				if err != nil {
					return nil, fmt.Errorf("build partition residual evaluator for spec %d: %w", specID, err)
				}
				residualEvaluators[specID] = residualEvaluator
			}
			if residualEvaluator != nil {
				var simplified bool
				task.Residual, simplified, err = residualEvaluator.residual(dataFilePartition(task.File))
				if err != nil {
					return nil, fmt.Errorf("evaluate partition residual for %s: %w", task.File.FilePath(), err)
				}
				if !simplified {
					task.Residual = nil
				}
			}
		}

		acc.addResultDeleteMetrics(task)
		acc.totalFileSize += task.File.FileSizeBytes()
		if splitTasks, split := splitParquetScanTask(task, splitTargetSize); split {
			results = append(results, splitTasks...)
		} else {
			results = append(results, task)
		}
	}

	return results, nil
}

func dataFilesWithoutColumnStats(
	files []iceberg.DataFile,
	memo map[iceberg.DataFile]iceberg.DataFile,
) []iceberg.DataFile {
	if len(files) == 0 {
		return files
	}

	projected := make([]iceberg.DataFile, len(files))
	for i, file := range files {
		if file != nil && reflect.TypeOf(file).Comparable() {
			if cached, ok := memo[file]; ok {
				projected[i] = cached

				continue
			}
			projected[i] = iceberg.DataFileWithoutColumnStats(file)
			memo[file] = projected[i]

			continue
		}
		projected[i] = iceberg.DataFileWithoutColumnStats(file)
	}

	return projected
}

// canLimitLocalPlanning reports whether the manifest-list row counts are
// sufficient to safely narrow local planning for this scan. A row filter can
// remove rows from a data file, and delete manifests can remove rows at read
// time, so both cases keep the existing full planning path.
func (scan *Scan) canLimitLocalPlanning(acc *scanMetricsAccumulator) bool {
	return scan.limit > 0 &&
		(scan.rowFilter == nil || scan.rowFilter.Equals(iceberg.AlwaysTrue{})) &&
		acc.totalDeleteManifests == 0
}

// limitManifestListByRows returns the shortest manifest prefix whose live row
// counts reach limit. It falls back to the complete list when a count is
// unknown, overflows, or when every manifest is needed anyway.
func limitManifestListByRows(manifestList []iceberg.ManifestFile, limit int64) ([]iceberg.ManifestFile, bool) {
	if limit <= 0 || len(manifestList) < 2 {
		return manifestList, false
	}

	remaining := limit
	for i, manifest := range manifestList {
		addedRows, existingRows := manifest.AddedRows(), manifest.ExistingRows()
		if addedRows < 0 || existingRows < 0 || existingRows > math.MaxInt64-addedRows {
			return manifestList, false
		}

		rows := addedRows + existingRows
		if rows >= remaining {
			if i+1 == len(manifestList) {
				return manifestList, false
			}

			return manifestList[:i+1], true
		}
		remaining -= rows
	}

	return manifestList, false
}

func (scan *Scan) planFilesRemote(ctx context.Context) ([]FileScanTask, error) {
	if scan.requiresLastUpdatedSequenceNumber() {
		return nil, fmt.Errorf(
			"%w: remote scan planning cannot populate %s",
			ErrInvalidOperation,
			iceberg.LastUpdatedSequenceNumberColumnName,
		)
	}

	if scan.planner == nil || !scan.planner.SupportsRemoteScanPlanning() {
		return nil, fmt.Errorf("%w: remote scan planning is unavailable", ErrInvalidOperation)
	}

	var schema *iceberg.Schema
	if scan.metadata != nil {
		var err error
		schema, err = scan.effectiveSchema()
		if err != nil {
			return nil, err
		}
	}

	selectedFields, err := remotePlanningSelectedFields(scan, schema)
	if err != nil {
		return nil, err
	}

	caseSensitive := scan.caseSensitive
	useSnapshotSchema := scan.snapshotSchemaEnabled()
	var minRowsRequested *int64
	if scan.limit >= 0 {
		minRows := scan.limit
		minRowsRequested = &minRows
	}

	result, err := scan.planner.PlanFiles(ctx, ScanPlanningRequest{
		Identifier:        slices.Clone(scan.identifier),
		Metadata:          scan.metadata,
		Schema:            schema,
		MetadataLocation:  scan.metadataLocation,
		FileIOProperties:  maps.Clone(scan.scanPlanningIOProps),
		SnapshotID:        scan.snapshotID,
		SelectedFields:    selectedFields,
		RowFilter:         scan.rowFilter,
		MinRowsRequested:  minRowsRequested,
		MaxConcurrency:    scan.concurrency,
		CaseSensitive:     &caseSensitive,
		UseSnapshotSchema: &useSnapshotSchema,
	})
	if err != nil {
		return nil, err
	}

	// REST plans contain whole files and split offsets. Apply the same range
	// policy as local planning; already-partial tasks from other planners are
	// preserved by splitParquetScanTask.
	targetSize := int64(ReadSplitTargetSizeDefault)
	if scan.metadata != nil {
		targetSize = scan.metadata.Properties().GetInt64(ReadSplitTargetSizeKey, targetSize)
	}
	result.Tasks = splitRemoteScanTasks(result.Tasks, targetSize)

	// Replace the current plan only after the new plan is available. A planner
	// failure or invalid PlanIO must not destroy a previously usable plan.
	if scan.planIO != nil && scan.planIO.matches(result.IO) {
		return result.Tasks, nil
	}

	planIO, err := newPlanIOState(result.IO)
	if err != nil {
		return nil, err
	}

	oldPlanIO := scan.planIO
	scan.planIO = planIO
	if oldPlanIO != nil {
		_ = oldPlanIO.releaseOwner()
	}

	return result.Tasks, nil
}

// requiresLastUpdatedSequenceNumber reports whether the scan projection needs
// the manifest-entry data sequence number. The REST FileScanTask payload does
// not carry that value, so remote planning cannot safely synthesize the
// _last_updated_sequence_number metadata column for files that do not store it
// physically. Auto mode falls back to local planning; explicit remote mode
// fails before making a request rather than returning silently incomplete data.
func (scan *Scan) requiresLastUpdatedSequenceNumber() bool {
	if scan.includeRowLineage {
		return true
	}

	for _, field := range scan.selectedFields {
		if scan.caseSensitive {
			if field == iceberg.LastUpdatedSequenceNumberColumnName {
				return true
			}
		} else if strings.EqualFold(field, iceberg.LastUpdatedSequenceNumberColumnName) {
			return true
		}
	}

	return false
}

func (scan *Scan) remoteSelectedFields(schema *iceberg.Schema) []string {
	if !slices.Contains(scan.selectedFields, "*") {
		return slices.Clone(scan.selectedFields)
	}
	if schema == nil {
		return nil
	}

	fields := schema.Fields()
	selected := make([]string, 0, len(fields))
	for _, field := range fields {
		selected = append(selected, field.Name)
	}

	return selected
}

type planIOState struct {
	io PlanIO

	mu        sync.Mutex
	owners    int
	readers   int
	closeOnce sync.Once
	closeErr  error
}

func newPlanIOState(planIO PlanIO) (*planIOState, error) {
	if planIO == nil {
		return nil, nil
	}

	value := reflect.ValueOf(planIO)
	if !value.Comparable() {
		return nil, fmt.Errorf("%w: PlanIO type %T must be comparable", iceberg.ErrInvalidArgument, planIO)
	}
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if value.IsNil() {
			return nil, fmt.Errorf("%w: PlanIO must not be nil", iceberg.ErrInvalidArgument)
		}
	}

	return &planIOState{io: planIO, owners: 1}, nil
}

func (p *planIOState) matches(planIO PlanIO) bool {
	return planIO != nil && p.io == planIO
}

func (p *planIOState) retain() {
	p.mu.Lock()
	p.owners++
	p.mu.Unlock()
}

func (p *planIOState) acquire(ctx context.Context) (io.IO, func(), error) {
	p.mu.Lock()
	if p.owners == 0 {
		p.mu.Unlock()

		return nil, nil, fmt.Errorf("%w: remote scan plan is no longer current", ErrInvalidOperation)
	}
	p.readers++
	p.mu.Unlock()

	fs, err := p.io.Load(ctx)
	if err != nil {
		p.release()

		return nil, nil, err
	}

	var releaseOnce sync.Once

	return fs, func() { releaseOnce.Do(p.release) }, nil
}

func (p *planIOState) release() {
	p.mu.Lock()
	p.readers--
	closeNow := p.owners == 0 && p.readers == 0
	p.mu.Unlock()

	if closeNow {
		_ = p.close()
	}
}

func (p *planIOState) releaseOwner() error {
	p.mu.Lock()
	p.owners--
	closeNow := p.owners == 0 && p.readers == 0
	p.mu.Unlock()

	if closeNow {
		return p.close()
	}

	return nil
}

func (p *planIOState) close() error {
	p.closeOnce.Do(func() { p.closeErr = p.io.Close() })

	return p.closeErr
}

type FileScanTask struct {
	File                iceberg.DataFile
	DeleteFiles         []iceberg.DataFile // positional delete files
	EqualityDeleteFiles []iceberg.DataFile // equality delete files
	DeletionVectorFiles []iceberg.DataFile // deletion vectors (puffin files)
	Start, Length       int64
	// Residual is the portion of the scan filter that must still be evaluated
	// for this task. Local and remote planners may simplify the original filter using
	// file metadata; nil means the caller did not provide a task residual.
	// ReadTasks applies the scan's original row filter and each task residual.
	Residual iceberg.BooleanExpression

	// Row lineage (v3): constants used when reading to synthesize _row_id and _last_updated_sequence_number.
	// FirstRowID is the effective first_row_id for this file (from manifest entry, after inheritance).
	// DataSequenceNumber is the data sequence number of the file's manifest entry.
	FirstRowID         *int64
	DataSequenceNumber *int64
}

// ToArrowRecords returns the arrow schema of the expected records and an interator
// that can be used with a range expression to read the records as they are available.
// If an error is encountered, during the planning and setup then this will return the
// error directly. If the error occurs while iterating the records, it will be returned
// by the iterator.
// Deletion-vector manifest references are validated during setup. Deletion-vector
// format and blob-level validation happen when the Puffin file is read. Deletion-vector
// Puffin files are loaded lazily, so a Puffin read error is returned by the iterator
// when a task referencing that file is reached. If no such task is processed, the file
// is not read and its error is not returned.
//
// The purpose for returning the schema up front is to handle the case where there are no
// rows returned. The resulting Arrow Schema of the projection will still be known.
// The memory held while tasks are decoded in parallel is described on [Scan.ReadTasks].
func (scan *Scan) ToArrowRecords(ctx context.Context) (*arrow.Schema, iter.Seq2[arrow.RecordBatch, error], error) {
	tasks, err := scan.planFiles(ctx, true)
	if err != nil {
		return nil, nil, err
	}

	return scan.ReadTasks(ctx, tasks)
}

// ReadTasks reads Arrow records from a specific set of FileScanTasks, applying the
// scan's projection, per-task residual filters, and delete handling. This is useful
// when the caller has already planned or selected specific tasks to read.
// Positional- and equality-delete read errors are delivered through the iterator
// only if iteration reaches the task that encounters the error; a row limit or
// early termination may finish the scan before the error is observed.
// Deletion-vector manifest references are validated during setup. Deletion-vector
// format and blob-level validation happen when the Puffin file is read. Deletion-vector
// read errors are delivered through the iterator when a task referencing that file is
// reached; if no such task is processed, the file is not read and its error is not
// returned. The returned iterator is single-use.
//
// With [WithMaxConcurrency] above one, tasks are decoded in parallel and the
// batches are returned in task order. Each worker holds the decoded batches of
// at most one task until the iterator has returned them, so while an early task
// is slow to open or decode, the batches waiting behind it total at most the
// worker count times the rows in the largest task, rounded up to a multiple of
// the batch size (see [WithArrowBatchSize]). One large file among small ones can
// therefore still hold its whole decoded contents while it waits. The bound for
// a compaction pipeline is stated on [WithCompactionArrowBatchSize].
func (scan *Scan) ReadTasks(ctx context.Context, tasks []FileScanTask) (*arrow.Schema, iter.Seq2[arrow.RecordBatch, error], error) {
	if atomic.LoadUint32(&scan.closed) != 0 {
		return nil, nil, fmt.Errorf("%w: scan is closed", ErrInvalidOperation)
	}

	if scan.selectorErr != nil {
		return nil, nil, scan.selectorErr
	}
	var (
		boundFilter iceberg.BooleanExpression
		err         error
	)

	effectiveSchema, err := scan.effectiveSchema()
	if err != nil {
		return nil, nil, err
	}

	if scan.rowFilter != nil {
		boundFilter, err = iceberg.BindExpr(effectiveSchema, scan.rowFilter, scan.caseSensitive)
		if err != nil {
			return nil, nil, err
		}
	}

	schema, err := scan.Projection()
	if err != nil {
		return nil, nil, err
	}

	// Bind task residuals against the schema selected by this scan, which may
	// be an older snapshot schema rather than the table's current schema. Keep
	// the caller's task slice untouched because the same plan may be reused.
	readTasks := slices.Clone(tasks)
	for i := range readTasks {
		if readTasks[i].Residual == nil {
			continue
		}

		readTasks[i].Residual, err = bindTaskFilter(effectiveSchema,
			readTasks[i].Residual, scan.caseSensitive)
		if err != nil {
			return nil, nil, fmt.Errorf("bind residual for task %d: %w", i, err)
		}
	}

	// A plan-scoped FileIO (from remote planning) takes precedence over the
	// table's default FileIO. The iterator keeps a reader lease so replanning
	// cannot close the old plan's resources while records are still being read.
	planIO := scan.planIO
	var fs io.IO
	var releasePlanIO func()
	if planIO != nil {
		fs, releasePlanIO, err = planIO.acquire(ctx)
		if err != nil {
			return nil, nil, err
		}
	} else {
		fs, err = scan.ioF(ctx)
	}
	if err != nil {
		return nil, nil, err
	}

	outSchema, records, err := (&arrowScan{
		metadata:        scan.metadata,
		fs:              fs,
		scanSchema:      effectiveSchema,
		projectedSchema: schema,
		boundRowFilter:  boundFilter,
		filterSchema:    effectiveSchema,
		caseSensitive:   scan.caseSensitive,
		rowLimit:        scan.limit,
		options:         scan.options,
		concurrency:     scan.concurrency,
		arrowBatchSize:  scan.arrowBatchSize,
	}).GetRecords(ctx, readTasks)
	if err != nil {
		// No iterator to drive cleanup on a setup error, so release here.
		if releasePlanIO != nil {
			releasePlanIO()
		}

		return nil, nil, err
	}

	if releasePlanIO != nil {
		records = releasePlanIOAfter(records, releasePlanIO)
	}

	return outSchema, records, nil
}

// closePlanIO releases the scoped resources associated with the current
// remote plan. It is safe to call when no remote plan has been installed.
func (scan *Scan) closePlanIO() error {
	if scan.planIO == nil {
		return nil
	}

	planIO := scan.planIO
	scan.planIO = nil

	return planIO.releaseOwner()
}

// Close releases the plan-scoped resources owned by this scan. It is safe to
// call more than once. Active ReadTasks iterators retain their reader lease and
// can finish; the plan IO closes after the last lease is released. A scan must
// not be used after Close.
func (scan *Scan) Close() error {
	if scan == nil || !atomic.CompareAndSwapUint32(&scan.closed, 0, 1) {
		return nil
	}

	return scan.closePlanIO()
}

// releasePlanIOAfter wraps an arrow record iterator so its plan-scoped IO lease
// is released once iteration ends, whether the consumer exhausts the iterator
// or stops early. A caller that never ranges over the iterator does not release
// the lease; that is an accepted edge for an unread result.
func releasePlanIOAfter(seq iter.Seq2[arrow.RecordBatch, error], release func()) iter.Seq2[arrow.RecordBatch, error] {
	return func(yield func(arrow.RecordBatch, error) bool) {
		defer release()
		for rec, err := range seq {
			if !yield(rec, err) {
				return
			}
		}
	}
}

// ToArrowTable calls ToArrowRecords and then gathers all of the records together
// and returns an arrow.Table make from those records.
func (scan *Scan) ToArrowTable(ctx context.Context) (arrow.Table, error) {
	schema, itr, err := scan.ToArrowRecords(ctx)
	if err != nil {
		return nil, err
	}

	records := make([]arrow.RecordBatch, 0)
	for rec, err := range itr {
		if err != nil {
			return nil, err
		}

		defer rec.Release()
		records = append(records, rec)
	}

	return array.NewTableFromRecords(schema, records), nil
}
