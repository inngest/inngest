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
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/apache/iceberg-go"
)

type ErrIncompatibleSchema struct {
	fields        []IncompatibleField
	formatVersion int
}

func (e ErrIncompatibleSchema) Error() string {
	var problems strings.Builder
	for _, f := range e.fields {
		if f.UnsupportedType != nil {
			fmt.Fprintf(&problems, "\n- invalid type for %s: %s is not supported until v%d", f.ColName, f.Field.Type, f.UnsupportedType.MinFormatVersion)
		}
		if f.InvalidDefault != nil {
			if f.InvalidDefault.MustBeNullForType {
				if f.Field.InitialDefault != nil {
					fmt.Fprintf(&problems, "\n- invalid initial default for %s: %s columns must default to null", f.ColName, f.Field.Type)
				}
				if f.Field.WriteDefault != nil {
					fmt.Fprintf(&problems, "\n- invalid write default for %s: %s columns must default to null", f.ColName, f.Field.Type)
				}
			} else {
				if f.Field.InitialDefault != nil {
					fmt.Fprintf(&problems, "\n- invalid initial default for %s: non-null default (%v) is not supported until v%d", f.ColName, formatDefaultValue(f.Field.InitialDefault), f.InvalidDefault.MinFormatVersion)
				}
				if f.Field.WriteDefault != nil {
					fmt.Fprintf(&problems, "\n- invalid write default for %s: non-null default (%v) is not supported until v%d", f.ColName, formatDefaultValue(f.Field.WriteDefault), f.InvalidDefault.MinFormatVersion)
				}
			}
		}
	}

	return fmt.Sprintf("%s: for v%d:%s", iceberg.ErrInvalidSchema, e.formatVersion, problems.String())
}

func (e ErrIncompatibleSchema) Unwrap() error {
	return iceberg.ErrInvalidSchema
}

type IncompatibleField struct {
	Field           iceberg.NestedField
	ColName         string
	UnsupportedType *UnsupportedType
	InvalidDefault  *InvalidDefault
}

type UnsupportedType struct {
	MinFormatVersion int
}

type InvalidDefault struct {
	MinFormatVersion  int
	MustBeNullForType bool
}

func formatDefaultValue(value any) any {
	v := reflect.ValueOf(value)
	for v.IsValid() && v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return nil
	}

	return v.Interface()
}

// checkSchemaCompatibility checks that the schema is compatible with the table's format version.
// This validates that the schema does not contain types or features that were released
// in later format versions.
// Java: Schema::checkCompatibility
// This check runs when a schema is added to a MetadataBuilder during table
// construction or schema evolution. ParseMetadataBytes unmarshals existing
// metadata directly and does not call this check. We intentionally validate
// both default fields here, including write-default for pre-v3 schemas.
func checkSchemaCompatibility(sc *iceberg.Schema, formatVersion int) error {
	const defaultValuesMinFormatVersion = 3
	problems := make([]IncompatibleField, 0)

	if err := validateUnknownTypes(sc); err != nil {
		return fmt.Errorf("failed to validate unknown types: %w", err)
	}

	if err := validateComplexTypeDefaults(sc); err != nil {
		return fmt.Errorf("failed to validate complex type defaults: %w", err)
	}

	if _, err := iceberg.IndexNameByID(sc); err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}

	fieldsIt, err := sc.FlatFields()
	if err != nil {
		return fmt.Errorf("failed to check Schema compatibility: %w", err)
	}

	for _, field := range slices.SortedFunc(fieldsIt, func(a, b iceberg.NestedField) int {
		return cmp.Compare(a.ID, b.ID)
	}) {
		colName, found := sc.FindColumnName(field.ID)
		if !found {
			panic("invalid schema: field with id " + strconv.Itoa(field.ID) + " not found, this is a bug, please report.")
		}

		// Reject row-lineage metadata columns (_row_id,
		// _last_updated_sequence_number) if a caller adds them to a stored
		// schema during evolution. Other spec-reserved IDs (e.g. position-delete
		// file_path/pos) are intentionally permitted: internal writers build
		// throwaway metadata from those schemas via AddSchema. User schemas are
		// validated against the full reserved range separately, before
		// reassignIDs, in NewMetadataWithUUID.
		if iceberg.IsMetadataColumn(field.ID) {
			return fmt.Errorf("%w: field '%s' uses reserved metadata column ID %d",
				iceberg.ErrInvalidSchema, colName, field.ID)
		}

		minFormatVersion := minFormatVersionForType(field.Type)
		if formatVersion < minFormatVersion {
			problems = append(problems, IncompatibleField{
				Field:           field,
				ColName:         colName,
				UnsupportedType: &UnsupportedType{MinFormatVersion: minFormatVersion},
			})
		}

		switch field.Type.(type) {
		case iceberg.GeometryType, iceberg.GeographyType:
			if field.InitialDefault != nil || field.WriteDefault != nil {
				problems = append(problems, IncompatibleField{
					Field:          field,
					ColName:        colName,
					InvalidDefault: &InvalidDefault{MustBeNullForType: true},
				})
			}
		default:
			if (field.InitialDefault != nil || field.WriteDefault != nil) && formatVersion < defaultValuesMinFormatVersion {
				problems = append(problems, IncompatibleField{
					Field:          field,
					ColName:        colName,
					InvalidDefault: &InvalidDefault{MinFormatVersion: defaultValuesMinFormatVersion},
				})
			}
		}
	}

	if len(problems) != 0 {
		return ErrIncompatibleSchema{fields: problems, formatVersion: formatVersion}
	}

	return nil
}

