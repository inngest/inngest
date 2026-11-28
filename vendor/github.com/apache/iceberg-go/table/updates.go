// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/io"
	"github.com/google/uuid"
)

// These are the various update actions defined in the iceberg spec
const (
	UpdateAddSpec      = "add-spec"
	UpdateAddSchema    = "add-schema"
	UpdateAddSnapshot  = "add-snapshot"
	UpdateAddSortOrder = "add-sort-order"

	UpdateAssignUUID = "assign-uuid"

	UpdateAddEncryptionKey          = "add-encryption-key"
	UpdateRemoveEncryptionKey       = "remove-encryption-key"
	UpdateRemovePartitionStatistics = "remove-partition-statistics"
	UpdateRemoveProperties          = "remove-properties"
	UpdateRemoveSchemas             = "remove-schemas"
	UpdateRemoveSnapshots           = "remove-snapshots"
	UpdateRemoveSnapshotRef         = "remove-snapshot-ref"
	UpdateRemoveSpec                = "remove-partition-specs"
	UpdateRemoveStatistics          = "remove-statistics"

	UpdateSetCurrentSchema       = "set-current-schema"
	UpdateSetDefaultSortOrder    = "set-default-sort-order"
	UpdateSetDefaultSpec         = "set-default-spec"
	UpdateSetLocation            = "set-location"
	UpdateSetPartitionStatistics = "set-partition-statistics"
	UpdateSetProperties          = "set-properties"
	UpdateSetSnapshotRef         = "set-snapshot-ref"
	UpdateSetStatistics          = "set-statistics"

	UpdateUpgradeFormatVersion = "upgrade-format-version"
)

// Update represents a change to a table's metadata.
type Update interface {
	// Action returns the name of the action that the update represents.
	Action() string
	// Apply applies the update to the given metadata builder.
	Apply(*MetadataBuilder) error
	// PostCommit is called after successful commit of the update
	PostCommit(context.Context, *Table, *Table) error
}

type Updates []Update

var requiredUpdateFields = map[string][]string{
	UpdateAssignUUID:                {"uuid"},
	UpdateUpgradeFormatVersion:      {"format-version"},
	UpdateAddSchema:                 {"schema"},
	UpdateSetCurrentSchema:          {"schema-id"},
	UpdateAddSpec:                   {"spec"},
	UpdateSetDefaultSpec:            {"spec-id"},
	UpdateAddSortOrder:              {"sort-order"},
	UpdateSetDefaultSortOrder:       {"sort-order-id"},
	UpdateAddSnapshot:               {"snapshot"},
	UpdateSetSnapshotRef:            {"ref-name", "type", "snapshot-id"},
	UpdateRemoveSnapshots:           {"snapshot-ids"},
	UpdateRemoveSnapshotRef:         {"ref-name"},
	UpdateSetLocation:               {"location"},
	UpdateSetProperties:             {"updates"},
	UpdateRemoveProperties:          {"removals"},
	UpdateRemoveSpec:                {"spec-ids"},
	UpdateRemoveSchemas:             {"schema-ids"},
	UpdateSetStatistics:             {"statistics"},
	UpdateRemoveStatistics:          {"snapshot-id"},
	UpdateSetPartitionStatistics:    {"partition-statistics"},
	UpdateRemovePartitionStatistics: {"snapshot-id"},
	UpdateAddEncryptionKey:          {"encryption-key"},
	UpdateRemoveEncryptionKey:       {"key-id"},
}

func normalizeLegacyPropertyFields(action string, object map[string]json.RawMessage) bool {
	var modern, legacy string
	switch action {
	case UpdateSetProperties:
		modern, legacy = "updates", "updated"
	case UpdateRemoveProperties:
		modern, legacy = "removals", "removed"
	default:
		return false
	}

	normalized := false
	if _, ok := object[modern]; !ok {
		if value, ok := object[legacy]; ok {
			object[modern] = value
			normalized = true
		}
	}
	if _, ok := object[legacy]; ok {
		delete(object, legacy)
		normalized = true
	}

	return normalized
}

