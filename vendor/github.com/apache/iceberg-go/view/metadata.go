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

package view

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/apache/iceberg-go"
	iceinternal "github.com/apache/iceberg-go/internal"
	"github.com/apache/iceberg-go/table"
	"github.com/apache/iceberg-go/view/internal"

	"github.com/google/uuid"
)

var (
	ErrInvalidViewMetadata              = errors.New("invalid view metadata")
	ErrInvalidViewMetadataFormatVersion = errors.New("invalid or missing format-version in view metadata")
)

const (
	// LastAddedID is used in place of ID fields (e.g. schema, version) to indicate that
	// the last added instance of that type should be used.
	LastAddedID      = -1
	InitialVersionID = 1

	SupportedViewFormatVersion = 1
	DefaultViewFormatVersion   = SupportedViewFormatVersion
)

const (
	initialSchemaID = 0

	viewEngineProperty = "engine-name"
	defaultViewEngine  = "iceberg-go"
)

// Metadata for an iceberg view as specified in the Iceberg spec
// https://iceberg.apache.org/view-spec/
type Metadata interface {
	// FormatVersion indicates the version of this metadata, 1 for V1
	FormatVersion() int
	// ViewUUID returns a UUID that identifies the view, generated when the
	// view is created. Implementations must throw an exception if a view's
	// UUID does not match the expected UUID after refreshing metadata.
	ViewUUID() uuid.UUID
	// Location is the table's base location. This is used by writers to determine
	// where to store data files, manifest files, and table metadata files.
	Location() string
	// Schemas returns the list of view schemas
	Schemas() []*iceberg.Schema
	// CurrentVersionID returns the ID of the current version of the view (version-id)
	CurrentVersionID() int64
	// CurrentVersion returns the current version of the view
	CurrentVersion() *Version
	// CurrentSchemaID returns the ID of the current schema
	CurrentSchemaID() int
	// CurrentSchema returns the current schema of the view
	CurrentSchema() *iceberg.Schema
	// SchemasByID returns a map of schema IDs to schemas
	SchemasByID() map[int]*iceberg.Schema
	// Versions returns the list of view versions
	Versions() []*Version
	// VersionLog returns a list of version log entries
	// with the timestamp and version-id for every change to current-version-id
	VersionLog() []VersionLogEntry
	// Properties is a string to string map of view properties.
	Properties() iceberg.Properties

	Equals(Metadata) bool
}

// VersionSummary is string to string map of summary metadata about a view's version
type VersionSummary map[string]string

// Representation is a struct containing information about a view's representation
// https://iceberg.apache.org/view-spec/#sql-representation
type Representation struct {
	// Must be sql
	Type string `json:"type"`
	// A SQL SELECT statement
	Sql string `json:"sql"`
	// The dialect of the sql SELECT statement (e.g., "trino" or "spark")
	Dialect string `json:"dialect"`
}
type Representations []Representation

func NewRepresentation(sql string, dialect string) Representation {
	return Representation{
		Type:    "sql",
		Sql:     sql,
		Dialect: dialect,
	}
}

type Version struct {
	// ID for the version
	VersionID int64 `json:"version-id"`
	// ID of the schema for the view version
	SchemaID int `json:"schema-id"`
	// Timestamp when the version was created (ms from epoch)
	TimestampMS int64 `json:"timestamp-ms"`
	// A string to string map of summary metadata about the version
	Summary VersionSummary `json:"summary"`
	// A list of representations for the view definition
	Representations []Representation `json:"representations"`
	// The default view namespace (as a list of strings)
	DefaultNamespace table.Identifier `json:"default-namespace"`
	// An (optional) default catalog name for querying the view
	DefaultCatalog string `json:"default-catalog,omitempty"`
}

type VersionOpt func(*Version)

func WithVersionSummary(summary VersionSummary) VersionOpt {
	return func(v *Version) {
		if v.Summary == nil {
			v.Summary = summary
		} else {
			maps.Copy(v.Summary, summary)
		}
	}
}

func WithDefaultViewCatalog(catalogName string) func(*Version) {
	return func(version *Version) {
		version.DefaultCatalog = catalogName
	}
}

