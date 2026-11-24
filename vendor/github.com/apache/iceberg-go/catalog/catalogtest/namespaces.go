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

package catalogtest

// This file covers namespaces: the structural lifecycle (create, exists, list,
// drop) and properties (create with properties, load, set, update, remove).
// Nested namespaces are deferred to a later slice of #1691, together with the
// Config.SupportsNestedNamespaces flag that would gate them.
//
// Java splits property writes across SupportsNamespaces.setProperties and
// removeProperties; the Go catalog has a single UpdateNamespaceProperties
// taking both, so the tests below drive the same behaviors through it and also
// check the PropertiesUpdateSummary it reports.

import (
	"context"
	"testing"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCreateNamespace asserts that a created namespace is reported as existing.
// Listing behavior is covered by testListNamespaces.
func testCreateNamespace(t *testing.T, cfg Config) {
	ctx := context.Background()
	cat := cfg.NewCatalog(t)
	namespace, _ := newIdentifiers()

	exists, err := cat.CheckNamespaceExists(ctx, namespace)
	require.NoError(t, err)
	assert.False(t, exists, "namespace should not exist before create")

	require.NoError(t, cat.CreateNamespace(ctx, namespace, nil))
	t.Cleanup(func() { _ = cat.DropNamespace(ctx, namespace) })

	exists, err = cat.CheckNamespaceExists(ctx, namespace)
	require.NoError(t, err)
	assert.True(t, exists, "namespace should exist after create")
}

// testCreateNamespaceThatAlreadyExists asserts that creating a namespace that
// already exists is rejected and leaves the original namespace intact.
func testCreateNamespaceThatAlreadyExists(t *testing.T, cfg Config) {
	ctx := context.Background()
	cat := cfg.NewCatalog(t)
	namespace, _ := newIdentifiers()

	exists, err := cat.CheckNamespaceExists(ctx, namespace)
	require.NoError(t, err)
	assert.False(t, exists, "namespace should not exist before create")

	require.NoError(t, cat.CreateNamespace(ctx, namespace, nil))
	t.Cleanup(func() { _ = cat.DropNamespace(ctx, namespace) })

	err = cat.CreateNamespace(ctx, namespace, nil)
	assert.ErrorIs(t, err, catalog.ErrNamespaceAlreadyExists)

	exists, err = cat.CheckNamespaceExists(ctx, namespace)
	require.NoError(t, err)
	assert.True(t, exists, "original namespace should still exist after a rejected duplicate create")
}

// testDropNamespace asserts that a dropped namespace no longer exists.
func testDropNamespace(t *testing.T, cfg Config) {
	ctx := context.Background()
	cat := cfg.NewCatalog(t)
	namespace, _ := newIdentifiers()

	require.NoError(t, cat.CreateNamespace(ctx, namespace, nil))
	t.Cleanup(func() { _ = cat.DropNamespace(ctx, namespace) })

	exists, err := cat.CheckNamespaceExists(ctx, namespace)
	require.NoError(t, err)
	assert.True(t, exists, "namespace should exist after create")

	require.NoError(t, cat.DropNamespace(ctx, namespace))

	exists, err = cat.CheckNamespaceExists(ctx, namespace)
	require.NoError(t, err)
	assert.False(t, exists, "namespace should not exist after drop")
}

// testDropMissingNamespace asserts that dropping a namespace that was never
// created reports that the namespace does not exist. This is a deliberate
// Go-idiom divergence from Java's SupportsNamespaces.dropNamespace, which
// returns false (and no error) for a missing namespace; the Go catalog
// interface returns catalog.ErrNoSuchNamespace instead.
func testDropMissingNamespace(t *testing.T, cfg Config) {
	ctx := context.Background()
	cat := cfg.NewCatalog(t)
	namespace, _ := newIdentifiers()

	err := cat.DropNamespace(ctx, namespace)
	assert.ErrorIs(t, err, catalog.ErrNoSuchNamespace)
}

// testDropNamespaceNotEmpty asserts that a namespace containing a table cannot
// be dropped.
func testDropNamespaceNotEmpty(t *testing.T, cfg Config) {
	ctx := context.Background()
	cat := cfg.NewCatalog(t)
	namespace, ident := newIdentifiers()

	require.NoError(t, cat.CreateNamespace(ctx, namespace, nil))
	t.Cleanup(func() {
		_ = cat.DropTable(ctx, ident)
		_ = cat.DropNamespace(ctx, namespace)
	})

	_, err := cat.CreateTable(ctx, ident, Schema)
	require.NoError(t, err)

	err = cat.DropNamespace(ctx, namespace)
	assert.ErrorIs(t, err, catalog.ErrNamespaceNotEmpty)

	// A rejected drop must leave the namespace and its table intact.
	nsExists, err := cat.CheckNamespaceExists(ctx, namespace)
	require.NoError(t, err)
	assert.True(t, nsExists, "namespace should still exist after a rejected non-empty drop")

	tableExists, err := cat.CheckTableExists(ctx, ident)
	require.NoError(t, err)
	assert.True(t, tableExists, "table should still exist after a rejected namespace drop")
}

// testListNamespaces asserts that created namespaces appear in the top-level
// listing and that a dropped namespace disappears from it while the others
// remain.
func testListNamespaces(t *testing.T, cfg Config) {
	ctx := context.Background()
	cat := cfg.NewCatalog(t)
	first, _ := newIdentifiers()
	second, _ := newIdentifiers()

	require.NoError(t, cat.CreateNamespace(ctx, first, nil))
	t.Cleanup(func() { _ = cat.DropNamespace(ctx, first) })
	require.NoError(t, cat.CreateNamespace(ctx, second, nil))
	t.Cleanup(func() { _ = cat.DropNamespace(ctx, second) })

	namespaces, err := cat.ListNamespaces(ctx, nil)
	require.NoError(t, err)
	assert.Contains(t, namespaces, first, "first created namespace should be listed")
	assert.Contains(t, namespaces, second, "second created namespace should be listed")

	require.NoError(t, cat.DropNamespace(ctx, first))

	namespaces, err = cat.ListNamespaces(ctx, nil)
	require.NoError(t, err)
	assert.NotContains(t, namespaces, first, "dropped namespace should no longer be listed")
	assert.Contains(t, namespaces, second, "remaining namespace should still be listed")
}

// skipWithoutNamespaceProperties exempts catalogs that cannot store namespace
// properties from the tests that write them.
func skipWithoutNamespaceProperties(t *testing.T, cfg Config) {
	t.Helper()

	if !cfg.SupportsNamespaceProperties {
		t.Skip("catalog does not support namespace properties")
	}
}

// testCreateNamespaceWithProperties asserts that properties passed to
// CreateNamespace are readable afterwards. A catalog may add properties of its
// own, so only the created one is checked.
func testCreateNamespaceWithProperties(t *testing.T, cfg Config) {
	skipWithoutNamespaceProperties(t, cfg)

	ctx := context.Background()
	cat := cfg.NewCatalog(t)
	namespace, _ := newIdentifiers()

	require.NoError(t, cat.CreateNamespace(ctx, namespace, iceberg.Properties{"prop": "val"}))
	t.Cleanup(func() { _ = cat.DropNamespace(ctx, namespace) })

	props, err := cat.LoadNamespaceProperties(ctx, namespace)
	require.NoError(t, err)
	assert.Equal(t, "val", props["prop"], "create properties should be readable after create")
}

// testLoadNamespaceProperties asserts that properties can be loaded for a
// namespace that exists and that loading a missing namespace is an error. A
// catalog is free to return whatever properties it likes for a namespace
// created without any, so nothing is asserted about their contents.
func testLoadNamespaceProperties(t *testing.T, cfg Config) {
	ctx := context.Background()
	cat := cfg.NewCatalog(t)
	namespace, _ := newIdentifiers()

	_, err := cat.LoadNamespaceProperties(ctx, namespace)
	assert.ErrorIs(t, err, catalog.ErrNoSuchNamespace)

	require.NoError(t, cat.CreateNamespace(ctx, namespace, nil))
	t.Cleanup(func() { _ = cat.DropNamespace(ctx, namespace) })

	props, err := cat.LoadNamespaceProperties(ctx, namespace)
	require.NoError(t, err)
	assert.NotNil(t, props, "loaded properties should be non-nil")
}

// testSetNamespaceProperties asserts that properties set on an existing
// namespace are reported as updated and are readable afterwards.
func testSetNamespaceProperties(t *testing.T, cfg Config) {
	skipWithoutNamespaceProperties(t, cfg)

	ctx := context.Background()
	cat := cfg.NewCatalog(t)
	namespace, _ := newIdentifiers()

	require.NoError(t, cat.CreateNamespace(ctx, namespace, nil))
	t.Cleanup(func() { _ = cat.DropNamespace(ctx, namespace) })

	updates := iceberg.Properties{"owner": "user", "created-at": "sometime"}
	summary, err := cat.UpdateNamespaceProperties(ctx, namespace, nil, updates)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"owner", "created-at"}, summary.Updated, "both properties should be reported as updated")
	assert.Empty(t, summary.Removed, "nothing was removed")

	props, err := cat.LoadNamespaceProperties(ctx, namespace)
	require.NoError(t, err)
	assert.Subset(t, props, updates, "set properties should be readable")
}