func validateRequiredUpdateFields(action string, object map[string]json.RawMessage) error {
	fields := requiredUpdateFields[action]
	if len(fields) == 0 {
		return nil
	}

	for _, field := range fields {
		value, ok := object[field]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("%w: update %q requires field %q", iceberg.ErrInvalidArgument, action, field)
		}
	}

	return nil
}

func validateUpdateActionDecode(raw json.RawMessage) error {
	var actionWire struct {
		Action *string `json:"action"`
	}

	return json.Unmarshal(raw, &actionWire)
}

func (u *Updates) UnmarshalJSON(data []byte) error {
	var rawUpdates []json.RawMessage
	if err := json.Unmarshal(data, &rawUpdates); err != nil {
		return err
	}

	var updates Updates
	if len(rawUpdates) > 0 {
		updates = make(Updates, 0, len(rawUpdates))
	}
	for _, raw := range rawUpdates {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(raw, &object); err != nil {
			return err
		}

		actionValue, exactAction := object["action"]
		actionFieldCount := 0
		caseVariantAction := false
		for field := range object {
			if strings.EqualFold(field, "action") {
				actionFieldCount++
				if field != "action" {
					caseVariantAction = true
				}
			}
		}

		var action string
		if exactAction && !caseVariantAction {
			if bytes.Equal(bytes.TrimSpace(actionValue), []byte("null")) {
				if err := validateUpdateActionDecode(raw); err != nil {
					return err
				}

				return fmt.Errorf("%w: update requires field %q", iceberg.ErrInvalidArgument, "action")
			}
			if err := json.Unmarshal(actionValue, &action); err != nil {
				return err
			}
		} else {
			var baseWire struct {
				Action *string `json:"action"`
			}
			if err := json.Unmarshal(raw, &baseWire); err != nil {
				return err
			}
			if baseWire.Action == nil {
				return fmt.Errorf("%w: update requires field %q", iceberg.ErrInvalidArgument, "action")
			}
			action = *baseWire.Action
		}

		base := baseUpdate{ActionName: action}

		var upd Update
		switch base.ActionName {
		case UpdateAssignUUID:
			upd = &assignUUIDUpdate{}
		case UpdateUpgradeFormatVersion:
			upd = &upgradeFormatVersionUpdate{}
		case UpdateAddSchema:
			upd = &addSchemaUpdate{}
		case UpdateSetCurrentSchema:
			upd = &setCurrentSchemaUpdate{}
		case UpdateAddSpec:
			upd = &addPartitionSpecUpdate{}
		case UpdateSetDefaultSpec:
			upd = &setDefaultSpecUpdate{}
		case UpdateAddSortOrder:
			upd = &addSortOrderUpdate{}
		case UpdateSetDefaultSortOrder:
			upd = &setDefaultSortOrderUpdate{}
		case UpdateAddSnapshot:
			upd = &addSnapshotUpdate{}
		case UpdateSetSnapshotRef:
			upd = &setSnapshotRefUpdate{}
		case UpdateRemoveSnapshots:
			upd = &removeSnapshotsUpdate{}
		case UpdateRemoveSnapshotRef:
			upd = &removeSnapshotRefUpdate{}
		case UpdateSetLocation:
			upd = &setLocationUpdate{}
		case UpdateSetProperties:
			upd = &setPropertiesUpdate{}
		case UpdateRemoveProperties:
			upd = &removePropertiesUpdate{}
		case UpdateRemoveSpec:
			upd = &removeSpecUpdate{}
		case UpdateRemoveSchemas:
			upd = &removeSchemasUpdate{}
		case UpdateSetStatistics:
			upd = &setStatisticsUpdate{}
		case UpdateRemoveStatistics:
			upd = &removeStatisticsUpdate{}
		case UpdateSetPartitionStatistics:
			upd = &setPartitionStatisticsUpdate{}
		case UpdateRemovePartitionStatistics:
			upd = &removePartitionStatisticsUpdate{}
		case UpdateAddEncryptionKey:
			upd = &addEncryptionKeyUpdate{}
		case UpdateRemoveEncryptionKey:
			upd = &removeEncryptionKeyUpdate{}
		default:
			if err := validateUpdateActionDecode(raw); err != nil {
				return err
			}

			return fmt.Errorf("%w: unknown update action: %s", iceberg.ErrInvalidArgument, base.ActionName)
		}
		if normalizeLegacyPropertyFields(base.ActionName, object) {
			if err := validateUpdateActionDecode(raw); err != nil {
				return err
			}
			if actionFieldCount > 1 {
				for field := range object {
					if strings.EqualFold(field, "action") {
						delete(object, field)
					}
				}
				actionJSON, err := json.Marshal(base.ActionName)
				if err != nil {
					return err
				}
				object["action"] = actionJSON
			}
			normalized, err := json.Marshal(object)
			if err != nil {
				return err
			}
			raw = normalized
		}
		if err := validateRequiredUpdateFields(base.ActionName, object); err != nil {
			if actionErr := validateUpdateActionDecode(raw); actionErr != nil {
				return actionErr
			}

			return err
		}

		if err := json.Unmarshal(raw, upd); err != nil {
			return err
		}
		updates = append(updates, upd)
	}

	*u = updates

	return nil
}

