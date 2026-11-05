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
	"bytes"
	"encoding/json"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/apache/iceberg-go/internal"
)

// Schema is an Iceberg table schema, represented as a struct with
// multiple fields. The fields are only exported via accessor methods
// rather than exposing the slice directly in order to ensure a schema
// as immutable.
type Schema struct {
	ID                 int   `json:"schema-id"`
	IdentifierFieldIDs []int `json:"identifier-field-ids"`

	fields []NestedField

	// the following maps are lazily populated as needed.
	// rather than have lock contention with a mutex, we can use
	// atomic pointers to Store/Load the values.
	idToName      atomic.Pointer[map[int]string]
	idToField     atomic.Pointer[map[int]NestedField]
	nameToID      atomic.Pointer[map[string]int]
	nameToIDLower atomic.Pointer[map[string]int]
	idToAccessor  atomic.Pointer[map[int]accessor]

	lazyIDToParent  func() (map[int]int, error)
	lazyNameMapping func() NameMapping
}

// NewSchemaFromJsonFields constructs a new schema with the provided ID and fields in json form
func NewSchemaFromJsonFields(id int, jsonFieldsStr string) (*Schema, error) {
	var fields []NestedField
	err := json.Unmarshal([]byte(jsonFieldsStr), &fields)
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	return NewSchema(id, fields...), nil
}

// NewSchema constructs a new schema with the provided ID
// and list of fields.
func NewSchema(id int, fields ...NestedField) *Schema {
	return NewSchemaWithIdentifiers(id, []int{}, fields...)
}

// NewSchemaWithIdentifiers constructs a new schema with the provided ID
// and fields, along with a slice of field IDs to be listed as identifier
// fields.
func NewSchemaWithIdentifiers(id int, identifierIDs []int, fields ...NestedField) *Schema {
	s := &Schema{ID: id, fields: fields, IdentifierFieldIDs: identifierIDs}
	s.init()

	// Eagerly check for duplicate field IDs, mirroring Java's Schema.<init>.
	// We use a lightweight walker that only inspects IDs and does not
	// dereference field types, so schemas with nil types (used in some
	// tests) are not affected.
	if err := checkDuplicateFieldIDs(nil, fields); err != nil {
		panic(err)
	}

	return s
}

