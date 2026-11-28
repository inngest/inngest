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
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"slices"
	"strings"

	"github.com/apache/iceberg-go"
)

type SortDirection string

const (
	SortASC  SortDirection = "asc"
	SortDESC SortDirection = "desc"
)

type NullOrder string

const (
	NullsFirst NullOrder = "nulls-first"
	NullsLast  NullOrder = "nulls-last"
)

var (
	ErrInvalidSortOrderID   = errors.New("invalid sort order ID")
	ErrInvalidSortSourceID  = errors.New("invalid sort source ID")
	ErrInvalidTransform     = errors.New("invalid transform, must be a valid transform string or a transform object")
	ErrInvalidSortDirection = errors.New("invalid sort direction, must be 'asc' or 'desc'")
	ErrInvalidNullOrder     = errors.New("invalid null order, must be 'nulls-first' or 'nulls-last'")
)

// orderBinding tells the decoder whether the sort order it reads is already
// bound to a schema, which decides how strictly source IDs are validated.
type orderBinding bool

const (
	boundOrder   orderBinding = true
	unboundOrder orderBinding = false
)

// SortField describes a field used in a sort order definition.
type SortField struct {
	// SourceIDs contains the source column ids from the table's schema.
	// For single-argument transforms this will have exactly one element.
	// For multi-argument transforms this will have multiple elements.
	SourceIDs []int `json:"-"`
	// Transform is the tranformation used to produce values to be
	// sorted on from the source column.
	Transform iceberg.Transform `json:"transform"`
	// Direction is an enum indicating ascending or descending direction.
	Direction SortDirection `json:"direction"`
	// NullOrder describes the order of null values when sorting
	// should be only either nulls-first or nulls-last enum values.
	NullOrder NullOrder `json:"null-order"`
}

// SourceID returns the first source column id.
func (s SortField) SourceID() int {
	if len(s.SourceIDs) == 0 {
		return 0
	}

	return s.SourceIDs[0]
}

func (s SortField) Equals(other SortField) bool {
	return slices.Equal(s.SourceIDs, other.SourceIDs) &&
		s.Transform.Equals(other.Transform) &&
		s.Direction == other.Direction &&
		s.NullOrder == other.NullOrder
}

func (s *SortField) String() string {
	if _, ok := s.Transform.(iceberg.IdentityTransform); ok {
		if len(s.SourceIDs) > 1 {
			return fmt.Sprintf("%v %s %s", s.SourceIDs, s.Direction, s.NullOrder)
		}

		return fmt.Sprintf("%d %s %s", s.SourceID(), s.Direction, s.NullOrder)
	}

	if len(s.SourceIDs) > 1 {
		return fmt.Sprintf("%s(%v) %s %s", s.Transform, s.SourceIDs, s.Direction, s.NullOrder)
	}

	return fmt.Sprintf("%s(%d) %s %s", s.Transform, s.SourceID(), s.Direction, s.NullOrder)
}

func (s SortField) MarshalJSON() ([]byte, error) {
	direction := s.Direction
	if direction == "" {
		direction = SortASC
	}

	nullOrder := s.NullOrder
	if nullOrder == "" {
		if direction == SortASC {
			nullOrder = NullsFirst
		} else {
			nullOrder = NullsLast
		}
	}

	if len(s.SourceIDs) > 1 {
		return json.Marshal(struct {
			SourceIDs []int             `json:"source-ids"`
			Transform iceberg.Transform `json:"transform"`
			Direction SortDirection     `json:"direction"`
			NullOrder NullOrder         `json:"null-order"`
		}{s.SourceIDs, s.Transform, direction, nullOrder})
	}

	return json.Marshal(struct {
		SourceID  int               `json:"source-id"`
		Transform iceberg.Transform `json:"transform"`
		Direction SortDirection     `json:"direction"`
		NullOrder NullOrder         `json:"null-order"`
	}{s.SourceID(), s.Transform, direction, nullOrder})
}

func (s *SortField) UnmarshalJSON(b []byte) error {
	return s.unmarshal(b, boundOrder)
}