// validateNoReservedFieldIDs rejects user-supplied schemas that assign field IDs
// in the range the spec reserves for metadata columns (_row_id,
// _last_updated_sequence_number, _file, _pos, _deleted, ...). Field IDs must not
// exceed iceberg.MaxStructFieldID. The walk is recursive: FlatFields yields every
// leaf and nested field, so a reserved ID buried in a struct, list element, or
// map key/value is caught too.
//
// This guards the table-creation path only. NewMetadataWithUUID calls it on the
// user schema before reassignIDs, because reassignment overwrites every ID with
// a fresh, non-reserved value and would otherwise mask a reserved ID the caller
// supplied (see #1107). It intentionally covers the full reserved range, unlike
// checkSchemaCompatibility, which permits internal writers to add position-delete
// schemas (file_path/pos) via AddSchema.
func validateNoReservedFieldIDs(sc *iceberg.Schema) error {
	fieldsIt, err := sc.FlatFields()
	if err != nil {
		return fmt.Errorf("failed to enumerate schema fields: %w", err)
	}

	// Sort by ID so the reported field is deterministic when several are reserved.
	for _, field := range slices.SortedFunc(fieldsIt, func(a, b iceberg.NestedField) int {
		return cmp.Compare(a.ID, b.ID)
	}) {
		if !iceberg.IsReservedFieldID(field.ID) {
			continue
		}

		// Report the schema's own name/path for the field, not the canonical
		// metadata column name, so the caller can locate the offending field.
		name, ok := sc.FindColumnName(field.ID)
		if !ok {
			name = field.Name
		}

		return fmt.Errorf("%w: field '%s' uses reserved metadata column ID %d",
			iceberg.ErrInvalidSchema, name, field.ID)
	}

	return nil
}

// minFormatVersionForType returns the minimum table format version required
// for the given type. Returns 1 for types supported in all versions, or a higher
// version number for types that require newer format versions.
func minFormatVersionForType(t iceberg.Type) int {
	switch t.(type) {
	case iceberg.TimestampNsType, iceberg.TimestampTzNsType, iceberg.UnknownType, iceberg.VariantType, iceberg.GeometryType, iceberg.GeographyType:
		return 3
	default:
		// All other types supported in v1+
		return 1
	}
}

func validateUnknownTypes(sc *iceberg.Schema) error {
	validator := &unknownTypeValidator{}
	result, err := iceberg.Visit(sc, validator)
	if err != nil {
		return err
	}

	return result
}

