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
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"maps"
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/apache/iceberg-go"
	iceinternal "github.com/apache/iceberg-go/internal"
	"github.com/google/uuid"
)

const (
	partitionFieldStartID       = 1000
	supportedTableFormatVersion = 3
	minFormatVersionRowLineage  = 3
	initialRowID                = int64(0)
	initialSequenceNumber       = int64(0)
	oneMinuteInMs               = int64(60_000)
)

func generateSnapshotID() int64 {
	return snapshotIDFromUUID(uuid.New())
}

func snapshotIDFromUUID(id uuid.UUID) int64 {
	var out [8]byte

	for i := range 8 {
		lhs, rhs := id[i], id[i+8]
		out[i] = lhs ^ rhs
	}

	return int64(binary.LittleEndian.Uint64(out[:]) & math.MaxInt64)
}

type snapshotIndexData struct {
	positions map[int64]int
	// firstSnapshot identifies the snapshot slice used to build positions.
	// It lets read-only lookups distinguish a complete index from an index
	// left behind by an in-package fixture that replaced the slice.
	firstSnapshot *Snapshot
	// shared means positions is owned by more than one builder or metadata
	// value and must be copied before a builder mutates it.
	shared bool
}

func snapshotListFirst(snapshots []Snapshot) *Snapshot {
	if len(snapshots) == 0 {
		return nil
	}

	return &snapshots[0]
}

func buildSnapshotIndex(snapshots []Snapshot) *snapshotIndexData {
	positions := make(map[int64]int, len(snapshots))
	for i, snapshot := range snapshots {
		if _, exists := positions[snapshot.SnapshotID]; !exists {
			positions[snapshot.SnapshotID] = i
		}
	}

	return &snapshotIndexData{positions: positions, firstSnapshot: snapshotListFirst(snapshots)}
}

func cloneSnapshotIndex(index *snapshotIndexData) *snapshotIndexData {
	if index == nil {
		return nil
	}

	return &snapshotIndexData{
		positions:     maps.Clone(index.positions),
		firstSnapshot: index.firstSnapshot,
	}
}

func snapshotIndexNeedsRebuild(index *snapshotIndexData, snapshots []Snapshot) bool {
	if index == nil || len(index.positions) != len(snapshots) {
		return true
	}

	return len(snapshots) > 0 && index.firstSnapshot != &snapshots[0]
}

// snapshotIndexPosition returns the position for id. Metadata loaded or built
// through the normal paths always has a current index, so the linear scan is
// only a compatibility fallback for in-package fixtures that replace the slice.
func snapshotIndexPosition(index *snapshotIndexData, snapshots []Snapshot, id int64) (int, bool) {
	if index != nil {
		if i, ok := index.positions[id]; ok {
			if i >= 0 && i < len(snapshots) && snapshots[i].SnapshotID == id {
				return i, true
			}

			index = buildSnapshotIndex(snapshots)
			if i, ok := index.positions[id]; ok {
				return i, true
			}

			return 0, false
		}

		if !snapshotIndexNeedsRebuild(index, snapshots) {
			return 0, false
		}
	}

	for i := range snapshots {
		if snapshots[i].SnapshotID == id {
			return i, true
		}
	}

	return 0, false
}

type schemaIndexData struct {
	schemas map[int]*iceberg.Schema
	// sourceCount is kept separately because duplicate IDs make the map
	// smaller than the schema slice in invalid or in-package fixture state.
	sourceCount int
	// firstSchemaSlot identifies the slice used to build the index.
	firstSchemaSlot **iceberg.Schema
	// shared means the map is owned by more than one builder or metadata value.
	shared bool
}

func schemaListFirst(schemas []*iceberg.Schema) **iceberg.Schema {
	if len(schemas) == 0 {
		return nil
	}

	return &schemas[0]
}

func buildSchemaIndex(schemas []*iceberg.Schema) *schemaIndexData {
	byID := make(map[int]*iceberg.Schema, len(schemas))
	for _, schema := range schemas {
		if schema == nil {
			continue
		}
		if _, exists := byID[schema.ID]; !exists {
			byID[schema.ID] = schema
		}
	}

	return &schemaIndexData{
		schemas:         byID,
		sourceCount:     len(schemas),
		firstSchemaSlot: schemaListFirst(schemas),
	}
}

func cloneSchemaIndex(index *schemaIndexData) *schemaIndexData {
	if index == nil {
		return nil
	}

	return &schemaIndexData{
		schemas:         maps.Clone(index.schemas),
		sourceCount:     index.sourceCount,
		firstSchemaSlot: index.firstSchemaSlot,
	}
}

func schemaIndexNeedsRebuild(index *schemaIndexData, schemas []*iceberg.Schema) bool {
	if index == nil || index.sourceCount != len(schemas) {
		return true
	}

	return index.firstSchemaSlot != schemaListFirst(schemas)
}

func schemaIndexLookup(index *schemaIndexData, schemas []*iceberg.Schema, id int) (*iceberg.Schema, bool) {
	if index != nil {
		if schema, ok := index.schemas[id]; ok && schema != nil && schema.ID == id {
			return schema, true
		}
	}

	// Builder lookups return mutable schemas, so a changed ID may not have
	// an entry in the index even when the slice itself is unchanged.
	for _, schema := range schemas {
		if schema != nil && schema.ID == id {
			return schema, true
		}
	}

	return nil, false
}

type partitionSpecIndexData struct {
	positions map[int]int
	// firstSpec identifies the first element of the spec slice used to build
	// positions. It lets read-only lookups detect an index left behind by an
	// in-package fixture that replaced the slice.
	firstSpec *iceberg.PartitionSpec
	// shared means positions is owned by more than one builder or metadata
	// value and must be copied before a builder mutates it.
	shared bool
}

// Small spec slices are faster to search directly than to hash-map lookup.
// 32 is a conservative cutoff; keep it aligned with the lookup benchmarks.
const partitionSpecIndexMinSize = 32

func partitionSpecListFirst(specs []iceberg.PartitionSpec) *iceberg.PartitionSpec {
	if len(specs) == 0 {
		return nil
	}

	return &specs[0]
}

func buildPartitionSpecIndex(specs []iceberg.PartitionSpec) *partitionSpecIndexData {
	if len(specs) < partitionSpecIndexMinSize {
		return nil
	}

	positions := make(map[int]int, len(specs))
	for i := range specs {
		id := specs[i].ID()
		if _, exists := positions[id]; !exists {
			positions[id] = i
		}
	}

	return &partitionSpecIndexData{
		positions: positions,
		firstSpec: partitionSpecListFirst(specs),
	}
}

func clonePartitionSpecIndex(index *partitionSpecIndexData) *partitionSpecIndexData {
	if index == nil {
		return nil
	}

	return &partitionSpecIndexData{
		positions: maps.Clone(index.positions),
		firstSpec: index.firstSpec,
	}
}

func partitionSpecIndexNeedsRebuild(index *partitionSpecIndexData, specs []iceberg.PartitionSpec) bool {
	if len(specs) < partitionSpecIndexMinSize {
		return false
	}
	if index == nil {
		return true
	}
	// Persisted metadata rejects duplicate IDs, so map and source cardinality
	// are equal on the indexed read path.
	if index.positions == nil || len(index.positions) != len(specs) {
		return true
	}

	return index.firstSpec != partitionSpecListFirst(specs)
}

// partitionSpecIndexPosition returns the position for id. The map is the fast
// path, while the linear scan preserves lookup behavior if an in-package
// fixture mutates a spec in place without rebuilding the derived index. An
// absent map entry is indistinguishable from an in-place mutation that added a
// new ID, so misses intentionally scan the slice too.
func partitionSpecIndexPosition(index *partitionSpecIndexData, specs []iceberg.PartitionSpec, id int) (int, bool) {
	if index != nil && len(specs) >= partitionSpecIndexMinSize {
		if i, ok := index.positions[id]; ok {
			if i >= 0 && i < len(specs) && specs[i].ID() == id {
				return i, true
			}
		}
	}

	for i := range specs {
		if specs[i].ID() == id {
			return i, true
		}
	}

	return 0, false
}

// Metadata for an iceberg table as specified in the Iceberg spec
//
// https://iceberg.apache.org/spec/#iceberg-table-spec
type Metadata interface {
	// Version indicates the version of this metadata, 1 for V1, 2 for V2, etc.
	Version() int
	// TableUUID returns a UUID that identifies the table, generated when the
	// table is created. Implementations must throw an exception if a table's
	// UUID does not match the expected UUID after refreshing metadata.
	TableUUID() uuid.UUID
	// Location is the table's base location. This is used by writerFactory to determine
	// where to store data files, manifest files, and table metadata files.
	Location() string
	// LastUpdatedMillis is the timestamp in milliseconds from the unix epoch when
	// the table was last updated. Each table metadata file should update this
	// field just before writing.
	LastUpdatedMillis() int64
	// LastColumnID returns the highest assigned column ID for the table.
	// This is used to ensure fields are always assigned an unused ID when
	// evolving schemas.
	LastColumnID() int
	// Schemas returns the list of schemas, stored as objects with their
	// schema-id.
	Schemas() []*iceberg.Schema
	// CurrentSchema returns the table's current schema.
	CurrentSchema() *iceberg.Schema
	// PartitionSpecs returns the list of all partition specs in the table.
	PartitionSpecs() []iceberg.PartitionSpec
	// PartitionSpec returns the current partition spec that the table is using.
	PartitionSpec() iceberg.PartitionSpec
	// PartitionSpecByID returns the partition spec with the given ID. Returns
	// nil if the ID is not found in the list of partition specs.
	PartitionSpecByID(int) *iceberg.PartitionSpec
	// DefaultPartitionSpec is the ID of the current spec that writerFactory should
	// use by default.
	DefaultPartitionSpec() int
	// LastPartitionSpecID returns the persisted last assigned partition field ID,
	// which may be stale relative to partition spec history.
	LastPartitionSpecID() *int
	// Snapshots returns the list of valid snapshots. Valid snapshots are
	// snapshots for which all data files exist in the file system. A data
	// file must not be deleted from the file system until the last snapshot
	// in which it was listed is garbage collected.
	Snapshots() []Snapshot
	// SnapshotByID find and return a specific snapshot by its ID. Returns
	// nil if the ID is not found in the list of snapshots.
	SnapshotByID(int64) *Snapshot
	// SnapshotByName searches the list of snapshots for a snapshot with a given
	// ref name. Returns nil if there's no ref with this name for a snapshot.
	SnapshotByName(name string) *Snapshot
	// CurrentSnapshot returns the table's current snapshot.
	CurrentSnapshot() *Snapshot
	// Ref returns the snapshot ref for the main branch.
	Ref() SnapshotRef
	// Refs returns a list of snapshot name/reference pairs.
	Refs() iter.Seq2[string, SnapshotRef]
	// SnapshotLogs returns the list of snapshot logs for the table.
	SnapshotLogs() iter.Seq[SnapshotLogEntry]
	// SortOrder returns the table's current sort order, ie: the one with the
	// ID that matches the default-sort-order-id.
	SortOrder() SortOrder
	// SortOrders returns the list of sort orders in the table.
	SortOrders() []SortOrder
	// DefaultSortOrder returns the ID of the current sort order that writerFactory
	// should use by default.
	DefaultSortOrder() int
	// Properties is a string to string map of table properties. This is used
	// to control settings that affect reading and writing and is not intended
	// to be used for arbitrary metadata. For example, commit.retry.num-retries
	// is used to control the number of commit retries.
	Properties() iceberg.Properties
	// PreviousFiles returns the list of metadata log entries for the table.
	PreviousFiles() iter.Seq[MetadataLogEntry]
	Equals(Metadata) bool

	NameMapping() iceberg.NameMapping

	LastSequenceNumber() int64
	// NextRowID returns the next available row ID for v3 tables.
	// Returns 0 for v1/v2 tables or if not set.
	NextRowID() int64
	// Statistics returns an optional list of table statistics.
	// Table statistics files are valid Puffin files.
	// StatisticsFile are informational. A reader can choose to ignore statistics information.
	// StatisticsFile support is not required to read the table correctly.
	// A table can contain many statistics files associated with different table snapshots.
	Statistics() iter.Seq[StatisticsFile]
	// PartitionStatistics returns an optional list of partition statistics files.
	// Partition statistics are not required for reading or planning
	// and readers may ignore them. Each table snapshot may be associated
	// with at most one partition statistics file. A writer can optionally
	// write the partition statistics file during each write operation,
	// or it can also be computed on demand.
	PartitionStatistics() iter.Seq[PartitionStatisticsFile]
	// EncryptionKeys returns the list of encryption keys stored in table metadata (V3+).
	// Returns an empty sequence for V1/V2 tables.
	EncryptionKeys() iter.Seq[EncryptionKey]
}

// MetadataBuilder is a struct used for building and updating Iceberg table metadata.
//
// It keeps track of applied changes in the `updates` field. This can be used to commit changes made to a table to the
// catalog.
type MetadataBuilder struct {
	base    Metadata
	updates []Update

	// common fields
	formatVersion      int
	uuid               uuid.UUID
	loc                string
	lastUpdatedMS      int64
	lastColumnId       int
	schemaList         []*iceberg.Schema
	schemaIndex        *schemaIndexData // Derived from schemaList; not serialized.
	currentSchemaID    int
	specs              []iceberg.PartitionSpec
	partitionSpecIndex *partitionSpecIndexData // Derived from specs; not serialized.
	defaultSpecID      int
	lastPartitionID    *int
	props              iceberg.Properties
	snapshotList       []Snapshot
	snapshotIndex      *snapshotIndexData // Derived from snapshotList; not serialized.
	currentSnapshotID  *int64
	snapshotLog        []SnapshotLogEntry
	metadataLog        []MetadataLogEntry
	sortOrderList      []SortOrder
	defaultSortOrderID int
	refs               map[string]SnapshotRef
	statisticsList     []StatisticsFile
	partitionStatsList []PartitionStatisticsFile
	encryptionKeyList  []EncryptionKey

	previousFileEntry *MetadataLogEntry
	// >v1 specific
	lastSequenceNumber *int64
	// >v2 specific (V3)
	nextRowID *int64
	// update tracking
	lastAddedSchemaID    *int
	lastAddedPartitionID *int
	lastAddedSortOrderID *int
}

func NewMetadataBuilder(formatVersion int) (*MetadataBuilder, error) {
	if formatVersion < 1 || formatVersion > supportedTableFormatVersion {
		return nil, fmt.Errorf("%w: %d", iceberg.ErrInvalidFormatVersion, formatVersion)
	}

	return &MetadataBuilder{
		updates:            make([]Update, 0),
		schemaList:         make([]*iceberg.Schema, 0),
		schemaIndex:        buildSchemaIndex(nil),
		specs:              make([]iceberg.PartitionSpec, 0),
		partitionSpecIndex: buildPartitionSpecIndex(nil),
		props:              make(iceberg.Properties),
		snapshotList:       make([]Snapshot, 0),
		snapshotIndex:      buildSnapshotIndex(nil),
		snapshotLog:        make([]SnapshotLogEntry, 0),
		metadataLog:        make([]MetadataLogEntry, 0),
		sortOrderList:      make([]SortOrder, 0),
		refs:               make(map[string]SnapshotRef),
		encryptionKeyList:  make([]EncryptionKey, 0),
		currentSchemaID:    -1,
		defaultSortOrderID: -1,
		defaultSpecID:      -1,
		lastColumnId:       -1,
		formatVersion:      formatVersion,
	}, nil
}

