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
	"errors"
	"fmt"

	"github.com/apache/iceberg-go"
)

// UpdateSpec implements a builder for evolving a table's partition specification.
//
// It accumulates a sequence of partition spec update operations (e.g., AddField, RemoveField, RenameField)
// which are applied during BuildUpdates.
//
// Use the builder methods to chain operations, and call BuildUpdates to apply them and produce the
// final set of partition fields and update requirements, or call Commit to apply the updates in the transaction.
type UpdateSpec struct {
	operations []updateSpecOp

	txn *Transaction
	// meta is an immutable snapshot of the transaction's staged metadata, frozen
	// when this UpdateSpec is constructed. Schema and partition-spec resolution
	// read from it so partitioning can reference columns/specs staged earlier in
	// the transaction; concurrency assertions are handled separately (BuildUpdates).
	// schema is the single defensive copy of the current staged schema reused by
	// all schema-dependent operations in this update.
	meta                  Metadata
	schema                *iceberg.Schema
	historicalFields      map[transformKey][]iceberg.PartitionField
	err                   error
	nameToField           map[string]iceberg.PartitionField
	nameToAddedField      map[string]iceberg.PartitionField
	transformToField      map[transformKey]iceberg.PartitionField
	transformToAddedField map[transformKey]iceberg.PartitionField
	renames               map[string]string
	caseSensitive         bool
	adds                  []iceberg.PartitionField
	deletes               map[int]bool
	lastAssignedFieldId   int
}
type updateSpecOp func() error

type transformKey struct {
	SourceId  int
	Transform string
}

// NewUpdateSpec starts a partition spec evolution.
//
// If the table's current spec contains an unknown transform, the returned
// UpdateSpec carries an error and no evolution is possible -- including
// renaming or dropping an unrelated field. That matches Java's
// BaseUpdatePartitionSpec, which rejects at construction rather than at commit.
// Loading such a table is still fine; only evolving its spec is blocked.
func NewUpdateSpec(t *Transaction, caseSensitive bool) *UpdateSpec {
	us := &UpdateSpec{
		txn:                   t,
		nameToField:           make(map[string]iceberg.PartitionField),
		nameToAddedField:      make(map[string]iceberg.PartitionField),
		transformToField:      make(map[transformKey]iceberg.PartitionField),
		transformToAddedField: make(map[transformKey]iceberg.PartitionField),
		renames:               make(map[string]string),
		caseSensitive:         caseSensitive,
		adds:                  make([]iceberg.PartitionField, 0),
		deletes:               make(map[int]bool),
	}

	if t == nil {
		us.err = fmt.Errorf("%w: transaction is nil", ErrInvalidMetadata)

		return us
	}
	// Resolve schema and partition state from the transaction's staged metadata
	// rather than the frozen table snapshot captured when the transaction began,
	// so that columns and specs added earlier in the same transaction are
	// visible immediately.
	meta, err := t.txnMeta()
	if err != nil {
		us.err = err

		return us
	}
	stagedMeta, err := meta.Build() // immutable snapshot
	if err != nil {
		us.err = err

		return us
	}
	us.meta = stagedMeta

	transformToField := make(map[transformKey]iceberg.PartitionField)
	nameToField := make(map[string]iceberg.PartitionField)
	partitionSpec := us.meta.PartitionSpec()
	for _, partitionField := range partitionSpec.Fields() {
		if _, ok := partitionField.Transform.(iceberg.UnknownTransform); ok {
			us.err = fmt.Errorf("%w: cannot update partition spec with unknown transform: %s",
				iceberg.ErrInvalidTransform, partitionField.Transform)

			return us
		}
		transformToField[transformKey{
			SourceId:  partitionField.SourceID(),
			Transform: partitionField.Transform.String(),
		}] = partitionField
		nameToField[partitionField.Name] = partitionField
	}
	us.schema = stagedMeta.CurrentSchema()
	us.nameToField = nameToField
	us.transformToField = transformToField
	us.lastAssignedFieldId = partitionFieldIDFloor(us.meta.LastPartitionSpecID(), us.meta.PartitionSpecs())

	return us
}

