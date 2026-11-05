package bun

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

const (
	discardUnknownColumns internal.Flag = 1 << iota
)

// DBStats tracks aggregate query counters collected by Bun.
type DBStats struct {
	Queries uint32
	Errors  uint32
}

// DBOption mutates DB configuration during construction.
type DBOption func(db *DB)

// WithOptions applies multiple DBOption values at once.
func WithOptions(opts ...DBOption) DBOption {
	return func(db *DB) {
		for _, opt := range opts {
			opt(db)
		}
	}
}

// WithDiscardUnknownColumns ignores columns returned by queries that are not present in models.
func WithDiscardUnknownColumns() DBOption {
	return func(db *DB) {
		db.flags = db.flags.Set(discardUnknownColumns)
	}
}

// ConnResolver enables routing queries to multiple databases.
type ConnResolver interface {
	ResolveConn(ctx context.Context, query Query) IConn
	Close() error
}

// WithConnResolver registers a connection resolver that chooses a connection per query.
func WithConnResolver(resolver ConnResolver) DBOption {
	return func(db *DB) {
		db.resolver = resolver
	}
}

// DB is the central access point for building and executing Bun queries.
type DB struct {
	// Must be a pointer so we copy the whole state, not individual fields.
	*noCopyState

	gen        schema.QueryGen
	queryHooks []QueryHook
}

// noCopyState contains DB fields that must not be copied on clone(),
// for example, it is forbidden to copy atomic.Pointer.
type noCopyState struct {
	*sql.DB
	dialect  schema.Dialect
	resolver ConnResolver

	flags  internal.Flag
	closed atomic.Bool

	stats DBStats
}

// NewDB wraps an existing *sql.DB with Bun using the given dialect and options.
func NewDB(sqldb *sql.DB, dialect schema.Dialect, opts ...DBOption) *DB {
	dialect.Init(sqldb)

	db := &DB{
		noCopyState: &noCopyState{
			DB:      sqldb,
			dialect: dialect,
		},
		gen: schema.NewQueryGen(dialect),
	}

	for _, opt := range opts {
		opt(db)
	}

	return db
}

// String returns a string representation of the DB showing its dialect.
func (db *DB) String() string {
	var b strings.Builder
	b.WriteString("DB<dialect=")
	b.WriteString(db.dialect.Name().String())
	b.WriteString(">")
	return b.String()
}

