// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package sql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"
	_ "unsafe"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	"github.com/apache/iceberg-go/catalog/internal"
	"github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/metrics"
	"github.com/apache/iceberg-go/table"
	"github.com/apache/iceberg-go/view"
	mysql "github.com/go-sql-driver/mysql"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/dialect/mssqldialect"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/oracledialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/schema"
)

type SupportedDialect string

const (
	Postgres SupportedDialect = "postgres"
	MySQL    SupportedDialect = "mysql"
	SQLite   SupportedDialect = "sqlite"
	MSSQL    SupportedDialect = "mssql"
	Oracle   SupportedDialect = "oracle"
)

const encodedNamespacePrefix = "__iceberg_namespace_v1__:"

func namespaceToString(namespace table.Identifier) string {
	legacy := strings.Join(namespace, ".")
	if !strings.Contains(legacy, encodedNamespacePrefix) && !slices.ContainsFunc(namespace, func(part string) bool {
		return strings.Contains(part, ".")
	}) {
		return legacy
	}

	encoded, err := json.Marshal(namespace)
	if err != nil {
		panic(err) // table.Identifier contains only strings
	}

	return encodedNamespacePrefix + string(encoded)
}

func namespaceFromString(namespace string) (table.Identifier, error) {
	if !strings.HasPrefix(namespace, encodedNamespacePrefix) {
		return strings.Split(namespace, "."), nil
	}

	var ident table.Identifier
	if err := json.Unmarshal([]byte(strings.TrimPrefix(namespace, encodedNamespacePrefix)), &ident); err != nil ||
		len(ident) == 0 || namespaceToString(ident) != namespace {
		// Before encoded namespaces existed, the prefix was valid ordinary
		// namespace text. Only canonical, non-empty encodings claim the marker.
		return strings.Split(namespace, "."), nil
	}

	return ident, nil
}

func namespaceStorageKeys(namespace table.Identifier) []string {
	primary := namespaceToString(namespace)
	legacy := strings.Join(namespace, ".")
	if primary == legacy || slices.ContainsFunc(namespace, func(part string) bool {
		return strings.Contains(part, ".")
	}) {
		return []string{primary}
	}

	// Only marker collisions get a legacy fallback. Dotted components must
	// never fall back because their legacy key belongs to a different namespace.
	return []string{primary, legacy}
}

func namespaceDescendantPrefixes(namespace table.Identifier) []string {
	legacy := strings.Join(namespace, ".")
	prefixes := make([]string, 0, 2)
	if !slices.ContainsFunc(namespace, func(part string) bool { return strings.Contains(part, ".") }) {
		prefixes = append(prefixes, legacy+".")
	}

	encoded, err := json.Marshal(namespace)
	if err != nil {
		panic(err) // table.Identifier contains only strings
	}
	prefixes = append(prefixes, encodedNamespacePrefix+string(encoded[:len(encoded)-1])+",")

	return prefixes
}

func namespaceLikePrefix(prefix string) string {
	return strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(prefix) + "%"
}

func whereNamespaceParent(query *bun.SelectQuery, column string, parent table.Identifier) *bun.SelectQuery {
	return query.WhereGroup(" AND ", func(query *bun.SelectQuery) *bun.SelectQuery {
		keys := namespaceStorageKeys(parent)
		query = query.Where("? = ?", bun.Ident(column), keys[0])
		for _, key := range keys[1:] {
			query = query.WhereOr("? = ?", bun.Ident(column), key)
		}
		for _, prefix := range namespaceDescendantPrefixes(parent) {
			query = query.WhereOr("? LIKE ? ESCAPE '!'", bun.Ident(column), namespaceLikePrefix(prefix))
		}

		return query
	})
}

const (
	DialectKey           = "sql.dialect"
	DriverKey            = "sql.driver"
	initCatalogTablesKey = "init_catalog_tables"

	// SchemaVersionKey opts in to migrating a legacy V0 catalog (one that lacks
	// the iceberg_type column) up to the V1 schema.
	//
	// When set to SchemaVersionV1 the catalog issues the one-way ALTER TABLE
	// that adds the iceberg_type column; otherwise it stays on V0 and emits
	// column-free queries that never reference iceberg_type.
	SchemaVersionKey = "jdbc.schema-version"
)

const (
	SchemaVersionV1 = "V1"
)

// schemaVer tracks which physical layout of the iceberg_tables table the catalog is operating against.
//
// schemaV1 has the iceberg_type column. schemaV0 is the legacy JDBC default that lacks the column entirely.
// Queries against a V0 catalog must never reference iceberg_type.
//
// The zero value is schemaV0
type schemaVer int

const (
	schemaV0 schemaVer = iota
	schemaV1
)

var v0WarnOnce sync.Once

var errViewsUnsupportedOnV0 = fmt.Errorf(
	"views are not supported on a legacy V0 catalog schema (no iceberg_type column); "+
		"migrate to V1 by setting %q=%q", SchemaVersionKey, SchemaVersionV1)

const (
	TableType = "TABLE"
	ViewType  = "VIEW"
)

func init() {
	catalog.Register("sql", catalog.RegistrarFunc(func(ctx context.Context, name string, p iceberg.Properties) (c catalog.Catalog, err error) {
		driver, ok := p[DriverKey]
		if !ok {
			return nil, errors.New("must provide driver to pass to sql.Open")
		}

		dialect := strings.ToLower(p[DialectKey])
		if dialect == "" {
			return nil, errors.New("must provide sql dialect to use")
		}

		uri := strings.TrimPrefix(p.Get("uri", ""), "sql://")
		sqldb, err := sql.Open(driver, uri)
		if err != nil {
			return nil, err
		}

		// Here, name is the loader-supplied catalog name and is persisted as the
		// catalog_name partition key on every row in the iceberg_tables and iceberg_namespace_properties.
		cat, err := NewCatalog(name, sqldb, SupportedDialect(dialect), p)
		if err != nil {
			_ = sqldb.Close()

			return nil, err
		}

		// This registrar opened sqldb itself, so nobody else holds the handle;
		// Close must release its connection pool. NewCatalog leaves ownsDB false
		// for caller-supplied handles, which stay owned by the caller.
		cat.ownsDB = true

		return cat, nil
	}))
}

var _ catalog.Catalog = (*Catalog)(nil)

const namespaceExistsProperty = "exists"

var (
	minimalNamespaceProps = iceberg.Properties{namespaceExistsProperty: "true"}

	dialects  = map[SupportedDialect]schema.Dialect{}
	dialectMx sync.Mutex
)

func isReservedNamespaceProperty(key string) bool {
	return key == namespaceExistsProperty
}

func createDialect(d SupportedDialect) (schema.Dialect, error) {
	switch d {
	case Postgres:
		return pgdialect.New(), nil
	case MySQL:
		return mysqldialect.New(), nil
	case SQLite:
		return sqlitedialect.New(), nil
	case MSSQL:
		return mssqldialect.New(), nil
	case Oracle:
		return oracledialect.New(), nil
	default:
		return nil, fmt.Errorf("unsupported sql dialect: %q", d)
	}
}

func getDialect(d SupportedDialect) (schema.Dialect, error) {
	dialectMx.Lock()
	defer dialectMx.Unlock()
	ret, ok := dialects[d]
	if !ok {
		var err error
		ret, err = createDialect(d)
		if err != nil {
			return nil, err
		}
		dialects[d] = ret
	}

	return ret, nil
}

