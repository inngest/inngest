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

package iceberg

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/apache/iceberg-go/internal"
)

type MappedField struct {
	Names []string `json:"names"`
	// iceberg spec says this is optional, but I don't see any examples
	// of this being left empty. Does pyiceberg need to be updated or should
	// the spec not say field-id is optional?
	FieldID *int          `json:"field-id,omitempty"`
	Fields  []MappedField `json:"fields,omitempty"`
}

func (m *MappedField) ID() int {
	if m.FieldID == nil {
		return -1
	}

	return *m.FieldID
}

func (m *MappedField) GetField(field string) *MappedField {
	for _, f := range m.Fields {
		if slices.Contains(f.Names, field) {
			return &f
		}
	}

	return nil
}

func (m *MappedField) Len() int { return len(m.Fields) }

func (m *MappedField) String() string {
	var bldr strings.Builder
	bldr.WriteString("([")
	bldr.WriteString(strings.Join(m.Names, ", "))
	bldr.WriteString("] -> ")

	if m.FieldID != nil {
		bldr.WriteString(strconv.Itoa(*m.FieldID))
	} else {
		bldr.WriteByte('?')
	}

	if len(m.Fields) > 0 {
		bldr.WriteByte(' ')
		for i, f := range m.Fields {
			if i != 0 {
				bldr.WriteString(", ")
			}
			bldr.WriteString(f.String())
		}
	}

	bldr.WriteByte(')')

	return bldr.String()
}

type NameMapping []MappedField

func (nm NameMapping) String() string {
	var bldr strings.Builder
	bldr.WriteString("[\n")
	for _, f := range nm {
		bldr.WriteByte('\t')
		bldr.WriteString(f.String())
		bldr.WriteByte('\n')
	}
	bldr.WriteByte(']')

	return bldr.String()
}

type NameMappingVisitor[S, T any] interface {
	Mapping(nm NameMapping, fieldResults S) S
	Fields(st []MappedField, fieldResults []T) S
	Field(field MappedField, fieldResult S) T
}

func VisitNameMapping[S, T any](obj NameMapping, visitor NameMappingVisitor[S, T]) (res S, err error) {
	if obj == nil {
		err = fmt.Errorf("%w: cannot visit nil NameMapping", ErrInvalidArgument)

		return res, err
	}

	defer internal.RecoverError(&err)

	return visitor.Mapping(obj, visitMappedFields([]MappedField(obj), visitor)), err
}

func VisitMappedFields[S, T any](fields []MappedField, visitor NameMappingVisitor[S, T]) (res S, err error) {
	defer internal.RecoverError(&err)

	return visitMappedFields(fields, visitor), err
}

func visitMappedFields[S, T any](fields []MappedField, visitor NameMappingVisitor[S, T]) S {
	results := make([]T, len(fields))
	for i, f := range fields {
		results[i] = visitor.Field(f, visitMappedFields(f.Fields, visitor))
	}

	return visitor.Fields(fields, results)
}

// UpdateNameMapping performs incremental updates to an existing NameMapping, preserving
// backward compatibility by maintaining existing field name mappings while adding new ones.
// This is different from createMappingFromSchema which creates a completely new mapping
// and loses all historical field name mappings.
//
// For example, when updating a field name:
//
//	Original: {FieldID: 1, Names: ["foo"]}
//	After update: {FieldID: 1, Names: ["foo", "foo_update"]}
//
// This preserves compatibility with existing data files that reference the old field names.
func UpdateNameMapping(nameMapping NameMapping, updates map[int]NestedField, adds map[int][]NestedField) (NameMapping, error) {
	result, err := VisitNameMapping(nameMapping, &updateNameMappingVisitor{updates: updates, adds: adds})
	if err != nil {
		return nil, err
	}

	return NameMapping(result), nil
}

type updateNameMappingVisitor struct {
	updates map[int]NestedField
	adds    map[int][]NestedField
}

func (u *updateNameMappingVisitor) Mapping(nm NameMapping, fieldResults []MappedField) []MappedField {
	return u.addNewFields(fieldResults, -1)
}