type unknownTypeValidator struct{}

func (v *unknownTypeValidator) Schema(_ *iceberg.Schema, structResult error) error {
	return structResult
}

func (v *unknownTypeValidator) Struct(_ iceberg.StructType, fieldResults []error) error {
	for _, err := range fieldResults {
		if err != nil {
			return err
		}
	}

	return nil
}

func typeRequiresNullDefaults(t iceberg.Type) bool {
	switch t.(type) {
	case iceberg.UnknownType, iceberg.VariantType:
		return true
	default:
		return false
	}
}

func (v *unknownTypeValidator) Field(field iceberg.NestedField, fieldResult error) error {
	if fieldResult != nil {
		return fieldResult
	}
	// Optionality: unknown must be optional per spec; variant has no such constraint.
	// Both require null defaults (enforced below via typeRequiresNullDefaults).
	if _, isUnknown := field.Type.(iceberg.UnknownType); isUnknown {
		if field.Required {
			return fmt.Errorf("unknown type field '%s' (id: %d) must be optional, but was marked as required",
				field.Name, field.ID)
		}
	}
	if typeRequiresNullDefaults(field.Type) {
		if field.InitialDefault != nil {
			return fmt.Errorf("%s type field '%s' (id: %d) must have null initial-default, but got: %v",
				field.Type, field.Name, field.ID, field.InitialDefault)
		}
		if field.WriteDefault != nil {
			return fmt.Errorf("%s type field '%s' (id: %d) must have null write-default, but got: %v",
				field.Type, field.Name, field.ID, field.WriteDefault)
		}
	}

	return nil
}

func (v *unknownTypeValidator) List(list iceberg.ListType, elemResult error) error {
	if elemResult != nil {
		return elemResult
	}
	elem := list.ElementField()

	if _, isUnknown := elem.Type.(iceberg.UnknownType); isUnknown {
		if elem.Required {
			return fmt.Errorf("unknown type field '%s' (id: %d) must be optional, but was marked required",
				elem.Name, elem.ID)
		}
	}
	if typeRequiresNullDefaults(elem.Type) {
		if elem.InitialDefault != nil {
			return fmt.Errorf("%s type field '%s' (id: %d) must have null initial-default, but got: %v",
				elem.Type, elem.Name, elem.ID, elem.InitialDefault)
		}
		if elem.WriteDefault != nil {
			return fmt.Errorf("%s type field '%s' (id: %d) must have null write-default, but got: %v",
				elem.Type, elem.Name, elem.ID, elem.WriteDefault)
		}
	}

	return nil
}

func (v *unknownTypeValidator) Map(mapType iceberg.MapType, keyResult, valueResult error) error {
	if keyResult != nil {
		return keyResult
	}

	if valueResult != nil {
		return valueResult
	}

	key := mapType.KeyField()

	if _, isKeyUnknown := key.Type.(iceberg.UnknownType); isKeyUnknown {
		if key.Required {
			return fmt.Errorf("unknown type field '%s' (id: %d) must be optional, but was marked required",
				key.Name, key.ID)
		}
	}
	if typeRequiresNullDefaults(key.Type) {
		if key.InitialDefault != nil {
			return fmt.Errorf("%s type field '%s' (id: %d) must have null initial-default, but got: %v",
				key.Type, key.Name, key.ID, key.InitialDefault)
		}
		if key.WriteDefault != nil {
			return fmt.Errorf("%s type field '%s' (id: %d) must have null write-default, but got: %v",
				key.Type, key.Name, key.ID, key.WriteDefault)
		}
	}

	value := mapType.ValueField()

	if _, isValueUnknown := value.Type.(iceberg.UnknownType); isValueUnknown {
		if value.Required {
			return fmt.Errorf("unknown type field '%s' (id: %d) must be optional, but was marked required",
				value.Name, value.ID)
		}
	}
	if typeRequiresNullDefaults(value.Type) {
		if value.InitialDefault != nil {
			return fmt.Errorf("%s type field '%s' (id: %d) must have null initial-default, but got: %v",
				value.Type, value.Name, value.ID, value.InitialDefault)
		}
		if value.WriteDefault != nil {
			return fmt.Errorf("%s type field '%s' (id: %d) must have null write-default, but got: %v",
				value.Type, value.Name, value.ID, value.WriteDefault)
		}
	}

	return nil
}