func (us *UpdateSpec) AddField(sourceColName string, transform iceberg.Transform, partitionFieldName string) *UpdateSpec {
	us.operations = append(us.operations, us.addField(sourceColName, transform, partitionFieldName))

	return us
}

func (us *UpdateSpec) AddIdentity(sourceColName string) *UpdateSpec {
	return us.AddField(sourceColName, iceberg.IdentityTransform{}, "")
}

func (us *UpdateSpec) RemoveField(name string) *UpdateSpec {
	us.operations = append(us.operations, us.removeField(name))

	return us
}

func (us *UpdateSpec) RenameField(name string, newName string) *UpdateSpec {
	us.operations = append(us.operations, us.renameField(name, newName))

	return us
}

func (us *UpdateSpec) BuildUpdates() ([]Update, []Requirement, error) {
	if us.err != nil {
		return nil, nil, us.err
	}

	for _, op := range us.operations {
		if err := op(); err != nil {
			return nil, nil, err
		}
	}

	newSpec, err := us.Apply()
	if err != nil {
		return nil, nil, err
	}
	updates := make([]Update, 0)
	requirements := make([]Requirement, 0)

	if us.meta.DefaultPartitionSpec() != newSpec.ID() {
		if us.isNewPartitionSpec(newSpec.ID()) {
			updates = append(updates, NewAddPartitionSpecUpdate(&newSpec, false))
			updates = append(updates, NewSetDefaultSpecUpdate(-1))
		} else {
			updates = append(updates, NewSetDefaultSpecUpdate(newSpec.ID()))
		}
		// This concurrency assertion must describe the base (pre-transaction)
		// catalog state, not the staged/advancing one: every chained
		// UpdateSpec.Commit() in the transaction then asserts the same value,
		// so they collapse to one via the ordinary semantic-key dedupe instead
		// of reaching the catalog as several contradictory values.
		requiredLastAssignedPartitionID := us.txn.tbl.Metadata().LastPartitionSpecID()
		if requiredLastAssignedPartitionID == nil {
			// Mirror the constructor's guard: an unpartitioned table may not
			// have a last-assigned partition id yet.
			base := iceberg.PartitionDataIDStart - 1
			requiredLastAssignedPartitionID = &base
		}
		requirements = append(requirements, AssertLastAssignedPartitionID(*requiredLastAssignedPartitionID))
		// Java UpdateRequirements.forUpdateTable registers
		// AssertDefaultSpecID on SetDefaultPartitionSpec: two racing
		// remove-only spec evolutions (which assign no new field ids)
		// can both pass last-assigned-partition-id, so the default
		// spec id itself must be pinned.
		requirements = append(requirements, AssertDefaultSpecID(us.txn.tbl.Metadata().DefaultPartitionSpec()))
	}

	return updates, requirements, nil
}

