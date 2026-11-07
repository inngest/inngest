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
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/apache/iceberg-go"
	"github.com/google/uuid"
)

const TableRootID = -1

type MoveOp string

const (
	MoveOpFirst  MoveOp = "first"
	MoveOpBefore MoveOp = "before"
	MoveOpAfter  MoveOp = "after"
)

type move struct {
	FieldID    int
	RelativeTo int
	Op         MoveOp
}

// UpdateSchema manages schema evolution operations within a transaction.
// It supports adding, deleting, renaming, updating, and reordering columns,
// and ensures all changes are validated before being committed.
//
// Operations can be chained together and are applied in the order they are called.
// Changes are not persisted until Commit() is called.
//
// Basic Usage:
//
//	txn := table.NewTransaction()
//	updateSchema := txn.UpdateSchema(true, false)
//
//	// Add a new column
//	updateSchema.AddColumn([]string{"email"}, iceberg.PrimitiveTypes.String, "Email address", false, nil)
//
//	// Commit changes
//	if err := updateSchema.Commit(); err != nil {
//	    return err
//	}
//	if _, err := txn.Commit(ctx); err != nil {
//	    return err
//	}
//
// Chaining Operations:
//
//	updateSchema.
//	    AddColumn([]string{"age"}, iceberg.PrimitiveTypes.Int, "User age", false, nil).
//	    RenameColumn([]string{"name"}, "full_name").
//	    MoveFirst([]string{"id"}).
//	    Commit()
//
// Adding Nested Columns:
//
//	// Add a column to a struct field
//	updateSchema.AddColumn([]string{"address", "country"}, iceberg.PrimitiveTypes.String, "Country code", false, iceberg.StringLiteral("US"))
//
//	// Commit the schema update
//	if err := updateSchema.Commit(); err != nil {
//	    return err
//	}
//	if _, err := txn.Commit(ctx); err != nil {
//	    return err
//	}
type UpdateSchema struct {
	txn          *Transaction
	schema       *iceberg.Schema
	lastColumnID int
	err          error

	deletes map[int]struct{}
	updates map[int]map[int]iceberg.NestedField
	adds    map[int][]iceberg.NestedField
	moves   map[int][]move

	identifierFieldNames map[string]struct{}
	parentID             map[int]int

	addedNameToID            map[string]int
	allowIncompatibleChanges bool
	caseSensitive            bool
	nameMapping              iceberg.NameMapping
	ops                      []func() error
}

// UpdateSchemaOption is a functional option for configuring UpdateSchema.
type UpdateSchemaOption func(*UpdateSchema)

// WithNameMapping configures the UpdateSchema to use the provided name mapping
// for tracking field name changes and ensuring consistency during schema evolution.
func WithNameMapping(nameMapping iceberg.NameMapping) UpdateSchemaOption {
	return func(u *UpdateSchema) {
		u.nameMapping = nameMapping
	}
}

// NewUpdateSchema creates a new UpdateSchema instance for managing schema changes
// within a transaction.
//
// Parameters:
//   - txn: The transaction that this schema update will be applied to.
//   - caseSensitive: If true, field name lookups are case-sensitive; if false,
//     field names are matched case-insensitively.
//   - allowIncompatibleChanges: If true, allows schema changes that would normally
//     be rejected for being incompatible (e.g., adding required fields without
//     default values, changing field types in non-promotable ways, or changing
//     column nullability from optional to required).
//   - opts: Optional configuration functions to customize the UpdateSchema behavior.
//
// Returns an UpdateSchema instance that can be used to build and apply schema changes.
func NewUpdateSchema(txn *Transaction, caseSensitive bool, allowIncompatibleChanges bool, opts ...UpdateSchemaOption) *UpdateSchema {
	u := &UpdateSchema{
		txn:    txn,
		schema: nil,

		deletes: make(map[int]struct{}),
		updates: make(map[int]map[int]iceberg.NestedField),
		adds:    make(map[int][]iceberg.NestedField),
		moves:   make(map[int][]move),

		identifierFieldNames: nil,
		parentID:             make(map[int]int),

		addedNameToID:            make(map[string]int),
		allowIncompatibleChanges: allowIncompatibleChanges,
		caseSensitive:            caseSensitive,
		nameMapping:              nil,
		ops:                      make([]func() error, 0),
	}

	if txn == nil {
		u.err = fmt.Errorf("%w: transaction is nil", ErrInvalidMetadata)

		return u
	}
	meta, err := txn.txnMeta()
	if err != nil {
		u.err = err

		return u
	}

	// Seed from metadata's last-column-id rather than the current schema's
	// highest field id. Per the Iceberg spec, last-column-id is a monotonic
	// counter that preserves ids across schema evolution (including
	// deletions) so that newly-allocated ids never collide with ids still
	// referenced by historical schemas.
	u.lastColumnID = meta.LastColumnID()

	for _, opt := range opts {
		opt(u)
	}

	return u
}

func (u *UpdateSchema) init() error {
	if u.err != nil {
		return u.err
	}
	meta, err := u.txn.txnMeta()
	if err != nil {
		return err
	}

	u.schema = meta.CurrentSchema()
	if u.schema == nil {
		return errors.New("current schema is nil")
	}

	if err := u.initIdentifierFieldNames(); err != nil {
		return err
	}

	if err := u.initParentID(); err != nil {
		return err
	}

	return nil
}