func (v *unknownTypeValidator) Primitive(_ iceberg.PrimitiveType) error {
	return nil
}

func (v *unknownTypeValidator) Variant(_ iceberg.VariantType) error {
	return nil
}

func validateComplexTypeDefaults(sc *iceberg.Schema) error {
	validator := &complexTypeDefaultValidator{}
	result, err := iceberg.Visit(sc, validator)
	if err != nil {
		return err
	}

	return result
}

type complexTypeDefaultValidator struct{}

func (v *complexTypeDefaultValidator) Schema(_ *iceberg.Schema, structResult error) error {
	return structResult
}

func (v *complexTypeDefaultValidator) Struct(_ iceberg.StructType, fieldResults []error) error {
	for _, err := range fieldResults {
		if err != nil {
			return err
		}
	}

	return nil
}

func (v *complexTypeDefaultValidator) Field(field iceberg.NestedField, fieldResult error) error {
	if fieldResult != nil {
		return fieldResult
	}

	return validateComplexDefault(field)
}

func (v *complexTypeDefaultValidator) List(list iceberg.ListType, elemResult error) error {
	if elemResult != nil {
		return elemResult
	}

	return validateComplexDefault(list.ElementField())
}

func (v *complexTypeDefaultValidator) Map(mapType iceberg.MapType, keyResult, valueResult error) error {
	if keyResult != nil {
		return keyResult
	}

	if valueResult != nil {
		return valueResult
	}

	if err := validateComplexDefault(mapType.KeyField()); err != nil {
		return err
	}

	return validateComplexDefault(mapType.ValueField())
}

func (v *complexTypeDefaultValidator) Primitive(_ iceberg.PrimitiveType) error {
	return nil
}

func (v *complexTypeDefaultValidator) Variant(_ iceberg.VariantType) error {
	return nil
}

func validateComplexDefault(field iceberg.NestedField) error {
	switch field.Type.(type) {
	case *iceberg.StructType:
		if field.InitialDefault != nil {
			if _, ok := field.InitialDefault.(map[string]any); !ok {
				return fmt.Errorf("struct type field '%s' (id: %d) must have null or JSON object initial-default, but got: %v", field.Name, field.ID, field.InitialDefault)
			}
		}
		if field.WriteDefault != nil {
			if _, ok := field.WriteDefault.(map[string]any); !ok {
				return fmt.Errorf("struct type field '%s' (id: %d) must have null or JSON object write-default, but got: %v", field.Name, field.ID, field.WriteDefault)
			}
		}
	case *iceberg.ListType:
		if field.InitialDefault != nil {
			if _, ok := field.InitialDefault.([]any); !ok {
				return fmt.Errorf("list type field '%s' (id: %d) must have null or JSON array initial-default, but got: %v", field.Name, field.ID, field.InitialDefault)
			}
		}
		if field.WriteDefault != nil {
			if _, ok := field.WriteDefault.([]any); !ok {
				return fmt.Errorf("list type field '%s' (id: %d) must have null or JSON array write-default, but got: %v", field.Name, field.ID, field.WriteDefault)
			}
		}
	case *iceberg.MapType:
		if field.InitialDefault != nil {
			if _, ok := field.InitialDefault.(map[string]any); !ok {
				return fmt.Errorf("map type field '%s' (id: %d) must have null or JSON object initial-default, but got: %v", field.Name, field.ID, field.InitialDefault)
			}
		}
		if field.WriteDefault != nil {
			if _, ok := field.WriteDefault.(map[string]any); !ok {
				return fmt.Errorf("map type field '%s' (id: %d) must have null or JSON object write-default, but got: %v", field.Name, field.ID, field.WriteDefault)
			}
		}
	}

	return nil
}
