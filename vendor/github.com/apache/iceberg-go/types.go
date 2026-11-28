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
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/apache/arrow-go/v18/arrow/decimal"
	"github.com/apache/iceberg-go/internal"
	"github.com/geoarrow/geoarrow-go"
)

var (
	fixedRegex     = regexp.MustCompile(`^fixed\[(\d+)\]$`)
	decimalRegex   = regexp.MustCompile(`^decimal\(\s*(\d+)\s*,\s*(\d+)\s*\)$`)
	geometryRegex  = regexp.MustCompile(`(?i)^geometry\s*(?:\(\s*([^),]+?)\s*\))?$`)
	geographyRegex = regexp.MustCompile(`(?i)^geography\s*(?:\(\s*([^\s,)]+)\s*(?:,\s*(\w+)\s*)?\))?$`)
)

type Properties map[string]string

// Get returns the value of the key if it exists, otherwise it returns the default value.
func (p Properties) Get(key, defVal string) string {
	if v, ok := p[key]; ok {
		return v
	}

	return defVal
}

func (p Properties) GetBool(key string, defVal bool) bool {
	if v, ok := p[key]; ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return defVal
		}

		return b
	}

	return defVal
}

func (p Properties) GetInt(key string, defVal int) int {
	if v, ok := p[key]; ok {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return defVal
		}

		return int(i)
	}

	return defVal
}

// GetInt64 reads a 64-bit integer property by key. A missing key or an
// unparseable value returns defVal. Unlike GetInt, this avoids truncating
// large int64 sentinel values (such as math.MaxInt64) on 32-bit platforms
// where int is only 32 bits wide.
func (p Properties) GetInt64(key string, defVal int64) int64 {
	if v, ok := p[key]; ok {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return defVal
		}

		return i
	}

	return defVal
}

// GetUInt64 reads an unsigned-integer property by key. A missing key,
// an unparseable value, or a negative value returns defVal. GetUInt64
// uses strconv.ParseUint, which rejects negatives rather than silently
// wrapping them to a large positive number.
func (p Properties) GetUInt64(key string, defVal uint64) uint64 {
	v, ok := p[key]
	if !ok {
		return defVal
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return defVal
	}

	return n
}

// PropUInt reads an unsigned-integer property by key, preserving the legacy
// fallback behavior while avoiding truncation on platforms where uint is
// narrower than uint64.
func PropUInt(p Properties, key string, defVal uint) uint {
	n := p.GetUInt64(key, uint64(defVal))
	if uint64(uint(n)) != n {
		return defVal
	}

	return uint(n)
}

// Type is an interface representing any of the available iceberg types,
// such as primitives (int32/int64/etc.) or nested types (list/struct/map).
type Type interface {
	fmt.Stringer
	Type() string
	Equals(Type) bool
}

// NestedType is an interface that allows access to the child fields of
// a nested type such as a list/struct/map type.
type NestedType interface {
	Type
	Fields() []NestedField
}

type typeIFace struct {
	Type
}

func (t *typeIFace) MarshalJSON() ([]byte, error) {
	if t.Type == nil {
		return nil, fmt.Errorf("%w: field type is nil; use the \"type\" JSON key with a valid Iceberg type such as \"long\", \"string\", or \"boolean\"", ErrInvalidSchema)
	}
	if nested, ok := t.Type.(NestedType); ok {
		return json.Marshal(nested)
	}

	return []byte(`"` + t.Type.Type() + `"`), nil
}