type metadataBuilderSource interface {
	metadataBuilderCommon() *commonMetadata
}

// MetadataBuilderFromBase creates a MetadataBuilder from an existing Metadata object.
// currentFileLocation is the location where the current version of the metadata
// file is stored. This is used to update the metadata log. If currentFileLocation is
// empty, the metadata log will not be updated. This should only be used to stage-create tables.
func MetadataBuilderFromBase(metadata Metadata, currentFileLocation string) (*MetadataBuilder, error) {
	b := &MetadataBuilder{}
	b.base = metadata

	b.formatVersion = metadata.Version()
	b.uuid = metadata.TableUUID()
	b.loc = metadata.Location()
	b.lastUpdatedMS = 0
	b.lastColumnId = metadata.LastColumnID()

	if source, ok := metadata.(metadataBuilderSource); ok {
		common := source.metadataBuilderCommon()
		if common == nil || !slices.ContainsFunc(common.SchemaList, func(schema *iceberg.Schema) bool {
			return schema != nil && schema.ID == common.CurrentSchemaID
		}) {
			return nil, fmt.Errorf("%w: current schema is missing", ErrInvalidMetadata)
		}

		b.currentSchemaID = common.CurrentSchemaID
		b.defaultSpecID = metadata.DefaultPartitionSpec()
		b.defaultSortOrderID = metadata.DefaultSortOrder()
		b.schemaList = cloneSchemas(common.SchemaList)
		b.specs = clonePartitionSpecs(common.Specs)
		b.lastPartitionID = clonePtr(common.LastPartitionID)
		b.props = maps.Clone(common.Props)
		if b.props == nil {
			b.props = iceberg.Properties{}
		}
		snapshots, err := common.snapshotsForMarshal()
		if err != nil {
			return nil, err
		}
		b.snapshotList = cloneSnapshots(snapshots)
		b.snapshotLog = cloneCollected(common.SnapshotLog)
		b.metadataLog = cloneCollected(common.MetadataLog)
		b.sortOrderList = cloneSortOrders(common.SortOrderList)
		b.refs = cloneSnapshotRefs(common.SnapshotRefs)
		if len(common.StatisticsList) > 0 {
			b.statisticsList = cloneStatisticsFiles(common.StatisticsList)
		}
		b.partitionStatsList = cloneCollected(common.PartitionStatsList)
		if len(common.EncryptionKeyList) > 0 {
			b.encryptionKeyList = cloneEncryptionKeys(common.EncryptionKeyList)
		}
		b.snapshotIndex = buildSnapshotIndex(b.snapshotList)
		if metadata.Version() > 1 {
			seq := metadata.LastSequenceNumber()
			b.lastSequenceNumber = &seq
		}
		if metadata.Version() >= 3 {
			nextRowID := metadata.NextRowID()
			b.nextRowID = &nextRowID
		}
		if common.CurrentSnapshotID != nil {
			if _, ok := snapshotIndexPosition(b.snapshotIndex, b.snapshotList, *common.CurrentSnapshotID); ok {
				b.currentSnapshotID = clonePtr(common.CurrentSnapshotID)
			}
		}
	} else {
		b.schemaList = slices.Clone(metadata.Schemas())
		currentSchema := metadata.CurrentSchema()
		if currentSchema == nil {
			return nil, fmt.Errorf("%w: current schema is missing", ErrInvalidMetadata)
		}
		b.currentSchemaID = currentSchema.ID
		b.specs = slices.Clone(metadata.PartitionSpecs())
		b.defaultSpecID = metadata.DefaultPartitionSpec()
		b.lastPartitionID = metadata.LastPartitionSpecID()
		b.props = maps.Clone(metadata.Properties())
		b.snapshotList = slices.Clone(metadata.Snapshots())
		b.snapshotIndex = buildSnapshotIndex(b.snapshotList)
		b.sortOrderList = slices.Clone(metadata.SortOrders())
		b.defaultSortOrderID = metadata.DefaultSortOrder()
		if metadata.Version() > 1 {
			seq := metadata.LastSequenceNumber()
			b.lastSequenceNumber = &seq
		}
		if metadata.Version() >= 3 {
			nextRowID := metadata.NextRowID()
			b.nextRowID = &nextRowID
		}
		if metadata.CurrentSnapshot() != nil {
			b.currentSnapshotID = &metadata.CurrentSnapshot().SnapshotID
		}
		b.refs = maps.Collect(metadata.Refs())
		b.snapshotLog = slices.Collect(metadata.SnapshotLogs())
		b.metadataLog = slices.Collect(metadata.PreviousFiles())
		b.statisticsList = slices.Collect(metadata.Statistics())
		b.partitionStatsList = slices.Collect(metadata.PartitionStatistics())
		b.encryptionKeyList = slices.Collect(metadata.EncryptionKeys())
	}
	b.schemaIndex = buildSchemaIndex(b.schemaList)

	b.partitionSpecIndex = buildPartitionSpecIndex(b.specs)

	if currentFileLocation != "" {
		b.previousFileEntry = &MetadataLogEntry{
			MetadataFile: currentFileLocation,
			TimestampMs:  metadata.LastUpdatedMillis(),
		}
	}

	return b, nil
}

// clonePtr returns a pointer to a shallow copy of *value, or nil if value is nil.
func clonePtr[T any](value *T) *T {
	if value == nil {
		return nil
	}

	cloned := *value

	return &cloned
}

// clone returns a metadata-builder copy suitable for staging updates.
// The copy shares only immutable data with the original builder and keeps
// value types separated so partial update attempts remain atomic.
func (b *MetadataBuilder) clone() *MetadataBuilder {
	if b == nil {
		return nil
	}

	cloned := &MetadataBuilder{
		base:                 b.base,
		updates:              append([]Update(nil), b.updates...),
		formatVersion:        b.formatVersion,
		uuid:                 b.uuid,
		loc:                  b.loc,
		lastUpdatedMS:        b.lastUpdatedMS,
		lastColumnId:         b.lastColumnId,
		schemaList:           slices.Clone(b.schemaList),
		currentSchemaID:      b.currentSchemaID,
		specs:                slices.Clone(b.specs),
		defaultSpecID:        b.defaultSpecID,
		lastPartitionID:      clonePtr(b.lastPartitionID),
		props:                maps.Clone(b.props),
		snapshotList:         slices.Clone(b.snapshotList),
		currentSnapshotID:    clonePtr(b.currentSnapshotID),
		snapshotLog:          slices.Clone(b.snapshotLog),
		metadataLog:          slices.Clone(b.metadataLog),
		sortOrderList:        slices.Clone(b.sortOrderList),
		defaultSortOrderID:   b.defaultSortOrderID,
		refs:                 maps.Clone(b.refs),
		statisticsList:       slices.Clone(b.statisticsList),
		partitionStatsList:   slices.Clone(b.partitionStatsList),
		encryptionKeyList:    slices.Clone(b.encryptionKeyList),
		previousFileEntry:    clonePtr(b.previousFileEntry),
		lastSequenceNumber:   clonePtr(b.lastSequenceNumber),
		nextRowID:            clonePtr(b.nextRowID),
		lastAddedSchemaID:    clonePtr(b.lastAddedSchemaID),
		lastAddedPartitionID: clonePtr(b.lastAddedPartitionID),
		lastAddedSortOrderID: clonePtr(b.lastAddedSortOrderID),
	}
	if b.schemaIndex != nil {
		cloned.schemaIndex = &schemaIndexData{
			schemas:         b.schemaIndex.schemas,
			sourceCount:     b.schemaIndex.sourceCount,
			firstSchemaSlot: schemaListFirst(cloned.schemaList),
			shared:          true,
		}
		b.schemaIndex.shared = true
	}
	if b.partitionSpecIndex != nil {
		cloned.partitionSpecIndex = &partitionSpecIndexData{
			positions: b.partitionSpecIndex.positions,
			firstSpec: partitionSpecListFirst(cloned.specs),
			shared:    true,
		}
		b.partitionSpecIndex.shared = true
	}
	if b.snapshotIndex != nil {
		cloned.snapshotIndex = &snapshotIndexData{
			positions:     b.snapshotIndex.positions,
			firstSnapshot: snapshotListFirst(cloned.snapshotList),
			shared:        true,
		}
		b.snapshotIndex.shared = true
	}

	return cloned
}

func (b *MetadataBuilder) HasChanges() bool { return len(b.updates) > 0 }

func (b *MetadataBuilder) CurrentSpec() (*iceberg.PartitionSpec, error) {
	return b.GetSpecByID(b.defaultSpecID)
}

func (b *MetadataBuilder) CurrentSchema() *iceberg.Schema {
	s, _ := b.GetSchemaByID(b.currentSchemaID)

	return s
}

func (b *MetadataBuilder) LastUpdatedMS() int64 { return b.lastUpdatedMS }

// LastColumnID returns the highest field id ever assigned in this table's
// lifetime, as tracked by the Iceberg spec's last-column-id counter.
func (b *MetadataBuilder) LastColumnID() int { return b.lastColumnId }

func (b *MetadataBuilder) nextSequenceNumber() int64 {
	if b.formatVersion > 1 {
		if b.lastSequenceNumber == nil {
			return 0
		}

		return *b.lastSequenceNumber + 1
	}

	return 0
}

func (b *MetadataBuilder) newSnapshotID() int64 {
	b.ensureSnapshotIndex()

	for {
		snapshotID := generateSnapshotID()
		if _, exists := b.snapshotIndex.positions[snapshotID]; !exists {
			return snapshotID
		}
	}
}

func (b *MetadataBuilder) ensureSchemaIndex() {
	if schemaIndexNeedsRebuild(b.schemaIndex, b.schemaList) {
		b.schemaIndex = buildSchemaIndex(b.schemaList)
	}
}

func (b *MetadataBuilder) ensureSchemaIndexMutable() {
	b.ensureSchemaIndex()
	if b.schemaIndex.shared {
		b.schemaIndex = cloneSchemaIndex(b.schemaIndex)
	}
}

func (b *MetadataBuilder) ensureSnapshotIndex() {
	if snapshotIndexNeedsRebuild(b.snapshotIndex, b.snapshotList) {
		b.snapshotIndex = buildSnapshotIndex(b.snapshotList)
	}
}

func (b *MetadataBuilder) ensureSnapshotIndexMutable() {
	b.ensureSnapshotIndex()
	if b.snapshotIndex.shared {
		b.snapshotIndex = cloneSnapshotIndex(b.snapshotIndex)
	}
}

func (b *MetadataBuilder) ensurePartitionSpecIndex() {
	if partitionSpecIndexNeedsRebuild(b.partitionSpecIndex, b.specs) {
		b.partitionSpecIndex = buildPartitionSpecIndex(b.specs)
	}
}

func (b *MetadataBuilder) ensurePartitionSpecIndexMutable() {
	b.ensurePartitionSpecIndex()
	if b.partitionSpecIndex != nil && b.partitionSpecIndex.shared {
		b.partitionSpecIndex = clonePartitionSpecIndex(b.partitionSpecIndex)
	}
}

func (b *MetadataBuilder) currentSnapshot() *Snapshot {
	if b.currentSnapshotID == nil {
		return nil
	}

	s, _ := b.SnapshotByID(*b.currentSnapshotID)

	return s
}

// currentSnapshotForRef returns the snapshot a write targeting ref must be
// parented on (createSnapshotProducer) and plan against
// (Transaction.planningSnapshot). An unknown branch falls back to
// currentSnapshot() so a new branch forks from main. This is a head lookup
// only: the commit's AssertRefSnapshotID requirement is built
// separately from the base table's ref (Transaction.baseRefSnapshotID), which
// returns nil for an absent branch so the requirement can prove the branch does
// not exist yet — never derive one from the other. A ref cannot outlive its
// snapshot: SetSnapshotRef rejects unknown snapshot ids and checkRefsExist
// rejects loaded metadata whose ref dangles, so the lookup below cannot turn a
// dangling ref into a parentless write.
func (b *MetadataBuilder) currentSnapshotForRef(ref string) *Snapshot {
	if ref == "" || ref == MainBranch {
		return b.currentSnapshot()
	}

	r, ok := b.refs[ref]
	if !ok {
		return b.currentSnapshot()
	}

	s, _ := b.SnapshotByID(r.SnapshotID)

	return s
}

func (b *MetadataBuilder) AddSchema(schema *iceberg.Schema) error {
	if err := checkSchemaCompatibility(schema, b.formatVersion); err != nil {
		return err
	}

	newSchemaID := b.reuseOrCreateNewSchemaID(schema)

	if _, err := b.GetSchemaByID(newSchemaID); err == nil {
		if b.lastAddedSchemaID == nil || *b.lastAddedSchemaID != newSchemaID {
			b.updates = append(b.updates, NewAddSchemaUpdate(schema))
			b.lastAddedSchemaID = &newSchemaID
		}

		return nil
	}

	b.lastColumnId = max(b.lastColumnId, schema.HighestFieldID())

	schema.ID = newSchemaID

	b.ensureSchemaIndexMutable()
	b.schemaList = append(b.schemaList, schema)
	b.schemaIndex.schemas[newSchemaID] = schema
	b.schemaIndex.sourceCount = len(b.schemaList)
	b.schemaIndex.firstSchemaSlot = schemaListFirst(b.schemaList)
	b.updates = append(b.updates, NewAddSchemaUpdate(schema))
	b.lastAddedSchemaID = &newSchemaID

	return nil
}

// partitionFieldIDFloor returns an allocation floor that accounts for the
// persisted counter and all partition field IDs in spec history.
func partitionFieldIDFloor(lastPartitionID *int, specs []iceberg.PartitionSpec) int {
	floor := iceberg.PartitionDataIDStart - 1
	if lastPartitionID != nil {
		floor = max(floor, *lastPartitionID)
	}
	for _, spec := range specs {
		floor = max(floor, spec.LastAssignedFieldID())
	}

	return floor
}

func (b *MetadataBuilder) AddPartitionSpec(spec *iceberg.PartitionSpec, initial bool) error {
	newSpecID := b.reuseOrCreateNewPartitionSpecID(*spec)
	curSchema := b.CurrentSchema()
	if curSchema == nil {
		return errors.New("can't add sort order with no current schema")
	}

	fieldIDFloor := partitionFieldIDFloor(b.lastPartitionID, b.specs)
	freshSpec, err := spec.BindToSchema(curSchema, &fieldIDFloor, &newSpecID)
	if err != nil {
		return err
	}

	spec = nil
	if _, err := b.GetSpecByID(newSpecID); err == nil {
		if b.lastAddedPartitionID == nil || *b.lastAddedPartitionID != newSpecID {
			b.updates = append(b.updates, NewAddPartitionSpecUpdate(&freshSpec, initial))
			b.lastAddedPartitionID = &newSpecID
		}

		return nil
	}

	maxFieldID := 0
	fieldCount := 0
	for _, f := range freshSpec.Fields() {
		maxFieldID = max(maxFieldID, f.FieldID)
		if b.formatVersion <= 1 {
			expectedID := partitionFieldStartID + fieldCount
			if f.FieldID != expectedID {
				return fmt.Errorf("v1 constraint: partition field IDs are not sequential: expected %d, got %d", expectedID, f.FieldID)
			}
			fieldCount++
		}

	}

	lastPartitionID := max(maxFieldID, fieldIDFloor)

	if initial {
		b.specs = []iceberg.PartitionSpec{freshSpec}
		b.partitionSpecIndex = buildPartitionSpecIndex(b.specs)
	} else {
		b.ensurePartitionSpecIndexMutable()
		b.specs = append(b.specs, freshSpec)
		if len(b.specs) >= partitionSpecIndexMinSize {
			if len(b.specs) == partitionSpecIndexMinSize ||
				b.partitionSpecIndex == nil || b.partitionSpecIndex.positions == nil {
				b.partitionSpecIndex = buildPartitionSpecIndex(b.specs)
			} else {
				b.partitionSpecIndex.positions[newSpecID] = len(b.specs) - 1
				b.partitionSpecIndex.firstSpec = partitionSpecListFirst(b.specs)
			}
		}
	}

	b.lastPartitionID = &lastPartitionID
	b.lastAddedPartitionID = &newSpecID
	b.updates = append(b.updates, NewAddPartitionSpecUpdate(&freshSpec, initial))

	return nil
}