// checkDuplicateFieldIDs recursively walks a list of fields and returns an
// error if any field ID appears more than once anywhere in the tree. The
// seen map is lazily created on the first call so the top-level caller can
// pass nil.
func checkDuplicateFieldIDs(seen map[int]string, fields []NestedField) error {
	if seen == nil {
		seen = make(map[int]string)
	}
	for _, f := range fields {
		// Negative IDs (e.g. -1) are used as unassigned placeholders during
		// schema construction; only positive IDs must be unique per the spec.
		if f.ID > 0 {
			if prev, ok := seen[f.ID]; ok {
				return fmt.Errorf("%w: multiple fields for id %d: %s and %s",
					ErrInvalidSchema, f.ID, prev, f.Name)
			}
			seen[f.ID] = f.Name
		}

		// Recurse into nested types without touching the type pointer for
		// primitives — only StructType, ListType, and MapType carry child fields.
		switch t := f.Type.(type) {
		case *StructType:
			if t != nil {
				if err := checkDuplicateFieldIDs(seen, t.FieldList); err != nil {
					return err
				}
			}
		case *ListType:
			if t != nil {
				elem := t.ElementField()
				if err := checkDuplicateFieldIDs(seen, []NestedField{elem}); err != nil {
					return err
				}
			}
		case *MapType:
			if t != nil {
				if err := checkDuplicateFieldIDs(seen, []NestedField{t.KeyField(), t.ValueField()}); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (s *Schema) init() {
	s.lazyIDToParent = sync.OnceValues(func() (map[int]int, error) {
		return IndexParents(s)
	})
	s.lazyNameMapping = sync.OnceValue(func() NameMapping {
		return createMappingFromSchema(s)
	})
}

func (s *Schema) String() string {
	var b strings.Builder
	b.WriteString("table {")
	for _, f := range s.fields {
		b.WriteString("\n\t")
		b.WriteString(f.String())
	}
	b.WriteString("\n}")

	return b.String()
}

func (s *Schema) lazyNameToID() (map[string]int, error) {
	index := s.nameToID.Load()
	if index != nil {
		return *index, nil
	}

	idx, err := IndexByName(s)
	if err != nil {
		return nil, err
	}

	s.nameToID.Store(&idx)

	return idx, nil
}

// FlatFields returns an iterator over the flattened fields in the schema
// The fields are returned in arbitrary order.
func (s *Schema) FlatFields() (iter.Seq[NestedField], error) {
	fields, err := s.lazyIDToField()
	if err != nil {
		return nil, err
	}

	return func(yield func(NestedField) bool) {
		for field := range maps.Values(fields) {
			if !yield(cloneField(field)) {
				return
			}
		}
	}, nil
}

func (s *Schema) lazyIDToField() (map[int]NestedField, error) {
	index := s.idToField.Load()
	if index != nil {
		return *index, nil
	}

	idx, err := IndexByID(s)
	if err != nil {
		return nil, err
	}

	s.idToField.Store(&idx)

	return idx, nil
}

func (s *Schema) lazyIDToName() (map[int]string, error) {
	index := s.idToName.Load()
	if index != nil {
		return *index, nil
	}

	idx, err := IndexNameByID(s)
	if err != nil {
		return nil, err
	}

	s.idToName.Store(&idx)

	return idx, nil
}

func (s *Schema) lazyNameToIDLower() (map[string]int, error) {
	index := s.nameToIDLower.Load()
	if index != nil {
		return *index, nil
	}

	idx, err := s.lazyNameToID()
	if err != nil {
		return nil, err
	}

	out := make(map[string]int)
	for k, v := range idx {
		out[strings.ToLower(k)] = v
	}

	s.nameToIDLower.Store(&out)

	return out, nil
}

func (s *Schema) lazyIdToAccessor() (map[int]accessor, error) {
	index := s.idToAccessor.Load()
	if index != nil {
		return *index, nil
	}

	idx, err := buildAccessors(s)
	if err != nil {
		return nil, err
	}

	s.idToAccessor.Store(&idx)

	return idx, nil
}

func (s *Schema) NameMapping() NameMapping { return s.lazyNameMapping() }

func (s *Schema) Type() string { return "struct" }

// AsStruct returns a Struct with the same fields as the schema which can
// then be used as a Type.
func (s *Schema) AsStruct() StructType { return StructType{FieldList: cloneFields(s.fields)} }

// asStructRef returns a read-only view over the schema field slice for
// internal callers that only traverse fields and must avoid clone-on-read
// overhead on hot paths.
func (s *Schema) asStructRef() StructType { return StructType{FieldList: s.fields} }

func (s *Schema) NumFields() int          { return len(s.fields) }
func (s *Schema) Field(i int) NestedField { return cloneField(s.fields[i]) }
func (s *Schema) Fields() []NestedField   { return cloneFields(s.fields) }

// FieldsRef returns the schema-owned fields for trusted internal callers.
// The returned slice and everything reachable through the fields must be
// treated as read-only.
func (s *Schema) FieldsRef(_ internal.SchemaRef) []NestedField { return s.fields }

func (s *Schema) FieldIDs() []int {
	idx, _ := s.lazyNameToID()

	return slices.Collect(maps.Values(idx))
}

func (s *Schema) UnmarshalJSON(b []byte) error {
	type Alias Schema
	var decoded Schema
	aux := struct {
		Type   *string         `json:"type"`
		Fields json.RawMessage `json:"fields"`
		*Alias
	}{Alias: (*Alias)(&decoded)}

	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	if aux.Type == nil {
		return fmt.Errorf("%w: schema type is required", ErrInvalidSchema)
	}
	if *aux.Type != "struct" {
		return fmt.Errorf("%w: schema type must be struct, got %q", ErrInvalidSchema, *aux.Type)
	}
	fieldsJSON := bytes.TrimSpace(aux.Fields)
	if len(fieldsJSON) == 0 || bytes.Equal(fieldsJSON, []byte("null")) {
		return fmt.Errorf("%w: schema fields are required", ErrInvalidSchema)
	}

	var fields []NestedField
	if err := json.Unmarshal(fieldsJSON, &fields); err != nil {
		return fmt.Errorf("%w: invalid schema fields: %w", ErrInvalidSchema, err)
	}

	if err := checkDuplicateFieldIDs(nil, fields); err != nil {
		return err
	}

	decoded.fields = fields
	if decoded.IdentifierFieldIDs == nil {
		decoded.IdentifierFieldIDs = []int{}
	}

	s.ID = decoded.ID
	s.IdentifierFieldIDs = decoded.IdentifierFieldIDs
	s.fields = decoded.fields
	s.idToName.Store(nil)
	s.idToField.Store(nil)
	s.nameToID.Store(nil)
	s.nameToIDLower.Store(nil)
	s.idToAccessor.Store(nil)
	s.init()

	return nil
}

func (s *Schema) MarshalJSON() ([]byte, error) {
	ids := s.IdentifierFieldIDs
	if ids == nil {
		ids = []int{}
	}
	fields := s.fields
	if fields == nil {
		fields = []NestedField{}
	}

	type Alias Schema

	// Keep this literal in sync with every JSON-marshaled Schema field. Copying
	// the whole Schema would also copy its lazy atomic caches after they are used.
	aliasCopy := Alias{ID: s.ID, IdentifierFieldIDs: ids}

	return json.Marshal(struct {
		Type   string        `json:"type"`
		Fields []NestedField `json:"fields"`
		*Alias
	}{Type: "struct", Fields: fields, Alias: &aliasCopy})
}

func cloneFields(fields []NestedField) []NestedField {
	if len(fields) == 0 {
		return nil
	}

	cloned := make([]NestedField, len(fields))
	for i, field := range fields {
		cloned[i] = cloneField(field)
	}

	return cloned
}

func cloneField(field NestedField) NestedField {
	return NestedField{
		ID:             field.ID,
		Name:           field.Name,
		Type:           cloneType(field.Type),
		Required:       field.Required,
		Doc:            field.Doc,
		InitialDefault: CloneDefaultValue(field.InitialDefault),
		WriteDefault:   CloneDefaultValue(field.WriteDefault),
	}
}

func cloneType(t Type) Type {
	if t == nil {
		return nil
	}

	switch typed := t.(type) {
	case *StructType:
		return &StructType{FieldList: cloneFields(typed.FieldList)}
	case *ListType:
		cloned := *typed
		cloned.Element = cloneType(typed.Element)

		return &cloned
	case *MapType:
		cloned := *typed
		cloned.KeyType = cloneType(typed.KeyType)
		cloned.ValueType = cloneType(typed.ValueType)

		return &cloned
	default:
		// Leaf and parameterized non-container types are immutable value objects,
		// so sharing them between clones is safe.
		return t
	}
}

// FindColumnName returns the name of the column identified by the
// passed in field id. The second return value reports whether or
// not the field id was found in the schema.
func (s *Schema) FindColumnName(fieldID int) (string, bool) {
	idx, _ := s.lazyIDToName()
	col, ok := idx[fieldID]

	return col, ok
}

// FindFieldByName returns the field identified by the name given,
// the second return value will be false if no field by this name
// is found.
//
// Note: This search is done in a case sensitive manner. To perform
// a case insensitive search, use [*Schema.FindFieldByNameCaseInsensitive].
func (s *Schema) FindFieldByName(name string) (NestedField, bool) {
	idx, err := s.lazyNameToID()
	if err != nil {
		return NestedField{}, false
	}

	id, ok := idx[name]
	if !ok {
		return NestedField{}, false
	}

	return s.FindFieldByID(id)
}

// FindFieldByNameCaseInsensitive is like [*Schema.FindFieldByName],
// but performs a case insensitive search.
func (s *Schema) FindFieldByNameCaseInsensitive(name string) (NestedField, bool) {
	idx, err := s.lazyNameToIDLower()
	if err != nil {
		return NestedField{}, false
	}

	id, ok := idx[strings.ToLower(name)]
	if !ok {
		return NestedField{}, false
	}

	return s.FindFieldByID(id)
}

// FindFieldByID is like [*Schema.FindColumnName], but returns the whole
// field rather than just the field name.
func (s *Schema) FindFieldByID(id int) (NestedField, bool) {
	f, ok := s.FindFieldByIDRef(id, internal.SchemaRef{})
	if ok {
		f = cloneField(f)
	}

	return f, ok
}

// FindFieldByIDRef returns a schema-owned field without cloning it. The returned
// field is a value copy, but its Type and everything reachable from that Type
// are shared with the schema and must be treated as read-only. This method is
// limited to internal callers that need to avoid clone-on-read overhead.
func (s *Schema) FindFieldByIDRef(id int, _ internal.SchemaRef) (NestedField, bool) {
	idx, _ := s.lazyIDToField()
	f, ok := idx[id]

	return f, ok
}

// FindTypeByID is like [*Schema.FindFieldByID], but returns only the data
// type of the field.
func (s *Schema) FindTypeByID(id int) (Type, bool) {
	f, ok := s.FindFieldByID(id)
	if !ok {
		return nil, false
	}

	return f.Type, true
}

// FindTypeByName is a convenience function for calling [*Schema.FindFieldByName],
// and then returning just the type.
func (s *Schema) FindTypeByName(name string) (Type, bool) {
	f, ok := s.FindFieldByName(name)
	if !ok {
		return nil, false
	}

	return f.Type, true
}

// FindTypeByNameCaseInsensitive is like [*Schema.FindTypeByName] but
// performs a case insensitive search.
func (s *Schema) FindTypeByNameCaseInsensitive(name string) (Type, bool) {
	f, ok := s.FindFieldByNameCaseInsensitive(name)
	if !ok {
		return nil, false
	}

	return f.Type, true
}

func (s *Schema) accessorForField(id int) (accessor, bool) {
	idx, err := s.lazyIdToAccessor()
	if err != nil {
		return accessor{}, false
	}

	acc, ok := idx[id]

	return acc, ok
}

// columnPathSegments returns the own-name path segments from the top-level column down to id, so a name containing '.' stays one segment.
func (s *Schema) columnPathSegments(id int) []string {
	idToField, err := s.lazyIDToField()
	if err != nil {
		return nil
	}
	parents, err := s.lazyIDToParent()
	if err != nil {
		return nil
	}
	var segs []string
	for cur := id; ; {
		f, ok := idToField[cur]
		if !ok {
			return nil
		}
		segs = append([]string{f.Name}, segs...)
		p, ok := parents[cur]
		if !ok {
			break
		}
		cur = p
	}

	return segs
}

// Equals compares the fields and identifierIDs, but does not compare
// the schema ID itself.
func (s *Schema) Equals(other *Schema) bool {
	if other == nil {
		return false
	}

	if s == other {
		return true
	}

	if len(s.fields) != len(other.fields) {
		return false
	}

	if !slices.Equal(s.IdentifierFieldIDs, other.IdentifierFieldIDs) {
		return false
	}

	return slices.EqualFunc(s.fields, other.fields, func(a, b NestedField) bool {
		return a.Equals(b)
	})
}

// HighestFieldID returns the value of the numerically highest field ID
// in this schema.
func (s *Schema) HighestFieldID() int {
	id, _ := Visit(s, findLastFieldID{})

	return id
}

type Void = struct{}

var void = Void{}

// Select creates a new schema with just the fields identified by name
// passed in the order they are provided. If caseSensitive is false,
// then fields will be identified by case insensitive search.
//
// An error is returned if a requested name cannot be found.
func (s *Schema) Select(caseSensitive bool, names ...string) (*Schema, error) {
	ids := make(map[int]Void)
	if caseSensitive {
		nameMap, err := s.lazyNameToID()
		if err != nil {
			return nil, err
		}

		for _, n := range names {
			id, ok := nameMap[n]
			if !ok {
				return nil, fmt.Errorf("%w: could not find column %s", ErrInvalidSchema, n)
			}
			ids[id] = void
		}
	} else {
		nameMap, err := s.lazyNameToIDLower()
		if err != nil {
			return nil, err
		}

		for _, n := range names {
			id, ok := nameMap[strings.ToLower(n)]
			if !ok {
				return nil, fmt.Errorf("%w: could not find column %s", ErrInvalidSchema, n)
			}
			ids[id] = void
		}
	}

	return PruneColumns(s, ids, true)
}

func (s *Schema) FieldHasOptionalParent(id int) bool {
	idToParent, _ := s.lazyIDToParent()
	idToField, _ := s.lazyIDToField()

	f, ok := idToField[id]
	if !ok {
		return false
	}

	for {
		parent, ok := idToParent[f.ID]
		if !ok {
			return false
		}

		if f = idToField[parent]; !f.Required {
			return true
		}
	}
}

// SchemaVisitor is an interface that can be implemented to allow for
// easy traversal and processing of a schema.
//
// A SchemaVisitor can also optionally implement the Before/After Field,
// ListElement, MapKey, or MapValue interfaces to allow them to get called
// at the appropriate points within schema traversal.
type SchemaVisitor[T any] interface {
	Schema(schema *Schema, structResult T) T
	Struct(st StructType, fieldResults []T) T
	Field(field NestedField, fieldResult T) T
	List(list ListType, elemResult T) T
	Map(mapType MapType, keyResult, valueResult T) T
	Primitive(p PrimitiveType) T
	Variant(v VariantType) T
}

type BeforeFieldVisitor interface {
	BeforeField(field NestedField)
}

type AfterFieldVisitor interface {
	AfterField(field NestedField)
}

type BeforeListElementVisitor interface {
	BeforeListElement(elem NestedField)
}

type AfterListElementVisitor interface {
	AfterListElement(elem NestedField)
}

type BeforeMapKeyVisitor interface {
	BeforeMapKey(key NestedField)
}

type AfterMapKeyVisitor interface {
	AfterMapKey(key NestedField)
}

type BeforeMapValueVisitor interface {
	BeforeMapValue(value NestedField)
}

type AfterMapValueVisitor interface {
	AfterMapValue(value NestedField)
}

type SchemaVisitorPerPrimitiveType[T any] interface {
	SchemaVisitor[T]

	VisitFixed(FixedType) T
	VisitDecimal(DecimalType) T
	VisitBoolean() T
	VisitInt32() T
	VisitInt64() T
	VisitFloat32() T
	VisitFloat64() T
	VisitDate() T
	VisitTime() T
	VisitTimestamp() T
	VisitTimestampNs() T
	VisitTimestampTz() T
	VisitTimestampNsTz() T
	VisitString() T
	VisitBinary() T
	VisitUUID() T
	VisitUnknown() T
	VisitVariant() T
	VisitGeometry(GeometryType) T
	VisitGeography(GeographyType) T
}

// Visit accepts a visitor and performs a post-order traversal of the given schema.
func Visit[T any](sc *Schema, visitor SchemaVisitor[T]) (res T, err error) {
	if sc == nil {
		err = fmt.Errorf("%w: cannot visit nil schema", ErrInvalidArgument)

		return res, err
	}

	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case string:
				err = fmt.Errorf("%w: %s", ErrInvalidSchema, e)
			case error:
				err = fmt.Errorf("error encountered during schema visitor: %w", e)
			}
		}
	}()

	return visitor.Schema(sc, visitStruct(sc.asStructRef(), visitor)), nil
}

func visitStruct[T any](obj StructType, visitor SchemaVisitor[T]) T {
	results := make([]T, len(obj.FieldList))

	bf, _ := visitor.(BeforeFieldVisitor)
	af, _ := visitor.(AfterFieldVisitor)

	for i, f := range obj.FieldList {
		if bf != nil {
			bf.BeforeField(f)
		}

		res := visitField(f, visitor)

		if af != nil {
			af.AfterField(f)
		}

		results[i] = visitor.Field(f, res)
	}

	return visitor.Struct(obj, results)
}

func visitList[T any](obj ListType, visitor SchemaVisitor[T]) T {
	elemField := obj.ElementField()

	if bl, ok := visitor.(BeforeListElementVisitor); ok {
		bl.BeforeListElement(elemField)
	} else if bf, ok := visitor.(BeforeFieldVisitor); ok {
		bf.BeforeField(elemField)
	}

	res := visitField(elemField, visitor)

	if al, ok := visitor.(AfterListElementVisitor); ok {
		al.AfterListElement(elemField)
	} else if af, ok := visitor.(AfterFieldVisitor); ok {
		af.AfterField(elemField)
	}

	return visitor.List(obj, res)
}

func visitMap[T any](obj MapType, visitor SchemaVisitor[T]) T {
	keyField, valueField := obj.KeyField(), obj.ValueField()

	if bmk, ok := visitor.(BeforeMapKeyVisitor); ok {
		bmk.BeforeMapKey(keyField)
	} else if bf, ok := visitor.(BeforeFieldVisitor); ok {
		bf.BeforeField(keyField)
	}

	keyRes := visitField(keyField, visitor)

	if amk, ok := visitor.(AfterMapKeyVisitor); ok {
		amk.AfterMapKey(keyField)
	} else if af, ok := visitor.(AfterFieldVisitor); ok {
		af.AfterField(keyField)
	}

	if bmk, ok := visitor.(BeforeMapValueVisitor); ok {
		bmk.BeforeMapValue(valueField)
	} else if bf, ok := visitor.(BeforeFieldVisitor); ok {
		bf.BeforeField(valueField)
	}

	valueRes := visitField(valueField, visitor)

	if amk, ok := visitor.(AfterMapValueVisitor); ok {
		amk.AfterMapValue(valueField)
	} else if af, ok := visitor.(AfterFieldVisitor); ok {
		af.AfterField(valueField)
	}

	return visitor.Map(obj, keyRes, valueRes)
}

func visitField[T any](f NestedField, visitor SchemaVisitor[T]) T {
	switch typ := f.Type.(type) {
	case *StructType:
		return visitStruct(*typ, visitor)
	case *ListType:
		return visitList(*typ, visitor)
	case *MapType:
		return visitMap(*typ, visitor)
	case VariantType:
		if perPrimitive, ok := visitor.(SchemaVisitorPerPrimitiveType[T]); ok {
			return perPrimitive.VisitVariant()
		}

		return visitor.Variant(typ)
	default: // primitive
		if perPrimitive, ok := visitor.(SchemaVisitorPerPrimitiveType[T]); ok {
			switch t := typ.(type) {
			case BooleanType:
				return perPrimitive.VisitBoolean()
			case Int32Type:
				return perPrimitive.VisitInt32()
			case Int64Type:
				return perPrimitive.VisitInt64()
			case Float32Type:
				return perPrimitive.VisitFloat32()
			case Float64Type:
				return perPrimitive.VisitFloat64()
			case DateType:
				return perPrimitive.VisitDate()
			case TimeType:
				return perPrimitive.VisitTime()
			case TimestampType:
				return perPrimitive.VisitTimestamp()
			case TimestampNsType:
				return perPrimitive.VisitTimestampNs()
			case TimestampTzType:
				return perPrimitive.VisitTimestampTz()
			case TimestampTzNsType:
				return perPrimitive.VisitTimestampNsTz()
			case StringType:
				return perPrimitive.VisitString()
			case BinaryType:
				return perPrimitive.VisitBinary()
			case UUIDType:
				return perPrimitive.VisitUUID()
			case DecimalType:
				return perPrimitive.VisitDecimal(t)
			case FixedType:
				return perPrimitive.VisitFixed(t)
			case UnknownType:
				return perPrimitive.VisitUnknown()
			case GeographyType:
				return perPrimitive.VisitGeography(t)
			case GeometryType:
				return perPrimitive.VisitGeometry(t)
			}
		}

		return visitor.Primitive(typ.(PrimitiveType))
	}
}

type PreOrderSchemaVisitor[T any] interface {
	Schema(*Schema, func() T) T
	Struct(StructType, []func() T) T
	Field(NestedField, func() T) T
	List(ListType, func() T) T
	Map(MapType, func() T, func() T) T
	Primitive(PrimitiveType) T
	Variant(VariantType) T
}

func PreOrderVisit[T any](sc *Schema, visitor PreOrderSchemaVisitor[T]) (res T, err error) {
	if sc == nil {
		err = fmt.Errorf("%w: cannot visit nil schema", ErrInvalidArgument)

		return res, err
	}

	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case string:
				err = fmt.Errorf("%w: %s", ErrInvalidSchema, e)
			case error:
				err = fmt.Errorf("error encountered during schema visitor: %w", e)
			}
		}
	}()

	return visitor.Schema(sc, func() T {
		return visitStructPreOrder(sc.asStructRef(), visitor)
	}), nil
}

