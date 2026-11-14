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

package view

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/view/internal"

	"github.com/google/uuid"
)

type MetadataBuilder struct {
	base    Metadata
	updates Updates

	// common fields
	formatVersion    int
	uuid             uuid.UUID
	loc              string
	schemaList       []*iceberg.Schema
	versionList      []*Version
	currentVersionID int64
	versionLog       []VersionLogEntry
	props            iceberg.Properties

	// lookup maps
	versionsById map[int64]*Version
	schemasById  map[int]*iceberg.Schema

	// update tracking
	versionHistoryEntry *VersionLogEntry
	previousVersion     *Version
	lastAddedVersionID  *int64
	lastAddedSchemaID   *int

	// error tracking for build chaining
	// if set, subsequent operations become a noop
	err error
}

// MetadataBuildResult is returned by a successful Build() command
// This type contains the resulting view metadata from a build, as
// well as the logical updates performed to generate it
type MetadataBuildResult struct {
	Metadata
	Changes Updates
}

func (m *MetadataBuildResult) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Metadata)
}

func NewMetadataBuilder() (*MetadataBuilder, error) {
	return &MetadataBuilder{
		formatVersion: DefaultViewFormatVersion,
		updates:       make([]Update, 0),
		schemaList:    make([]*iceberg.Schema, 0),
		versionList:   make([]*Version, 0),
		versionLog:    make([]VersionLogEntry, 0),
		props:         make(iceberg.Properties),
		versionsById:  make(map[int64]*Version),
		schemasById:   make(map[int]*iceberg.Schema),
	}, nil
}

func MetadataBuilderFromBase(metadata Metadata) (*MetadataBuilder, error) {
	b := &MetadataBuilder{}
	b.base = metadata

	// Copy fields
	b.formatVersion = metadata.FormatVersion()
	b.uuid = metadata.ViewUUID()
	b.loc = metadata.Location()
	b.schemaList = slices.Clone(metadata.Schemas())
	b.currentVersionID = metadata.CurrentVersionID()
	b.versionList = cloneSlice(metadata.Versions())
	b.versionLog = slices.Clone(metadata.VersionLog())
	b.props = maps.Clone(metadata.Properties())

	// Build lookup maps
	b.versionsById = indexBy(b.versionList, func(vl *Version) int64 { return vl.VersionID })
	b.schemasById = indexBy(b.schemaList, func(sm *iceberg.Schema) int { return sm.ID })

	// Version tracking
	b.previousVersion = metadata.CurrentVersion()

	return b, nil
}

func (b *MetadataBuilder) HasChanges() bool { return len(b.updates) > 0 }

func (b *MetadataBuilder) SetCurrentVersion(version *Version, schema *iceberg.Schema) *MetadataBuilder {
	if b.err != nil {
		return b
	}

	newSchemaID, err := b.addSchema(schema)
	if b.setErr(err) {
		return b
	}

	newVersion := version.Clone()
	newVersion.SchemaID = newSchemaID
	newVersionID, err := b.addVersion(newVersion)
	if b.setErr(err) {
		return b
	}

	return b.SetCurrentVersionID(newVersionID)
}

func (b *MetadataBuilder) AddVersion(newVersion *Version) *MetadataBuilder {
	if b.err != nil {
		return b
	}

	_, err := b.addVersion(newVersion)
	if b.setErr(err) {
		return b
	}

	return b
}