func (b *MetadataBuilder) AddSnapshot(snapshot *Snapshot) error {
	return b.addSnapshotInternal(snapshot, nil)
}

// AddSnapshotUpdate adds a snapshot to the builder and stores the supplied
// *addSnapshotUpdate as the corresponding entry in builder.updates, preserving
// runtime-only fields (such as the manifest-list rebuild closure used by the
// OCC retry path) that would be lost if a fresh update object were constructed.
//
// Callers without runtime-only fields should keep using AddSnapshot, which
// constructs a default *addSnapshotUpdate internally.
func (b *MetadataBuilder) AddSnapshotUpdate(u *addSnapshotUpdate) error {
	if u == nil {
		return nil
	}

	return b.addSnapshotInternal(u.Snapshot, u)
}

// addSnapshotInternal contains the shared validation and bookkeeping for both
// AddSnapshot and AddSnapshotUpdate. When preserveUpdate is non-nil it is
// appended to b.updates verbatim; otherwise a fresh *addSnapshotUpdate is
// created via NewAddSnapshotUpdate.
func (b *MetadataBuilder) addSnapshotInternal(snapshot *Snapshot, preserveUpdate *addSnapshotUpdate) error {
	if snapshot == nil {
		return nil
	}

	lastSeqNum := b.currentLastSequenceNumber()
	if len(b.schemaList) == 0 {
		return errors.New("can't add snapshot with no added schemas")
	} else if len(b.specs) == 0 {
		return errors.New("can't add snapshot with no added partition specs")
	} else if s, _ := b.SnapshotByID(snapshot.SnapshotID); s != nil {
		return fmt.Errorf("can't add snapshot with id %d, already exists", snapshot.SnapshotID)
	} else if b.formatVersion >= 2 &&
		snapshot.ParentSnapshotID != nil &&
		snapshot.SequenceNumber <= lastSeqNum {
		return fmt.Errorf("can't add snapshot with sequence number %d, must be > than last sequence number %d",
			snapshot.SequenceNumber, lastSeqNum)
	}

	if len(b.snapshotLog) > 0 {
		last := b.snapshotLog[len(b.snapshotLog)-1]
		if (snapshot.TimestampMs - last.TimestampMs) < -oneMinuteInMs {
			return fmt.Errorf("invalid snapshot timestamp %d: before last snapshot timestamp %d",
				snapshot.TimestampMs, last.TimestampMs)
		}
	}

	// b.base is nil for builders created by NewMetadataBuilder (the create-table path);
	// only MetadataBuilderFromBase populates it. Treat the missing base as "no previous update"
	// instead of dereferencing it, mirroring the fallback in currentNextRowID.
	var baseLastUpdated int64
	if b.base != nil {
		baseLastUpdated = b.base.LastUpdatedMillis()
	}
	maxTS := max(b.lastUpdatedMS, baseLastUpdated)
	if snapshot.TimestampMs-maxTS < -oneMinuteInMs {
		return fmt.Errorf("invalid snapshot timestamp %d: before last updated timestamp %d",
			snapshot.TimestampMs, maxTS)
	}

	if err := b.validateAndUpdateRowLineage(snapshot); err != nil {
		return err
	}

	upd := preserveUpdate
	if upd == nil {
		upd = NewAddSnapshotUpdate(snapshot)
	}
	b.updates = append(b.updates, upd)
	b.lastUpdatedMS = snapshot.TimestampMs
	b.lastSequenceNumber = &snapshot.SequenceNumber
	b.ensureSnapshotIndexMutable()
	b.snapshotList = append(b.snapshotList, *snapshot)
	b.snapshotIndex.positions[snapshot.SnapshotID] = len(b.snapshotList) - 1
	b.snapshotIndex.firstSnapshot = snapshotListFirst(b.snapshotList)

	return nil
}

func (b *MetadataBuilder) validateAndUpdateRowLineage(snapshot *Snapshot) error {
	if b.formatVersion < minFormatVersionRowLineage {
		return nil
	}

	if err := snapshot.ValidateRowLineage(); err != nil {
		return err
	}

	if snapshot.FirstRowID == nil {
		return fmt.Errorf("%w: first-row-id is required for v3 snapshots", ErrInvalidRowLineage)
	}

	nextRowID := b.currentNextRowID()
	if *snapshot.FirstRowID < nextRowID {
		return fmt.Errorf("%w: first-row-id %d is behind table next-row-id %d",
			ErrInvalidRowLineage, *snapshot.FirstRowID, nextRowID)
	}

	// next-row-id is non-decreasing across snapshots, not strictly increasing:
	// zero-added-rows snapshots (delete-only or metadata-only commits) are
	// spec-legal and leave the cursor unchanged, so this accepts a sum equal to
	// the current cursor rather than requiring it to advance. At this point
	// AddedRows is non-nil and non-negative (guaranteed by the FirstRowID != nil
	// check above plus Snapshot.ValidateRowLineage), and nextRowID is
	// non-negative per spec, so the only thing left to guard is int64 overflow
	// of the running cursor.
	if *snapshot.AddedRows > math.MaxInt64-nextRowID {
		return fmt.Errorf("%w: adding %d rows to next-row-id %d overflows int64",
			ErrInvalidRowLineage, *snapshot.AddedRows, nextRowID)
	}

	newNextRowID := nextRowID + *snapshot.AddedRows
	b.nextRowID = &newNextRowID

	return nil
}

// currentLastSequenceNumber returns the last sequence number assigned to a
// snapshot. b.lastSequenceNumber is nil for v1 tables and for builders created
// by NewMetadataBuilder that have not added a snapshot yet, in which case the
// spec's initial sequence number is the correct floor.
func (b *MetadataBuilder) currentLastSequenceNumber() int64 {
	if b.lastSequenceNumber != nil {
		return *b.lastSequenceNumber
	}

	return initialSequenceNumber
}

func (b *MetadataBuilder) currentNextRowID() int64 {
	if b.nextRowID != nil {
		return *b.nextRowID
	}
	if b.base != nil {
		return b.base.NextRowID()
	}

	return initialRowID
}

// NextRowID returns the next available row ID (for v3 row lineage). For v1/v2 returns 0.
func (b *MetadataBuilder) NextRowID() int64 {
	return b.currentNextRowID()
}

func (b *MetadataBuilder) RemoveSnapshots(snapshotIds []int64, postCommit bool) error {
	removedIDs := make(map[int64]struct{}, len(snapshotIds))
	for _, snapshotID := range snapshotIds {
		removedIDs[snapshotID] = struct{}{}
	}

	if b.currentSnapshotID != nil {
		if _, ok := removedIDs[*b.currentSnapshotID]; ok {
			return errors.New("current snapshot cannot be removed")
		}
	}

	previousSnapshotCount := len(b.snapshotList)
	b.snapshotList = slices.DeleteFunc(b.snapshotList, func(e Snapshot) bool {
		_, ok := removedIDs[e.SnapshotID]

		return ok
	})
	if len(b.snapshotList) != previousSnapshotCount {
		b.snapshotIndex = buildSnapshotIndex(b.snapshotList)
	}
	// Snapshot-log pruning is deferred to updateSnapshotLog during Build so
	// removed entries remain available when trimming history gaps.

	validSnapshotIDs := make(map[int64]struct{}, len(b.snapshotList))
	for _, snapshot := range b.snapshotList {
		validSnapshotIDs[snapshot.SnapshotID] = struct{}{}
	}

	newRefs := make(map[string]SnapshotRef, len(b.refs))
	for name, ref := range b.refs {
		if _, ok := validSnapshotIDs[ref.SnapshotID]; ok {
			newRefs[name] = ref
		}
	}
	b.refs = newRefs

	// Prune statistics entries whose snapshot was removed so that table
	// metadata does not retain stale statistics references.
	b.statisticsList = slices.DeleteFunc(b.statisticsList, func(e StatisticsFile) bool {
		_, ok := removedIDs[e.SnapshotID]

		return ok
	})
	b.partitionStatsList = slices.DeleteFunc(b.partitionStatsList, func(e PartitionStatisticsFile) bool {
		_, ok := removedIDs[e.SnapshotID]

		return ok
	})

	b.updates = append(b.updates, NewRemoveSnapshotsUpdate(snapshotIds, postCommit))

	return nil
}

func (b *MetadataBuilder) AddSortOrder(sortOrder *SortOrder) error {
	curSchema := b.CurrentSchema()
	if curSchema == nil {
		return errors.New("can't add sort order with no current schema")
	}

	newOrderID := b.reuseOrCreateNewSortOrderID(sortOrder)
	if _, err := b.GetSortOrderByID(newOrderID); err == nil {
		// The order already exists in this metadata: reuse its ID and
		// do NOT emit an add-sort-order update (REST catalogs reject
		// adding a sort order under an existing ID). Record it as
		// last-added so a following SetDefaultSortOrderID(-1) resolves
		// to the reused ID; SetDefaultSortOrderID encodes the concrete
		// ID in that case because no add update precedes it.
		sortOrder.orderID = newOrderID
		b.lastAddedSortOrderID = &newOrderID

		return nil
	}
	sortOrder.orderID = newOrderID

	sortOrders := b.sortOrderList
	if err := sortOrder.CheckCompatibility(curSchema); err != nil {
		return fmt.Errorf("sort order %s is not compatible with current schema: %w", sortOrder, err)
	}

	for _, s := range sortOrders {
		if s.OrderID() == sortOrder.OrderID() {
			return fmt.Errorf("sort order with id %d already exists", sortOrder.orderID)
		}
	}

	b.sortOrderList = append(sortOrders, *sortOrder)
	b.lastAddedSortOrderID = &sortOrder.orderID
	b.updates = append(b.updates, NewAddSortOrderUpdate(sortOrder))

	return nil
}

func (b *MetadataBuilder) RemoveProperties(keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	for _, reserved := range ReservedProperties {
		if slices.Contains(keys, reserved) {
			return fmt.Errorf("can't remove reserved property %s", reserved)
		}
	}
	b.updates = append(b.updates, NewRemovePropertiesUpdate(keys))
	for _, key := range keys {
		delete(b.props, key)
	}

	return nil
}

func (b *MetadataBuilder) SetCurrentSchemaID(currentSchemaID int) error {
	if currentSchemaID == -1 {
		if b.lastAddedSchemaID == nil {
			return fmt.Errorf("%w: can't set current schema to last added schema, no schema has been added", iceberg.ErrInvalidArgument)
		}
		currentSchemaID = *b.lastAddedSchemaID
	}

	if currentSchemaID == b.currentSchemaID {
		return nil
	}

	_, err := b.GetSchemaByID(currentSchemaID)
	if err != nil {
		return fmt.Errorf("%w: can't set current schema to schema with id %d: %w", iceberg.ErrInvalidArgument, currentSchemaID, err)
	}

	if b.lastAddedSchemaID != nil && currentSchemaID == *b.lastAddedSchemaID {
		b.updates = append(b.updates, NewSetCurrentSchemaUpdate(-1))
	} else {
		b.updates = append(b.updates, NewSetCurrentSchemaUpdate(currentSchemaID))
	}
	b.currentSchemaID = currentSchemaID

	return nil
}

func (b *MetadataBuilder) SetDefaultSortOrderID(defaultSortOrderID int) error {
	if defaultSortOrderID == -1 {
		if b.lastAddedSortOrderID == nil {
			return fmt.Errorf("%w: can't set default sort order to last added with no added sort orders", iceberg.ErrInvalidArgument)
		}
		defaultSortOrderID = *b.lastAddedSortOrderID
	}

	if defaultSortOrderID == b.defaultSortOrderID {
		return nil
	}

	if _, err := b.GetSortOrderByID(defaultSortOrderID); err != nil {
		return fmt.Errorf("%w: can't set default sort order to sort order with id %d: %w", iceberg.ErrInvalidArgument, defaultSortOrderID, err)
	}

	// Encode -1 ("last added") only when this builder actually emitted
	// an add-sort-order update for that ID; when the last AddSortOrder
	// reused an existing order no add update precedes this one, so the
	// payload must carry the concrete ID to stay self-contained.
	if b.lastAddedSortOrderID != nil && defaultSortOrderID == *b.lastAddedSortOrderID && b.hasAddSortOrderUpdate(defaultSortOrderID) {
		b.updates = append(b.updates, NewSetDefaultSortOrderUpdate(-1))
	} else {
		b.updates = append(b.updates, NewSetDefaultSortOrderUpdate(defaultSortOrderID))
	}

	b.defaultSortOrderID = defaultSortOrderID

	return nil
}

func (b *MetadataBuilder) SetDefaultSpecID(defaultSpecID int) error {
	if defaultSpecID == -1 {
		if b.lastAddedPartitionID != nil {
			defaultSpecID = *b.lastAddedPartitionID
		} else {
			return fmt.Errorf("%w: can't set default spec to last added with no added partition specs", iceberg.ErrInvalidArgument)
		}
	}

	if defaultSpecID == b.defaultSpecID {
		return nil
	}

	if _, err := b.GetSpecByID(defaultSpecID); err != nil {
		return fmt.Errorf("%w: can't set default spec to spec with id %d: %w", iceberg.ErrInvalidArgument, defaultSpecID, err)
	}

	if b.lastAddedPartitionID != nil && defaultSpecID == *b.lastAddedPartitionID {
		b.updates = append(b.updates, NewSetDefaultSpecUpdate(-1))
	} else {
		b.updates = append(b.updates, NewSetDefaultSpecUpdate(defaultSpecID))
	}
	b.defaultSpecID = defaultSpecID

	return nil
}

func (b *MetadataBuilder) SetFormatVersion(formatVersion int) error {
	if formatVersion < b.formatVersion {
		return fmt.Errorf("%w: downgrading format version from %d to %d is not allowed",
			iceberg.ErrInvalidArgument, b.formatVersion, formatVersion)
	}

	if formatVersion > supportedTableFormatVersion {
		return fmt.Errorf("%w: %d", iceberg.ErrInvalidFormatVersion, formatVersion)
	}

	if formatVersion == b.formatVersion {
		return nil
	}

	if err := b.validateLastColumnID(); err != nil {
		return err
	}

	previousVersion := b.formatVersion
	b.updates = append(b.updates, NewUpgradeFormatVersionUpdate(formatVersion))
	b.formatVersion = formatVersion

	if previousVersion < 2 && formatVersion >= 2 {
		if b.uuid == (uuid.UUID{}) {
			b.uuid = uuid.New()
		}

		if b.lastSequenceNumber == nil {
			seq := int64(0)
			b.lastSequenceNumber = &seq
		}
	}

	if previousVersion < 3 && formatVersion >= 3 {
		if b.nextRowID == nil {
			nextRowID := int64(0)
			b.nextRowID = &nextRowID
		}
	}

	return nil
}