func (u *UpdateSchema) initIdentifierFieldNames() error {
	if u.identifierFieldNames != nil {
		return nil
	}

	identifierFieldNames := make(map[string]struct{})
	for _, id := range u.schema.IdentifierFieldIDs {
		name, ok := u.schema.FindColumnName(id)
		if !ok {
			return fmt.Errorf("identifier field %d not found", id)
		}
		identifierFieldNames[name] = struct{}{}
	}

	u.identifierFieldNames = identifierFieldNames

	return nil
}

func (u *UpdateSchema) initParentID() error {
	parents, err := iceberg.IndexParents(u.schema)
	if err != nil {
		return err
	}

	maps.Copy(u.parentID, parents)

	return nil
}

func (u *UpdateSchema) assignNewColumnID() int {
	u.lastColumnID++

	return u.lastColumnID
}

func (u *UpdateSchema) findField(name string) (iceberg.NestedField, bool) {
	if u.caseSensitive {
		return u.schema.FindFieldByName(name)
	} else {
		return u.schema.FindFieldByNameCaseInsensitive(name)
	}
}

func (u *UpdateSchema) isDeleted(fieldID int) bool {
	_, ok := u.deletes[fieldID]

	return ok
}

func (u *UpdateSchema) findParentID(fieldID int) int {
	parentID, ok := u.parentID[fieldID]
	if !ok {
		return TableRootID
	}

	return parentID
}

func (u *UpdateSchema) AddColumn(path []string, fieldType iceberg.Type, doc string, required bool, defaultValue iceberg.Literal) *UpdateSchema {
	u.ops = append(u.ops, func() error {
		return u.addColumn(path, fieldType, doc, required, defaultValue)
	})

	return u
}

func (u *UpdateSchema) addColumn(path []string, fieldType iceberg.Type, doc string, required bool, defaultValue iceberg.Literal) error {
	if len(path) == 0 {
		return errors.New("path is empty")
	}

	fullName := strings.Join(path, ".")

	switch t := fieldType.(type) {
	case *iceberg.ListType, *iceberg.MapType, *iceberg.StructType:
		if defaultValue != nil {
			return fmt.Errorf("default values are not supported for %s", t.String())
		}
	case iceberg.PrimitiveType:
		if required && defaultValue == nil && !u.allowIncompatibleChanges {
			return fmt.Errorf("required field %s has no default value", fullName)
		}
		if defaultValue != nil && !defaultValue.Type().Equals(t) {
			return fmt.Errorf("default value type mismatch: %s != %s", defaultValue.Type(), t)
		}
	default:
		return fmt.Errorf("invalid field type: %T", t)
	}

	parent := path[:len(path)-1]
	parentID := TableRootID

	if len(parent) > 0 {
		parentFullPath := strings.Join(parent, ".")
		parentField, ok := u.findField(parentFullPath)
		if !ok {
			return fmt.Errorf("parent field not found: %s", parentFullPath)
		}

		switch parentType := parentField.Type.(type) {
		case *iceberg.ListType:
			f := parentType.ElementField()
			parentField = f
		case *iceberg.MapType:
			f := parentType.ValueField()
			parentField = f
		}

		if _, ok := parentField.Type.(*iceberg.StructType); !ok {
			return fmt.Errorf("cannot add field to non-struct type: %s", parentFullPath)
		}

		parentID = parentField.ID
	}

	name := path[len(path)-1]
	for _, add := range u.adds[parentID] {
		if add.Name == name {
			return fmt.Errorf("field already exists in adds: %s", fullName)
		}
	}

	// support add field with the same name as deleted field and renamed field
	if field, ok := u.findField(fullName); ok {
		if !u.isDeleted(field.ID) {
			for _, upd := range u.updates[parentID] {
				if upd.Name == name {
					return fmt.Errorf("field already exists: %s", fullName)
				}
			}
		}
	}

	field := iceberg.NestedField{
		Name:     name,
		Type:     fieldType,
		Required: required,
		Doc:      doc,
	}
	if defaultValue != nil {
		field.InitialDefault = defaultValue.Any()
		field.WriteDefault = defaultValue.Any()
	}

	sch, err := iceberg.AssignFreshSchemaIDs(iceberg.NewSchema(0, field), u.assignNewColumnID)
	if err != nil {
		return fmt.Errorf("failed to assign field id: %w", err)
	}
	u.adds[parentID] = append(u.adds[parentID], sch.Field(0))
	u.addedNameToID[fullName] = sch.Field(0).ID

	return nil
}

func (u *UpdateSchema) DeleteColumn(path []string) *UpdateSchema {
	u.ops = append(u.ops, func() error {
		return u.deleteColumn(path)
	})

	return u
}

func (u *UpdateSchema) deleteColumn(path []string) error {
	fullName := strings.Join(path, ".")
	field, ok := u.findField(fullName)
	if !ok {
		return fmt.Errorf("field not found: %s", fullName)
	}

	if _, ok := u.adds[field.ID]; ok {
		return fmt.Errorf("field that has additions cannot be deleted: %s", fullName)
	}

	// u.updates is keyed by parent ID, then by the field's own ID, so a
	// field's own pending update lives under its parent's map
	parentID := u.findParentID(field.ID)
	if upds, ok := u.updates[parentID]; ok {
		if _, ok := upds[field.ID]; ok {
			return fmt.Errorf("field that has updates cannot be deleted: %s", fullName)
		}
	}
	for _, move := range u.moves[parentID] {
		if move.RelativeTo == field.ID {
			return fmt.Errorf("field that is used as a move target cannot be deleted: %s", fullName)
		}
	}

	delete(u.identifierFieldNames, fullName)

	u.deletes[field.ID] = struct{}{}

	return nil
}