func (u *updateNameMappingVisitor) Fields(st []MappedField, fieldResults []MappedField) []MappedField {
	reassignments := make(map[string]int)
	for _, field := range fieldResults {
		if field.FieldID != nil {
			if update, exists := u.updates[*field.FieldID]; exists {
				reassignments[update.Name] = update.ID
			}
		}
	}

	var result []MappedField
	for _, field := range fieldResults {
		updatedField := u.removeReassignedNames(field, reassignments)
		if updatedField != nil {
			result = append(result, *updatedField)
		}
	}

	return result
}

func (u *updateNameMappingVisitor) Field(field MappedField, fieldResult []MappedField) MappedField {
	if field.FieldID == nil {
		return MappedField{
			Names:  slices.Clone(field.Names),
			Fields: fieldResult,
		}
	}

	fieldID := *field.FieldID
	fieldNames := slices.Clone(field.Names)
	if update, exists := u.updates[*field.FieldID]; exists && !slices.Contains(fieldNames, update.Name) {
		fieldNames = append(fieldNames, update.Name)
	}

	return MappedField{
		FieldID: &fieldID,
		Names:   fieldNames,
		Fields:  u.addNewFields(fieldResult, *field.FieldID),
	}
}

func (u *updateNameMappingVisitor) removeReassignedNames(field MappedField, assignments map[string]int) *MappedField {
	removedNames := make(map[string]struct{})
	for _, name := range field.Names {
		assignedID, exists := assignments[name]
		if exists && (field.FieldID == nil || assignedID != *field.FieldID) {
			removedNames[name] = struct{}{}
		}
	}

	remainingNames := make([]string, 0, len(field.Names))
	for _, name := range field.Names {
		if _, exists := removedNames[name]; !exists {
			remainingNames = append(remainingNames, name)
		}
	}

	if len(remainingNames) == 0 {
		return nil
	}

	return &MappedField{
		Names:   remainingNames,
		FieldID: field.FieldID,
		Fields:  field.Fields,
	}
}

func (u *updateNameMappingVisitor) addNewFields(mappedFields []MappedField, parentID int) []MappedField {
	fieldsToAdd, exists := u.adds[parentID]
	if !exists {
		return mappedFields
	}

	newFields := make([]MappedField, 0, len(fieldsToAdd))
	for _, add := range fieldsToAdd {
		fields := visitField(add, createMapping{})
		if len(fields) == 0 {
			fields = nil
		}

		newFields = append(newFields, MappedField{
			FieldID: &add.ID,
			Names:   []string{add.Name},
			Fields:  fields,
		})
	}
	reassignments := make(map[string]int)
	for _, add := range fieldsToAdd {
		reassignments[add.Name] = add.ID
	}

	fields := make([]MappedField, 0)
	for _, field := range mappedFields {
		if updatedField := u.removeReassignedNames(field, reassignments); updatedField != nil {
			fields = append(fields, *updatedField)
		}
	}

	return append(fields, newFields...)
}

type NameMappingAccessor struct{}

func (NameMappingAccessor) SchemaPartner(partner *MappedField) *MappedField {
	return partner
}

func (NameMappingAccessor) FieldPartner(partnerStruct *MappedField, _ int, fieldName string) *MappedField {
	if partnerStruct == nil {
		return nil
	}

	return partnerStruct.GetField(fieldName)
}

func (NameMappingAccessor) ListElementPartner(partnerList *MappedField) *MappedField {
	if partnerList == nil {
		return nil
	}

	return partnerList.GetField("element")
}

func (NameMappingAccessor) MapKeyPartner(partnerMap *MappedField) *MappedField {
	if partnerMap == nil {
		return nil
	}

	return partnerMap.GetField("key")
}

func (NameMappingAccessor) MapValuePartner(partnerMap *MappedField) *MappedField {
	if partnerMap == nil {
		return nil
	}

	return partnerMap.GetField("value")
}

type indexedNameMappingAccessor struct {
	fieldsByName map[*MappedField]map[string]*MappedField
}

func newIndexedNameMappingAccessor(root *MappedField) indexedNameMappingAccessor {
	accessor := indexedNameMappingAccessor{
		fieldsByName: make(map[*MappedField]map[string]*MappedField),
	}
	accessor.indexFields(root)

	return accessor
}

