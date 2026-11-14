package bun

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

const (
	forceDeleteFlag internal.Flag = 1 << iota
	deletedFlag
	allWithDeletedFlag
)

// WithQuery defines a common table expression used by another query.
type WithQuery struct {
	name            string
	query           Query
	recursive       bool
	materialized    bool
	notMaterialized bool
}

// IConn is a common interface for *sql.DB, *sql.Conn, and *sql.Tx.
type IConn interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var (
	_ IConn = (*sql.DB)(nil)
	_ IConn = (*sql.Conn)(nil)
	_ IConn = (*sql.Tx)(nil)
	_ IConn = (*DB)(nil)
	_ IConn = (*Conn)(nil)
	_ IConn = (*Tx)(nil)
)

// IDB is a common interface for *bun.DB, bun.Conn, and bun.Tx.
type IDB interface {
	IConn
	Dialect() schema.Dialect

	NewValues(model any) *ValuesQuery
	NewSelect() *SelectQuery
	NewInsert() *InsertQuery
	NewUpdate() *UpdateQuery
	NewDelete() *DeleteQuery
	NewMerge() *MergeQuery
	NewRaw(query string, args ...any) *RawQuery
	NewCreateTable() *CreateTableQuery
	NewDropTable() *DropTableQuery
	NewCreateIndex() *CreateIndexQuery
	NewDropIndex() *DropIndexQuery
	NewTruncateTable() *TruncateTableQuery
	NewAddColumn() *AddColumnQuery
	NewDropColumn() *DropColumnQuery

	BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)
	RunInTx(ctx context.Context, opts *sql.TxOptions, f func(ctx context.Context, tx Tx) error) error
}

var (
	_ IDB = (*DB)(nil)
	_ IDB = (*Conn)(nil)
	_ IDB = (*Tx)(nil)
)

// QueryBuilder exposes shared filtering helpers across query builders.
type QueryBuilder interface {
	Query
	Where(query string, args ...any) QueryBuilder
	WhereGroup(sep string, fn func(QueryBuilder) QueryBuilder) QueryBuilder
	WhereOr(query string, args ...any) QueryBuilder
	WhereDeleted() QueryBuilder
	WhereAllWithDeleted() QueryBuilder
	WherePK(cols ...string) QueryBuilder
	Unwrap() any
}

var (
	_ QueryBuilder = (*selectQueryBuilder)(nil)
	_ QueryBuilder = (*updateQueryBuilder)(nil)
	_ QueryBuilder = (*deleteQueryBuilder)(nil)
)

type baseQuery struct {
	db   *DB
	conn IConn

	model Model
	err   error

	tableModel TableModel
	table      *schema.Table

	with           []WithQuery
	modelTableName schema.QueryWithArgs
	tables         []schema.QueryWithArgs
	columns        []schema.QueryWithArgs

	flags internal.Flag
}

func (q *baseQuery) DB() *DB {
	return q.db
}

func (q *baseQuery) resolveConn(ctx context.Context, query Query) IConn {
	if q.conn != nil {
		return q.conn
	}
	if q.db.resolver != nil {
		if conn := q.db.resolver.ResolveConn(ctx, query); conn != nil {
			return conn
		}
	}
	return q.db.DB
}

func (q *baseQuery) GetModel() Model {
	return q.model
}

func (q *baseQuery) GetTableName() string {
	if q.table != nil {
		return q.table.Name
	}

	for _, wq := range q.with {
		if model := wq.query.GetModel(); model != nil {
			return wq.query.GetTableName()
		}
	}

	if q.modelTableName.Query != "" {
		return q.modelTableName.Query
	}

	if len(q.tables) > 0 {
		b, _ := q.tables[0].AppendQuery(q.db.gen, nil)
		if len(b) < 64 {
			return string(b)
		}
	}

	return ""
}

func (q *baseQuery) setConn(db IConn) {
	// Unwrap Bun wrappers to not call query hooks twice.
	switch db := db.(type) {
	case *DB:
		q.conn = db.DB
	case Conn:
		q.conn = db.Conn
	case Tx:
		q.conn = db.Tx
	default:
		q.conn = db
	}
}

func (q *baseQuery) setModel(modeli any) {
	model, err := newSingleModel(q.db, modeli)
	if err != nil {
		q.setErr(err)
		return
	}

	q.model = model
	if tm, ok := model.(TableModel); ok {
		q.tableModel = tm
		q.table = tm.Table()
	}
}

func (q *baseQuery) setErr(err error) {
	if q.err == nil {
		q.err = err
	}
}

func (q *baseQuery) getModel(dest []any) (Model, error) {
	if len(dest) > 0 {
		return newModel(q.db, dest)
	}
	if q.model != nil {
		return q.model, nil
	}
	return nil, errNilModel
}

func (q *baseQuery) beforeAppendModel(ctx context.Context, query Query) error {
	if q.tableModel != nil {
		return q.tableModel.BeforeAppendModel(ctx, query)
	}
	return nil
}

func (q *baseQuery) hasFeature(feature feature.Feature) bool {
	return q.db.HasFeature(feature)
}

//------------------------------------------------------------------------------

