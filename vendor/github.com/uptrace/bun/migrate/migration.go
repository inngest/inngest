package migrate

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/uptrace/bun"
)

// Migration represents a single database migration with up and down functions.
type Migration struct {
	bun.BaseModel

	ID         int64 `bun:",pk,autoincrement"`
	Name       string
	Comment    string `bun:"-"`
	GroupID    int64
	MigratedAt time.Time `bun:",notnull,nullzero,default:current_timestamp"`

	Up   internalMigrationFunc `bun:"-"`
	Down internalMigrationFunc `bun:"-"`
}

// String returns the migration name and comment.
func (m Migration) String() string {
	return fmt.Sprintf("%s_%s", m.Name, m.Comment)
}

// IsApplied reports whether the migration has been applied.
func (m Migration) IsApplied() bool {
	return m.ID > 0
}

// MigrationFunc is a function that executes a migration against a database.
type MigrationFunc func(ctx context.Context, db *bun.DB) error

type internalMigrationFunc func(ctx context.Context, migrator *Migrator, migration *Migration) error

func wrapGoMigrationFunc(fn MigrationFunc) internalMigrationFunc {
	return func(ctx context.Context, migrator *Migrator, migration *Migration) error {
		if migrator.beforeMigrationHook != nil {
			if err := migrator.beforeMigrationHook(ctx, migrator.db, migration); err != nil {
				return err
			}
		}

		if err := fn(ctx, migrator.db); err != nil {
			return err
		}

		if migrator.afterMigrationHook != nil {
			if err := migrator.afterMigrationHook(ctx, migrator.db, migration); err != nil {
				return err
			}
		}

		return nil
	}
}

func newSQLMigrationFunc(fsys fs.FS, name string) internalMigrationFunc {
	return func(ctx context.Context, migrator *Migrator, migration *Migration) error {
		sqlFile, err := fsys.Open(name)
		if err != nil {
			return err
		}

		contents, err := io.ReadAll(sqlFile)
		if err != nil {
			return err
		}

		var reader io.Reader = bytes.NewReader(contents)
		if migrator.templateData != nil {
			buf, err := renderTemplate(contents, migrator.templateData)
			if err != nil {
				return err
			}
			reader = buf
		}

		scanner := bufio.NewScanner(reader)
		var queries []string

		var query []byte
		for scanner.Scan() {
			b := scanner.Bytes()

			const prefix = "--bun:"
			if bytes.HasPrefix(b, []byte(prefix)) {
				b = b[len(prefix):]
				if bytes.Equal(b, []byte("split")) {
					queries = append(queries, string(query))
					query = query[:0]
					continue
				}
				return fmt.Errorf("bun: unknown directive: %q", b)
			}

			query = append(query, b...)
			query = append(query, '\n')
		}

		if len(query) > 0 {
			queries = append(queries, string(query))
		}
		if err := scanner.Err(); err != nil {
			return err
		}

		var idb bun.IConn

		isTx := strings.HasSuffix(name, ".tx.up.sql") || strings.HasSuffix(name, ".tx.down.sql")
		if isTx {
			tx, err := migrator.db.BeginTx(ctx, nil)
			if err != nil {
				return err
			}
			idb = tx
		} else {
			conn, err := migrator.db.Conn(ctx)
			if err != nil {
				return err
			}
			idb = conn
		}

		var retErr error
		var execErr error

		defer func() {
			if tx, ok := idb.(bun.Tx); ok {
				if execErr != nil {
					retErr = tx.Rollback()
				} else {
					retErr = tx.Commit()
				}
				return
			}

			if conn, ok := idb.(bun.Conn); ok {
				retErr = conn.Close()
				return
			}

			panic("not reached")
		}()

		execErr = migrator.exec(ctx, idb, migration, queries)
		if execErr != nil {
			return execErr
		}
		return retErr
	}
}

