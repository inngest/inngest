// Package icebergtest starts an embedded Iceberg catalog for tests that need
// a lake (see icebergcatalog.Open).
package icebergtest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/driver"
	"github.com/inngest/inngest/pkg/duckdb/icebergcatalog"
)

// Start starts an embedded catalog (sqlite + a local warehouse) on a temp
// dir for the rest of the test, and returns the options to attach it w/.
// pkg/duckdb/driver's own tests have a copy (testIceberg), since they can't
// import this package.
func Start(t testing.TB) *driver.IcebergOptions {
	t.Helper()
	dir := t.TempDir()
	e, err := icebergcatalog.Open(context.Background(), icebergcatalog.Config{
		SQLitePath: filepath.Join(dir, "catalog.db"),
		Warehouse:  filepath.Join(dir, "warehouse"),
	})
	if err != nil {
		t.Fatalf("starting iceberg catalog: %v", err)
	}
	t.Cleanup(func() { _ = e.Close() })
	return Options(e)
}

// Options are the driver options to attach e w/.
func Options(e *icebergcatalog.Embedded) *driver.IcebergOptions {
	return &driver.IcebergOptions{URI: e.URL, Warehouse: icebergcatalog.CatalogName, Token: e.Token, DataPath: e.Warehouse}
}
