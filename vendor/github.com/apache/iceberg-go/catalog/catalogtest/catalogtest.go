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

// Package catalogtest provides a conformance suite that every
// [catalog.Catalog] implementation can run against itself, so that shared
// behavior is specified once rather than re-derived in each implementation's
// tests. It mirrors the role of CatalogTests in the Java implementation.
//
// An implementation opts in from its own test package by describing itself
// with a [Config] and calling [RunCatalogTests]:
//
//	func TestCatalogConformance(t *testing.T) {
//		catalogtest.RunCatalogTests(t, catalogtest.Config{
//			NewCatalog: func(t *testing.T) catalog.Catalog { ... },
//		})
//	}
//
// Each test gets its own catalog from Config.NewCatalog, so tests never share
// state and may run in any order.
package catalogtest

import (
	"context"
	"strings"
	"testing"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/table"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newIdentifiers returns a namespace and a table within it, named uniquely per
// call. Catalogs backed by a shared server (a REST fixture, a Hive metastore)
// outlive a single test, so fixed names would collide between tests and
// between runs.
func newIdentifiers() (namespace, tbl table.Identifier) {
	namespace = table.Identifier{"conformance_" + strings.ReplaceAll(uuid.NewString(), "-", "_")}

	return namespace, table.Identifier{namespace[0], "tbl"}
}

// Schema is the schema the conformance tests create tables with. Its field IDs
// are deliberately not 1 and 2: catalogs assign fresh IDs on create, and
// [TableSchema] is what the created table is expected to report back.
var Schema = iceberg.NewSchema(0,
	iceberg.NestedField{ID: 3, Name: "id", Type: iceberg.PrimitiveTypes.Int32, Required: true, Doc: "unique ID 🤪"},
	iceberg.NestedField{ID: 4, Name: "data", Type: iceberg.PrimitiveTypes.String, Required: true},
)

// TableSchema is [Schema] with the fresh field IDs a catalog assigns on create.
var TableSchema = iceberg.NewSchema(0,
	iceberg.NestedField{ID: 1, Name: "id", Type: iceberg.PrimitiveTypes.Int32, Required: true, Doc: "unique ID 🤪"},
	iceberg.NestedField{ID: 2, Name: "data", Type: iceberg.PrimitiveTypes.String, Required: true},
)

// OtherSchema is a schema that is not [Schema], used to check that a rejected
// create leaves the existing table alone.
var OtherSchema = iceberg.NewSchema(0,
	iceberg.NestedField{ID: 1, Name: "some_id", Type: iceberg.PrimitiveTypes.Int32, Required: true},
)

// Config describes the catalog under test.
type Config struct {
	// NewCatalog returns a fresh, empty catalog for a single test. It is called
	// once per test; implementations should register any teardown with
	// t.Cleanup rather than relying on the suite to release resources.
	NewCatalog func(t *testing.T) catalog.Catalog

	// SupportsNamespaceProperties reports whether the catalog can store
	// properties against a namespace. Tests that write properties are skipped
	// when it is false, as they are for the Hadoop catalog, whose namespaces
	// are directories with nowhere to keep them. Loading is required of every
	// catalog and is tested either way.
	SupportsNamespaceProperties bool
}

// RunCatalogTests runs the conformance suite against the catalog described by
// cfg. Each test runs as a subtest so a failure names the behavior that broke.
func RunCatalogTests(t *testing.T, cfg Config) {
	t.Helper()
	require.NotNil(t, cfg.NewCatalog, "catalogtest.Config.NewCatalog must be set")

	t.Run("BasicCreateTable", func(t *testing.T) { testBasicCreateTable(t, cfg) })
	t.Run("BasicCreateTableThatAlreadyExists", func(t *testing.T) { testBasicCreateTableThatAlreadyExists(t, cfg) })
	t.Run("LoadMissingTable", func(t *testing.T) { testLoadMissingTable(t, cfg) })
	t.Run("LoadTableWithNonExistingNamespace", func(t *testing.T) { testLoadTableWithNonExistingNamespace(t, cfg) })

	t.Run("CreateNamespace", func(t *testing.T) { testCreateNamespace(t, cfg) })
	t.Run("CreateNamespaceThatAlreadyExists", func(t *testing.T) { testCreateNamespaceThatAlreadyExists(t, cfg) })
	t.Run("DropNamespace", func(t *testing.T) { testDropNamespace(t, cfg) })
	t.Run("DropMissingNamespace", func(t *testing.T) { testDropMissingNamespace(t, cfg) })
	t.Run("DropNamespaceNotEmpty", func(t *testing.T) { testDropNamespaceNotEmpty(t, cfg) })
	t.Run("ListNamespaces", func(t *testing.T) { testListNamespaces(t, cfg) })

	t.Run("CreateNamespaceWithProperties", func(t *testing.T) { testCreateNamespaceWithProperties(t, cfg) })
	t.Run("LoadNamespaceProperties", func(t *testing.T) { testLoadNamespaceProperties(t, cfg) })
	t.Run("SetNamespaceProperties", func(t *testing.T) { testSetNamespaceProperties(t, cfg) })
	t.Run("UpdateNamespaceProperties", func(t *testing.T) { testUpdateNamespaceProperties(t, cfg) })
	t.Run("UpdateAndSetNamespaceProperties", func(t *testing.T) { testUpdateAndSetNamespaceProperties(t, cfg) })
	t.Run("RemoveNamespaceProperties", func(t *testing.T) { testRemoveNamespaceProperties(t, cfg) })
	t.Run("SetNamespacePropertiesNamespaceDoesNotExist", func(t *testing.T) { testSetNamespacePropertiesNamespaceDoesNotExist(t, cfg) })
	t.Run("RemoveNamespacePropertiesNamespaceDoesNotExist", func(t *testing.T) { testRemoveNamespacePropertiesNamespaceDoesNotExist(t, cfg) })
}

// testBasicCreateTable asserts that a newly created table is visible to the
// catalog and reports the settings it was created with.
func testBasicCreateTable(t *testing.T, cfg Config) {
	ctx := context.Background()
	cat := cfg.NewCatalog(t)
	namespace, ident := newIdentifiers()

	exists, err := cat.CheckTableExists(ctx, ident)
	require.NoError(t, err)
	assert.False(t, exists, "table should not exist before create")

	require.NoError(t, cat.CreateNamespace(ctx, namespace, nil))
	t.Cleanup(func() { _ = cat.DropNamespace(ctx, namespace) })

	tbl, err := cat.CreateTable(ctx, ident, Schema)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cat.DropTable(ctx, ident) })

	exists, err = cat.CheckTableExists(ctx, ident)
	require.NoError(t, err)
	assert.True(t, exists, "table should exist after create")

	assert.Equal(t, ident, tbl.Identifier(), "table should report the identifier it was created with")
	assert.Equal(t, TableSchema.AsStruct(), tbl.Schema().AsStruct(), "schema should match expected ID assignment")
	assert.NotEmpty(t, tbl.Location(), "table should have a location")
	assert.True(t, tbl.Spec().IsUnpartitioned(), "table should be unpartitioned")
	assert.True(t, tbl.SortOrder().IsUnsorted(), "table should be unsorted")
	assert.NotNil(t, tbl.Properties(), "table should have properties")
}