// baseUpdate contains the common fields for all updates. It is used to identify the type
// of the update.
type baseUpdate struct {
	ActionName string `json:"action"`
}

func (u *baseUpdate) Action() string {
	return u.ActionName
}

func (u *baseUpdate) PostCommit(_ context.Context, _ *Table, _ *Table) error {
	return nil
}

type assignUUIDUpdate struct {
	baseUpdate
	UUID uuid.UUID `json:"uuid"`
}

// NewAssignUUIDUpdate creates a new update to assign a UUID to the table metadata.
func NewAssignUUIDUpdate(uuid uuid.UUID) *assignUUIDUpdate {
	return &assignUUIDUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateAssignUUID},
		UUID:       uuid,
	}
}

func (u *assignUUIDUpdate) Apply(builder *MetadataBuilder) error {
	return builder.SetUUID(u.UUID)
}

type upgradeFormatVersionUpdate struct {
	baseUpdate
	FormatVersion int `json:"format-version"`
}

// NewUpgradeFormatVersionUpdate creates a new update that upgrades the format version
// of the table metadata to the given formatVersion.
func NewUpgradeFormatVersionUpdate(formatVersion int) *upgradeFormatVersionUpdate {
	return &upgradeFormatVersionUpdate{
		baseUpdate:    baseUpdate{ActionName: UpdateUpgradeFormatVersion},
		FormatVersion: formatVersion,
	}
}

func (u *upgradeFormatVersionUpdate) Apply(builder *MetadataBuilder) error {
	return builder.SetFormatVersion(u.FormatVersion)
}

type addSchemaUpdate struct {
	baseUpdate
	Schema  *iceberg.Schema `json:"schema"`
	initial bool
}

// NewAddSchemaUpdate creates a new update that adds the given schema and updates the lastColumnID based on the schema.
func NewAddSchemaUpdate(schema *iceberg.Schema) *addSchemaUpdate {
	return &addSchemaUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateAddSchema},
		Schema:     schema,
	}
}

func (u *addSchemaUpdate) Apply(builder *MetadataBuilder) error {
	return builder.AddSchema(u.Schema)
}

type setCurrentSchemaUpdate struct {
	baseUpdate
	SchemaID int `json:"schema-id"`
}

// NewSetCurrentSchemaUpdate creates a new update that sets the current schema of the table
// metadata to the given schema ID.
func NewSetCurrentSchemaUpdate(id int) *setCurrentSchemaUpdate {
	return &setCurrentSchemaUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateSetCurrentSchema},
		SchemaID:   id,
	}
}

func (u *setCurrentSchemaUpdate) Apply(builder *MetadataBuilder) error {
	return builder.SetCurrentSchemaID(u.SchemaID)
}

type addPartitionSpecUpdate struct {
	baseUpdate
	// The spec arrives unbound: Apply binds it to the table's current schema,
	// so its source IDs are only required to resolve there. A dropped
	// partition field in particular is a void transform over source ID 0,
	// which BindToSchema carries across but a bound decode rejects.
	Spec    *iceberg.UnboundPartitionSpec `json:"spec"`
	initial bool
}

