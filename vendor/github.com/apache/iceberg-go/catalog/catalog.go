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

// Package catalog provides an interface for Catalog implementations along with
// a registry for registering catalog implementations.
//
// Subpackages of this package provide some default implementations for select
// catalog types which will register themselves if imported. For instance,
// adding the following import:
//
//	import _ "github.com/apache/iceberg-go/catalog/rest"
//
// Will register the REST catalog implementation.
package catalog

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"maps"
	"strings"
	"unicode"

	"github.com/apache/iceberg-go"
	iceinternal "github.com/apache/iceberg-go/internal"
	"github.com/apache/iceberg-go/table"
)

type Type string

const (
	REST     Type = "rest"
	Hive     Type = "hive"
	Glue     Type = "glue"
	DynamoDB Type = "dynamodb"
	SQL      Type = "sql"
	Hadoop   Type = "hadoop"
)

var (
	// ErrNoSuchTable is returned when a table does not exist in the catalog.
	ErrNoSuchTable            = errors.New("table does not exist")
	ErrNoSuchNamespace        = errors.New("namespace does not exist")
	ErrInvalidIdentifier      = errors.New("identifier is invalid")
	ErrNamespaceAlreadyExists = errors.New("namespace already exists")
	ErrTableAlreadyExists     = errors.New("table already exists")
	ErrCatalogNotFound        = errors.New("catalog type not registered")
	ErrNamespaceNotEmpty      = errors.New("namespace is not empty")
	ErrNoSuchView             = errors.New("view does not exist")
	ErrViewAlreadyExists      = errors.New("view already exists")
	ErrNoSuchFunction         = errors.New("function does not exist")
	ErrEmptyCommitList        = errors.New("commit list must not be empty")
	ErrMissingIdentifier      = errors.New("every table commit must have a valid identifier")
)

type PropertiesUpdateSummary struct {
	Removed []string `json:"removed"`
	Updated []string `json:"updated"`
	Missing []string `json:"missing"`
}

// commonCreateCfg represents common configuration to create table and view operations
type commonCreateCfg struct {
	Location   string
	Properties iceberg.Properties
}

// CreateTableCfg represents the configuration used for CreateTable operations
type CreateTableCfg struct {
	commonCreateCfg
	PartitionSpec *iceberg.PartitionSpec
	SortOrder     table.SortOrder
	// StagedUpdates holds additional table.Update operations that cause
	// the REST catalog to use a two-phase staged creation. Phase 1
	// sends a minimal create with stage-create=true; phase 2 commits
	// with an assert-create requirement and all updates atomically.
	// Non-REST catalogs ignore this field.
	StagedUpdates []table.Update
}

func NewCreateTableCfg() CreateTableCfg {
	return CreateTableCfg{
		commonCreateCfg: commonCreateCfg{},
		PartitionSpec:   nil,
		SortOrder:       table.UnsortedSortOrder,
	}
}

// CreateViewCfg represents the configuration used for CreateView operations
type CreateViewCfg struct {
	commonCreateCfg
}

func NewCreateViewCfg() CreateViewCfg {
	return CreateViewCfg{
		commonCreateCfg{
			Properties: iceberg.Properties{},
		},
	}
}

// Catalog for iceberg table operations like create, drop, load, list and others.
type Catalog interface {
	// CatalogType returns the type of the catalog.
	CatalogType() Type

	// CreateTable creates a new iceberg table in the catalog using the provided identifier
	// and schema. Options can be used to optionally provide location, partition spec, sort order,
	// and custom properties.
	CreateTable(ctx context.Context, identifier table.Identifier, schema *iceberg.Schema, opts ...CreateTableOpt) (*table.Table, error)
	// CommitTable commits the table metadata and updates to the catalog, returning the new metadata
	CommitTable(ctx context.Context, identifier table.Identifier, requirements []table.Requirement, updates []table.Update) (table.Metadata, string, error)
	// ListTables returns a list of table identifiers in the catalog, with the returned
	// identifiers containing the information required to load the table via that catalog.
	ListTables(ctx context.Context, namespace table.Identifier) iter.Seq2[table.Identifier, error]
	// LoadTable loads a table from the catalog and returns a Table with the metadata.
	LoadTable(ctx context.Context, identifier table.Identifier) (*table.Table, error)
	// DropTable tells the catalog to drop the table entirely.
	DropTable(ctx context.Context, identifier table.Identifier) error
	// RenameTable tells the catalog to rename a given table by the identifiers
	// provided, and then loads and returns the destination table
	RenameTable(ctx context.Context, from, to table.Identifier) (*table.Table, error)
	// CheckTableExists reports whether the table exists.
	CheckTableExists(ctx context.Context, identifier table.Identifier) (bool, error)
	// ListNamespaces returns the list of available namespaces, optionally filtering by a
	// parent namespace
	ListNamespaces(ctx context.Context, parent table.Identifier) ([]table.Identifier, error)
	// CreateNamespace tells the catalog to create a new namespace with the given properties
	CreateNamespace(ctx context.Context, namespace table.Identifier, props iceberg.Properties) error
	// DropNamespace tells the catalog to drop the namespace and all tables in that namespace
	DropNamespace(ctx context.Context, namespace table.Identifier) error
	// CheckNamespaceExists returns if the namespace exists
	CheckNamespaceExists(ctx context.Context, namespace table.Identifier) (bool, error)
	// LoadNamespaceProperties returns the current properties in the catalog for
	// a given namespace
	LoadNamespaceProperties(ctx context.Context, namespace table.Identifier) (iceberg.Properties, error)
	// UpdateNamespaceProperties allows removing, adding, and/or updating properties of a namespace
	UpdateNamespaceProperties(ctx context.Context, namespace table.Identifier,
		removals []string, updates iceberg.Properties) (PropertiesUpdateSummary, error)
}