func (b *MetadataBuilder) addVersion(newVersion *Version) (int64, error) {
	newVersionID := b.reuseOrCreateNewVersionID(newVersion)
	version := newVersion.Clone()
	if newVersionID != version.VersionID {
		version.VersionID = newVersionID
	}

	// Check if this version was added in an update already
	if _, ok := b.versionsById[newVersionID]; ok {
		for _, upd := range b.updates {
			if vvu, ok := upd.(*addViewVersionUpdate); ok && vvu.Version.VersionID == newVersionID {
				b.lastAddedVersionID = &newVersionID

				return newVersionID, nil
			}
		}
	}

	// If the SchemaID of the version is unset (nil), we attach the lastAddedSchemaID to it.
	// If we have no lastAddedSchemaID, we fail
	if version.SchemaID == LastAddedID {
		if b.lastAddedSchemaID == nil {
			return 0, errors.New("cannot set last added schema: no schema has been added")
		}
		version.SchemaID = *b.lastAddedSchemaID
	}

	if _, ok := b.schemasById[version.SchemaID]; !ok {
		return 0, fmt.Errorf("cannot add version with unknown schema: %d", version.SchemaID)
	}

	if len(version.Representations) == 0 {
		return 0, errors.New("cannot add version with no representations")
	}
	dialects := make(map[string]struct{})
	for _, repr := range version.Representations {
		normalizedDialect, err := validateRepresentation(repr)
		if err != nil {
			return 0, fmt.Errorf("%w: %s", ErrInvalidViewMetadata, err)
		}

		if _, ok := dialects[normalizedDialect]; ok {
			return 0, fmt.Errorf("%w: Invalid view version: Cannot add multiple queries for dialect %s", ErrInvalidViewMetadata, repr.Dialect)
		}
		dialects[normalizedDialect] = struct{}{}
	}

	b.versionList = append(b.versionList, version)
	b.versionsById[version.VersionID] = version

	if b.lastAddedSchemaID != nil && version.SchemaID == *b.lastAddedSchemaID {
		updateVersion := version.Clone()
		updateVersion.SchemaID = LastAddedID
		b.updates = append(b.updates, NewAddViewVersionUpdate(updateVersion))
	} else {
		b.updates = append(b.updates, NewAddViewVersionUpdate(version))
	}

	b.lastAddedVersionID = &newVersionID

	return newVersionID, nil
}

func (b *MetadataBuilder) AddSchema(schema *iceberg.Schema) *MetadataBuilder {
	if b.err != nil {
		return b
	}

	_, err := b.addSchema(schema)
	if b.setErr(err) {
		return b
	}

	return b
}

func (b *MetadataBuilder) addSchema(schema *iceberg.Schema) (int, error) {
	newSchemaID := b.reuseOrCreateNewSchemaID(schema)

	if _, ok := b.schemasById[newSchemaID]; ok {
		return newSchemaID, nil
	}

	newSchema := schema
	// Build a fresh schema if we reset the ID
	if schema.ID != newSchemaID {
		newSchema = iceberg.NewSchemaWithIdentifiers(newSchemaID, schema.IdentifierFieldIDs, schema.Fields()...)
	}

	b.schemaList = append(b.schemaList, newSchema)
	b.schemasById[newSchema.ID] = newSchema
	b.updates = append(b.updates, NewAddSchemaUpdate(newSchema))
	b.lastAddedSchemaID = &newSchemaID

	return newSchemaID, nil
}

func (b *MetadataBuilder) RemoveProperties(keys []string) *MetadataBuilder {
	if b.err != nil {
		return b
	}

	if len(keys) == 0 {
		return b
	}

	b.updates = append(b.updates, NewRemovePropertiesUpdate(keys))
	for _, key := range keys {
		delete(b.props, key)
	}

	return b
}

func (b *MetadataBuilder) SetCurrentVersionID(newVersionID int64) *MetadataBuilder {
	if b.err != nil {
		return b
	}

	if newVersionID == LastAddedID {
		if b.lastAddedVersionID == nil {
			b.err = errors.New("cannot set current version to last added, no version has been added")

			return b
		}
		newVersionID = *b.lastAddedVersionID
	}

	if newVersionID == b.currentVersionID {
		return b
	}

	_, ok := b.versionsById[newVersionID]
	if !ok {
		b.err = fmt.Errorf("cannot set current version to unknown version with id %d", newVersionID)

		return b
	}

	if b.lastAddedVersionID != nil && *b.lastAddedVersionID == newVersionID {
		b.updates = append(b.updates, NewSetCurrentVersionUpdate(LastAddedID))
	} else {
		b.updates = append(b.updates, NewSetCurrentVersionUpdate(newVersionID))
	}
	b.currentVersionID = newVersionID

	// Set the current history entry.
	// If the version was added in the current set of changes, use its timestamp,
	// otherwise use system time.
	updateTimestampMS := time.Now().UnixMilli()
	for _, update := range b.updates {
		if v, ok := update.(*addViewVersionUpdate); ok && v.Version.VersionID == newVersionID {
			updateTimestampMS = v.Version.TimestampMS

			break
		}
	}

	b.versionHistoryEntry = &VersionLogEntry{
		VersionID:   newVersionID,
		TimestampMS: updateTimestampMS,
	}

	return b
}