// NewAddPartitionSpecUpdate creates a new update that adds the given partition spec to the table
// metadata. If the initial flag is set to true, the spec is considered the initial spec of the table,
// and all other previously added specs in the metadata builder are removed.
func NewAddPartitionSpecUpdate(spec *iceberg.PartitionSpec, initial bool) *addPartitionSpecUpdate {
	upd := &addPartitionSpecUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateAddSpec},
		initial:    initial,
	}
	if spec != nil {
		upd.Spec = &iceberg.UnboundPartitionSpec{PartitionSpec: *spec}
	}

	return upd
}

func (u *addPartitionSpecUpdate) Apply(builder *MetadataBuilder) error {
	if u.Spec == nil {
		return fmt.Errorf("%w: update %q requires field %q", iceberg.ErrInvalidArgument, UpdateAddSpec, "spec")
	}

	return builder.AddPartitionSpec(&u.Spec.PartitionSpec, u.initial)
}

type setDefaultSpecUpdate struct {
	baseUpdate
	SpecID int `json:"spec-id"`
}

// NewSetDefaultSpecUpdate creates a new update that sets the default partition spec of the
// table metadata to the given spec ID.
func NewSetDefaultSpecUpdate(id int) *setDefaultSpecUpdate {
	return &setDefaultSpecUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateSetDefaultSpec},
		SpecID:     id,
	}
}

func (u *setDefaultSpecUpdate) Apply(builder *MetadataBuilder) error {
	return builder.SetDefaultSpecID(u.SpecID)
}

type addSortOrderUpdate struct {
	baseUpdate
	// Unbound so decoding accepts placeholder source IDs. Unlike the spec
	// above, none are exempt at Apply: CheckCompatibility still requires every
	// source ID to resolve in the current schema, so this only defers the error.
	SortOrder *UnboundSortOrder `json:"sort-order"`
	initial   bool
}

// NewAddSortOrderUpdate creates a new update that adds the given sort order to the table metadata.
// If the initial flag is set to true, the sort order is considered the initial sort order of the table,
// and all previously added sort orders in the metadata builder are removed.
func NewAddSortOrderUpdate(sortOrder *SortOrder) *addSortOrderUpdate {
	upd := &addSortOrderUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateAddSortOrder},
	}
	if sortOrder != nil {
		upd.SortOrder = &UnboundSortOrder{SortOrder: *sortOrder}
	}

	return upd
}

func (u *addSortOrderUpdate) Apply(builder *MetadataBuilder) error {
	if u.SortOrder == nil {
		return fmt.Errorf("%w: update %q requires field %q", iceberg.ErrInvalidArgument, UpdateAddSortOrder, "sort-order")
	}

	// AddSortOrder writes the fresh order ID into the order it is given and
	// keeps a pointer to that field, so the builder aliases this update.
	return builder.AddSortOrder(&u.SortOrder.SortOrder)
}

type setDefaultSortOrderUpdate struct {
	baseUpdate
	SortOrderID int `json:"sort-order-id"`
}

// NewSetDefaultSortOrderUpdate creates a new update that sets the default sort order of the table metadata
// to the given sort order ID.
func NewSetDefaultSortOrderUpdate(id int) *setDefaultSortOrderUpdate {
	return &setDefaultSortOrderUpdate{
		baseUpdate:  baseUpdate{ActionName: UpdateSetDefaultSortOrder},
		SortOrderID: id,
	}
}

func (u *setDefaultSortOrderUpdate) Apply(builder *MetadataBuilder) error {
	return builder.SetDefaultSortOrderID(u.SortOrderID)
}

