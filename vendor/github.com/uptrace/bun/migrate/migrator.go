package migrate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/feature"
)

const (
	defaultTable      = "bun_migrations"
	defaultLocksTable = "bun_migration_locks"
)

// MigratorOption configures a Migrator.
type MigratorOption func(m *Migrator)

// WithTableName overrides default migrations table name.
func WithTableName(table string) MigratorOption {
	return func(m *Migrator) {
		m.table = table
	}
}

// WithLocksTableName overrides default migration locks table name.
func WithLocksTableName(table string) MigratorOption {
	return func(m *Migrator) {
		m.locksTable = table
	}
}

// WithMarkAppliedOnSuccess sets the migrator to only mark migrations as applied/unapplied
// when their up/down is successful.
func WithMarkAppliedOnSuccess(enabled bool) MigratorOption {
	return func(m *Migrator) {
		m.markAppliedOnSuccess = enabled
	}
}

// WithUpsert enables upsert (ON CONFLICT / ON DUPLICATE KEY / MERGE) in MarkApplied.
// This is required when re-running already-applied migrations via RunMigration.
// Init automatically creates a unique index on the name column.
func WithUpsert(enabled bool) MigratorOption {
	return func(m *Migrator) {
		m.useUpsert = enabled
	}
}

// WithTemplateData sets data passed to SQL migration templates during rendering.
func WithTemplateData(data any) MigratorOption {
	return func(m *Migrator) {
		m.templateData = data
	}
}

// MigrationHook is a callback invoked before or after each migration runs.
type MigrationHook func(ctx context.Context, db bun.IConn, migration *Migration) error

// BeforeMigration registers a hook that runs before each migration.
func BeforeMigration(hook MigrationHook) MigratorOption {
	return func(m *Migrator) {
		m.beforeMigrationHook = hook
	}
}

// AfterMigration registers a hook that runs after each migration.
func AfterMigration(hook MigrationHook) MigratorOption {
	return func(m *Migrator) {
		m.afterMigrationHook = hook
	}
}

// Migrator manages the lifecycle of database migrations.
type Migrator struct {
	db         *bun.DB
	migrations *Migrations

	table                string
	locksTable           string
	markAppliedOnSuccess bool
	useUpsert            bool
	templateData         any

	beforeMigrationHook MigrationHook
	afterMigrationHook  MigrationHook
}

// NewMigrator creates a new Migrator for the given database and migrations.
func NewMigrator(db *bun.DB, migrations *Migrations, opts ...MigratorOption) *Migrator {
	m := &Migrator{
		db:         db,
		migrations: migrations,

		table:      defaultTable,
		locksTable: defaultLocksTable,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// DB returns the underlying bun.DB.
func (m *Migrator) DB() *bun.DB {
	return m.db
}

// MigrationsWithStatus returns migrations with status in ascending order.
func (m *Migrator) MigrationsWithStatus(ctx context.Context) (MigrationSlice, error) {
	sorted, _, err := m.migrationsWithStatus(ctx)
	return sorted, err
}

func (m *Migrator) migrationsWithStatus(ctx context.Context) (MigrationSlice, int64, error) {
	sorted := m.migrations.Sorted()

	applied, err := m.AppliedMigrations(ctx)
	if err != nil {
		return nil, 0, err
	}

	appliedMap := migrationMap(applied)
	for i := range sorted {
		m1 := &sorted[i]
		if m2, ok := appliedMap[m1.Name]; ok {
			m1.ID = m2.ID
			m1.GroupID = m2.GroupID
			m1.MigratedAt = m2.MigratedAt
		}
	}

	return sorted, applied.LastGroupID(), nil
}

// Init creates the migration tables if they do not already exist.
func (m *Migrator) Init(ctx context.Context) error {
	if _, err := m.db.NewCreateTable().
		Model((*Migration)(nil)).
		ModelTableExpr(m.table).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}
	if m.useUpsert {
		if _, err := m.db.NewCreateIndex().
			Unique().
			TableExpr(m.table).
			Index(m.table + "_name_unique").
			Column("name").
			IfNotExists().
			Exec(ctx); err != nil && !isIndexAlreadyExistsError(err) {
			return err
		}
	}
	if _, err := m.db.NewCreateTable().
		Model((*migrationLock)(nil)).
		ModelTableExpr(m.locksTable).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}
	return nil
}