type ColumnUpdate struct {
	Name         iceberg.Optional[string]
	FieldType    iceberg.Optional[iceberg.Type]
	Required     iceberg.Optional[bool]
	WriteDefault iceberg.Optional[iceberg.Literal]
	Doc          iceberg.Optional[string]
}

func (u *UpdateSchema) UpdateColumn(path []string, update ColumnUpdate) *UpdateSchema {
	u.ops = append(u.ops, func() error {
		return u.updateColumn(path, update)
	})

	return u
}

func (u *UpdateSchema) updateColumn(path []string, update ColumnUpdate) error {
	if !update.Name.Valid &&
		!update.FieldType.Valid &&
		!update.Required.Valid &&
		!update.WriteDefault.Valid &&
		!update.Doc.Valid {
		return nil
	}

	fullName := strings.Join(path, ".")

	field, ok := u.findField(fullName)
	if !ok {
		return fmt.Errorf("field not found: %s", fullName)
	}

	if u.isDeleted(field.ID) {
		return fmt.Errorf("field that has been deleted cannot be updated: %s", fullName)
	}

	parentID := u.findParentID(field.ID)

	if update.Name.Valid {
		if update.Name.Val == "" {
			return fmt.Errorf("cannot rename field to empty name: %s", fullName)
		}
		if field.Name == update.Name.Val {
			return fmt.Errorf("cannot rename field to the same name: %s", fullName)
		}

		newFullName := strings.Join(append(path[:len(path)-1], update.Name.Val), ".")
		if existingField, ok := u.findField(newFullName); ok {
			if !u.isDeleted(existingField.ID) {
				return fmt.Errorf("field already exists: %s", newFullName)
			}
		}

		for _, add := range u.adds[parentID] {
			if add.Name == update.Name.Val {
				return fmt.Errorf("cannot rename field to added field: %s", newFullName)
			}
		}

		for _, upd := range u.updates[parentID] {
			if upd.Name == update.Name.Val && upd.ID != field.ID {
				return fmt.Errorf("cannot rename field to renamed field: %s", newFullName)
			}
		}

		if _, ok := u.identifierFieldNames[fullName]; ok {
			delete(u.identifierFieldNames, fullName)
			u.identifierFieldNames[newFullName] = struct{}{}
		}
	}

	if update.FieldType.Valid {
		if _, ok := field.Type.(iceberg.PrimitiveType); !ok {
			return fmt.Errorf("cannot update field type for non-primitive type: %s", fullName)
		}
		if !update.FieldType.Val.Equals(field.Type) && !u.allowIncompatibleChanges {
			fieldType, err := iceberg.PromoteType(field.Type, update.FieldType.Val)
			if err != nil {
				return err
			}
			update.FieldType.Val = fieldType
		}
	}

	if update.Required.Valid {
		if field.Required != update.Required.Val {
			if !u.allowIncompatibleChanges && update.Required.Val {
				return fmt.Errorf("cannot change column nullability from optional to required: %s", fullName)
			}
		}
	}

	if update.WriteDefault.Valid {
		if update.WriteDefault.Val == nil {
			if field.Required && !u.allowIncompatibleChanges {
				return fmt.Errorf("cannot change default value of required column to nil: %s", fullName)
			}
		} else {
			// Validate the new write-default against the column's (possibly promoted) type.
			effectiveType := field.Type
			if update.FieldType.Valid {
				effectiveType = update.FieldType.Val
			}
			if err := validateDefaultValue(effectiveType, update.WriteDefault.Val.Any()); err != nil {
				return fmt.Errorf("invalid write-default for %s: %w", fullName, err)
			}
		}
	}

	if _, ok := u.updates[parentID]; !ok {
		u.updates[parentID] = make(map[int]iceberg.NestedField)
	}

	updatedField, ok := u.updates[parentID][field.ID]
	if !ok {
		updatedField = field
	}
	if update.Name.Valid {
		updatedField.Name = update.Name.Val
	}
	if update.FieldType.Valid {
		updatedField.Type = update.FieldType.Val
	}
	if update.Required.Valid {
		updatedField.Required = update.Required.Val
	}
	if update.WriteDefault.Valid {
		// A nil literal clears the write-default;
		// Calling Any() on it would dereference a nil interface and panic.
		if update.WriteDefault.Val == nil {
			updatedField.WriteDefault = nil
		} else {
			updatedField.WriteDefault = update.WriteDefault.Val.Any()
		}
	}
	if update.Doc.Valid {
		updatedField.Doc = update.Doc.Val
	}
	u.updates[parentID][field.ID] = updatedField

	return nil
}

func (u *UpdateSchema) RenameColumn(path []string, newName string) *UpdateSchema {
	u.ops = append(u.ops, func() error {
		return u.updateColumn(path, ColumnUpdate{
			Name: iceberg.Optional[string]{
				Valid: true,
				Val:   newName,
			},
		})
	})

	return u
}

func (u *UpdateSchema) MoveColumn(op MoveOp, path, relativeTo []string) *UpdateSchema {
	u.ops = append(u.ops, func() error {
		return u.moveColumn(op, path, relativeTo)
	})

	return u
}