type sqlIcebergTable struct {
	bun.BaseModel `bun:"table:iceberg_tables"`

	CatalogName              string         `bun:",pk"`
	TableNamespace           string         `bun:",pk"`
	TableName                string         `bun:",pk"`
	IcebergType              sql.NullString // TableType, ViewType, or NULL for legacy V0 rows
	MetadataLocation         sql.NullString
	PreviousMetadataLocation sql.NullString
}

type sqlIcebergNamespaceProps struct {
	bun.BaseModel `bun:"table:iceberg_namespace_properties"`

	CatalogName   string `bun:",pk"`
	Namespace     string `bun:",pk"`
	PropertyKey   string `bun:",pk"`
	PropertyValue sql.NullString
}

func withReadTx[R any](ctx context.Context, db *bun.DB, fn func(context.Context, bun.Tx) (R, error)) (result R, err error) {
	err = db.RunInTx(ctx, &sql.TxOptions{ReadOnly: true}, func(ctx context.Context, tx bun.Tx) error {
		result, err = fn(ctx, tx)

		return err
	})

	return result, err
}

func withWriteTx(ctx context.Context, db *bun.DB, fn func(context.Context, bun.Tx) error) error {
	return db.RunInTx(ctx, &sql.TxOptions{Isolation: sql.LevelDefault}, func(ctx context.Context, tx bun.Tx) error {
		return fn(ctx, tx)
	})
}

const (
	serializableWriteMaxAttempts = 3
	serializableWriteRetryDelay  = 10 * time.Millisecond
)

func withSerializableWriteTx(ctx context.Context, db *bun.DB, fn func(context.Context, bun.Tx) error) error {
	return retrySerializableWriteTx(ctx, func() error {
		return db.RunInTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable}, func(ctx context.Context, tx bun.Tx) error {
			return fn(ctx, tx)
		})
	})
}

func retrySerializableWriteTx(ctx context.Context, run func() error) error {
	var err error
	for attempt := range serializableWriteMaxAttempts {
		err = run()
		if err == nil || !isRetryableSerializableError(err) {
			return err
		}
		if attempt == serializableWriteMaxAttempts-1 {
			break
		}
		delay := serializableWriteRetryDelay << attempt
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()

			return context.Cause(ctx)
		case <-timer.C:
		}
	}

	return err
}

func isRetryableSerializableError(err error) bool {
	var stateErr interface{ SQLState() string }
	if errors.As(err, &stateErr) {
		switch stateErr.SQLState() {
		case "40001", "40P01":
			return true
		}
	}

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1205, 1213: // lock wait timeout, deadlock
			return true
		}
	}

	msg := err.Error()

	// sqliteshim selects different SQLite drivers by build target. Their typed
	// error APIs are incompatible, but both expose these stable lock messages.
	return strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "database table is locked") ||
		strings.Contains(msg, "ORA-08177") ||
		strings.Contains(msg, "ORA-00060")
}

var _ catalog.PurgeableTable = (*Catalog)(nil)

type Catalog struct {
	db            *bun.DB
	name          string
	props         iceberg.Properties
	schemaVersion schemaVer
	// reporter builds and caches the catalog's metrics reporter once, so it is
	// constructed per-catalog rather than per table load. Released by Close.
	reporter metrics.CachedReporter
	// ownsDB reports whether this catalog opened its own *sql.DB (the registrar
	// path) and must therefore close it in Close. It stays false for handles
	// supplied to NewCatalog, which remain owned by the caller.
	ownsDB bool
}

// isV0 reports whether the catalog is operating against a legacy V0 schema that
// lacks the iceberg_type column. Callers use this to emit column-free queries.
func (c *Catalog) isV0() bool { return c.schemaVersion == schemaV0 }

// NewCatalog creates a new sql-based catalog using the provided sql.DB handle to perform any queries.
//
// The dialect parameter determines the SQL dialect to use for query generation and must be one of the
// supported dialects, i.e. one of the exported SupportedDialect values. The separation here allows for
// the use of different drivers/databases provided they support the chosen sql dialect (e.g. if a particular
// database supports the MySQL dialect, then the database can still be used with this catalog even though
// it's not explicitly implemented).
//
// If the "init_catalog_tables" property is set to "true", then creating the catalog will also attempt to
// to verify whether the necessary tables (iceberg_tables and iceberg_namespace_properties) exist, creating
// them if they do not already exist.
//
// The environment variable ICEBERG_SQL_DEBUG can be set to automatically log the sql queries to the terminal:
// - ICEBERG_SQL_DEBUG=1 logs only failed queries
// - ICEBERG_SQL_DEBUG=2 logs all queries
//
// All interactions with the db are performed within transactions to ensure atomicity and transactional isolation
// of catalog changes.
//
// SQLite callers that allow concurrent writers should configure WAL and a busy timeout in the driver DSN so the
// settings apply to every pooled connection. NewCatalog does not mutate connection settings on the caller-owned db.
func NewCatalog(name string, db *sql.DB, dialect SupportedDialect, props iceberg.Properties) (*Catalog, error) {
	d, err := getDialect(dialect)
	if err != nil {
		return nil, err
	}

	cat := &Catalog{db: bun.NewDB(db, d), name: name, props: props}

	cat.db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithEnabled(false),
		// ICEBERG_SQL_DEBUG=1 logs only failed queries
		// ICEBERG_SQL_DEBUG=2 log all queries
		bundebug.FromEnv("ICEBERG_SQL_DEBUG")))

	ctx := context.Background()

	if cat.props.GetBool(initCatalogTablesKey, true) {
		if err := cat.CreateSQLTables(ctx); err != nil {
			return nil, err
		}
	}

	if err := cat.initSchemaVersion(ctx); err != nil {
		return nil, err
	}

	return cat, nil
}

func (c *Catalog) Name() string { return c.name }

func (c *Catalog) CatalogType() catalog.Type {
	return catalog.SQL
}

// Close releases the catalog's metrics reporter. When the catalog opened its own
// database handle (the registry [catalog.Load] path), Close also closes that
// handle's connection pool; a handle supplied to [NewCatalog] stays owned by the
// caller and is left open. Callers holding a [catalog.Catalog] can reach this via
// a [catalog.Closer] type assertion.
func (c *Catalog) Close() error {
	err := c.reporter.Close()
	if c.ownsDB {
		if dbErr := c.db.Close(); dbErr != nil {
			if err == nil {
				err = dbErr
			} else {
				err = errors.Join(err, dbErr)
			}
		}
	}

	return err
}

var _ catalog.Closer = (*Catalog)(nil)

func (c *Catalog) CreateSQLTables(ctx context.Context) error {
	_, err := c.db.NewCreateTable().Model((*sqlIcebergTable)(nil)).
		IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	_, err = c.db.NewCreateTable().Model((*sqlIcebergNamespaceProps)(nil)).
		IfNotExists().Exec(ctx)

	return err
}

func (c *Catalog) DropSQLTables(ctx context.Context) error {
	_, err := c.db.NewDropTable().Model((*sqlIcebergTable)(nil)).
		IfExists().Exec(ctx)
	if err != nil {
		return err
	}

	_, err = c.db.NewDropTable().Model((*sqlIcebergNamespaceProps)(nil)).
		IfExists().Exec(ctx)

	return err
}