// Close closes the database connection and any registered connection resolver.
// It returns the first error encountered during closure.
func (db *DB) Close() error {
	if db.closed.Swap(true) {
		return nil
	}

	firstErr := db.DB.Close()

	if db.resolver != nil {
		if err := db.resolver.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// DBStats returns aggregated query statistics including total queries and errors.
func (db *DB) DBStats() DBStats {
	return DBStats{
		Queries: atomic.LoadUint32(&db.stats.Queries),
		Errors:  atomic.LoadUint32(&db.stats.Errors),
	}
}

// NewValues creates a VALUES query for inserting multiple rows efficiently.
func (db *DB) NewValues(model any) *ValuesQuery {
	return NewValuesQuery(db, model)
}

// NewMerge creates a MERGE (UPSERT) query for insert-or-update operations.
func (db *DB) NewMerge() *MergeQuery {
	return NewMergeQuery(db)
}

// NewSelect creates a SELECT query builder.
func (db *DB) NewSelect() *SelectQuery {
	return NewSelectQuery(db)
}

// NewInsert creates an INSERT query builder.
func (db *DB) NewInsert() *InsertQuery {
	return NewInsertQuery(db)
}

// NewUpdate creates an UPDATE query builder.
func (db *DB) NewUpdate() *UpdateQuery {
	return NewUpdateQuery(db)
}

// NewDelete creates a DELETE query builder.
func (db *DB) NewDelete() *DeleteQuery {
	return NewDeleteQuery(db)
}

// NewRaw creates a raw SQL query with the given query string and arguments.
func (db *DB) NewRaw(query string, args ...any) *RawQuery {
	return NewRawQuery(db, query, args...)
}

// NewCreateTable creates a CREATE TABLE DDL query builder.
func (db *DB) NewCreateTable() *CreateTableQuery {
	return NewCreateTableQuery(db)
}

// NewDropTable creates a DROP TABLE DDL query builder.
func (db *DB) NewDropTable() *DropTableQuery {
	return NewDropTableQuery(db)
}

// NewCreateIndex creates a CREATE INDEX DDL query builder.
func (db *DB) NewCreateIndex() *CreateIndexQuery {
	return NewCreateIndexQuery(db)
}

// NewDropIndex creates a DROP INDEX DDL query builder.
func (db *DB) NewDropIndex() *DropIndexQuery {
	return NewDropIndexQuery(db)
}

// NewTruncateTable creates a TRUNCATE TABLE DDL query builder.
func (db *DB) NewTruncateTable() *TruncateTableQuery {
	return NewTruncateTableQuery(db)
}

// NewAddColumn creates an ALTER TABLE ADD COLUMN DDL query builder.
func (db *DB) NewAddColumn() *AddColumnQuery {
	return NewAddColumnQuery(db)
}

// NewDropColumn creates an ALTER TABLE DROP COLUMN DDL query builder.
func (db *DB) NewDropColumn() *DropColumnQuery {
	return NewDropColumnQuery(db)
}

// ResetModel drops and recreates tables for the given models.
// This is useful for testing and development but should not be used in production.
func (db *DB) ResetModel(ctx context.Context, models ...any) error {
	for _, model := range models {
		if _, err := db.NewDropTable().Model(model).IfExists().Cascade().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateTable().Model(model).Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Dialect returns the database dialect being used (e.g., PostgreSQL, MySQL, SQLite).
func (db *DB) Dialect() schema.Dialect {
	return db.dialect
}

// ScanRows scans all rows from the result set into the destination values.
// It closes the rows when complete.
func (db *DB) ScanRows(ctx context.Context, rows *sql.Rows, dest ...any) error {
	defer rows.Close()

	model, err := newModel(db, dest)
	if err != nil {
		return err
	}

	_, err = model.ScanRows(ctx, rows)
	if err != nil {
		return err
	}

	return rows.Err()
}

// ScanRow scans a single row from the result set into the destination values.
func (db *DB) ScanRow(ctx context.Context, rows *sql.Rows, dest ...any) error {
	model, err := newModel(db, dest)
	if err != nil {
		return err
	}

	rs, ok := model.(rowScanner)
	if !ok {
		return fmt.Errorf("bun: %T does not support ScanRow", model)
	}

	return rs.ScanRow(ctx, rows)
}

// Table returns the schema table metadata for the given type.
func (db *DB) Table(typ reflect.Type) *schema.Table {
	return db.dialect.Tables().Get(typ)
}

// RegisterModel registers models by name so they can be referenced in table relations
// and fixtures.
func (db *DB) RegisterModel(models ...any) {
	db.dialect.Tables().Register(models...)
}

// clone creates a shallow copy of the DB with independent query hooks.
func (db *DB) clone() *DB {
	clone := *db

	l := len(clone.queryHooks)
	clone.queryHooks = clone.queryHooks[:l:l]

	return &clone
}

// WithNamedArg returns a copy of the DB with an additional named argument
// bound into its query generator. Named arguments can later be referenced
// in SQL queries using placeholders (e.g. ?name). This method does not
// mutate the original DB instance but instead creates a cloned copy.
func (db *DB) WithNamedArg(name string, value any) *DB {
	clone := db.clone()
	clone.gen = clone.gen.WithNamedArg(name, value)
	return clone
}

// QueryGen returns the query generator used for formatting SQL queries.
func (db *DB) QueryGen() schema.QueryGen {
	return db.gen
}

type queryHookIniter interface {
	Init(db *DB)
}

// WithQueryHook returns a copy of the DB with the provided query hook
// attached. A query hook allows inspection or modification of queries
// before/after execution (e.g. for logging, tracing, metrics).
// If the hook implements queryHookIniter, its Init method is invoked
// with the current DB before cloning. Like other modifiers, this
// method leaves the original DB unmodified.
func (db *DB) WithQueryHook(hook QueryHook) *DB {
	if initer, ok := hook.(queryHookIniter); ok {
		initer.Init(db)
	}

	clone := db.clone()
	clone.queryHooks = append(clone.queryHooks, hook)
	return clone
}

// DEPRECATED: use WithQueryHook instead
func (db *DB) AddQueryHook(hook QueryHook) {
	if initer, ok := hook.(queryHookIniter); ok {
		initer.Init(db)
	}
	db.queryHooks = append(db.queryHooks, hook)
}

// DEPRECATED: use WithQueryHook instead
func (db *DB) ResetQueryHooks() {
	for i := range db.queryHooks {
		db.queryHooks[i] = nil
	}
	db.queryHooks = nil
}

// UpdateFQN returns a fully qualified column name. For MySQL, it returns the column name with
// the table alias. For other RDBMS, it returns just the column name.
func (db *DB) UpdateFQN(alias, column string) Ident {
	if db.HasFeature(feature.UpdateMultiTable) {
		return Ident(alias + "." + column)
	}
	return Ident(column)
}

// HasFeature uses feature package to report whether the underlying DBMS supports this feature.
func (db *DB) HasFeature(feat feature.Feature) bool {
	return db.dialect.Features().Has(feat)
}

//------------------------------------------------------------------------------

// Exec executes a query without returning rows using a background context.
// Arguments are formatted using the dialect's placeholder syntax.
func (db *DB) Exec(query string, args ...any) (sql.Result, error) {
	return db.ExecContext(context.Background(), query, args...)
}

// ExecContext executes a query without returning rows.
// Arguments are formatted using the dialect's placeholder syntax.
// Query hooks are invoked before and after execution.
func (db *DB) ExecContext(
	ctx context.Context, query string, args ...any,
) (sql.Result, error) {
	formattedQuery := db.format(query, args)
	ctx, event := db.beforeQuery(ctx, nil, query, args, formattedQuery, nil)
	res, err := db.DB.ExecContext(ctx, formattedQuery)
	db.afterQuery(ctx, event, res, err)
	return res, err
}

// Query executes a query returning rows using a background context.
// Arguments are formatted using the dialect's placeholder syntax.
func (db *DB) Query(query string, args ...any) (*sql.Rows, error) {
	return db.QueryContext(context.Background(), query, args...)
}

// QueryContext executes a query returning rows.
// Arguments are formatted using the dialect's placeholder syntax.
// Query hooks are invoked before and after execution.
func (db *DB) QueryContext(
	ctx context.Context, query string, args ...any,
) (*sql.Rows, error) {
	formattedQuery := db.format(query, args)
	ctx, event := db.beforeQuery(ctx, nil, query, args, formattedQuery, nil)
	rows, err := db.DB.QueryContext(ctx, formattedQuery)
	db.afterQuery(ctx, event, nil, err)
	return rows, err
}

// QueryRow executes a query expected to return at most one row using a background context.
// Arguments are formatted using the dialect's placeholder syntax.
func (db *DB) QueryRow(query string, args ...any) *sql.Row {
	return db.QueryRowContext(context.Background(), query, args...)
}

// QueryRowContext executes a query expected to return at most one row.
// Arguments are formatted using the dialect's placeholder syntax.
// Query hooks are invoked before and after execution.
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	formattedQuery := db.format(query, args)
	ctx, event := db.beforeQuery(ctx, nil, query, args, formattedQuery, nil)
	row := db.DB.QueryRowContext(ctx, formattedQuery)
	db.afterQuery(ctx, event, nil, row.Err())
	return row
}

func (db *DB) format(query string, args []any) string {
	return db.gen.FormatQuery(query, args...)
}

//------------------------------------------------------------------------------

// Conn wraps *sql.Conn so queries continue to use Bun features and hooks.
type Conn struct {
	db *DB
	*sql.Conn
}

// Conn returns a Conn wrapping a dedicated *sql.Conn from the connection pool.
// Query hooks and dialect features remain active on the returned connection.
func (db *DB) Conn(ctx context.Context) (Conn, error) {
	conn, err := db.DB.Conn(ctx)
	if err != nil {
		return Conn{}, err
	}
	return Conn{
		db:   db,
		Conn: conn,
	}, nil
}

// ExecContext executes a query without returning rows on this connection.
func (c Conn) ExecContext(
	ctx context.Context, query string, args ...any,
) (sql.Result, error) {
	formattedQuery := c.db.format(query, args)
	ctx, event := c.db.beforeQuery(ctx, nil, query, args, formattedQuery, nil)
	res, err := c.Conn.ExecContext(ctx, formattedQuery)
	c.db.afterQuery(ctx, event, res, err)
	return res, err
}

// QueryContext executes a query returning rows on this connection.
func (c Conn) QueryContext(
	ctx context.Context, query string, args ...any,
) (*sql.Rows, error) {
	formattedQuery := c.db.format(query, args)
	ctx, event := c.db.beforeQuery(ctx, nil, query, args, formattedQuery, nil)
	rows, err := c.Conn.QueryContext(ctx, formattedQuery)
	c.db.afterQuery(ctx, event, nil, err)
	return rows, err
}

// QueryRowContext executes a query expected to return at most one row on this connection.
func (c Conn) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	formattedQuery := c.db.format(query, args)
	ctx, event := c.db.beforeQuery(ctx, nil, query, args, formattedQuery, nil)
	row := c.Conn.QueryRowContext(ctx, formattedQuery)
	c.db.afterQuery(ctx, event, nil, row.Err())
	return row
}

// Dialect returns the database dialect for this connection.
func (c Conn) Dialect() schema.Dialect {
	return c.db.Dialect()
}

// NewValues creates a VALUES query bound to this connection.
func (c Conn) NewValues(model any) *ValuesQuery {
	return NewValuesQuery(c.db, model).Conn(c)
}

// NewMerge creates a MERGE query bound to this connection.
func (c Conn) NewMerge() *MergeQuery {
	return NewMergeQuery(c.db).Conn(c)
}

// NewSelect creates a SELECT query bound to this connection.
func (c Conn) NewSelect() *SelectQuery {
	return NewSelectQuery(c.db).Conn(c)
}

// NewInsert creates an INSERT query bound to this connection.
func (c Conn) NewInsert() *InsertQuery {
	return NewInsertQuery(c.db).Conn(c)
}

// NewUpdate creates an UPDATE query bound to this connection.
func (c Conn) NewUpdate() *UpdateQuery {
	return NewUpdateQuery(c.db).Conn(c)
}

// NewDelete creates a DELETE query bound to this connection.
func (c Conn) NewDelete() *DeleteQuery {
	return NewDeleteQuery(c.db).Conn(c)
}

// NewRaw creates a raw SQL query bound to this connection.
func (c Conn) NewRaw(query string, args ...any) *RawQuery {
	return NewRawQuery(c.db, query, args...).Conn(c)
}

// NewCreateTable creates a CREATE TABLE query bound to this connection.
func (c Conn) NewCreateTable() *CreateTableQuery {
	return NewCreateTableQuery(c.db).Conn(c)
}

// NewDropTable creates a DROP TABLE query bound to this connection.
func (c Conn) NewDropTable() *DropTableQuery {
	return NewDropTableQuery(c.db).Conn(c)
}

// NewCreateIndex creates a CREATE INDEX query bound to this connection.
func (c Conn) NewCreateIndex() *CreateIndexQuery {
	return NewCreateIndexQuery(c.db).Conn(c)
}

// NewDropIndex creates a DROP INDEX query bound to this connection.
func (c Conn) NewDropIndex() *DropIndexQuery {
	return NewDropIndexQuery(c.db).Conn(c)
}

// NewTruncateTable creates a TRUNCATE TABLE query bound to this connection.
func (c Conn) NewTruncateTable() *TruncateTableQuery {
	return NewTruncateTableQuery(c.db).Conn(c)
}

// NewAddColumn creates an ALTER TABLE ADD COLUMN query bound to this connection.
func (c Conn) NewAddColumn() *AddColumnQuery {
	return NewAddColumnQuery(c.db).Conn(c)
}

// NewDropColumn creates an ALTER TABLE DROP COLUMN query bound to this connection.
func (c Conn) NewDropColumn() *DropColumnQuery {
	return NewDropColumnQuery(c.db).Conn(c)
}

// RunInTx runs the function in a transaction. If the function returns an error,
// the transaction is rolled back. Otherwise, the transaction is committed.
func (c Conn) RunInTx(
	ctx context.Context, opts *sql.TxOptions, fn func(ctx context.Context, tx Tx) error,
) error {
	tx, err := c.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	var done bool

	defer func() {
		if !done {
			_ = tx.Rollback()
		}
	}()

	if err := fn(ctx, tx); err != nil {
		return err
	}

	done = true
	return tx.Commit()
}

// BeginTx starts a transaction on this connection with the given options.
func (c Conn) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	ctx, event := c.db.beforeQuery(ctx, nil, "BEGIN", nil, "BEGIN", nil)
	tx, err := c.Conn.BeginTx(ctx, opts)
	c.db.afterQuery(ctx, event, nil, err)
	if err != nil {
		return Tx{}, err
	}
	return Tx{
		ctx: ctx,
		db:  c.db,
		Tx:  tx,
	}, nil
}

//------------------------------------------------------------------------------

// Stmt wraps *sql.Stmt so prepared statements participate in Bun logging.
type Stmt struct {
	*sql.Stmt
}

// Prepare creates a prepared statement using a background context.
func (db *DB) Prepare(query string) (Stmt, error) {
	return db.PrepareContext(context.Background(), query)
}

// PrepareContext creates a prepared statement for repeated execution.
func (db *DB) PrepareContext(ctx context.Context, query string) (Stmt, error) {
	stmt, err := db.DB.PrepareContext(ctx, query)
	if err != nil {
		return Stmt{}, err
	}
	return Stmt{Stmt: stmt}, nil
}

//------------------------------------------------------------------------------

// Tx wraps *sql.Tx and preserves Bun-specific context such as hooks and dialect.
type Tx struct {
	ctx context.Context
	db  *DB
	// name is the name of a savepoint
	name string
	*sql.Tx
}

// RunInTx runs the function in a transaction. If the function returns an error,
// the transaction is rolled back. Otherwise, the transaction is committed.
func (db *DB) RunInTx(
	ctx context.Context, opts *sql.TxOptions, fn func(ctx context.Context, tx Tx) error,
) error {
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	var done bool

	defer func() {
		if !done {
			_ = tx.Rollback()
		}
	}()

	if err := fn(ctx, tx); err != nil {
		return err
	}

	done = true
	return tx.Commit()
}

// Begin starts a transaction with default options using a background context.
func (db *DB) Begin() (Tx, error) {
	return db.BeginTx(context.Background(), nil)
}

// BeginTx starts a transaction with the given options.
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	ctx, event := db.beforeQuery(ctx, nil, "BEGIN", nil, "BEGIN", nil)
	tx, err := db.DB.BeginTx(ctx, opts)
	db.afterQuery(ctx, event, nil, err)
	if err != nil {
		return Tx{}, err
	}
	return Tx{
		ctx: ctx,
		db:  db,
		Tx:  tx,
	}, nil
}

