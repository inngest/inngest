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

// Package udf provides the metadata model for Iceberg SQL UDFs as
// specified by the Iceberg UDF spec.
// https://iceberg.apache.org/udf-spec/
package udf

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/apache/iceberg-go"

	"github.com/google/uuid"
)

var (
	ErrInvalidUDFMetadata              = errors.New("invalid UDF metadata")
	ErrInvalidUDFMetadataFormatVersion = errors.New("invalid or missing format-version in UDF metadata")
)

const (
	SupportedUDFFormatVersion = 1

	initialVersionID = 1
)

// FunctionType indicates whether a definition is a scalar function (udf)
// or a table function (udtf).
type FunctionType string

const (
	FunctionTypeUDF  FunctionType = "udf"
	FunctionTypeUDTF FunctionType = "udtf"
)

// OnNullInput defines how a UDF behaves when any input parameter is NULL.
type OnNullInput string

const (
	// OnNullInputCall means the function may handle NULLs internally, so
	// engines must execute it even if some inputs are NULL. This is the
	// default behavior when on-null-input is absent.
	OnNullInputCall OnNullInput = "call"
	// OnNullInputReturnNull means the function always returns NULL if any
	// input argument is NULL, allowing engines to skip evaluation.
	OnNullInputReturnNull OnNullInput = "return-null"
)

// Representation is a single implementation of a definition version. The
// only representation type defined by the UDF spec is SQL; unrecognized
// types round-trip through UnknownRepresentation.
type Representation interface {
	// RepresentationType returns the representation's type discriminator,
	// e.g. "sql".
	RepresentationType() string

	// isRepresentation seals the interface to the implementations in
	// this package.
	isRepresentation()
	clone() Representation
}

// SQLRepresentation stores the function body as a SQL expression in a
// specific dialect. The SQL must reference parameters using the names
// declared in the definition's parameters.
type SQLRepresentation struct {
	Dialect string `json:"dialect"`
	SQL     string `json:"sql"`
}

// NewSQLRepresentation creates a SQL representation from a dialect
// identifier (e.g. "spark", "trino") and a SQL expression.
func NewSQLRepresentation(dialect, sql string) (SQLRepresentation, error) {
	if dialect == "" {
		return SQLRepresentation{}, fmt.Errorf("%w: sql representation requires a dialect", ErrInvalidUDFMetadata)
	}
	if sql == "" {
		return SQLRepresentation{}, fmt.Errorf("%w: sql representation requires a sql expression", ErrInvalidUDFMetadata)
	}

	return SQLRepresentation{Dialect: dialect, SQL: sql}, nil
}

func (SQLRepresentation) RepresentationType() string { return "sql" }
func (SQLRepresentation) isRepresentation()          {}
func (r SQLRepresentation) clone() Representation    { return r }

func (r SQLRepresentation) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type    string `json:"type"`
		Dialect string `json:"dialect"`
		SQL     string `json:"sql"`
	}{Type: "sql", Dialect: r.Dialect, SQL: r.SQL})
}

// UnknownRepresentation preserves a representation of an unrecognized type
// so that metadata written by newer or extended writers round-trips intact.
type UnknownRepresentation struct {
	TypeName string

	raw json.RawMessage
}

// Raw returns a copy of the representation's original JSON.
func (r UnknownRepresentation) Raw() json.RawMessage { return slices.Clone(r.raw) }

func (r UnknownRepresentation) RepresentationType() string { return r.TypeName }
func (UnknownRepresentation) isRepresentation()            {}
func (r UnknownRepresentation) clone() Representation {
	r.raw = slices.Clone(r.raw)

	return r
}

func (r UnknownRepresentation) MarshalJSON() ([]byte, error) {
	return slices.Clone(r.raw), nil
}

func representationsEqual(a, b Representation) bool {
	switch av := a.(type) {
	case SQLRepresentation:
		bv, ok := b.(SQLRepresentation)

		return ok && av == bv
	case UnknownRepresentation:
		bv, ok := b.(UnknownRepresentation)

		return ok && av.TypeName == bv.TypeName && bytes.Equal(av.raw, bv.raw)
	default:
		return false
	}
}