func (c *Catalog) initSchemaVersion(ctx context.Context) error {
	hasCol, err := c.icebergTypeColumnExists(ctx)
	if err != nil {
		return fmt.Errorf("detecting catalog schema version: %w", err)
	}
	if hasCol {
		c.schemaVersion = schemaV1

		return nil
	}

	tableExists, err := c.icebergTablesExists(ctx)
	if err != nil {
		return fmt.Errorf("detecting catalog schema version: %w", err)
	}
	if !tableExists {
		c.schemaVersion = schemaV1

		return nil
	}

	// legacy V0 table: only migrate when explicitly opted in.
	//
	// The comparison is intentionally case-insensitive: matching Java's JdbcCatalog
	if !strings.EqualFold(c.props.Get(SchemaVersionKey, ""), SchemaVersionV1) {
		v0WarnOnce.Do(func() {
			log.Printf("WARNING: iceberg_tables is on the legacy V0 schema (no iceberg_type column); "+
				"set %q=%q to migrate to V1. Operating in V0-compatible mode (column-free queries).",
				SchemaVersionKey, SchemaVersionV1)
		})
		c.schemaVersion = schemaV0

		return nil
	}

	return c.migrateV0ToV1(ctx)
}

func (c *Catalog) migrateV0ToV1(ctx context.Context) error {
	ddl, err := addIcebergTypeColumnDDL(c.db.Dialect().Name())
	if err != nil {
		return err
	}

	if _, err := c.db.ExecContext(ctx, ddl); err != nil {
		has, reprobeErr := c.icebergTypeColumnExists(ctx)
		if has {
			c.schemaVersion = schemaV1

			return nil
		}
		if reprobeErr != nil {
			return fmt.Errorf("migration of V0 schema failed: %w (re-probe also failed: %v)", err, reprobeErr)
		}

		return fmt.Errorf("migration of V0 schema failed: %w", err)
	}

	c.schemaVersion = schemaV1

	return nil
}

func addIcebergTypeColumnDDL(name dialect.Name) (string, error) {
	switch name {
	case dialect.PG, dialect.SQLite:
		return "ALTER TABLE iceberg_tables ADD COLUMN iceberg_type VARCHAR", nil
	case dialect.MySQL:
		return "ALTER TABLE iceberg_tables ADD COLUMN iceberg_type VARCHAR(255)", nil
	case dialect.MSSQL:
		return "ALTER TABLE iceberg_tables ADD iceberg_type VARCHAR(255)", nil
	case dialect.Oracle:
		return "ALTER TABLE iceberg_tables ADD (iceberg_type VARCHAR2(255))", nil
	default:
		return "", fmt.Errorf("unsupported dialect for V0 migration: %s", name)
	}
}

func icebergTypeColumnExistsQuery(name dialect.Name) (string, error) {
	switch name {
	case dialect.PG:
		return `SELECT 1 FROM information_schema.columns
		         WHERE table_schema = current_schema()
		           AND table_name = 'iceberg_tables'
		           AND column_name = 'iceberg_type' LIMIT 1`, nil
	case dialect.MySQL:
		return `SELECT 1 FROM information_schema.columns
		         WHERE table_schema = DATABASE()
		           AND table_name = 'iceberg_tables'
		           AND column_name = 'iceberg_type' LIMIT 1`, nil
	case dialect.MSSQL:
		return `SELECT TOP 1 1 FROM information_schema.columns
		         WHERE table_schema = SCHEMA_NAME()
		           AND table_name = 'iceberg_tables'
		           AND column_name = 'iceberg_type'`, nil
	case dialect.SQLite:
		return `SELECT 1 FROM pragma_table_info('iceberg_tables')
		         WHERE name = 'iceberg_type' LIMIT 1`, nil
	case dialect.Oracle:
		return `SELECT 1 FROM user_tab_columns
		         WHERE UPPER(table_name) = 'ICEBERG_TABLES'
		           AND UPPER(column_name) = 'ICEBERG_TYPE'
		           AND ROWNUM = 1`, nil
	default:
		return "", fmt.Errorf("unsupported dialect for V0 migration: %s", name)
	}
}

func icebergTablesExistsQuery(name dialect.Name) (string, error) {
	switch name {
	case dialect.PG:
		return `SELECT 1 FROM information_schema.tables
		         WHERE table_schema = current_schema()
		           AND table_name = 'iceberg_tables' LIMIT 1`, nil
	case dialect.MySQL:
		return `SELECT 1 FROM information_schema.tables
		         WHERE table_schema = DATABASE()
		           AND table_name = 'iceberg_tables' LIMIT 1`, nil
	case dialect.MSSQL:
		return `SELECT TOP 1 1 FROM information_schema.tables
		         WHERE table_schema = SCHEMA_NAME()
		           AND table_name = 'iceberg_tables'`, nil
	case dialect.SQLite:
		return `SELECT 1 FROM sqlite_master
		         WHERE type = 'table' AND name = 'iceberg_tables' LIMIT 1`, nil
	case dialect.Oracle:
		return `SELECT 1 FROM user_tables
		         WHERE UPPER(table_name) = 'ICEBERG_TABLES' AND ROWNUM = 1`, nil
	default:
		return "", fmt.Errorf("unsupported dialect for V0 migration: %s", name)
	}
}

func (c *Catalog) icebergTypeColumnExists(ctx context.Context) (bool, error) {
	query, err := icebergTypeColumnExistsQuery(c.db.Dialect().Name())
	if err != nil {
		return false, err
	}

	return c.probeExists(ctx, query, "probing for iceberg_type column")
}

func (c *Catalog) icebergTablesExists(ctx context.Context) (bool, error) {
	query, err := icebergTablesExistsQuery(c.db.Dialect().Name())
	if err != nil {
		return false, err
	}

	return c.probeExists(ctx, query, "probing for iceberg_tables table")
}

func (c *Catalog) probeExists(ctx context.Context, query, what string) (bool, error) {
	return withReadTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) (bool, error) {
		var exists int
		if err := tx.QueryRowContext(ctx, query).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return false, nil
			}

			return false, fmt.Errorf("%s: %w", what, err)
		}

		return true, nil
	})
}

func (c *Catalog) namespaceExists(ctx context.Context, ns string) (bool, error) {
	return withReadTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) (bool, error) {
		return c.namespaceExistsInTx(ctx, tx, ns)
	})
}

func (c *Catalog) namespaceExistsInTx(ctx context.Context, tx bun.Tx, ns string) (bool, error) {
	// ColumnExpr("1") keeps the existence probe from selecting iceberg_type,
	// which would not exist on a V0 schema.
	exists, err := tx.NewSelect().Model((*sqlIcebergTable)(nil)).
		ColumnExpr("1").
		Where("catalog_name = ?", c.name).
		Where("table_namespace = ?", ns).
		Limit(1).Exists(ctx)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	return tx.NewSelect().Model((*sqlIcebergNamespaceProps)(nil)).
		ColumnExpr("1").
		Where("catalog_name = ?", c.name).Where("namespace = ?", ns).
		Limit(1).Exists(ctx)
}

func (c *Catalog) resolveNamespaceKey(ctx context.Context, namespace table.Identifier) (string, bool, error) {
	keys := namespaceStorageKeys(namespace)
	for _, key := range keys {
		exists, err := c.namespaceExists(ctx, key)
		if err != nil {
			return "", false, err
		}
		if exists {
			return key, true, nil
		}
	}

	return keys[0], false, nil
}

func (c *Catalog) resolveNamespaceKeyInTx(ctx context.Context, tx bun.Tx, namespace table.Identifier) (string, bool, error) {
	keys := namespaceStorageKeys(namespace)
	for _, key := range keys {
		exists, err := c.namespaceExistsInTx(ctx, tx, key)
		if err != nil {
			return "", false, err
		}
		if exists {
			return key, true, nil
		}
	}

	return keys[0], false, nil
}

func (c *Catalog) namespaceKey(ctx context.Context, namespace table.Identifier) (string, error) {
	keys := namespaceStorageKeys(namespace)
	if len(keys) == 1 {
		return keys[0], nil
	}

	for _, key := range keys {
		exists, err := c.namespaceExists(ctx, key)
		if err != nil {
			return "", err
		}
		if exists {
			return key, nil
		}
	}

	return keys[0], nil
}