// Reset drops and re-creates the migration tables.
func (m *Migrator) Reset(ctx context.Context) error {
	if _, err := m.db.NewDropTable().
		Model((*Migration)(nil)).
		ModelTableExpr(m.table).
		IfExists().
		Exec(ctx); err != nil {
		return err
	}
	if _, err := m.db.NewDropTable().
		Model((*migrationLock)(nil)).
		ModelTableExpr(m.locksTable).
		IfExists().
		Exec(ctx); err != nil {
		return err
	}
	return m.Init(ctx)
}

// Migrate runs unapplied migrations. If a migration fails, migrate immediately exits.
func (m *Migrator) Migrate(ctx context.Context, opts ...MigrationOption) (*MigrationGroup, error) {
	cfg := newMigrationConfig(opts)

	group := new(MigrationGroup)

	if err := m.validate(); err != nil {
		return group, err
	}

	migrations, lastGroupID, err := m.migrationsWithStatus(ctx)
	if err != nil {
		return group, err
	}
	migrations = migrations.Unapplied()
	if len(migrations) == 0 {
		return group, nil
	}
	group.ID = lastGroupID + 1

	for i := range migrations {
		migration := &migrations[i]
		migration.GroupID = group.ID

		if !m.markAppliedOnSuccess {
			if err := m.MarkApplied(ctx, migration); err != nil {
				return group, err
			}
		}

		group.Migrations = migrations[:i+1]

		if !cfg.nop && migration.Up != nil {
			if err := migration.Up(ctx, m, migration); err != nil {
				return group, fmt.Errorf("%s: up: %w", migration.Name, err)
			}
		}

		if m.markAppliedOnSuccess {
			if err := m.MarkApplied(ctx, migration); err != nil {
				return group, err
			}
		}
	}

	return group, nil
}

// RunMigration runs the up migration with the given name and marks it as applied.
// It runs the migration even if it is already marked as applied.
// The migration is added as a new applied record, creating a separate migration group.
func (m *Migrator) RunMigration(
	ctx context.Context, migrationName string, opts ...MigrationOption,
) error {
	cfg := newMigrationConfig(opts)

	if err := m.validate(); err != nil {
		return err
	}
	if migrationName == "" {
		return errors.New("migrate: migration name cannot be empty")
	}
	if !m.useUpsert {
		return errors.New("migrate: RunMigration requires WithUpsert(true)")
	}

	migrations, lastGroupID, err := m.migrationsWithStatus(ctx)
	if err != nil {
		return err
	}

	var migration *Migration
	for i := range migrations {
		if migrations[i].Name == migrationName {
			migration = &migrations[i]
			break
		}
	}
	if migration == nil {
		return fmt.Errorf("migrate: migration with name %q not found", migrationName)
	}
	if migration.Up == nil {
		return fmt.Errorf("migrate: migration %s does not have up migration", migration.Name)
	}
	if cfg.nop {
		return nil
	}

	migration.GroupID = lastGroupID + 1

	if !m.markAppliedOnSuccess {
		if err := m.MarkApplied(ctx, migration); err != nil {
			return err
		}
	}

	if err := migration.Up(ctx, m, migration); err != nil {
		return fmt.Errorf("%s: up: %w", migration.Name, err)
	}

	if m.markAppliedOnSuccess {
		if err := m.MarkApplied(ctx, migration); err != nil {
			return err
		}
	}

	return nil
}

// Rollback rolls back the last migration group.
func (m *Migrator) Rollback(ctx context.Context, opts ...MigrationOption) (*MigrationGroup, error) {
	cfg := newMigrationConfig(opts)

	lastGroup := new(MigrationGroup)

	if err := m.validate(); err != nil {
		return lastGroup, err
	}

	migrations, err := m.MigrationsWithStatus(ctx)
	if err != nil {
		return lastGroup, err
	}

	lastGroup = migrations.LastGroup()

	for i := len(lastGroup.Migrations) - 1; i >= 0; i-- {
		migration := &lastGroup.Migrations[i]

		if !m.markAppliedOnSuccess {
			if err := m.MarkUnapplied(ctx, migration); err != nil {
				return lastGroup, err
			}
		}

		if !cfg.nop && migration.Down != nil {
			if err := migration.Down(ctx, m, migration); err != nil {
				return lastGroup, fmt.Errorf("%s: down: %w", migration.Name, err)
			}
		}

		if m.markAppliedOnSuccess {
			if err := m.MarkUnapplied(ctx, migration); err != nil {
				return lastGroup, err
			}
		}
	}

	return lastGroup, nil
}

type goMigrationConfig struct {
	packageName string
	goTemplate  string
}

// GoMigrationOption configures Go migration file generation.
type GoMigrationOption func(cfg *goMigrationConfig)