func (u *UpdateSchema) MoveFirst(path []string) *UpdateSchema {
	u.ops = append(u.ops, func() error {
		return u.moveColumn(MoveOpFirst, path, nil)
	})

	return u
}

func (u *UpdateSchema) MoveBefore(path, relativeTo []string) *UpdateSchema {
	u.ops = append(u.ops, func() error {
		return u.moveColumn(MoveOpBefore, path, relativeTo)
	})

	return u
}

func (u *UpdateSchema) MoveAfter(path, relativeTo []string) *UpdateSchema {
	u.ops = append(u.ops, func() error {
		return u.moveColumn(MoveOpAfter, path, relativeTo)
	})

	return u
}

func (u *UpdateSchema) findFieldForMove(name string) (int, bool) {
	field, ok := u.findField(name)
	if ok {
		return field.ID, true
	}
	id, ok := u.addedNameToID[name]

	return id, ok
}

func (u *UpdateSchema) moveColumn(op MoveOp, path []string, relativeTo []string) error {
	fullName := strings.Join(path, ".")
	fieldID, ok := u.findFieldForMove(fullName)
	if !ok {
		return fmt.Errorf("field not found: %s", fullName)
	}

	if u.isDeleted(fieldID) {
		return fmt.Errorf("field that has been deleted cannot be moved: %s", fullName)
	}

	parentID := u.findParentID(fieldID)

	switch op {
	case MoveOpFirst:
		u.moves[parentID] = append(u.moves[parentID], move{
			FieldID:    fieldID,
			RelativeTo: -1,
			Op:         op,
		})

		return nil
	case MoveOpBefore, MoveOpAfter:
		relativeToFullName := strings.Join(relativeTo, ".")
		relativeToFieldID, ok := u.findFieldForMove(relativeToFullName)
		if !ok {
			return fmt.Errorf("relative to field not found: %s", relativeToFullName)
		}

		if relativeToFieldID == fieldID {
			return fmt.Errorf("cannot move a field to itself: %s", fullName)
		}
		if u.isDeleted(relativeToFieldID) {
			return fmt.Errorf("field that has been deleted cannot be used as a move target: %s", relativeToFullName)
		}

		if u.findParentID(relativeToFieldID) != parentID {
			return fmt.Errorf("relative to field is not a child of the parent: %s", relativeToFullName)
		}
		u.moves[parentID] = append(u.moves[parentID], move{
			FieldID:    fieldID,
			RelativeTo: relativeToFieldID,
			Op:         op,
		})

		return nil
	default:

		return fmt.Errorf("invalid move operation: %s", op)
	}
}

func (u *UpdateSchema) SetIdentifierField(paths [][]string) *UpdateSchema {
	// Defer the mutation so it runs during Apply() in the order it was chained.
	u.ops = append(u.ops, func() error {
		identifierFieldNames := make(map[string]struct{})
		for _, path := range paths {
			identifierFieldNames[strings.Join(path, ".")] = struct{}{}
		}
		u.identifierFieldNames = identifierFieldNames

		return nil
	})

	return u
}

// UnionByNameWith stages changes to evolve the current schema into the union of
// itself and newSchema, matching fields by name (honoring caseSensitive):
//
//   - New fields are added as optional (regardless of their incoming required
//     flag), keeping their Doc/InitialDefault/WriteDefault and getting fresh ids.
//   - Existing fields may be evolved: required->optional is applied,
//     optional->required is skipped. A primitive type change is applied only if
//     it is a valid promotion (int->long, float->double, decimal same-scale
//     wider-precision, or exact match); narrowing is ignored and any other
//     change errors.
//   - A Doc is updated only when the incoming Doc is non-empty and differs (an
//     empty Doc never clears an existing one).
//   - A WriteDefault is updated when the incoming value differs; an existing
//     column's InitialDefault is never modified.
//   - Map keys are immutable and cross-kind changes (list->map, struct->list,
//     etc.) are rejected rather than silently grafted onto the existing column.
//
// Matching consults only the current schema, not columns staged by an earlier
// op in the same UpdateSchema; pre-merge such schemas before a single call.
func (u *UpdateSchema) UnionByNameWith(newSchema *iceberg.Schema) *UpdateSchema {
	u.ops = append(u.ops, func() error {
		return u.unionByName(newSchema)
	})

	return u
}

func (u *UpdateSchema) unionByName(newSchema *iceberg.Schema) error {
	if newSchema == nil {
		return errors.New("cannot union with nil schema")
	}

	for _, field := range newSchema.Fields() {
		if err := u.unionField([]string{field.Name}, field); err != nil {
			return err
		}
	}

	return nil
}

func (u *UpdateSchema) unionField(path []string, newField iceberg.NestedField) error {
	existing, ok := u.findField(strings.Join(path, "."))
	if !ok {
		return u.unionAddColumn(path, newField)
	}

	if err := u.unionUpdateColumn(path, existing, newField); err != nil {
		return err
	}

	switch t := newField.Type.(type) {
	case *iceberg.StructType:
		for _, child := range t.FieldList {
			if err := u.unionField(childPath(path, child.Name), child); err != nil {
				return err
			}
		}
	case *iceberg.ListType:
		if err := u.unionField(childPath(path, "element"), t.ElementField()); err != nil {
			return err
		}
	case *iceberg.MapType:
		// Map keys are immutable; reject any key change up front
		existingMap, ok := existing.Type.(*iceberg.MapType)
		if !ok {
			return fmt.Errorf("cannot change field to map: %s", strings.Join(path, "."))
		}
		if !isIgnorableTypeUpdate(existingMap.KeyType, t.KeyType) {
			return fmt.Errorf("cannot update map key: %s", strings.Join(childPath(path, "key"), "."))
		}
		if err := u.unionField(childPath(path, "value"), t.ValueField()); err != nil {
			return err
		}
	default:
		// Primitive and Variant types have no nested children to recurse into.
	}

	return nil
}