func WithTimestampMS(timestampMS int64) VersionOpt {
	return func(version *Version) {
		version.TimestampMS = timestampMS
	}
}

// NewVersion creates a Version instance from the provided parameters.
// If building updates using the MetadataBuilder, and one desires to use the last
// added schema ID, one should use the LastAddedID constant as the provided schemaID
//
// Note that NewVersion automatically seeds TimestampMS to time.Now().UnixMilli(),
// and one should use the option WithTimestampMS to override this behavior.
func NewVersion(id int64, schemaID int, representations []Representation, defaultNS table.Identifier, opts ...VersionOpt) (*Version, error) {
	if id < 1 {
		return nil, errors.New("id should be greater than 0")
	}

	if len(representations) == 0 {
		return nil, errors.New("invalid view version: must have at least one representation")
	}

	seenDialects := make(map[string]struct{})
	for _, repr := range representations {
		normalizedDialect, err := validateRepresentation(repr)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidViewMetadata, err)
		}
		if _, ok := seenDialects[normalizedDialect]; ok {
			return nil, fmt.Errorf("%w: Invalid view version: Cannot add multiple queries for dialect %s", ErrInvalidViewMetadata, repr.Dialect)
		}
		seenDialects[normalizedDialect] = struct{}{}
	}

	version := &Version{
		VersionID:        id,
		SchemaID:         schemaID,
		Representations:  representations,
		Summary:          VersionSummary{viewEngineProperty: defaultViewEngine},
		TimestampMS:      time.Now().UnixMilli(),
		DefaultNamespace: defaultNS,
	}

	for _, opt := range opts {
		opt(version)
	}

	return version, nil
}

// validateRepresentation validates a SQL representation and returns its
// normalized dialect for duplicate detection.
func validateRepresentation(repr Representation) (string, error) {
	if repr.Type != "sql" {
		return "", errors.New("invalid view representation: type should be \"sql\"")
	}

	if strings.TrimSpace(repr.Sql) == "" {
		return "", errors.New("invalid view representation: sql is required")
	}

	dialect := strings.TrimSpace(repr.Dialect)
	if dialect == "" {
		return "", errors.New("invalid view representation: dialect is required")
	}

	return strings.ToLower(dialect), nil
}

// NewVersionFromSQL creates a new Version with a single representation
// using the provided SQL and dialect "default".
func NewVersionFromSQL(id int64, schemaID int, sql string, defaultNS table.Identifier, opts ...VersionOpt) (*Version, error) {
	return NewVersion(id, schemaID, Representations{NewRepresentation(sql, "default")}, defaultNS, opts...)
}

// Equals checks whether the other Version would behave the same
// while ignoring the view version id, and the creation timestamp
func (v *Version) Equals(other *Version) bool {
	return v.SchemaID == other.SchemaID &&
		v.DefaultCatalog == other.DefaultCatalog &&
		slices.Equal(v.DefaultNamespace, other.DefaultNamespace) &&
		maps.Equal(v.Summary, other.Summary) &&
		slices.Equal(v.Representations, other.Representations)
}

func (v *Version) Clone() *Version {
	if v == nil {
		return nil
	}

	cloned := *v
	cloned.Summary = maps.Clone(v.Summary)
	cloned.Representations = slices.Clone(v.Representations)
	cloned.DefaultNamespace = slices.Clone(v.DefaultNamespace)

	return &cloned
}

// sqlDialects returns a set of strings representing the SQL dialects supported in this version.
// Dialects are deduplicated by trimmed, lowercase comparison.
func (v *Version) sqlDialects() internal.Set[string] {
	dialects := make([]string, 0, len(v.Representations))
	for _, repr := range v.Representations {
		if repr.Type == "sql" {
			dialects = append(dialects, strings.ToLower(strings.TrimSpace(repr.Dialect)))
		}
	}

	return internal.ToSet(dialects)
}

type VersionLogEntry struct {
	// Timestamp when the view's current-version-id was updated (ms from epoch)
	TimestampMS int64 `json:"timestamp-ms"`
	// ID that current-version-id was set to
	VersionID int64 `json:"version-id"`
}

// ParseMetadata parses json metadata provided by the passed in reader,
// returning an error if one is encountered.
func ParseMetadata(r io.Reader) (Metadata, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return ParseMetadataBytes(data)
}