func checkValidNamespace(ident table.Identifier) error {
	if len(ident) < 1 {
		return fmt.Errorf("%w: empty namespace identifier", catalog.ErrNoSuchNamespace)
	}

	return nil
}

func removeUncommittedMetadata(ctx context.Context, metadataLocation string, loadFS table.FSysF) error {
	fs, err := loadFS(context.WithoutCancel(ctx))
	if err != nil {
		return fmt.Errorf("failed to load filesystem while removing uncommitted metadata %s: %w", metadataLocation, err)
	}

	if err := fs.Remove(metadataLocation); err != nil {
		return fmt.Errorf("failed to remove uncommitted metadata %s: %w", metadataLocation, err)
	}

	return nil
}

func (c *Catalog) checkIdentifierAvailable(
	ctx context.Context, tx bun.Tx, namespace, name string, identifier table.Identifier,
) error {
	entry := &sqlIcebergTable{
		CatalogName:    c.name,
		TableNamespace: namespace,
		TableName:      name,
	}
	query := tx.NewSelect().Model(entry).WherePK()
	if c.isV0() {
		query = query.ExcludeColumn("iceberg_type")
	}

	err := query.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("error checking identifier availability: %w", err)
	}

	identifierName := strings.Join(identifier, ".")
	if !c.isV0() && entry.IcebergType.Valid && entry.IcebergType.String == ViewType {
		return fmt.Errorf("%w: %s", catalog.ErrViewAlreadyExists, identifierName)
	}

	return fmt.Errorf("%w: %s", catalog.ErrTableAlreadyExists, identifierName)
}

func (c *Catalog) checkIdentifierAvailableInCatalog(
	ctx context.Context, namespace, name string, identifier table.Identifier,
) error {
	return c.db.RunInTx(ctx, &sql.TxOptions{ReadOnly: true}, func(ctx context.Context, tx bun.Tx) error {
		return c.checkIdentifierAvailable(ctx, tx, namespace, name, identifier)
	})
}

func (c *Catalog) CreateTable(ctx context.Context, ident table.Identifier, sc *iceberg.Schema, opts ...catalog.CreateTableOpt) (*table.Table, error) {
	// Resolve the reporter before any mutation: this method ends in LoadTable,
	// which is where the reporter is otherwise first built, so a bad
	// metrics-reporter-impl would only surface after the row was inserted —
	// turning a successful create into a reported failure whose retry hits
	// ErrTableAlreadyExists. CachedReporter caches this for that LoadTable.
	if _, err := c.reporter.Get(c.props); err != nil {
		return nil, fmt.Errorf("failed to initialize metrics reporter: %w", err)
	}

	nsIdent := catalog.NamespaceFromIdent(ident)
	tblIdent := catalog.TableNameFromIdent(ident)
	ns, exists, err := c.resolveNamespaceKey(ctx, nsIdent)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("%w: %s", catalog.ErrNoSuchNamespace, ns)
	}

	staged, err := internal.CreateStagedTableWithNamespaceKey(
		ctx, c.props, c.LoadNamespaceProperties, ident, ns, sc, opts...)
	if err != nil {
		return nil, err
	}

	if err := internal.WriteMetadata(ctx, staged.Table); err != nil {
		return nil, err
	}

	err = withSerializableWriteTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) error {
		exists, err := c.namespaceExistsInTx(ctx, tx, ns)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w: %s", catalog.ErrNoSuchNamespace, ns)
		}

		if err := c.checkIdentifierAvailable(ctx, tx, ns, tblIdent, ident); err != nil {
			return err
		}

		ins := tx.NewInsert().Model(&sqlIcebergTable{
			CatalogName:      c.name,
			TableNamespace:   ns,
			TableName:        tblIdent,
			MetadataLocation: sql.NullString{String: staged.MetadataLocation(), Valid: true},
			IcebergType:      sql.NullString{String: TableType, Valid: true},
		})
		if c.isV0() {
			ins = ins.ExcludeColumn("iceberg_type")
		}
		if _, err := ins.Exec(ctx); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}

		return nil
	})
	if err != nil {
		if !errors.Is(err, catalog.ErrTableAlreadyExists) && !errors.Is(err, catalog.ErrViewAlreadyExists) {
			if collisionErr := c.checkIdentifierAvailableInCatalog(ctx, ns, tblIdent, ident); errors.Is(collisionErr, catalog.ErrTableAlreadyExists) ||
				errors.Is(collisionErr, catalog.ErrViewAlreadyExists) {
				err = collisionErr
			}
		}
		if cleanupErr := removeUncommittedMetadata(ctx, staged.MetadataLocation(), staged.FS); cleanupErr != nil {
			err = errors.Join(err, cleanupErr)
		}

		return nil, err
	}

	return c.LoadTable(ctx, ident)
}

func (c *Catalog) CommitTable(ctx context.Context, ident table.Identifier, reqs []table.Requirement, updates []table.Update) (table.Metadata, string, error) {
	if err := catalog.ValidateTableIdentifier(ident); err != nil {
		return nil, "", err
	}

	ns := catalog.NamespaceFromIdent(ident)
	tblName := catalog.TableNameFromIdent(ident)
	nsKey, err := c.namespaceKey(ctx, ns)
	if err != nil {
		return nil, "", err
	}

	current, err := c.LoadTable(ctx, ident)
	if err != nil && !errors.Is(err, catalog.ErrNoSuchTable) {
		return nil, "", err
	}

	staged, err := internal.UpdateAndStageTable(ctx, c.props, current, ident, reqs, updates, c)
	if err != nil {
		return nil, "", err
	}

	if current != nil && staged.Metadata().Equals(current.Metadata()) {
		// no changes, do nothing
		return current.Metadata(), current.MetadataLocation(), nil
	}

	if err := internal.WriteMetadata(ctx, staged.Table); err != nil {
		return nil, "", err
	}

	runWriteTx := withWriteTx
	if current == nil {
		runWriteTx = withSerializableWriteTx
	}
	err = runWriteTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) error {
		if current != nil {
			// On V1, the model-driven UPDATE emits every non-PK field, writing
			// iceberg_type=TABLE; this also heals legacy NULLs on commit, and the
			// predicate treats NULL and TABLE identically. On V0 the column does
			// not exist, so it is excluded from both the SET list and the WHERE.
			upd := tx.NewUpdate().Model(&sqlIcebergTable{
				CatalogName:              c.name,
				TableNamespace:           nsKey,
				TableName:                tblName,
				IcebergType:              sql.NullString{String: TableType, Valid: true},
				MetadataLocation:         sql.NullString{Valid: true, String: staged.MetadataLocation()},
				PreviousMetadataLocation: sql.NullString{Valid: true, String: current.MetadataLocation()},
			}).WherePK().Where("metadata_location = ?", current.MetadataLocation())
			if c.isV0() {
				upd = upd.ExcludeColumn("iceberg_type")
			} else {
				upd = upd.Where("(iceberg_type = ? OR iceberg_type IS NULL)", TableType)
			}
			res, err := upd.Exec(ctx)
			if err != nil {
				return fmt.Errorf("error updating table information: %w", err)
			}

			n, err := res.RowsAffected()
			if err != nil {
				return fmt.Errorf("error updating table information: %w", err)
			}

			if n == 0 {
				// Wrap with table.ErrCommitFailed so the retry loop in
				// Table.doCommit treats this as a retryable conflict
				// (matches catalog/rest's HTTP 409 mapping).
				return fmt.Errorf("%w: table %s.%s metadata-location moved underneath us",
					table.ErrCommitFailed, strings.Join(ns, "."), tblName)
			}

			return nil
		}

		exists, err := c.namespaceExistsInTx(ctx, tx, nsKey)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w: %s", catalog.ErrNoSuchNamespace, nsKey)
		}

		ins := tx.NewInsert().Model(&sqlIcebergTable{
			CatalogName:      c.name,
			TableNamespace:   nsKey,
			TableName:        tblName,
			IcebergType:      sql.NullString{String: TableType, Valid: true},
			MetadataLocation: sql.NullString{Valid: true, String: staged.MetadataLocation()},
		})
		if c.isV0() {
			ins = ins.ExcludeColumn("iceberg_type")
		}
		if _, err := ins.Exec(ctx); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, "", err
	}

	return staged.Metadata(), staged.MetadataLocation(), nil
}