type addSnapshotUpdate struct {
	baseUpdate
	Snapshot *Snapshot `json:"snapshot"`

	// ownManifests holds the manifests written by this producer (those
	// NOT inherited from the parent snapshot). Populated by
	// snapshotProducer.commit and used by rebuildManifestList below.
	ownManifests []iceberg.ManifestFile

	// rebuildManifestList, when non-nil, regenerates the snapshot's
	// manifest list to inherit from freshParent and combines it with
	// ownManifests. Called by doCommit on every retry attempt so that
	// each retry snapshot correctly inherits all files committed by
	// concurrent writers since the original build.
	rebuildManifestList func(ctx context.Context, freshMeta Metadata, freshParent *Snapshot, fio io.WriteFileIO, attempt int) (*Snapshot, error)

	// supersededSource, when non-nil, is the producer whose merged manifests a
	// rebuild on a prior attempt wrote and a later attempt replaced. doCommit
	// reads its accumulated paths and removes them once the commit resolves.
	// Distinct from the manifest-LIST orphans rebuildSnapshotUpdates returns:
	// these are the inner data manifests a rewrite re-merges on every retry.
	supersededSource supersededAccumulator
}

// NewAddSnapshotUpdate creates a new update that adds the given snapshot to the table metadata.
func NewAddSnapshotUpdate(snapshot *Snapshot) *addSnapshotUpdate {
	return &addSnapshotUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateAddSnapshot},
		Snapshot:   snapshot,
	}
}

// Apply records this snapshot in the metadata builder. It delegates to
// MetadataBuilder.AddSnapshotUpdate so that runtime-only fields
// (ownManifests and rebuildManifestList) are preserved on the stored update
// without reaching back into builder.updates after the fact. doCommit's retry
// loop relies on these fields to regenerate the manifest list after an OCC
// conflict.
func (u *addSnapshotUpdate) Apply(builder *MetadataBuilder) error {
	return builder.AddSnapshotUpdate(u)
}

type setSnapshotRefUpdate struct {
	baseUpdate
	RefName            string  `json:"ref-name"`
	RefType            RefType `json:"type"`
	SnapshotID         int64   `json:"snapshot-id"`
	MaxRefAgeMs        int64   `json:"max-ref-age-ms,omitempty"`
	MaxSnapshotAgeMs   int64   `json:"max-snapshot-age-ms,omitempty"`
	MinSnapshotsToKeep int     `json:"min-snapshots-to-keep,omitempty"`
}

// NewSetSnapshotRefUpdate creates a new update that sets the given snapshot reference
// as the current snapshot of the table metadata. MaxRefAgeMs, MaxSnapshotAgeMs,
// and MinSnapshotsToKeep are optional, and any non-positive values are ignored.
func NewSetSnapshotRefUpdate(
	name string,
	snapshotID int64,
	refType RefType,
	maxRefAgeMs, maxSnapshotAgeMs int64,
	minSnapshotsToKeep int,
) *setSnapshotRefUpdate {
	return &setSnapshotRefUpdate{
		baseUpdate:         baseUpdate{ActionName: UpdateSetSnapshotRef},
		RefName:            name,
		RefType:            refType,
		SnapshotID:         snapshotID,
		MaxRefAgeMs:        maxRefAgeMs,
		MaxSnapshotAgeMs:   maxSnapshotAgeMs,
		MinSnapshotsToKeep: minSnapshotsToKeep,
	}
}

func (u *setSnapshotRefUpdate) Apply(builder *MetadataBuilder) error {
	opts := []setSnapshotRefOption{}
	if u.MaxRefAgeMs > 0 {
		opts = append(opts, WithMaxRefAgeMs(u.MaxRefAgeMs))
	}
	if u.MaxSnapshotAgeMs > 0 {
		opts = append(opts, WithMaxSnapshotAgeMs(u.MaxSnapshotAgeMs))
	}
	if u.MinSnapshotsToKeep > 0 {
		opts = append(opts, WithMinSnapshotsToKeep(u.MinSnapshotsToKeep))
	}

	return builder.SetSnapshotRef(
		u.RefName,
		u.SnapshotID,
		u.RefType,
		opts...,
	)
}

type setLocationUpdate struct {
	baseUpdate
	Location string `json:"location"`
}

// NewSetLocationUpdate creates a new update that sets the location of the table metadata.
func NewSetLocationUpdate(loc string) Update {
	return &setLocationUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateSetLocation},
		Location:   loc,
	}
}

func (u *setLocationUpdate) Apply(builder *MetadataBuilder) error {
	return builder.SetLoc(u.Location)
}

