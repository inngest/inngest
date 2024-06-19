package sqlitecqrs

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"sync"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/inngest/inngest/pkg/consts"
	_ "modernc.org/sqlite"
)

var (
	o  sync.Once
	db *sql.DB
)

func New() (*sql.DB, error) {
	var err error
	o.Do(func() {
		// make the dir if it doesn't exist
		if _, err := os.Stat(consts.DevServerTempDir); os.IsNotExist(err) {
			err = os.Mkdir(consts.DevServerTempDir, 0755)
			if err != nil {
				return
			}
		}

		db, err = sql.Open("sqlite", fmt.Sprintf("file:%s?cache=shared", fmt.Sprintf("%s/%s", consts.DevServerTempDir, consts.DevServerDbFile)))
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

	m, err := migrate.NewWithInstance("iofs", source, fmt.Sprintf("file:%s?cache=shared", fmt.Sprintf("%s/%s", consts.DevServerTempDir, consts.DevServerDbFile)), driver)
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