// WithPackageName sets the Go package name used in generated migration files.
func WithPackageName(name string) GoMigrationOption {
	return func(cfg *goMigrationConfig) {
		cfg.packageName = name
	}
}

// WithGoTemplate sets the Go template string used for generated migration files.
func WithGoTemplate(template string) GoMigrationOption {
	return func(cfg *goMigrationConfig) {
		cfg.goTemplate = template
	}
}

// CreateGoMigration creates a Go migration file.
func (m *Migrator) CreateGoMigration(
	ctx context.Context, name string, opts ...GoMigrationOption,
) (*MigrationFile, error) {
	cfg := &goMigrationConfig{
		packageName: "migrations",
		goTemplate:  goTemplate,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	name, err := genMigrationName(name)
	if err != nil {
		return nil, err
	}

	fname := name + ".go"
	fpath := filepath.Join(m.migrations.getDirectory(), fname)
	content := fmt.Sprintf(cfg.goTemplate, cfg.packageName)

	if err := os.WriteFile(fpath, []byte(content), 0o644); err != nil {
		return nil, err
	}

	mf := &MigrationFile{
		Name:    fname,
		Path:    fpath,
		Content: content,
	}
	return mf, nil
}

// CreateTxSQLMigration creates transactional up and down SQL migration files.
func (m *Migrator) CreateTxSQLMigrations(ctx context.Context, name string) ([]*MigrationFile, error) {
	name, err := genMigrationName(name)
	if err != nil {
		return nil, err
	}

	up, err := m.createSQL(ctx, name+".tx.up.sql", true)
	if err != nil {
		return nil, err
	}

	down, err := m.createSQL(ctx, name+".tx.down.sql", true)
	if err != nil {
		return nil, err
	}

	return []*MigrationFile{up, down}, nil
}

// CreateSQLMigrations creates up and down SQL migration files.
func (m *Migrator) CreateSQLMigrations(ctx context.Context, name string) ([]*MigrationFile, error) {
	name, err := genMigrationName(name)
	if err != nil {
		return nil, err
	}

	up, err := m.createSQL(ctx, name+".up.sql", false)
	if err != nil {
		return nil, err
	}

	down, err := m.createSQL(ctx, name+".down.sql", false)
	if err != nil {
		return nil, err
	}

	return []*MigrationFile{up, down}, nil
}

func (m *Migrator) createSQL(_ context.Context, fname string, transactional bool) (*MigrationFile, error) {
	fpath := filepath.Join(m.migrations.getDirectory(), fname)

	template := sqlTemplate
	if transactional {
		template = transactionalSQLTemplate
	}

	if err := os.WriteFile(fpath, []byte(template), 0o644); err != nil {
		return nil, err
	}

	mf := &MigrationFile{
		Name:    fname,
		Path:    fpath,
		Content: goTemplate,
	}
	return mf, nil
}

var nameRE = regexp.MustCompile(`^[0-9a-z_\-]+$`)

func genMigrationName(name string) (string, error) {
	const timeFormat = "20060102150405"

	if name == "" {
		return "", errors.New("migrate: migration name can't be empty")
	}
	if !nameRE.MatchString(name) {
		return "", fmt.Errorf("migrate: invalid migration name: %q", name)
	}

	version := time.Now().UTC().Format(timeFormat)
	return fmt.Sprintf("%s_%s", version, name), nil
}

// MarkApplied marks the migration as applied (completed).
func (m *Migrator) MarkApplied(ctx context.Context, migration *Migration) error {
	q := m.db.NewInsert().Model(migration).
		ModelTableExpr(m.table)

	if m.useUpsert {
		switch {
		case m.db.HasFeature(feature.InsertOnConflict):
			q = q.On("CONFLICT (name) DO UPDATE").
				Set("group_id = EXCLUDED.group_id").
				Set("migrated_at = EXCLUDED.migrated_at")
		case m.db.HasFeature(feature.InsertOnDuplicateKey):
			q = q.On("DUPLICATE KEY UPDATE").
				Set("group_id = VALUES(group_id)").
				Set("migrated_at = VALUES(migrated_at)")
		case m.db.HasFeature(feature.Merge):
			source := MigrationSlice{*migration}
			_, err := m.db.NewMerge().
				Model(migration).
				ModelTableExpr("? AS migration", bun.Name(m.table)).
				With("_data", m.db.NewValues(&source)).
				Using("_data").
				On("migration.name = _data.name").
				WhenUpdate("MATCHED", func(q *bun.UpdateQuery) *bun.UpdateQuery {
					return q.
						Set("group_id = _data.group_id").
						Set("migrated_at = _data.migrated_at")
				}).
				WhenInsert("NOT MATCHED", func(q *bun.InsertQuery) *bun.InsertQuery {
					return q.
						Value("name", "_data.name").
						Value("group_id", "_data.group_id").
						Value("migrated_at", "_data.migrated_at")
				}).
				Exec(ctx)
			return err
		default:
			return errors.New("migrate: dialect does not support upsert or merge")
		}
	}

	_, err := q.Exec(ctx)
	return err
}

// MarkUnapplied marks the migration as unapplied (new).
func (m *Migrator) MarkUnapplied(ctx context.Context, migration *Migration) error {
	_, err := m.db.NewDelete().
		Model(migration).
		ModelTableExpr(m.table).
		Where("id = ?", migration.ID).
		Exec(ctx)
	return err
}

// TruncateTable removes all rows from the migrations table.
func (m *Migrator) TruncateTable(ctx context.Context) error {
	_, err := m.db.NewTruncateTable().
		Model((*Migration)(nil)).
		ModelTableExpr(m.table).
		Exec(ctx)
	return err
}

// MissingMigrations returns applied migrations that can no longer be found.
func (m *Migrator) MissingMigrations(ctx context.Context) (MigrationSlice, error) {
	applied, err := m.AppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	existing := migrationMap(m.migrations.ms)
	for i := len(applied) - 1; i >= 0; i-- {
		m := &applied[i]
		if _, ok := existing[m.Name]; ok {
			applied = append(applied[:i], applied[i+1:]...)
		}
	}

	return applied, nil
}

// AppliedMigrations returns applied (applied) migrations in descending order.
func (m *Migrator) AppliedMigrations(ctx context.Context) (MigrationSlice, error) {
	var ms MigrationSlice
	if err := m.db.NewSelect().
		ColumnExpr("*").
		Model(&ms).
		ModelTableExpr(m.table).
		Scan(ctx); err != nil {
		return nil, err
	}
	return ms, nil
}

func (m *Migrator) formattedTableName(db *bun.DB) string {
	return db.QueryGen().FormatQuery(m.table)
}

func (m *Migrator) validate() error {
	if len(m.migrations.ms) == 0 {
		return errors.New("migrate: there are no migrations")
	}
	return nil
}

func (m *Migrator) exec(
	ctx context.Context, db bun.IConn, migration *Migration, queries []string,
) error {
	if m.beforeMigrationHook != nil {
		if err := m.beforeMigrationHook(ctx, db, migration); err != nil {
			return err
		}
	}

	for _, query := range queries {
		if strings.TrimSpace(query) == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, query); err != nil {
			return err
		}
	}

	if m.afterMigrationHook != nil {
		if err := m.afterMigrationHook(ctx, db, migration); err != nil {
			return err
		}
	}

	return nil
}

