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
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/apache/iceberg-go"

	"github.com/google/uuid"
)

// DefinitionOpt configures optional fields of a new Definition.
type DefinitionOpt func(*Definition)

// WithSpecificName sets the definition's optional specific name, a stable
// user-assignable handle that must be unique among all definitions.
func WithSpecificName(name string) DefinitionOpt {
	return func(d *Definition) {
		d.SpecificName = name
	}
}

// WithDefinitionDoc sets the definition's documentation string.
func WithDefinitionDoc(doc string) DefinitionOpt {
	return func(d *Definition) {
		d.Doc = doc
	}
}

// WithReturnNullable sets the definition's return-nullable hint.
func WithReturnNullable(nullable bool) DefinitionOpt {
	return func(d *Definition) {
		d.ReturnNullable = &nullable
	}
}

// NewDefinition creates a definition for the given signature, deriving its
// definition-id from the parameter types. The definition starts with no
// versions; add them with MetadataBuilder.AddDefinitionVersion.
func NewDefinition(functionType FunctionType, params []Parameter, returnType Type, opts ...DefinitionOpt) (*Definition, error) {
	definitionID, err := CanonicalDefinitionID(params)
	if err != nil {
		return nil, err
	}

	// Keep parameters non-nil so a zero-parameter definition serializes as
	// the required empty list rather than null.
	parameters := slices.Clone(params)
	for i := range parameters {
		parameters[i].Type = cloneType(parameters[i].Type)
	}
	if parameters == nil {
		parameters = []Parameter{}
	}

	def := &Definition{
		DefinitionID:     definitionID,
		Parameters:       parameters,
		ReturnType:       cloneType(returnType),
		CurrentVersionID: -1,
		FunctionType:     functionType,
	}

	for _, opt := range opts {
		opt(def)
	}

	return def, nil
}

// DefinitionVersionOpt configures optional fields of a new DefinitionVersion.
type DefinitionVersionOpt func(*DefinitionVersion)

// WithTimestampMS overrides the version's creation timestamp.
func WithTimestampMS(timestampMS int64) DefinitionVersionOpt {
	return func(v *DefinitionVersion) {
		v.TimestampMS = timestampMS
	}
}

// WithDeterministic marks whether the function version is deterministic.
func WithDeterministic(deterministic bool) DefinitionVersionOpt {
	return func(v *DefinitionVersion) {
		v.Deterministic = deterministic
	}
}

// WithOnNullInput sets the version's behavior for NULL input parameters.
func WithOnNullInput(behavior OnNullInput) DefinitionVersionOpt {
	return func(v *DefinitionVersion) {
		v.OnNullInput = behavior
	}
}

// NewDefinitionVersion creates a definition version from the given
// representations. The version id is assigned when the version is added to
// a definition via MetadataBuilder.AddDefinitionVersion.
//
// NewDefinitionVersion seeds TimestampMS to time.Now().UnixMilli(); use
// WithTimestampMS to override.
func NewDefinitionVersion(representations []Representation, opts ...DefinitionVersionOpt) (*DefinitionVersion, error) {
	if len(representations) == 0 {
		return nil, fmt.Errorf("%w: a definition version must have at least one representation", ErrInvalidUDFMetadata)
	}
	for _, repr := range representations {
		if repr == nil {
			return nil, fmt.Errorf("%w: representations must not contain nil entries", ErrInvalidUDFMetadata)
		}
	}

	version := &DefinitionVersion{
		Representations: cloneRepresentations(representations),
		TimestampMS:     time.Now().UnixMilli(),
	}

	for _, opt := range opts {
		opt(version)
	}

	return version, nil
}