// TransactionalCatalog is an optional interface implemented by catalogs
// that support atomic multi-table commits. Callers should check for this
// capability via a type assertion:
//
//	if tc, ok := cat.(catalog.TransactionalCatalog); ok {
//	    err := tc.CommitTransaction(ctx, commits)
//	}
//
// The endpoint is all-or-nothing: either all table changes are applied
// atomically, or none are. On success the method returns nil; the caller
// must LoadTable individually to obtain updated metadata because the
// server returns 204 No Content.
type TransactionalCatalog interface {
	CommitTransaction(ctx context.Context, commits []table.TableCommit) error
}

// PurgeableTable is an optional interface that catalogs can implement
// to support physical table deletion (catalog entry + underlying files).
// Callers should check for this capability via a type assertion:
//
//	if purger, ok := cat.(catalog.PurgeableTable); ok {
//	    err := purger.PurgeTable(ctx, ident)
//	}
//
// For REST catalogs the purge is delegated server-side. For client-side
// catalogs (SQL, Glue, Hive, Hadoop) the table is first dropped from
// the catalog, then all files under the table's [table.Metadata.Location]
// root, plus any referenced files written outside the root (e.g. via
// write.data.path or write.metadata.path table properties), are physically deleted.
//
// File physical deletion is performed on a best-effort basis. If gc.enabled is
// set to false on the table metadata, physical deletion is skipped entirely.
// Any file-deletion, directory walk, or listing errors encountered are logged
// as warnings, and the operation succeeds (returns nil) so that the catalog drop
// remains the single source of truth.
type PurgeableTable interface {
	PurgeTable(ctx context.Context, identifier table.Identifier) error
}

// Closer is an optional interface implemented by catalogs that hold releasable
// resources — currently a metrics reporter and stateful resources (e.g.
// HTTP-backed clients). It is not part of [Catalog] because adding a method to that
// widely-implemented interface would break every external implementation; a
// follow-up may promote it. Callers holding a [Catalog] from [Load] should
// release it via a type assertion:
//
//	cat, err := catalog.Load(ctx, name, props)
//	if err != nil {
//	    return err
//	}
//	if closer, ok := cat.(catalog.Closer); ok {
//	    defer closer.Close()
//	}
//
// All built-in catalogs (REST, SQL, Glue, Hive, Hadoop) implement Closer. Close
// releases the catalog's own resources; a catalog built on a caller-owned handle
// (e.g. the SQL catalog's *sql.DB) does not close that handle.
type Closer interface {
	Close() error
}

func ToIdentifier(ident ...string) table.Identifier {
	if len(ident) == 1 {
		if ident[0] == "" {
			return nil
		}

		return table.Identifier(strings.Split(ident[0], "."))
	}

	return table.Identifier(ident)
}

// ObjectNameFromIdent returns the name of a catalog object (a table, view,
// or function), which is the last element of its identifier.
func ObjectNameFromIdent(ident table.Identifier) string {
	if len(ident) == 0 {
		return ""
	}

	return ident[len(ident)-1]
}

func TableNameFromIdent(ident table.Identifier) string {
	return ObjectNameFromIdent(ident)
}

func NamespaceFromIdent(ident table.Identifier) table.Identifier {
	if len(ident) == 0 {
		return nil
	}

	return ident[:len(ident)-1]
}

func validateIdentifier(ident table.Identifier, notFoundErr error) error {
	if len(ident) < 2 {
		return fmt.Errorf("%w: missing namespace or invalid identifier %v",
			notFoundErr, strings.Join(ident, "."))
	}

	for _, part := range ident {
		if part == "" || part == "." || part == ".." || strings.Contains(part, "/") {
			return fmt.Errorf("%w: invalid identifier component %q in %v",
				notFoundErr, part, strings.Join(ident, "."))
		}

		for _, r := range part {
			if unicode.IsControl(r) {
				return fmt.Errorf("%w: invalid control character in identifier component %q in %v",
					notFoundErr, part, strings.Join(ident, "."))
			}
		}
	}

	return nil
}