func cloneRepresentations(representations []Representation) []Representation {
	cloned := slices.Clone(representations)
	for i, repr := range representations {
		if repr != nil {
			cloned[i] = repr.clone()
		}
	}

	return cloned
}

func unmarshalRepresentation(b json.RawMessage) (Representation, error) {
	aux := struct {
		Type string `json:"type"`
	}{}
	if err := json.Unmarshal(b, &aux); err != nil {
		return nil, err
	}
	if aux.Type == "" {
		return nil, fmt.Errorf("%w: representation requires a type", ErrInvalidUDFMetadata)
	}

	if aux.Type == "sql" {
		var r SQLRepresentation
		if err := json.Unmarshal(b, &r); err != nil {
			return nil, err
		}

		return r, nil
	}

	// Store the raw JSON compacted so that round-tripped representations
	// compare equal: encoding/json compacts MarshalJSON output.
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, b); err != nil {
		return nil, err
	}

	return UnknownRepresentation{TypeName: aux.Type, raw: compacted.Bytes()}, nil
}

// Parameter is a single function parameter.
type Parameter struct {
	Name string `json:"name"`
	Type Type   `json:"type"`
	Doc  string `json:"doc,omitempty"`
}

func (p *Parameter) UnmarshalJSON(b []byte) error {
	type Alias Parameter
	next := Parameter{}
	aux := struct {
		Type json.RawMessage `json:"type"`
		*Alias
	}{Alias: (*Alias)(&next)}

	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	if len(aux.Type) == 0 {
		return fmt.Errorf("%w: parameter %q requires a type", ErrInvalidUDFMetadata, next.Name)
	}

	typ, err := unmarshalType(aux.Type)
	if err != nil {
		return err
	}
	next.Type = typ
	*p = next

	return nil
}

func (p Parameter) equals(other Parameter) bool {
	return p.Name == other.Name && p.Doc == other.Doc &&
		p.Type != nil && other.Type != nil && p.Type.Equals(other.Type)
}

// DefinitionVersion is a specific implementation of a definition at a
// given point in time. Versions are immutable: changes to a definition
// introduce a new version.
type DefinitionVersion struct {
	// VersionID is a monotonically increasing identifier of the
	// definition version.
	VersionID int `json:"version-id"`
	// Representations lists the UDF implementations of this version.
	Representations []Representation `json:"representations"`
	// Deterministic indicates whether the function is deterministic.
	// Defaults to false.
	Deterministic bool `json:"deterministic,omitempty"`
	// OnNullInput defines how the UDF behaves when any input parameter
	// is NULL. Defaults to OnNullInputCall when empty.
	OnNullInput OnNullInput `json:"on-null-input,omitempty"`
	// TimestampMS is the creation timestamp of this version (ms from epoch).
	TimestampMS int64 `json:"timestamp-ms"`
}

func (v *DefinitionVersion) UnmarshalJSON(b []byte) error {
	type Alias DefinitionVersion
	next := DefinitionVersion{VersionID: -1, TimestampMS: -1}
	aux := struct {
		Representations []json.RawMessage `json:"representations"`
		*Alias
	}{Alias: (*Alias)(&next)}

	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	next.Representations = make([]Representation, len(aux.Representations))
	for i, raw := range aux.Representations {
		repr, err := unmarshalRepresentation(raw)
		if err != nil {
			return err
		}
		next.Representations[i] = repr
	}
	*v = next

	return nil
}

// NullInputBehavior returns the effective on-null-input behavior,
// applying the spec default (OnNullInputCall) when the field is absent.
func (v *DefinitionVersion) NullInputBehavior() OnNullInput {
	if v.OnNullInput == "" {
		return OnNullInputCall
	}

	return v.OnNullInput
}