// MetadataBuilder builds UDF metadata, either from scratch or on top of an
// existing metadata instance. Because metadata files are immutable, every
// change produces a complete new metadata object; when any definition's
// selected version changes, Build appends a definition-log entry capturing
// the selected version of every definition.
type MetadataBuilder struct {
	base Metadata

	uuid          uuid.UUID
	loc           string
	props         iceberg.Properties
	secure        bool
	doc           string
	definitions   []*Definition
	definitionLog []DefinitionLogEntry

	definitionsByID map[string]*Definition

	logTimestampMS   int64
	selectionChanged bool
	hasChanges       bool

	// error tracking for build chaining
	// if set, subsequent operations become a noop
	err error
}

// NewMetadataBuilder creates a builder for a new UDF's metadata.
func NewMetadataBuilder() (*MetadataBuilder, error) {
	return &MetadataBuilder{
		props:           iceberg.Properties{},
		definitions:     make([]*Definition, 0),
		definitionLog:   make([]DefinitionLogEntry, 0),
		definitionsByID: make(map[string]*Definition),
		logTimestampMS:  -1,
	}, nil
}

// MetadataBuilderFromBase creates a builder seeded with the state of an
// existing metadata instance.
func MetadataBuilderFromBase(base Metadata) (*MetadataBuilder, error) {
	if base == nil {
		return nil, fmt.Errorf("%w: cannot create metadata builder from nil base", ErrInvalidUDFMetadata)
	}

	b := &MetadataBuilder{
		base:           base,
		uuid:           base.FunctionUUID(),
		loc:            base.Location(),
		props:          maps.Clone(base.Properties()),
		secure:         base.Secure(),
		doc:            base.Doc(),
		definitions:    cloneSlice(base.Definitions()),
		definitionLog:  cloneDefinitionLog(base.DefinitionLog()),
		logTimestampMS: -1,
	}
	if b.props == nil {
		b.props = iceberg.Properties{}
	}
	b.definitionsByID = indexBy(b.definitions, func(d *Definition) string { return d.DefinitionID })

	return b, nil
}

// HasChanges reports whether any builder operation changed the state
// relative to the base metadata.
func (b *MetadataBuilder) HasChanges() bool { return b.hasChanges }

// Err returns the first error encountered by builder operations, if any.
func (b *MetadataBuilder) Err() error { return b.err }

// SetUUID assigns the function's UUID. The UUID is generated once at
// creation and cannot be reassigned.
func (b *MetadataBuilder) SetUUID(newUUID uuid.UUID) *MetadataBuilder {
	if b.err != nil {
		return b
	}

	if newUUID == uuid.Nil {
		b.err = errors.New("cannot set uuid to null")

		return b
	}

	if b.uuid == newUUID {
		return b
	}

	if b.uuid != uuid.Nil {
		b.err = errors.New("cannot reassign uuid")

		return b
	}

	b.uuid = newUUID
	b.hasChanges = true

	return b
}

// SetLoc sets the function's base location, used to create metadata
// file locations.
func (b *MetadataBuilder) SetLoc(loc string) *MetadataBuilder {
	if b.err != nil || b.loc == loc {
		return b
	}

	b.loc = loc
	b.hasChanges = true

	return b
}

// SetProperties adds or updates the given properties.
func (b *MetadataBuilder) SetProperties(props iceberg.Properties) *MetadataBuilder {
	if b.err != nil || len(props) == 0 {
		return b
	}

	maps.Copy(b.props, props)
	b.hasChanges = true

	return b
}

// RemoveProperties removes the given property keys.
func (b *MetadataBuilder) RemoveProperties(keys []string) *MetadataBuilder {
	if b.err != nil || len(keys) == 0 {
		return b
	}

	for _, key := range keys {
		delete(b.props, key)
	}
	b.hasChanges = true

	return b
}

// SetSecure marks whether this is a secure function.
func (b *MetadataBuilder) SetSecure(secure bool) *MetadataBuilder {
	if b.err != nil || b.secure == secure {
		return b
	}

	b.secure = secure
	b.hasChanges = true

	return b
}

// SetDoc sets the function's documentation string.
func (b *MetadataBuilder) SetDoc(doc string) *MetadataBuilder {
	if b.err != nil || b.doc == doc {
		return b
	}

	b.doc = doc
	b.hasChanges = true

	return b
}

