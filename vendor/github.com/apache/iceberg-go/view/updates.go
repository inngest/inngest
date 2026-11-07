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

package view

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/apache/iceberg-go"
	"github.com/google/uuid"
)

// These are the various update actions defined in the iceberg spec
const (
	UpdateAddSchema      = "add-schema"
	UpdateAddViewVersion = "add-view-version"

	UpdateAssignUUID = "assign-uuid"

	UpdateRemoveProperties = "remove-properties"

	UpdateSetCurrentViewVersion = "set-current-view-version"
	UpdateSetLocation           = "set-location"
	UpdateSetProperties         = "set-properties"

	UpdateUpgradeFormatVersion = "upgrade-format-version"
)

// Update represents a change to a view's metadata.
type Update interface {
	// Action returns the name of the action that the update represents.
	Action() string
	// Apply applies the update to the given view metadata builder.
	Apply(viewBuilder *MetadataBuilder) error
}

type Updates []Update

// updateForAction returns an instance of an update struct corresponding to a given action
func updateForAction(action string) (Update, error) {
	var upd Update
	switch action {
	case UpdateAssignUUID:
		upd = &assignUUIDUpdate{}
	case UpdateUpgradeFormatVersion:
		upd = &upgradeFormatVersionUpdate{}
	case UpdateAddSchema:
		upd = &addSchemaUpdate{}
	case UpdateSetLocation:
		upd = &setLocationUpdate{}
	case UpdateSetProperties:
		upd = &setPropertiesUpdate{}
	case UpdateRemoveProperties:
		upd = &removePropertiesUpdate{}
	case UpdateAddViewVersion:
		upd = &addViewVersionUpdate{}
	case UpdateSetCurrentViewVersion:
		upd = &setCurrentViewVersionUpdate{}
	default:
		return nil, fmt.Errorf("unknown update action: %s", action)
	}

	return upd, nil
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
		var base baseUpdate
		if err := json.Unmarshal(raw, &base); err != nil {
			return err
		}

		update, err := updateForAction(base.ActionName)
		if err != nil {
			return err
		}

		if err := json.Unmarshal(raw, update); err != nil {
			return err
		}
		updates = append(updates, update)
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
	builder.SetUUID(u.UUID)

	return builder.Err()
}

type upgradeFormatVersionUpdate struct {
	baseUpdate
	FormatVersion int `json:"format-version"`
}

// NewUpgradeFormatVersionUpdate creates a new update that upgrades the format version
// of the view metadata to the given formatVersion.
func NewUpgradeFormatVersionUpdate(formatVersion int) *upgradeFormatVersionUpdate {
	return &upgradeFormatVersionUpdate{
		baseUpdate:    baseUpdate{ActionName: UpdateUpgradeFormatVersion},
		FormatVersion: formatVersion,
	}
}

func (u *upgradeFormatVersionUpdate) Apply(builder *MetadataBuilder) error {
	builder.SetFormatVersion(u.FormatVersion)

	return builder.Err()
}

type addSchemaUpdate struct {
	baseUpdate
	Schema *iceberg.Schema `json:"schema"`
}

// NewAddSchemaUpdate creates a new update that adds the given schema and updates the lastColumnID based on the schema.
func NewAddSchemaUpdate(schema *iceberg.Schema) *addSchemaUpdate {
	return &addSchemaUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateAddSchema},
		Schema:     schema,
	}
}

func (u *addSchemaUpdate) Apply(builder *MetadataBuilder) error {
	builder.AddSchema(u.Schema)

	return builder.Err()
}

type setLocationUpdate struct {
	baseUpdate
	Location string `json:"location"`
}

// NewSetLocationUpdate creates a new update that sets the location of the view metadata.
func NewSetLocationUpdate(loc string) *setLocationUpdate {
	return &setLocationUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateSetLocation},
		Location:   loc,
	}
}

func (u *setLocationUpdate) Apply(builder *MetadataBuilder) error {
	builder.SetLoc(u.Location)

	return builder.Err()
}

type setPropertiesUpdate struct {
	baseUpdate
	Updates iceberg.Properties `json:"updates"`
}

// NewSetPropertiesUpdate creates a new update that sets the given properties in the
// view metadata.
func NewSetPropertiesUpdate(updates iceberg.Properties) *setPropertiesUpdate {
	return &setPropertiesUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateSetProperties},
		Updates:    maps.Clone(updates),
	}
}

func (u *setPropertiesUpdate) Apply(builder *MetadataBuilder) error {
	builder.SetProperties(u.Updates)

	return builder.Err()
}

type removePropertiesUpdate struct {
	baseUpdate
	Removals []string `json:"removals"`
}

// NewRemovePropertiesUpdate creates a new update that removes properties from the view metadata.
// The properties are identified by their names, and if a property with the given name does not exist,
// it is ignored.
func NewRemovePropertiesUpdate(removals []string) *removePropertiesUpdate {
	return &removePropertiesUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateRemoveProperties},
		Removals:   slices.Clone(removals),
	}
}

func (u *removePropertiesUpdate) Apply(builder *MetadataBuilder) error {
	builder.RemoveProperties(u.Removals)

	return builder.Err()
}

type addViewVersionUpdate struct {
	baseUpdate
	Version *Version `json:"view-version"`
}

// NewAddViewVersionUpdate creates a new ViewUpdate that adds a version to a view
func NewAddViewVersionUpdate(version *Version) *addViewVersionUpdate {
	return &addViewVersionUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateAddViewVersion},
		Version:    version,
	}
}

func (u *addViewVersionUpdate) Apply(builder *MetadataBuilder) error {
	builder.AddVersion(u.Version)

	return builder.Err()
}

type setCurrentViewVersionUpdate struct {
	baseUpdate
	VersionID int64 `json:"view-version-id"`
}

// NewSetCurrentVersionUpdate creates a new ViewUpdate that sets the current version of a view
// to the given version ID.
func NewSetCurrentVersionUpdate(id int64) *setCurrentViewVersionUpdate {
	return &setCurrentViewVersionUpdate{
		baseUpdate: baseUpdate{ActionName: UpdateSetCurrentViewVersion},
		VersionID:  id,
	}
}

func (u *setCurrentViewVersionUpdate) Apply(builder *MetadataBuilder) error {
	builder.SetCurrentVersionID(u.VersionID)

	return builder.Err()
}