func renderTemplate(contents []byte, templateData any) (*bytes.Buffer, error) {
	tmpl, err := template.New("migration").Parse(string(contents))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, templateData); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return &rendered, nil
}

const goTemplate = `package %s

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] ")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [down migration] ")
		return nil
	})
}
`

const sqlTemplate = `SET statement_timeout = 0;

--bun:split

SELECT 1

--bun:split

SELECT 2
`

const transactionalSQLTemplate = `SET statement_timeout = 0;

SELECT 1;
`

//------------------------------------------------------------------------------

// MigrationSlice is a slice of migrations that provides helper methods for filtering and grouping.
type MigrationSlice []Migration

func (ms MigrationSlice) String() string {
	if len(ms) == 0 {
		return "empty"
	}

	if len(ms) > 5 {
		return fmt.Sprintf("%d migrations (%s ... %s)", len(ms), ms[0].Name, ms[len(ms)-1].Name)
	}

	var sb strings.Builder

	for i := range ms {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(ms[i].String())
	}

	return sb.String()
}

// Applied returns applied migrations in descending order
// (the order is important and is used in Rollback).
func (ms MigrationSlice) Applied() MigrationSlice {
	var applied MigrationSlice
	for i := range ms {
		if ms[i].IsApplied() {
			applied = append(applied, ms[i])
		}
	}
	sortDesc(applied)
	return applied
}

// Unapplied returns unapplied migrations in ascending order
// (the order is important and is used in Migrate).
func (ms MigrationSlice) Unapplied() MigrationSlice {
	var unapplied MigrationSlice
	for i := range ms {
		if !ms[i].IsApplied() {
			unapplied = append(unapplied, ms[i])
		}
	}
	sortAsc(unapplied)
	return unapplied
}

// LastGroupID returns the last applied migration group id.
// The id is 0 when there are no migration groups.
func (ms MigrationSlice) LastGroupID() int64 {
	var lastGroupID int64
	for i := range ms {
		groupID := ms[i].GroupID
		if groupID > lastGroupID {
			lastGroupID = groupID
		}
	}
	return lastGroupID
}

// LastGroup returns the last applied migration group.
func (ms MigrationSlice) LastGroup() *MigrationGroup {
	group := &MigrationGroup{
		ID: ms.LastGroupID(),
	}
	if group.ID == 0 {
		return group
	}
	for i := range ms {
		if ms[i].GroupID == group.ID {
			group.Migrations = append(group.Migrations, ms[i])
		}
	}
	return group
}

// MigrationGroup is a group of migrations that were applied together in a single Migrate call.
type MigrationGroup struct {
	ID         int64
	Migrations MigrationSlice
}

// IsZero reports whether the group is empty.
func (g MigrationGroup) IsZero() bool {
	return g.ID == 0 && len(g.Migrations) == 0
}

func (g MigrationGroup) String() string {
	if g.IsZero() {
		return "nil"
	}
	return fmt.Sprintf("group #%d (%s)", g.ID, g.Migrations)
}

// MigrationFile represents a generated migration file on disk.
type MigrationFile struct {
	Name    string
	Path    string
	Content string
}

//------------------------------------------------------------------------------

type migrationConfig struct {
	nop bool
}

func newMigrationConfig(opts []MigrationOption) *migrationConfig {
	cfg := new(migrationConfig)
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// MigrationOption configures how a migration is executed.
type MigrationOption func(cfg *migrationConfig)

// WithNopMigration creates a no-op migration that marks itself as applied without running.
func WithNopMigration() MigrationOption {
	return func(cfg *migrationConfig) {
		cfg.nop = true
	}
}

//------------------------------------------------------------------------------

func sortAsc(ms MigrationSlice) {
	slices.SortFunc(ms, func(a, b Migration) int {
		return strings.Compare(a.Name, b.Name)
	})
}

func sortDesc(ms MigrationSlice) {
	slices.SortFunc(ms, func(a, b Migration) int {
		return strings.Compare(b.Name, a.Name)
	})
}