// testBasicCreateTableThatAlreadyExists asserts that creating a table over an
// existing one is rejected and leaves the original table intact.
func testBasicCreateTableThatAlreadyExists(t *testing.T, cfg Config) {
	ctx := context.Background()
	cat := cfg.NewCatalog(t)
	namespace, ident := newIdentifiers()

	require.NoError(t, cat.CreateNamespace(ctx, namespace, nil))
	t.Cleanup(func() { _ = cat.DropNamespace(ctx, namespace) })

	exists, err := cat.CheckTableExists(ctx, ident)
	require.NoError(t, err)
	assert.False(t, exists, "table should not exist before create")

	_, err = cat.CreateTable(ctx, ident, Schema)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cat.DropTable(ctx, ident) })

	exists, err = cat.CheckTableExists(ctx, ident)
	require.NoError(t, err)
	assert.True(t, exists, "table should exist after create")

	_, err = cat.CreateTable(ctx, ident, OtherSchema)
	assert.ErrorIs(t, err, catalog.ErrTableAlreadyExists)

	tbl, err := cat.LoadTable(ctx, ident)
	require.NoError(t, err)
	assert.Equal(t, TableSchema.AsStruct(), tbl.Schema().AsStruct(), "schema should match the original table schema")
}

// testLoadMissingTable asserts that loading a table that was never created
// reports that the table does not exist.
func testLoadMissingTable(t *testing.T, cfg Config) {
	ctx := context.Background()
	cat := cfg.NewCatalog(t)
	namespace, ident := newIdentifiers()

	require.NoError(t, cat.CreateNamespace(ctx, namespace, nil))
	t.Cleanup(func() { _ = cat.DropNamespace(ctx, namespace) })

	exists, err := cat.CheckTableExists(ctx, ident)
	require.NoError(t, err)
	assert.False(t, exists, "table should not exist")

	_, err = cat.LoadTable(ctx, ident)
	assert.ErrorIs(t, err, catalog.ErrNoSuchTable)
}

// testLoadTableWithNonExistingNamespace asserts that a missing namespace is
// reported as a missing table, not as a missing namespace.
func testLoadTableWithNonExistingNamespace(t *testing.T, cfg Config) {
	ctx := context.Background()
	cat := cfg.NewCatalog(t)
	_, ident := newIdentifiers()

	exists, err := cat.CheckTableExists(ctx, ident)
	require.NoError(t, err)
	assert.False(t, exists, "table should not exist")

	_, err = cat.LoadTable(ctx, ident)
	assert.ErrorIs(t, err, catalog.ErrNoSuchTable)
}
