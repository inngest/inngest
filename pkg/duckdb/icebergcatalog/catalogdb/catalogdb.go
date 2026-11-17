// Package catalogdb opens an embedded Iceberg catalog on a sqlite file or a
// postgres database, for the OSS binaries and tests. It registers the sqlite
// and pgx database/sql drivers, which icebergcatalog itself doesn't, so a
// program that brings its own icebergcatalog.Config.DB doesn't link them.
package catalogdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/inngest/inngest/pkg/duckdb/icebergcatalog"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// Backend is where a catalog keeps its table pointers: exactly one of a
// sqlite file or a postgres URI.
type Backend struct {
	SQLitePath  string
	PostgresURI string
}

// SQLite is a sqlite file backend.
func SQLite(path string) Backend { return Backend{SQLitePath: path} }

// Postgres is a postgres backend. iceberg-go creates its tables in the
// connection's default schema, so set `search_path` in uri to keep them
// somewhere else.
func Postgres(uri string) Backend { return Backend{PostgresURI: uri} }

// OpenDB opens b's database and returns its dialect. A sqlite one gets WAL
// and a busy timeout (per iceberg-go's NewCatalog docs), since duckdb and
// readers can hit the catalog concurrently.
func (b Backend) OpenDB() (*sql.DB, icebergcatalog.Dialect, error) {
	if (b.SQLitePath == "") == (b.PostgresURI == "") {
		return nil, "", errors.New("catalogdb: exactly one of SQLitePath/PostgresURI is required")
	}
	if b.PostgresURI != "" {
		db, err := sql.Open("pgx", b.PostgresURI)
		return db, icebergcatalog.Postgres, err
	}
	if err := os.MkdirAll(filepath.Dir(b.SQLitePath), 0o755); err != nil {
		return nil, "", fmt.Errorf("catalogdb: creating sqlite dir: %w", err)
	}
	db, err := sql.Open("sqlite", sqliteDSN(b.SQLitePath))
	return db, icebergcatalog.SQLite, err
}

// sqliteDSN is a file: URI for path w/ the pragmas OpenDB wants. The path is
// escaped, so a `#`, `?` or `%` in it (IE in --sqlite-dir) can't end it
// early or change which file sqlite opens.
func sqliteDSN(path string) string {
	u := url.URL{Scheme: "file", RawQuery: "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"}
	if filepath.IsAbs(path) {
		u.Path = filepath.ToSlash(path)
	} else {
		// file:rel/path, a relative path can't go in an authority-less
		// URL's Path.
		u.Opaque = (&url.URL{Path: filepath.ToSlash(path)}).EscapedPath()
	}
	return u.String()
}

// Catalog is an embedded catalog that owns its backing database.
type Catalog struct {
	*icebergcatalog.Embedded
	db *sql.DB
}

// Open opens b's database and starts an embedded catalog on it w/ cfg (its
// DB and Dialect are set here).
func Open(ctx context.Context, b Backend, cfg icebergcatalog.Config) (*Catalog, error) {
	db, dialect, err := b.OpenDB()
	if err != nil {
		return nil, fmt.Errorf("catalogdb: opening catalog db: %w", err)
	}
	cfg.DB, cfg.Dialect = db, dialect
	e, err := icebergcatalog.Open(ctx, cfg)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Catalog{Embedded: e, db: db}, nil
}

// Close stops the catalog, then closes its database.
func (c *Catalog) Close() error {
	return errors.Join(c.Embedded.Close(), c.db.Close())
}