func childPath(parent []string, name string) []string {
	child := make([]string, 0, len(parent)+1)
	child = append(child, parent...)

	return append(child, name)
}

func (u *UpdateSchema) unionAddColumn(path []string, newField iceberg.NestedField) error {
	fullName := strings.Join(path, ".")

	parent := path[:len(path)-1]
	parentID := TableRootID

	if len(parent) > 0 {
		parentFullPath := strings.Join(parent, ".")
		parentField, ok := u.findField(parentFullPath)
		if !ok {
			return fmt.Errorf("parent field not found: %s", parentFullPath)
		}

		switch parentType := parentField.Type.(type) {
		case *iceberg.ListType:
			parentField = parentType.ElementField()
		case *iceberg.MapType:
			parentField = parentType.ValueField()
		}

		if _, ok := parentField.Type.(*iceberg.StructType); !ok {
			return fmt.Errorf("cannot add field to non-struct type: %s", parentFullPath)
		}

		parentID = parentField.ID
	}

	name := path[len(path)-1]
	for _, add := range u.adds[parentID] {
		if add.Name == name {
			return fmt.Errorf("field already exists in adds: %s", fullName)
		}
	}

	// guard against colliding with a pending rename
	if field, ok := u.findField(fullName); ok {
		if !u.isDeleted(field.ID) {
			for _, upd := range u.updates[parentID] {
				if upd.Name == name {
					return fmt.Errorf("field already exists: %s", fullName)
				}
			}
		}
	}

	if err := validateFieldDefaults(newField); err != nil {
		return err
	}

	field := iceberg.NestedField{
		Name:           name,
		Type:           newField.Type,
		Required:       false,
		Doc:            newField.Doc,
		InitialDefault: newField.InitialDefault,
		WriteDefault:   newField.WriteDefault,
	}

	sch, err := iceberg.AssignFreshSchemaIDs(iceberg.NewSchema(0, field), u.assignNewColumnID)
	if err != nil {
		return fmt.Errorf("failed to assign field id: %w", err)
	}
	u.adds[parentID] = append(u.adds[parentID], sch.Field(0))
	u.addedNameToID[fullName] = sch.Field(0).ID

	return nil
}

func (u *UpdateSchema) unionUpdateColumn(path []string, existing, newField iceberg.NestedField) error {
	update := ColumnUpdate{}

	if !newField.Required && existing.Required {
		update.Required = iceberg.Optional[bool]{Valid: true, Val: false}
	}

	if !isIgnorableTypeUpdate(existing.Type, newField.Type) {
		// Only a valid primitive promotion may change a column's type. Anything
		// else (non-primitive, cross-kind, or a disallowed promotion) is an error.
		existingPrim, existingIsPrim := existing.Type.(iceberg.PrimitiveType)
		newPrim, newIsPrim := newField.Type.(iceberg.PrimitiveType)
		if !existingIsPrim || !newIsPrim || !isPromotionAllowed(existingPrim, newPrim) {
			return fmt.Errorf("cannot change column type: %s: %s -> %s",
				strings.Join(path, "."), existing.Type, newField.Type)
		}
		update.FieldType = iceberg.Optional[iceberg.Type]{Valid: true, Val: newField.Type}
	}

	if newField.Doc != "" && newField.Doc != existing.Doc {
		update.Doc = iceberg.Optional[string]{Valid: true, Val: newField.Doc}
	}

	if update.Required.Valid || update.FieldType.Valid || update.Doc.Valid {
		if err := u.updateColumn(path, update); err != nil {
			return err
		}
	}

	if newField.WriteDefault != nil && !reflect.DeepEqual(newField.WriteDefault, existing.WriteDefault) {
		// Validate against the promoted type when this same op changes the
		// column's type
		effectiveType := existing.Type
		if update.FieldType.Valid {
			effectiveType = update.FieldType.Val
		}
		if err := u.setWriteDefault(existing, effectiveType, newField.WriteDefault); err != nil {
			return err
		}
	}

	return nil
}

func (u *UpdateSchema) setWriteDefault(existing iceberg.NestedField, fieldType iceberg.Type, writeDefault any) error {
	if u.isDeleted(existing.ID) {
		return fmt.Errorf("field that has been deleted cannot be updated: %s", existing.Name)
	}
	if _, ok := fieldType.(iceberg.PrimitiveType); !ok {
		return fmt.Errorf("default values are not supported for %s", fieldType)
	}
	if err := validateDefaultValue(fieldType, writeDefault); err != nil {
		return fmt.Errorf("invalid write-default for %s: %w", existing.Name, err)
	}

	parentID := u.findParentID(existing.ID)
	if u.updates[parentID] == nil {
		u.updates[parentID] = make(map[int]iceberg.NestedField)
	}

	updatedField, ok := u.updates[parentID][existing.ID]
	if !ok {
		updatedField = existing
	}
	updatedField.WriteDefault = writeDefault
	u.updates[parentID][existing.ID] = updatedField

	return nil
}