// Commit commits the transaction or releases the savepoint if this is a nested transaction.
func (tx Tx) Commit() error {
	if tx.name == "" {
		return tx.commitTX()
	}
	return tx.commitSP()
}

func (tx Tx) commitTX() error {
	ctx, event := tx.db.beforeQuery(tx.ctx, nil, "COMMIT", nil, "COMMIT", nil)
	err := tx.Tx.Commit()
	tx.db.afterQuery(ctx, event, nil, err)
	return err
}

func (tx Tx) commitSP() error {
	if tx.db.HasFeature(feature.MSSavepoint) {
		return nil
	}
	query := "RELEASE SAVEPOINT " + tx.name
	_, err := tx.ExecContext(tx.ctx, query)
	return err
}

// Rollback rolls back the transaction or rolls back to the savepoint if this is a nested transaction.
func (tx Tx) Rollback() error {
	if tx.name == "" {
		return tx.rollbackTX()
	}
	return tx.rollbackSP()
}

func (tx Tx) rollbackTX() error {
	ctx, event := tx.db.beforeQuery(tx.ctx, nil, "ROLLBACK", nil, "ROLLBACK", nil)
	err := tx.Tx.Rollback()
	tx.db.afterQuery(ctx, event, nil, err)
	return err
}

func (tx Tx) rollbackSP() error {
	query := "ROLLBACK TO SAVEPOINT " + tx.name
	if tx.db.HasFeature(feature.MSSavepoint) {
		query = "ROLLBACK TRANSACTION " + tx.name
	}
	_, err := tx.ExecContext(tx.ctx, query)
	return err
}