//------------------------------------------------------------------------------

type migrationLock struct {
	ID        int64  `bun:",pk,autoincrement"`
	TableName string `bun:",unique"`
}

// Lock acquires an advisory lock on the migration table to prevent concurrent migrations.
func (m *Migrator) Lock(ctx context.Context) error {
	lock := &migrationLock{
		TableName: m.formattedTableName(m.db),
	}
	if _, err := m.db.NewInsert().
		Model(lock).
		ModelTableExpr(m.locksTable).
		Exec(ctx); err != nil {
		return fmt.Errorf("migrate: migrations table is already locked (%w)", err)
	}
	return nil
}

// Unlock releases the advisory lock on the migration table.
func (m *Migrator) Unlock(ctx context.Context) error {
	tableName := m.formattedTableName(m.db)
	_, err := m.db.NewDelete().
		Model((*migrationLock)(nil)).
		ModelTableExpr(m.locksTable).
		Where("? = ?", bun.Ident("table_name"), tableName).
		Exec(ctx)
	return err
}

// isIndexAlreadyExistsError checks whether err indicates the index already exists.
// This is needed for dialects that do not support CREATE INDEX IF NOT EXISTS
// (e.g. MySQL, MSSQL), where a duplicate-index error is expected on repeated Init calls.
func isIndexAlreadyExistsError(err error) bool {
	s := strings.ToLower(err.Error())
	// MySQL:  Error 1061: Duplicate key name '...'
	// MSSQL:  The index '...' already exists on table '...'
	// Oracle: ORA-00955: name is already used by an existing object
	return strings.Contains(s, "duplicate key name") || strings.Contains(s, "already exist")
}

func migrationMap(ms MigrationSlice) map[string]*Migration {
	mp := make(map[string]*Migration)
	for i := range ms {
		m := &ms[i]
		mp[m.Name] = m
	}
	return mp
}
