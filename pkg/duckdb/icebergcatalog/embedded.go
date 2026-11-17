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
)

// CatalogName is the name the sql catalog stores in its `catalog_name`
// column. It only matters if several catalogs share one backing database.
const CatalogName = "inngest"

// Dialect is the SQL a catalog's backing database speaks.
type Dialect = sqlcat.SupportedDialect

// The backing databases the catalog supports.
const (
	SQLite   = sqlcat.SQLite
	Postgres = sqlcat.Postgres
)

// Config configures an embedded catalog started w/ Open.
type Config struct {
	// DB is the backing database the catalog keeps its table pointers in,
	// opened by the caller w/ whichever database/sql driver it uses (this
	// package imports none), and Dialect says which SQL it speaks. The
	// caller owns DB: Close doesn't close it.
	//
	// A sqlite DB wants WAL and a busy timeout (IE
	// `file:<path>?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)`
	// w/ modernc.org/sqlite), since duckdb and readers can hit the catalog
	// concurrently. On postgres iceberg-go creates its `iceberg_tables` and
	// `iceberg_namespace_properties` tables in the connection's default
	// schema, so set `search_path` to keep them somewhere else.
	DB      *sql.DB
	Dialect Dialect
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
	// Retention configures snapshot expiry + metadata file cleanup, see
	// Retention. The zero value is on w/ the defaults.
	Retention Retention
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

	cancel context.CancelFunc
	done   <-chan error
}

// Open opens (and if needed initializes) the catalog's backing database and
// serves the REST API on a random 127.0.0.1 port until Close.
func Open(ctx context.Context, cfg Config) (*Embedded, error) {
	if cfg.DB == nil {
		return nil, errors.New("icebergcatalog: DB is required")
	}
	if cfg.Dialect != SQLite && cfg.Dialect != Postgres {
		return nil, fmt.Errorf("icebergcatalog: unsupported dialect %q", cfg.Dialect)
	}
	if cfg.Warehouse == "" {
		return nil, errors.New("icebergcatalog: Warehouse is required")
	}

	warehouse := cfg.Warehouse
	if !strings.Contains(warehouse, "://") {
		// Iceberg locations are URLs (iceberg-go, and duckdb on the other
		// end, parse them as such), so a local path w/ a character that
		// means something in a URL would be cut short (IE `/tmp/my#lake`
		// at the `#`) somewhere down the line. Refuse it here, before
		// anything's created.
		if i := strings.IndexAny(warehouse, "#?%"); i >= 0 {
			return nil, fmt.Errorf("icebergcatalog: local warehouse %q can't contain %q", warehouse, warehouse[i])
		}
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

	db, dialect := cfg.DB, cfg.Dialect
	props := iceberg.Properties{}
	for k, v := range cfg.Properties {
		props[k] = v
	}
	props["warehouse"] = warehouse
	cat, err := sqlcat.NewCatalog(CatalogName, db, dialect, props)
	if err != nil {
		return nil, fmt.Errorf("icebergcatalog: initializing sql catalog: %w", err)
	}

	srvCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cancel()
		return nil, fmt.Errorf("icebergcatalog: listening: %w", err)
	}
	srv := New(cat, Options{
		Warehouse: warehouse,
		Token:     token,
		Logger:    cfg.Logger,
		Retention: cfg.Retention,
		CatalogDB: &CatalogDB{DB: db, Dialect: dialect, Name: CatalogName, Properties: props},
	})
	done := make(chan error, 1)
	go func() { done <- srv.Start(srvCtx, l) }()

	return &Embedded{
		URL:       "http://" + l.Addr().String(),
		Token:     token,
		Warehouse: warehouse,
		cancel:    cancel,
		done:      done,
	}, nil
}

// Close stops the server. The backing database is the caller's to close.
func (e *Embedded) Close() error {
	e.cancel()
	return <-e.done
}