// Equals compares two versions for semantic equality: the default
// on-null-input behavior compares equal to an explicit "call".
func (v *DefinitionVersion) Equals(other *DefinitionVersion) bool {
	return v.VersionID == other.VersionID &&
		v.Deterministic == other.Deterministic &&
		v.NullInputBehavior() == other.NullInputBehavior() &&
		v.TimestampMS == other.TimestampMS &&
		slices.EqualFunc(v.Representations, other.Representations, representationsEqual)
}

func (v *DefinitionVersion) Clone() *DefinitionVersion {
	if v == nil {
		return nil
	}

	cloned := *v
	cloned.Representations = cloneRepresentations(v.Representations)

	return &cloned
}

func (v *DefinitionVersion) validate(definitionID string) error {
	if v.VersionID == -1 {
		return fmt.Errorf("%w: definition %q has a version missing version-id",
			ErrInvalidUDFMetadata, definitionID)
	}
	if v.VersionID < 0 {
		return fmt.Errorf("%w: definition %q has invalid negative version-id %d",
			ErrInvalidUDFMetadata, definitionID, v.VersionID)
	}

	if v.TimestampMS == -1 {
		return fmt.Errorf("%w: definition %q version %d is missing timestamp-ms",
			ErrInvalidUDFMetadata, definitionID, v.VersionID)
	}
	if v.TimestampMS < 0 {
		return fmt.Errorf("%w: definition %q version %d has invalid negative timestamp-ms %d",
			ErrInvalidUDFMetadata, definitionID, v.VersionID, v.TimestampMS)
	}

	if len(v.Representations) == 0 {
		return fmt.Errorf("%w: definition %q version %d must have at least one representation",
			ErrInvalidUDFMetadata, definitionID, v.VersionID)
	}

	switch v.OnNullInput {
	case "", OnNullInputCall, OnNullInputReturnNull:
	default:
		return fmt.Errorf("%w: definition %q version %d has invalid on-null-input %q",
			ErrInvalidUDFMetadata, definitionID, v.VersionID, v.OnNullInput)
	}

	seenDialects := make(map[string]bool)
	for _, repr := range v.Representations {
		if repr == nil {
			return fmt.Errorf("%w: definition %q version %d representations must not contain null entries",
				ErrInvalidUDFMetadata, definitionID, v.VersionID)
		}
		if unknown, ok := repr.(UnknownRepresentation); ok && len(unknown.raw) == 0 {
			return fmt.Errorf("%w: definition %q version %d has an unknown representation without raw JSON; unknown representations must originate from parsed metadata",
				ErrInvalidUDFMetadata, definitionID, v.VersionID)
		}
		sqlRepr, ok := repr.(SQLRepresentation)
		if !ok {
			continue
		}
		if sqlRepr.Dialect == "" {
			return fmt.Errorf("%w: definition %q version %d has a sql representation without a dialect",
				ErrInvalidUDFMetadata, definitionID, v.VersionID)
		}
		if sqlRepr.SQL == "" {
			return fmt.Errorf("%w: definition %q version %d has a sql representation without a sql expression",
				ErrInvalidUDFMetadata, definitionID, v.VersionID)
		}
		dialect := strings.ToLower(sqlRepr.Dialect)
		if seenDialects[dialect] {
			return fmt.Errorf("%w: definition %q version %d has duplicate sql dialect %s",
				ErrInvalidUDFMetadata, definitionID, v.VersionID, sqlRepr.Dialect)
		}
		seenDialects[dialect] = true
	}

	return nil
}