func (c *Catalog) LoadTable(ctx context.Context, identifier table.Identifier) (*table.Table, error) {
	if err := catalog.ValidateTableIdentifier(identifier); err != nil {
		return nil, err
	}

	ns := catalog.NamespaceFromIdent(identifier)
	tbl := catalog.TableNameFromIdent(identifier)
	nsKey, err := c.namespaceKey(ctx, ns)
	if err != nil {
		return nil, err
	}

	result, err := withReadTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) (*sqlIcebergTable, error) {
		t := new(sqlIcebergTable)
		sel := tx.NewSelect().Model(t).
			Where("catalog_name = ?", c.name).
			Where("table_namespace = ?", nsKey).
			Where("table_name = ?", tbl)
		if c.isV0() {
			sel = sel.ExcludeColumn("iceberg_type")
		} else {
			sel = sel.Where("(iceberg_type = ? OR iceberg_type IS NULL)", TableType)
		}
		err := sel.Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", catalog.ErrNoSuchTable, identifier)
		}

		if err != nil {
			return nil, fmt.Errorf("error encountered loading table %s: %w", identifier, err)
		}

		return t, nil
	})
	if err != nil {
		return nil, err
	}

	if !result.MetadataLocation.Valid {
		return nil, fmt.Errorf("%w: %s, metadata location is missing", catalog.ErrNoSuchTable, identifier)
	}

	reporter, err := c.reporter.Get(c.props)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metrics reporter: %w", err)
	}

	return table.NewFromLocation(
		ctx,
		identifier,
		result.MetadataLocation.String,
		io.LoadFSFunc(c.props, result.MetadataLocation.String),
		c,
		table.WithMetricsReporter(reporter),
	)
}

func (c *Catalog) DropTable(ctx context.Context, identifier table.Identifier) error {
	if err := catalog.ValidateTableIdentifier(identifier); err != nil {
		return err
	}

	ns, err := c.namespaceKey(ctx, catalog.NamespaceFromIdent(identifier))
	if err != nil {
		return err
	}
	tbl := catalog.TableNameFromIdent(identifier)

	return withWriteTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) error {
		del := tx.NewDelete().Model(&sqlIcebergTable{
			CatalogName:    c.name,
			TableNamespace: ns,
			TableName:      tbl,
		}).WherePK()
		if !c.isV0() {
			del = del.Where("(iceberg_type = ? OR iceberg_type IS NULL)", TableType)
		}
		res, err := del.Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete table entry: %w", err)
		}

		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("error encountered when deleting table entry: %w", err)
		}

		if n == 0 {
			return fmt.Errorf("%w: %s", catalog.ErrNoSuchTable, identifier)
		}

		return nil
	})
}

func (c *Catalog) PurgeTable(ctx context.Context, identifier table.Identifier) error {
	if err := catalog.ValidateTableIdentifier(identifier); err != nil {
		return err
	}

	tbl, err := c.LoadTable(ctx, identifier)
	if err != nil {
		return err
	}

	// Drop the table entry from the catalog first
	err = c.DropTable(ctx, identifier)
	if err != nil {
		return err
	}

	// Physically delete all table files on storage best-effort
	if purgeErr := tbl.PurgeFiles(ctx); purgeErr != nil {
		log.Printf("WARNING: dropped table %s but failed to purge files: %v", identifier, purgeErr)
	}

	return nil
}

func (c *Catalog) RenameTable(ctx context.Context, from, to table.Identifier) (*table.Table, error) {
	if err := catalog.ValidateTableIdentifier(from); err != nil {
		return nil, err
	}
	if err := catalog.ValidateTableIdentifier(to); err != nil {
		return nil, err
	}

	fromNs, err := c.namespaceKey(ctx, catalog.NamespaceFromIdent(from))
	if err != nil {
		return nil, err
	}
	fromTbl := catalog.TableNameFromIdent(from)

	toNs, exists, err := c.resolveNamespaceKey(ctx, catalog.NamespaceFromIdent(to))
	if err != nil {
		return nil, err
	}
	toTbl := catalog.TableNameFromIdent(to)

	if !exists {
		return nil, fmt.Errorf("%w: %s", catalog.ErrNoSuchNamespace, toNs)
	}

	err = withSerializableWriteTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) error {
		sourceExistsQuery := tx.NewSelect().Model(&sqlIcebergTable{
			CatalogName:    c.name,
			TableNamespace: fromNs,
			TableName:      fromTbl,
		}).ColumnExpr("1").WherePK()
		if !c.isV0() {
			sourceExistsQuery = sourceExistsQuery.Where("(iceberg_type = ? OR iceberg_type IS NULL)", TableType)
		}
		sourceExists, err := sourceExistsQuery.Exists(ctx)
		if err != nil {
			return fmt.Errorf("error encountered checking existence of table '%s': %w", from, err)
		}
		if !sourceExists {
			return fmt.Errorf("%w: %s", catalog.ErrNoSuchTable, from)
		}

		sourceNamespaceExists, err := c.namespaceExistsInTx(ctx, tx, fromNs)
		if err != nil {
			return err
		}
		if !sourceNamespaceExists {
			return fmt.Errorf("%w: %s", catalog.ErrNoSuchNamespace, fromNs)
		}

		destinationNamespaceExists, err := c.namespaceExistsInTx(ctx, tx, toNs)
		if err != nil {
			return err
		}
		if !destinationNamespaceExists {
			return fmt.Errorf("%w: %s", catalog.ErrNoSuchNamespace, toNs)
		}

		exists, err := tx.NewSelect().Model(&sqlIcebergTable{
			CatalogName:    c.name,
			TableNamespace: toNs,
			TableName:      toTbl,
		}).ColumnExpr("1").WherePK().Exists(ctx)
		if err != nil {
			return fmt.Errorf("error encountered checking existence of table '%s': %w", to, err)
		}

		if exists {
			return catalog.ErrTableAlreadyExists
		}

		// Only table_namespace/table_name are updated via Set, so iceberg_type is
		// never written; on V1 we still scope the row with the type predicate.
		upd := tx.NewUpdate().Model(&sqlIcebergTable{
			CatalogName:    c.name,
			TableNamespace: fromNs,
			TableName:      fromTbl,
		}).WherePK().
			Set("table_namespace = ?", toNs).
			Set("table_name = ?", toTbl)
		if !c.isV0() {
			upd = upd.Where("(iceberg_type = ? OR iceberg_type IS NULL)", TableType)
		}
		res, err := upd.Exec(ctx)
		if err != nil {
			return fmt.Errorf("error renaming table from '%s' to %s': %w", from, to, err)
		}

		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("error renaming table from '%s' to %s': %w", from, to, err)
		}

		if n == 0 {
			return fmt.Errorf("%w: %s", catalog.ErrNoSuchTable, from)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return c.LoadTable(ctx, to)
}