func (us *UpdateSpec) Apply() (iceberg.PartitionSpec, error) {
	if us.err != nil {
		return iceberg.PartitionSpec{}, us.err
	}

	spec := us.meta.PartitionSpec()
	specFields := spec.Fields()
	capacity := spec.NumFields() + len(us.adds)
	partitionFields := make([]iceberg.PartitionField, 0, capacity)
	partitionNames := make(map[string]bool, capacity)
	for _, field := range specFields {
		var newField iceberg.PartitionField
		var err error
		if _, deleted := us.deletes[field.FieldID]; !deleted {
			if rename, renamed := us.renames[field.Name]; renamed {
				newField, err = us.addNewField(us.schema, field.SourceID(), field.FieldID, rename, field.Transform, partitionNames)
			} else {
				newField, err = us.addNewField(us.schema, field.SourceID(), field.FieldID, field.Name, field.Transform, partitionNames)
			}
			if err != nil {
				return iceberg.PartitionSpec{}, err
			}
			partitionFields = append(partitionFields, newField)
		} else if us.meta.Version() == 1 {
			if rename, renamed := us.renames[field.Name]; renamed {
				newField, err = us.addNewField(us.schema, field.SourceID(), field.FieldID, rename, iceberg.VoidTransform{}, partitionNames)
			} else {
				newField, err = us.addNewField(us.schema, field.SourceID(), field.FieldID, field.Name, iceberg.VoidTransform{}, partitionNames)
			}
			if err != nil {
				return iceberg.PartitionSpec{}, err
			}
			partitionFields = append(partitionFields, newField)
		}
	}

	partitionFields = append(partitionFields, us.adds...)
	if err := validateNoRedundantTimeFields(partitionFields); err != nil {
		return iceberg.PartitionSpec{}, err
	}
	candidate := iceberg.NewPartitionSpec(partitionFields...)
	newSpec, err := candidate.BindToSchema(us.schema, nil, nil)
	if err != nil {
		return iceberg.PartitionSpec{}, err
	}
	newSpecId := iceberg.InitialPartitionSpecID
	for _, spec = range us.meta.PartitionSpecs() {
		if newSpec.CompatibleWith(&spec) {
			newSpecId = spec.ID()

			break
		} else if newSpecId <= spec.ID() {
			newSpecId = spec.ID() + 1
		}
	}

	return iceberg.NewPartitionSpecID(newSpecId, partitionFields...), nil
}

// validateNoRedundantTimeFields rejects hour(ts) alongside day(ts).
// Taking the assembled set is load bearing: only it knows which fields survive
// the deletes, so AddField(month) -> RemoveField(year) works in either order.
//
// So a redundancy inherited from the current spec blocks every update, even
// unrelated ones, until it is removed here: the update authors a whole new
// spec, and passing the old fields through would carry the redundancy into it.
// Loading such metadata still works, under the lenient equality-only rule.
func validateNoRedundantTimeFields(fields []iceberg.PartitionField) error {
	timeFields := make(map[int]iceberg.PartitionField, len(fields))
	for _, field := range fields {
		// Void is not a TimeTransform, so a v1 tombstone left by a removed
		// year(ts) does not block adding month(ts) on the same column.
		if _, isTime := field.Transform.(iceberg.TimeTransform); !isTime {
			continue
		}
		source := field.SourceID()
		if existing, exists := timeFields[source]; exists {
			return fmt.Errorf("%w: redundant partition field for source ID %d: %s (%s) conflicts with %s (%s); remove one of them in this update",
				iceberg.ErrInvalidPartitionSpec, source,
				field.Name, field.Transform, existing.Name, existing.Transform)
		}
		timeFields[source] = field
	}

	return nil
}

func (us *UpdateSpec) Commit() error {
	if us.err != nil {
		return us.err
	}

	updates, requirements, err := us.BuildUpdates()
	if err != nil {
		return err
	}

	if len(updates) == 0 {
		return nil
	}

	return us.txn.apply(updates, requirements)
}