func (s *SortField) unmarshal(b []byte, binding orderBinding) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("%w: failed to unmarshal sort field", err)
	}

	_, hasSourceID := raw["source-id"]
	_, hasSourceIDs := raw["source-ids"]
	if hasSourceID && hasSourceIDs {
		return errors.New("sort field cannot contain both source-id and source-ids")
	}
	if !hasSourceID && !hasSourceIDs {
		return fmt.Errorf("%w: exactly one of source-id or source-ids is required", ErrInvalidSortSourceID)
	}

	transformJSON, hasTransform := raw["transform"]
	if !hasTransform || string(transformJSON) == "null" {
		return fmt.Errorf("%w: sort field requires a transform", iceberg.ErrInvalidTransform)
	}

	var sourceID int
	if hasSourceID {
		if err := unmarshalJSONField(raw["source-id"], "source-id", &sourceID); err != nil {
			return err
		}
	}
	var sourceIDs []int
	if hasSourceIDs {
		if err := unmarshalJSONField(raw["source-ids"], "source-ids", &sourceIDs); err != nil {
			return err
		}
	}
	var transformString string
	if err := unmarshalJSONField(transformJSON, "transform", &transformString); err != nil {
		return err
	}
	var direction SortDirection
	if directionJSON, ok := raw["direction"]; ok {
		if err := unmarshalJSONField(directionJSON, "direction", &direction); err != nil {
			return err
		}
	}
	var nullOrder NullOrder
	if nullOrderJSON, ok := raw["null-order"]; ok {
		if err := unmarshalJSONField(nullOrderJSON, "null-order", &nullOrder); err != nil {
			return err
		}
	}

	next := SortField{
		Direction: direction,
		NullOrder: nullOrder,
	}

	if hasSourceIDs {
		next.SourceIDs = sourceIDs
	} else {
		next.SourceIDs = []int{sourceID}
	}

	if err := validateSortSourceIDs(next.SourceIDs, binding); err != nil {
		return err
	}

	var err error
	if next.Transform, err = iceberg.ParseTransform(transformString); err != nil {
		return err
	}

	switch next.Direction {
	case SortASC, SortDESC:
	default:
		return ErrInvalidSortDirection
	}

	switch next.NullOrder {
	case NullsFirst, NullsLast:
	default:
		return ErrInvalidNullOrder
	}

	*s = next

	return nil
}

// Source IDs are schema field IDs, and therefore positive, once an order is
// bound to a schema. Unbound orders carry client placeholders that start at zero.
func validateSortSourceID(id int, binding orderBinding) error {
	if binding == boundOrder && id <= 0 {
		return fmt.Errorf("%w: source ID must be positive: %d", ErrInvalidSortSourceID, id)
	}
	if id < 0 {
		return fmt.Errorf("%w: source ID must be non-negative: %d", ErrInvalidSortSourceID, id)
	}

	return nil
}

func unmarshalJSONField(data json.RawMessage, field string, value any) error {
	if err := json.Unmarshal(data, value); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) {
			typeErr.Struct = ""
			typeErr.Field = field
		}

		return err
	}

	return nil
}

func validateSortSourceIDs(ids []int, binding orderBinding) error {
	if len(ids) == 0 {
		return fmt.Errorf("%w: source-ids must not be empty", ErrInvalidSortSourceID)
	}

	for _, id := range ids {
		if err := validateSortSourceID(id, binding); err != nil {
			return err
		}
	}

	return nil
}

const (
	InitialSortOrderID  = 1
	UnsortedSortOrderID = 0
)

// A default Sort Order indicating no sort order at all
var UnsortedSortOrder = SortOrder{orderID: UnsortedSortOrderID, fields: []SortField{}}

// SortOrder describes how the data is sorted within the table.
//
// Data can be sorted within partitions by columns to gain performance. The
// order of the sort fields within the list defines the order in which the
// sort is applied to the data.
type SortOrder struct {
	orderID int
	fields  []SortField
}

// UnboundSortOrder decodes a sort order whose source IDs have not yet been
// resolved against the schema that decides them, so unlike bound field IDs they
// need not be positive. Two wire forms arrive this way:
//
//   - A create-table request's write order, which carries the client's
//     placeholder source IDs rather than table field IDs. A client numbers those
//     placeholders however it likes: Spark numbers the columns of a new table
//     from zero, so sorting by the first column arrives as source-id 0. Binding
//     the embedded order to the schema the client sent resolves them.
//   - An add-sort-order commit payload, which applying the update checks against
//     the table's current schema.
//
// Decoding unbound defers the check rather than dropping it: every source ID
// must still resolve in the schema it is checked against, and no transform is
// exempt, so an order the schema cannot satisfy is rejected when the update is
// applied.
//
// Use SortOrder for orders read from table metadata, where source IDs are bound
// field IDs and must be positive. Catalog implementations should decode both the
// REST create-table request's write order and an add-sort-order commit payload
// into this type.
type UnboundSortOrder struct {
	SortOrder
}