func validateNestedDefaults(t iceberg.Type) error {
	switch typ := t.(type) {
	case *iceberg.StructType:
		for _, f := range typ.FieldList {
			if err := validateFieldDefaults(f); err != nil {
				return err
			}
		}
	case *iceberg.ListType:
		return validateFieldDefaults(typ.ElementField())
	case *iceberg.MapType:
		if err := validateFieldDefaults(typ.KeyField()); err != nil {
			return err
		}

		return validateFieldDefaults(typ.ValueField())
	}

	return nil
}

func validateFieldDefaults(f iceberg.NestedField) error {
	switch f.Type.(type) {
	case *iceberg.StructType, *iceberg.ListType, *iceberg.MapType:
		if err := validateComplexDefault(f); err != nil {
			return err
		}

		return validateNestedDefaults(f.Type)
	case iceberg.PrimitiveType:
		if err := validateDefaultValue(f.Type, f.InitialDefault); err != nil {
			return fmt.Errorf("invalid initial-default for %s: %w", f.Name, err)
		}
		if err := validateDefaultValue(f.Type, f.WriteDefault); err != nil {
			return fmt.Errorf("invalid write-default for %s: %w", f.Name, err)
		}
	}

	return nil
}

// nativeDefaultLiteral wraps a default already in a native iceberg Go type so
// its Literal type can be checked against the column type. Values whose Go
// encoding is shared across iceberg types (e.g. string, []byte, float64,
// Decimal) are excluded, deferring to the materialization check below.
func nativeDefaultLiteral(v any) (iceberg.Literal, bool) {
	switch val := v.(type) {
	case bool:
		return iceberg.NewLiteral(val), true
	case int32:
		return iceberg.NewLiteral(val), true
	case int64:
		return iceberg.NewLiteral(val), true
	case float32:
		return iceberg.NewLiteral(val), true
	case iceberg.Date:
		return iceberg.NewLiteral(val), true
	case iceberg.Time:
		return iceberg.NewLiteral(val), true
	case iceberg.Timestamp:
		return iceberg.NewLiteral(val), true
	case iceberg.TimestampNano:
		return iceberg.NewLiteral(val), true
	case uuid.UUID:
		return iceberg.NewLiteral(val), true
	}

	return nil, false
}

func validateDefaultValue(t iceberg.Type, v any) (err error) {
	if v == nil {
		return nil
	}
	if _, ok := t.(iceberg.PrimitiveType); !ok {
		return fmt.Errorf("default values are not supported for %s", t)
	}

	if lit, ok := nativeDefaultLiteral(v); ok && !lit.Type().Equals(t) {
		return fmt.Errorf("value of type %s does not match column type %s", lit.Type(), t)
	}

	dt, err := TypeToArrowType(t, false, false)
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("value %#v is not valid for type %s: %v", v, t, r)
		}
	}()
	_ = defaultToScalar(v, t, dt)

	return nil
}

// isIgnorableTypeUpdate reports whether changing existingType to newType is a
// no-op for union-by-name: either the incoming primitive is narrower-or-equal
// (so the existing, wider type already covers it), or both types are the same
// complex kind. Cross-kind changes return false so the caller can reject them.
//
// Note: the promotion check is called with the arguments reversed relative to
// the rest of the file. Here we ask whether newType can be promoted *up* to
// existingType; a true result means the change is narrowing and ignorable.
func isIgnorableTypeUpdate(existingType, newType iceberg.Type) bool {
	// An unchanged type is always a no-op. This also covers parameterless
	// non-primitive types such as variant.
	if existingType.Equals(newType) {
		return true
	}

	if existingPrim, ok := existingType.(iceberg.PrimitiveType); ok {
		newPrimitive, ok := newType.(iceberg.PrimitiveType)
		if !ok {
			return false
		}

		return isPromotionAllowed(newPrimitive, existingPrim)
	}

	switch existingType.(type) {
	case *iceberg.StructType:
		_, ok := newType.(*iceberg.StructType)

		return ok
	case *iceberg.ListType:
		_, ok := newType.(*iceberg.ListType)

		return ok
	case *iceberg.MapType:
		_, ok := newType.(*iceberg.MapType)

		return ok
	}

	return false
}

func isPromotionAllowed(from, to iceberg.PrimitiveType) bool {
	if from.Equals(to) {
		return true
	}
	switch f := from.(type) {
	case iceberg.Int32Type:
		_, ok := to.(iceberg.Int64Type)

		return ok
	case iceberg.Float32Type:
		_, ok := to.(iceberg.Float64Type)

		return ok
	case iceberg.DecimalType:
		t, ok := to.(iceberg.DecimalType)
		if !ok {
			return false
		}

		return f.Scale() == t.Scale() && f.Precision() <= t.Precision()
	}

	return false
}