type setPropertiesUpdate struct {
	baseUpdate
	Updates iceberg.Properties `json:"updates"`
}

// NewSetPropertiesUpdate creates a new update that sets the given properties in the
// table metadata.
func NewSetPropertiesUpdate(updates iceberg.Properties) *setPropertiesUpdate {
	return &setPropertiesUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateSetProperties},
		Updates:    updates,
	}
}

func (u *setPropertiesUpdate) Apply(builder *MetadataBuilder) error {
	return builder.SetProperties(u.Updates)
}

type removePropertiesUpdate struct {
	baseUpdate
	Removals []string `json:"removals"`
}

// NewRemovePropertiesUpdate creates a new update that removes properties from the table metadata.
// The properties are identified by their names, and if a property with the given name does not exist,
// it is ignored.
func NewRemovePropertiesUpdate(removals []string) *removePropertiesUpdate {
	return &removePropertiesUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateRemoveProperties},
		Removals:   removals,
	}
}

func (u *removePropertiesUpdate) Apply(builder *MetadataBuilder) error {
	return builder.RemoveProperties(u.Removals)
}

type removeSnapshotsUpdate struct {
	baseUpdate
	SnapshotIDs []int64 `json:"snapshot-ids"`
	postCommit  bool
}

// NewRemoveSnapshotsUpdate creates a new update that removes all snapshots from
// the table metadata with the given snapshot IDs.
func NewRemoveSnapshotsUpdate(ids []int64, postCommit bool) *removeSnapshotsUpdate {
	return &removeSnapshotsUpdate{
		baseUpdate:  baseUpdate{ActionName: UpdateRemoveSnapshots},
		SnapshotIDs: ids,
		postCommit:  postCommit,
	}
}

func (u *removeSnapshotsUpdate) Apply(builder *MetadataBuilder) error {
	return builder.RemoveSnapshots(u.SnapshotIDs, u.postCommit)
}