func (t *typeIFace) UnmarshalJSON(b []byte) error {
	var typename string
	err := json.Unmarshal(b, &typename)
	if err == nil {
		switch typename {
		case "boolean":
			t.Type = BooleanType{}
		case "int":
			t.Type = Int32Type{}
		case "long":
			t.Type = Int64Type{}
		case "float":
			t.Type = Float32Type{}
		case "double":
			t.Type = Float64Type{}
		case "date":
			t.Type = DateType{}
		case "time":
			t.Type = TimeType{}
		case "timestamp":
			t.Type = TimestampType{}
		case "timestamp_ns":
			t.Type = TimestampNsType{}
		case "timestamptz_ns":
			t.Type = TimestampTzNsType{}
		case "timestamptz":
			t.Type = TimestampTzType{}
		case "string":
			t.Type = StringType{}
		case "uuid":
			t.Type = UUIDType{}
		case "binary":
			t.Type = BinaryType{}
		case "unknown":
			t.Type = UnknownType{}
		case "variant":
			t.Type = VariantType{}
		case "geometry":
			t.Type = GeometryType{}
		case "geography":
			t.Type = GeographyType{}
		default:
			switch {
			case strings.HasPrefix(typename, "fixed"):
				matches := fixedRegex.FindStringSubmatch(typename)
				if len(matches) != 2 {
					return fmt.Errorf("%w: %s", ErrInvalidTypeString, typename)
				}

				n, err := strconv.Atoi(matches[1])
				if err != nil {
					return fmt.Errorf("%w: invalid fixed length %q: %v", ErrInvalidTypeString, matches[1], err)
				}
				t.Type = FixedType{len: n}
			case strings.HasPrefix(typename, "decimal"):
				matches := decimalRegex.FindStringSubmatch(typename)
				if len(matches) != 3 {
					return fmt.Errorf("%w: %s", ErrInvalidTypeString, typename)
				}

				prec, err := strconv.Atoi(matches[1])
				if err != nil {
					return fmt.Errorf("%w: %s", ErrInvalidTypeString, typename)
				}
				scale, err := strconv.Atoi(matches[2])
				if err != nil {
					return fmt.Errorf("%w: %s", ErrInvalidTypeString, typename)
				}
				if err := validateDecimalPrecisionScale(prec, scale); err != nil {
					return fmt.Errorf("%w: %w", ErrInvalidTypeString, err)
				}

				t.Type = DecimalType{precision: prec, scale: scale}
			// note that geo type names are case insensitive but other type names are case sensitive.
			// matches java behavior - this behavior is intentional
			case strings.HasPrefix(strings.ToLower(typename), "geometry"):
				matches := geometryRegex.FindStringSubmatch(typename)
				if len(matches) != 2 {
					return fmt.Errorf("%w: %s", ErrInvalidTypeString, typename)
				}

				if matches[1] != "" {
					geom, err := GeometryTypeOf(strings.TrimSpace(matches[1]))
					if err != nil {
						return err
					}
					t.Type = geom
				} else {
					t.Type = GeometryType{}
				}
			case strings.HasPrefix(strings.ToLower(typename), "geography"):
				matches := geographyRegex.FindStringSubmatch(typename)
				if len(matches) != 3 {
					return fmt.Errorf("%w: %s", ErrInvalidTypeString, typename)
				}

				crs := DefaultGeoCRS
				if matches[1] != "" {
					crs = strings.TrimSpace(matches[1])
				}

				algorithm := defaultGeographyAlgorithm
				if matches[2] != "" {
					algorithm = strings.TrimSpace(matches[2])
				}

				geog, err := GeographyTypeOf(crs, algorithm)
				if err != nil {
					return err
				}
				t.Type = geog
			default:
				return fmt.Errorf("%w: unrecognized field type", ErrInvalidSchema)
			}
		}

		return nil
	}

	aux := struct {
		TypeName string `json:"type"`
	}{}
	if err = json.Unmarshal(b, &aux); err != nil {
		return err
	}

	switch aux.TypeName {
	case "list":
		t.Type = &ListType{}
	case "map":
		t.Type = &MapType{}
	case "struct":
		t.Type = &StructType{}
	default:
		return fmt.Errorf("%w: %s", ErrInvalidTypeString, aux.TypeName)
	}

	return json.Unmarshal(b, t.Type)
}

type NestedField struct {
	Type `json:"-"`

	ID             int    `json:"id"`
	Name           string `json:"name"`
	Required       bool   `json:"required"`
	Doc            string `json:"doc,omitempty"`
	InitialDefault any    `json:"initial-default,omitempty"`
	WriteDefault   any    `json:"write-default,omitempty"`
}

func optOrReq(required bool) string {
	if required {
		return "required"
	}

	return "optional"
}

func (n NestedField) String() string {
	doc := n.Doc
	if doc != "" {
		doc = " (" + doc + ")"
	}

	return fmt.Sprintf("%d: %s: %s %s%s",
		n.ID, n.Name, optOrReq(n.Required), n.Type, doc)
}