func (n indexedNameMappingAccessor) indexFields(field *MappedField) {
	if field == nil || len(field.Fields) == 0 {
		return
	}

	fieldsByName := make(map[string]*MappedField, len(field.Fields))
	n.fieldsByName[field] = fieldsByName

	for i := range field.Fields {
		mappedField := &field.Fields[i]
		for _, name := range mappedField.Names {
			if _, exists := fieldsByName[name]; !exists {
				fieldsByName[name] = mappedField
			}
		}

		n.indexFields(mappedField)
	}
}

func (n indexedNameMappingAccessor) getField(partnerStruct *MappedField, fieldName string) *MappedField {
	if partnerStruct == nil {
		return nil
	}

	if fieldsByName, ok := n.fieldsByName[partnerStruct]; ok {
		return fieldsByName[fieldName]
	}

	return partnerStruct.GetField(fieldName)
}

func (n indexedNameMappingAccessor) SchemaPartner(partner *MappedField) *MappedField {
	return partner
}

func (n indexedNameMappingAccessor) FieldPartner(partnerStruct *MappedField, _ int, fieldName string) *MappedField {
	return n.getField(partnerStruct, fieldName)
}

func (n indexedNameMappingAccessor) ListElementPartner(partnerList *MappedField) *MappedField {
	return n.getField(partnerList, "element")
}

func (n indexedNameMappingAccessor) MapKeyPartner(partnerMap *MappedField) *MappedField {
	return n.getField(partnerMap, "key")
}

func (n indexedNameMappingAccessor) MapValuePartner(partnerMap *MappedField) *MappedField {
	return n.getField(partnerMap, "value")
}

type nameMapProjectVisitor struct {
	currentPath []string
	accessor    indexedNameMappingAccessor
}

func (n *nameMapProjectVisitor) popPath() {
	n.currentPath = n.currentPath[:len(n.currentPath)-1]
}

func (n *nameMapProjectVisitor) BeforeField(f NestedField, _ *MappedField) {
	n.currentPath = append(n.currentPath, f.Name)
}

func (n *nameMapProjectVisitor) AfterField(NestedField, *MappedField) {
	n.popPath()
}

func (n *nameMapProjectVisitor) BeforeListElement(NestedField, *MappedField) {
	n.currentPath = append(n.currentPath, "element")
}

func (n *nameMapProjectVisitor) AfterListElement(NestedField, *MappedField) {
	n.popPath()
}

func (n *nameMapProjectVisitor) BeforeMapKey(NestedField, *MappedField) {
	n.currentPath = append(n.currentPath, "key")
}

func (n *nameMapProjectVisitor) AfterMapKey(NestedField, *MappedField) {
	n.popPath()
}

func (n *nameMapProjectVisitor) BeforeMapValue(NestedField, *MappedField) {
	n.currentPath = append(n.currentPath, "value")
}

func (n *nameMapProjectVisitor) AfterMapValue(NestedField, *MappedField) {
	n.popPath()
}

func (n *nameMapProjectVisitor) Schema(_ *Schema, _ *MappedField, structResult NestedField) NestedField {
	return structResult
}

func (n *nameMapProjectVisitor) Struct(_ StructType, _ *MappedField, fieldResults []NestedField) NestedField {
	return NestedField{
		Type: &StructType{FieldList: fieldResults},
	}
}

func (n *nameMapProjectVisitor) Field(field NestedField, fieldPartner *MappedField, fieldResult NestedField) NestedField {
	if fieldPartner == nil {
		panic(fmt.Errorf("%w: field missing from name mapping: %s",
			ErrInvalidArgument, strings.Join(n.currentPath, ".")))
	}

	return NestedField{
		ID:             *fieldPartner.FieldID,
		Name:           field.Name,
		Type:           fieldResult.Type,
		Required:       field.Required,
		Doc:            field.Doc,
		InitialDefault: field.InitialDefault,
		WriteDefault:   field.WriteDefault,
	}
}

