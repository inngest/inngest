package sqlitecqrs

import (
	"database/sql"
	"embed"
	"fmt"
	"sync"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"
)

var (
	o  sync.Once
	db *sql.DB
)

func New() (*sql.DB, error) {
	var err error
	o.Do(func() {
		db, err = sql.Open("sqlite", "file:inngest?mode=memory&cache=shared")
	})

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Run migrations.
	if err := up(db); err != nil {
		return nil, err
	}

	return db, err
}

// FS contains the filesystem of the stdlib, containing all migrations in subdirs
// relative to this package.
//
//go:embed **/*.sql
var FS embed.FS

func up(db *sql.DB) error {
	source, err := iofs.New(FS, "migrations")
	if err != nil {
		return err
	}

	// Grab the migration driver.
	driver, err := sqlite.WithInstance(db, &sqlite.Config{
		MigrationsTable: "migrations",
		NoTxWrap:        true,
	})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", source, "file:inngest?mode=memory&cache=shared", driver)
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