// Exec executes a query without returning rows within this transaction.
func (tx Tx) Exec(query string, args ...any) (sql.Result, error) {
	return tx.ExecContext(context.TODO(), query, args...)
}

// ExecContext executes a query without returning rows within this transaction.
func (tx Tx) ExecContext(
	ctx context.Context, query string, args ...any,
) (sql.Result, error) {
	formattedQuery := tx.db.format(query, args)
	ctx, event := tx.db.beforeQuery(ctx, nil, query, args, formattedQuery, nil)
	res, err := tx.Tx.ExecContext(ctx, formattedQuery)
	tx.db.afterQuery(ctx, event, res, err)
	return res, err
}

// Query executes a query returning rows within this transaction.
func (tx Tx) Query(query string, args ...any) (*sql.Rows, error) {
	return tx.QueryContext(context.TODO(), query, args...)
}

// QueryContext executes a query returning rows within this transaction.
func (tx Tx) QueryContext(
	ctx context.Context, query string, args ...any,
) (*sql.Rows, error) {
	formattedQuery := tx.db.format(query, args)
	ctx, event := tx.db.beforeQuery(ctx, nil, query, args, formattedQuery, nil)
	rows, err := tx.Tx.QueryContext(ctx, formattedQuery)
	tx.db.afterQuery(ctx, event, nil, err)
	return rows, err
}