// CheckTableExists reports whether the identifier has a table row in the
// catalog. It does not validate that the referenced metadata file is readable.
func (c *Catalog) CheckTableExists(ctx context.Context, identifier table.Identifier) (bool, error) {
	if err := catalog.ValidateTableIdentifier(identifier); err != nil {
		return false, err
	}

	ns, err := c.namespaceKey(ctx, catalog.NamespaceFromIdent(identifier))
	if err != nil {
		return false, err
	}
	tableName := catalog.TableNameFromIdent(identifier)

	return withReadTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) (bool, error) {
		query := tx.NewSelect().Model(&sqlIcebergTable{
			CatalogName:    c.name,
			TableNamespace: ns,
			TableName:      tableName,
		}).WherePK()
		if c.isV0() {
			query = query.ExcludeColumn("iceberg_type")
		} else {
			query = query.Where("(iceberg_type = ? OR iceberg_type IS NULL)", TableType)
		}

		exists, err := query.Exists(ctx)
		if err != nil {
			return false, fmt.Errorf("error checking table existence: %w", err)
		}

		return exists, nil
	})
}

func (c *Catalog) CreateNamespace(ctx context.Context, namespace table.Identifier, props iceberg.Properties) error {
	if err := checkValidNamespace(namespace); err != nil {
		return err
	}

	for key := range props {
		if isReservedNamespaceProperty(key) {
			return fmt.Errorf("%w: cannot create namespace with reserved namespace property %q", iceberg.ErrInvalidArgument, key)
		}
	}

	_, exists, err := c.resolveNamespaceKey(ctx, namespace)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("%w: %s", catalog.ErrNamespaceAlreadyExists, strings.Join(namespace, "."))
	}

	if len(props) == 0 {
		props = minimalNamespaceProps
	}

	nsToCreate := namespaceToString(namespace)

	return withWriteTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) error {
		toInsert := make([]sqlIcebergNamespaceProps, 0, len(props))
		for k, v := range props {
			toInsert = append(toInsert, sqlIcebergNamespaceProps{
				CatalogName:   c.name,
				Namespace:     nsToCreate,
				PropertyKey:   k,
				PropertyValue: sql.NullString{String: v, Valid: true},
			})
		}

		// A concurrent writer may have won the race; return the sentinel in the same
		// form as the pre-check above. This must stay the first read on the outer tx
		// so MySQL's REPEATABLE READ snapshot lands after the winner committed.
		recheck := func(insertErr error) error {
			_, exists, checkErr := c.resolveNamespaceKeyInTx(ctx, tx, namespace)
			if checkErr == nil && exists {
				return fmt.Errorf("%w: %s", catalog.ErrNamespaceAlreadyExists, strings.Join(namespace, "."))
			}
			if checkErr != nil {
				return fmt.Errorf("error inserting namespace properties for namespace '%s': %w", namespace, errors.Join(insertErr, checkErr))
			}

			return fmt.Errorf("error inserting namespace properties for namespace '%s': %w", namespace, insertErr)
		}

		// Postgres aborts the whole tx on a failed insert, so only it needs the
		// savepoint for the re-check; SQLite/MySQL don't, and Oracle can't RELEASE it.
		if tx.Dialect().Name() == dialect.PG {
			sp, err := tx.BeginTx(ctx, nil)
			if err != nil {
				return fmt.Errorf("error creating savepoint for namespace '%s': %w", namespace, err)
			}

			if _, err = sp.NewInsert().Model(&toInsert).Exec(ctx); err != nil {
				if rbErr := sp.Rollback(); rbErr != nil {
					return fmt.Errorf("error inserting namespace properties for namespace '%s': %w", namespace, errors.Join(err, rbErr))
				}

				return recheck(err)
			}

			if err = sp.Commit(); err != nil {
				return fmt.Errorf("error releasing savepoint for namespace '%s': %w", namespace, err)
			}

			return nil
		}

		if _, err := tx.NewInsert().Model(&toInsert).Exec(ctx); err != nil {
			return recheck(err)
		}

		return nil
	})
}

func (c *Catalog) DropNamespace(ctx context.Context, namespace table.Identifier) error {
	if err := checkValidNamespace(namespace); err != nil {
		return err
	}

	return withSerializableWriteTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) error {
		nsToDelete, exists, err := c.resolveNamespaceKeyInTx(ctx, tx, namespace)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w: %s", catalog.ErrNoSuchNamespace, nsToDelete)
		}

		objectsExist, err := tx.NewSelect().Model((*sqlIcebergTable)(nil)).
			ColumnExpr("1").
			Where("catalog_name = ?", c.name).
			Where("table_namespace = ?", nsToDelete).
			Limit(1).Exists(ctx)
		if err != nil {
			return fmt.Errorf("error checking namespace '%s' for catalog objects: %w", nsToDelete, err)
		}
		if objectsExist {
			return fmt.Errorf("%w: catalog objects exist in namespace %s", catalog.ErrNamespaceNotEmpty, nsToDelete)
		}

		_, err = tx.NewDelete().Model((*sqlIcebergNamespaceProps)(nil)).
			Where("catalog_name = ?", c.name).
			Where("namespace = ?", nsToDelete).Exec(ctx)
		if err != nil {
			return fmt.Errorf("error deleting namespace '%s': %w", nsToDelete, err)
		}

		return nil
	})
}

func (c *Catalog) LoadNamespaceProperties(ctx context.Context, namespace table.Identifier) (iceberg.Properties, error) {
	if err := checkValidNamespace(namespace); err != nil {
		return nil, err
	}

	nsToLoad, exists, err := c.resolveNamespaceKey(ctx, namespace)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("%w: %s", catalog.ErrNoSuchNamespace, nsToLoad)
	}

	return withReadTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) (iceberg.Properties, error) {
		var props []sqlIcebergNamespaceProps
		err := tx.NewSelect().Model(&props).
			Where("catalog_name = ?", c.name).
			Where("namespace = ?", nsToLoad).Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("error loading namespace properties for '%s': %w", namespace, err)
		}

		result := make(iceberg.Properties)
		for _, p := range props {
			if isReservedNamespaceProperty(p.PropertyKey) {
				continue
			}
			result[p.PropertyKey] = p.PropertyValue.String
		}

		return result, nil
	})
}

func validateNamespacePropertyUpdates(removals []string, updates iceberg.Properties) error {
	for key := range updates {
		if isReservedNamespaceProperty(key) {
			return fmt.Errorf("%w: cannot update reserved namespace property %q", iceberg.ErrInvalidArgument, key)
		}
	}

	for _, key := range removals {
		if isReservedNamespaceProperty(key) {
			// Removing the marker would make a namespace without user properties disappear.
			return fmt.Errorf("%w: cannot remove reserved namespace property %q", iceberg.ErrInvalidArgument, key)
		}
	}

	return nil
}

func (c *Catalog) ListTables(ctx context.Context, namespace table.Identifier) iter.Seq2[table.Identifier, error] {
	tables, err := c.listTablesAll(ctx, namespace)
	if err != nil {
		return func(yield func(table.Identifier, error) bool) {
			yield(table.Identifier{}, err)
		}
	}

	return func(yield func(table.Identifier, error) bool) {
		for _, t := range tables {
			if !yield(t, nil) {
				return
			}
		}
	}
}