func (n *NestedField) Equals(other NestedField) bool {
	return n.ID == other.ID &&
		n.Name == other.Name &&
		n.Required == other.Required &&
		n.Doc == other.Doc &&
		reflect.DeepEqual(n.InitialDefault, other.InitialDefault) &&
		reflect.DeepEqual(n.WriteDefault, other.WriteDefault) &&
		n.Type.Equals(other.Type)
}

func (n NestedField) MarshalJSON() ([]byte, error) {
	var initialDefault, writeDefault *any
	if n.InitialDefault != nil {
		value := defaultValueToJSON(n.Type, n.InitialDefault)
		initialDefault = &value
	}
	if n.WriteDefault != nil {
		value := defaultValueToJSON(n.Type, n.WriteDefault)
		writeDefault = &value
	}

	return json.Marshal(struct {
		Type           *typeIFace `json:"type"`
		ID             int        `json:"id"`
		Name           string     `json:"name"`
		Required       bool       `json:"required"`
		Doc            string     `json:"doc,omitempty"`
		InitialDefault *any       `json:"initial-default,omitempty"`
		WriteDefault   *any       `json:"write-default,omitempty"`
	}{
		Type: &typeIFace{n.Type},
		ID:   n.ID, Name: n.Name, Required: n.Required, Doc: n.Doc,
		InitialDefault: initialDefault, WriteDefault: writeDefault,
	})
}

func defaultValueToJSON(typ Type, value any) any {
	switch typ.(type) {
	case BinaryType, FixedType:
		switch value := value.(type) {
		case []byte:
			return internal.EncodeDefaultBytes(value)
		case BinaryLiteral:
			return internal.EncodeDefaultBytes(value)
		case FixedLiteral:
			return internal.EncodeDefaultBytes(value)
		}
	}

	return value
}

func (n *NestedField) UnmarshalJSON(b []byte) error {
	type Alias NestedField
	var next NestedField
	aux := struct {
		ID       *int      `json:"id"`
		Required *bool     `json:"required"`
		Type     typeIFace `json:"type"`
		*Alias
	}{
		Alias: (*Alias)(&next),
	}

	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	next.Type = aux.Type.Type

	if aux.ID == nil {
		return fmt.Errorf("%w: field is missing required 'id' key in JSON", ErrInvalidSchema)
	}
	next.ID = *aux.ID

	if next.Name == "" {
		return fmt.Errorf("%w: field is missing required 'name' key in JSON", ErrInvalidSchema)
	}

	if next.Type == nil {
		return fmt.Errorf("%w: field %q is missing required 'type' key in JSON", ErrInvalidSchema, next.Name)
	}

	if aux.Required == nil {
		return fmt.Errorf("%w: field is missing required 'required' key in JSON", ErrInvalidSchema)
	}
	next.Required = *aux.Required

	*n = next

	return nil
}

type StructType struct {
	FieldList []NestedField `json:"fields"`
}

func (s *StructType) Equals(other Type) bool {
	st, ok := other.(*StructType)
	if !ok {
		return false
	}

	return slices.EqualFunc(s.FieldList, st.FieldList, func(a, b NestedField) bool {
		return a.Equals(b)
	})
}

func (s *StructType) Fields() []NestedField { return s.FieldList }

func (s *StructType) MarshalJSON() ([]byte, error) {
	type Alias StructType

	return json.Marshal(struct {
		Type string `json:"type"`
		*Alias
	}{Type: s.Type(), Alias: (*Alias)(s)})
}

func (*StructType) Type() string { return "struct" }
func (s *StructType) String() string {
	var b strings.Builder
	b.WriteString("struct<")
	for i, f := range s.FieldList {
		if i != 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%d: %s: ",
			f.ID, f.Name)
		if f.Required {
			b.WriteString("required ")
		} else {
			b.WriteString("optional ")
		}
		b.WriteString(f.Type.String())
		if f.Doc != "" {
			b.WriteString(" (")
			b.WriteString(f.Doc)
			b.WriteByte(')')
		}
	}
	b.WriteString(">")

	return b.String()
}