func (n nameMapProjectVisitor) mappedFieldID(mapped *MappedField, name string) int {
	field := n.accessor.getField(mapped, name)
	if field != nil && field.FieldID != nil {
		return *field.FieldID
	}

	return -1
}

func (n *nameMapProjectVisitor) List(lt ListType, listPartner *MappedField, elemResult NestedField) NestedField {
	if listPartner == nil {
		panic(fmt.Errorf("%w: field missing from name mapping: %s",
			ErrInvalidArgument, strings.Join(n.currentPath, ".")))
	}

	elementID := n.mappedFieldID(listPartner, "element")

	return NestedField{
		Type: &ListType{
			ElementID:       elementID,
			Element:         elemResult.Type,
			ElementRequired: lt.ElementRequired,
		},
	}
}

func (n *nameMapProjectVisitor) Map(m MapType, mapPartner *MappedField, keyResult, valResult NestedField) NestedField {
	if mapPartner == nil {
		panic(fmt.Errorf("%w: field missing from name mapping: %s",
			ErrInvalidArgument, strings.Join(n.currentPath, ".")))
	}

	keyID := n.mappedFieldID(mapPartner, "key")
	valID := n.mappedFieldID(mapPartner, "value")

	return NestedField{
		Type: &MapType{
			KeyID:         keyID,
			KeyType:       keyResult.Type,
			ValueID:       valID,
			ValueType:     valResult.Type,
			ValueRequired: m.ValueRequired,
		},
	}
}

func (n *nameMapProjectVisitor) Primitive(p PrimitiveType, primitivePartner *MappedField) NestedField {
	if primitivePartner == nil {
		panic(fmt.Errorf("%w: field missing from name mapping: %s",
			ErrInvalidArgument, strings.Join(n.currentPath, ".")))
	}

	return NestedField{Type: p}
}

func (n *nameMapProjectVisitor) Variant(v VariantType, variantPartner *MappedField) NestedField {
	if variantPartner == nil {
		panic(fmt.Errorf("%w: field missing from name mapping: %s",
			ErrInvalidArgument, strings.Join(n.currentPath, ".")))
	}

	return NestedField{Type: v}
}

func ApplyNameMapping(schemaWithoutIDs *Schema, nameMapping NameMapping) (*Schema, error) {
	partner := &MappedField{Fields: nameMapping}
	accessor := newIndexedNameMappingAccessor(partner)
	top, err := VisitSchemaWithPartner(schemaWithoutIDs,
		partner,
		&nameMapProjectVisitor{
			currentPath: make([]string, 0, 1),
			accessor:    accessor,
		},
		accessor)
	if err != nil {
		return nil, err
	}

	return NewSchema(schemaWithoutIDs.ID,
		top.Type.(*StructType).FieldList...), nil
}

type createMapping struct{}

func (createMapping) Schema(_ *Schema, result []MappedField) []MappedField {
	return result
}

func (createMapping) Struct(st StructType, result [][]MappedField) []MappedField {
	output := make([]MappedField, len(st.FieldList))
	for i, field := range st.FieldList {
		output[i] = MappedField{
			Names:   []string{field.Name},
			FieldID: &field.ID,
			Fields:  result[i],
		}
	}

	return output
}

func (createMapping) Field(_ NestedField, result []MappedField) []MappedField {
	return result
}

func (createMapping) List(listType ListType, elemResult []MappedField) []MappedField {
	return []MappedField{{
		Names:   []string{"element"},
		FieldID: &listType.ElementID,
		Fields:  elemResult,
	}}
}

func (createMapping) Map(mapType MapType, keyResult, valResult []MappedField) []MappedField {
	return []MappedField{
		{
			Names:   []string{"key"},
			FieldID: &mapType.KeyID,
			Fields:  keyResult,
		},
		{
			Names:   []string{"value"},
			FieldID: &mapType.ValueID,
			Fields:  valResult,
		},
	}
}

func (createMapping) Primitive(_ PrimitiveType) []MappedField {
	return []MappedField{}
}

func (createMapping) Variant(_ VariantType) []MappedField {
	return []MappedField{}
}

func createMappingFromSchema(schema *Schema) NameMapping {
	result, _ := Visit(schema, createMapping{})

	return NameMapping(result)
}