func (b *MetadataBuilder) validateLastColumnID() error {
	highestFieldID := 0
	for _, schema := range b.schemaList {
		if id := schema.HighestFieldID(); id > highestFieldID {
			highestFieldID = id
		}
	}

	if highestFieldID > 0 && b.lastColumnId < highestFieldID {
		return fmt.Errorf("%w: last-column-id %d is less than the highest field ID %d in schemas",
			ErrInvalidMetadata, b.lastColumnId, highestFieldID)
	}

	return nil
}

func (b *MetadataBuilder) SetLoc(loc string) error {
	if b.loc == loc {
		return nil
	}

	b.updates = append(b.updates, NewSetLocationUpdate(loc))
	b.loc = loc

	return nil
}

func (b *MetadataBuilder) SetProperties(props iceberg.Properties) error {
	if len(props) == 0 {
		return nil
	}

	for _, key := range ReservedProperties {
		if _, ok := props[key]; ok {
			return fmt.Errorf("%w: can't set reserved property %s", iceberg.ErrInvalidArgument, key)
		}
	}

	updates := maps.Clone(props)
	b.updates = append(b.updates, NewSetPropertiesUpdate(updates))
	if b.props == nil {
		b.props = maps.Clone(updates)
	} else {
		maps.Copy(b.props, updates)
	}

	return nil
}

type setSnapshotRefOption func(*SnapshotRef) error

func WithMaxRefAgeMs(maxRefAgeMs int64) setSnapshotRefOption {
	return func(ref *SnapshotRef) error {
		if maxRefAgeMs <= 0 {
			return fmt.Errorf("%w: maxRefAgeMs %d, must be > 0", iceberg.ErrInvalidArgument, maxRefAgeMs)
		}
		ref.MaxRefAgeMs = &maxRefAgeMs

		return nil
	}
}

func WithMaxSnapshotAgeMs(maxSnapshotAgeMs int64) setSnapshotRefOption {
	return func(ref *SnapshotRef) error {
		if maxSnapshotAgeMs <= 0 {
			return fmt.Errorf("%w: maxSnapshotAgeMs %d, must be > 0", iceberg.ErrInvalidArgument, maxSnapshotAgeMs)
		}
		ref.MaxSnapshotAgeMs = &maxSnapshotAgeMs

		return nil
	}
}

func WithMinSnapshotsToKeep(minSnapshotsToKeep int) setSnapshotRefOption {
	return func(ref *SnapshotRef) error {
		if minSnapshotsToKeep <= 0 {
			return fmt.Errorf("%w: minSnapshotsToKeep %d, must be > 0", iceberg.ErrInvalidArgument, minSnapshotsToKeep)
		}
		ref.MinSnapshotsToKeep = &minSnapshotsToKeep

		return nil
	}
}

func (b *MetadataBuilder) SetSnapshotRef(
	name string,
	snapshotID int64,
	refType RefType,
	options ...setSnapshotRefOption,
) error {
	ref := SnapshotRef{
		SnapshotID:      snapshotID,
		SnapshotRefType: refType,
	}
	for _, opt := range options {
		if err := opt(&ref); err != nil {
			return fmt.Errorf("%w: invalid snapshot ref option: %w", iceberg.ErrInvalidArgument, err)
		}
	}
	if err := ref.validate(); err != nil {
		return fmt.Errorf("%w: invalid snapshot ref: %w", iceberg.ErrInvalidArgument, err)
	}

	var maxRefAgeMs, maxSnapshotAgeMs int64
	var minSnapshotsToKeep int
	if ref.MaxRefAgeMs != nil {
		maxRefAgeMs = *ref.MaxRefAgeMs
	}
	if ref.MaxSnapshotAgeMs != nil {
		maxSnapshotAgeMs = *ref.MaxSnapshotAgeMs
	}
	if ref.MinSnapshotsToKeep != nil {
		minSnapshotsToKeep = *ref.MinSnapshotsToKeep
	}

	if existingRef, ok := b.refs[name]; ok && existingRef.Equals(ref) {
		return nil
	}

	snapshot, err := b.SnapshotByID(snapshotID)
	if err != nil {
		return fmt.Errorf("%w: can't set snapshot ref %s to unknown snapshot %d: %w", iceberg.ErrInvalidArgument, name, snapshotID, err)
	}

	isAddedSnapshot := slices.ContainsFunc(b.updates, func(u Update) bool {
		return u.Action() == UpdateAddSnapshot && u.(*addSnapshotUpdate).Snapshot.SnapshotID == snapshotID
	})
	if isAddedSnapshot {
		b.lastUpdatedMS = snapshot.TimestampMs
	}

	b.updates = append(b.updates, NewSetSnapshotRefUpdate(name, snapshotID, refType, maxRefAgeMs, maxSnapshotAgeMs, minSnapshotsToKeep))
	if name == MainBranch {
		b.currentSnapshotID = &snapshotID
		if !isAddedSnapshot {
			b.lastUpdatedMS = time.Now().Local().UnixMilli()
		}
		b.snapshotLog = append(b.snapshotLog, SnapshotLogEntry{
			SnapshotID:  snapshotID,
			TimestampMs: b.lastUpdatedMS,
		})
	}

	if slices.ContainsFunc(b.updates, func(u Update) bool {
		return u.Action() == UpdateAddSnapshot && u.(*addSnapshotUpdate).Snapshot.SnapshotID == snapshotID
	}) {
		b.lastUpdatedMS = snapshot.TimestampMs
	}

	b.refs[name] = ref

	return nil
}

func (b *MetadataBuilder) NewRetainingSnapshotRefUpdate(name string, snapshotID int64, refType RefType) *setSnapshotRefUpdate {
	var maxRefAgeMs, maxSnapshotAgeMs int64
	var minSnapshotsToKeep int
	if existing, ok := b.refs[name]; ok && existing.SnapshotRefType == refType {
		if existing.MaxRefAgeMs != nil {
			maxRefAgeMs = *existing.MaxRefAgeMs
		}
		if existing.MaxSnapshotAgeMs != nil {
			maxSnapshotAgeMs = *existing.MaxSnapshotAgeMs
		}
		if existing.MinSnapshotsToKeep != nil {
			minSnapshotsToKeep = *existing.MinSnapshotsToKeep
		}
	}

	return NewSetSnapshotRefUpdate(name, snapshotID, refType, maxRefAgeMs, maxSnapshotAgeMs, minSnapshotsToKeep)
}

func (b *MetadataBuilder) RemoveSnapshotRef(name string) error {
	if _, found := b.refs[name]; !found {
		return fmt.Errorf("%w: snapshot ref not found: %s", iceberg.ErrInvalidArgument, name)
	}

	if name == MainBranch {
		b.currentSnapshotID = nil
	}

	delete(b.refs, name)
	b.updates = append(b.updates, NewRemoveSnapshotRefUpdate(name))

	return nil
}

func (b *MetadataBuilder) SetUUID(newUUID uuid.UUID) error {
	if newUUID == uuid.Nil {
		return fmt.Errorf("%w: cannot set uuid to nil", iceberg.ErrInvalidArgument)
	}

	if b.uuid == newUUID {
		return nil
	}

	b.updates = append(b.updates, NewAssignUUIDUpdate(newUUID))
	b.uuid = newUUID

	return nil
}

func (b *MetadataBuilder) SetLastUpdatedMS() *MetadataBuilder {
	b.lastUpdatedMS = time.Now().UnixMilli()

	return b
}

func (b *MetadataBuilder) buildCommonMetadata() (*commonMetadata, error) {
	b.ensureSchemaIndex()
	if b.schemaIndex != nil {
		b.schemaIndex.shared = true
	}
	b.ensurePartitionSpecIndex()
	if b.partitionSpecIndex != nil {
		b.partitionSpecIndex.shared = true
	}
	b.ensureSnapshotIndex()
	if b.snapshotIndex != nil {
		b.snapshotIndex.shared = true
	}

	if _, err := b.GetSpecByID(b.defaultSpecID); err != nil {
		return nil, fmt.Errorf("%w: defaultSpecID is invalid: %w", ErrInvalidMetadata, err)
	}
	defaultSpecID := b.defaultSpecID

	if err := b.updateSnapshotLog(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidMetadata, err)
	}

	if b.lastUpdatedMS == 0 {
		b.lastUpdatedMS = time.Now().UnixMilli()
	}

	if b.previousFileEntry != nil && b.HasChanges() {
		maxMetadataLogEntries := max(1,
			b.props.GetInt(
				MetadataPreviousVersionsMaxKey, MetadataPreviousVersionsMaxDefault))
		b.AppendMetadataLog(*b.previousFileEntry)
		b.TrimMetadataLogs(maxMetadataLogEntries)
	}

	return &commonMetadata{
		FormatVersion:      b.formatVersion,
		UUID:               b.uuid,
		Loc:                b.loc,
		LastUpdatedMS:      b.lastUpdatedMS,
		LastColumnId:       b.lastColumnId,
		SchemaList:         b.schemaList,
		schemaIndex:        b.schemaIndex,
		CurrentSchemaID:    b.currentSchemaID,
		Specs:              b.specs,
		partitionSpecIndex: b.partitionSpecIndex,
		DefaultSpecID:      defaultSpecID,
		LastPartitionID:    b.lastPartitionID,
		Props:              b.props,
		SnapshotList:       b.snapshotList,
		snapshotIndex:      b.snapshotIndex,
		CurrentSnapshotID:  b.currentSnapshotID,
		SnapshotLog:        b.snapshotLog,
		MetadataLog:        b.metadataLog,
		SortOrderList:      b.sortOrderList,
		DefaultSortOrderID: b.defaultSortOrderID,
		SnapshotRefs:       b.refs,
		StatisticsList:     b.statisticsList,
		PartitionStatsList: b.partitionStatsList,
		EncryptionKeyList:  b.encryptionKeyList,
	}, nil
}

func (b *MetadataBuilder) updateSnapshotLog() error {
	addedIDs := make(map[int64]struct{}, 2)
	hasRemoved := false
	for _, upd := range b.updates {
		if up, ok := upd.(*addSnapshotUpdate); ok {
			addedIDs[up.Snapshot.SnapshotID] = struct{}{}
		}
		if _, ok := upd.(*removeSnapshotsUpdate); ok {
			hasRemoved = true
		}
	}
	intermediateIDs := make(map[int64]struct{}, 2)
	for _, upd := range b.updates {
		if s, ok := upd.(*setSnapshotRefUpdate); ok {
			if _, ok := addedIDs[s.SnapshotID]; ok && s.RefName == MainBranch && ((b.currentSnapshotID != nil && s.SnapshotID != *b.currentSnapshotID) || (s.SnapshotID == -1)) {
				intermediateIDs[s.SnapshotID] = struct{}{}
			}
		}
	}
	if len(intermediateIDs) != 0 || hasRemoved {
		validSnapshotIDs := make(map[int64]struct{}, len(b.snapshotList))
		for _, snapshot := range b.snapshotList {
			validSnapshotIDs[snapshot.SnapshotID] = struct{}{}
		}

		newSnapsLog := make([]SnapshotLogEntry, 0, len(b.snapshotLog))
		for _, s := range b.snapshotLog {
			if _, ok := validSnapshotIDs[s.SnapshotID]; ok {
				if _, ok := intermediateIDs[s.SnapshotID]; !ok {
					newSnapsLog = append(newSnapsLog, s)
				}
			} else if hasRemoved {
				newSnapsLog = newSnapsLog[:0]
			}
		}
		if b.currentSnapshotID != nil && len(newSnapsLog) != 0 {
			last := newSnapsLog[len(newSnapsLog)-1]
			if last.SnapshotID != *b.currentSnapshotID {
				return fmt.Errorf("%w: cannot set invalid snapshot log: latest entry is not the current snapshot", iceberg.ErrInvalidArgument)
			}
		}
		b.snapshotLog = newSnapsLog
	}

	return nil
}

func (b *MetadataBuilder) GetSchemaByID(id int) (*iceberg.Schema, error) {
	index := b.schemaIndex
	if schemaIndexNeedsRebuild(index, b.schemaList) {
		index = buildSchemaIndex(b.schemaList)
	}
	if schema, ok := schemaIndexLookup(index, b.schemaList, id); ok {
		return schema, nil
	}

	return nil, fmt.Errorf("%w: schema with id %d not found", iceberg.ErrInvalidArgument, id)
}

func (b *MetadataBuilder) GetSpecByID(id int) (*iceberg.PartitionSpec, error) {
	b.ensurePartitionSpecIndex()

	i, ok := partitionSpecIndexPosition(b.partitionSpecIndex, b.specs, id)
	if ok {
		spec := b.specs[i]

		return &spec, nil
	}

	return nil, fmt.Errorf("%w: id %d", ErrPartitionSpecNotFound, id)
}

func (b *MetadataBuilder) GetSortOrderByID(id int) (*SortOrder, error) {
	for _, s := range b.sortOrderList {
		if s.OrderID() == id {
			return &s, nil
		}
	}

	return nil, fmt.Errorf("sort order with id %d not found", id)
}

func (b *MetadataBuilder) SnapshotByID(id int64) (*Snapshot, error) {
	index := b.snapshotIndex
	if snapshotIndexNeedsRebuild(index, b.snapshotList) {
		index = buildSnapshotIndex(b.snapshotList)
	}

	i, ok := snapshotIndexPosition(index, b.snapshotList, id)
	if ok {
		snapshot := b.snapshotList[i]

		return &snapshot, nil
	}

	return nil, fmt.Errorf("%w: id %d", ErrSnapshotNotFound, id)
}

func (b *MetadataBuilder) NameMapping() iceberg.NameMapping {
	if nameMappingJson, ok := b.props[DefaultNameMappingKey]; ok {
		nm := iceberg.NameMapping{}
		if err := json.Unmarshal([]byte(nameMappingJson), &nm); err == nil {
			return nm
		}
	}

	return nil
}

func (b *MetadataBuilder) TrimMetadataLogs(maxEntries int) *MetadataBuilder {
	if len(b.metadataLog) <= maxEntries {
		return b
	}

	b.metadataLog = b.metadataLog[len(b.metadataLog)-maxEntries:]

	return b
}

func (b *MetadataBuilder) AppendMetadataLog(entry MetadataLogEntry) *MetadataBuilder {
	b.metadataLog = append(b.metadataLog, entry)

	return b
}

func (b *MetadataBuilder) Build() (Metadata, error) {
	common, err := b.buildCommonMetadata()
	if err != nil {
		return nil, err
	}

	if err := common.validate(); err != nil {
		return nil, err
	}

	switch b.formatVersion {
	case 1:
		schema, err := b.GetSchemaByID(b.currentSchemaID)
		if err != nil {
			return nil, fmt.Errorf("%w: can't build metadata, missing schema for schema ID %d: %w", ErrInvalidMetadata, b.currentSchemaID, err)
		}

		partition, err := b.GetSpecByID(common.DefaultSpecID)
		if err != nil {
			return nil, fmt.Errorf("%w: can't build metadata, missing partition spec for spec ID %d: %w", ErrInvalidMetadata, b.defaultSpecID, err)
		}

		partitionFields := make([]iceberg.PartitionField, 0)
		for _, f := range partition.Fields() {
			partitionFields = append(partitionFields, f)
		}

		return &metadataV1{
			Schema:         schema,
			Partition:      partitionFields,
			commonMetadata: *common,
		}, nil

	case 2:
		var lastSequenceNumber int64

		if b.lastSequenceNumber != nil {
			lastSequenceNumber = *b.lastSequenceNumber
		}

		return &metadataV2{
			LastSeqNum:     lastSequenceNumber,
			commonMetadata: *common,
		}, nil

	case 3:
		var lastSequenceNumber int64
		var nextRowID int64

		if b.lastSequenceNumber != nil {
			lastSequenceNumber = *b.lastSequenceNumber
		}

		if b.nextRowID != nil {
			nextRowID = *b.nextRowID
		}

		return &metadataV3{
			LastSeqNum:     lastSequenceNumber,
			NextRowIDValue: nextRowID,
			commonMetadata: *common,
		}, nil

	default:
		return nil, fmt.Errorf("%w: unknown format version %d", ErrInvalidMetadata, b.formatVersion)
	}
}