func visitStructPreOrder[T any](obj StructType, visitor PreOrderSchemaVisitor[T]) T {
	results := make([]func() T, len(obj.FieldList))

	for i, f := range obj.FieldList {
		results[i] = func() T {
			return visitFieldPreOrder(f, visitor)
		}
	}

	return visitor.Struct(obj, results)
}

func visitListPreOrder[T any](obj ListType, visitor PreOrderSchemaVisitor[T]) T {
	return visitor.List(obj, func() T {
		return visitFieldPreOrder(obj.ElementField(), visitor)
	})
}

func visitMapPreOrder[T any](obj MapType, visitor PreOrderSchemaVisitor[T]) T {
	return visitor.Map(obj, func() T {
		return visitFieldPreOrder(obj.KeyField(), visitor)
	}, func() T {
		return visitFieldPreOrder(obj.ValueField(), visitor)
	})
}

func visitFieldPreOrder[T any](f NestedField, visitor PreOrderSchemaVisitor[T]) T {
	var fn func() T
	switch typ := f.Type.(type) {
	case *StructType:
		fn = func() T { return visitStructPreOrder(*typ, visitor) }
	case *ListType:
		fn = func() T { return visitListPreOrder(*typ, visitor) }
	case *MapType:
		fn = func() T { return visitMapPreOrder(*typ, visitor) }
	case VariantType:
		fn = func() T { return visitor.Variant(typ) }
	default:
		fn = func() T { return visitor.Primitive(typ.(PrimitiveType)) }
	}

	return visitor.Field(f, fn)
}