type ListType struct {
	ElementID       int  `json:"element-id"`
	Element         Type `json:"-"`
	ElementRequired bool `json:"element-required"`
}

func (l *ListType) MarshalJSON() ([]byte, error) {
	type Alias ListType

	return json.Marshal(struct {
		Type string `json:"type"`
		*Alias
		Element *typeIFace `json:"element"`
	}{Type: l.Type(), Alias: (*Alias)(l), Element: &typeIFace{l.Element}})
}

func (l *ListType) Equals(other Type) bool {
	rhs, ok := other.(*ListType)
	if !ok {
		return false
	}

	return l.ElementID == rhs.ElementID &&
		l.Element.Equals(rhs.Element) &&
		l.ElementRequired == rhs.ElementRequired
}

func (l *ListType) Fields() []NestedField {
	return []NestedField{l.ElementField()}
}

func (l *ListType) ElementField() NestedField {
	return NestedField{
		ID:       l.ElementID,
		Name:     "element",
		Type:     l.Element,
		Required: l.ElementRequired,
	}
}

func (*ListType) Type() string     { return "list" }
func (l *ListType) String() string { return fmt.Sprintf("list<%s>", l.Element) }

func (l *ListType) UnmarshalJSON(b []byte) error {
	aux := struct {
		ID   *int      `json:"element-id"`
		Elem typeIFace `json:"element"`
		Req  *bool     `json:"element-required"`
	}{}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	if aux.ID == nil {
		return fmt.Errorf("%w: field is missing required 'element-id' key in JSON", ErrInvalidSchema)
	}
	if aux.Elem.Type == nil {
		return fmt.Errorf("%w: field is missing required 'element' key in JSON", ErrInvalidSchema)
	}
	if aux.Req == nil {
		return fmt.Errorf("%w: field is missing required 'element-required' key in JSON", ErrInvalidSchema)
	}

	l.ElementID = *aux.ID
	l.Element = aux.Elem.Type
	l.ElementRequired = *aux.Req

	return nil
}

type MapType struct {
	KeyID         int  `json:"key-id"`
	KeyType       Type `json:"-"`
	ValueID       int  `json:"value-id"`
	ValueType     Type `json:"-"`
	ValueRequired bool `json:"value-required"`
}

func (m *MapType) MarshalJSON() ([]byte, error) {
	type Alias MapType

	return json.Marshal(struct {
		Type string `json:"type"`
		*Alias
		KeyType   *typeIFace `json:"key"`
		ValueType *typeIFace `json:"value"`
	}{
		Type: m.Type(), Alias: (*Alias)(m),
		KeyType:   &typeIFace{m.KeyType},
		ValueType: &typeIFace{m.ValueType},
	})
}

func (m *MapType) Equals(other Type) bool {
	rhs, ok := other.(*MapType)
	if !ok {
		return false
	}

	return m.KeyID == rhs.KeyID &&
		m.KeyType.Equals(rhs.KeyType) &&
		m.ValueID == rhs.ValueID &&
		m.ValueType.Equals(rhs.ValueType) &&
		m.ValueRequired == rhs.ValueRequired
}

func (m *MapType) Fields() []NestedField {
	return []NestedField{m.KeyField(), m.ValueField()}
}

func (m *MapType) KeyField() NestedField {
	return NestedField{
		Name:     "key",
		ID:       m.KeyID,
		Type:     m.KeyType,
		Required: true,
	}
}

func (m *MapType) ValueField() NestedField {
	return NestedField{
		Name:     "value",
		ID:       m.ValueID,
		Type:     m.ValueType,
		Required: m.ValueRequired,
	}
}

func (*MapType) Type() string { return "map" }
func (m *MapType) String() string {
	return fmt.Sprintf("map<%s, %s>", m.KeyType, m.ValueType)
}