// testUpdateNamespaceProperties asserts that setting a property that already
// has a value overwrites it.
func testUpdateNamespaceProperties(t *testing.T, cfg Config) {
	skipWithoutNamespaceProperties(t, cfg)

	ctx := context.Background()
	cat := cfg.NewCatalog(t)
	namespace, _ := newIdentifiers()

	require.NoError(t, cat.CreateNamespace(ctx, namespace, nil))
	t.Cleanup(func() { _ = cat.DropNamespace(ctx, namespace) })

	_, err := cat.UpdateNamespaceProperties(ctx, namespace, nil, iceberg.Properties{"owner": "user"})
	require.NoError(t, err)

	summary, err := cat.UpdateNamespaceProperties(ctx, namespace, nil, iceberg.Properties{"owner": "newuser"})
	require.NoError(t, err)
	assert.Equal(t, []string{"owner"}, summary.Updated, "overwritten property should be reported as updated")

	props, err := cat.LoadNamespaceProperties(ctx, namespace)
	require.NoError(t, err)
	assert.Equal(t, "newuser", props["owner"], "property should hold the new value")
}

// testUpdateAndSetNamespaceProperties asserts that a single update may both
// overwrite an existing property and add a new one.
func testUpdateAndSetNamespaceProperties(t *testing.T, cfg Config) {
	skipWithoutNamespaceProperties(t, cfg)

	ctx := context.Background()
	cat := cfg.NewCatalog(t)
	namespace, _ := newIdentifiers()

	require.NoError(t, cat.CreateNamespace(ctx, namespace, nil))
	t.Cleanup(func() { _ = cat.DropNamespace(ctx, namespace) })

	_, err := cat.UpdateNamespaceProperties(ctx, namespace, nil, iceberg.Properties{"owner": "user"})
	require.NoError(t, err)

	updates := iceberg.Properties{"owner": "newuser", "last-modified-at": "now"}
	summary, err := cat.UpdateNamespaceProperties(ctx, namespace, nil, updates)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"owner", "last-modified-at"}, summary.Updated, "the overwritten and the added property should both be reported as updated")

	props, err := cat.LoadNamespaceProperties(ctx, namespace)
	require.NoError(t, err)
	assert.Subset(t, props, updates, "updated properties should be readable")
}