func (c *Catalog) listTablesAll(ctx context.Context, namespace table.Identifier) ([]table.Identifier, error) {
	ns := namespaceToString(namespace)
	if len(namespace) > 0 {
		var exists bool
		var err error
		ns, exists, err = c.resolveNamespaceKey(ctx, namespace)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("%w: %s", catalog.ErrNoSuchNamespace, strings.Join(namespace, "."))
		}
	}

	tables, err := withReadTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) ([]sqlIcebergTable, error) {
		var tables []sqlIcebergTable
		sel := tx.NewSelect().Model(&tables).
			Where("catalog_name = ?", c.name).
			Where("table_namespace = ?", ns)
		if c.isV0() {
			sel = sel.ExcludeColumn("iceberg_type")
		} else {
			sel = sel.Where("(iceberg_type = ? OR iceberg_type IS NULL)", TableType)
		}
		err := sel.Scan(ctx)

		return tables, err
	})
	if err != nil {
		return nil, fmt.Errorf("error listing tables for namespace '%s': %w", namespace, err)
	}

	ret := make([]table.Identifier, len(tables))
	for i, t := range tables {
		namespace, err := namespaceFromString(t.TableNamespace)
		if err != nil {
			return nil, err
		}
		ret[i] = append(namespace, t.TableName)
	}

	return ret, nil
}

func (c *Catalog) ListNamespaces(ctx context.Context, parent table.Identifier) ([]table.Identifier, error) {
	tableQuery := c.db.NewSelect().Model((*sqlIcebergTable)(nil)).
		Column("table_namespace").Where("catalog_name = ?", c.name)
	nsQuery := c.db.NewSelect().Model((*sqlIcebergNamespaceProps)(nil)).
		Column("namespace").Where("catalog_name = ?", c.name)

	if len(parent) > 0 {
		_, exists, err := c.resolveNamespaceKey(ctx, parent)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("%w: %s", catalog.ErrNoSuchNamespace, strings.Join(parent, "."))
		}

		tableQuery = whereNamespaceParent(tableQuery, "table_namespace", parent)
		nsQuery = whereNamespaceParent(nsQuery, "namespace", parent)
	}

	namespaces, err := withReadTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) ([]string, error) {
		var namespaces []string

		rows, err := tx.QueryContext(ctx, tableQuery.String()+" UNION "+nsQuery.String())
		if err != nil {
			return nil, fmt.Errorf("error listing namespaces for '%s': %w", parent, err)
		}

		err = c.db.ScanRows(ctx, rows, &namespaces)

		return namespaces, err
	})
	if err != nil {
		return nil, err
	}

	ret := make([]table.Identifier, 0, len(namespaces))
	for _, n := range namespaces {
		ident, err := namespaceFromString(n)
		if err != nil {
			return nil, err
		}
		if len(parent) == 0 || (len(ident) >= len(parent) && slices.Equal(ident[:len(parent)], parent)) {
			ret = append(ret, ident)
		}
	}

	return ret, nil
}

// avoid circular dependency while still avoiding having to export the getUpdatedPropsAndUpdateSummary function
// so that we can re-use it in the catalog implementations without duplicating the code.

//go:linkname getUpdatedPropsAndUpdateSummary github.com/apache/iceberg-go/catalog.getUpdatedPropsAndUpdateSummary
func getUpdatedPropsAndUpdateSummary(currentProps iceberg.Properties, removals []string, updates iceberg.Properties) (iceberg.Properties, catalog.PropertiesUpdateSummary, error)

func (c *Catalog) UpdateNamespaceProperties(ctx context.Context, namespace table.Identifier, removals []string, updates iceberg.Properties) (catalog.PropertiesUpdateSummary, error) {
	var summary catalog.PropertiesUpdateSummary
	if err := checkValidNamespace(namespace); err != nil {
		return summary, err
	}
	if err := validateNamespacePropertyUpdates(removals, updates); err != nil {
		return summary, err
	}

	err := withSerializableWriteTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) error {
		nsToUpdate, exists, err := c.resolveNamespaceKeyInTx(ctx, tx, namespace)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w: %s", catalog.ErrNoSuchNamespace, nsToUpdate)
		}

		var storedProps []sqlIcebergNamespaceProps
		err = tx.NewSelect().Model(&storedProps).
			Where("catalog_name = ?", c.name).
			Where("namespace = ?", nsToUpdate).Scan(ctx)
		if err != nil {
			return fmt.Errorf("error loading namespace properties for '%s': %w", nsToUpdate, err)
		}

		currentProps := make(iceberg.Properties, len(storedProps))
		for _, prop := range storedProps {
			currentProps[prop.PropertyKey] = prop.PropertyValue.String
		}

		_, nextSummary, err := getUpdatedPropsAndUpdateSummary(currentProps, removals, updates)
		if err != nil {
			return err
		}

		var m *sqlIcebergNamespaceProps
		if len(removals) > 0 {
			_, err := tx.NewDelete().Model(m).
				Where("catalog_name = ?", c.name).
				Where("namespace = ?", nsToUpdate).
				Where("property_key in (?)", bun.List(removals)).Exec(ctx)
			if err != nil {
				return fmt.Errorf("error deleting properties for '%s': %w", nsToUpdate, err)
			}
		}

		if len(updates) > 0 {
			props := make([]sqlIcebergNamespaceProps, 0, len(updates))
			for k, v := range updates {
				props = append(props, sqlIcebergNamespaceProps{
					CatalogName:   c.name,
					Namespace:     nsToUpdate,
					PropertyKey:   k,
					PropertyValue: sql.NullString{String: v, Valid: true},
				})
			}

			q := tx.NewInsert().Model(&props)
			switch {
			case c.db.HasFeature(feature.InsertOnConflict):
				q = q.On("CONFLICT (catalog_name, namespace, property_key) DO UPDATE").
					Set("property_value = EXCLUDED.property_value")
			case c.db.HasFeature(feature.InsertOnDuplicateKey):
				q = q.On("DUPLICATE KEY UPDATE")
			default:
				_, err := tx.NewDelete().Model(m).
					Where("catalog_name = ?", c.name).
					Where("namespace = ?", nsToUpdate).
					Where("property_key in (?)", bun.List(slices.Collect(maps.Keys(updates)))).
					Exec(ctx)
				if err != nil {
					return fmt.Errorf("error deleting properties for '%s': %w", nsToUpdate, err)
				}
			}

			_, err := q.Exec(ctx)
			if err != nil {
				return fmt.Errorf("error updating namespace properties for '%s': %w", nsToUpdate, err)
			}
		}

		summary = nextSummary

		return nil
	})

	return summary, err
}

func (c *Catalog) CheckNamespaceExists(ctx context.Context, namespace table.Identifier) (bool, error) {
	_, exists, err := c.resolveNamespaceKey(ctx, namespace)

	return exists, err
}