// SetLogTimestampMS overrides the timestamp used for the definition-log
// entry appended by Build. Writers that need deterministic metadata (or a
// timestamp consistent with an external commit) should set this; when
// unset it defaults to time.Now().UnixMilli().
func (b *MetadataBuilder) SetLogTimestampMS(timestampMS int64) *MetadataBuilder {
	if b.err != nil {
		return b
	}

	b.logTimestampMS = timestampMS

	return b
}

// AddDefinition adds a definition for a new signature. The definition's
// definition-id is derived from its parameters; if the definition already
// carries an id it must match the canonical form. Adding a definition
// whose signature or specific name already exists fails: there can be
// only one definition for a given signature.
func (b *MetadataBuilder) AddDefinition(def *Definition) *MetadataBuilder {
	if b.err != nil {
		return b
	}

	if def == nil {
		b.err = fmt.Errorf("%w: cannot add nil definition", ErrInvalidUDFMetadata)

		return b
	}

	canonicalID, err := CanonicalDefinitionID(def.Parameters)
	if b.setErr(err) {
		return b
	}

	if def.DefinitionID != "" && def.DefinitionID != canonicalID {
		b.err = fmt.Errorf("%w: definition-id %q does not match canonical form %q of its parameters",
			ErrInvalidUDFMetadata, def.DefinitionID, canonicalID)

		return b
	}

	if _, ok := b.definitionsByID[canonicalID]; ok {
		b.err = fmt.Errorf("%w: a definition for signature %q already exists", ErrInvalidUDFMetadata, canonicalID)

		return b
	}

	if def.SpecificName != "" {
		for _, existing := range b.definitions {
			if existing.SpecificName == def.SpecificName {
				b.err = fmt.Errorf("%w: a definition with specific-name %q already exists",
					ErrInvalidUDFMetadata, def.SpecificName)

				return b
			}
		}
	}

	added := def.Clone()
	added.DefinitionID = canonicalID
	// Keep parameters non-nil so a zero-parameter definition serializes as
	// the required empty list rather than null.
	if added.Parameters == nil {
		added.Parameters = []Parameter{}
	}
	b.definitions = append(b.definitions, added)
	b.definitionsByID[canonicalID] = added
	b.hasChanges = true
	b.selectionChanged = true

	return b
}

// RemoveDefinition removes the definition with the given definition-id.
func (b *MetadataBuilder) RemoveDefinition(definitionID string) *MetadataBuilder {
	if b.err != nil {
		return b
	}

	if _, ok := b.definitionsByID[definitionID]; !ok {
		b.err = fmt.Errorf("%w: no definition with definition-id %q", ErrInvalidUDFMetadata, definitionID)

		return b
	}

	delete(b.definitionsByID, definitionID)
	b.definitions = slices.DeleteFunc(b.definitions, func(d *Definition) bool {
		return d.DefinitionID == definitionID
	})
	b.hasChanges = true
	b.selectionChanged = true

	return b
}

// AddDefinitionVersion adds a version to the definition with the given
// definition-id and selects it as the definition's current version. A
// version id of 0 is assigned the next id for the definition; an explicit
// version id must not already exist. Use SetCurrentVersion to roll back to
// an earlier version.
func (b *MetadataBuilder) AddDefinitionVersion(definitionID string, version *DefinitionVersion) *MetadataBuilder {
	if b.err != nil {
		return b
	}

	if version == nil {
		b.err = fmt.Errorf("%w: cannot add nil definition version", ErrInvalidUDFMetadata)

		return b
	}

	def, ok := b.definitionsByID[definitionID]
	if !ok {
		b.err = fmt.Errorf("%w: no definition with definition-id %q", ErrInvalidUDFMetadata, definitionID)

		return b
	}

	added := version.Clone()
	if added.VersionID < 0 {
		b.err = fmt.Errorf("%w: invalid negative version-id %d", ErrInvalidUDFMetadata, added.VersionID)

		return b
	}
	if added.VersionID == 0 {
		added.VersionID = nextVersionID(def)
	} else if def.Version(added.VersionID) != nil {
		b.err = fmt.Errorf("%w: definition %q already has version-id %d",
			ErrInvalidUDFMetadata, definitionID, added.VersionID)

		return b
	}

	def.Versions = append(def.Versions, added)
	def.CurrentVersionID = added.VersionID
	b.hasChanges = true
	b.selectionChanged = true

	return b
}