// Definition represents one function signature (e.g. add_one(int) vs
// add_one(float)). A definition is uniquely identified by its signature:
// the ordered list of parameter types, canonicalized as its DefinitionID.
type Definition struct {
	// DefinitionID is the canonical string derived from the parameter
	// types (see CanonicalDefinitionID).
	DefinitionID string `json:"definition-id"`
	// SpecificName is an optional user-assignable name for this
	// definition, unique among all definitions of the function.
	SpecificName string `json:"specific-name,omitempty"`
	// Parameters is the ordered list of function parameters. Invocation
	// order must match this list.
	Parameters []Parameter `json:"parameters"`
	// ReturnType is the declared return type. All versions must produce
	// values of this type.
	ReturnType Type `json:"return-type"`
	// ReturnNullable hints whether the return value is nullable.
	// Defaults to true when absent.
	ReturnNullable *bool `json:"return-nullable,omitempty"`
	// Versions lists the versioned implementations of this definition.
	Versions []*DefinitionVersion `json:"versions"`
	// CurrentVersionID identifies the current version of this definition.
	CurrentVersionID int `json:"current-version-id"`
	// FunctionType is "udf" for scalar functions or "udtf" for table
	// functions. For "udtf", ReturnType must be a struct describing the
	// output schema.
	FunctionType FunctionType `json:"function-type"`
	// Doc is an optional documentation string.
	Doc string `json:"doc,omitempty"`
}

func (d *Definition) UnmarshalJSON(b []byte) error {
	type Alias Definition
	next := Definition{CurrentVersionID: -1}
	// definition-id and parameters are shadowed to distinguish a required
	// field that is absent from one that is legitimately empty: a
	// zero-parameter definition has "" as its definition-id and [] as its
	// parameters.
	aux := struct {
		DefinitionID *string         `json:"definition-id"`
		Parameters   json.RawMessage `json:"parameters"`
		ReturnType   json.RawMessage `json:"return-type"`
		*Alias
	}{Alias: (*Alias)(&next)}

	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	if aux.DefinitionID == nil {
		return fmt.Errorf("%w: definition is missing definition-id", ErrInvalidUDFMetadata)
	}
	next.DefinitionID = *aux.DefinitionID

	if len(aux.Parameters) == 0 {
		return fmt.Errorf("%w: definition %q is missing parameters", ErrInvalidUDFMetadata, next.DefinitionID)
	}
	if err := json.Unmarshal(aux.Parameters, &next.Parameters); err != nil {
		return err
	}
	if next.Parameters == nil {
		return fmt.Errorf("%w: definition %q is missing parameters", ErrInvalidUDFMetadata, next.DefinitionID)
	}

	if len(aux.ReturnType) == 0 {
		return fmt.Errorf("%w: definition %q requires a return-type", ErrInvalidUDFMetadata, next.DefinitionID)
	}

	typ, err := unmarshalType(aux.ReturnType)
	if err != nil {
		return err
	}
	next.ReturnType = typ
	*d = next

	return nil
}

// ReturnsNullable returns the effective return-nullable hint, applying
// the spec default (true) when the field is absent.
func (d *Definition) ReturnsNullable() bool {
	return d.ReturnNullable == nil || *d.ReturnNullable
}

// CurrentVersion returns the definition version identified by
// CurrentVersionID, or nil if no such version exists.
func (d *Definition) CurrentVersion() *DefinitionVersion {
	return d.Version(d.CurrentVersionID)
}

// Version returns the definition version with the given id, or nil if no
// such version exists.
func (d *Definition) Version(versionID int) *DefinitionVersion {
	for _, v := range d.Versions {
		if v.VersionID == versionID {
			return v
		}
	}

	return nil
}

// Equals compares two definitions for semantic equality: absent optional
// fields compare equal to their spec defaults.
func (d *Definition) Equals(other *Definition) bool {
	return d.DefinitionID == other.DefinitionID &&
		d.SpecificName == other.SpecificName &&
		slices.EqualFunc(d.Parameters, other.Parameters, Parameter.equals) &&
		d.ReturnType != nil && other.ReturnType != nil && d.ReturnType.Equals(other.ReturnType) &&
		d.ReturnsNullable() == other.ReturnsNullable() &&
		d.CurrentVersionID == other.CurrentVersionID &&
		d.FunctionType == other.FunctionType &&
		d.Doc == other.Doc &&
		slices.EqualFunc(d.Versions, other.Versions, (*DefinitionVersion).Equals)
}