func (u *removeSnapshotsUpdate) PostCommit(ctx context.Context, preTable *Table, postTable *Table) error {
	if !u.postCommit || postTable == nil || !isGCEnabled(postTable.Properties()) {
		return nil
	}

	prefs, err := preTable.FS(ctx)
	if err != nil {
		return err
	}

	filesToDelete := make(map[string]struct{})

	for _, snapId := range u.SnapshotIDs {
		snap := preTable.Metadata().SnapshotByID(snapId)
		if snap == nil {
			return errors.New("snapshot should never be nil")
		}

		filesToDelete[snap.ManifestList] = struct{}{}
	}

	expiredIDs := make(map[int64]struct{}, len(u.SnapshotIDs))
	for _, id := range u.SnapshotIDs {
		expiredIDs[id] = struct{}{}
	}

	for sf := range preTable.Metadata().Statistics() {
		if _, ok := expiredIDs[sf.SnapshotID]; ok && sf.StatisticsPath != "" {
			filesToDelete[sf.StatisticsPath] = struct{}{}
		}
	}

	for psf := range preTable.Metadata().PartitionStatistics() {
		if _, ok := expiredIDs[psf.SnapshotID]; ok && psf.StatisticsPath != "" {
			filesToDelete[psf.StatisticsPath] = struct{}{}
		}
	}

	// Preload retained manifest lists once so the live-data-file pass below can walk them without re-fetching.
	retainedSnapshots := postTable.Metadata().Snapshots()
	retainedManifests := make(map[string]struct{})
	retainedSnapshotManifests := make([]iceberg.ManifestFile, 0, len(retainedSnapshots))
	for _, snap := range retainedSnapshots {
		mans, err := snap.Manifests(prefs)
		if err != nil {
			return err
		}
		for _, man := range mans {
			manPath := man.FilePath()
			if _, ok := retainedManifests[manPath]; !ok {
				retainedManifests[manPath] = struct{}{}
				retainedSnapshotManifests = append(retainedSnapshotManifests, man)
			}
		}
	}

	// Open each orphaned manifest at most once: skip manifests that
	// retained snapshots still reference, and dedupe across expired
	// snapshots that share manifests by reference.
	visitedManifests := make(map[string]struct{})

	for _, snapId := range u.SnapshotIDs {
		snap := preTable.SnapshotByID(snapId)
		if snap == nil {
			return fmt.Errorf("missing snapshot %d", snapId)
		}

		mans, err := snap.Manifests(prefs)
		if err != nil {
			return err
		}

		for _, man := range mans {
			manPath := man.FilePath()

			if _, ok := retainedManifests[manPath]; ok {
				continue
			}
			if _, ok := visitedManifests[manPath]; ok {
				continue
			}
			visitedManifests[manPath] = struct{}{}

			filesToDelete[manPath] = struct{}{}

			for entry, err := range man.Entries(prefs, false) {
				if err != nil {
					return err
				}
				filesToDelete[entry.DataFile().FilePath()] = struct{}{}
			}
		}
	}

	// Keep files still referenced (non-DELETED) by retained manifests,
	// including data files carried forward as EXISTING by manifest merges.
	if len(filesToDelete) > 0 {
		for _, man := range retainedSnapshotManifests {
			for entry, err := range man.Entries(prefs, false) {
				if err != nil {
					return err
				}
				if entry.Status() != iceberg.EntryStatusDELETED {
					delete(filesToDelete, entry.DataFile().FilePath())
				}
			}
		}
	}

	for sf := range postTable.Metadata().Statistics() {
		delete(filesToDelete, sf.StatisticsPath)
	}

	for psf := range postTable.Metadata().PartitionStatistics() {
		delete(filesToDelete, psf.StatisticsPath)
	}

	if len(filesToDelete) == 0 {
		return nil
	}

	paths := slices.Collect(maps.Keys(filesToDelete))

	// Try bulk delete first; on failure fall through to per-file delete.
	if bulk, ok := prefs.(io.BulkRemovableIO); ok {
		deleted, err := bulk.DeleteFiles(ctx, paths)
		if err == nil {
			return nil
		}

		// Remove successfully deleted files so the fallback loop
		// only retries what the bulk call missed.
		deletedSet := make(map[string]struct{}, len(deleted))
		for _, d := range deleted {
			deletedSet[d] = struct{}{}
		}

		paths = slices.DeleteFunc(paths, func(s string) bool {
			_, ok := deletedSet[s]

			return ok
		})
	}

	var res error

	for _, f := range paths {
		if err := prefs.Remove(f); err != nil {
			res = errors.Join(res, err)
		}
	}

	return res
}

type removeSnapshotRefUpdate struct {
	baseUpdate
	RefName string `json:"ref-name"`
}

// NewRemoveSnapshotRefUpdate creates a new update that removes a snapshot reference
// from the table metadata.
func NewRemoveSnapshotRefUpdate(ref string) *removeSnapshotRefUpdate {
	return &removeSnapshotRefUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateRemoveSnapshotRef},
		RefName:    ref,
	}
}

func (u *removeSnapshotRefUpdate) Apply(builder *MetadataBuilder) error {
	return builder.RemoveSnapshotRef(u.RefName)
}

type removeSpecUpdate struct {
	baseUpdate
	SpecIds []int `json:"spec-ids"`
}

// NewRemoveSpecUpdate creates a new Update that removes a list of partition specs
// from the table metadata.
func NewRemoveSpecUpdate(specIds []int) *removeSpecUpdate {
	return &removeSpecUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateRemoveSpec},
		SpecIds:    specIds,
	}
}

func (u *removeSpecUpdate) Apply(builder *MetadataBuilder) error {
	return builder.RemovePartitionSpecs(u.SpecIds)
}

type removeSchemasUpdate struct {
	baseUpdate
	SchemaIDs []int `json:"schema-ids"`
}

// NewRemoveSchemasUpdate creates a new Update that removes a list of schemas from
// the table metadata.
func NewRemoveSchemasUpdate(schemaIds []int) *removeSchemasUpdate {
	return &removeSchemasUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateRemoveSchemas},
		SchemaIDs:  schemaIds,
	}
}

func (u *removeSchemasUpdate) Apply(builder *MetadataBuilder) error {
	return builder.RemoveSchemas(u.SchemaIDs)
}