// SetCurrentVersion sets the definition's current version to an existing
// version id, e.g. to roll back to an earlier implementation.
func (b *MetadataBuilder) SetCurrentVersion(definitionID string, versionID int) *MetadataBuilder {
	if b.err != nil {
		return b
	}

	def, ok := b.definitionsByID[definitionID]
	if !ok {
		b.err = fmt.Errorf("%w: no definition with definition-id %q", ErrInvalidUDFMetadata, definitionID)

		return b
	}

	if def.CurrentVersionID == versionID {
		return b
	}

	if def.Version(versionID) == nil {
		b.err = fmt.Errorf("%w: definition %q has no version-id %d", ErrInvalidUDFMetadata, definitionID, versionID)

		return b
	}

	def.CurrentVersionID = versionID
	b.hasChanges = true
	b.selectionChanged = true

	return b
}

// Build validates and returns the metadata. When any definition's selected
// version changed, a definition-log entry capturing the current selection
// of every definition is appended, with entries ordered by definition-id.
func (b *MetadataBuilder) Build() (Metadata, error) {
	if b.err != nil {
		return nil, b.err
	}

	uuid_ := b.uuid
	if uuid_ == uuid.Nil {
		uuid_ = uuid.New()
	}

	definitionLog := cloneDefinitionLog(b.definitionLog)
	if b.selectionChanged {
		timestampMS := b.logTimestampMS
		if timestampMS == -1 {
			timestampMS = time.Now().UnixMilli()
		}

		refs := make([]DefinitionVersionRef, len(b.definitions))
		for i, def := range b.definitions {
			refs[i] = DefinitionVersionRef{DefinitionID: def.DefinitionID, VersionID: def.CurrentVersionID}
		}
		slices.SortFunc(refs, func(a, c DefinitionVersionRef) int {
			return strings.Compare(a.DefinitionID, c.DefinitionID)
		})

		definitionLog = append(definitionLog, DefinitionLogEntry{
			TimestampMS:        timestampMS,
			DefinitionVersions: refs,
		})
	}

	// Clone the mutable state so that further builder operations cannot
	// alter previously built metadata.
	md := &metadata{
		UUID:               &uuid_,
		FormatVersionValue: SupportedUDFFormatVersion,
		DefinitionList:     cloneSlice(b.definitions),
		DefinitionLogList:  definitionLog,
		Loc:                b.loc,
		Props:              maps.Clone(b.props),
		SecureValue:        b.secure,
		DocValue:           b.doc,
	}
	md.init()

	if err := md.validate(); err != nil {
		return nil, err
	}

	return md, nil
}

// setErr is a helper for enforcing build error assignment during
// operations. It returns true if the error was non-nil and the builder has
// entered an error state.
func (b *MetadataBuilder) setErr(err error) (buildHasFailed bool) {
	if err != nil {
		b.err = err

		return true
	}

	return false
}

func nextVersionID(def *Definition) int {
	next := initialVersionID
	for _, v := range def.Versions {
		if v.VersionID >= next {
			next = v.VersionID + 1
		}
	}

	return next
}

func cloneDefinitionLog(log []DefinitionLogEntry) []DefinitionLogEntry {
	cloned := slices.Clone(log)
	for i := range cloned {
		cloned[i].DefinitionVersions = slices.Clone(log[i].DefinitionVersions)
	}

	return cloned
}