func (m *MapType) UnmarshalJSON(b []byte) error {
	aux := struct {
		KeyID    *int      `json:"key-id"`
		Key      typeIFace `json:"key"`
		ValueID  *int      `json:"value-id"`
		Value    typeIFace `json:"value"`
		ValueReq *bool     `json:"value-required"`
	}{}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	if aux.KeyID == nil {
		return fmt.Errorf("%w: field is missing required 'key-id' key in JSON", ErrInvalidSchema)
	}
	if aux.Key.Type == nil {
		return fmt.Errorf("%w: field is missing required 'key' key in JSON", ErrInvalidSchema)
	}

	if aux.ValueID == nil {
		return fmt.Errorf("%w: field is missing required 'value-id' key in JSON", ErrInvalidSchema)
	}
	if aux.Value.Type == nil {
		return fmt.Errorf("%w: field is missing required 'value' key in JSON", ErrInvalidSchema)
	}
	if aux.ValueReq == nil {
		return fmt.Errorf("%w: field is missing required 'value-required' key in JSON", ErrInvalidSchema)
	}

	m.KeyID, m.KeyType = *aux.KeyID, aux.Key.Type
	m.ValueID, m.ValueType = *aux.ValueID, aux.Value.Type
	m.ValueRequired = *aux.ValueReq

	return nil
}

func validateFixedLength(length int) error {
	if length < 0 {
		return fmt.Errorf("fixed length must be non-negative, got %d", length)
	}

	return nil
}

func FixedTypeOf(n int) FixedType {
	if err := validateFixedLength(n); err != nil {
		panic(fmt.Errorf("%w: %s", ErrInvalidArgument, err))
	}

	return FixedType{len: n}
}

type FixedType struct {
	len int
}

func (f FixedType) Equals(other Type) bool {
	rhs, ok := other.(FixedType)
	if !ok {
		return false
	}

	return f.len == rhs.len
}
func (f FixedType) Len() int       { return f.len }
func (f FixedType) Type() string   { return fmt.Sprintf("fixed[%d]", f.len) }
func (f FixedType) String() string { return fmt.Sprintf("fixed[%d]", f.len) }
func (f FixedType) primitive()     {}

func DecimalTypeOf(prec, scale int) DecimalType {
	if err := validateDecimalPrecisionScale(prec, scale); err != nil {
		panic(fmt.Errorf("%w: %w", ErrInvalidArgument, err))
	}

	return DecimalType{precision: prec, scale: scale}
}

func validateDecimalPrecisionScale(precision, scale int) error {
	if precision <= 0 {
		return fmt.Errorf("invalid precision %d: must be greater than 0", precision)
	}
	if precision > 38 {
		return fmt.Errorf("invalid precision %d: must be less than or equal to 38", precision)
	}
	if scale < 0 {
		return fmt.Errorf("invalid scale %d: must be greater than or equal to 0", scale)
	}
	if scale > precision {
		return fmt.Errorf("invalid scale %d: must be less than or equal to precision %d", scale, precision)
	}

	return nil
}

type DecimalType struct {
	precision, scale int
}

func (d DecimalType) Equals(other Type) bool {
	rhs, ok := other.(DecimalType)
	if !ok {
		return false
	}

	return d.precision == rhs.precision &&
		d.scale == rhs.scale
}

func (d DecimalType) Type() string   { return fmt.Sprintf("decimal(%d, %d)", d.precision, d.scale) }
func (d DecimalType) String() string { return fmt.Sprintf("decimal(%d, %d)", d.precision, d.scale) }
func (d DecimalType) Precision() int { return d.precision }
func (d DecimalType) Scale() int     { return d.scale }
func (DecimalType) primitive()       {}

type Decimal struct {
	Val   decimal.Decimal128
	Scale int
}

func (d Decimal) String() string {
	return d.Val.ToString(int32(d.Scale))
}

type PrimitiveType interface {
	Type
	primitive()
}

type BooleanType struct{}

func (BooleanType) Equals(other Type) bool {
	_, ok := other.(BooleanType)

	return ok
}

func (BooleanType) primitive()     {}
func (BooleanType) Type() string   { return "boolean" }
func (BooleanType) String() string { return "boolean" }

// Int32Type is the "int"/"integer" type of the iceberg spec.
type Int32Type struct{}

func (Int32Type) Equals(other Type) bool {
	_, ok := other.(Int32Type)

	return ok
}

func (Int32Type) primitive()     {}
func (Int32Type) Type() string   { return "int" }
func (Int32Type) String() string { return "int" }

// Int64Type is the "long" type of the iceberg spec.
type Int64Type struct{}