// IndexByID performs a post-order traversal of the given schema and
// returns a mapping from field ID to field.
func IndexByID(schema *Schema) (map[int]NestedField, error) {
	return Visit(schema, &indexByID{index: make(map[int]NestedField)})
}

type indexByID struct {
	index map[int]NestedField
}

func (i *indexByID) Schema(*Schema, map[int]NestedField) map[int]NestedField {
	return i.index
}

func (i *indexByID) Struct(StructType, []map[int]NestedField) map[int]NestedField {
	return i.index
}

func (i *indexByID) Field(field NestedField, _ map[int]NestedField) map[int]NestedField {
	i.index[field.ID] = field

	return i.index
}

func (i *indexByID) List(list ListType, _ map[int]NestedField) map[int]NestedField {
	i.index[list.ElementID] = list.ElementField()

	return i.index
}

func (i *indexByID) Map(mapType MapType, _, _ map[int]NestedField) map[int]NestedField {
	i.index[mapType.KeyID] = mapType.KeyField()
	i.index[mapType.ValueID] = mapType.ValueField()

	return i.index
}

func (i *indexByID) Primitive(PrimitiveType) map[int]NestedField {
	return i.index
}

func (i *indexByID) Variant(VariantType) map[int]NestedField {
	return i.index
}

// IndexByName performs a post-order traversal of the schema and returns
// a mapping from field name to field ID.
func IndexByName(schema *Schema) (map[string]int, error) {
	if schema == nil {
		return nil, fmt.Errorf("%w: cannot index nil schema", ErrInvalidArgument)
	}

	if len(schema.fields) > 0 {
		indexer := &indexByName{
			index:           make(map[string]int),
			shortNameId:     make(map[string]int),
			fieldNames:      make([]string, 0),
			shortFieldNames: make([]string, 0),
		}
		if _, err := Visit(schema, indexer); err != nil {
			return nil, err
		}

		return indexer.ByName(), nil
	}

	return map[string]int{}, nil
}