// ParseMetadataString is like [ParseMetadata], but for a string rather than
// an io.Reader.
func ParseMetadataString(s string) (Metadata, error) {
	return ParseMetadataBytes([]byte(s))
}

// ParseMetadataBytes is like [ParseMetadataString] but for a byte slice.
func ParseMetadataBytes(b []byte) (Metadata, error) {
	ver := struct {
		FormatVersion int `json:"format-version"`
	}{}
	if err := json.Unmarshal(b, &ver); err != nil {
		return nil, err
	}

	var ret Metadata = &metadata{}

	return ret, json.Unmarshal(b, ret)
}

// indexBy indexes a slice into a map, using a provided extractKey function
// The extractKey function will be called on each item in the slice and assign the
// item as a value for that key in the resultant map.
func indexBy[T any, K comparable](s []T, extractKey func(T) K) map[K]T {
	index := make(map[K]T)
	for _, v := range s {
		index[extractKey(v)] = v
	}

	return index
}

// Cloner is an interface which implements a Clone method for deep copying itself
type cloner[T any] interface {
	// Clone returns a deep copy of the underlying object
	Clone() T
}

// cloneSlice returns a deep-clone of a Slice of elements implementing Cloner
func cloneSlice[T cloner[T]](val []T) []T {
	cloned := make([]T, len(val))
	for i, elem := range val {
		cloned[i] = elem.Clone()
	}

	return cloned
}

// https://iceberg.apache.org/view-spec/
type metadata struct {
	FormatVersionValue    int                `json:"format-version"`
	UUID                  *uuid.UUID         `json:"view-uuid"`
	Loc                   string             `json:"location"`
	CurrentVersionIDValue int64              `json:"current-version-id"`
	VersionList           []*Version         `json:"versions"`
	SchemaList            []*iceberg.Schema  `json:"schemas"`
	VersionLogList        []VersionLogEntry  `json:"version-log"`
	Props                 iceberg.Properties `json:"properties,omitempty"`

	// cached lookup helpers, must be initialized in init()
	lazyVersionsByID func() map[int64]*Version
	lazySchemasByID  func() map[int]*iceberg.Schema
}

func (m *metadata) Equals(other Metadata) bool {
	if other == nil {
		return false
	}

	if m == other {
		return true
	}

	return *m.UUID == other.ViewUUID() &&
		m.Loc == other.Location() &&
		m.FormatVersionValue == other.FormatVersion() &&
		iceinternal.SliceEqualHelper(m.Schemas(), other.Schemas()) &&
		iceinternal.SliceEqualHelper(m.Versions(), other.Versions()) &&
		m.CurrentVersionIDValue == other.CurrentVersionID() &&
		slices.Equal(m.VersionLogList, other.VersionLog())
}

func (m *metadata) FormatVersion() int         { return m.FormatVersionValue }
func (m *metadata) ViewUUID() uuid.UUID        { return *m.UUID }
func (m *metadata) Location() string           { return m.Loc }
func (m *metadata) Versions() []*Version       { return cloneSlice(m.VersionList) }
func (m *metadata) Schemas() []*iceberg.Schema { return cloneSchemas(m.SchemaList) }
func (m *metadata) SchemasByID() map[int]*iceberg.Schema {
	clones := make(map[int]*iceberg.Schema, len(m.SchemaList))
	for id, schema := range m.lazySchemasByID() {
		clones[id] = cloneSchema(schema)
	}

	return clones
}

func (m *metadata) CurrentVersionID() int64 {
	return m.CurrentVersionIDValue
}

func (m *metadata) CurrentVersion() *Version {
	return m.currentVersion().Clone()
}

func (m *metadata) currentVersion() *Version {
	version, ok := m.lazyVersionsByID()[m.CurrentVersionIDValue]
	if !ok {
		panic("current version not found")
	}

	return version
}

func (m *metadata) CurrentSchemaID() int {
	return m.currentVersion().SchemaID
}

func (m *metadata) CurrentSchema() *iceberg.Schema {
	schema, ok := m.lazySchemasByID()[m.CurrentSchemaID()]
	if !ok {
		panic("current schema not found")
	}

	return cloneSchema(schema)
}

