package migrations

import (
	"database/sql"
	"fmt"
	"io"
	nurl "net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/atomic"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/hashicorp/go-multierror"
)

func init() {
	database.Register("duckdb", &DuckDB{})
}

var DefaultMigrationsTable = "schema_migrations"
var (
	ErrDatabaseDirty  = fmt.Errorf("database is dirty")
	ErrNilConfig      = fmt.Errorf("no config")
	ErrNoDatabaseName = fmt.Errorf("no database name")
)

type Config struct {
	MigrationsTable string
	DatabaseName    string
	NoTxWrap        bool
}

type DuckDB struct {
	db       *sql.DB
	isLocked atomic.Bool

	config *Config
}

func WithInstance(instance *sql.DB, config *Config) (database.Driver, error) {
	if config == nil {
		return nil, ErrNilConfig
	}

	if err := instance.Ping(); err != nil {
		return nil, err
	}

	if len(config.MigrationsTable) == 0 {
		config.MigrationsTable = DefaultMigrationsTable
	}

	mx := &DuckDB{
		db:     instance,
		config: config,
	}
	if err := mx.ensureVersionTable(); err != nil {
		return nil, err
	}
	return mx, nil
}

// ensureVersionTable checks if versions table exists and, if not, creates it.
// Note that this function locks the database, which deviates from the usual
// convention of "caller locks" in the Sqlite type.
func (m *DuckDB) ensureVersionTable() (err error) {
	if err = m.Lock(); err != nil {
		return err
	}

	defer func() {
		if e := m.Unlock(); e != nil {
			if err == nil {
				err = e
			} else {
				err = multierror.Append(err, e)
			}
		}
	}()

	query := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (version INTEGER, dirty BOOL, created_at TIMESTAMP);
  `, m.config.MigrationsTable)

	if _, err := m.db.Exec(query); err != nil {
		return err
	}
	return nil
}

func (m *DuckDB) Open(url string) (database.Driver, error) {
	purl, err := nurl.Parse(url)
	if err != nil {
		return nil, err
	}
	dbfile := strings.Replace(migrate.FilterCustomQuery(purl).String(), "duckdb://", "", 1)
	db, err := sql.Open("duckdb", dbfile)
	if err != nil {
		return nil, err
	}

	qv := purl.Query()

	migrationsTable := qv.Get("x-migrations-table")
	if len(migrationsTable) == 0 {
		migrationsTable = DefaultMigrationsTable
	}

	noTxWrap := false
	if v := qv.Get("x-no-tx-wrap"); v != "" {
		noTxWrap, err = strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("x-no-tx-wrap: %s", err)
		}
	}

	mx, err := WithInstance(db, &Config{
		DatabaseName:    purl.Path,
		MigrationsTable: migrationsTable,
		NoTxWrap:        noTxWrap,
	})
	if err != nil {
		return nil, err
	}
	return mx, nil
}

func (m *DuckDB) Close() error {
	return m.db.Close()
}

func (m *DuckDB) Drop() (err error) {
	return fmt.Errorf("unimplemented")
}

func (m *DuckDB) Lock() error {
	if !m.isLocked.CAS(false, true) {
		return database.ErrLocked
	}
	return nil
}

func (m *DuckDB) Unlock() error {
	if !m.isLocked.CAS(true, false) {
		return database.ErrNotLocked
	}
	return nil
}

func (m *DuckDB) Run(migration io.Reader) error {
	migr, err := io.ReadAll(migration)
	if err != nil {
		return err
	}
	query := string(migr[:])

	if m.config.NoTxWrap {
		return m.executeQueryNoTx(query)
	}
	return m.executeQuery(query)
}

func (m *DuckDB) executeQuery(query string) error {
	tx, err := m.db.Begin()
	if err != nil {
		return &database.Error{OrigErr: err, Err: "transaction start failed"}
	}
	if _, err := tx.Exec(query); err != nil {
		if errRollback := tx.Rollback(); errRollback != nil {
			err = multierror.Append(err, errRollback)
		}
		return &database.Error{OrigErr: err, Query: []byte(query)}
	}
	if err := tx.Commit(); err != nil {
		return &database.Error{OrigErr: err, Err: "transaction commit failed"}
	}
	return nil
}

func (m *DuckDB) executeQueryNoTx(query string) error {
	if _, err := m.db.Exec(query); err != nil {
		return &database.Error{OrigErr: err, Query: []byte(query)}
	}
	return nil
}

func (m *DuckDB) SetVersion(version int, dirty bool) error {
	// Also re-write the schema version for nil dirty versions to prevent
	// empty schema version for failed down migration on the first migration
	// See: https://github.com/golang-migrate/migrate/issues/330
	if version >= 0 || (version == database.NilVersion && dirty) {
		query := fmt.Sprintf(`INSERT INTO %s (version, dirty, created_at) VALUES (?, ?, ?)`, m.config.MigrationsTable)
		if _, err := m.db.Exec(query, version, dirty, time.Now()); err != nil {
			return &database.Error{OrigErr: err, Query: []byte(query)}
		}
	}

	return nil
}

func (m *DuckDB) Version() (version int, dirty bool, err error) {
	query := "SELECT version, dirty FROM " + m.config.MigrationsTable + " ORDER BY created_at DESC LIMIT 1"
	err = m.db.QueryRow(query).Scan(&version, &dirty)
	if err != nil {
		return database.NilVersion, false, nil
	}
	return version, dirty, nil
}