// IndexNameByID performs a post-order traversal of the schema and returns
// a mapping from field ID to field name.
func IndexNameByID(schema *Schema) (map[int]string, error) {
	indexer := &indexByName{
		index:           make(map[string]int),
		shortNameId:     make(map[string]int),
		fieldNames:      make([]string, 0),
		shortFieldNames: make([]string, 0),
	}
	if _, err := Visit(schema, indexer); err != nil {
		return nil, err
	}

	return indexer.ByID(), nil
}

type indexByName struct {
	index           map[string]int
	shortNameId     map[string]int
	combinedIndex   map[string]int
	fieldNames      []string
	shortFieldNames []string
}

func (i *indexByName) ByID() map[int]string {
	idToName := make(map[int]string)
	for k, v := range i.index {
		idToName[v] = k
	}

	return idToName
}

func (i *indexByName) ByName() map[string]int {
	i.combinedIndex = maps.Clone(i.shortNameId)
	maps.Copy(i.combinedIndex, i.index)

	return i.combinedIndex
}

func (i *indexByName) Primitive(PrimitiveType) map[string]int { return i.index }
func (i *indexByName) Variant(VariantType) map[string]int     { return i.index }
func (i *indexByName) addField(name string, fieldID int) {
	fullName := name
	if len(i.fieldNames) > 0 {
		fullName = strings.Join(i.fieldNames, ".") + "." + name
	}

	if _, ok := i.index[fullName]; ok {
		panic(fmt.Errorf("%w: multiple fields for name %s: %d and %d",
			ErrInvalidSchema, fullName, i.index[fullName], fieldID))
	}

	i.index[fullName] = fieldID
	if len(i.shortFieldNames) > 0 {
		shortName := strings.Join(i.shortFieldNames, ".") + "." + name
		i.shortNameId[shortName] = fieldID
	}
}

func (i *indexByName) Schema(*Schema, map[string]int) map[string]int {
	return i.index
}

func (i *indexByName) Struct(StructType, []map[string]int) map[string]int {
	return i.index
}

func (i *indexByName) Field(field NestedField, _ map[string]int) map[string]int {
	i.addField(field.Name, field.ID)

	return i.index
}

func (i *indexByName) List(list ListType, _ map[string]int) map[string]int {
	i.addField(list.ElementField().Name, list.ElementID)

	return i.index
}

func (i *indexByName) Map(mapType MapType, _, _ map[string]int) map[string]int {
	i.addField(mapType.KeyField().Name, mapType.KeyID)
	i.addField(mapType.ValueField().Name, mapType.ValueID)

	return i.index
}

func (i *indexByName) BeforeListElement(elem NestedField) {
	if _, ok := elem.Type.(*StructType); !ok {
		i.shortFieldNames = append(i.shortFieldNames, elem.Name)
	}
	i.fieldNames = append(i.fieldNames, elem.Name)
}

func (i *indexByName) AfterListElement(elem NestedField) {
	if _, ok := elem.Type.(*StructType); !ok {
		i.shortFieldNames = i.shortFieldNames[:len(i.shortFieldNames)-1]
	}
	i.fieldNames = i.fieldNames[:len(i.fieldNames)-1]
}

func (i *indexByName) BeforeField(field NestedField) {
	i.fieldNames = append(i.fieldNames, field.Name)
	i.shortFieldNames = append(i.shortFieldNames, field.Name)
}

func (i *indexByName) AfterField(field NestedField) {
	i.fieldNames = i.fieldNames[:len(i.fieldNames)-1]
	i.shortFieldNames = i.shortFieldNames[:len(i.shortFieldNames)-1]
}

// PruneColumns visits a schema pruning any columns which do not exist in the
// provided selected set. Parent fields of a selected child will be retained.
func PruneColumns(schema *Schema, selected map[int]Void, selectFullTypes bool) (*Schema, error) {
	result, err := Visit(schema, &pruneColVisitor{
		selected:  selected,
		fullTypes: selectFullTypes,
	})
	if err != nil {
		return nil, err
	}

	n, ok := result.(NestedType)
	if !ok {
		n = &StructType{}
	}

	newIdentifierIDs := make([]int, 0, len(schema.IdentifierFieldIDs))
	for _, id := range schema.IdentifierFieldIDs {
		if _, ok := selected[id]; ok {
			newIdentifierIDs = append(newIdentifierIDs, id)
		}
	}

	return &Schema{
		fields:             n.Fields(),
		ID:                 schema.ID,
		IdentifierFieldIDs: newIdentifierIDs,
	}, nil
}

type pruneColVisitor struct {
	selected  map[int]Void
	fullTypes bool
}

func (p *pruneColVisitor) Schema(_ *Schema, structResult Type) Type {
	return structResult
}

func (p *pruneColVisitor) Struct(st StructType, fieldResults []Type) Type {
	selected, fields := []NestedField{}, st.FieldList
	sameType := true

	for i, t := range fieldResults {
		field := fields[i]
		if field.Type == t {
			selected = append(selected, field)
		} else if t != nil {
			sameType = false
			// type has changed, create a new field with the projected type
			selected = append(selected, NestedField{
				ID:             field.ID,
				Name:           field.Name,
				Type:           t,
				Doc:            field.Doc,
				Required:       field.Required,
				InitialDefault: field.InitialDefault,
				WriteDefault:   field.WriteDefault,
			})
		}
	}

	if len(selected) > 0 {
		if len(selected) == len(fields) && sameType {
			// nothing changed, return the original
			return &st
		} else {
			return &StructType{FieldList: selected}
		}
	}

	return nil
}