func (q *baseQuery) checkSoftDelete() error {
	if q.table == nil {
		return errors.New("bun: can't use soft deletes without a table")
	}
	if q.table.SoftDeleteField == nil {
		return fmt.Errorf("%s does not have a soft delete field", q.table)
	}
	if q.tableModel == nil {
		return errors.New("bun: can't use soft deletes without a table model")
	}
	return nil
}

// Deleted adds `WHERE deleted_at IS NOT NULL` clause for soft deleted models.
func (q *baseQuery) whereDeleted() {
	if err := q.checkSoftDelete(); err != nil {
		q.setErr(err)
		return
	}
	q.flags = q.flags.Set(deletedFlag)
	q.flags = q.flags.Remove(allWithDeletedFlag)
}

// AllWithDeleted changes query to return all rows including soft deleted ones.
func (q *baseQuery) whereAllWithDeleted() {
	if err := q.checkSoftDelete(); err != nil {
		q.setErr(err)
		return
	}
	q.flags = q.flags.Set(allWithDeletedFlag).Remove(deletedFlag)
}

func (q *baseQuery) isSoftDelete() bool {
	if q.table != nil {
		return q.table.SoftDeleteField != nil &&
			!q.flags.Has(allWithDeletedFlag) &&
			(!q.flags.Has(forceDeleteFlag) || q.flags.Has(deletedFlag))
	}
	return false
}

//------------------------------------------------------------------------------

// NewWithQuery creates a CTE with the given name and underlying query.
func NewWithQuery(name string, query Query) *WithQuery {
	return &WithQuery{
		name:  name,
		query: query,
	}
}

// Recursive marks the CTE as recursive, enabling self-referential queries.
func (q *WithQuery) Recursive() *WithQuery {
	q.recursive = true
	return q
}

// Materialized hints the planner to evaluate the CTE once and store the results.
func (q *WithQuery) Materialized() *WithQuery {
	q.materialized = true
	return q
}

// NotMaterialized hints the planner to inline the CTE, potentially re-evaluating it.
func (q *WithQuery) NotMaterialized() *WithQuery {
	q.notMaterialized = true
	return q
}

func (q *baseQuery) addWith(query *WithQuery) {
	if query == nil {
		return
	}
	q.with = append(q.with, *query)
}

func (q *baseQuery) appendWith(gen schema.QueryGen, b []byte) (_ []byte, err error) {
	if len(q.with) == 0 {
		return b, nil
	}

	b = append(b, "WITH "...)
	for i, with := range q.with {
		if i > 0 {
			b = append(b, ", "...)
		}

		if with.recursive {
			b = append(b, "RECURSIVE "...)
		}

		b, err = q.appendCTE(gen, b, with)
		if err != nil {
			return nil, err
		}
	}
	b = append(b, ' ')
	return b, nil
}

func (q *baseQuery) appendCTE(
	gen schema.QueryGen, b []byte, cte WithQuery,
) (_ []byte, err error) {
	if !gen.Dialect().Features().Has(feature.WithValues) {
		if values, ok := cte.query.(*ValuesQuery); ok {
			return q.appendSelectFromValues(gen, b, cte, values)
		}
	}

	b = gen.AppendIdent(b, cte.name)

	if q, ok := cte.query.(schema.ColumnsAppender); ok {
		b = append(b, " ("...)
		b, err = q.AppendColumns(gen, b)
		if err != nil {
			return nil, err
		}
		b = append(b, ")"...)
	}

	switch {
	case cte.materialized && gen.Dialect().Name() == dialect.PG:
		b = append(b, " AS MATERIALIZED ("...)
	case cte.notMaterialized && gen.Dialect().Name() == dialect.PG:
		b = append(b, " AS NOT MATERIALIZED ("...)
	default:
		b = append(b, " AS ("...)
	}

	b, err = cte.query.AppendQuery(gen, b)
	if err != nil {
		return nil, err
	}

	b = append(b, ")"...)
	return b, nil
}

func (q *baseQuery) appendSelectFromValues(
	gen schema.QueryGen, b []byte, cte WithQuery, values *ValuesQuery,
) (_ []byte, err error) {
	b = gen.AppendIdent(b, cte.name)
	b = append(b, " AS (SELECT * FROM ("...)

	b, err = cte.query.AppendQuery(gen, b)
	if err != nil {
		return nil, err
	}

	b = append(b, ") AS t"...)
	if q, ok := cte.query.(schema.ColumnsAppender); ok {
		b = append(b, " ("...)
		b, err = q.AppendColumns(gen, b)
		if err != nil {
			return nil, err
		}
		b = append(b, ")"...)
	}
	b = append(b, ")"...)

	return b, nil
}

//------------------------------------------------------------------------------

func (q *baseQuery) addTable(table schema.QueryWithArgs) {
	q.tables = append(q.tables, table)
}

func (q *baseQuery) addColumn(column schema.QueryWithArgs) {
	q.columns = append(q.columns, column)
}

func (q *baseQuery) excludeColumn(columns []string) {
	if q.table == nil {
		q.setErr(errNilModel)
		return
	}

	if q.columns == nil {
		for _, f := range q.table.Fields {
			q.columns = append(q.columns, schema.UnsafeIdent(f.Name))
		}
	}

	if len(columns) == 1 && columns[0] == "*" {
		q.columns = make([]schema.QueryWithArgs, 0)
		return
	}

	for _, column := range columns {
		if !q._excludeColumn(column) {
			q.setErr(fmt.Errorf("bun: can't find column=%q", column))
			return
		}
	}
}