func (Int64Type) Equals(other Type) bool {
	_, ok := other.(Int64Type)

	return ok
}

func (Int64Type) primitive()     {}
func (Int64Type) Type() string   { return "long" }
func (Int64Type) String() string { return "long" }

// Float32Type is the "float" type in the iceberg spec.
type Float32Type struct{}

func (Float32Type) Equals(other Type) bool {
	_, ok := other.(Float32Type)

	return ok
}

func (Float32Type) primitive()     {}
func (Float32Type) Type() string   { return "float" }
func (Float32Type) String() string { return "float" }

// Float64Type represents the "double" type of the iceberg spec.
type Float64Type struct{}

func (Float64Type) Equals(other Type) bool {
	_, ok := other.(Float64Type)

	return ok
}

func (Float64Type) primitive()     {}
func (Float64Type) Type() string   { return "double" }
func (Float64Type) String() string { return "double" }

type Date int32

func (d Date) ToTime() time.Time {
	return epochTM.AddDate(0, 0, int(d))
}

// DateType represents a calendar date without a timezone or time,
// represented as a 32-bit integer denoting the number of days since
// the unix epoch.
type DateType struct{}

func (DateType) Equals(other Type) bool {
	_, ok := other.(DateType)

	return ok
}

func (DateType) primitive()     {}
func (DateType) Type() string   { return "date" }
func (DateType) String() string { return "date" }

type Time int64

func (t Time) ToTime() time.Time {
	return time.UnixMicro(int64(t)).UTC()
}

// TimeType represents a number of microseconds since midnight.
type TimeType struct{}

func (TimeType) Equals(other Type) bool {
	_, ok := other.(TimeType)

	return ok
}

func (TimeType) primitive()     {}
func (TimeType) Type() string   { return "time" }
func (TimeType) String() string { return "time" }

type Timestamp int64

func (t Timestamp) ToTime() time.Time {
	return time.UnixMicro(int64(t)).UTC()
}

func (t Timestamp) ToDate() Date {
	const microsecondsPerDay = int64((time.Hour * 24) / time.Microsecond)

	return Date(int32(internal.FloorDiv(int64(t), microsecondsPerDay)))
}

func (t Timestamp) ToNanos() TimestampNano {
	return TimestampNano(int64(t) * 1000)
}

type TimestampNano int64

func (t TimestampNano) ToTime() time.Time {
	return time.Unix(0, int64(t)).UTC()
}

func (t TimestampNano) ToMicros() Timestamp {
	return Timestamp(internal.FloorDiv(int64(t), 1000))
}

func (t TimestampNano) ToDate() Date {
	const nanosecondsPerDay = int64(time.Hour * 24)

	return Date(int32(internal.FloorDiv(int64(t), nanosecondsPerDay)))
}

// TimestampType represents a number of microseconds since the unix epoch
// without regard for timezone.
type TimestampType struct{}

func (TimestampType) Equals(other Type) bool {
	_, ok := other.(TimestampType)

	return ok
}

func (TimestampType) primitive()     {}
func (TimestampType) Type() string   { return "timestamp" }
func (TimestampType) String() string { return "timestamp" }

// TimestampTzType represents a timestamp stored as UTC representing the
// number of microseconds since the unix epoch.
type TimestampTzType struct{}

func (TimestampTzType) Equals(other Type) bool {
	_, ok := other.(TimestampTzType)

	return ok
}

func (TimestampTzType) primitive()     {}
func (TimestampTzType) Type() string   { return "timestamptz" }
func (TimestampTzType) String() string { return "timestamptz" }

type StringType struct{}

func (StringType) Equals(other Type) bool {
	_, ok := other.(StringType)

	return ok
}

func (StringType) primitive()     {}
func (StringType) Type() string   { return "string" }
func (StringType) String() string { return "string" }

type UUIDType struct{}

func (UUIDType) Equals(other Type) bool {
	_, ok := other.(UUIDType)

	return ok
}

func (UUIDType) primitive()     {}
func (UUIDType) Type() string   { return "uuid" }
func (UUIDType) String() string { return "uuid" }

type BinaryType struct{}

func (BinaryType) Equals(other Type) bool {
	_, ok := other.(BinaryType)

	return ok
}