func (u *UnboundSortOrder) UnmarshalJSON(b []byte) error {
	return u.unmarshal(b, unboundOrder)
}

func (s SortOrder) OrderID() int {
	return s.orderID
}

func (s SortOrder) Fields() iter.Seq2[int, SortField] {
	return func(yield func(int, SortField) bool) {
		for i, field := range s.fields {
			if !yield(i, cloneSortField(field)) {
				return
			}
		}
	}
}

func (s SortOrder) Len() int {
	return len(s.fields)
}

// Field returns a copy of the sort field at index i, like Fields does, so a
// caller can't reach the sort order's internals through SortField.SourceIDs.
// It panics if i is out of range.
func (s SortOrder) Field(i int) SortField {
	return cloneSortField(s.fields[i])
}

func (s SortOrder) MarshalJSON() ([]byte, error) {
	type Alias struct {
		OrderID int         `json:"order-id"`
		Fields  []SortField `json:"fields"`
	}

	return json.Marshal(Alias{
		s.orderID,
		s.fields,
	})
}

func (s *SortOrder) UnmarshalJSON(b []byte) error {
	return s.unmarshal(b, boundOrder)
}

func (s *SortOrder) unmarshal(b []byte, binding orderBinding) error {
	aux := struct {
		OrderID *int               `json:"order-id"`
		Fields  *[]json.RawMessage `json:"fields"`
	}{}

	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	if aux.OrderID == nil {
		return fmt.Errorf("%w: sort order is missing required 'order-id' key in JSON", iceberg.ErrInvalidArgument)
	}

	if aux.Fields == nil {
		return fmt.Errorf("%w: sort order is missing required 'fields' key in JSON", iceberg.ErrInvalidArgument)
	}

	fields := make([]SortField, len(*aux.Fields))
	for i, rawField := range *aux.Fields {
		if err := fields[i].unmarshal(rawField, binding); err != nil {
			return err
		}
	}

	newOrder, err := newSortOrder(*aux.OrderID, fields, false)
	if err != nil {
		return err
	}

	*s = newOrder

	return nil
}

// NewSortOrder creates a new SortOrder.
//
// The orderID must be greater than or equal to 0.
// If orderID is 0, no fields can be passed, this is equal to UnsortedSortOrder.
// If fields is empty, orderID must be 0.
// Fields need to have non-nil Transform, valid Direction and NullOrder values,
// and non-empty source IDs.
func NewSortOrder(orderID int, fields []SortField) (SortOrder, error) {
	return newSortOrder(orderID, fields, true)
}

func newSortOrder(orderID int, fields []SortField, validateSourceIDs bool) (SortOrder, error) {
	if orderID < 0 {
		return SortOrder{}, fmt.Errorf("%w: sort order ID %d must be a non-negative integer",
			ErrInvalidSortOrderID, orderID)
	}

	if orderID == 0 && len(fields) != 0 {
		return SortOrder{}, fmt.Errorf("%w: sort order ID 0 is reserved for unsorted order", ErrInvalidSortOrderID)
	}

	if orderID != UnsortedSortOrderID && len(fields) == 0 {
		return SortOrder{}, fmt.Errorf("%w: sort order ID %d requires at least one sort field", ErrInvalidSortOrderID, orderID)
	}

	if fields == nil {
		fields = []SortField{}
	}
	for idx, field := range fields {
		if field.Transform == nil {
			return SortOrder{}, fmt.Errorf("%w: sort field at index %d has no transform", ErrInvalidTransform, idx)
		}
		if marshaler, ok := field.Transform.(encoding.TextMarshaler); ok {
			if _, err := marshaler.MarshalText(); err != nil {
				return SortOrder{}, fmt.Errorf("%w: sort field at index %d: %w", ErrInvalidTransform, idx, err)
			}
		}
		if field.Direction != SortASC && field.Direction != SortDESC {
			return SortOrder{}, fmt.Errorf("%w: sort field at index %d", ErrInvalidSortDirection, idx)
		}
		if field.NullOrder != NullsFirst && field.NullOrder != NullsLast {
			return SortOrder{}, fmt.Errorf("%w: sort field at index %d", ErrInvalidNullOrder, idx)
		}
		if validateSourceIDs {
			if err := validateSortSourceIDs(field.SourceIDs, boundOrder); err != nil {
				return SortOrder{}, fmt.Errorf("sort field at index %d has invalid source IDs: %w", idx, err)
			}
		}
	}

	fieldCopies := make([]SortField, len(fields))
	for i, field := range fields {
		fieldCopies[i] = cloneSortField(field)
	}

	return SortOrder{orderID, fieldCopies}, nil
}

