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

package udf

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/apache/iceberg-go"
)

// ErrInvalidUDFType is returned when a UDF parameter or return type
// does not conform to the UDF spec's Types section.
var ErrInvalidUDFType = errors.New("invalid UDF type")

// Type represents a UDF parameter or return type as defined by the UDF
// spec. UDF types are Iceberg types without field IDs: primitives are
// encoded as plain strings (e.g. "int", "decimal(9,2)") and nested types
// (list, map, struct) as JSON objects carrying only the fields the UDF
// spec requires.
//
// String returns the canonical string representation used to build
// definition IDs (see CanonicalDefinitionID).
type Type interface {
	fmt.Stringer

	Equals(Type) bool

	// canonicalTo writes the canonical string form, sealing the
	// interface to the implementations in this package.
	canonicalTo(sb *strings.Builder)
	clone() Type
}

// PrimitiveType is a primitive or semi-structured UDF type, stored as the
// verbatim type string (e.g. "int", "decimal(9,2)", "variant"). Values are
// always validated: construct them with PrimitiveTypeOf.
type PrimitiveType struct {
	name string
}

// PrimitiveTypeOf validates s against the Iceberg type system and the UDF
// spec's rules for type strings (no spaces or quote characters) and returns
// the corresponding PrimitiveType.
func PrimitiveTypeOf(s string) (PrimitiveType, error) {
	if s == "" {
		return PrimitiveType{}, fmt.Errorf("%w: type string must not be empty", ErrInvalidUDFType)
	}

	if strings.ContainsFunc(s, unicode.IsSpace) || strings.ContainsAny(s, `"'`) {
		return PrimitiveType{}, fmt.Errorf("%w: type string %q must not contain spaces or quote characters",
			ErrInvalidUDFType, s)
	}

	// Validate through the iceberg type parser so the accepted set of
	// primitive type strings always matches the rest of the module.
	// NestedField is the exported entry point to that parser.
	payload, err := json.Marshal(map[string]any{"id": 1, "name": "f", "required": false, "type": s})
	if err != nil {
		return PrimitiveType{}, err
	}
	var field iceberg.NestedField
	if err := json.Unmarshal(payload, &field); err != nil {
		return PrimitiveType{}, fmt.Errorf("%w: %q is not a valid Iceberg type string: %w", ErrInvalidUDFType, s, err)
	}

	return PrimitiveType{name: s}, nil
}

func (p PrimitiveType) String() string { return p.name }

func (p PrimitiveType) Equals(other Type) bool {
	o, ok := other.(PrimitiveType)

	return ok && p.name == o.name
}

func (p PrimitiveType) canonicalTo(sb *strings.Builder) { sb.WriteString(p.name) }
func (p PrimitiveType) clone() Type                     { return p }

func (p PrimitiveType) MarshalJSON() ([]byte, error) { return json.Marshal(p.name) }

// ListType is a UDF list type carrying only its element type.
type ListType struct {
	Element Type
}

func (l ListType) String() string { return canonicalString(l) }

func (l ListType) Equals(other Type) bool {
	o, ok := other.(ListType)

	return ok && l.Element.Equals(o.Element)
}

func (l ListType) canonicalTo(sb *strings.Builder) {
	sb.WriteString("list<")
	l.Element.canonicalTo(sb)
	sb.WriteString(">")
}

func (l ListType) clone() Type { return ListType{Element: cloneType(l.Element)} }

func (l ListType) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type    string `json:"type"`
		Element Type   `json:"element"`
	}{Type: "list", Element: l.Element})
}

// MapType is a UDF map type carrying only its key and value types.
type MapType struct {
	Key   Type
	Value Type
}

func (m MapType) String() string { return canonicalString(m) }

func (m MapType) Equals(other Type) bool {
	o, ok := other.(MapType)

	return ok && m.Key.Equals(o.Key) && m.Value.Equals(o.Value)
}

func (m MapType) canonicalTo(sb *strings.Builder) {
	sb.WriteString("map<")
	m.Key.canonicalTo(sb)
	sb.WriteString(",")
	m.Value.canonicalTo(sb)
	sb.WriteString(">")
}

func (m MapType) clone() Type {
	return MapType{Key: cloneType(m.Key), Value: cloneType(m.Value)}
}

func (m MapType) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type  string `json:"type"`
		Key   Type   `json:"key"`
		Value Type   `json:"value"`
	}{Type: "map", Key: m.Key, Value: m.Value})
}

// StructField is a single field of a UDF struct type, carrying only a name
// and a type.
type StructField struct {
	Name string
	Type Type
}

// StructType is a UDF struct type carrying only its fields.
type StructType struct {
	Fields []StructField
}

func (s StructType) String() string { return canonicalString(s) }

func (s StructType) Equals(other Type) bool {
	o, ok := other.(StructType)
	if !ok || len(s.Fields) != len(o.Fields) {
		return false
	}
	for i, f := range s.Fields {
		if f.Name != o.Fields[i].Name || !f.Type.Equals(o.Fields[i].Type) {
			return false
		}
	}

	return true
}

func (s StructType) canonicalTo(sb *strings.Builder) {
	sb.WriteString("struct<")
	for i, f := range s.Fields {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(f.Name)
		sb.WriteString(":")
		f.Type.canonicalTo(sb)
	}
	sb.WriteString(">")
}

func (s StructType) clone() Type {
	fields := slices.Clone(s.Fields)
	for i := range fields {
		fields[i].Type = cloneType(fields[i].Type)
	}

	return StructType{Fields: fields}
}