// testRemoveNamespaceProperties asserts that a removed property is gone and
// that the properties left alone survive.
func testRemoveNamespaceProperties(t *testing.T, cfg Config) {
	skipWithoutNamespaceProperties(t, cfg)

	ctx := context.Background()
	cat := cfg.NewCatalog(t)
	namespace, _ := newIdentifiers()

	require.NoError(t, cat.CreateNamespace(ctx, namespace, nil))
	t.Cleanup(func() { _ = cat.DropNamespace(ctx, namespace) })

	updates := iceberg.Properties{"owner": "user", "created-at": "sometime"}
	_, err := cat.UpdateNamespaceProperties(ctx, namespace, nil, updates)
	require.NoError(t, err)

	summary, err := cat.UpdateNamespaceProperties(ctx, namespace, []string{"created-at"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"created-at"}, summary.Removed, "removed property should be reported as removed")
	assert.Empty(t, summary.Updated, "nothing was updated")

	props, err := cat.LoadNamespaceProperties(ctx, namespace)
	require.NoError(t, err)
	assert.NotContains(t, props, "created-at", "removed property should be gone")
	assert.Equal(t, "user", props["owner"], "untouched property should survive the removal")
}

// testSetNamespacePropertiesNamespaceDoesNotExist asserts that setting
// properties on a namespace that was never created is an error.
func testSetNamespacePropertiesNamespaceDoesNotExist(t *testing.T, cfg Config) {
	skipWithoutNamespaceProperties(t, cfg)

	ctx := context.Background()
	cat := cfg.NewCatalog(t)
	namespace, _ := newIdentifiers()

	_, err := cat.UpdateNamespaceProperties(ctx, namespace, nil, iceberg.Properties{"test": "value"})
	assert.ErrorIs(t, err, catalog.ErrNoSuchNamespace)
}

// testRemoveNamespacePropertiesNamespaceDoesNotExist asserts that removing
// properties from a namespace that was never created is an error.
func testRemoveNamespacePropertiesNamespaceDoesNotExist(t *testing.T, cfg Config) {
	skipWithoutNamespaceProperties(t, cfg)

	ctx := context.Background()
	cat := cfg.NewCatalog(t)
	namespace, _ := newIdentifiers()

	_, err := cat.UpdateNamespaceProperties(ctx, namespace, []string{"a", "b"}, nil)
	assert.ErrorIs(t, err, catalog.ErrNoSuchNamespace)
}
