package migrate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/migrate/sqlschema"
	"github.com/uptrace/bun/schema"
)

// AutoMigratorOption configures an AutoMigrator.
type AutoMigratorOption func(m *AutoMigrator)

// WithModel adds a bun.Model to the migration scope.
func WithModel(models ...any) AutoMigratorOption {
	return func(m *AutoMigrator) {
		m.includeModels = append(m.includeModels, models...)
	}
}

// WithExcludeTable tells AutoMigrator to exclude database tables from the migration scope.
// This prevents AutoMigrator from dropping tables which may exist in the schema
// but which are not used by the application.
//
// Expressions may make use of the wildcards supported by the SQL LIKE operator:
//   - % as a wildcard
//   - _ as a single character
//
// Do not exclude tables included via WithModel, as BunModelInspector ignores this setting.
func WithExcludeTable(tables ...string) AutoMigratorOption {
	return func(m *AutoMigrator) {
		m.excludeTables = append(m.excludeTables, tables...)
	}
}

// WithExcludeForeignKeys tells AutoMigrator to exclude a foreign key constaint
// from the migration scope. This prevents AutoMigrator from dropping foreign keys
// that are defined manually via CreateTableQuery.ForeignKey().
func WithExcludeForeignKeys(fks ...sqlschema.ForeignKey) AutoMigratorOption {
	return func(m *AutoMigrator) {
		m.excludeForeignKeys = append(m.excludeForeignKeys, fks...)
	}
}

// WithSchemaName sets the database schema to migrate objects in.
// By default, dialects' default schema is used.
func WithSchemaName(schemaName string) AutoMigratorOption {
	return func(m *AutoMigrator) {
		m.schemaName = schemaName
	}
}

// WithTableNameAuto overrides default migrations table name.
func WithTableNameAuto(table string) AutoMigratorOption {
	return func(m *AutoMigrator) {
		m.table = table
		m.migratorOpts = append(m.migratorOpts, WithTableName(table))
	}
}

// WithLocksTableNameAuto overrides default migration locks table name.
func WithLocksTableNameAuto(table string) AutoMigratorOption {
	return func(m *AutoMigrator) {
		m.locksTable = table
		m.migratorOpts = append(m.migratorOpts, WithLocksTableName(table))
	}
}

// WithMarkAppliedOnSuccessAuto sets the migrator to only mark migrations as applied/unapplied
// when their up/down is successful.
func WithMarkAppliedOnSuccessAuto(enabled bool) AutoMigratorOption {
	return func(m *AutoMigrator) {
		m.migratorOpts = append(m.migratorOpts, WithMarkAppliedOnSuccess(enabled))
	}
}

// WithMigrationsDirectoryAuto overrides the default directory for migration files.
func WithMigrationsDirectoryAuto(directory string) AutoMigratorOption {
	return func(m *AutoMigrator) {
		m.migrationsOpts = append(m.migrationsOpts, WithMigrationsDirectory(directory))
	}
}

// AutoMigrator performs automated schema migrations.
//
// It is designed to be a drop-in replacement for some Migrator functionality and supports all existing
// configuration options.
// Similarly to Migrator, it has methods to create SQL migrations, write them to a file, and apply them.
// Unlike Migrator, it detects the differences between the state defined by bun models and the current
// database schema automatically.
//
// Usage:
//  1. Generate migrations and apply them at once with AutoMigrator.Migrate().
//  2. Create up- and down-SQL migration files and apply migrations using Migrator.Migrate().
//
// While both methods produce complete, reversible migrations (with entries in the database
// and SQL migration files), prefer creating migrations and applying them separately for
// any non-trivial cases to ensure AutoMigrator detects expected changes correctly.
//
// Limitations:
//   - AutoMigrator only supports a subset of the possible ALTER TABLE modifications.
//   - Some changes are not automatically reversible. For example, you would need to manually
//     add a CREATE TABLE query to the .down migration file to revert a DROP TABLE migration.
//   - Does not validate most dialect-specific constraints. For example, when changing column
//     data type, make sure the data con be auto-casted to the new type.
//   - Due to how the schema-state diff is calculated, it is not possible to rename a table and
//     modify any of its columns' _data type_ in a single run. This will cause the AutoMigrator
//     to drop and re-create the table under a different name; it is better to apply this change in 2 steps.
//     Renaming a table and renaming its columns at the same time is possible.
//   - Renaming table/column to an existing name, i.e. like this [A->B] [B->C], is not possible due to how
//     AutoMigrator distinguishes "rename" and "unchanged" columns.
//
// Dialect must implement both sqlschema.Inspector and sqlschema.Migrator to be used with AutoMigrator.
type AutoMigrator struct {
	db *bun.DB

	// dbInspector creates the current state for the target database.
	dbInspector sqlschema.Inspector

	// modelInspector creates the desired state based on the model definitions.
	modelInspector sqlschema.Inspector

	// dbMigrator executes ALTER TABLE queries.
	dbMigrator sqlschema.Migrator

	table      string // Migrations table (excluded from database inspection)
	locksTable string // Migration locks table (excluded from database inspection)

	// schemaName is the database schema considered for migration.
	schemaName string

	// includeModels define the migration scope.
	includeModels []any

	excludeTables      []string               // excludeTables are excluded from database inspection.
	excludeForeignKeys []sqlschema.ForeignKey // excludeForeignKeys are excluded from database inspection.

	// diffOpts are passed to detector constructor.
	diffOpts []diffOption

	// migratorOpts are passed to Migrator constructor.
	migratorOpts []MigratorOption

	// migrationsOpts are passed to Migrations constructor.
	migrationsOpts []MigrationsOption
}