type setStatisticsUpdate struct {
	baseUpdate
	SnapshotID int64          `json:"snapshot-id"`
	Statistics StatisticsFile `json:"statistics"`
}

// NewSetStatisticsUpdate creates a new Update that adds or replaces the statistics file
// for the given snapshot ID in the table metadata.
func NewSetStatisticsUpdate(stats StatisticsFile) *setStatisticsUpdate {
	return &setStatisticsUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateSetStatistics},
		SnapshotID: stats.SnapshotID,
		Statistics: stats,
	}
}

func (u *setStatisticsUpdate) Apply(builder *MetadataBuilder) error {
	return builder.SetStatistics(u.Statistics)
}

type removeStatisticsUpdate struct {
	baseUpdate
	SnapshotID int64 `json:"snapshot-id"`
}

// NewRemoveStatisticsUpdate creates a new Update that removes the statistics file
// for the given snapshot ID from the table metadata.
func NewRemoveStatisticsUpdate(snapshotID int64) *removeStatisticsUpdate {
	return &removeStatisticsUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateRemoveStatistics},
		SnapshotID: snapshotID,
	}
}

func (u *removeStatisticsUpdate) Apply(builder *MetadataBuilder) error {
	return builder.RemoveStatistics(u.SnapshotID)
}

type setPartitionStatisticsUpdate struct {
	baseUpdate
	PartitionStatistics PartitionStatisticsFile `json:"partition-statistics"`
}

// NewSetPartitionStatisticsUpdate creates a new Update that adds or replaces the partition
// statistics file for the given snapshot ID in the table metadata.
func NewSetPartitionStatisticsUpdate(stats PartitionStatisticsFile) *setPartitionStatisticsUpdate {
	return &setPartitionStatisticsUpdate{
		baseUpdate:          baseUpdate{ActionName: UpdateSetPartitionStatistics},
		PartitionStatistics: stats,
	}
}

func (u *setPartitionStatisticsUpdate) Apply(builder *MetadataBuilder) error {
	return builder.SetPartitionStatistics(u.PartitionStatistics)
}

type removePartitionStatisticsUpdate struct {
	baseUpdate
	SnapshotID int64 `json:"snapshot-id"`
}

// NewRemovePartitionStatisticsUpdate creates a new Update that removes the partition statistics
// file for the given snapshot ID from the table metadata.
func NewRemovePartitionStatisticsUpdate(snapshotID int64) *removePartitionStatisticsUpdate {
	return &removePartitionStatisticsUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateRemovePartitionStatistics},
		SnapshotID: snapshotID,
	}
}

func (u *removePartitionStatisticsUpdate) Apply(builder *MetadataBuilder) error {
	return builder.RemovePartitionStatistics(u.SnapshotID)
}

type addEncryptionKeyUpdate struct {
	baseUpdate
	EncryptionKey EncryptionKey `json:"encryption-key"`
}

// NewAddEncryptionKeyUpdate creates a new Update that adds or replaces an encryption key
// (indexed by its key-id) in the table metadata.
func NewAddEncryptionKeyUpdate(key EncryptionKey) *addEncryptionKeyUpdate {
	return &addEncryptionKeyUpdate{
		baseUpdate:    baseUpdate{ActionName: UpdateAddEncryptionKey},
		EncryptionKey: key,
	}
}

func (u *addEncryptionKeyUpdate) Apply(builder *MetadataBuilder) error {
	return builder.AddEncryptionKey(u.EncryptionKey)
}

type removeEncryptionKeyUpdate struct {
	baseUpdate
	KeyID string `json:"key-id"`
}

// NewRemoveEncryptionKeyUpdate creates a new Update that removes the encryption key
// with the given key-id from the table metadata.
func NewRemoveEncryptionKeyUpdate(keyID string) *removeEncryptionKeyUpdate {
	return &removeEncryptionKeyUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateRemoveEncryptionKey},
		KeyID:      keyID,
	}
}

func (u *removeEncryptionKeyUpdate) Apply(builder *MetadataBuilder) error {
	return builder.RemoveEncryptionKey(u.KeyID)
}