func (us *UpdateSpec) addField(sourceColName string, transform iceberg.Transform, partitionFieldName string) updateSpecOp {
	return func() error {
		// Finds the column in the schema and binds it with case sensitivity.
		ref := iceberg.Reference(sourceColName)
		boundTerm, err := ref.Bind(us.schema, us.caseSensitive)
		if err != nil {
			return err
		}

		// Validate the transform
		if _, ok := transform.(iceberg.UnknownTransform); ok {
			return fmt.Errorf("%w: cannot add partition field with unknown transform: %s", iceberg.ErrInvalidTransform, transform)
		}
		outputType := boundTerm.Type()
		if !transform.CanTransform(outputType) {
			return fmt.Errorf("%s cannot transform %s values from %s", transform.String(), outputType.String(), boundTerm.Ref().Field().Name)
		}

		// Check for duplicate partition on same source
		key := transformKey{
			SourceId:  boundTerm.Ref().Field().ID,
			Transform: transform.String(),
		}
		existingPartitionField, exists := us.transformToField[key]

		if exists && transform.Equals(existingPartitionField.Transform) {
			if _, deleted := us.deletes[existingPartitionField.FieldID]; deleted {
				return us.rewriteDeleteAndAddField(existingPartitionField, partitionFieldName)
			}
		}

		if exists && us.isDuplicatePartition(transform, existingPartitionField) {
			return fmt.Errorf("duplicate partition field for %s=%v, %v already exists", ref.String(), ref, existingPartitionField)
		}

		// Check if this transform was already added
		added, exists := us.transformToAddedField[key]
		if exists {
			return fmt.Errorf("already added partition: %s ", added.Name)
		}

		// Create the new partition field and Check for name collisions
		// with existing fields
		newField, err := us.partitionField(key, partitionFieldName)
		if err != nil {
			return err
		}
		if _, exists = us.nameToAddedField[newField.Name]; exists {
			return fmt.Errorf("already added partition field with name: %s", newField.Name)
		}

		// Time conflicts are not checked here: they depend on which existing
		// fields survive this update's deletes, so validateNoRedundantTimeFields
		// checks the set Apply assembles once every operation has run.
		us.transformToAddedField[key] = newField

		// If name matches an existing field, rename it if it's VOID transform
		existingPartitionField, exists = us.nameToField[newField.Name]
		if _, inDelete := us.deletes[existingPartitionField.FieldID]; exists && !inDelete {
			if _, isVoidTransform := existingPartitionField.Transform.(iceberg.VoidTransform); isVoidTransform {
				if err := us.renameField(
					existingPartitionField.Name,
					fmt.Sprintf("%s_%d", existingPartitionField.Name, existingPartitionField.FieldID),
				)(); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("cannot add duplicate partition field name: %s", existingPartitionField.Name)
			}
		}

		// Register the new field
		us.nameToAddedField[newField.Name] = newField
		us.adds = append(us.adds, newField)

		return nil
	}
}

func (us *UpdateSpec) removeField(name string) updateSpecOp {
	return func() error {
		if _, added := us.nameToAddedField[name]; added {
			return fmt.Errorf("cannot remove newly added field %s", name)
		}
		if _, renamed := us.renames[name]; renamed {
			return fmt.Errorf("cannot rename and delete field %s", name)
		}
		field, exists := us.nameToField[name]
		if !exists {
			return fmt.Errorf("cannot find partition field %s", name)
		}
		us.deletes[field.FieldID] = true

		return nil
	}
}

// rewriteDeleteAndAddField restores a field removed earlier in this same
// update, keeping its permanent ID and renaming it if a different name is
// requested.
func (us *UpdateSpec) rewriteDeleteAndAddField(existing iceberg.PartitionField, name string) error {
	delete(us.deletes, existing.FieldID)
	if name == "" || existing.Name == name {
		return nil
	}

	return us.renameField(existing.Name, name)()
}

func (us *UpdateSpec) historicalPartitionFields(key transformKey) []iceberg.PartitionField {
	if us.historicalFields == nil {
		us.historicalFields = make(map[transformKey][]iceberg.PartitionField)
		for _, spec := range us.meta.PartitionSpecs() {
			for _, field := range spec.Fields() {
				historicalKey := transformKey{
					SourceId:  field.SourceID(),
					Transform: field.Transform.String(),
				}
				us.historicalFields[historicalKey] = append(us.historicalFields[historicalKey], field)
			}
		}
	}

	return us.historicalFields[key]
}

func (us *UpdateSpec) renameField(name string, newName string) updateSpecOp {
	return func() error {
		existingField, exists := us.nameToField[newName]
		_, isVoidTransform := existingField.Transform.(iceberg.VoidTransform)
		if exists && isVoidTransform {
			return us.renameField(
				name,
				fmt.Sprintf("%s_%d", name, existingField.FieldID),
			)()
		}
		if _, added := us.nameToAddedField[name]; added {
			return errors.New("cannot rename recently added partitions")
		}

		field, exists := us.nameToField[name]
		if !exists {
			return fmt.Errorf("cannot find partition field: %s", name)
		}
		if _, deleted := us.deletes[field.FieldID]; deleted {
			return fmt.Errorf("cannot delete and rename partition field: %s", name)
		}
		us.renames[name] = newName

		return nil
	}
}

