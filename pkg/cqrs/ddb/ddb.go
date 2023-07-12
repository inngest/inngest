package ddb

import (
	"database/sql"
	"embed"
	"fmt"
	"sync"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/inngest/inngest/pkg/cqrs/ddb/migrations"
	_ "github.com/marcboeker/go-duckdb"
)

var (
	o  sync.Once
	db *sql.DB
)

func New() (*sql.DB, error) {
	var err error
	o.Do(func() {
		db, err = sql.Open("duckdb", "?access_mode=READ_WRITE")
	})
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Run migrations.
	// TODO: Build a golang-migrate driver for DuckDB.
	if err := up(db); err != nil {
		return nil, err
	}

	return db, err
}

// FS contains the filesystem of the stdlib, containing all migrations in subdirs
// relative to this package.
//go:embed **/*.sql
var FS embed.FS

func up(db *sql.DB) error {
	source, err := iofs.New(FS, "migrations")
	if err != nil {
		return err
	}

	// Grab the migration driver.
	driver, err := migrations.WithInstance(db, &migrations.Config{
		MigrationsTable: "migrations",
	})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", source, "duckdb", driver)
	if err != nil {
		return err
	}

	v, dirty, err := m.Version()
	if err != migrate.ErrNilVersion && err != nil {
		return err
	}

	if dirty {
		if err = m.Migrate(v); err != nil {
			return fmt.Errorf("error migrating to version %d resetting dirty: %w", v, err)
		}
	}

	err = m.Up()
	if err == migrate.ErrNoChange {
		return nil
	}

	return err
}