func (p *pruneColVisitor) Field(field NestedField, fieldResult Type) Type {
	_, ok := p.selected[field.ID]
	if !ok {
		if fieldResult != nil {
			return fieldResult
		}

		return nil
	}

	if p.fullTypes {
		return field.Type
	}

	if _, ok := field.Type.(*StructType); ok {
		return p.projectSelectedStruct(fieldResult)
	}

	if _, ok := field.Type.(VariantType); ok {
		return field.Type
	}

	typ, ok := field.Type.(PrimitiveType)
	if !ok {
		panic(fmt.Errorf("%w: cannot explicitly project List or Map types, %d:%s of type %s was selected",
			ErrInvalidSchema, field.ID, field.Name, field.Type))
	}

	return typ
}

func (p *pruneColVisitor) List(list ListType, elemResult Type) Type {
	_, ok := p.selected[list.ElementID]
	if !ok {
		if elemResult != nil {
			return p.projectList(&list, elemResult)
		}

		return nil
	}

	if p.fullTypes {
		return &list
	}

	_, ok = list.Element.(*StructType)
	if list.Element != nil && ok {
		projected := p.projectSelectedStruct(elemResult)

		return p.projectList(&list, projected)
	}

	if _, ok = list.Element.(PrimitiveType); !ok {
		panic(fmt.Errorf("%w: cannot explicitly project List or Map types, %d of type %s was selected",
			ErrInvalidSchema, list.ElementID, list.Element))
	}

	return &list
}

func (p *pruneColVisitor) Map(mapType MapType, keyResult, valueResult Type) Type {
	_, ok := p.selected[mapType.ValueID]
	if !ok {
		if valueResult != nil {
			return p.projectMap(&mapType, valueResult)
		}

		if _, ok = p.selected[mapType.KeyID]; ok {
			return &mapType
		}

		return nil
	}

	if p.fullTypes {
		return &mapType
	}

	_, ok = mapType.ValueType.(*StructType)
	if mapType.ValueType != nil && ok {
		projected := p.projectSelectedStruct(valueResult)

		return p.projectMap(&mapType, projected)
	}

	if _, ok = mapType.ValueType.(PrimitiveType); !ok {
		panic(fmt.Errorf("%w: cannot explicitly project List or Map types, Map value %d of type %s was selected",
			ErrInvalidSchema, mapType.ValueID, mapType.ValueType))
	}

	return &mapType
}

func (p *pruneColVisitor) Primitive(_ PrimitiveType) Type { return nil }
func (p *pruneColVisitor) Variant(_ VariantType) Type     { return nil }

func (*pruneColVisitor) projectSelectedStruct(projected Type) *StructType {
	if projected == nil {
		return &StructType{}
	}

	if ty, ok := projected.(*StructType); ok {
		return ty
	}

	panic(fmt.Errorf("%w: expected a struct type, got %T", ErrInvalidSchema, projected))
}

func (*pruneColVisitor) projectList(listType *ListType, elementResult Type) *ListType {
	if listType.Element.Equals(elementResult) {
		return listType
	}

	return &ListType{
		ElementID: listType.ElementID, Element: elementResult,
		ElementRequired: listType.ElementRequired,
	}
}

func (*pruneColVisitor) projectMap(mapType *MapType, valueResult Type) *MapType {
	if mapType.ValueType.Equals(valueResult) {
		return mapType
	}

	return &MapType{
		KeyID:         mapType.KeyID,
		ValueID:       mapType.ValueID,
		KeyType:       mapType.KeyType,
		ValueType:     valueResult,
		ValueRequired: mapType.ValueRequired,
	}
}

type findLastFieldID struct{}

func (findLastFieldID) Schema(_ *Schema, result int) int {
	return result
}

func (findLastFieldID) Struct(_ StructType, fieldResults []int) int {
	return slices.Max(fieldResults)
}

func (findLastFieldID) Field(field NestedField, fieldResult int) int {
	return max(field.ID, fieldResult)
}

func (findLastFieldID) List(field ListType, elemResult int) int {
	return max(field.ElementID, elemResult)
}

func (findLastFieldID) Map(field MapType, keyResult, valueResult int) int {
	return max(field.KeyID, field.ValueID, keyResult, valueResult)
}

func (findLastFieldID) Primitive(PrimitiveType) int { return 0 }
func (findLastFieldID) Variant(VariantType) int     { return 0 }

// IndexParents generates an index of field IDs to their parent field
// IDs. Root fields are not indexed
func IndexParents(schema *Schema) (map[int]int, error) {
	indexer := &indexParents{
		idToParent: make(map[int]int),
		idStack:    make([]int, 0),
	}

	return Visit(schema, indexer)
}

type indexParents struct {
	idToParent map[int]int
	idStack    []int
}

func (i *indexParents) BeforeField(field NestedField) {
	i.idStack = append(i.idStack, field.ID)
}

func (i *indexParents) AfterField(field NestedField) {
	i.idStack = i.idStack[:len(i.idStack)-1]
}

func (i *indexParents) Schema(schema *Schema, _ map[int]int) map[int]int {
	return i.idToParent
}

func (i *indexParents) Struct(st StructType, _ []map[int]int) map[int]int {
	var parent int
	stackLen := len(i.idStack)
	if stackLen > 0 {
		parent = i.idStack[stackLen-1]
		for _, f := range st.FieldList {
			i.idToParent[f.ID] = parent
		}
	}

	return i.idToParent
}

func (i *indexParents) Field(NestedField, map[int]int) map[int]int {
	return i.idToParent
}

func (i *indexParents) List(list ListType, _ map[int]int) map[int]int {
	i.idToParent[list.ElementID] = i.idStack[len(i.idStack)-1]

	return i.idToParent
}

func (i *indexParents) Map(mapType MapType, _, _ map[int]int) map[int]int {
	parent := i.idStack[len(i.idStack)-1]
	i.idToParent[mapType.KeyID] = parent
	i.idToParent[mapType.ValueID] = parent

	return i.idToParent
}

func (i *indexParents) Primitive(PrimitiveType) map[int]int {
	return i.idToParent
}

func (i *indexParents) Variant(VariantType) map[int]int {
	return i.idToParent
}

type buildPosAccessors struct{}

func (buildPosAccessors) Schema(_ *Schema, structResult map[int]accessor) map[int]accessor {
	return structResult
}

func (buildPosAccessors) Struct(st StructType, fieldResults []map[int]accessor) map[int]accessor {
	result := map[int]accessor{}
	for pos, f := range st.FieldList {
		if innerMap := fieldResults[pos]; len(innerMap) != 0 {
			for inner, acc := range innerMap {
				result[inner] = accessor{pos: pos, inner: &acc}
			}
		} else {
			result[f.ID] = accessor{pos: pos}
		}
	}

	return result
}