// hasAddSortOrderUpdate reports whether this builder has emitted an
// add-sort-order update for the given order ID.
func (b *MetadataBuilder) hasAddSortOrderUpdate(orderID int) bool {
	for _, u := range b.updates {
		if a, ok := u.(*addSortOrderUpdate); ok && a.SortOrder.OrderID() == orderID {
			return true
		}
	}

	return false
}

func (b *MetadataBuilder) reuseOrCreateNewSortOrderID(newOrder *SortOrder) int {
	if newOrder.IsUnsorted() {
		return UnsortedSortOrder.OrderID()
	}

	newOrderID := UnsortedSortOrderID + 1
	for _, order := range b.sortOrderList {
		if slices.EqualFunc(order.fields, newOrder.fields, func(a, b SortField) bool { return a.Equals(b) }) {
			return order.OrderID()
		}
		if order.OrderID() >= newOrderID {
			newOrderID = order.OrderID() + 1
		}
	}

	return newOrderID
}

func (b *MetadataBuilder) reuseOrCreateNewPartitionSpecID(newSpec iceberg.PartitionSpec) int {
	newSpecID := 0
	for _, spec := range b.specs {
		if spec.Equals(newSpec) {
			return spec.ID()
		}
		if spec.ID() >= newSpecID {
			newSpecID = spec.ID() + 1
		}
	}

	return newSpecID
}

func (b *MetadataBuilder) reuseOrCreateNewSchemaID(newSchema *iceberg.Schema) int {
	newSchemaID := newSchema.ID
	for _, schema := range b.schemaList {
		if newSchema.Equals(schema) {
			return schema.ID
		}
		if schema.ID >= newSchemaID {
			newSchemaID = schema.ID + 1
		}
	}

	return newSchemaID
}

func makeIntContainsFn(ints []int) func(int) bool {
	// Scan small removal lists; reserve the index for bulk requests.
	if len(ints) <= 16 {
		return func(id int) bool { return slices.Contains(ints, id) }
	}

	ids := make(map[int]struct{}, len(ints))
	for _, id := range ints {
		ids[id] = struct{}{}
	}

	return func(id int) bool {
		_, ok := ids[id]

		return ok
	}
}

func (b *MetadataBuilder) RemovePartitionSpecs(ints []int) error {
	if len(ints) == 0 {
		return nil
	}

	containsID := makeIntContainsFn(ints)
	if containsID(b.defaultSpecID) {
		return fmt.Errorf("%w: can't remove default partition spec with id %d", iceberg.ErrInvalidArgument, b.defaultSpecID)
	}

	newSpecs := make([]iceberg.PartitionSpec, 0, len(b.specs))
	removed := make([]int, 0, len(ints))
	for _, spec := range b.specs {
		if containsID(spec.ID()) {
			removed = append(removed, spec.ID())

			continue
		}
		newSpecs = append(newSpecs, spec)
	}

	if len(removed) != 0 {
		b.specs = newSpecs
		b.partitionSpecIndex = buildPartitionSpecIndex(b.specs)
		b.updates = append(b.updates, NewRemoveSpecUpdate(removed))
	}

	return nil
}

func (b *MetadataBuilder) RemoveSchemas(ints []int) error {
	if len(ints) == 0 {
		return nil
	}

	containsID := makeIntContainsFn(ints)
	if containsID(b.currentSchemaID) {
		return fmt.Errorf("%w: can't remove current schema with id %d", iceberg.ErrInvalidArgument, b.currentSchemaID)
	}

	removed := make([]int, 0, len(ints))
	b.schemaList = slices.DeleteFunc(b.schemaList, func(s *iceberg.Schema) bool {
		if containsID(s.ID) {
			removed = append(removed, s.ID)

			return true
		}

		return false
	})

	if len(removed) != 0 {
		b.schemaIndex = buildSchemaIndex(b.schemaList)
		b.updates = append(b.updates, NewRemoveSchemasUpdate(removed))
	}

	return nil
}

// SetStatistics adds or replaces a statistics file for the given snapshot. If a statistics
// file with the same snapshot ID already exists it is replaced, otherwise the file is appended.
func (b *MetadataBuilder) SetStatistics(stats StatisticsFile) error {
	replaced := false
	for i, s := range b.statisticsList {
		if s.SnapshotID == stats.SnapshotID {
			b.statisticsList[i] = stats
			replaced = true

			break
		}
	}

	if !replaced {
		b.statisticsList = append(b.statisticsList, stats)
	}

	b.updates = append(b.updates, NewSetStatisticsUpdate(stats))

	return nil
}

// RemoveStatistics removes the statistics file associated with the given snapshot ID.
// It is not an error if no such file exists.
func (b *MetadataBuilder) RemoveStatistics(snapshotID int64) error {
	b.statisticsList = slices.DeleteFunc(b.statisticsList, func(s StatisticsFile) bool {
		return s.SnapshotID == snapshotID
	})
	b.updates = append(b.updates, NewRemoveStatisticsUpdate(snapshotID))

	return nil
}

// SetPartitionStatistics adds or replaces a partition statistics file for the given snapshot.
// If a partition statistics file with the same snapshot ID already exists it is replaced,
// otherwise the file is appended.
func (b *MetadataBuilder) SetPartitionStatistics(stats PartitionStatisticsFile) error {
	replaced := false
	for i, s := range b.partitionStatsList {
		if s.SnapshotID == stats.SnapshotID {
			b.partitionStatsList[i] = stats
			replaced = true

			break
		}
	}

	if !replaced {
		b.partitionStatsList = append(b.partitionStatsList, stats)
	}

	b.updates = append(b.updates, NewSetPartitionStatisticsUpdate(stats))

	return nil
}

// RemovePartitionStatistics removes the partition statistics file associated with the given
// snapshot ID. It is not an error if no such file exists.
func (b *MetadataBuilder) RemovePartitionStatistics(snapshotID int64) error {
	b.partitionStatsList = slices.DeleteFunc(b.partitionStatsList, func(s PartitionStatisticsFile) bool {
		return s.SnapshotID == snapshotID
	})
	b.updates = append(b.updates, NewRemovePartitionStatisticsUpdate(snapshotID))

	return nil
}

// AddEncryptionKey adds or replaces an encryption key indexed by its key-id.
// Encryption keys are only supported for format version 3 and above.
func (b *MetadataBuilder) AddEncryptionKey(key EncryptionKey) error {
	if b.formatVersion < 3 {
		return fmt.Errorf("%w: encryption keys are only supported for format version 3 or higher, current version: %d",
			iceberg.ErrInvalidArgument, b.formatVersion)
	}
	if err := key.validate(); err != nil {
		return fmt.Errorf("%w: %v", iceberg.ErrInvalidArgument, err)
	}

	replaced := false
	for i, k := range b.encryptionKeyList {
		if k.KeyID == key.KeyID {
			b.encryptionKeyList[i] = key
			replaced = true

			break
		}
	}

	if !replaced {
		b.encryptionKeyList = append(b.encryptionKeyList, key)
	}

	b.updates = append(b.updates, NewAddEncryptionKeyUpdate(key))

	return nil
}

// RemoveEncryptionKey removes the encryption key with the given key-id.
// It is not an error if no such key exists.
func (b *MetadataBuilder) RemoveEncryptionKey(keyID string) error {
	b.encryptionKeyList = slices.DeleteFunc(b.encryptionKeyList, func(k EncryptionKey) bool {
		return k.KeyID == keyID
	})
	b.updates = append(b.updates, NewRemoveEncryptionKeyUpdate(keyID))

	return nil
}

// maxBy returns the maximum value of extract(e) for all e in elems.
// If elems is empty, returns 0.
func maxBy[S ~[]E, E any](elems S, extract func(e E) int) int {
	m := 0
	for _, e := range elems {
		m = max(m, extract(e))
	}

	return m
}

var (
	ErrInvalidMetadataFormatVersion = errors.New("invalid or missing format-version in table metadata")
	ErrInvalidMetadata              = errors.New("invalid metadata")
	ErrPartitionSpecNotFound        = errors.New("partition spec not found")
)

type versionScopedField struct {
	name       string
	introduced int
	present    bool
}

func rejectFieldsBeyondVersion(currentVersion int, fields ...versionScopedField) error {
	var errs []error
	for _, f := range fields {
		if f.present && currentVersion < f.introduced {
			errs = append(errs, fmt.Errorf("%w: v%d-only field '%s' present in v%d metadata",
				ErrInvalidMetadataFormatVersion, f.introduced, f.name, currentVersion))
		}
	}

	return errors.Join(errs...)
}

// ParseMetadata parses json metadata provided by the passed in reader,
// returning an error if one is encountered.
func ParseMetadata(r io.Reader) (Metadata, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return ParseMetadataBytes(data)
}

// ParseMetadataString is like [ParseMetadata], but for a string rather than
// an io.Reader.
func ParseMetadataString(s string) (Metadata, error) {
	return ParseMetadataBytes([]byte(s))
}

// ParseMetadataBytes is like [ParseMetadataString] but for a byte slice.
func ParseMetadataBytes(b []byte) (Metadata, error) {
	metadata, formatVersion, err := preflightMetadataBytes(b)
	if err != nil {
		return nil, err
	}

	var ret Metadata
	switch formatVersion {
	case 1:
		ret = &metadataV1{}
	case 2:
		ret = &metadataV2{}
	case 3:
		ret = &metadataV3{}
	default:
		return nil, ErrInvalidMetadataFormatVersion
	}

	normalized, err := assignMissingPartitionFieldIDsFromMetadata(b, metadata)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(normalized, ret); err != nil {
		if errors.Is(err, ErrInvalidMetadata) {
			return nil, err
		}

		return nil, fmt.Errorf("%w: %w", ErrInvalidMetadata, err)
	}
	if ret.Version() != formatVersion {
		return nil, fmt.Errorf("%w: preflight selected version %d, decoded version %d",
			ErrInvalidMetadataFormatVersion, formatVersion, ret.Version())
	}

	return ret, nil
}

// ParseMetadataBytesDeferredSnapshots parses metadata while retaining the full
// snapshot array as raw JSON. Snapshots targeted by refs are decoded eagerly;
// unreferenced history is materialized on first access. Operations requiring
// the complete snapshot collection, including MetadataBuilderFromBase, decode
// the full history and therefore may cost more than eager parsing overall.
func ParseMetadataBytesDeferredSnapshots(b []byte) (Metadata, error) {
	metadata, formatVersion, err := preflightMetadataBytes(b)
	if err != nil {
		return nil, err
	}

	normalized, err := assignMissingPartitionFieldIDsFromMetadata(b, metadata)
	if err != nil {
		return nil, err
	}

	return parseNormalizedMetadataBytesDeferredSnapshots(normalized, formatVersion)
}

// preflightMetadataBytes keeps the raw top-level object for normalization and
// format selection so eager and deferred parsing share one document scan.
func preflightMetadataBytes(b []byte) (map[string]json.RawMessage, int, error) {
	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(b, &metadata); err != nil {
		return nil, 0, fmt.Errorf("%w: %w", ErrInvalidMetadata, err)
	}

	var formatVersion int
	if rawVersion, ok := metadata["format-version"]; ok {
		if err := json.Unmarshal(rawVersion, &formatVersion); err != nil {
			return nil, 0, fmt.Errorf("%w: %w", ErrInvalidMetadata, err)
		}
	}

	return metadata, formatVersion, nil
}

func requirePartitionSpecIDs(b []byte) error {
	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(b, &metadata); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidMetadata, err)
	}

	return requirePartitionSpecIDsFromMetadata(metadata)
}

func requirePartitionSpecIDsFromMetadata(metadata map[string]json.RawMessage) error {
	rawSpecs, ok := metadata["partition-specs"]
	if !ok {
		return nil
	}

	var specs []rawPartitionSpec
	if err := json.Unmarshal(rawSpecs, &specs); err != nil {
		return fmt.Errorf("%w: invalid partition-specs: %w", ErrInvalidMetadata, err)
	}

	return validatePartitionSpecIDs(specs)
}

type rawPartitionSpec struct {
	ID     *int                         `json:"spec-id"`
	Fields []map[string]json.RawMessage `json:"fields"`
}

func validatePartitionSpecIDs(specs []rawPartitionSpec) error {
	for i, spec := range specs {
		if spec.ID == nil {
			return fmt.Errorf("%w: partition spec at index %d is missing required spec-id", ErrInvalidMetadata, i)
		}
	}

	return nil
}

func assignMissingPartitionFieldIDs(b []byte) ([]byte, error) {
	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(b, &metadata); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidMetadata, err)
	}

	return assignMissingPartitionFieldIDsFromMetadata(b, metadata)
}

func assignMissingPartitionFieldIDsFromMetadata(b []byte, metadata map[string]json.RawMessage) ([]byte, error) {
	if err := requireLastUpdatedMS(metadata); err != nil {
		return nil, err
	}

	var specs []rawPartitionSpec
	usesSpecList := false
	if rawSpecs, ok := metadata["partition-specs"]; ok {
		if err := json.Unmarshal(rawSpecs, &specs); err != nil {
			return nil, err
		}
		usesSpecList = true
		if err := validatePartitionSpecIDs(specs); err != nil {
			return nil, err
		}
	} else if rawFields, ok := metadata["partition-spec"]; ok {
		var fields []map[string]json.RawMessage
		if err := json.Unmarshal(rawFields, &fields); err != nil {
			return nil, err
		}
		specs = []rawPartitionSpec{{Fields: fields}}
	} else {
		return b, nil
	}

	lastAssignedID := iceberg.PartitionDataIDStart - 1
	lastPartitionIDSet := false
	if rawLastPartitionID, ok := metadata["last-partition-id"]; ok {
		var lastPartitionID *int
		if err := json.Unmarshal(rawLastPartitionID, &lastPartitionID); err == nil && lastPartitionID != nil {
			lastAssignedID = max(lastAssignedID, *lastPartitionID)
			lastPartitionIDSet = true
		}
	}

	missingFields := make([]map[string]json.RawMessage, 0)
	for _, spec := range specs {
		for _, field := range spec.Fields {
			rawFieldID, ok := field["field-id"]
			if !ok {
				missingFields = append(missingFields, field)

				continue
			}

			var fieldID *int
			if err := json.Unmarshal(rawFieldID, &fieldID); err == nil && fieldID != nil {
				lastAssignedID = max(lastAssignedID, *fieldID)
			}
		}
	}

	if len(missingFields) == 0 {
		return b, nil
	}

	for _, field := range missingFields {
		lastAssignedID++
		rawFieldID, err := json.Marshal(lastAssignedID)
		if err != nil {
			return nil, err
		}
		field["field-id"] = rawFieldID
	}

	if usesSpecList {
		rawSpecs, err := json.Marshal(specs)
		if err != nil {
			return nil, err
		}
		metadata["partition-specs"] = rawSpecs
	} else {
		rawFields, err := json.Marshal(specs[0].Fields)
		if err != nil {
			return nil, err
		}
		metadata["partition-spec"] = rawFields
	}

	if lastPartitionIDSet {
		rawLastPartitionID, err := json.Marshal(lastAssignedID)
		if err != nil {
			return nil, err
		}
		metadata["last-partition-id"] = rawLastPartitionID
	}

	return json.Marshal(metadata)
}