func (b *MetadataBuilder) SetFormatVersion(formatVersion int) *MetadataBuilder {
	if b.err != nil {
		return b
	}

	if formatVersion < b.formatVersion {
		b.err = fmt.Errorf("downgrading format version from %d to %d is not allowed",
			b.formatVersion, formatVersion)

		return b
	}

	if formatVersion > SupportedViewFormatVersion {
		b.err = fmt.Errorf("unsupported format version %d", formatVersion)

		return b
	}

	if formatVersion == b.formatVersion {
		return b
	}

	b.updates = append(b.updates, NewUpgradeFormatVersionUpdate(formatVersion))
	b.formatVersion = formatVersion

	return b
}

func (b *MetadataBuilder) SetLoc(loc string) *MetadataBuilder {
	if b.err != nil {
		return b
	}

	if b.loc == loc {
		return b
	}

	b.updates = append(b.updates, NewSetLocationUpdate(loc))
	b.loc = loc

	return b
}

func (b *MetadataBuilder) SetProperties(props iceberg.Properties) *MetadataBuilder {
	if b.err != nil {
		return b
	}

	if len(props) == 0 {
		return b
	}

	b.updates = append(b.updates, NewSetPropertiesUpdate(props))
	if b.props == nil {
		b.props = props
	} else {
		maps.Copy(b.props, props)
	}

	return b
}

func (b *MetadataBuilder) SetUUID(newUUID uuid.UUID) *MetadataBuilder {
	if b.err != nil {
		return b
	}

	if newUUID == uuid.Nil {
		b.err = errors.New("cannot set uuid to null")

		return b
	}

	// Noop - same UUID does not generate a change.
	if b.uuid == newUUID {
		return b
	}

	if b.uuid != uuid.Nil {
		b.err = errors.New("cannot reassign uuid")

		return b
	}

	b.updates = append(b.updates, NewAssignUUIDUpdate(newUUID))
	b.uuid = newUUID

	return b
}

func (b *MetadataBuilder) Err() error {
	return b.err
}

func (b *MetadataBuilder) buildMetadata(retainedVersions []*Version, retainedHistory []VersionLogEntry) (*metadata, error) {
	uuid_ := b.uuid
	if uuid_ == uuid.Nil {
		uuid_ = uuid.New()
	}

	md := &metadata{
		FormatVersionValue:    b.formatVersion,
		UUID:                  &uuid_,
		Loc:                   b.loc,
		CurrentVersionIDValue: b.currentVersionID,
		VersionList:           retainedVersions,
		VersionLogList:        retainedHistory,
		SchemaList:            b.schemaList,
		Props:                 b.props,
	}
	md.init()

	return md, md.validate()
}

// Build builds the view metadata and updates from the builder
func (b *MetadataBuilder) Build() (*MetadataBuildResult, error) {
	if b.err != nil {
		return nil, b.err
	}

	if b.formatVersion != SupportedViewFormatVersion {
		return nil, fmt.Errorf("unsupported format version %d", b.formatVersion)
	}

	if b.versionHistoryEntry != nil {
		b.versionLog = append(b.versionLog, *b.versionHistoryEntry)
	}

	// If we had a previous version, check if we allow dropping dialects and if we did drop one
	if b.previousVersion != nil &&
		!b.props.GetBool(ReplaceDropDialectAllowedKey, ReplaceDropDialectAllowedDefault) {
		currentVersion := b.versionsById[b.currentVersionID]
		err := checkIfDialectIsDropped(b.previousVersion, currentVersion)
		if err != nil {
			return nil, err
		}
	}

	historySize := b.props.GetInt(VersionHistorySizeKey, VersionHistorySizeDefault)
	if historySize < 1 {
		return nil, fmt.Errorf("%s must be positive, found %d", VersionHistorySizeKey, historySize)
	}
	versionsToKeep := max(b.distinctVersionsInBuild(), historySize)
	var retainedVersions []*Version
	var retainedHistory []VersionLogEntry
	if len(b.versionList) > versionsToKeep {
		retainedVersions = expireVersions(b.versionsById, versionsToKeep, b.versionsById[b.currentVersionID])
		retainedVersionIDs := internal.ToSet(internal.MapSlice(retainedVersions, func(v *Version) int64 { return v.VersionID }))
		retainedHistory = updateHistory(b.versionLog, retainedVersionIDs)
	} else {
		retainedVersions = b.versionList
		retainedHistory = b.versionLog
	}

	md, err := b.buildMetadata(retainedVersions, retainedHistory)
	if err != nil {
		return nil, err
	}

	return &MetadataBuildResult{Metadata: md, Changes: b.updates}, nil
}