// NewAutoMigrator creates an AutoMigrator that detects schema differences and generates migrations.
func NewAutoMigrator(db *bun.DB, opts ...AutoMigratorOption) (*AutoMigrator, error) {
	am := &AutoMigrator{
		db:         db,
		table:      defaultTable,
		locksTable: defaultLocksTable,
		schemaName: db.Dialect().DefaultSchema(),
	}

	for _, opt := range opts {
		opt(am)
	}
	am.excludeTables = append(am.excludeTables, am.table, am.locksTable)

	dbInspector, err := sqlschema.NewInspector(db,
		sqlschema.WithSchemaName(am.schemaName),
		sqlschema.WithExcludeTables(am.excludeTables...),
		sqlschema.WithExcludeForeignKeys(am.excludeForeignKeys...),
	)
	if err != nil {
		return nil, err
	}
	am.dbInspector = dbInspector
	am.diffOpts = append(am.diffOpts, withCompareTypeFunc(db.Dialect().(sqlschema.InspectorDialect).CompareType))

	dbMigrator, err := sqlschema.NewMigrator(db, am.schemaName)
	if err != nil {
		return nil, err
	}
	am.dbMigrator = dbMigrator

	tables := schema.NewTables(db.Dialect())
	tables.Register(am.includeModels...)
	am.modelInspector = sqlschema.NewBunModelInspector(tables, sqlschema.WithSchemaName(am.schemaName))

	return am, nil
}

func (am *AutoMigrator) plan(ctx context.Context) (*changeset, error) {
	var err error

	got, err := am.dbInspector.Inspect(ctx)
	if err != nil {
		return nil, err
	}

	want, err := am.modelInspector.Inspect(ctx)
	if err != nil {
		return nil, err
	}

	changes := diff(got, want, am.diffOpts...)
	if err := changes.ResolveDependencies(); err != nil {
		return nil, fmt.Errorf("plan migrations: %w", err)
	}
	return changes, nil
}