// https://iceberg.apache.org/spec/#iceberg-table-spec
type commonMetadata struct {
	FormatVersion      int                       `json:"format-version"`
	UUID               uuid.UUID                 `json:"table-uuid"`
	Loc                string                    `json:"location"`
	LastUpdatedMS      int64                     `json:"last-updated-ms"`
	LastColumnId       int                       `json:"last-column-id"`
	SchemaList         []*iceberg.Schema         `json:"schemas"`
	CurrentSchemaID    int                       `json:"current-schema-id"`
	Specs              []iceberg.PartitionSpec   `json:"partition-specs"`
	DefaultSpecID      int                       `json:"default-spec-id"`
	LastPartitionID    *int                      `json:"last-partition-id,omitempty"`
	Props              iceberg.Properties        `json:"properties,omitempty"`
	SnapshotList       []Snapshot                `json:"snapshots,omitempty"`
	CurrentSnapshotID  *int64                    `json:"current-snapshot-id,omitempty"`
	SnapshotLog        []SnapshotLogEntry        `json:"snapshot-log,omitempty"`
	MetadataLog        []MetadataLogEntry        `json:"metadata-log,omitempty"`
	SortOrderList      []SortOrder               `json:"sort-orders"`
	DefaultSortOrderID int                       `json:"default-sort-order-id"`
	SnapshotRefs       map[string]SnapshotRef    `json:"refs,omitempty"`
	StatisticsList     []StatisticsFile          `json:"statistics,omitempty"`
	PartitionStatsList []PartitionStatisticsFile `json:"partition-statistics,omitempty"`
	EncryptionKeyList  []EncryptionKey           `json:"encryption-keys,omitempty"`
	// V2+ fields
	LastSequenceNumber *int64 `json:"last-sequence-number,omitempty"`
	// V3+ fields
	NextRowID *int64 `json:"next-row-id,omitempty"` // V3: Next available row ID

	schemaIndex        *schemaIndexData
	partitionSpecIndex *partitionSpecIndexData
	snapshotIndex      *snapshotIndexData
	deferredSnapshots  *deferredSnapshotState
}

func (c *commonMetadata) metadataBuilderCommon() *commonMetadata { return c }

func initCommonMetadataForDeserialization() commonMetadata {
	return commonMetadata{
		// These fields use negative sentinels so validation can distinguish omitted
		// values from explicit zero values.
		LastColumnId:       -1,
		CurrentSchemaID:    -1,
		DefaultSpecID:      -1,
		DefaultSortOrderID: -1,
		SortOrderList:      nil,
		Specs:              nil,
	}
}

func requireLastUpdatedMS(fields map[string]json.RawMessage) error {
	value, ok := fields["last-updated-ms"]
	if !ok || string(value) == "null" {
		return fmt.Errorf("%w: last-updated-ms is absent or null", ErrInvalidMetadata)
	}

	return nil
}

// checkRequiredFields validates fields required when reading persisted table
// metadata. Keep this at the JSON boundary: catalog create-via-commit paths use
// builders with an empty staging location. Location becomes optional in format
// v4, so the location check must be version-gated when v4 support is added.
func (c *commonMetadata) checkRequiredFields() error {
	if c.UUID == uuid.Nil && c.FormatVersion > 1 {
		return fmt.Errorf("%w: table-uuid is required for format version %d", ErrInvalidMetadata, c.FormatVersion)
	}
	if c.Loc == "" {
		return fmt.Errorf("%w: location is required", ErrInvalidMetadata)
	}

	return nil
}

func (c *commonMetadata) Ref() SnapshotRef {
	return cloneSnapshotRef(c.SnapshotRefs[MainBranch])
}

func (c *commonMetadata) Refs() iter.Seq2[string, SnapshotRef] {
	return func(yield func(string, SnapshotRef) bool) {
		for name, ref := range c.SnapshotRefs {
			if !yield(name, cloneSnapshotRef(ref)) {
				return
			}
		}
	}
}

func (c *commonMetadata) SnapshotLogs() iter.Seq[SnapshotLogEntry] {
	return slices.Values(c.SnapshotLog)
}

func (c *commonMetadata) PreviousFiles() iter.Seq[MetadataLogEntry] {
	return slices.Values(c.MetadataLog)
}

func stringPointerEqual(left, right *string) bool {
	if left == right {
		return true
	}
	if left == nil || right == nil {
		return false
	}

	return *left == *right
}

func blobMetadataEqual(left, right BlobMetadata) bool {
	return left.Type == right.Type &&
		left.SnapshotID == right.SnapshotID &&
		left.SequenceNumber == right.SequenceNumber &&
		slices.Equal(left.Fields, right.Fields) &&
		maps.Equal(left.Properties, right.Properties)
}

func statisticsFileEqual(left, right StatisticsFile) bool {
	return left.SnapshotID == right.SnapshotID &&
		left.StatisticsPath == right.StatisticsPath &&
		left.FileSizeInBytes == right.FileSizeInBytes &&
		left.FileFooterSizeInBytes == right.FileFooterSizeInBytes &&
		stringPointerEqual(left.KeyMetadata, right.KeyMetadata) &&
		slices.EqualFunc(left.BlobMetadata, right.BlobMetadata, blobMetadataEqual)
}

func partitionStatisticsFileEqual(left, right PartitionStatisticsFile) bool {
	return left.SnapshotID == right.SnapshotID &&
		left.StatisticsPath == right.StatisticsPath &&
		left.FileSizeInBytes == right.FileSizeInBytes
}

func (c *commonMetadata) Equals(other *commonMetadata) bool {
	if other == nil {
		return false
	}

	if c == other {
		return true
	}

	switch {
	case c.LastPartitionID == nil && other.LastPartitionID != nil:
		fallthrough
	case c.LastPartitionID != nil && other.LastPartitionID == nil:
		fallthrough
	case c.CurrentSnapshotID == nil && other.CurrentSnapshotID != nil:
		fallthrough
	case c.CurrentSnapshotID != nil && other.CurrentSnapshotID == nil:
		return false
	}

	switch {
	case !iceinternal.SliceEqualHelper(c.SchemaList, other.SchemaList):
		fallthrough
	case !iceinternal.SliceEqualHelper(c.allSnapshots(), other.allSnapshots()):
		fallthrough
	case !iceinternal.SliceEqualHelper(c.Specs, other.Specs):
		fallthrough
	case !slices.EqualFunc(c.StatisticsList, other.StatisticsList, statisticsFileEqual):
		fallthrough
	case !slices.EqualFunc(c.PartitionStatsList, other.PartitionStatsList, partitionStatisticsFileEqual):
		fallthrough
	case !maps.Equal(c.Props, other.Props):
		fallthrough
	case !maps.EqualFunc(c.SnapshotRefs, other.SnapshotRefs, func(sr1, sr2 SnapshotRef) bool { return sr1.Equals(sr2) }):
		return false
	}

	return c.FormatVersion == other.FormatVersion && c.UUID == other.UUID &&
		((c.LastPartitionID == other.LastPartitionID) || (*c.LastPartitionID == *other.LastPartitionID)) &&
		((c.CurrentSnapshotID == other.CurrentSnapshotID) || (*c.CurrentSnapshotID == *other.CurrentSnapshotID)) &&
		c.Loc == other.Loc && c.LastUpdatedMS == other.LastUpdatedMS &&
		c.LastColumnId == other.LastColumnId && c.CurrentSchemaID == other.CurrentSchemaID &&
		c.DefaultSpecID == other.DefaultSpecID && c.DefaultSortOrderID == other.DefaultSortOrderID &&
		slices.Equal(c.SnapshotLog, other.SnapshotLog) && slices.Equal(c.MetadataLog, other.MetadataLog) &&
		iceinternal.SliceEqualHelper(c.SortOrderList, other.SortOrderList) &&
		iceinternal.SliceEqualHelper(c.EncryptionKeyList, other.EncryptionKeyList)
}

func (c *commonMetadata) TableUUID() uuid.UUID       { return c.UUID }
func (c *commonMetadata) Location() string           { return c.Loc }
func (c *commonMetadata) LastUpdatedMillis() int64   { return c.LastUpdatedMS }
func (c *commonMetadata) LastColumnID() int          { return c.LastColumnId }
func (c *commonMetadata) Schemas() []*iceberg.Schema { return cloneSchemas(c.SchemaList) }
func (c *commonMetadata) CurrentSchema() *iceberg.Schema {
	if schema := c.schemaByID(c.CurrentSchemaID); schema != nil {
		return schema
	}

	panic("should never get here")
}

// schemaByID returns a defensive copy of one schema without cloning the rest
// of the schema history. The method is promoted by the concrete metadata
// versions through commonMetadata and is intentionally not part of Metadata,
// so custom Metadata implementations keep the public Schemas fallback.
func (c *commonMetadata) schemaByID(id int) *iceberg.Schema {
	index := c.schemaIndex
	if schemaIndexNeedsRebuild(index, c.SchemaList) {
		index = buildSchemaIndex(c.SchemaList)
	}
	if schema, ok := schemaIndexLookup(index, c.SchemaList, id); ok {
		return cloneSchema(schema)
	}

	return nil
}

func (c *commonMetadata) PartitionSpecs() []iceberg.PartitionSpec {
	return clonePartitionSpecs(c.Specs)
}

func (c *commonMetadata) DefaultPartitionSpec() int {
	return c.DefaultSpecID
}

func (c *commonMetadata) partitionSpecIndexForLookup() *partitionSpecIndexData {
	index := c.partitionSpecIndex
	if partitionSpecIndexNeedsRebuild(index, c.Specs) {
		// Keep fixture recovery local: validated metadata has a stable index, and
		// read-only lookups must not write to shared metadata state.
		index = buildPartitionSpecIndex(c.Specs)
	}

	return index
}

func (c *commonMetadata) PartitionSpec() iceberg.PartitionSpec {
	index := c.partitionSpecIndexForLookup()

	if i, ok := partitionSpecIndexPosition(index, c.Specs, c.DefaultSpecID); ok {
		return c.Specs[i].Clone()
	}

	return iceberg.UnpartitionedSpec.Clone()
}

func (c *commonMetadata) PartitionSpecByID(id int) *iceberg.PartitionSpec {
	index := c.partitionSpecIndexForLookup()

	if i, ok := partitionSpecIndexPosition(index, c.Specs, id); ok {
		clone := c.Specs[i].Clone()

		return &clone
	}

	return nil
}

func (c *commonMetadata) LastPartitionSpecID() *int {
	if c.LastPartitionID == nil {
		return nil
	}

	id := *c.LastPartitionID

	return &id
}

func (c *commonMetadata) Snapshots() []Snapshot {
	return cloneSnapshots(c.allSnapshots())
}

func (c *commonMetadata) SnapshotByID(id int64) *Snapshot {
	index := c.snapshotIndex
	if snapshotIndexNeedsRebuild(index, c.SnapshotList) {
		index = buildSnapshotIndex(c.SnapshotList)
	}

	i, ok := snapshotIndexPosition(index, c.SnapshotList, id)
	if ok {
		return cloneSnapshotPtr(&c.SnapshotList[i])
	}
	if c.deferredSnapshots != nil {
		snapshot, err := c.deferredSnapshots.snapshotByID(id)
		if err != nil {
			// Parser-created deferred state is synchronously validated before
			// publication, so a later typed decode cannot fail. Keep this branch
			// for defensive handling of manually constructed in-package state.
			return nil
		}

		return snapshot
	}

	return nil
}

func (c *commonMetadata) allSnapshots() []Snapshot {
	snapshots, err := c.snapshotsForMarshal()
	if err != nil {
		// prepareDeferredSnapshots validates JSON syntax and every Snapshot
		// field whose typed decoding can fail before deferred state is
		// published. This error is therefore unreachable for parser-created
		// state; callers with an error channel use snapshotsForMarshal directly.
		return nil
	}

	return snapshots
}

func (c *commonMetadata) snapshotsForMarshal() ([]Snapshot, error) {
	if c.deferredSnapshots == nil {
		return c.SnapshotList, nil
	}

	snapshots, _, err := c.deferredSnapshots.load()

	return snapshots, err
}

func (c *commonMetadata) SnapshotByName(name string) *Snapshot {
	if ref, ok := c.SnapshotRefs[name]; ok {
		return c.SnapshotByID(ref.SnapshotID)
	}

	return nil
}

func (c *commonMetadata) CurrentSnapshot() *Snapshot {
	if c.CurrentSnapshotID == nil {
		return nil
	}

	return c.SnapshotByID(*c.CurrentSnapshotID)
}

func (c *commonMetadata) SortOrders() []SortOrder {
	return cloneSortOrders(c.SortOrderList)
}

func (c *commonMetadata) SortOrder() SortOrder {
	for _, s := range c.SortOrderList {
		if s.OrderID() == c.DefaultSortOrderID {
			return cloneSortOrder(s)
		}
	}

	return cloneSortOrder(UnsortedSortOrder)
}

func (c *commonMetadata) DefaultSortOrder() int {
	return c.DefaultSortOrderID
}

func (c *commonMetadata) Properties() iceberg.Properties {
	if c.Props == nil {
		return iceberg.Properties{}
	}

	return maps.Clone(c.Props)
}

func (c *commonMetadata) Statistics() iter.Seq[StatisticsFile] {
	return slices.Values(cloneStatisticsFiles(c.StatisticsList))
}

func (c *commonMetadata) PartitionStatistics() iter.Seq[PartitionStatisticsFile] {
	return slices.Values(slices.Clone(c.PartitionStatsList))
}

func (c *commonMetadata) EncryptionKeys() iter.Seq[EncryptionKey] {
	return slices.Values(cloneEncryptionKeys(c.EncryptionKeyList))
}

func cloneSchema(schema *iceberg.Schema) *iceberg.Schema {
	if schema == nil {
		return nil
	}

	return iceberg.NewSchemaWithIdentifiers(
		schema.ID,
		slices.Clone(schema.IdentifierFieldIDs),
		schema.Fields()...,
	)
}

func cloneSchemas(schemas []*iceberg.Schema) []*iceberg.Schema {
	if schemas == nil {
		return nil
	}

	clones := make([]*iceberg.Schema, len(schemas))
	for i, schema := range schemas {
		clones[i] = cloneSchema(schema)
	}

	return clones
}

func clonePartitionSpecs(specs []iceberg.PartitionSpec) []iceberg.PartitionSpec {
	if specs == nil {
		return nil
	}

	clones := make([]iceberg.PartitionSpec, len(specs))
	for i, spec := range specs {
		clones[i] = spec.Clone()
	}

	return clones
}

func cloneSnapshotRef(ref SnapshotRef) SnapshotRef {
	clone := ref
	if ref.MinSnapshotsToKeep != nil {
		value := *ref.MinSnapshotsToKeep
		clone.MinSnapshotsToKeep = &value
	}
	if ref.MaxSnapshotAgeMs != nil {
		value := *ref.MaxSnapshotAgeMs
		clone.MaxSnapshotAgeMs = &value
	}
	if ref.MaxRefAgeMs != nil {
		value := *ref.MaxRefAgeMs
		clone.MaxRefAgeMs = &value
	}

	return clone
}