func (q *baseQuery) _excludeColumn(column string) bool {
	for i, col := range q.columns {
		if col.Args == nil && col.Query == column {
			q.columns = append(q.columns[:i], q.columns[i+1:]...)
			return true
		}
	}
	return false
}

//------------------------------------------------------------------------------

func (q *baseQuery) modelHasTableName() bool {
	if !q.modelTableName.IsZero() {
		return q.modelTableName.Query != ""
	}
	return q.table != nil
}

func (q *baseQuery) hasTables() bool {
	return q.modelHasTableName() || len(q.tables) > 0
}

func (q *baseQuery) appendTables(
	gen schema.QueryGen, b []byte,
) (_ []byte, err error) {
	return q._appendTables(gen, b, false)
}

func (q *baseQuery) appendTablesWithAlias(
	gen schema.QueryGen, b []byte,
) (_ []byte, err error) {
	return q._appendTables(gen, b, true)
}

func (q *baseQuery) _appendTables(
	gen schema.QueryGen, b []byte, withAlias bool,
) (_ []byte, err error) {
	startLen := len(b)

	if q.modelHasTableName() {
		if !q.modelTableName.IsZero() {
			b, err = q.modelTableName.AppendQuery(gen, b)
			if err != nil {
				return nil, err
			}
		} else {
			b = gen.AppendQuery(b, string(q.table.SQLNameForSelects))
			if withAlias && q.table.SQLAlias != q.table.SQLNameForSelects {
				if q.db.dialect.Name() == dialect.Oracle {
					b = append(b, ' ')
				} else {
					b = append(b, " AS "...)
				}
				b = append(b, q.table.SQLAlias...)
			}
		}
	}

	for _, table := range q.tables {
		if len(b) > startLen {
			b = append(b, ", "...)
		}
		b, err = table.AppendQuery(gen, b)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (q *baseQuery) appendFirstTable(gen schema.QueryGen, b []byte) ([]byte, error) {
	return q._appendFirstTable(gen, b, false)
}

func (q *baseQuery) appendFirstTableWithAlias(
	gen schema.QueryGen, b []byte,
) ([]byte, error) {
	return q._appendFirstTable(gen, b, true)
}

func (q *baseQuery) _appendFirstTable(
	gen schema.QueryGen, b []byte, withAlias bool,
) ([]byte, error) {
	if !q.modelTableName.IsZero() {
		return q.modelTableName.AppendQuery(gen, b)
	}

	if q.table != nil {
		b = gen.AppendQuery(b, string(q.table.SQLName))
		if withAlias {
			b = append(b, " AS "...)
			b = append(b, q.table.SQLAlias...)
		}
		return b, nil
	}

	if len(q.tables) > 0 {
		return q.tables[0].AppendQuery(gen, b)
	}

	return nil, errors.New("bun: query does not have a table")
}

func (q *baseQuery) hasMultiTables() bool {
	if q.modelHasTableName() {
		return len(q.tables) >= 1
	}
	return len(q.tables) >= 2
}

func (q *baseQuery) appendOtherTables(gen schema.QueryGen, b []byte) (_ []byte, err error) {
	tables := q.tables
	if !q.modelHasTableName() {
		tables = tables[1:]
	}
	for i, table := range tables {
		if i > 0 {
			b = append(b, ", "...)
		}
		b, err = table.AppendQuery(gen, b)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

//------------------------------------------------------------------------------

func (q *baseQuery) appendColumns(gen schema.QueryGen, b []byte) (_ []byte, err error) {
	for i, f := range q.columns {
		if i > 0 {
			b = append(b, ", "...)
		}
		b, err = f.AppendQuery(gen, b)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (q *baseQuery) getFields() ([]*schema.Field, error) {
	if len(q.columns) == 0 {
		if q.table == nil {
			return nil, errNilModel
		}
		return q.table.Fields, nil
	}
	return q._getFields(false)
}

func (q *baseQuery) getDataFields() ([]*schema.Field, error) {
	if len(q.columns) == 0 {
		if q.table == nil {
			return nil, errNilModel
		}
		return q.table.DataFields, nil
	}
	return q._getFields(true)
}

func (q *baseQuery) _getFields(omitPK bool) ([]*schema.Field, error) {
	fields := make([]*schema.Field, 0, len(q.columns))
	for _, col := range q.columns {
		if col.Args != nil {
			continue
		}

		field, err := q.table.Field(col.Query)
		if err != nil {
			return nil, err
		}

		if omitPK && field.IsPK {
			continue
		}

		fields = append(fields, field)
	}
	return fields, nil
}

func (q *baseQuery) scan(
	ctx context.Context,
	iquery Query,
	query string,
	model Model,
	hasDest bool,
) (sql.Result, error) {
	ctx, event := q.db.beforeQuery(ctx, iquery, query, nil, query, q.model)
	res, err := q._scan(ctx, iquery, query, model, hasDest)
	q.db.afterQuery(ctx, event, res, err)
	return res, err
}

func (q *baseQuery) _scan(
	ctx context.Context,
	iquery Query,
	query string,
	model Model,
	hasDest bool,
) (sql.Result, error) {
	rows, err := q.resolveConn(ctx, iquery).QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	numRow, err := model.ScanRows(ctx, rows)
	if err != nil {
		return nil, err
	}

	if numRow == 0 && hasDest && isSingleRowModel(model) {
		return nil, sql.ErrNoRows
	}
	return driver.RowsAffected(numRow), nil
}

func (q *baseQuery) exec(
	ctx context.Context,
	iquery Query,
	query string,
) (sql.Result, error) {
	ctx, event := q.db.beforeQuery(ctx, iquery, query, nil, query, q.model)
	res, err := q.resolveConn(ctx, iquery).ExecContext(ctx, query)
	q.db.afterQuery(ctx, event, res, err)
	return res, err
}

//------------------------------------------------------------------------------

func (q *baseQuery) AppendNamedArg(gen schema.QueryGen, b []byte, name string) ([]byte, bool) {
	if q.table == nil {
		return b, false
	}

	if m, ok := q.tableModel.(*structTableModel); ok {
		if b, ok := m.AppendNamedArg(gen, b, name); ok {
			return b, ok
		}
	}

	switch name {
	case "TableName":
		b = gen.AppendQuery(b, string(q.table.SQLName))
		return b, true
	case "TableAlias":
		b = gen.AppendQuery(b, string(q.table.SQLAlias))
		return b, true
	case "PKs":
		b = appendColumns(b, "", q.table.PKs)
		return b, true
	case "TablePKs":
		b = appendColumns(b, q.table.SQLAlias, q.table.PKs)
		return b, true
	case "Columns":
		b = appendColumns(b, "", q.table.Fields)
		return b, true
	case "TableColumns":
		b = appendColumns(b, q.table.SQLAlias, q.table.Fields)
		return b, true
	}

	return b, false
}

//------------------------------------------------------------------------------

func (q *baseQuery) Dialect() schema.Dialect {
	return q.db.Dialect()
}

func (q *baseQuery) NewValues(model any) *ValuesQuery {
	return NewValuesQuery(q.db, model).Conn(q.conn)
}

func (q *baseQuery) NewSelect() *SelectQuery {
	return NewSelectQuery(q.db).Conn(q.conn)
}

func (q *baseQuery) NewInsert() *InsertQuery {
	return NewInsertQuery(q.db).Conn(q.conn)
}

func (q *baseQuery) NewUpdate() *UpdateQuery {
	return NewUpdateQuery(q.db).Conn(q.conn)
}

func (q *baseQuery) NewDelete() *DeleteQuery {
	return NewDeleteQuery(q.db).Conn(q.conn)
}

func (q *baseQuery) NewRaw(query string, args ...any) *RawQuery {
	return NewRawQuery(q.db, query, args...).Conn(q.conn)
}

func (q *baseQuery) NewCreateTable() *CreateTableQuery {
	return NewCreateTableQuery(q.db).Conn(q.conn)
}

func (q *baseQuery) NewDropTable() *DropTableQuery {
	return NewDropTableQuery(q.db).Conn(q.conn)
}

func (q *baseQuery) NewCreateIndex() *CreateIndexQuery {
	return NewCreateIndexQuery(q.db).Conn(q.conn)
}

func (q *baseQuery) NewDropIndex() *DropIndexQuery {
	return NewDropIndexQuery(q.db).Conn(q.conn)
}

func (q *baseQuery) NewTruncateTable() *TruncateTableQuery {
	return NewTruncateTableQuery(q.db).Conn(q.conn)
}

func (q *baseQuery) NewAddColumn() *AddColumnQuery {
	return NewAddColumnQuery(q.db).Conn(q.conn)
}

func (q *baseQuery) NewDropColumn() *DropColumnQuery {
	return NewDropColumnQuery(q.db).Conn(q.conn)
}

//------------------------------------------------------------------------------

func appendColumns(b []byte, table schema.Safe, fields []*schema.Field) []byte {
	for i, f := range fields {
		if i > 0 {
			b = append(b, ", "...)
		}

		if len(table) > 0 {
			b = append(b, table...)
			b = append(b, '.')
		}
		b = append(b, f.SQLName...)
	}
	return b
}

func formatterWithModel(gen schema.QueryGen, model schema.NamedArgAppender) schema.QueryGen {
	if gen.IsNop() {
		return gen
	}
	return gen.WithArg(model)
}

//------------------------------------------------------------------------------

type whereBaseQuery struct {
	baseQuery

	where       []schema.QueryWithSep
	whereFields []*schema.Field
}

func (q *whereBaseQuery) addWhere(where schema.QueryWithSep) {
	q.where = append(q.where, where)
}

func (q *whereBaseQuery) addWhereGroup(sep string, where []schema.QueryWithSep) {
	if len(where) == 0 {
		return
	}

	q.addWhere(schema.SafeQueryWithSep("", nil, sep))
	q.addWhere(schema.SafeQueryWithSep("", nil, "("))

	where[0].Sep = ""
	q.where = append(q.where, where...)

	q.addWhere(schema.SafeQueryWithSep("", nil, ")"))
}

func (q *whereBaseQuery) addWhereCols(cols []string) {
	if q.table == nil {
		err := fmt.Errorf("bun: got %T, but WherePK requires a struct or slice-based model", q.model)
		q.setErr(err)
		return
	}
	if q.whereFields != nil {
		err := errors.New("bun: WherePK can only be called once")
		q.setErr(err)
		return
	}

	if cols == nil {
		if err := q.table.CheckPKs(); err != nil {
			q.setErr(err)
			return
		}
		q.whereFields = q.table.PKs
		return
	}

	q.whereFields = make([]*schema.Field, len(cols))
	for i, col := range cols {
		field, err := q.table.Field(col)
		if err != nil {
			q.setErr(err)
			return
		}
		q.whereFields[i] = field
	}
}

func (q *whereBaseQuery) mustAppendWhere(
	gen schema.QueryGen, b []byte, withAlias bool,
) ([]byte, error) {
	if len(q.where) == 0 && q.whereFields == nil && !q.flags.Has(deletedFlag) {
		err := errors.New("bun: Update and Delete queries require at least one Where")
		return nil, err
	}
	return q.appendWhere(gen, b, withAlias)
}

func (q *whereBaseQuery) appendWhere(
	gen schema.QueryGen, b []byte, withAlias bool,
) (_ []byte, err error) {
	if len(q.where) == 0 && q.whereFields == nil && !q.isSoftDelete() {
		return b, nil
	}

	b = append(b, " WHERE "...)
	startLen := len(b)

	if len(q.where) > 0 {
		b, err = appendWhere(gen, b, q.where)
		if err != nil {
			return nil, err
		}
	}

	if q.isSoftDelete() {
		if len(b) > startLen {
			b = append(b, " AND "...)
		}

		if withAlias {
			b = append(b, q.tableModel.Table().SQLAlias...)
		} else {
			b = append(b, q.tableModel.Table().SQLName...)
		}
		b = append(b, '.')

		field := q.tableModel.Table().SoftDeleteField
		b = append(b, field.SQLName...)

		if field.IsPtr || field.NullZero {
			if q.flags.Has(deletedFlag) {
				b = append(b, " IS NOT NULL"...)
			} else {
				b = append(b, " IS NULL"...)
			}
		} else {
			if q.flags.Has(deletedFlag) {
				b = append(b, " != "...)
			} else {
				b = append(b, " = "...)
			}
			b = gen.Dialect().AppendTime(b, time.Time{})
		}
	}

	if q.whereFields != nil {
		if len(b) > startLen {
			b = append(b, " AND "...)
		}
		b, err = q.appendWhereFields(gen, b, q.whereFields, withAlias)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

func appendWhere(
	gen schema.QueryGen, b []byte, where []schema.QueryWithSep,
) (_ []byte, err error) {
	for i, where := range where {
		if i > 0 {
			b = append(b, where.Sep...)
		}

		if where.Query == "" {
			continue
		}

		b = append(b, '(')
		b, err = where.AppendQuery(gen, b)
		if err != nil {
			return nil, err
		}
		b = append(b, ')')
	}
	return b, nil
}

func (q *whereBaseQuery) appendWhereFields(
	gen schema.QueryGen, b []byte, fields []*schema.Field, withAlias bool,
) (_ []byte, err error) {
	if q.table == nil {
		err := fmt.Errorf("bun: got %T, but WherePK requires struct or slice-based model", q.model)
		return nil, err
	}

	switch model := q.tableModel.(type) {
	case *structTableModel:
		return q.appendWhereStructFields(gen, b, model, fields, withAlias)
	case *sliceTableModel:
		return q.appendWhereSliceFields(gen, b, model, fields, withAlias)
	default:
		return nil, fmt.Errorf("bun: WhereColumn does not support %T", q.tableModel)
	}
}

func (q *whereBaseQuery) appendWhereStructFields(
	gen schema.QueryGen,
	b []byte,
	model *structTableModel,
	fields []*schema.Field,
	withAlias bool,
) (_ []byte, err error) {
	if !model.strct.IsValid() {
		return nil, errNilModel
	}

	isTemplate := gen.IsNop()
	b = append(b, '(')
	for i, f := range fields {
		if i > 0 {
			b = append(b, " AND "...)
		}
		if withAlias {
			b = append(b, q.table.SQLAlias...)
			b = append(b, '.')
		}
		b = append(b, f.SQLName...)
		b = append(b, " = "...)
		if isTemplate {
			b = append(b, '?')
		} else {
			b = f.AppendValue(gen, b, model.strct)
		}
	}
	b = append(b, ')')
	return b, nil
}

func (q *whereBaseQuery) appendWhereSliceFields(
	gen schema.QueryGen,
	b []byte,
	model *sliceTableModel,
	fields []*schema.Field,
	withAlias bool,
) (_ []byte, err error) {
	if len(fields) > 1 {
		b = append(b, '(')
	}
	if withAlias {
		b = appendColumns(b, q.table.SQLAlias, fields)
	} else {
		b = appendColumns(b, "", fields)
	}
	if len(fields) > 1 {
		b = append(b, ')')
	}

	b = append(b, " IN ("...)

	isTemplate := gen.IsNop()
	slice := model.slice
	sliceLen := slice.Len()
	for i := 0; i < sliceLen; i++ {
		if i > 0 {
			if isTemplate {
				break
			}
			b = append(b, ", "...)
		}

		el := indirect(slice.Index(i))

		if len(fields) > 1 {
			b = append(b, '(')
		}
		for i, f := range fields {
			if i > 0 {
				b = append(b, ", "...)
			}
			if isTemplate {
				b = append(b, '?')
			} else {
				b = f.AppendValue(gen, b, el)
			}
		}
		if len(fields) > 1 {
			b = append(b, ')')
		}
	}

	b = append(b, ')')

	return b, nil
}

//------------------------------------------------------------------------------

type returningQuery struct {
	returning       []schema.QueryWithArgs
	returningFields []*schema.Field
}

func (q *returningQuery) addReturning(ret schema.QueryWithArgs) {
	q.returning = append(q.returning, ret)
}

func (q *returningQuery) addReturningField(field *schema.Field) {
	if len(q.returning) > 0 {
		return
	}
	for _, f := range q.returningFields {
		if f == field {
			return
		}
	}
	q.returningFields = append(q.returningFields, field)
}

func (q *returningQuery) appendReturning(
	gen schema.QueryGen, b []byte,
) (_ []byte, err error) {
	return q._appendReturning(gen, b, "")
}

func (q *returningQuery) appendOutput(
	gen schema.QueryGen, b []byte,
) (_ []byte, err error) {
	return q._appendReturning(gen, b, "INSERTED")
}

func (q *returningQuery) _appendReturning(
	gen schema.QueryGen, b []byte, table string,
) (_ []byte, err error) {
	for i, f := range q.returning {
		if i > 0 {
			b = append(b, ", "...)
		}
		b, err = f.AppendQuery(gen, b)
		if err != nil {
			return nil, err
		}
	}

	if len(q.returning) > 0 {
		return b, nil
	}

	b = appendColumns(b, schema.Safe(table), q.returningFields)
	return b, nil
}

func (q *returningQuery) hasReturning() bool {
	if len(q.returning) == 1 {
		if ret := q.returning[0]; len(ret.Args) == 0 {
			switch ret.Query {
			case "", "null", "NULL":
				return false
			}
		}
	}
	return len(q.returning) > 0 || len(q.returningFields) > 0
}

//------------------------------------------------------------------------------

type customValueQuery struct {
	modelValues map[string]schema.QueryWithArgs
	extraValues []columnValue
}

type columnValue struct {
	column string
	value  schema.QueryWithArgs
}

func (q *customValueQuery) addValue(
	table *schema.Table, column string, value string, args []any,
) {
	ok := false
	if table != nil {
		_, ok = table.FieldMap[column]
	}

	if ok {
		if q.modelValues == nil {
			q.modelValues = make(map[string]schema.QueryWithArgs)
		}
		q.modelValues[column] = schema.SafeQuery(value, args)
	} else {
		q.extraValues = append(q.extraValues, columnValue{
			column: column,
			value:  schema.SafeQuery(value, args),
		})
	}
}

//------------------------------------------------------------------------------

type setQuery struct {
	set       []schema.QueryWithArgs
	setValues *ValuesQuery
	omitZero  bool
	customValueQuery
}

func (q *setQuery) addSet(set schema.QueryWithArgs) {
	q.set = append(q.set, set)
}

func (q *setQuery) appendSet(gen schema.QueryGen, b []byte) (_ []byte, err error) {
	pos := len(b)

	if q.setValues != nil {
		b, err = q.setValues.appendSet(gen, b)
		if err != nil {
			return nil, err
		}
	}

	for _, query := range q.set {
		if len(b) > pos {
			b = append(b, ", "...)
		}
		b, err = query.AppendQuery(gen, b)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (q *setQuery) appendSetStruct(
	gen schema.QueryGen, b []byte, model *structTableModel, fields []*schema.Field,
) (_ []byte, err error) {
	defaultPlaceholder := gen.HasFeature(feature.DefaultPlaceholder)
	isTemplate := gen.IsNop()
	pos := len(b)
	for _, f := range fields {
		if f.SkipUpdate() {
			continue
		}

		app, hasValue := q.modelValues[f.Name]

		if !hasValue && q.omitZero && f.HasZeroValue(model.strct) {
			continue
		}

		if len(b) != pos {
			b = append(b, ", "...)
			pos = len(b)
		}

		b = append(b, f.SQLName...)
		b = append(b, " = "...)

		if isTemplate {
			b = append(b, '?')
			continue
		}

		if hasValue {
			b, err = app.AppendQuery(gen, b)
			if err != nil {
				return nil, err
			}
		} else if defaultPlaceholder {
			b = f.AppendValueOrDefault(gen, b, model.strct)
		} else {
			b = f.AppendValue(gen, b, model.strct)
		}
	}

	for i, v := range q.extraValues {
		if i > 0 || len(fields) > 0 {
			b = append(b, ", "...)
		}

		b = append(b, v.column...)
		b = append(b, " = "...)

		b, err = v.value.AppendQuery(gen, b)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

//------------------------------------------------------------------------------

type cascadeQuery struct {
	cascade  bool
	restrict bool
}

func (q cascadeQuery) appendCascade(gen schema.QueryGen, b []byte) []byte {
	if !gen.HasFeature(feature.TableCascade) {
		return b
	}
	if q.cascade {
		b = append(b, " CASCADE"...)
	}
	if q.restrict {
		b = append(b, " RESTRICT"...)
	}
	return b
}

//------------------------------------------------------------------------------

type idxHintsQuery struct {
	use    *indexHints
	ignore *indexHints
	force  *indexHints
}

type indexHints struct {
	names      []schema.QueryWithArgs
	forJoin    []schema.QueryWithArgs
	forOrderBy []schema.QueryWithArgs
	forGroupBy []schema.QueryWithArgs
}

func (ih *idxHintsQuery) lazyUse() *indexHints {
	if ih.use == nil {
		ih.use = new(indexHints)
	}
	return ih.use
}

func (ih *idxHintsQuery) lazyIgnore() *indexHints {
	if ih.ignore == nil {
		ih.ignore = new(indexHints)
	}
	return ih.ignore
}

func (ih *idxHintsQuery) lazyForce() *indexHints {
	if ih.force == nil {
		ih.force = new(indexHints)
	}
	return ih.force
}

func (ih *idxHintsQuery) appendIndexes(hints []schema.QueryWithArgs, indexes ...string) []schema.QueryWithArgs {
	for _, idx := range indexes {
		hints = append(hints, schema.UnsafeIdent(idx))
	}
	return hints
}

func (ih *idxHintsQuery) addUseIndex(indexes ...string) {
	if len(indexes) == 0 {
		return
	}
	ih.lazyUse().names = ih.appendIndexes(ih.use.names, indexes...)
}

func (ih *idxHintsQuery) addUseIndexForJoin(indexes ...string) {
	if len(indexes) == 0 {
		return
	}
	ih.lazyUse().forJoin = ih.appendIndexes(ih.use.forJoin, indexes...)
}

func (ih *idxHintsQuery) addUseIndexForOrderBy(indexes ...string) {
	if len(indexes) == 0 {
		return
	}
	ih.lazyUse().forOrderBy = ih.appendIndexes(ih.use.forOrderBy, indexes...)
}

func (ih *idxHintsQuery) addUseIndexForGroupBy(indexes ...string) {
	if len(indexes) == 0 {
		return
	}
	ih.lazyUse().forGroupBy = ih.appendIndexes(ih.use.forGroupBy, indexes...)
}

func (ih *idxHintsQuery) addIgnoreIndex(indexes ...string) {
	if len(indexes) == 0 {
		return
	}
	ih.lazyIgnore().names = ih.appendIndexes(ih.ignore.names, indexes...)
}

func (ih *idxHintsQuery) addIgnoreIndexForJoin(indexes ...string) {
	if len(indexes) == 0 {
		return
	}
	ih.lazyIgnore().forJoin = ih.appendIndexes(ih.ignore.forJoin, indexes...)
}

func (ih *idxHintsQuery) addIgnoreIndexForOrderBy(indexes ...string) {
	if len(indexes) == 0 {
		return
	}
	ih.lazyIgnore().forOrderBy = ih.appendIndexes(ih.ignore.forOrderBy, indexes...)
}

func (ih *idxHintsQuery) addIgnoreIndexForGroupBy(indexes ...string) {
	if len(indexes) == 0 {
		return
	}
	ih.lazyIgnore().forGroupBy = ih.appendIndexes(ih.ignore.forGroupBy, indexes...)
}

func (ih *idxHintsQuery) addForceIndex(indexes ...string) {
	if len(indexes) == 0 {
		return
	}
	ih.lazyForce().names = ih.appendIndexes(ih.force.names, indexes...)
}

func (ih *idxHintsQuery) addForceIndexForJoin(indexes ...string) {
	if len(indexes) == 0 {
		return
	}
	ih.lazyForce().forJoin = ih.appendIndexes(ih.force.forJoin, indexes...)
}

func (ih *idxHintsQuery) addForceIndexForOrderBy(indexes ...string) {
	if len(indexes) == 0 {
		return
	}
	ih.lazyForce().forOrderBy = ih.appendIndexes(ih.force.forOrderBy, indexes...)
}

func (ih *idxHintsQuery) addForceIndexForGroupBy(indexes ...string) {
	if len(indexes) == 0 {
		return
	}
	ih.lazyForce().forGroupBy = ih.appendIndexes(ih.force.forGroupBy, indexes...)
}

func (ih *idxHintsQuery) appendIndexHints(
	gen schema.QueryGen, b []byte,
) ([]byte, error) {
	type IdxHint struct {
		Name   string
		Values []schema.QueryWithArgs
	}

	var hints []IdxHint
	if ih.use != nil {
		hints = append(hints, []IdxHint{
			{
				Name:   "USE INDEX",
				Values: ih.use.names,
			},
			{
				Name:   "USE INDEX FOR JOIN",
				Values: ih.use.forJoin,
			},
			{
				Name:   "USE INDEX FOR ORDER BY",
				Values: ih.use.forOrderBy,
			},
			{
				Name:   "USE INDEX FOR GROUP BY",
				Values: ih.use.forGroupBy,
			},
		}...)
	}

	if ih.ignore != nil {
		hints = append(hints, []IdxHint{
			{
				Name:   "IGNORE INDEX",
				Values: ih.ignore.names,
			},
			{
				Name:   "IGNORE INDEX FOR JOIN",
				Values: ih.ignore.forJoin,
			},
			{
				Name:   "IGNORE INDEX FOR ORDER BY",
				Values: ih.ignore.forOrderBy,
			},
			{
				Name:   "IGNORE INDEX FOR GROUP BY",
				Values: ih.ignore.forGroupBy,
			},
		}...)
	}

	if ih.force != nil {
		hints = append(hints, []IdxHint{
			{
				Name:   "FORCE INDEX",
				Values: ih.force.names,
			},
			{
				Name:   "FORCE INDEX FOR JOIN",
				Values: ih.force.forJoin,
			},
			{
				Name:   "FORCE INDEX FOR ORDER BY",
				Values: ih.force.forOrderBy,
			},
			{
				Name:   "FORCE INDEX FOR GROUP BY",
				Values: ih.force.forGroupBy,
			},
		}...)
	}

	var err error
	for _, h := range hints {
		b, err = ih.bufIndexHint(h.Name, h.Values, gen, b)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (ih *idxHintsQuery) bufIndexHint(
	name string,
	hints []schema.QueryWithArgs,
	gen schema.QueryGen, b []byte,
) ([]byte, error) {
	var err error
	if len(hints) == 0 {
		return b, nil
	}
	b = append(b, fmt.Sprintf(" %s (", name)...)
	for i, f := range hints {
		if i > 0 {
			b = append(b, ", "...)
		}
		b, err = f.AppendQuery(gen, b)
		if err != nil {
			return nil, err
		}
	}
	b = append(b, ")"...)
	return b, nil
}

//------------------------------------------------------------------------------

type orderLimitOffsetQuery struct {
	order []schema.QueryWithArgs

	limit  int32
	offset int32
}

func (q *orderLimitOffsetQuery) addOrder(orders ...string) {
	for _, order := range orders {
		if order == "" {
			continue
		}

		index := strings.IndexByte(order, ' ')
		if index == -1 {
			q.order = append(q.order, schema.UnsafeIdent(order))
			continue
		}

		colName := order[:index]
		sortDir := order[index+1:]
		q.addOrderExpr("? ?", Ident(colName), Order(strings.ToUpper(sortDir)))
	}
}

func (q *orderLimitOffsetQuery) addOrderBy(colName string, sortDir Order) {
	q.addOrderExpr("? ?", Ident(colName), sortDir)
}

func (q *orderLimitOffsetQuery) addOrderExpr(query string, args ...any) {
	q.order = append(q.order, schema.SafeQuery(query, args))
}

func (q *orderLimitOffsetQuery) appendOrder(gen schema.QueryGen, b []byte) (_ []byte, err error) {
	if len(q.order) > 0 {
		b = append(b, " ORDER BY "...)

		for i, f := range q.order {
			if i > 0 {
				b = append(b, ", "...)
			}
			b, err = f.AppendQuery(gen, b)
			if err != nil {
				return nil, err
			}
		}

		return b, nil
	}

	// MSSQL: allows Limit() without Order() as per https://stackoverflow.com/a/36156953
	if q.limit > 0 && gen.Dialect().Name() == dialect.MSSQL {
		return append(b, " ORDER BY _temp_sort"...), nil
	}

	return b, nil
}

func (q *orderLimitOffsetQuery) setLimit(n int) {
	q.limit = int32(n)
}

func (q *orderLimitOffsetQuery) setOffset(n int) {
	q.offset = int32(n)
}

func (q *orderLimitOffsetQuery) appendLimitOffset(gen schema.QueryGen, b []byte) (_ []byte, err error) {
	if gen.Dialect().Features().Has(feature.OffsetFetch) {
		if q.limit > 0 && q.offset > 0 {
			b = append(b, " OFFSET "...)
			b = strconv.AppendInt(b, int64(q.offset), 10)
			b = append(b, " ROWS"...)

			b = append(b, " FETCH NEXT "...)
			b = strconv.AppendInt(b, int64(q.limit), 10)
			b = append(b, " ROWS ONLY"...)
		} else if q.limit > 0 {
			b = append(b, " OFFSET 0 ROWS"...)

			b = append(b, " FETCH NEXT "...)
			b = strconv.AppendInt(b, int64(q.limit), 10)
			b = append(b, " ROWS ONLY"...)
		} else if q.offset > 0 {
			b = append(b, " OFFSET "...)
			b = strconv.AppendInt(b, int64(q.offset), 10)
			b = append(b, " ROWS"...)
		}
	} else {
		if q.limit > 0 {
			b = append(b, " LIMIT "...)
			b = strconv.AppendInt(b, int64(q.limit), 10)
		}
		if q.offset > 0 {
			b = append(b, " OFFSET "...)
			b = strconv.AppendInt(b, int64(q.offset), 10)
		}
	}

	return b, nil
}

// IsReadOnlyQuery reports whether the provided query and its CTEs are SELECT-only.
func IsReadOnlyQuery(query Query) bool {
	sel, ok := query.(*SelectQuery)
	if !ok {
		return false
	}
	for _, el := range sel.with {
		if !IsReadOnlyQuery(el.query) {
			return false
		}
	}
	return true
}