func (m *metadata) VersionLog() []VersionLogEntry {
	return slices.Clone(m.VersionLogList)
}

func (m *metadata) Properties() iceberg.Properties {
	return maps.Clone(m.Props)
}

func cloneSchema(schema *iceberg.Schema) *iceberg.Schema {
	if schema == nil {
		return nil
	}

	return iceberg.NewSchemaWithIdentifiers(
		schema.ID,
		slices.Clone(schema.IdentifierFieldIDs),
		schema.Fields()...,
	)
}

func cloneSchemas(schemas []*iceberg.Schema) []*iceberg.Schema {
	if schemas == nil {
		return nil
	}

	clones := make([]*iceberg.Schema, len(schemas))
	for i, schema := range schemas {
		clones[i] = cloneSchema(schema)
	}

	return clones
}

func (m *metadata) validate() error {
	if m.Loc == "" {
		return fmt.Errorf("%w: location is required", ErrInvalidViewMetadata)
	}

	if m.UUID == nil {
		return fmt.Errorf("%w: view-uuid is required", ErrInvalidViewMetadata)
	}

	if m.FormatVersionValue == -1 {
		return fmt.Errorf("%w: format-version is required", ErrInvalidViewMetadataFormatVersion)
	}

	if m.FormatVersionValue < SupportedViewFormatVersion || m.FormatVersionValue > 1 {
		return fmt.Errorf("%w: format-version %d (only version %d is supported)",
			ErrInvalidViewMetadataFormatVersion, m.FormatVersionValue, SupportedViewFormatVersion)
	}

	if len(m.VersionList) == 0 {
		return fmt.Errorf("%w: at least one version is required", ErrInvalidViewMetadata)
	}

	if m.CurrentVersionIDValue == -1 {
		return fmt.Errorf("%w: current-version-id is required", ErrInvalidViewMetadata)
	}

	if len(m.SchemaList) == 0 {
		return fmt.Errorf("%w: at least one schema is required", ErrInvalidViewMetadata)
	}

	if err := m.checkSchemaAndVersionEntries(); err != nil {
		return err
	}

	if err := m.checkCurrentVersionExists(); err != nil {
		return err
	}

	if err := m.checkVersionSchemasExist(); err != nil {
		return err
	}

	if err := m.checkDialectsUnique(); err != nil {
		return err
	}

	return nil
}

func (m *metadata) checkSchemaAndVersionEntries() error {
	schemaIDs := make(map[int]struct{}, len(m.SchemaList))
	for i, schema := range m.SchemaList {
		if schema == nil {
			return fmt.Errorf("%w: schema at index %d is null", ErrInvalidViewMetadata, i)
		}
		if _, ok := schemaIDs[schema.ID]; ok {
			return fmt.Errorf("%w: duplicate schema-id %d", ErrInvalidViewMetadata, schema.ID)
		}
		schemaIDs[schema.ID] = struct{}{}
	}

	versionIDs := make(map[int64]struct{}, len(m.VersionList))
	for i, version := range m.VersionList {
		if version == nil {
			return fmt.Errorf("%w: version at index %d is null", ErrInvalidViewMetadata, i)
		}
		if _, ok := versionIDs[version.VersionID]; ok {
			return fmt.Errorf("%w: duplicate version-id %d", ErrInvalidViewMetadata, version.VersionID)
		}
		versionIDs[version.VersionID] = struct{}{}
	}

	return nil
}

func (m *metadata) checkCurrentVersionExists() error {
	for _, v := range m.VersionList {
		if v.VersionID == m.CurrentVersionIDValue {
			return nil
		}
	}

	return fmt.Errorf("%w: current-version-id %d not found in versions",
		ErrInvalidViewMetadata, m.CurrentVersionIDValue)
}

func (m *metadata) checkVersionSchemasExist() error {
	schemasByID := m.lazySchemasByID()

	for _, version := range m.VersionList {
		if _, ok := schemasByID[version.SchemaID]; !ok {
			return fmt.Errorf("%w: version %d references unknown schema-id %d",
				ErrInvalidViewMetadata, version.VersionID, version.SchemaID)
		}
	}

	return nil
}