func (u *UpdateSchema) BuildUpdates() ([]Update, []Requirement, error) {
	if u.err != nil {
		return nil, nil, u.err
	}

	newSchema, err := u.Apply()
	if err != nil {
		return nil, nil, err
	}

	meta, err := u.txn.txnMeta()
	if err != nil {
		return nil, nil, err
	}

	existingSchemaID := -1
	for _, schema := range meta.schemaList {
		if newSchema.Equals(schema) {
			existingSchemaID = schema.ID

			break
		}
	}

	requirements := make([]Requirement, 0)
	updates := make([]Update, 0)

	if existingSchemaID != u.schema.ID {
		requirements = append(requirements, AssertCurrentSchemaID(u.schema.ID))
		if existingSchemaID == -1 {
			updates = append(
				updates,
				NewAddSchemaUpdate(newSchema),
				NewSetCurrentSchemaUpdate(newSchema.ID),
			)
		} else {
			updates = append(updates,
				NewSetCurrentSchemaUpdate(existingSchemaID),
			)
		}

		if u.nameMapping != nil {
			updatesMap := make(map[int]iceberg.NestedField)
			for _, upds := range u.updates {
				maps.Copy(updatesMap, upds)
			}
			updatedNameMapping, err := iceberg.UpdateNameMapping(u.nameMapping, updatesMap, u.adds)
			if err != nil {
				return nil, nil, err
			}
			mappingJSON, err := json.Marshal(updatedNameMapping)
			if err != nil {
				return nil, nil, err
			}
			updates = append(updates, NewSetPropertiesUpdate(iceberg.Properties{
				DefaultNameMappingKey: string(mappingJSON),
			}))
		}
	}

	return updates, requirements, nil
}

func (u *UpdateSchema) Apply() (*iceberg.Schema, error) {
	if u.err != nil {
		return nil, u.err
	}

	if err := u.init(); err != nil {
		return nil, err
	}

	for _, op := range u.ops {
		if err := op(); err != nil {
			return nil, err
		}
	}

	updates := make(map[int]iceberg.NestedField)
	for _, upds := range u.updates {
		maps.Copy(updates, upds)
	}
	st, err := iceberg.Visit(u.schema, &applyChanges{
		adds:    u.adds,
		updates: updates,
		deletes: u.deletes,
		moves:   u.moves,
	})
	if err != nil {
		return nil, fmt.Errorf("error applying schema changes: %w", err)
	}

	identifierFieldIDs := make([]int, 0)
	newSchema := iceberg.NewSchema(0, st.(*iceberg.StructType).FieldList...)
	for name := range u.identifierFieldNames {
		var field iceberg.NestedField
		var ok bool
		if u.caseSensitive {
			field, ok = newSchema.FindFieldByName(name)
		} else {
			field, ok = newSchema.FindFieldByNameCaseInsensitive(name)
		}
		if !ok {
			return nil, fmt.Errorf("identifier field not found: %s", name)
		}
		identifierFieldIDs = append(identifierFieldIDs, field.ID)
	}

	// u.identifierFieldNames is a map, so the iteration above yields the ids in a
	// non-deterministic order. Thus, sorting them to produce a canonical ordering:
	// Schema.Equals compares IdentifierFieldIDs positionally, so,
	// an unsorted slice would make otherwise-identical schemas compare unequal at random.
	slices.Sort(identifierFieldIDs)

	meta, err := u.txn.txnMeta()
	if err != nil {
		return nil, err
	}

	nextSchemaID := 1
	if len(meta.schemaList) > 0 {
		nextSchemaID = 1 + slices.MaxFunc(meta.schemaList, func(a, b *iceberg.Schema) int {
			return a.ID - b.ID
		}).ID
	}

	return iceberg.NewSchemaWithIdentifiers(nextSchemaID, identifierFieldIDs, st.(*iceberg.StructType).FieldList...), nil
}

func (u *UpdateSchema) Commit() error {
	if u.err != nil {
		return u.err
	}

	updates, requirements, err := u.BuildUpdates()
	if err != nil {
		return err
	}
	if len(updates) == 0 {
		return nil
	}

	return u.txn.apply(updates, requirements)
}

type applyChanges struct {
	adds    map[int][]iceberg.NestedField
	updates map[int]iceberg.NestedField
	deletes map[int]struct{}
	moves   map[int][]move
}

func (a *applyChanges) Schema(schema *iceberg.Schema, structResult iceberg.Type) iceberg.Type {
	added := a.adds[TableRootID]
	moves := a.moves[TableRootID]

	if len(added) > 0 || len(moves) > 0 {
		if newFields := addAndMoveFields(structResult.(*iceberg.StructType).Fields(), added, moves); newFields != nil {
			return &iceberg.StructType{FieldList: newFields}
		}
	}

	return structResult
}

func (a *applyChanges) Struct(structType iceberg.StructType, fieldResults []iceberg.Type) iceberg.Type {
	hasChanges := false
	newFields := make([]iceberg.NestedField, 0)

	for i, resultType := range fieldResults {
		if resultType == nil {
			hasChanges = true

			continue
		}

		field := structType.Fields()[i]

		name := field.Name
		doc := field.Doc
		required := field.Required
		writeDefault := field.WriteDefault

		if update, ok := a.updates[field.ID]; ok {
			name = update.Name
			doc = update.Doc
			required = update.Required
			writeDefault = update.WriteDefault
		}
		if field.Name == name &&
			field.Type.Equals(resultType) &&
			field.Required == required &&
			field.Doc == doc &&
			field.WriteDefault == writeDefault {
			newFields = append(newFields, field)
		} else {
			hasChanges = true
			newFields = append(newFields, iceberg.NestedField{
				ID:             field.ID,
				Name:           name,
				Type:           resultType,
				Required:       required,
				Doc:            doc,
				InitialDefault: field.InitialDefault,
				WriteDefault:   writeDefault,
			})
		}
	}

	if hasChanges {
		return &iceberg.StructType{FieldList: newFields}
	}

	return &structType
}