func (BinaryType) primitive()     {}
func (BinaryType) Type() string   { return "binary" }
func (BinaryType) String() string { return "binary" }

// TimestampNsType represents a timestamp stored as nanoseconds since the unix epoch
// without regard for timezone. Requires format version 3+.
type TimestampNsType struct{}

func (TimestampNsType) Equals(other Type) bool {
	_, ok := other.(TimestampNsType)

	return ok
}

func (TimestampNsType) primitive()     {}
func (TimestampNsType) Type() string   { return "timestamp_ns" }
func (TimestampNsType) String() string { return "timestamp_ns" }

// TimestampTzNsType represents a timestamp stored as UTC with nanoseconds since
// the unix epoch. Requires format version 3+.
type TimestampTzNsType struct{}

func (TimestampTzNsType) Equals(other Type) bool {
	_, ok := other.(TimestampTzNsType)

	return ok
}

func (TimestampTzNsType) primitive()     {}
func (TimestampTzNsType) Type() string   { return "timestamptz_ns" }
func (TimestampTzNsType) String() string { return "timestamptz_ns" }

type UnknownType struct{}

func (UnknownType) Equals(other Type) bool {
	_, ok := other.(UnknownType)

	return ok
}

func (UnknownType) primitive()     {}
func (UnknownType) Type() string   { return "unknown" }
func (UnknownType) String() string { return "unknown" }

// VariantType represents semi-structured data stored using the Parquet Variant
// binary encoding. Requires Iceberg format version 3+.
type VariantType struct{}

func (VariantType) Equals(other Type) bool {
	_, ok := other.(VariantType)

	return ok
}

func (VariantType) Type() string   { return "variant" }
func (VariantType) String() string { return "variant" }

// DefaultGeoCRS is the CRS of the geometry and geography types when no CRS is
// given; a Parquet GEOMETRY or GEOGRAPHY logical type without a CRS means the
// same value.
const DefaultGeoCRS = "OGC:CRS84"

type GeometryType struct {
	crs string
}

func GeometryTypeOf(crs string) (GeometryType, error) {
	if crs == "" {
		return GeometryType{}, fmt.Errorf("%w: invalid CRS: (empty string)", ErrInvalidTypeString)
	}
	crs = strings.TrimSpace(crs)
	if crs == DefaultGeoCRS {
		return GeometryType{}, nil
	}

	return GeometryType{crs: crs}, nil
}

func (g GeometryType) CRS() string {
	if g.crs == "" {
		return DefaultGeoCRS
	}

	return g.crs
}

func (g GeometryType) Equals(other Type) bool {
	rhs, ok := other.(GeometryType)
	if !ok {
		return false
	}

	return g.crs == rhs.crs
}

func (GeometryType) primitive() {}
func (g GeometryType) Type() string {
	if g.crs == "" {
		return "geometry"
	}

	return fmt.Sprintf("geometry(%s)", g.crs)
}

func (g GeometryType) String() string {
	return g.Type()
}

const defaultGeographyAlgorithm = string(geoarrow.EdgeSpherical)

func toGeoArrowEdgeInterpolation(s string) (geoarrow.EdgeInterpolation, error) {
	algo := geoarrow.EdgeInterpolation(strings.ToLower(strings.TrimSpace(s)))
	switch algo {
	case geoarrow.EdgeSpherical, geoarrow.EdgeVincenty,
		geoarrow.EdgeThomas, geoarrow.EdgeAndoyer, geoarrow.EdgeKarney:
		return algo, nil
	default:
		return "", fmt.Errorf("%w: invalid edge interpolation algorithm", ErrInvalidTypeString)
	}
}

type GeographyType struct {
	crs       string
	algorithm string
}