func (d *Definition) Clone() *Definition {
	if d == nil {
		return nil
	}

	cloned := *d
	cloned.Parameters = make([]Parameter, len(d.Parameters))
	for i, parameter := range d.Parameters {
		cloned.Parameters[i] = parameter
		cloned.Parameters[i].Type = cloneType(parameter.Type)
	}
	cloned.ReturnType = cloneType(d.ReturnType)
	cloned.Versions = cloneSlice(d.Versions)
	if d.ReturnNullable != nil {
		nullable := *d.ReturnNullable
		cloned.ReturnNullable = &nullable
	}

	return &cloned
}

func (d *Definition) validate() error {
	seenNames := make(map[string]bool)
	for _, p := range d.Parameters {
		if p.Name == "" {
			return fmt.Errorf("%w: definition %q has a parameter without a name",
				ErrInvalidUDFMetadata, d.DefinitionID)
		}
		if seenNames[p.Name] {
			return fmt.Errorf("%w: definition %q has duplicate parameter name %q",
				ErrInvalidUDFMetadata, d.DefinitionID, p.Name)
		}
		seenNames[p.Name] = true
		if err := validateType(p.Type); err != nil {
			return fmt.Errorf("%w: definition %q parameter %q: %w",
				ErrInvalidUDFMetadata, d.DefinitionID, p.Name, err)
		}
	}

	canonicalID, err := CanonicalDefinitionID(d.Parameters)
	if err != nil {
		return fmt.Errorf("%w: definition %q: %w", ErrInvalidUDFMetadata, d.DefinitionID, err)
	}
	if d.DefinitionID != canonicalID {
		return fmt.Errorf("%w: definition-id %q does not match canonical form %q of its parameters",
			ErrInvalidUDFMetadata, d.DefinitionID, canonicalID)
	}

	if err := validateType(d.ReturnType); err != nil {
		return fmt.Errorf("%w: definition %q return-type: %w", ErrInvalidUDFMetadata, d.DefinitionID, err)
	}

	switch d.FunctionType {
	case FunctionTypeUDF:
	case FunctionTypeUDTF:
		st, ok := d.ReturnType.(StructType)
		if !ok {
			return fmt.Errorf("%w: definition %q is a udtf but its return-type is not a struct",
				ErrInvalidUDFMetadata, d.DefinitionID)
		}
		// The struct describes the output schema; a table without columns
		// is meaningless.
		if len(st.Fields) == 0 {
			return fmt.Errorf("%w: definition %q is a udtf but its return-type struct has no fields",
				ErrInvalidUDFMetadata, d.DefinitionID)
		}
	case "":
		return fmt.Errorf("%w: definition %q is missing function-type", ErrInvalidUDFMetadata, d.DefinitionID)
	default:
		return fmt.Errorf("%w: definition %q has invalid function-type %q (must be %q or %q)",
			ErrInvalidUDFMetadata, d.DefinitionID, d.FunctionType, FunctionTypeUDF, FunctionTypeUDTF)
	}

	if len(d.Versions) == 0 {
		return fmt.Errorf("%w: definition %q must have at least one version", ErrInvalidUDFMetadata, d.DefinitionID)
	}

	seenVersions := make(map[int]bool)
	for _, v := range d.Versions {
		if v == nil {
			return fmt.Errorf("%w: definition %q versions must not contain null entries",
				ErrInvalidUDFMetadata, d.DefinitionID)
		}
		if err := v.validate(d.DefinitionID); err != nil {
			return err
		}
		if seenVersions[v.VersionID] {
			return fmt.Errorf("%w: definition %q has duplicate version-id %d",
				ErrInvalidUDFMetadata, d.DefinitionID, v.VersionID)
		}
		seenVersions[v.VersionID] = true
	}

	if d.CurrentVersionID == -1 {
		return fmt.Errorf("%w: definition %q is missing current-version-id", ErrInvalidUDFMetadata, d.DefinitionID)
	}
	if d.CurrentVersionID < 0 {
		return fmt.Errorf("%w: definition %q has invalid negative current-version-id %d",
			ErrInvalidUDFMetadata, d.DefinitionID, d.CurrentVersionID)
	}
	if !seenVersions[d.CurrentVersionID] {
		return fmt.Errorf("%w: definition %q current-version-id %d not found in versions",
			ErrInvalidUDFMetadata, d.DefinitionID, d.CurrentVersionID)
	}

	return nil
}