// setErr is a helper for enforcing build error assignment during operations.
// This method returns true if the error was non-nil and the builder has entered
// an error state. In this case, the operation must fail immediately, while still
// returning the builder instance.
//
// Example usage inside a public builder method:
//
//	func (b *MetadataBuilder) SetSomething(param int) (*MetadataBuilder) {
//	  val, err := b.internalOperation(param)
//	  if b.setErr(err) {
//	    return b
//	  }
//
//	  b.val = val
//	  return b
//	}
func (b *MetadataBuilder) setErr(err error) (buildHasFailed bool) {
	if err != nil {
		b.err = err

		return true
	}

	return false
}

func (b *MetadataBuilder) reuseOrCreateNewVersionID(newVersion *Version) int64 {
	newVersionID := newVersion.VersionID
	for _, version := range b.versionList {
		if newVersion.Equals(version) {
			return version.VersionID
		} else if version.VersionID >= newVersionID {
			newVersionID = version.VersionID + 1
		}
	}

	return newVersionID
}

func (b *MetadataBuilder) reuseOrCreateNewSchemaID(newSchema *iceberg.Schema) int {
	newSchemaID := initialSchemaID

	// if the schema already exists, use its id; otherwise use the highest id + 1
	for _, schema := range b.schemaList {
		if newSchema.Equals(schema) {
			return schema.ID
		} else if schema.ID >= newSchemaID {
			newSchemaID = schema.ID + 1
		}
	}

	return newSchemaID
}

// distinctVersionsInBuild returns the number of unique versions added in this builder
// as well as the current active version.
func (b *MetadataBuilder) distinctVersionsInBuild() int {
	versions := map[int64]struct{}{b.currentVersionID: {}}
	for _, update := range b.updates {
		if avv, ok := update.(*addViewVersionUpdate); ok {
			versions[avv.Version.VersionID] = struct{}{}
		}
	}

	return len(versions)
}

func checkIfDialectIsDropped(previous *Version, current *Version) error {
	prevDialects := previous.sqlDialects()
	currDialects := current.sqlDialects()
	if !prevDialects.IsSubset(currDialects) {
		return fmt.Errorf(
			"dropping dialects is not enabled for this view (%s=false):\nprevious dialects:%v\nnew dialects:%v",
			ReplaceDropDialectAllowedKey,
			slices.Collect(maps.Keys(prevDialects)),
			slices.Collect(maps.Keys(currDialects)))
	}

	return nil
}

// expireVersions builds a list of versions, only retaining ones which have not expired out
// due to retention policy.
func expireVersions(versionsByID map[int64]*Version, numVersionsToKeep int, currentVersion *Version) []*Version {
	reverseOrderIDs := slices.SortedFunc(maps.Keys(versionsByID), func(a, b int64) int {
		return int(b - a)
	})
	retainedVersions := make([]*Version, 0, numVersionsToKeep)
	retainedVersions = append(retainedVersions, currentVersion)

	for _, versionID := range reverseOrderIDs[:numVersionsToKeep] {
		if len(retainedVersions) == numVersionsToKeep {
			break
		}

		if currentVersion.VersionID != versionID {
			retainedVersions = append(retainedVersions, versionsByID[versionID])
		}
	}

	return retainedVersions
}

func updateHistory(history []VersionLogEntry, retainedIDs internal.Set[int64]) []VersionLogEntry {
	retainedHistory := make([]VersionLogEntry, 0)
	for _, entry := range history {
		if retainedIDs.Contains(entry.VersionID) {
			retainedHistory = append(retainedHistory, entry)
		} else {
			// clear history past any unknown version
			retainedHistory = retainedHistory[:0]
		}
	}

	return retainedHistory
}