// ValidateNamespaceIdentifier checks that an identifier contains at least one
// namespace level and no null characters. Other namespace component rules are
// intentionally left to the catalog implementation so existing namespaces
// remain readable across clients.
func ValidateNamespaceIdentifier(ident table.Identifier) error {
	if len(ident) == 0 {
		return fmt.Errorf("%w: empty namespace identifier", ErrNoSuchNamespace)
	}

	for _, part := range ident {
		if strings.ContainsRune(part, '\x00') {
			return fmt.Errorf("%w: invalid null character in namespace identifier component %q in %v",
				ErrNoSuchNamespace, part, strings.Join(ident, "."))
		}
	}

	return nil
}

// ValidateTableIdentifier checks that an identifier contains at least one valid namespace level and a table name.
func ValidateTableIdentifier(ident table.Identifier) error {
	return validateIdentifier(ident, ErrNoSuchTable)
}

// ValidateViewIdentifier checks that an identifier contains at least one valid namespace level and a view name.
func ValidateViewIdentifier(ident table.Identifier) error {
	return validateIdentifier(ident, ErrNoSuchView)
}

// ValidateFunctionIdentifier checks that an identifier contains at least one valid namespace level and a function name.
func ValidateFunctionIdentifier(ident table.Identifier) error {
	return validateIdentifier(ident, ErrNoSuchFunction)
}

type CreateTableOpt func(*CreateTableCfg)

func WithLocation(location string) CreateTableOpt {
	return func(cfg *CreateTableCfg) {
		cfg.Location = strings.TrimRight(location, "/")
	}
}

func WithPartitionSpec(spec *iceberg.PartitionSpec) CreateTableOpt {
	return func(cfg *CreateTableCfg) {
		cfg.PartitionSpec = spec
	}
}

func WithSortOrder(order table.SortOrder) CreateTableOpt {
	return func(cfg *CreateTableCfg) {
		cfg.SortOrder = order
	}
}

func WithProperties(props iceberg.Properties) CreateTableOpt {
	return func(cfg *CreateTableCfg) {
		cfg.Properties = props
	}
}

// WithStagedUpdates provides additional table.Update operations that
// cause the REST catalog to use two-phase staged creation. This is
// useful for atomically creating a table with a custom UUID, initial
// snapshots, or snapshot references. Use constructors from the table
// package (e.g. table.NewAssignUUIDUpdate, table.NewAddSnapshotUpdate,
// table.NewSetSnapshotRefUpdate).
func WithStagedUpdates(updates ...table.Update) CreateTableOpt {
	return func(cfg *CreateTableCfg) {
		cfg.StagedUpdates = append(cfg.StagedUpdates, updates...)
	}
}

type CreateViewOpt func(*CreateViewCfg)

func WithViewLocation(location string) CreateViewOpt {
	return func(cfg *CreateViewCfg) {
		cfg.Location = strings.TrimRight(location, "/")
	}
}

func WithViewProperties(config iceberg.Properties) CreateViewOpt {
	return func(cfg *CreateViewCfg) {
		cfg.Properties = config
	}
}

//lint:ignore U1000 this is linked to by catalogs via go:linkname but we don't want to export it
func checkForOverlap(removals []string, updates iceberg.Properties) error {
	overlap := []string{}
	for _, key := range removals {
		if _, ok := updates[key]; ok {
			overlap = append(overlap, key)
		}
	}
	if len(overlap) > 0 {
		return fmt.Errorf("conflict between removals and updates for keys: %v", overlap)
	}

	return nil
}

//lint:ignore U1000 this is linked to by catalogs via go:linkname but we don't want to export it
func getUpdatedPropsAndUpdateSummary(currentProps iceberg.Properties, removals []string, updates iceberg.Properties) (iceberg.Properties, PropertiesUpdateSummary, error) {
	if err := checkForOverlap(removals, updates); err != nil {
		return nil, PropertiesUpdateSummary{}, err
	}
	var (
		updatedProps = maps.Clone(currentProps)
		removed      = make([]string, 0, len(removals))
		updated      = make([]string, 0, len(updates))
	)

	for _, key := range removals {
		if _, exists := updatedProps[key]; exists {
			delete(updatedProps, key)
			removed = append(removed, key)
		}
	}

	for key, value := range updates {
		if updatedProps[key] != value {
			updated = append(updated, key)
			updatedProps[key] = value
		}
	}

	summary := PropertiesUpdateSummary{
		Removed: removed,
		Updated: updated,
		Missing: iceinternal.Difference(removals, removed),
	}

	return updatedProps, summary, nil
}