// cloneSortField copies mutable sort-field state, including pointer-backed
// transforms whose values can be changed by metadata consumers.
func cloneSortField(field SortField) SortField {
	field.SourceIDs = slices.Clone(field.SourceIDs)
	switch transform := field.Transform.(type) {
	case *iceberg.BucketTransform:
		if transform != nil {
			cloned := *transform
			field.Transform = &cloned
		}
	case *iceberg.TruncateTransform:
		if transform != nil {
			cloned := *transform
			field.Transform = &cloned
		}
	}

	return field
}

func (s SortOrder) IsUnsorted() bool {
	return len(s.fields) == 0
}

func (s *SortOrder) CheckCompatibility(schema *iceberg.Schema) error {
	if s == nil {
		return nil
	}

	for _, field := range s.fields {
		if field.Transform == nil {
			return fmt.Errorf("%w: sort field with source id %d has no transform", ErrInvalidTransform, field.SourceID())
		}

		if err := validateSortSourceIDs(field.SourceIDs, boundOrder); err != nil {
			return fmt.Errorf("sort field has invalid source IDs: %w", err)
		}

		var firstField iceberg.NestedField
		for idx, sourceID := range field.SourceIDs {
			f, ok := schema.FindFieldByID(sourceID)
			if !ok {
				return fmt.Errorf("sort field with source id %d not found in schema", sourceID)
			}

			if _, ok := f.Type.(iceberg.PrimitiveType); !ok {
				return fmt.Errorf("cannot sort by non-primitive source field: %s", f.Type.Type())
			}

			if idx == 0 {
				firstField = f
			}
		}

		if !field.Transform.CanTransform(firstField.Type) {
			return fmt.Errorf("invalid source type %s for transform %s", firstField.Type.Type(), field.Transform)
		}
	}

	return nil
}

func (s SortOrder) Equals(rhs SortOrder) bool {
	return s.orderID == rhs.orderID &&
		slices.EqualFunc(s.fields, rhs.fields, func(a, b SortField) bool {
			return a.Equals(b)
		})
}

func (s SortOrder) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d: ", s.orderID)
	b.WriteByte('[')
	for i, f := range s.fields {
		if i == 0 {
			b.WriteByte('\n')
		}
		b.WriteString(f.String())
		b.WriteByte('\n')
	}
	b.WriteByte(']')

	return b.String()
}

// AssignFreshSortOrderIDs updates and reassigns the field source IDs from the old schema
// to the corresponding fields in the fresh schema, while also giving the Sort Order a fresh
// ID of 0 (the initial Sort Order ID).
func AssignFreshSortOrderIDs(sortOrder SortOrder, old, fresh *iceberg.Schema) (SortOrder, error) {
	return AssignFreshSortOrderIDsWithID(sortOrder, old, fresh, InitialSortOrderID)
}

// AssignFreshSortOrderIDsWithID is like AssignFreshSortOrderIDs but allows specifying the id of the
// returned SortOrder.
func AssignFreshSortOrderIDsWithID(sortOrder SortOrder, old, fresh *iceberg.Schema, sortOrderID int) (SortOrder, error) {
	if sortOrder.Equals(UnsortedSortOrder) {
		return UnsortedSortOrder, nil
	}

	fields := make([]SortField, 0, len(sortOrder.fields))
	for _, field := range sortOrder.fields {
		originalField, ok := old.FindColumnName(field.SourceID())
		if !ok {
			return SortOrder{}, fmt.Errorf("cannot find source column id %s in old schema", field.String())
		}
		freshField, ok := fresh.FindFieldByName(originalField)
		if !ok {
			return SortOrder{}, fmt.Errorf("cannot find field %s in fresh schema", originalField)
		}

		fields = append(fields, SortField{
			SourceIDs: []int{freshField.ID},
			Transform: field.Transform,
			Direction: field.Direction,
			NullOrder: field.NullOrder,
		})
	}

	return SortOrder{orderID: sortOrderID, fields: fields}, nil
}