// DefinitionVersionRef maps a definition to a selected version.
type DefinitionVersionRef struct {
	// DefinitionID may be empty: a zero-parameter definition has the
	// empty string as its definition-id.
	DefinitionID string `json:"definition-id"`
	VersionID    int    `json:"version-id"`
}

func (r *DefinitionVersionRef) UnmarshalJSON(b []byte) error {
	type Alias DefinitionVersionRef
	next := DefinitionVersionRef{VersionID: -1}
	// definition-id is shadowed to distinguish the required field being
	// absent from the legitimately empty id of a zero-parameter definition.
	aux := struct {
		DefinitionID *string `json:"definition-id"`
		*Alias
	}{Alias: (*Alias)(&next)}

	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	if aux.DefinitionID == nil {
		return fmt.Errorf("%w: definition-log reference is missing definition-id", ErrInvalidUDFMetadata)
	}
	next.DefinitionID = *aux.DefinitionID
	*r = next

	return nil
}

// DefinitionLogEntry records the selected version of every definition at
// the time of a change, enabling rollback without external state.
type DefinitionLogEntry struct {
	// TimestampMS is when the function was updated to use these
	// definition versions (ms from epoch).
	TimestampMS int64 `json:"timestamp-ms"`
	// DefinitionVersions maps each definition to its selected version at
	// this time.
	DefinitionVersions []DefinitionVersionRef `json:"definition-versions"`
}

func (e *DefinitionLogEntry) UnmarshalJSON(b []byte) error {
	type Alias DefinitionLogEntry
	next := DefinitionLogEntry{TimestampMS: -1}

	if err := json.Unmarshal(b, (*Alias)(&next)); err != nil {
		return err
	}
	*e = next

	return nil
}

func (e DefinitionLogEntry) validate() error {
	if e.TimestampMS == -1 {
		return fmt.Errorf("%w: definition-log entry is missing timestamp-ms", ErrInvalidUDFMetadata)
	}
	if e.TimestampMS < 0 {
		return fmt.Errorf("%w: definition-log entry has invalid negative timestamp-ms %d",
			ErrInvalidUDFMetadata, e.TimestampMS)
	}

	if len(e.DefinitionVersions) == 0 {
		return fmt.Errorf("%w: definition-log entry at timestamp %d must have definition-versions",
			ErrInvalidUDFMetadata, e.TimestampMS)
	}

	seen := make(map[string]bool)
	for _, ref := range e.DefinitionVersions {
		if ref.VersionID == -1 {
			return fmt.Errorf("%w: definition-log entry at timestamp %d has a reference to definition %q missing version-id",
				ErrInvalidUDFMetadata, e.TimestampMS, ref.DefinitionID)
		}
		if ref.VersionID < 0 {
			return fmt.Errorf("%w: definition-log entry at timestamp %d has invalid negative version-id %d for definition %q",
				ErrInvalidUDFMetadata, e.TimestampMS, ref.VersionID, ref.DefinitionID)
		}
		if seen[ref.DefinitionID] {
			return fmt.Errorf("%w: definition-log entry at timestamp %d references definition %q more than once",
				ErrInvalidUDFMetadata, e.TimestampMS, ref.DefinitionID)
		}
		seen[ref.DefinitionID] = true
	}

	return nil
}

func (e DefinitionLogEntry) equals(other DefinitionLogEntry) bool {
	return e.TimestampMS == other.TimestampMS &&
		slices.Equal(e.DefinitionVersions, other.DefinitionVersions)
}

