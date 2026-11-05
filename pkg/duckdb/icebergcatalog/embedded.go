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

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// Config configures an embedded catalog started w/ Open.
type Config struct {
	// SQLitePath is the sqlite file the catalog keeps its table pointers in.
	// Exactly one of SQLitePath/PostgresURI is required.
	SQLitePath string
	// PostgresURI points the catalog at a postgres database instead (e.g.
	// the Inngest postgres). The `iceberg_tables` and
	// `iceberg_namespace_properties` tables go in the connection's default
	// schema, so set `search_path` in the URI to keep them somewhere else.
	PostgresURI string
	// Warehouse is the local directory table data + metadata go in, created
	// if missing. See TODO(s3).
	Warehouse string
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

	if strings.Contains(cfg.Warehouse, "://") {
		return nil, fmt.Errorf("icebergcatalog: only local warehouses are supported, got %q", cfg.Warehouse)
	}
	warehouse, err := filepath.Abs(cfg.Warehouse)
	if err != nil {
		return nil, fmt.Errorf("icebergcatalog: resolving warehouse: %w", err)
	}
	if err := os.MkdirAll(warehouse, 0o755); err != nil {
		return nil, fmt.Errorf("icebergcatalog: creating warehouse: %w", err)
	}

	token := cfg.Token
	if token == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("icebergcatalog: generating token: %w", err)
		}
		token = hex.EncodeToString(b)
	}

	var db *sql.DB
	if cfg.SQLitePath != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0o755); err != nil {
			return nil, fmt.Errorf("icebergcatalog: creating sqlite dir: %w", err)
		}
		db, err = sql.Open("sqlite", sqliteDSN(cfg.SQLitePath))
	} else {
		db, err = sql.Open("pgx", cfg.PostgresURI)
	}
	if err != nil {
		return nil, fmt.Errorf("icebergcatalog: opening catalog db: %w", err)
	}

	srv, err := New(ctx, db, Options{Warehouse: warehouse, Token: token, Logger: cfg.Logger})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("icebergcatalog: initializing catalog: %w", err)
	}

	srvCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cancel()
		_ = db.Close()
		return nil, fmt.Errorf("icebergcatalog: listening: %w", err)
	}
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

// sqliteDSN opens path w/ WAL + a busy timeout, since duckdb and the
// federated readers can hit the catalog concurrently.
func sqliteDSN(path string) string {
	return "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
}

// Close stops the server and closes the backing database.
func (e *Embedded) Close() error {
	e.cancel()
	err := <-e.done
	return errors.Join(err, e.db.Close())
}