func cloneSnapshotRefs(refs map[string]SnapshotRef) map[string]SnapshotRef {
	if refs == nil {
		return nil
	}

	clones := make(map[string]SnapshotRef, len(refs))
	for name, ref := range refs {
		clones[name] = cloneSnapshotRef(ref)
	}

	return clones
}

func cloneCollected[T any](values []T) []T {
	if len(values) == 0 {
		return nil
	}

	return slices.Clone(values)
}

func cloneSnapshot(snapshot Snapshot) Snapshot {
	clone := snapshot
	if snapshot.ManifestLocations != nil {
		clone.ManifestLocations = make([]string, len(snapshot.ManifestLocations))
		copy(clone.ManifestLocations, snapshot.ManifestLocations)
	}
	if snapshot.ParentSnapshotID != nil {
		value := *snapshot.ParentSnapshotID
		clone.ParentSnapshotID = &value
	}
	if snapshot.SchemaID != nil {
		value := *snapshot.SchemaID
		clone.SchemaID = &value
	}
	if snapshot.FirstRowID != nil {
		value := *snapshot.FirstRowID
		clone.FirstRowID = &value
	}
	if snapshot.AddedRows != nil {
		value := *snapshot.AddedRows
		clone.AddedRows = &value
	}
	if snapshot.Summary != nil {
		clone.Summary = &Summary{
			Operation:  snapshot.Summary.Operation,
			Properties: maps.Clone(snapshot.Summary.Properties),
		}
	}

	return clone
}

func cloneSnapshotPtr(snapshot *Snapshot) *Snapshot {
	if snapshot == nil {
		return nil
	}

	clone := cloneSnapshot(*snapshot)

	return &clone
}

func cloneSnapshots(snapshots []Snapshot) []Snapshot {
	if snapshots == nil {
		return nil
	}

	clones := make([]Snapshot, len(snapshots))
	for i, snapshot := range snapshots {
		clones[i] = cloneSnapshot(snapshot)
	}

	return clones
}

func cloneSortOrder(order SortOrder) SortOrder {
	clone := order
	clone.fields = make([]SortField, len(order.fields))
	for i, field := range order.fields {
		clone.fields[i] = cloneSortField(field)
	}

	return clone
}

func cloneSortOrders(orders []SortOrder) []SortOrder {
	if orders == nil {
		return nil
	}

	clones := make([]SortOrder, len(orders))
	for i, order := range orders {
		clones[i] = cloneSortOrder(order)
	}

	return clones
}

func cloneStatisticsFiles(stats []StatisticsFile) []StatisticsFile {
	if stats == nil {
		return nil
	}

	clones := make([]StatisticsFile, len(stats))
	for i, stat := range stats {
		clones[i] = stat
		if stat.KeyMetadata != nil {
			value := *stat.KeyMetadata
			clones[i].KeyMetadata = &value
		}
		if stat.BlobMetadata != nil {
			clones[i].BlobMetadata = make([]BlobMetadata, len(stat.BlobMetadata))
			for j, blob := range stat.BlobMetadata {
				clones[i].BlobMetadata[j] = blob
				clones[i].BlobMetadata[j].Fields = slices.Clone(blob.Fields)
				clones[i].BlobMetadata[j].Properties = maps.Clone(blob.Properties)
			}
		}
	}

	return clones
}

func cloneEncryptionKeys(keys []EncryptionKey) []EncryptionKey {
	if keys == nil {
		return nil
	}

	clones := make([]EncryptionKey, len(keys))
	for i, key := range keys {
		clones[i] = key
		if key.EncryptedByID != nil {
			value := *key.EncryptedByID
			clones[i].EncryptedByID = &value
		}
		clones[i].Properties = maps.Clone(key.Properties)
	}

	return clones
}

// preValidate updates values in the metadata struct with defaults based on
// combinations of struct members. Such as initializing slices as empty slices
// if they were null in the metadata, or normalizing inconsistencies between
// metadata versions.
func (c *commonMetadata) preValidate() {
	if c.CurrentSnapshotID != nil && *c.CurrentSnapshotID == -1 {
		// treat -1 as the same as nil, clean this up in pre-validation
		// to make the validation logic simplified later
		c.CurrentSnapshotID = nil
	}

	if c.SnapshotRefs == nil {
		c.SnapshotRefs = map[string]SnapshotRef{}
	}

	if c.CurrentSnapshotID != nil {
		if _, ok := c.SnapshotRefs[MainBranch]; !ok {
			c.SnapshotRefs[MainBranch] = SnapshotRef{
				SnapshotID:      *c.CurrentSnapshotID,
				SnapshotRefType: BranchRef,
			}
		}
	}

	if c.MetadataLog == nil {
		c.MetadataLog = []MetadataLogEntry{}
	}

	if c.SnapshotRefs == nil {
		c.SnapshotRefs = make(map[string]SnapshotRef)
	}

	if c.SnapshotLog == nil {
		c.SnapshotLog = []SnapshotLogEntry{}
	}

	c.schemaIndex = buildSchemaIndex(c.SchemaList)
	c.partitionSpecIndex = buildPartitionSpecIndex(c.Specs)
	c.snapshotIndex = buildSnapshotIndex(c.SnapshotList)
}

func (c *commonMetadata) checkSchemas() error {
	seen := make(map[int]struct{}, len(c.SchemaList))
	currentFound := false
	for i, s := range c.SchemaList {
		if s == nil {
			return fmt.Errorf("%w: schema at index %d is null", ErrInvalidMetadata, i)
		}
		if _, ok := seen[s.ID]; ok {
			return fmt.Errorf("%w: duplicate schema ID %d", ErrInvalidMetadata, s.ID)
		}
		seen[s.ID] = struct{}{}

		if s.ID == c.CurrentSchemaID {
			currentFound = true
		}
	}
	if currentFound {
		return nil
	}

	return fmt.Errorf("%w: current-schema-id %d can't be found in any schema",
		ErrInvalidMetadata, c.CurrentSchemaID)
}

func (c *commonMetadata) checkPartitionSpecs() error {
	// Partition spec IDs are unique in persisted metadata. Keep this validation
	// aligned with the schema and snapshot ID checks so normal read paths never
	// need to tolerate duplicate IDs.
	seen := make(map[int]struct{}, len(c.Specs))
	defaultFound := false
	for i := range c.Specs {
		id := c.Specs[i].ID()
		if _, ok := seen[id]; ok {
			return fmt.Errorf("%w: duplicate partition spec ID %d", ErrInvalidMetadata, id)
		}
		seen[id] = struct{}{}

		if id == c.DefaultSpecID {
			defaultFound = true
		}
	}
	if defaultFound {
		return nil
	}

	return fmt.Errorf("%w: default-spec-id %d can't be found",
		ErrInvalidMetadata, c.DefaultSpecID)
}

func (c *commonMetadata) checkSortOrders() error {
	if c.DefaultSortOrderID == UnsortedSortOrderID {
		return nil
	}

	for _, o := range c.SortOrderList {
		if o.OrderID() == c.DefaultSortOrderID {
			if err := o.CheckCompatibility(c.CurrentSchema()); err != nil {
				return fmt.Errorf("%w: default sort order %d is not compatible with current schema: %w", ErrInvalidMetadata, o.OrderID(), err)
			}

			return nil
		}
		if o.OrderID() == UnsortedSortOrderID && o.Len() != 0 {
			return fmt.Errorf("%w: sort order ID %d is reserved for unsorted order", ErrInvalidMetadata, UnsortedSortOrderID)
		}
	}

	return fmt.Errorf("%w: default-sort-order-id %d can't be found in %+v",
		ErrInvalidMetadata, c.DefaultSortOrderID, c.SortOrderList)
}

func (c *commonMetadata) constructRefs() {
	if c.CurrentSnapshotID != nil {
		_, ok := c.SnapshotRefs[MainBranch]
		if !ok {
			c.SnapshotRefs[MainBranch] = SnapshotRef{
				SnapshotID:      *c.CurrentSnapshotID,
				SnapshotRefType: BranchRef,
			}
		}
	}
}

func (c *commonMetadata) validate() error {
	switch {
	case c.LastColumnId < 0:
		// last-column-id is required
		return fmt.Errorf("%w: missing last-column-id", ErrInvalidMetadata)
	case c.CurrentSchemaID < 0:
		return fmt.Errorf("%w: no valid schema configuration found in table metadata", ErrInvalidMetadata)
	case c.SortOrderList == nil && c.FormatVersion > 1:
		return fmt.Errorf("%w: missing sort-orders", ErrInvalidMetadata)
	case c.Specs == nil && c.FormatVersion > 1:
		return fmt.Errorf("%w: missing partition-specs", ErrInvalidMetadata)
	case c.DefaultSortOrderID < 0 && c.FormatVersion > 1:
		return fmt.Errorf("%w: default-sort-order-id must be set for FormatVersion > 1", ErrInvalidMetadata)
	case c.DefaultPartitionSpec() < 0 && c.FormatVersion > 1:
		return fmt.Errorf("%w: default-partition-spec-id must be set for FormatVersion > 1", ErrInvalidMetadata)
	case c.LastPartitionID == nil:
		if c.FormatVersion > 1 {
			return fmt.Errorf("%w: last-partition-id must be set for FormatVersion > 1", ErrInvalidMetadata)
		}
	}

	if err := c.validateChronologicalSnapshotLogs(); err != nil {
		return err
	}

	if err := c.validateChronologicalMetadataLogs(); err != nil {
		return err
	}

	if err := c.checkSchemas(); err != nil {
		return err
	}

	if err := c.checkPartitionSpecs(); err != nil {
		return err
	}

	if err := c.checkSortOrders(); err != nil {
		return err
	}

	if err := c.checkSnapshots(); err != nil {
		return err
	}

	c.constructRefs()

	if err := c.checkMainRefMatchesCurrentSnapshot(); err != nil {
		return err
	}

	if err := c.checkRefsExist(); err != nil {
		return err
	}

	return nil
}

func (c *commonMetadata) checkSnapshots() error {
	seen := make(map[int64]struct{}, len(c.SnapshotList))
	for _, snapshot := range c.SnapshotList {
		if _, ok := seen[snapshot.SnapshotID]; ok {
			return fmt.Errorf("%w: duplicate snapshot ID %d", ErrInvalidMetadata, snapshot.SnapshotID)
		}
		seen[snapshot.SnapshotID] = struct{}{}
	}

	return nil
}

func (c *commonMetadata) NameMapping() iceberg.NameMapping {
	if nameMappingJson, ok := c.Props[DefaultNameMappingKey]; ok {
		nm := iceberg.NameMapping{}
		if err := json.Unmarshal([]byte(nameMappingJson), &nm); err == nil {
			return nm
		}
	}

	return nil
}

func (c *commonMetadata) Version() int { return c.FormatVersion }

func (c *commonMetadata) checkMainRefMatchesCurrentSnapshot() error {
	if c.CurrentSnapshotID != nil {
		if ref, ok := c.SnapshotRefs[MainBranch]; ok {
			if ref.SnapshotID != *c.CurrentSnapshotID {
				return fmt.Errorf("%w: main branch snapshot ID %d does not match current snapshot ID %d",
					ErrInvalidMetadata, ref.SnapshotID, *c.CurrentSnapshotID)
			}
		}
	} else {
		if _, ok := c.SnapshotRefs[MainBranch]; ok {
			return fmt.Errorf("%w: main branch snapshot exists, but current snapshot ID is nil",
				ErrInvalidMetadata)
		}
	}

	return nil
}

func (c *commonMetadata) checkRefsExist() error {
	for name, ref := range c.SnapshotRefs {
		snap := c.SnapshotByID(ref.SnapshotID)
		if snap == nil {
			return fmt.Errorf("%w: snapshot ref %s with ID %d does not exist in snapshot list",
				ErrInvalidMetadata, name, ref.SnapshotID)
		}
	}

	return nil
}

func (c *commonMetadata) validateChronologicalSnapshotLogs() error {
	if !slices.IsSortedFunc(c.SnapshotLog, func(cur, prev SnapshotLogEntry) int {
		return cmp.Compare(cur.TimestampMs-prev.TimestampMs, -oneMinuteInMs)
	}) {
		return fmt.Errorf("%w: expected sorted snapshot log entries", ErrInvalidMetadata)
	}
	if len(c.SnapshotLog) > 0 {
		last := c.SnapshotLog[len(c.SnapshotLog)-1].TimestampMs
		if cmp.Compare(c.LastUpdatedMS-last, -oneMinuteInMs) < 0 {
			return fmt.Errorf("%w: invalid update timestamp %d: before last snapshot log entry at %d", ErrInvalidMetadata, c.LastUpdatedMS, last)
		}
	}

	return nil
}

func (c *commonMetadata) validateChronologicalMetadataLogs() error {
	if !slices.IsSortedFunc(c.MetadataLog, func(cur, prev MetadataLogEntry) int {
		return cmp.Compare(cur.TimestampMs-prev.TimestampMs, -oneMinuteInMs)
	}) {
		return fmt.Errorf("%w: expected sorted metadata log entries", ErrInvalidMetadata)
	}

	if len(c.MetadataLog) > 0 {
		last := c.MetadataLog[len(c.MetadataLog)-1].TimestampMs
		if cmp.Compare(c.LastUpdatedMS-last, -oneMinuteInMs) < 0 {
			return fmt.Errorf("%w: invalid update timestamp %d: before last metadata log entry at %d", ErrInvalidMetadata, c.LastUpdatedMS, last)
		}
	}

	return nil
}

type metadataV1 struct {
	Schema    *iceberg.Schema          `json:"schema,omitempty"`
	Partition []iceberg.PartitionField `json:"partition-spec,omitempty"`

	commonMetadata
}

func initMetadataV1Deser() *metadataV1 {
	meta := metadataV1{
		commonMetadata: initCommonMetadataForDeserialization(),
	}
	meta.DefaultSortOrderID = 0

	return &meta
}

func (m *metadataV1) LastSequenceNumber() int64 { return 0 }
func (m *metadataV1) NextRowID() int64          { return 0 } // V1 doesn't support row lineage

func (m *metadataV1) Equals(other Metadata) bool {
	rhs, ok := other.(*metadataV1)
	if !ok {
		return false
	}

	if m == rhs {
		return true
	}

	return m.Schema.Equals(rhs.Schema) && slices.EqualFunc(m.Partition, rhs.Partition, func(a, b iceberg.PartitionField) bool {
		return a.Equals(b)
	}) && m.commonMetadata.Equals(&rhs.commonMetadata)
}

func (m *metadataV1) preValidate() {
	if len(m.SchemaList) == 0 && m.Schema != nil {
		m.SchemaList = []*iceberg.Schema{m.Schema}
	}

	if len(m.Specs) == 0 {
		m.Specs = []iceberg.PartitionSpec{
			iceberg.NewPartitionSpec(m.Partition...),
		}
		m.DefaultSpecID = m.Specs[0].ID()
	}

	if m.DefaultSpecID == -1 && len(m.Specs) > 0 {
		m.DefaultSpecID = maxBy(m.Specs, func(s iceberg.PartitionSpec) int { return s.ID() })
	}

	if m.LastPartitionID == nil {
		id := m.Specs[0].LastAssignedFieldID()
		for _, spec := range m.Specs[1:] {
			last := spec.LastAssignedFieldID()
			if last > id {
				id = last
			}
		}
		m.LastPartitionID = &id
	}

	if len(m.SortOrderList) == 0 {
		m.SortOrderList = []SortOrder{UnsortedSortOrder}
	}

	m.commonMetadata.preValidate()
}