// CreateView creates a new view in the catalog.
func (c *Catalog) CreateView(ctx context.Context, identifier table.Identifier, schema *iceberg.Schema, viewSQL string, props iceberg.Properties) error {
	if c.isV0() {
		return errViewsUnsupportedOnV0
	}

	if err := catalog.ValidateViewIdentifier(identifier); err != nil {
		return err
	}

	nsIdent := catalog.NamespaceFromIdent(identifier)
	viewIdent := catalog.TableNameFromIdent(identifier)
	ns, exists, err := c.resolveNamespaceKey(ctx, nsIdent)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w: %s", catalog.ErrNoSuchNamespace, ns)
	}

	loc, err := internal.ResolveTableLocationWithNamespace(
		ctx, "", nsIdent, ns, viewIdent, c.props, c.LoadNamespaceProperties)
	if err != nil {
		return err
	}

	createdView, err := view.CreateView(ctx, c.name, identifier, schema, viewSQL, nsIdent, loc, props)
	if err != nil {
		return err
	}
	metadataLocation := createdView.MetadataLocation()

	err = withSerializableWriteTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) error {
		exists, err := c.namespaceExistsInTx(ctx, tx, ns)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w: %s", catalog.ErrNoSuchNamespace, ns)
		}

		if err := c.checkIdentifierAvailable(ctx, tx, ns, viewIdent, identifier); err != nil {
			return err
		}

		_, err = tx.NewInsert().Model(&sqlIcebergTable{
			CatalogName:      c.name,
			TableNamespace:   ns,
			TableName:        viewIdent,
			IcebergType:      sql.NullString{String: ViewType, Valid: true},
			MetadataLocation: sql.NullString{String: metadataLocation, Valid: true},
		}).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create view: %w", err)
		}

		return nil
	})
	if err != nil {
		if !errors.Is(err, catalog.ErrTableAlreadyExists) && !errors.Is(err, catalog.ErrViewAlreadyExists) {
			if collisionErr := c.checkIdentifierAvailableInCatalog(ctx, ns, viewIdent, identifier); errors.Is(collisionErr, catalog.ErrTableAlreadyExists) ||
				errors.Is(collisionErr, catalog.ErrViewAlreadyExists) {
				err = collisionErr
			}
		}
		loadFS := io.LoadFSFunc(c.props, metadataLocation)
		if cleanupErr := removeUncommittedMetadata(ctx, metadataLocation, loadFS); cleanupErr != nil {
			err = errors.Join(err, cleanupErr)
		}
	}

	return err
}

// ListViews returns a list of view identifiers in the catalog.
func (c *Catalog) ListViews(ctx context.Context, namespace table.Identifier) iter.Seq2[table.Identifier, error] {
	views, err := c.listViewsAll(ctx, namespace)
	if err != nil {
		return func(yield func(table.Identifier, error) bool) {
			yield(table.Identifier{}, err)
		}
	}

	return func(yield func(table.Identifier, error) bool) {
		for _, v := range views {
			if !yield(v, nil) {
				return
			}
		}
	}
}

func (c *Catalog) listViewsAll(ctx context.Context, namespace table.Identifier) ([]table.Identifier, error) {
	if c.isV0() {
		return nil, nil
	}

	ns := namespaceToString(namespace)
	if len(namespace) > 0 {
		var exists bool
		var err error
		ns, exists, err = c.resolveNamespaceKey(ctx, namespace)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("%w: %s", catalog.ErrNoSuchNamespace, strings.Join(namespace, "."))
		}
	}

	views, err := withReadTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) ([]sqlIcebergTable, error) {
		var views []sqlIcebergTable
		err := tx.NewSelect().Model(&views).
			Where("catalog_name = ?", c.name).
			Where("table_namespace = ?", ns).
			Where("iceberg_type = ?", ViewType).
			Scan(ctx)

		return views, err
	})
	if err != nil {
		return nil, fmt.Errorf("error listing views for namespace '%s': %w", namespace, err)
	}

	ret := make([]table.Identifier, len(views))
	for i, v := range views {
		namespace, err := namespaceFromString(v.TableNamespace)
		if err != nil {
			return nil, err
		}
		ret[i] = append(namespace, v.TableName)
	}

	return ret, nil
}

// DropView deletes a view from the catalog.
func (c *Catalog) DropView(ctx context.Context, identifier table.Identifier) error {
	if c.isV0() {
		return errViewsUnsupportedOnV0
	}

	ns, err := c.namespaceKey(ctx, catalog.NamespaceFromIdent(identifier))
	if err != nil {
		return err
	}
	viewName := catalog.TableNameFromIdent(identifier)

	metadataLocation := ""
	view, err := withReadTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) (*sqlIcebergTable, error) {
		v := new(sqlIcebergTable)
		err := tx.NewSelect().Model(v).
			Where("catalog_name = ?", c.name).
			Where("table_namespace = ?", ns).
			Where("table_name = ?", viewName).
			Where("iceberg_type = ?", ViewType).
			Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", catalog.ErrNoSuchView, identifier)
		}
		if err != nil {
			return nil, fmt.Errorf("error encountered loading view %s: %w", identifier, err)
		}

		return v, nil
	})
	if err != nil {
		return err
	}

	if view.MetadataLocation.Valid {
		metadataLocation = view.MetadataLocation.String
	}

	err = withWriteTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) error {
		res, err := tx.NewDelete().Model(&sqlIcebergTable{
			CatalogName:    c.name,
			TableNamespace: ns,
			TableName:      viewName,
		}).WherePK().Where("iceberg_type = ?", ViewType).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete view entry: %w", err)
		}

		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("error encountered when deleting view entry: %w", err)
		}

		if n == 0 {
			return fmt.Errorf("%w: %s", catalog.ErrNoSuchView, identifier)
		}

		return nil
	})
	if err != nil {
		return err
	}

	if metadataLocation != "" {
		fs, err := io.LoadFS(ctx, c.props, metadataLocation)
		if err != nil {
			return err
		}

		_ = fs.Remove(metadataLocation)
	}

	return nil
}

// CheckViewExists returns true if a view exists in the catalog.
func (c *Catalog) CheckViewExists(ctx context.Context, identifier table.Identifier) (bool, error) {
	if c.isV0() {
		return false, nil
	}

	if err := catalog.ValidateViewIdentifier(identifier); err != nil {
		return false, err
	}

	ns, err := c.namespaceKey(ctx, catalog.NamespaceFromIdent(identifier))
	if err != nil {
		return false, err
	}
	viewName := catalog.TableNameFromIdent(identifier)

	return withReadTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) (bool, error) {
		exists, err := tx.NewSelect().Model(&sqlIcebergTable{
			CatalogName:    c.name,
			TableNamespace: ns,
			TableName:      viewName,
		}).WherePK().Where("iceberg_type = ?", ViewType).Exists(ctx)
		if err != nil {
			return false, fmt.Errorf("error checking view existence: %w", err)
		}

		return exists, nil
	})
}

// LoadView loads a view from the catalog.
func (c *Catalog) LoadView(ctx context.Context, identifier table.Identifier) (view.Metadata, error) {
	if c.isV0() {
		return nil, errViewsUnsupportedOnV0
	}

	if err := catalog.ValidateViewIdentifier(identifier); err != nil {
		return nil, err
	}

	ns, err := c.namespaceKey(ctx, catalog.NamespaceFromIdent(identifier))
	if err != nil {
		return nil, err
	}
	viewName := catalog.TableNameFromIdent(identifier)

	v, err := withReadTx(ctx, c.db, func(ctx context.Context, tx bun.Tx) (*sqlIcebergTable, error) {
		v := new(sqlIcebergTable)
		err := tx.NewSelect().Model(v).
			Where("catalog_name = ?", c.name).
			Where("table_namespace = ?", ns).
			Where("table_name = ?", viewName).
			Where("iceberg_type = ?", ViewType).
			Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", catalog.ErrNoSuchView, identifier)
		}
		if err != nil {
			return nil, fmt.Errorf("error encountered loading view %s: %w", identifier, err)
		}

		return v, nil
	})
	if err != nil {
		return nil, err
	}

	if !v.MetadataLocation.Valid {
		return nil, fmt.Errorf("%w: %s, metadata location is missing", catalog.ErrNoSuchView, identifier)
	}

	loadedView, err := view.NewFromLocation(ctx, identifier, v.MetadataLocation.String, io.LoadFSFunc(c.props, v.MetadataLocation.String))
	if err != nil {
		return nil, err
	}

	return loadedView.Metadata(), nil
}