func GeographyTypeOf(crs string, algorithm string) (GeographyType, error) {
	if crs == "" {
		return GeographyType{}, fmt.Errorf("%w: invalid CRS: (empty string)", ErrInvalidTypeString)
	}
	if algorithm == "" {
		return GeographyType{}, fmt.Errorf("%w: invalid algorithm: (empty string)", ErrInvalidTypeString)
	}
	crs = strings.TrimSpace(crs)
	normalizedCRS := crs
	if normalizedCRS == DefaultGeoCRS {
		normalizedCRS = ""
	}

	validatedAlg, err := toGeoArrowEdgeInterpolation(algorithm) // validate algorithm
	if err != nil {
		return GeographyType{}, fmt.Errorf("invalid algorithm: %w", err)
	}
	normalizedAlgorithm := string(validatedAlg)
	if validatedAlg == geoarrow.EdgeInterpolation(defaultGeographyAlgorithm) {
		normalizedAlgorithm = ""
	}

	return GeographyType{crs: normalizedCRS, algorithm: normalizedAlgorithm}, nil
}

func (g GeographyType) CRS() string {
	if g.crs == "" {
		return DefaultGeoCRS
	}

	return g.crs
}

func (g GeographyType) Algorithm() string {
	if g.algorithm == "" {
		return defaultGeographyAlgorithm
	}

	return g.algorithm
}

func (g GeographyType) Equals(other Type) bool {
	rhs, ok := other.(GeographyType)
	if !ok {
		return false
	}

	return g.crs == rhs.crs && g.algorithm == rhs.algorithm
}

func (GeographyType) primitive() {}
func (g GeographyType) Type() string {
	hasCRS := g.crs != ""
	hasAlgo := g.algorithm != ""

	if !hasCRS && !hasAlgo {
		return "geography"
	}
	if hasCRS && !hasAlgo {
		return fmt.Sprintf("geography(%s)", g.crs)
	}
	if !hasCRS && hasAlgo {
		return fmt.Sprintf("geography(%s, %s)", DefaultGeoCRS, g.algorithm)
	}

	return fmt.Sprintf("geography(%s, %s)", g.crs, g.algorithm)
}

func (g GeographyType) String() string {
	return g.Type()
}

var PrimitiveTypes = struct {
	Bool          PrimitiveType
	Int32         PrimitiveType
	Int64         PrimitiveType
	Float32       PrimitiveType
	Float64       PrimitiveType
	Date          PrimitiveType
	Time          PrimitiveType
	Timestamp     PrimitiveType
	TimestampTz   PrimitiveType
	TimestampNs   PrimitiveType
	TimestampTzNs PrimitiveType
	String        PrimitiveType
	Binary        PrimitiveType
	UUID          PrimitiveType
	Unknown       PrimitiveType
}{
	Bool:          BooleanType{},
	Int32:         Int32Type{},
	Int64:         Int64Type{},
	Float32:       Float32Type{},
	Float64:       Float64Type{},
	Date:          DateType{},
	Time:          TimeType{},
	Timestamp:     TimestampType{},
	TimestampTz:   TimestampTzType{},
	TimestampNs:   TimestampNsType{},
	TimestampTzNs: TimestampTzNsType{},
	String:        StringType{},
	Binary:        BinaryType{},
	UUID:          UUIDType{},
	Unknown:       UnknownType{},
}

// PromoteType promotes the type being read from a file to a requested read type.
// fileType is the type from the file being read
// readType is the requested readType
func PromoteType(fileType, readType Type) (Type, error) {
	switch t := fileType.(type) {
	case Int32Type:
		if _, ok := readType.(Int64Type); ok {
			return readType, nil
		}
	case Float32Type:
		if _, ok := readType.(Float64Type); ok {
			return readType, nil
		}
	case StringType:
		if _, ok := readType.(BinaryType); ok {
			return readType, nil
		}
	case BinaryType:
		if _, ok := readType.(StringType); ok {
			return readType, nil
		}
	case DecimalType:
		if rt, ok := readType.(DecimalType); ok {
			// Only precision may widen; scale must stay the same (spec: widen precision only).
			if t.scale == rt.scale && t.precision <= rt.precision {
				return readType, nil
			}

			return nil, fmt.Errorf("%w: cannot promote %s to %s",
				ErrResolve, fileType, readType)
		}
	case FixedType:
		if _, ok := readType.(UUIDType); ok && t.len == 16 {
			return readType, nil
		}
	case UnknownType:
		return readType, nil
	default:
		if fileType.Equals(readType) {
			return fileType, nil
		}
	}

	return nil, fmt.Errorf("%w: cannot promote %s to %s", ErrResolve, fileType, readType)
}
