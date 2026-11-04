package icebergcatalog

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/apache/iceberg-go"
	sqlcat "github.com/apache/iceberg-go/catalog/sql"
	// Registers the s3:// scheme so the catalog can write table metadata to
	// an S3 warehouse. Local paths need nothing extra.
	_ "github.com/apache/iceberg-go/io/gocloud/s3"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// CatalogName is the name the sql catalog stores in its `catalog_name`
// column. It only matters if several catalogs share one backing database.
const CatalogName = "inngest"

// Config configures an embedded catalog started w/ Open.
type Config struct {
	// SQLitePath is the sqlite file the catalog keeps its table pointers in.
	// Exactly one of SQLitePath/PostgresURI is required.
	SQLitePath string
	// PostgresURI points the catalog at a postgres database instead (e.g.
	// the Inngest postgres). iceberg-go creates its `iceberg_tables` and
	// `iceberg_namespace_properties` tables in the connection's default
	// schema, so set `search_path` in the URI to keep them somewhere else.
	PostgresURI string
	// Warehouse is where table data + metadata go: a local directory or an
	// `s3://bucket/prefix` URI. Local directories are created if missing.
	Warehouse string
	// Properties are passed through to the sql catalog and its IO (e.g.
	// `s3.region`, `s3.endpoint`, `s3.access-key-id` for an S3 warehouse).
	Properties iceberg.Properties
	// Token is the bearer token clients must send. Left empty, Open
	// generates a random one (see Embedded.Token).
	Token string
	// Logger, when non-nil, gets a debug line per request.
	Logger *slog.Logger
}

// Embedded is a running catalog bound to 127.0.0.1.
type Embedded struct {
	// URL is the catalog base URL (`http://127.0.0.1:<port>`, no `/v1`),
	// what duckdb's `ATTACH ... (TYPE iceberg, URI ...)` wants.
	URL string
	// Token is the bearer token the server requires.
	Token string
	// Warehouse is Config.Warehouse w/ local paths made absolute.
	Warehouse string

	db     *sql.DB
	cancel context.CancelFunc
	done   <-chan error
}

// Open opens (and if needed initializes) the catalog's backing database and
// serves the REST API on a random 127.0.0.1 port until Close.
func Open(ctx context.Context, cfg Config) (*Embedded, error) {
	if (cfg.SQLitePath == "") == (cfg.PostgresURI == "") {
		return nil, errors.New("icebergcatalog: exactly one of SQLitePath/PostgresURI is required")
	}
	if cfg.Warehouse == "" {
		return nil, errors.New("icebergcatalog: Warehouse is required")
	}

	warehouse := cfg.Warehouse
	if !strings.Contains(warehouse, "://") {
		abs, err := filepath.Abs(warehouse)
		if err != nil {
			return nil, fmt.Errorf("icebergcatalog: resolving warehouse: %w", err)
		}
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return nil, fmt.Errorf("icebergcatalog: creating warehouse: %w", err)
		}
		warehouse = abs
	}

	token := cfg.Token
	if token == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("icebergcatalog: generating token: %w", err)
		}
		token = hex.EncodeToString(b)
	}

	var (
		db      *sql.DB
		dialect sqlcat.SupportedDialect
		err     error
	)
	if cfg.SQLitePath != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0o755); err != nil {
			return nil, fmt.Errorf("icebergcatalog: creating sqlite dir: %w", err)
		}
		// WAL + busy_timeout per iceberg-go's NewCatalog docs, since duckdb
		// and the federated readers can hit the catalog concurrently.
		db, err = sql.Open("sqlite", "file:"+cfg.SQLitePath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
		dialect = sqlcat.SQLite
	} else {
		db, err = sql.Open("pgx", cfg.PostgresURI)
		dialect = sqlcat.Postgres
	}
	if err != nil {
		return nil, fmt.Errorf("icebergcatalog: opening catalog db: %w", err)
	}

	props := iceberg.Properties{}
	for k, v := range cfg.Properties {
		props[k] = v
	}
	props["warehouse"] = warehouse
	cat, err := sqlcat.NewCatalog(CatalogName, db, dialect, props)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("icebergcatalog: initializing sql catalog: %w", err)
	}

	srvCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cancel()
		_ = db.Close()
		return nil, fmt.Errorf("icebergcatalog: listening: %w", err)
	}
	srv := New(cat, Options{Warehouse: warehouse, Token: token, Logger: cfg.Logger})
	done := make(chan error, 1)
	go func() { done <- srv.Start(srvCtx, l) }()

	return &Embedded{
		URL:       "http://" + l.Addr().String(),
		Token:     token,
		Warehouse: warehouse,
		db:        db,
		cancel:    cancel,
		done:      done,
	}, nil
}

// Close stops the server and closes the backing database.
func (e *Embedded) Close() error {
	e.cancel()
	err := <-e.done
	return errors.Join(err, e.db.Close())
}