func (s StructType) MarshalJSON() ([]byte, error) {
	fields := make([]struct {
		Name string `json:"name"`
		Type Type   `json:"type"`
	}, len(s.Fields))
	for i, f := range s.Fields {
		fields[i].Name, fields[i].Type = f.Name, f.Type
	}

	return json.Marshal(struct {
		Type   string `json:"type"`
		Fields any    `json:"fields"`
	}{Type: "struct", Fields: fields})
}

func canonicalString(t Type) string {
	var sb strings.Builder
	t.canonicalTo(&sb)

	return sb.String()
}

func cloneType(t Type) Type {
	if t == nil {
		return nil
	}

	return t.clone()
}

// validateType checks that t and all nested types are fully specified.
// PrimitiveType values are valid by construction; composite types built as
// literals may carry nil children, which this rejects.
func validateType(t Type) error {
	switch v := t.(type) {
	case nil:
		return fmt.Errorf("%w: type must not be nil", ErrInvalidUDFType)
	case PrimitiveType:
		if v.name == "" {
			return fmt.Errorf("%w: uninitialized primitive type; use PrimitiveTypeOf", ErrInvalidUDFType)
		}

		return nil
	case ListType:
		return validateType(v.Element)
	case MapType:
		if err := validateType(v.Key); err != nil {
			return err
		}

		return validateType(v.Value)
	case StructType:
		seen := make(map[string]bool)
		for _, f := range v.Fields {
			if f.Name == "" {
				return fmt.Errorf("%w: struct field name must not be empty", ErrInvalidUDFType)
			}
			if seen[f.Name] {
				return fmt.Errorf("%w: struct has duplicate field name %q", ErrInvalidUDFType, f.Name)
			}
			seen[f.Name] = true
			if err := validateType(f.Type); err != nil {
				return err
			}
		}

		return nil
	default:
		return fmt.Errorf("%w: unsupported type %T", ErrInvalidUDFType, t)
	}
}

// unmarshalType parses a UDF type from its JSON form: either a primitive
// type string or an object for list, map, and struct types. Fields beyond
// the ones the UDF spec requires are ignored.
func unmarshalType(b []byte) (Type, error) {
	var name string
	if err := json.Unmarshal(b, &name); err == nil {
		return PrimitiveTypeOf(name)
	}

	aux := struct {
		Type string `json:"type"`
	}{}
	if err := json.Unmarshal(b, &aux); err != nil {
		return nil, fmt.Errorf("%w: type must be a string or an object: %w", ErrInvalidUDFType, err)
	}

	switch aux.Type {
	case "list":
		lst := struct {
			Element json.RawMessage `json:"element"`
		}{}
		if err := json.Unmarshal(b, &lst); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidUDFType, err)
		}
		if len(lst.Element) == 0 {
			return nil, fmt.Errorf("%w: list type requires an element type", ErrInvalidUDFType)
		}
		elem, err := unmarshalType(lst.Element)
		if err != nil {
			return nil, err
		}

		return ListType{Element: elem}, nil
	case "map":
		mp := struct {
			Key   json.RawMessage `json:"key"`
			Value json.RawMessage `json:"value"`
		}{}
		if err := json.Unmarshal(b, &mp); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidUDFType, err)
		}
		if len(mp.Key) == 0 || len(mp.Value) == 0 {
			return nil, fmt.Errorf("%w: map type requires key and value types", ErrInvalidUDFType)
		}
		key, err := unmarshalType(mp.Key)
		if err != nil {
			return nil, err
		}
		value, err := unmarshalType(mp.Value)
		if err != nil {
			return nil, err
		}

		return MapType{Key: key, Value: value}, nil
	case "struct":
		st := struct {
			Fields []struct {
				Name string          `json:"name"`
				Type json.RawMessage `json:"type"`
			} `json:"fields"`
		}{}
		if err := json.Unmarshal(b, &st); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidUDFType, err)
		}
		if st.Fields == nil {
			return nil, fmt.Errorf("%w: struct type requires fields", ErrInvalidUDFType)
		}
		fields := make([]StructField, len(st.Fields))
		for i, f := range st.Fields {
			if f.Name == "" {
				return nil, fmt.Errorf("%w: struct field requires a name", ErrInvalidUDFType)
			}
			if len(f.Type) == 0 {
				return nil, fmt.Errorf("%w: struct field %q requires a type", ErrInvalidUDFType, f.Name)
			}
			typ, err := unmarshalType(f.Type)
			if err != nil {
				return nil, err
			}
			fields[i] = StructField{Name: f.Name, Type: typ}
		}

		return StructType{Fields: fields}, nil
	default:
		return nil, fmt.Errorf("%w: unsupported type object %q", ErrInvalidUDFType, aux.Type)
	}
}

// CanonicalDefinitionID derives the definition-id for a parameter list: the
// canonical string forms of the parameter types joined by commas with no
// spaces, e.g. "int,list<int>,struct<id:int,name:string>". A definition
// with no parameters has the empty string as its definition-id.
func CanonicalDefinitionID(params []Parameter) (string, error) {
	var sb strings.Builder
	for i, p := range params {
		if err := validateType(p.Type); err != nil {
			return "", err
		}
		if i > 0 {
			sb.WriteString(",")
		}
		p.Type.canonicalTo(&sb)
	}

	return sb.String(), nil
}