// QueryRow executes a query expected to return at most one row within this transaction.
func (tx Tx) QueryRow(query string, args ...any) *sql.Row {
	return tx.QueryRowContext(context.TODO(), query, args...)
}

// QueryRowContext executes a query expected to return at most one row within this transaction.
func (tx Tx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	formattedQuery := tx.db.format(query, args)
	ctx, event := tx.db.beforeQuery(ctx, nil, query, args, formattedQuery, nil)
	row := tx.Tx.QueryRowContext(ctx, formattedQuery)
	tx.db.afterQuery(ctx, event, nil, row.Err())
	return row
}

//------------------------------------------------------------------------------

// Begin creates a savepoint, effectively starting a nested transaction.
func (tx Tx) Begin() (Tx, error) {
	return tx.BeginTx(tx.ctx, nil)
}

// BeginTx will save a point in the running transaction.
func (tx Tx) BeginTx(ctx context.Context, _ *sql.TxOptions) (Tx, error) {
	// mssql savepoint names are limited to 32 characters
	sp := make([]byte, 14)
	_, err := cryptorand.Read(sp)
	if err != nil {
		return Tx{}, err
	}

	qName := "SP_" + hex.EncodeToString(sp)
	query := "SAVEPOINT " + qName
	if tx.db.HasFeature(feature.MSSavepoint) {
		query = "SAVE TRANSACTION " + qName
	}
	_, err = tx.ExecContext(ctx, query)
	if err != nil {
		return Tx{}, err
	}
	return Tx{
		ctx:  ctx,
		db:   tx.db,
		Tx:   tx.Tx,
		name: qName,
	}, nil
}