func (us *UpdateSpec) partitionField(key transformKey, name string) (iceberg.PartitionField, error) {
	transform, err := iceberg.ParseTransform(key.Transform)
	if err != nil {
		return iceberg.PartitionField{}, fmt.Errorf("%w: invalid partition transform %q: %w",
			iceberg.ErrInvalidArgument, key.Transform, err)
	}

	// Reuse applies to format v2+ (v1 has no permanent field-ID contract) and
	// resurrects fields removed in an earlier committed update; same-update
	// remove/re-add is handled ahead of this call by rewriteDeleteAndAddField.
	if us.meta.Version() >= 2 {
		// PartitionSpecs() is ordered by ascending spec ID, so the cached slice
		// keeps the lowest-spec-ID match first when a field was renamed across
		// specs. The match's own name is returned, which for the no-name case may
		// be an older name than the current schema uses; this precedence is
		// deterministic and preserves the original (permanent) field ID.
		for _, field := range us.historicalPartitionFields(key) {
			// The cache key uses Transform.String(), which re-normalizes any
			// non-canonical on-disk text (e.g. "bucket[016]" -> "bucket[16]").
			// This distinguishes parameterized transforms (bucket[16] vs
			// bucket[8]) without a structural comparison.
			// Reuse the historical field's ID when no explicit name is
			// requested (match on source + transform alone) or when the
			// requested name matches.
			if len(name) == 0 || field.Name == name {
				return iceberg.PartitionField{
					SourceIDs: []int{key.SourceId},
					FieldID:   field.FieldID,
					Name:      field.Name,
					Transform: field.Transform,
				}, nil
			}
		}
	}
	newFieldId := us.newFieldId()
	if name == "" {
		tmpField := iceberg.PartitionField{
			SourceIDs: []int{key.SourceId},
			FieldID:   newFieldId,
			Name:      "",
			Transform: transform,
		}
		var err error
		name, err = iceberg.GeneratePartitionFieldName(us.schema, tmpField)
		if err != nil {
			return iceberg.PartitionField{}, err
		}
	}

	return iceberg.PartitionField{
		SourceIDs: []int{key.SourceId},
		FieldID:   newFieldId,
		Name:      name,
		Transform: transform,
	}, nil
}

func (us *UpdateSpec) newFieldId() int {
	us.lastAssignedFieldId += 1

	return us.lastAssignedFieldId
}

func (us *UpdateSpec) isDuplicatePartition(transform iceberg.Transform, partitionField iceberg.PartitionField) bool {
	_, deleted := us.deletes[partitionField.FieldID]

	return !deleted && transform.Equals(partitionField.Transform)
}

func (us *UpdateSpec) checkAndAddPartitionName(schema *iceberg.Schema, name string, sourceId int, transform iceberg.Transform, partitionNames map[string]bool) error {
	field, found := schema.FindFieldByName(name)
	_, isVoid := transform.(iceberg.VoidTransform)
	if found && field.ID != sourceId && (sourceId != 0 || !isVoid) {
		return fmt.Errorf("cannot create partition from name that exists in schema %s", name)
	}
	if _, exists := partitionNames[name]; exists {
		return fmt.Errorf("partition name has to be unique: %s", name)
	}
	partitionNames[name] = true

	return nil
}

func (us *UpdateSpec) addNewField(schema *iceberg.Schema, sourceId int, fieldId int, name string, transform iceberg.Transform, partitionNames map[string]bool) (iceberg.PartitionField, error) {
	err := us.checkAndAddPartitionName(schema, name, sourceId, transform, partitionNames)
	if err != nil {
		return iceberg.PartitionField{}, err
	}

	return iceberg.PartitionField{
		SourceIDs: []int{sourceId},
		FieldID:   fieldId,
		Name:      name,
		Transform: transform,
	}, nil
}

func (us *UpdateSpec) isNewPartitionSpec(newSpecId int) bool {
	spec := us.meta.PartitionSpecByID(newSpecId)

	return spec == nil
}