func (a *applyChanges) Field(field iceberg.NestedField, fieldResult iceberg.Type) iceberg.Type {
	if _, ok := a.deletes[field.ID]; ok {
		return nil
	}

	if update, ok := a.updates[field.ID]; ok && !field.Type.Equals(update.Type) {
		return update.Type
	}

	st, ok := fieldResult.(*iceberg.StructType)
	if !ok {
		return fieldResult
	}

	added := a.adds[field.ID]
	moves := a.moves[field.ID]
	if len(added) > 0 || len(moves) > 0 {
		newFields := addAndMoveFields(st.FieldList, added, moves)
		if len(newFields) > 0 {
			return &iceberg.StructType{FieldList: newFields}
		}
	}

	return fieldResult
}

// updateColumn seeds each pending update from a copy of the original
// NestedField. As a result, updates that only modify other properties
// still preserve the original Required value here.
//
// Note: If updateColumn is refactored to lazily populate only the
// touched fields, this helper must gain an explicit "Required was set"
// signal or it will silently return the zero-value false.
func (a *applyChanges) updatedRequired(fieldID int, fallback bool) bool {
	if update, ok := a.updates[fieldID]; ok {
		return update.Required
	}

	return fallback
}

func (a *applyChanges) List(listType iceberg.ListType, elementResult iceberg.Type) iceberg.Type {
	elementType := a.Field(listType.ElementField(), elementResult)
	if elementType == nil {
		panic(fmt.Sprintf("cannot delete element type from list: %s", elementResult))
	}

	return &iceberg.ListType{
		ElementID:       listType.ElementID,
		Element:         elementType,
		ElementRequired: a.updatedRequired(listType.ElementID, listType.ElementRequired),
	}
}

func (a *applyChanges) Map(mapType iceberg.MapType, keyResult, valueResult iceberg.Type) iceberg.Type {
	keyID := mapType.KeyID
	if _, ok := a.deletes[keyID]; ok {
		panic(fmt.Errorf("cannot delete map keys: %s", mapType.String()))
	}

	if _, ok := a.updates[keyID]; ok {
		panic(fmt.Errorf("cannot update map keys: %s", mapType.String()))
	}

	if _, ok := a.adds[keyID]; ok {
		panic(fmt.Errorf("cannot add fields to map keys: %s", mapType.String()))
	}

	if !mapType.KeyType.Equals(keyResult) {
		panic(fmt.Errorf("cannot alter map keys: %s", mapType.String()))
	}

	valueField := mapType.ValueField()
	valueType := a.Field(valueField, valueResult)

	if valueType == nil {
		panic(fmt.Errorf("cannot delete value type from map: %s", mapType.String()))
	}

	return &iceberg.MapType{
		KeyID:         mapType.KeyID,
		KeyType:       mapType.KeyType,
		ValueID:       mapType.ValueID,
		ValueType:     valueType,
		ValueRequired: a.updatedRequired(mapType.ValueID, mapType.ValueRequired),
	}
}

func (a *applyChanges) Primitive(primitive iceberg.PrimitiveType) iceberg.Type {
	return primitive
}

func (a *applyChanges) Variant(variant iceberg.VariantType) iceberg.Type {
	return variant
}

func addFields(fields []iceberg.NestedField, adds []iceberg.NestedField) []iceberg.NestedField {
	return append(fields, adds...)
}

func moveFields(fields []iceberg.NestedField, moves []move) []iceberg.NestedField {
	reordered := slices.Clone(fields)
	for _, move := range moves {
		var fieldToMove iceberg.NestedField
		var fieldIndex int
		found := false
		for i, field := range reordered {
			if field.ID == move.FieldID {
				fieldToMove = field
				fieldIndex = i
				found = true

				break
			}
		}
		if !found {
			panic(fmt.Errorf("cannot move field %d: field not found", move.FieldID))
		}

		reordered = append(reordered[:fieldIndex], reordered[fieldIndex+1:]...)

		switch move.Op {
		case MoveOpFirst:
			reordered = append([]iceberg.NestedField{fieldToMove}, reordered...)
		case MoveOpBefore, MoveOpAfter:
			var relativeIndex int
			found = false
			for i, field := range reordered {
				if field.ID == move.RelativeTo {
					relativeIndex = i
					found = true

					break
				}
			}
			if !found {
				panic(fmt.Errorf("cannot move field %d: target field %d not found", move.FieldID, move.RelativeTo))
			}

			if move.Op == MoveOpBefore {
				reordered = append(reordered[:relativeIndex], append([]iceberg.NestedField{fieldToMove}, reordered[relativeIndex:]...)...)
			} else {
				reordered = append(reordered[:relativeIndex+1], append([]iceberg.NestedField{fieldToMove}, reordered[relativeIndex+1:]...)...)
			}
		}
	}

	return reordered
}

func addAndMoveFields(fields []iceberg.NestedField, adds []iceberg.NestedField, moves []move) []iceberg.NestedField {
	if len(adds) > 0 {
		added := addFields(fields, adds)
		if len(moves) > 0 {
			return moveFields(added, moves)
		} else {
			return added
		}
	} else if len(moves) > 0 {
		return moveFields(fields, moves)
	}
	if len(adds) == 0 {
		return nil
	}

	return fields
}