func (m *metadata) checkDialectsUnique() error {
	for _, version := range m.VersionList {
		seenDialects := make(map[string]bool)
		for _, repr := range version.Representations {
			if repr.Type != "sql" {
				continue
			}

			dialect, err := validateRepresentation(repr)
			if err != nil {
				return fmt.Errorf("%w: version %d: %s", ErrInvalidViewMetadata, version.VersionID, err)
			}

			if seenDialects[dialect] {
				return fmt.Errorf("%w: version %d has duplicate dialect %s",
					ErrInvalidViewMetadata, version.VersionID, repr.Dialect)
			}
			seenDialects[dialect] = true
		}
	}

	return nil
}

// init performs state initialization for metadata instances,
// such as constructing the version and schema indexes.
// It should be called on a new metadata instance before returning
// to a caller.
func (m *metadata) init() {
	if m.SchemaList == nil {
		m.SchemaList = []*iceberg.Schema{}
	}

	if m.VersionList == nil {
		m.VersionList = []*Version{}
	}

	if m.VersionLogList == nil {
		m.VersionLogList = []VersionLogEntry{}
	}

	if m.Props == nil {
		m.Props = iceberg.Properties{}
	}

	m.lazyVersionsByID = sync.OnceValue(func() map[int64]*Version {
		return indexBy(m.VersionList, func(v *Version) int64 { return v.VersionID })
	})
	m.lazySchemasByID = sync.OnceValue(func() map[int]*iceberg.Schema {
		return indexBy(m.SchemaList, func(v *iceberg.Schema) int { return v.ID })
	})
}

func (m *metadata) UnmarshalJSON(b []byte) error {
	type Alias metadata
	next := metadata{
		FormatVersionValue:    -1,
		CurrentVersionIDValue: -1,
	}
	aux := (*Alias)(&next)

	if err := json.Unmarshal(b, aux); err != nil {
		return err
	}

	next.init()

	if err := next.validate(); err != nil {
		return err
	}

	*m = next
	// init() rebinds the lazy-index closures to m after the value copy.
	m.init()

	return nil
}

// NewMetadata creates a new view metadata object using the provided version, schema, location, and props,
// generating a fresh UUID for the new table metadata.
func NewMetadata(version *Version, sc *iceberg.Schema, location string, props iceberg.Properties) (Metadata, error) {
	return NewMetadataWithUUID(version, sc, location, props, uuid.Nil)
}

// NewMetadataWithUUID is like NewMetadata, but allows the caller to specify the UUID of the view rather than creating a new one.
func NewMetadataWithUUID(version *Version, sc *iceberg.Schema, location string, props iceberg.Properties, viewUUID uuid.UUID) (Metadata, error) {
	// Don't call AssignFreshSchemaIDs here as it reassigns field IDs which breaks RCK tests.
	// The MetadataBuilder.SetCurrentVersion method will handle schema ID normalization.

	if viewUUID == uuid.Nil {
		viewUUID = uuid.New()
	}

	var inputProps iceberg.Properties
	formatVersion := DefaultViewFormatVersion
	if props != nil {
		inputProps = maps.Clone(props)
		verStr, ok := inputProps[table.PropertyFormatVersion]
		if ok {
			var err error
			if formatVersion, err = strconv.Atoi(verStr); err != nil {
				return nil, fmt.Errorf("%w: %s", iceberg.ErrInvalidFormatVersion, verStr)
			}
			delete(inputProps, table.PropertyFormatVersion)
		}
	}

	if formatVersion > SupportedViewFormatVersion {
		return nil, fmt.Errorf("%w: %d", iceberg.ErrInvalidFormatVersion, formatVersion)
	}

	// We assume that this constructor is used for building metadata for a new view.
	// Thus, we enforce that the VersionID is the initial one defined in the spec.
	if version.VersionID != InitialVersionID {
		clonedVersion := *version
		version = &clonedVersion
		version.VersionID = InitialVersionID
	}

	builder, err := NewMetadataBuilder()
	if err != nil {
		return nil, err
	}

	return builder.
		SetFormatVersion(formatVersion).
		SetUUID(viewUUID).
		SetLoc(location).
		SetProperties(inputProps).
		SetCurrentVersion(version, sc).
		Build()
}