func (buildPosAccessors) Field(_ NestedField, fieldResult map[int]accessor) map[int]accessor {
	return fieldResult
}

func (buildPosAccessors) List(ListType, map[int]accessor) map[int]accessor {
	return map[int]accessor{}
}

func (buildPosAccessors) Map(_ MapType, _, _ map[int]accessor) map[int]accessor {
	return map[int]accessor{}
}

func (buildPosAccessors) Primitive(PrimitiveType) map[int]accessor {
	return map[int]accessor{}
}

func (buildPosAccessors) Variant(VariantType) map[int]accessor {
	return map[int]accessor{}
}

func buildAccessors(schema *Schema) (map[int]accessor, error) {
	return Visit(schema, buildPosAccessors{})
}

type setFreshIDs struct {
	oldIdToNew map[int]int
	nextIDFunc func() int
}

func (s *setFreshIDs) getAndInc(currentID int) int {
	next := s.nextIDFunc()
	s.oldIdToNew[currentID] = next

	return next
}

func (s *setFreshIDs) Schema(_ *Schema, structResult func() Type) Type {
	return structResult()
}

func (s *setFreshIDs) Struct(st StructType, fieldResults []func() Type) Type {
	newFields := make([]NestedField, len(st.FieldList))
	for idx, f := range st.FieldList {
		newFields[idx] = NestedField{
			ID:             s.getAndInc(f.ID),
			Name:           f.Name,
			Type:           fieldResults[idx](),
			Doc:            f.Doc,
			Required:       f.Required,
			InitialDefault: f.InitialDefault,
			WriteDefault:   f.WriteDefault,
		}
	}

	return &StructType{FieldList: newFields}
}

func (s *setFreshIDs) Field(_ NestedField, fieldResult func() Type) Type {
	return fieldResult()
}

func (s *setFreshIDs) List(list ListType, elemResult func() Type) Type {
	elemID := s.getAndInc(list.ElementID)

	return &ListType{
		ElementID:       elemID,
		Element:         elemResult(),
		ElementRequired: list.ElementRequired,
	}
}

func (s *setFreshIDs) Map(mapType MapType, keyResult, valueResult func() Type) Type {
	keyID := s.getAndInc(mapType.KeyID)
	valueID := s.getAndInc(mapType.ValueID)

	return &MapType{
		KeyID:         keyID,
		ValueID:       valueID,
		KeyType:       keyResult(),
		ValueType:     valueResult(),
		ValueRequired: mapType.ValueRequired,
	}
}

func (s *setFreshIDs) Primitive(p PrimitiveType) Type {
	return p
}

func (s *setFreshIDs) Variant(v VariantType) Type {
	return v
}

// AssignFreshSchemaIDs creates a new schema with fresh field IDs for all of the
// fields in it. The nextID function is used to iteratively generate the ids, if
// it is nil then a simple incrementing counter is used starting at 1.
func AssignFreshSchemaIDs(sc *Schema, nextID func() int) (*Schema, error) {
	if nextID == nil {
		id := 0
		nextID = func() int {
			id++

			return id
		}
	}
	visitor := &setFreshIDs{oldIdToNew: make(map[int]int), nextIDFunc: nextID}
	outType, err := PreOrderVisit(sc, visitor)
	if err != nil {
		return nil, err
	}

	fields := outType.(*StructType).FieldList
	var newIdentifierIDs []int
	if len(sc.IdentifierFieldIDs) != 0 {
		newIdentifierIDs = make([]int, len(sc.IdentifierFieldIDs))
		for i, id := range sc.IdentifierFieldIDs {
			newIdentifierIDs[i] = visitor.oldIdToNew[id]
		}
	}

	return NewSchemaWithIdentifiers(0, newIdentifierIDs, fields...), nil
}

type SchemaWithPartnerVisitor[T, P any] interface {
	Schema(sc *Schema, schemaPartner P, structResult T) T
	Struct(st StructType, structPartner P, fieldResults []T) T
	Field(field NestedField, fieldPartner P, fieldResult T) T
	List(l ListType, listPartner P, elemResult T) T
	Map(m MapType, mapPartner P, keyResult, valResult T) T
	Primitive(p PrimitiveType, primitivePartner P) T
	Variant(v VariantType, variantPartner P) T
}

type PartnerAccessor[P any] interface {
	SchemaPartner(P) P
	FieldPartner(partnerStruct P, fieldID int, fieldName string) P
	ListElementPartner(P) P
	MapKeyPartner(P) P
	MapValuePartner(P) P
}

func VisitSchemaWithPartner[T, P any](sc *Schema, partner P, visitor SchemaWithPartnerVisitor[T, P], accessor PartnerAccessor[P]) (res T, err error) {
	if sc == nil {
		err = fmt.Errorf("%w: cannot visit nil schema", ErrInvalidArgument)

		return res, err
	}

	if visitor == nil || accessor == nil {
		err = fmt.Errorf("%w: cannot visit with nil visitor or accessor", ErrInvalidArgument)

		return res, err
	}

	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case string:
				err = fmt.Errorf("%w: %s", ErrInvalidSchema, e)
			case error:
				err = fmt.Errorf("error encountered during schema visitor: %w", e)
			}
		}
	}()

	structPartner := accessor.SchemaPartner(partner)

	return visitor.Schema(sc, partner, visitStructWithPartner(sc.asStructRef(), structPartner, visitor, accessor)), nil
}

func visitStructWithPartner[T, P any](st StructType, partner P, visitor SchemaWithPartnerVisitor[T, P], accessor PartnerAccessor[P]) T {
	type (
		beforeField interface {
			BeforeField(NestedField, P)
		}
		afterField interface {
			AfterField(NestedField, P)
		}
	)

	bf, _ := visitor.(beforeField)
	af, _ := visitor.(afterField)

	fieldResults := make([]T, len(st.FieldList))

	for i, f := range st.FieldList {
		fieldPartner := accessor.FieldPartner(partner, f.ID, f.Name)
		if bf != nil {
			bf.BeforeField(f, fieldPartner)
		}
		fieldResult := visitTypeWithPartner(f.Type, fieldPartner, visitor, accessor)
		fieldResults[i] = visitor.Field(f, fieldPartner, fieldResult)
		if af != nil {
			af.AfterField(f, fieldPartner)
		}
	}

	return visitor.Struct(st, partner, fieldResults)
}