// Metadata is the self-contained metadata of an Iceberg SQL UDF as
// specified by the UDF spec. Metadata files are immutable: any change
// creates a new file, and catalogs atomically swap the file linked to a
// catalog identifier. UDF names are not part of the metadata; mapping
// names to metadata locations is the catalog's responsibility.
type Metadata interface {
	// FormatVersion returns the UDF spec format version (1).
	FormatVersion() int
	// FunctionUUID returns the UUID identifying this UDF, generated once
	// at creation.
	FunctionUUID() uuid.UUID
	// Location returns the function's base location, used to create
	// metadata file locations. It may be empty.
	Location() string
	// Definitions returns deep copies of the function's definitions, one per signature.
	Definitions() []*Definition
	// DefinitionByID returns a deep copy of the definition with the given definition-id.
	DefinitionByID(definitionID string) (*Definition, bool)
	// DefinitionLog returns a deep copy of the definition version selection history.
	DefinitionLog() []DefinitionLogEntry
	// Properties returns a copy of the function's properties. Entries are
	// treated as hints, not strict rules.
	Properties() iceberg.Properties
	// Secure reports whether this is a secure function whose sensitive
	// information engines should not leak to end users.
	Secure() bool
	// Doc returns the function's documentation string.
	Doc() string

	Equals(Metadata) bool
}

// ParseMetadata parses UDF metadata JSON provided by the passed in
// reader, returning an error if one is encountered.
func ParseMetadata(r io.Reader) (Metadata, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return ParseMetadataBytes(data)
}

// ParseMetadataString is like [ParseMetadata], but for a string rather
// than an io.Reader.
func ParseMetadataString(s string) (Metadata, error) {
	return ParseMetadataBytes([]byte(s))
}

// ParseMetadataBytes is like [ParseMetadataString] but for a byte slice.
func ParseMetadataBytes(b []byte) (Metadata, error) {
	var ret Metadata = &metadata{}

	return ret, json.Unmarshal(b, ret)
}

type metadata struct {
	UUID               *uuid.UUID           `json:"function-uuid"`
	FormatVersionValue int                  `json:"format-version"`
	DefinitionList     []*Definition        `json:"definitions"`
	DefinitionLogList  []DefinitionLogEntry `json:"definition-log"`
	Loc                string               `json:"location,omitempty"`
	Props              iceberg.Properties   `json:"properties,omitempty"`
	SecureValue        bool                 `json:"secure,omitempty"`
	DocValue           string               `json:"doc,omitempty"`

	// cached lookup helpers, must be initialized in init()
	lazyDefinitionsByID func() map[string]*Definition
}

func (m *metadata) FormatVersion() int         { return m.FormatVersionValue }
func (m *metadata) FunctionUUID() uuid.UUID    { return *m.UUID }
func (m *metadata) Location() string           { return m.Loc }
func (m *metadata) Definitions() []*Definition { return cloneSlice(m.DefinitionList) }
func (m *metadata) DefinitionLog() []DefinitionLogEntry {
	return cloneDefinitionLog(m.DefinitionLogList)
}
func (m *metadata) Properties() iceberg.Properties { return maps.Clone(m.Props) }
func (m *metadata) Secure() bool                   { return m.SecureValue }
func (m *metadata) Doc() string                    { return m.DocValue }

func (m *metadata) DefinitionByID(definitionID string) (*Definition, bool) {
	def, ok := m.lazyDefinitionsByID()[definitionID]
	if !ok {
		return nil, false
	}

	return def.Clone(), true
}

func (m *metadata) Equals(other Metadata) bool {
	if other == nil {
		return false
	}
	if o, ok := other.(*metadata); ok && o == nil {
		return false
	}

	if m == other {
		return true
	}

	return *m.UUID == other.FunctionUUID() &&
		m.FormatVersionValue == other.FormatVersion() &&
		m.Loc == other.Location() &&
		m.SecureValue == other.Secure() &&
		m.DocValue == other.Doc() &&
		maps.Equal(m.Props, other.Properties()) &&
		slices.EqualFunc(m.DefinitionList, other.Definitions(), (*Definition).Equals) &&
		slices.EqualFunc(m.DefinitionLogList, other.DefinitionLog(), DefinitionLogEntry.equals)
}