// Migrate writes required changes to a new migration file and runs the migration.
// This will create an entry in the migrations table, making it possible to revert
// the changes with Migrator.Rollback(). MigrationOptions are passed on to Migrator.Migrate().
func (am *AutoMigrator) Migrate(ctx context.Context, opts ...MigrationOption) (*MigrationGroup, error) {
	migrations, _, err := am.createSQLMigrations(ctx, false)
	if err != nil {
		if err == errNothingToMigrate {
			return new(MigrationGroup), nil
		}
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	migrator := NewMigrator(am.db, migrations, am.migratorOpts...)
	if err := migrator.Init(ctx); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	group, err := migrator.Migrate(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}
	return group, nil
}

// CreateSQLMigration writes required changes to a new migration file.
// Use migrate.Migrator to apply the generated migrations.
func (am *AutoMigrator) CreateSQLMigrations(ctx context.Context) ([]*MigrationFile, error) {
	_, files, err := am.createSQLMigrations(ctx, false)
	if err == errNothingToMigrate {
		return files, nil
	}
	return files, err
}

// CreateTxSQLMigration writes required changes to a new migration file making sure they will be executed
// in a transaction when applied. Use migrate.Migrator to apply the generated migrations.
func (am *AutoMigrator) CreateTxSQLMigrations(ctx context.Context) ([]*MigrationFile, error) {
	_, files, err := am.createSQLMigrations(ctx, true)
	if err == errNothingToMigrate {
		return files, nil
	}
	return files, err
}

// errNothingToMigrate is a sentinel error which means the database is already in a desired state.
// Should not be returned to the user -- return a nil-error instead.
var errNothingToMigrate = errors.New("nothing to migrate")

func (am *AutoMigrator) createSQLMigrations(ctx context.Context, transactional bool) (*Migrations, []*MigrationFile, error) {
	changes, err := am.plan(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("create sql migrations: %w", err)
	}

	if changes.Len() == 0 {
		return nil, nil, errNothingToMigrate
	}

	name, _ := genMigrationName(am.schemaName + "_auto")
	migrations := NewMigrations(am.migrationsOpts...)
	migrations.Add(Migration{
		Name:    name,
		Up:      wrapGoMigrationFunc(changes.Up(am.dbMigrator)),
		Down:    wrapGoMigrationFunc(changes.Down(am.dbMigrator)),
		Comment: "Changes detected by bun.AutoMigrator",
	})

	// Append .tx.up.sql or .up.sql to migration name, depending if it should be transactional.
	fname := func(direction string) string {
		return name + map[bool]string{true: ".tx.", false: "."}[transactional] + direction + ".sql"
	}

	up, err := am.createSQL(ctx, migrations, fname("up"), changes, transactional)
	if err != nil {
		return nil, nil, fmt.Errorf("create sql migration up: %w", err)
	}

	down, err := am.createSQL(ctx, migrations, fname("down"), changes.GetReverse(), transactional)
	if err != nil {
		return nil, nil, fmt.Errorf("create sql migration down: %w", err)
	}
	return migrations, []*MigrationFile{up, down}, nil
}

func (am *AutoMigrator) createSQL(_ context.Context, migrations *Migrations, fname string, changes *changeset, transactional bool) (*MigrationFile, error) {
	var buf bytes.Buffer

	if transactional {
		buf.WriteString("SET statement_timeout = 0;")
	}

	if err := changes.WriteTo(&buf, am.dbMigrator); err != nil {
		return nil, err
	}
	content := buf.Bytes()

	fpath := filepath.Join(migrations.getDirectory(), fname)
	if err := os.WriteFile(fpath, content, 0o644); err != nil {
		return nil, err
	}

	mf := &MigrationFile{
		Name:    fname,
		Path:    fpath,
		Content: string(content),
	}
	return mf, nil
}

func (c *changeset) Len() int {
	return len(c.operations)
}

// Func creates a MigrationFunc that applies all operations all the changeset.
func (c *changeset) Func(m sqlschema.Migrator) MigrationFunc {
	return func(ctx context.Context, db *bun.DB) error {
		return c.apply(ctx, db, m)
	}
}

// GetReverse returns a new changeset with each operation in it "reversed" and in reverse order.
func (c *changeset) GetReverse() *changeset {
	var reverse changeset
	for i := len(c.operations) - 1; i >= 0; i-- {
		reverse.Add(c.operations[i].GetReverse())
	}
	return &reverse
}

// Up is syntactic sugar.
func (c *changeset) Up(m sqlschema.Migrator) MigrationFunc {
	return c.Func(m)
}

// Down is syntactic sugar.
func (c *changeset) Down(m sqlschema.Migrator) MigrationFunc {
	return c.GetReverse().Func(m)
}

// apply generates SQL for each operation and executes it.
func (c *changeset) apply(ctx context.Context, db *bun.DB, m sqlschema.Migrator) error {
	if len(c.operations) == 0 {
		return nil
	}

	for _, op := range c.operations {
		if _, skip := op.(*Unimplemented); skip {
			continue
		}

		b := internal.MakeQueryBytes()
		b, err := m.AppendSQL(b, op)
		if err != nil {
			return fmt.Errorf("apply changes: %w", err)
		}

		query := internal.String(b)
		if _, err = db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("apply changes: %w", err)
		}
	}
	return nil
}

func (c *changeset) WriteTo(w io.Writer, m sqlschema.Migrator) error {
	var err error

	b := internal.MakeQueryBytes()
	for _, op := range c.operations {
		if comment, isComment := op.(*Unimplemented); isComment {
			b = append(b, "/*\n"...)
			b = append(b, *comment...)
			b = append(b, "\n*/"...)
			continue
		}

		// Append each query separately, merge later.
		// Dialects assume that the []byte only holds
		// the contents of a single query and may be misled.
		queryBytes := internal.MakeQueryBytes()
		queryBytes, err = m.AppendSQL(queryBytes, op)
		if err != nil {
			return fmt.Errorf("write changeset: %w", err)
		}
		b = append(b, queryBytes...)
		b = append(b, ";\n"...)
	}
	if _, err := w.Write(b); err != nil {
		return fmt.Errorf("write changeset: %w", err)
	}
	return nil
}

func (c *changeset) ResolveDependencies() error {
	if len(c.operations) <= 1 {
		return nil
	}

	const (
		unvisited = iota
		current
		visited
	)

	status := make(map[Operation]int, len(c.operations))
	for _, op := range c.operations {
		status[op] = unvisited
	}

	var resolved []Operation
	var nextOp Operation
	var visit func(op Operation) error

	next := func() bool {
		for op, s := range status {
			if s == unvisited {
				nextOp = op
				return true
			}
		}
		return false
	}

	// visit iterates over c.operations until it finds all operations that depend on the current one
	// or runs into circular dependency, in which case it will return an error.
	visit = func(op Operation) error {
		switch status[op] {
		case visited:
			return nil
		case current:
			// TODO: add details (circle) to the error message
			return errors.New("detected circular dependency")
		}

		status[op] = current

		for _, another := range c.operations {
			if dop, hasDeps := another.(interface {
				DependsOn(Operation) bool
			}); another == op || !hasDeps || !dop.DependsOn(op) {
				continue
			}
			if err := visit(another); err != nil {
				return err
			}
		}

		status[op] = visited

		// Any dependent nodes would've already been added to the list by now, so we prepend.
		resolved = append([]Operation{op}, resolved...)
		return nil
	}

	for next() {
		if err := visit(nextOp); err != nil {
			return err
		}
	}

	c.operations = resolved
	return nil
}