// RunInTx creates a savepoint and runs the function within that savepoint.
// If the function returns an error, the savepoint is rolled back.
func (tx Tx) RunInTx(
	ctx context.Context, _ *sql.TxOptions, fn func(ctx context.Context, tx Tx) error,
) error {
	sp, err := tx.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	var done bool

	defer func() {
		if !done {
			_ = sp.Rollback()
		}
	}()

	if err := fn(ctx, sp); err != nil {
		return err
	}

	done = true
	return sp.Commit()
}

// Dialect returns the database dialect for this transaction.
func (tx Tx) Dialect() schema.Dialect {
	return tx.db.Dialect()
}

// NewValues creates a VALUES query bound to this transaction.
func (tx Tx) NewValues(model any) *ValuesQuery {
	return NewValuesQuery(tx.db, model).Conn(tx)
}

// NewMerge creates a MERGE query bound to this transaction.
func (tx Tx) NewMerge() *MergeQuery {
	return NewMergeQuery(tx.db).Conn(tx)
}

// NewSelect creates a SELECT query bound to this transaction.
func (tx Tx) NewSelect() *SelectQuery {
	return NewSelectQuery(tx.db).Conn(tx)
}

// NewInsert creates an INSERT query bound to this transaction.
func (tx Tx) NewInsert() *InsertQuery {
	return NewInsertQuery(tx.db).Conn(tx)
}