func visitListWithPartner[T, P any](listType ListType, partner P, visitor SchemaWithPartnerVisitor[T, P], accessor PartnerAccessor[P]) T {
	type (
		beforeListElem interface {
			BeforeListElement(NestedField, P)
		}
		afterListElem interface {
			AfterListElement(NestedField, P)
		}
	)

	elemPartner := accessor.ListElementPartner(partner)
	if ble, ok := visitor.(beforeListElem); ok {
		ble.BeforeListElement(listType.ElementField(), elemPartner)
	}
	elemResult := visitTypeWithPartner(listType.Element, elemPartner, visitor, accessor)
	if ale, ok := visitor.(afterListElem); ok {
		ale.AfterListElement(listType.ElementField(), elemPartner)
	}

	return visitor.List(listType, partner, elemResult)
}

func visitMapWithPartner[T, P any](m MapType, partner P, visitor SchemaWithPartnerVisitor[T, P], accessor PartnerAccessor[P]) T {
	type (
		beforeMapKey interface {
			BeforeMapKey(NestedField, P)
		}
		afterMapKey interface {
			AfterMapKey(NestedField, P)
		}

		beforeMapValue interface {
			BeforeMapValue(NestedField, P)
		}
		afterMapValue interface {
			AfterMapValue(NestedField, P)
		}
	)

	keyPartner := accessor.MapKeyPartner(partner)
	if bmk, ok := visitor.(beforeMapKey); ok {
		bmk.BeforeMapKey(m.KeyField(), keyPartner)
	}
	keyResult := visitTypeWithPartner(m.KeyType, keyPartner, visitor, accessor)
	if amk, ok := visitor.(afterMapKey); ok {
		amk.AfterMapKey(m.KeyField(), keyPartner)
	}

	valPartner := accessor.MapValuePartner(partner)
	if bmv, ok := visitor.(beforeMapValue); ok {
		bmv.BeforeMapValue(m.ValueField(), valPartner)
	}
	valResult := visitTypeWithPartner(m.ValueType, valPartner, visitor, accessor)
	if amv, ok := visitor.(afterMapValue); ok {
		amv.AfterMapValue(m.ValueField(), valPartner)
	}

	return visitor.Map(m, partner, keyResult, valResult)
}

func visitTypeWithPartner[T, P any](t Type, fieldPartner P, visitor SchemaWithPartnerVisitor[T, P], accessor PartnerAccessor[P]) T {
	switch t := t.(type) {
	case *ListType:
		return visitListWithPartner(*t, fieldPartner, visitor, accessor)
	case *StructType:
		return visitStructWithPartner(*t, fieldPartner, visitor, accessor)
	case *MapType:
		return visitMapWithPartner(*t, fieldPartner, visitor, accessor)
	case VariantType:
		return visitor.Variant(t, fieldPartner)
	default:
		return visitor.Primitive(t.(PrimitiveType), fieldPartner)
	}
}

func makeCompatibleName(n string) string {
	if !validAvroName(n) {
		return sanitizeName(n)
	}

	return n
}

func validAvroName(n string) bool {
	if len(n) == 0 {
		return false
	}

	for i, r := range n {
		if i == 0 {
			if !isAvroNameStart(r) {
				return false
			}

			continue
		}

		if !isAvroNamePart(r) {
			return false
		}
	}

	return true
}

const maxBMPRune rune = 0xFFFF

func isAvroNameStart(r rune) bool {
	return r <= maxBMPRune && (r == '_' || unicode.IsLetter(r))
}

func isAvroNamePart(r rune) bool {
	return r <= maxBMPRune && (isAvroNameStart(r) || unicode.IsDigit(r))
}

func sanitize(r rune) string {
	if r > maxBMPRune {
		high, low := utf16.EncodeRune(r)

		return fmt.Sprintf("_x%X_x%X", high, low)
	}

	if unicode.IsDigit(r) {
		return "_" + string(r)
	}

	return fmt.Sprintf("_x%X", r)
}

func sanitizeName(n string) string {
	if len(n) == 0 {
		return ""
	}

	var b strings.Builder
	b.Grow(len(n) * 3)

	for i, r := range n {
		valid := isAvroNamePart(r)
		if i == 0 {
			valid = isAvroNameStart(r)
		}

		if !valid {
			b.WriteString(sanitize(r))
		} else {
			b.WriteRune(r)
		}
	}

	return b.String()
}

// SanitizeColumnNames returns a copy of sc whose field names are compatible
// with Java Iceberg's Avro name sanitization. Characters outside the BMP are
// escaped as UTF-16 surrogate pairs, and only Unicode decimal digits are
// treated as digits; other numeric categories are escaped.
//
// Empty or invalid UTF-8 field names and names that collide after sanitization
// return an error wrapping ErrInvalidSchema. Collision errors are reported here
// before a downstream Avro schema builder encounters the duplicate field name.
func SanitizeColumnNames(sc *Schema) (*Schema, error) {
	result, err := Visit(sc, sanitizeColumnNameVisitor{})
	if err != nil {
		return nil, err
	}

	return NewSchemaWithIdentifiers(sc.ID, sc.IdentifierFieldIDs,
		result.Type.(*StructType).FieldList...), nil
}

type sanitizeColumnNameVisitor struct{}

func (sanitizeColumnNameVisitor) Schema(_ *Schema, structResult NestedField) NestedField {
	return structResult
}

func (sanitizeColumnNameVisitor) Field(field NestedField, fieldResult NestedField) NestedField {
	field.Type = fieldResult.Type

	if field.Name == "" {
		panic(fmt.Errorf("%w: field name cannot be empty", ErrInvalidSchema))
	}
	if !utf8.ValidString(field.Name) {
		panic(fmt.Errorf("%w: field %d name is not valid UTF-8", ErrInvalidSchema, field.ID))
	}

	field.Name = makeCompatibleName(field.Name)

	return field
}

func (sanitizeColumnNameVisitor) Struct(_ StructType, fieldResults []NestedField) NestedField {
	seen := make(map[string]int, len(fieldResults))
	for _, field := range fieldResults {
		if previousID, ok := seen[field.Name]; ok {
			panic(fmt.Errorf(
				"%w: fields %d and %d produce duplicate sanitized name %q",
				ErrInvalidSchema, previousID, field.ID, field.Name))
		}
		seen[field.Name] = field.ID
	}

	return NestedField{Type: &StructType{FieldList: fieldResults}}
}

func (sanitizeColumnNameVisitor) List(list ListType, elemResult NestedField) NestedField {
	list.Element = elemResult.Type

	return NestedField{Type: &list}
}

func (sanitizeColumnNameVisitor) Map(mapType MapType, keyResult, valueResult NestedField) NestedField {
	mapType.KeyType = keyResult.Type
	mapType.ValueType = valueResult.Type

	return NestedField{Type: &mapType}
}

func (sanitizeColumnNameVisitor) Primitive(p PrimitiveType) NestedField {
	return NestedField{Type: p}
}

func (sanitizeColumnNameVisitor) Variant(v VariantType) NestedField {
	return NestedField{Type: v}
}