func (m *metadataV1) UnmarshalJSON(b []byte) error {
	type Alias metadataV1
	next := initMetadataV1Deser()
	aux := (*Alias)(next)

	if err := json.Unmarshal(b, aux); err != nil {
		return err
	}
	if err := next.finishUnmarshal(); err != nil {
		return err
	}

	*m = *next

	return nil
}

func (m *metadataV1) finishUnmarshal() error {
	if err := rejectFieldsBeyondVersion(
		m.FormatVersion,
		versionScopedField{name: "last-sequence-number", introduced: 2, present: m.commonMetadata.LastSequenceNumber != nil},
		versionScopedField{name: "next-row-id", introduced: 3, present: m.commonMetadata.NextRowID != nil},
		versionScopedField{name: "encryption-keys", introduced: 3, present: len(m.EncryptionKeyList) > 0},
	); err != nil {
		return err
	}

	// CurrentSchemaID was optional in v1, it can also be expressed via Schema.
	if m.CurrentSchemaID == -1 && m.Schema != nil {
		m.CurrentSchemaID = m.Schema.ID
		if !slices.ContainsFunc(m.SchemaList, func(s *iceberg.Schema) bool {
			return s.Equals(m.Schema) && s.ID == m.CurrentSchemaID
		}) {
			m.SchemaList = append(m.SchemaList, m.Schema)
		}
	}

	m.preValidate()
	if err := m.checkRequiredFields(); err != nil {
		return err
	}

	if err := m.validate(); err != nil {
		return err
	}

	return nil
}

func (m metadataV1) MarshalJSON() ([]byte, error) {
	snapshots, err := m.snapshotsForMarshal()
	if err != nil {
		return nil, err
	}

	type Alias metadataV1

	return json.Marshal(&struct {
		*Alias
		SnapshotList []Snapshot `json:"snapshots,omitempty"`
	}{
		Alias:        (*Alias)(&m),
		SnapshotList: snapshots,
	})
}

func (m *metadataV1) ToV2() metadataV2 {
	commonOut := m.commonMetadata
	commonOut.SnapshotList = m.allSnapshots()
	commonOut.snapshotIndex = buildSnapshotIndex(commonOut.SnapshotList)
	commonOut.deferredSnapshots = nil
	commonOut.FormatVersion = 2
	if commonOut.UUID == uuid.Nil {
		commonOut.UUID = uuid.New()
	}

	return metadataV2{commonMetadata: commonOut}
}

type metadataV2 struct {
	LastSeqNum int64 `json:"last-sequence-number"`

	commonMetadata
}

func initMetadataV2Deser() *metadataV2 {
	return &metadataV2{
		LastSeqNum:     -1,
		commonMetadata: initCommonMetadataForDeserialization(),
	}
}

func (m *metadataV2) LastSequenceNumber() int64 { return m.LastSeqNum }
func (m *metadataV2) NextRowID() int64          { return 0 } // V2 doesn't support row lineage

func (m *metadataV2) Equals(other Metadata) bool {
	rhs, ok := other.(*metadataV2)
	if !ok {
		return false
	}

	if m == rhs {
		return true
	}

	return m.LastSeqNum == rhs.LastSeqNum &&
		m.commonMetadata.Equals(&rhs.commonMetadata)
}

func (m *metadataV2) UnmarshalJSON(b []byte) error {
	type Alias metadataV2
	next := initMetadataV2Deser()
	aux := (*Alias)(next)

	if err := json.Unmarshal(b, aux); err != nil {
		return err
	}
	if err := next.finishUnmarshal(); err != nil {
		return err
	}

	*m = *next

	return nil
}

func (m *metadataV2) finishUnmarshal() error {
	if err := rejectFieldsBeyondVersion(
		m.FormatVersion,
		versionScopedField{name: "next-row-id", introduced: 3, present: m.commonMetadata.NextRowID != nil},
		versionScopedField{name: "encryption-keys", introduced: 3, present: len(m.EncryptionKeyList) > 0},
	); err != nil {
		return err
	}

	m.preValidate()
	if err := m.checkRequiredFields(); err != nil {
		return err
	}

	if err := m.validate(); err != nil {
		return err
	}

	return nil
}

func (m metadataV2) MarshalJSON() ([]byte, error) {
	snapshots, err := m.snapshotsForMarshal()
	if err != nil {
		return nil, err
	}

	type Alias metadataV2

	return json.Marshal(&struct {
		*Alias
		SnapshotList []Snapshot `json:"snapshots,omitempty"`
	}{
		Alias:        (*Alias)(&m),
		SnapshotList: snapshots,
	})
}

func (m *metadataV2) validate() error {
	if err := checkLastSequenceNumber(m, m.SnapshotList); err != nil {
		return err
	}

	if err := m.commonMetadata.validate(); err != nil {
		return err
	}

	return nil
}

// metadataV3 represents table metadata for format version 3
type metadataV3 struct {
	LastSeqNum     int64 `json:"last-sequence-number"`
	NextRowIDValue int64 `json:"next-row-id"`

	commonMetadata
}

func initMetadataV3Deser() *metadataV3 {
	return &metadataV3{
		LastSeqNum:     -1,
		NextRowIDValue: -1,
		commonMetadata: initCommonMetadataForDeserialization(),
	}
}

func (m *metadataV3) LastSequenceNumber() int64 { return m.LastSeqNum }
func (m *metadataV3) NextRowID() int64 {
	if m.NextRowIDValue == -1 {
		return 0 // For v1/v2 compatibility when field is missing
	}

	return max(0, m.NextRowIDValue)
}

func (m *metadataV3) Equals(other Metadata) bool {
	rhs, ok := other.(*metadataV3)
	if !ok {
		return false
	}

	if m == rhs {
		return true
	}

	return m.LastSeqNum == rhs.LastSeqNum &&
		m.NextRowIDValue == rhs.NextRowIDValue &&
		m.commonMetadata.Equals(&rhs.commonMetadata)
}

func (m *metadataV3) UnmarshalJSON(b []byte) error {
	type Alias metadataV3
	next := initMetadataV3Deser()
	aux := (*Alias)(next)

	if err := json.Unmarshal(b, aux); err != nil {
		return err
	}
	if err := next.finishUnmarshal(); err != nil {
		return err
	}

	*m = *next

	return nil
}

func (m *metadataV3) finishUnmarshal() error {
	if err := rejectFieldsBeyondVersion(m.FormatVersion); err != nil {
		return err
	}

	m.preValidate()
	if err := m.checkRequiredFields(); err != nil {
		return err
	}

	if err := m.validate(); err != nil {
		return err
	}

	return nil
}

// MarshalJSON serializes v3 metadata with the spec-mandated emission of
// `current-snapshot-id: null` for empty tables, rather than dropping the
// key via the underlying `omitempty` tag (Java reference behaviour per
// apache/iceberg#12335). The override re-declares `CurrentSnapshotID`
// without omitempty in an outer struct that embeds an alias of
// `metadataV3`; Go's encoding/json resolves the field-name collision in
// favour of the shallower (outer) declaration, so the value is always
// emitted — null when nil, the id otherwise.
//
// v1/v2 keep the omitempty path since their fixtures use the -1 sentinel
// for "no current snapshot".
//
// `last-partition-id` is required for v2+ per spec — the existing parse-
// time validation enforces this on read; the symmetric check below
// rejects writes of malformed in-memory state (builder/parser paths
// always populate it, so this only fires when code constructs metadataV3
// directly and skips the builder).
func (m metadataV3) MarshalJSON() ([]byte, error) {
	if m.LastPartitionID == nil {
		return nil, fmt.Errorf("%w: last-partition-id must be set for v3 metadata", ErrInvalidMetadata)
	}
	snapshots, err := m.snapshotsForMarshal()
	if err != nil {
		return nil, err
	}

	// Alias strips the MarshalJSON method off metadataV3 so json.Marshal
	// doesn't recurse back into this function via the embedded pointer.
	type Alias metadataV3

	return json.Marshal(&struct {
		*Alias
		CurrentSnapshotID *int64     `json:"current-snapshot-id"`
		SnapshotList      []Snapshot `json:"snapshots,omitempty"`
	}{
		Alias:             (*Alias)(&m),
		CurrentSnapshotID: m.CurrentSnapshotID,
		SnapshotList:      snapshots,
	})
}

func (m *metadataV3) validate() error {
	if err := checkLastSequenceNumber(m, m.SnapshotList); err != nil {
		return err
	}

	if err := m.checkNextRowID(); err != nil {
		return err
	}

	if err := m.commonMetadata.validate(); err != nil {
		return err
	}

	return nil
}

func (m *metadataV3) checkNextRowID() error {
	if m.NextRowIDValue < 0 {
		if m.NextRowIDValue == -1 {
			return fmt.Errorf("%w: next-row-id is required for v3 tables", ErrInvalidMetadata)
		}

		return fmt.Errorf("%w: next-row-id must be non-negative, got %d", ErrInvalidMetadata, m.NextRowIDValue)
	}

	for _, snap := range m.SnapshotList {
		if snap.FirstRowID != nil && *snap.FirstRowID < 0 {
			return fmt.Errorf("%w: snapshot %d has invalid first-row-id %d",
				ErrInvalidMetadata, snap.SnapshotID, *snap.FirstRowID)
		}
	}

	return nil
}

// SequenceNumberValidator defines an interface for types that can validate sequence numbers
type SequenceNumberValidator interface {
	LastSequenceNumber() int64
}

// checkLastSequenceNumber validates that all snapshots have sequence numbers <= the last sequence number
func checkLastSequenceNumber(validator SequenceNumberValidator, snapshotList []Snapshot) error {
	if err := requireLastSequenceNumber(validator); err != nil {
		return err
	}

	lastSequenceNumber := validator.LastSequenceNumber()
	for _, snap := range snapshotList {
		if snap.SequenceNumber > lastSequenceNumber {
			return fmt.Errorf("%w: snapshot %d has sequence number %d which is greater than last-sequence-number %d",
				ErrInvalidMetadata, snap.SnapshotID, snap.SequenceNumber, lastSequenceNumber)
		}
	}

	return nil
}

func requireLastSequenceNumber(validator SequenceNumberValidator) error {
	if validator.LastSequenceNumber() == -1 {
		return fmt.Errorf("%w: last-sequence-number is required for format versions greater than 1", ErrInvalidMetadata)
	}

	return nil
}

const DefaultFormatVersion = 2

// NewMetadata creates a new table metadata object using the provided schema, information, generating a fresh UUID for
// the new table metadata. By default, this will generate a V2 table metadata, but this can be modified
// by adding a "format-version" property to the props map. An error will be returned if the "format-version"
// property exists and is not a valid version number.
//
// location should be non-empty: [ParseMetadata] rejects persisted metadata
// without one, so metadata built with an empty location cannot be read back.
// The constructor stays permissive because catalog create-via-commit paths seed
// a location-less base and supply the location as a set-location update.
func NewMetadata(sc *iceberg.Schema, partitions *iceberg.PartitionSpec, sortOrder SortOrder, location string, props iceberg.Properties) (Metadata, error) {
	return NewMetadataWithUUID(sc, partitions, sortOrder, location, props, uuid.Nil)
}

// NewMetadataWithUUID is like NewMetadata, but allows the caller to specify the UUID of the table rather than creating a new one.
func NewMetadataWithUUID(sc *iceberg.Schema, partitions *iceberg.PartitionSpec, sortOrder SortOrder, location string, props iceberg.Properties, tableUuid uuid.UUID) (Metadata, error) {
	if tableUuid == uuid.Nil {
		tableUuid = uuid.New()
	}
	var inputProps iceberg.Properties
	var err error
	formatVersion := DefaultFormatVersion
	if props != nil {
		inputProps = maps.Clone(props)
		verStr, ok := inputProps[PropertyFormatVersion]
		if ok {
			if formatVersion, err = strconv.Atoi(verStr); err != nil {
				return nil, fmt.Errorf("%w: %s", iceberg.ErrInvalidFormatVersion, verStr)
			}
			delete(inputProps, PropertyFormatVersion)
		}
	}

	// Reject reserved metadata column IDs on the user-supplied schema before
	// reassignIDs runs: reassignment overwrites every field ID with a fresh,
	// non-reserved value, so a reserved ID (e.g. _row_id) would otherwise be
	// silently remapped and later resolve to the wrong physical column against
	// Java-written files (see #1107). This is stricter than Java, which relies
	// on assignFreshIds to remap them; rejecting is the safer contract here.
	if err := validateNoReservedFieldIDs(sc); err != nil {
		return nil, err
	}

	reassignedIds, err := reassignIDs(sc, partitions, sortOrder)
	if err != nil {
		return nil, err
	}

	builder, err := NewMetadataBuilder(formatVersion)
	if err != nil {
		return nil, err
	}

	if err = builder.SetUUID(tableUuid); err != nil {
		return nil, err
	}

	if err = builder.AddSchema(reassignedIds.schema); err != nil {
		return nil, err
	}

	if err = builder.SetCurrentSchemaID(-1); err != nil {
		return nil, err
	}

	if err = builder.AddSortOrder(&reassignedIds.sortOrder); err != nil {
		return nil, err
	}

	if err = builder.SetDefaultSortOrderID(-1); err != nil {
		return nil, err
	}

	if err = builder.AddPartitionSpec(reassignedIds.partitionSpec, true); err != nil {
		return nil, err
	}

	if err = builder.SetDefaultSpecID(-1); err != nil {
		return nil, err
	}

	if err = builder.SetLoc(location); err != nil {
		return nil, err
	}

	if err = builder.SetProperties(inputProps); err != nil {
		return nil, err
	}

	return builder.Build()
}

type ReassignedIds struct {
	schema        *iceberg.Schema
	partitionSpec *iceberg.PartitionSpec
	sortOrder     SortOrder
}

func reassignIDs(sc *iceberg.Schema, partitions *iceberg.PartitionSpec, sortOrder SortOrder) (*ReassignedIds, error) {
	previousMapFn := sc.FindColumnName
	freshSc, err := iceberg.AssignFreshSchemaIDs(sc, nil)
	if err != nil {
		return nil, err
	}

	if partitions == nil {
		partitions = iceberg.UnpartitionedSpec
	}
	opts := make([]iceberg.PartitionOption, 0)
	for _, f := range partitions.Fields() {
		var s string
		var ok bool
		if s, ok = previousMapFn(f.SourceID()); !ok {
			return nil, fmt.Errorf("%w: partition source field %d not found in schema", ErrInvalidMetadata, f.SourceID())
		}
		opts = append(opts, iceberg.AddPartitionFieldByName(s, f.Name, f.Transform, freshSc, nil))
	}
	freshPartitions, err := iceberg.NewPartitionSpecOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidMetadata, err)
	}

	freshSortOrder, err := AssignFreshSortOrderIDs(sortOrder, sc, freshSc)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidMetadata, err)
	}

	return &ReassignedIds{
		schema:        freshSc,
		partitionSpec: &freshPartitions,
		sortOrder:     freshSortOrder,
	}, nil
}

func UpdateTableMetadata(base Metadata, updates []Update, metadataLoc string) (Metadata, error) {
	bldr, err := MetadataBuilderFromBase(base, metadataLoc)
	if err != nil {
		return nil, err
	}

	for _, update := range updates {
		if err := update.Apply(bldr); err != nil {
			return nil, err
		}
	}

	return bldr.Build()
}