// NewUpdate creates an UPDATE query bound to this transaction.
func (tx Tx) NewUpdate() *UpdateQuery {
	return NewUpdateQuery(tx.db).Conn(tx)
}

// NewDelete creates a DELETE query bound to this transaction.
func (tx Tx) NewDelete() *DeleteQuery {
	return NewDeleteQuery(tx.db).Conn(tx)
}

// NewRaw creates a raw SQL query bound to this transaction.
func (tx Tx) NewRaw(query string, args ...any) *RawQuery {
	return NewRawQuery(tx.db, query, args...).Conn(tx)
}

// NewCreateTable creates a CREATE TABLE query bound to this transaction.
func (tx Tx) NewCreateTable() *CreateTableQuery {
	return NewCreateTableQuery(tx.db).Conn(tx)
}

// NewDropTable creates a DROP TABLE query bound to this transaction.
func (tx Tx) NewDropTable() *DropTableQuery {
	return NewDropTableQuery(tx.db).Conn(tx)
}

// NewCreateIndex creates a CREATE INDEX query bound to this transaction.
func (tx Tx) NewCreateIndex() *CreateIndexQuery {
	return NewCreateIndexQuery(tx.db).Conn(tx)
}

// NewDropIndex creates a DROP INDEX query bound to this transaction.
func (tx Tx) NewDropIndex() *DropIndexQuery {
	return NewDropIndexQuery(tx.db).Conn(tx)
}

// NewTruncateTable creates a TRUNCATE TABLE query bound to this transaction.
func (tx Tx) NewTruncateTable() *TruncateTableQuery {
	return NewTruncateTableQuery(tx.db).Conn(tx)
}

// NewAddColumn creates an ALTER TABLE ADD COLUMN query bound to this transaction.
func (tx Tx) NewAddColumn() *AddColumnQuery {
	return NewAddColumnQuery(tx.db).Conn(tx)
}

// NewDropColumn creates an ALTER TABLE DROP COLUMN query bound to this transaction.
func (tx Tx) NewDropColumn() *DropColumnQuery {
	return NewDropColumnQuery(tx.db).Conn(tx)
}

func (db *DB) makeQueryBytes() []byte {
	return internal.MakeQueryBytes()
}