func (m *metadata) validate() error {
	if m.UUID == nil {
		return fmt.Errorf("%w: function-uuid is required", ErrInvalidUDFMetadata)
	}

	if m.FormatVersionValue == -1 {
		return fmt.Errorf("%w: format-version is required", ErrInvalidUDFMetadataFormatVersion)
	}

	if m.FormatVersionValue != SupportedUDFFormatVersion {
		return fmt.Errorf("%w: format-version %d (only version %d is supported)",
			ErrInvalidUDFMetadataFormatVersion, m.FormatVersionValue, SupportedUDFFormatVersion)
	}

	if len(m.DefinitionList) == 0 {
		return fmt.Errorf("%w: at least one definition is required", ErrInvalidUDFMetadata)
	}

	seenIDs := make(map[string]bool)
	seenSpecificNames := make(map[string]bool)
	for _, def := range m.DefinitionList {
		if def == nil {
			return fmt.Errorf("%w: definitions must not contain null entries", ErrInvalidUDFMetadata)
		}
		if err := def.validate(); err != nil {
			return err
		}
		if seenIDs[def.DefinitionID] {
			return fmt.Errorf("%w: duplicate definition-id %q; there can be only one definition for a given signature",
				ErrInvalidUDFMetadata, def.DefinitionID)
		}
		seenIDs[def.DefinitionID] = true
		if def.SpecificName != "" {
			if seenSpecificNames[def.SpecificName] {
				return fmt.Errorf("%w: duplicate specific-name %q; specific names must be unique among all definitions",
					ErrInvalidUDFMetadata, def.SpecificName)
			}
			seenSpecificNames[def.SpecificName] = true
		}
	}

	for _, entry := range m.DefinitionLogList {
		if err := entry.validate(); err != nil {
			return err
		}
	}

	return nil
}

// init performs state initialization for metadata instances, such as
// constructing the definition index. It should be called on a new
// metadata instance before returning to a caller.
func (m *metadata) init() {
	if m.DefinitionList == nil {
		m.DefinitionList = []*Definition{}
	}

	if m.DefinitionLogList == nil {
		m.DefinitionLogList = []DefinitionLogEntry{}
	}

	if m.Props == nil {
		m.Props = iceberg.Properties{}
	}

	m.lazyDefinitionsByID = sync.OnceValue(func() map[string]*Definition {
		return indexBy(m.DefinitionList, func(d *Definition) string { return d.DefinitionID })
	})
}

func (m *metadata) UnmarshalJSON(b []byte) error {
	type Alias metadata
	next := metadata{FormatVersionValue: -1}
	// definition-log is shadowed to reject the required field being absent
	// or null, which the default decoding cannot distinguish from empty.
	aux := struct {
		DefinitionLog json.RawMessage `json:"definition-log"`
		*Alias
	}{Alias: (*Alias)(&next)}

	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	if len(aux.DefinitionLog) == 0 {
		return fmt.Errorf("%w: definition-log is required", ErrInvalidUDFMetadata)
	}
	if err := json.Unmarshal(aux.DefinitionLog, &next.DefinitionLogList); err != nil {
		return err
	}
	if next.DefinitionLogList == nil {
		return fmt.Errorf("%w: definition-log is required", ErrInvalidUDFMetadata)
	}
	if err := next.validate(); err != nil {
		return err
	}

	*m = next
	m.init()

	return nil
}

// indexBy indexes a slice into a map, using a provided extractKey function.
func indexBy[T any, K comparable](s []T, extractKey func(T) K) map[K]T {
	index := make(map[K]T, len(s))
	for _, v := range s {
		index[extractKey(v)] = v
	}

	return index
}

// cloner is an interface which implements a Clone method for deep copying itself.
type cloner[T any] interface {
	Clone() T
}

// cloneSlice returns a deep-clone of a slice of elements implementing cloner.
func cloneSlice[T cloner[T]](val []T) []T {
	cloned := make([]T, len(val))
	for i, elem := range val {
		cloned[i] = elem.Clone()
	}

	return cloned
}
