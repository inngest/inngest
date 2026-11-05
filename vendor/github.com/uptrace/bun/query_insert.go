package bun

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

// InsertQuery builds SQL INSERT statements.
type InsertQuery struct {
	whereBaseQuery
	returningQuery
	customValueQuery

	on schema.QueryWithArgs
	setQuery

	ignore  bool
	replace bool
	comment string
}

var _ Query = (*InsertQuery)(nil)

// NewInsertQuery returns an InsertQuery tied to the provided DB.
func NewInsertQuery(db *DB) *InsertQuery {
	q := &InsertQuery{
		whereBaseQuery: whereBaseQuery{
			baseQuery: baseQuery{
				db: db,
			},
		},
	}
	return q
}

func (q *InsertQuery) Conn(db IConn) *InsertQuery {
	q.setConn(db)
	return q
}

func (q *InsertQuery) Model(model any) *InsertQuery {
	q.setModel(model)
	return q
}

func (q *InsertQuery) Err(err error) *InsertQuery {
	q.setErr(err)
	return q
}

// Apply calls each function in fns, passing the InsertQuery as an argument.
func (q *InsertQuery) Apply(fns ...func(*InsertQuery) *InsertQuery) *InsertQuery {
	for _, fn := range fns {
		if fn != nil {
			q = fn(q)
		}
	}
	return q
}

func (q *InsertQuery) With(name string, query Query) *InsertQuery {
	q.addWith(NewWithQuery(name, query))
	return q
}

func (q *InsertQuery) WithRecursive(name string, query Query) *InsertQuery {
	q.addWith(NewWithQuery(name, query).Recursive())
	return q
}

func (q *InsertQuery) WithQuery(query *WithQuery) *InsertQuery {
	q.addWith(query)
	return q
}


//------------------------------------------------------------------------------

func (q *InsertQuery) Table(tables ...string) *InsertQuery {
	for _, table := range tables {
		q.addTable(schema.UnsafeIdent(table))
	}
	return q
}

func (q *InsertQuery) TableExpr(query string, args ...any) *InsertQuery {
	q.addTable(schema.SafeQuery(query, args))
	return q
}

func (q *InsertQuery) ModelTableExpr(query string, args ...any) *InsertQuery {
	q.modelTableName = schema.SafeQuery(query, args)
	return q
}

//------------------------------------------------------------------------------

func (q *InsertQuery) Column(columns ...string) *InsertQuery {
	for _, column := range columns {
		q.addColumn(schema.UnsafeIdent(column))
	}
	return q
}

func (q *InsertQuery) ColumnExpr(query string, args ...any) *InsertQuery {
	q.addColumn(schema.SafeQuery(query, args))
	return q
}

func (q *InsertQuery) ExcludeColumn(columns ...string) *InsertQuery {
	q.excludeColumn(columns)
	return q
}

// Value overwrites model value for the column.
func (q *InsertQuery) Value(column string, expr string, args ...any) *InsertQuery {
	if q.table == nil {
		q.setErr(errNilModel)
		return q
	}
	q.addValue(q.table, column, expr, args)
	return q
}

func (q *InsertQuery) Where(query string, args ...any) *InsertQuery {
	q.addWhere(schema.SafeQueryWithSep(query, args, " AND "))
	return q
}

func (q *InsertQuery) WhereOr(query string, args ...any) *InsertQuery {
	q.addWhere(schema.SafeQueryWithSep(query, args, " OR "))
	return q
}

//------------------------------------------------------------------------------

// Returning adds a RETURNING clause to the query.
//
// To suppress the auto-generated RETURNING clause, use `Returning("")`.
func (q *InsertQuery) Returning(query string, args ...any) *InsertQuery {
	q.addReturning(schema.SafeQuery(query, args))
	return q
}

//------------------------------------------------------------------------------

// Ignore generates different queries depending on the DBMS:
//   - On MySQL, it generates `INSERT IGNORE INTO`.
//   - On PostgreSQL, it generates `ON CONFLICT DO NOTHING`.
func (q *InsertQuery) Ignore() *InsertQuery {
	if q.db.gen.HasFeature(feature.InsertOnConflict) {
		return q.On("CONFLICT DO NOTHING")
	}
	if q.db.gen.HasFeature(feature.InsertIgnore) {
		q.ignore = true
	}
	return q
}

// Replaces generates a `REPLACE INTO` query (MySQL and MariaDB).
func (q *InsertQuery) Replace() *InsertQuery {
	q.replace = true
	return q
}

//------------------------------------------------------------------------------

// Comment adds a comment to the query, wrapped by /* ... */.
func (q *InsertQuery) Comment(comment string) *InsertQuery {
	q.comment = comment
	return q
}

//------------------------------------------------------------------------------

func (q *InsertQuery) Operation() string {
	return "INSERT"
}

func (q *InsertQuery) AppendQuery(gen schema.QueryGen, b []byte) (_ []byte, err error) {
	if q.err != nil {
		return nil, q.err
	}

	b = appendComment(b, q.comment)

	gen = formatterWithModel(gen, q)

	b, err = q.appendWith(gen, b)
	if err != nil {
		return nil, err
	}

	if q.replace {
		b = append(b, "REPLACE "...)
	} else {
		b = append(b, "INSERT "...)
		if q.ignore {
			b = append(b, "IGNORE "...)
		}
	}
	b = append(b, "INTO "...)

	if q.db.HasFeature(feature.InsertTableAlias) && !q.on.IsZero() {
		b, err = q.appendFirstTableWithAlias(gen, b)
	} else {
		b, err = q.appendFirstTable(gen, b)
	}
	if err != nil {
		return nil, err
	}

	b, err = q.appendColumnsValues(gen, b, false)
	if err != nil {
		return nil, err
	}

	b, err = q.appendOn(gen, b)
	if err != nil {
		return nil, err
	}

	if q.hasFeature(feature.InsertReturning) && q.hasReturning() {
		b = append(b, " RETURNING "...)
		b, err = q.appendReturning(gen, b)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (q *InsertQuery) appendColumnsValues(
	gen schema.QueryGen, b []byte, skipOutput bool,
) (_ []byte, err error) {
	if q.hasMultiTables() {
		if q.columns != nil {
			b = append(b, " ("...)
			b, err = q.appendColumns(gen, b)
			if err != nil {
				return nil, err
			}
			b = append(b, ")"...)
		}

		if q.hasFeature(feature.Output) && q.hasReturning() {
			b = append(b, " OUTPUT "...)
			b, err = q.appendOutput(gen, b)
			if err != nil {
				return nil, err
			}
		}

		b = append(b, " SELECT "...)

		if q.columns != nil {
			b, err = q.appendColumns(gen, b)
			if err != nil {
				return nil, err
			}
		} else {
			b = append(b, "*"...)
		}

		b = append(b, " FROM "...)
		b, err = q.appendOtherTables(gen, b)
		if err != nil {
			return nil, err
		}

		return b, nil
	}

	if m, ok := q.model.(*mapModel); ok {
		return m.appendColumnsValues(gen, b), nil
	}
	if _, ok := q.model.(*mapSliceModel); ok {
		return nil, fmt.Errorf("Insert(*[]map[string]any) is not supported")
	}

	if q.model == nil {
		return nil, errNilModel
	}

	// Build fields to populate RETURNING clause.
	fields, err := q.getFields()
	if err != nil {
		return nil, err
	}

	b = append(b, " ("...)
	b = q.appendFields(gen, b, fields)
	b = append(b, ")"...)

	if q.hasFeature(feature.Output) && q.hasReturning() && !skipOutput {
		b = append(b, " OUTPUT "...)
		b, err = q.appendOutput(gen, b)
		if err != nil {
			return nil, err
		}
	}

	b = append(b, " VALUES ("...)

	switch model := q.tableModel.(type) {
	case *structTableModel:
		b, err = q.appendStructValues(gen, b, fields, model.strct)
		if err != nil {
			return nil, err
		}
	case *sliceTableModel:
		b, err = q.appendSliceValues(gen, b, fields, model.slice)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("bun: Insert does not support %T", q.tableModel)
	}

	b = append(b, ')')

	return b, nil
}

func (q *InsertQuery) appendStructValues(
	gen schema.QueryGen, b []byte, fields []*schema.Field, strct reflect.Value,
) (_ []byte, err error) {
	isTemplate := gen.IsNop()
	for i, f := range fields {
		if i > 0 {
			b = append(b, ", "...)
		}

		app, ok := q.modelValues[f.Name]
		if ok {
			b, err = app.AppendQuery(gen, b)
			if err != nil {
				return nil, err
			}
			q.addReturningField(f)
			continue
		}

		switch {
		case isTemplate:
			b = append(b, '?')
		case q.marshalsToDefault(f, strct):
			if q.db.HasFeature(feature.DefaultPlaceholder) {
				b = append(b, "DEFAULT"...)
			} else if f.SQLDefault != "" {
				b = append(b, f.SQLDefault...)
			} else {
				b = append(b, "NULL"...)
			}
			q.addReturningField(f)
		default:
			b = f.AppendValue(gen, b, strct)
		}
	}

	for i, v := range q.extraValues {
		if i > 0 || len(fields) > 0 {
			b = append(b, ", "...)
		}

		b, err = v.value.AppendQuery(gen, b)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (q *InsertQuery) appendSliceValues(
	gen schema.QueryGen, b []byte, fields []*schema.Field, slice reflect.Value,
) (_ []byte, err error) {
	if gen.IsNop() {
		return q.appendStructValues(gen, b, fields, reflect.Value{})
	}

	sliceLen := slice.Len()
	for i := 0; i < sliceLen; i++ {
		if i > 0 {
			b = append(b, "), ("...)
		}
		el := indirect(slice.Index(i))
		b, err = q.appendStructValues(gen, b, fields, el)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (q *InsertQuery) getFields() ([]*schema.Field, error) {
	hasIdentity := q.db.HasFeature(feature.Identity)

	if len(q.columns) > 0 || q.db.HasFeature(feature.DefaultPlaceholder) && !hasIdentity {
		return q.baseQuery.getFields()
	}

	var strct reflect.Value

	switch model := q.tableModel.(type) {
	case *structTableModel:
		strct = model.strct
	case *sliceTableModel:
		if model.sliceLen == 0 {
			return nil, fmt.Errorf("bun: Insert(empty %T)", model.slice.Type())
		}
		strct = indirect(model.slice.Index(0))
	default:
		return nil, errNilModel
	}

	fields := make([]*schema.Field, 0, len(q.table.Fields))

	for _, f := range q.table.Fields {
		if hasIdentity && f.AutoIncrement {
			q.addReturningField(f)
			continue
		}
		if f.NotNull && q.marshalsToDefault(f, strct) {
			q.addReturningField(f)
			continue
		}
		fields = append(fields, f)
	}

	return fields, nil
}

// marshalsToDefault checks if the value will be marshaled as DEFAULT or NULL (if DEFAULT placeholder is not supported)
// when appending it to the VALUES clause in place of the given field.
func (q InsertQuery) marshalsToDefault(f *schema.Field, v reflect.Value) bool {
	return (f.IsPtr && f.HasNilValue(v)) ||
		(f.HasZeroValue(v) && (f.NullZero || f.SQLDefault != ""))
}

func (q *InsertQuery) appendFields(
	gen schema.QueryGen, b []byte, fields []*schema.Field,
) []byte {
	b = appendColumns(b, "", fields)
	for i, v := range q.extraValues {
		if i > 0 || len(fields) > 0 {
			b = append(b, ", "...)
		}
		b = gen.AppendIdent(b, v.column)
	}
	return b
}

//------------------------------------------------------------------------------

func (q *InsertQuery) On(s string, args ...any) *InsertQuery {
	q.on = schema.SafeQuery(s, args)
	return q
}

func (q *InsertQuery) Set(query string, args ...any) *InsertQuery {
	q.addSet(schema.SafeQuery(query, args))
	return q
}

func (q *InsertQuery) SetValues(values *ValuesQuery) *InsertQuery {
	q.setValues = values
	return q
}

func (q *InsertQuery) appendOn(gen schema.QueryGen, b []byte) (_ []byte, err error) {
	if q.on.IsZero() {
		return b, nil
	}

	b = append(b, " ON "...)
	b, err = q.on.AppendQuery(gen, b)
	if err != nil {
		return nil, err
	}

	if len(q.set) > 0 || q.setValues != nil {
		if gen.HasFeature(feature.InsertOnDuplicateKey) {
			b = append(b, ' ')
		} else {
			b = append(b, " SET "...)
		}

		b, err = q.appendSet(gen, b)
		if err != nil {
			return nil, err
		}
	} else if q.onConflictDoUpdate() {
		fields, err := q.getDataFields()
		if err != nil {
			return nil, err
		}
		b = q.appendSetExcluded(b, fields)
	} else if q.onDuplicateKeyUpdate() {
		fields, err := q.getDataFields()
		if err != nil {
			return nil, err
		}
		b = q.appendSetValues(b, fields)
	}

	if len(q.where) > 0 {
		b = append(b, " WHERE "...)

		b, err = appendWhere(gen, b, q.where)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (q *InsertQuery) onConflictDoUpdate() bool {
	return strings.HasSuffix(strings.ToUpper(q.on.Query), " DO UPDATE")
}

func (q *InsertQuery) onDuplicateKeyUpdate() bool {
	return strings.ToUpper(q.on.Query) == "DUPLICATE KEY UPDATE"
}

func (q *InsertQuery) appendSetExcluded(b []byte, fields []*schema.Field) []byte {
	b = append(b, " SET "...)
	for i, f := range fields {
		if i > 0 {
			b = append(b, ", "...)
		}
		b = append(b, f.SQLName...)
		b = append(b, " = EXCLUDED."...)
		b = append(b, f.SQLName...)
	}
	return b
}

func (q *InsertQuery) appendSetValues(b []byte, fields []*schema.Field) []byte {
	b = append(b, " "...)
	for i, f := range fields {
		if i > 0 {
			b = append(b, ", "...)
		}
		b = append(b, f.SQLName...)
		b = append(b, " = VALUES("...)
		b = append(b, f.SQLName...)
		b = append(b, ")"...)
	}
	return b
}

//------------------------------------------------------------------------------

func (q *InsertQuery) Scan(ctx context.Context, dest ...any) error {
	_, err := q.scanOrExec(ctx, dest, true)
	return err
}

func (q *InsertQuery) Exec(ctx context.Context, dest ...any) (sql.Result, error) {
	return q.scanOrExec(ctx, dest, len(dest) > 0)
}

func (q *InsertQuery) scanOrExec(
	ctx context.Context, dest []any, hasDest bool,
) (sql.Result, error) {
	if q.err != nil {
		return nil, q.err
	}

	if q.table != nil {
		if err := q.beforeInsertHook(ctx); err != nil {
			return nil, err
		}
	}

	// Run append model hooks before generating the query.
	if err := q.beforeAppendModel(ctx, q); err != nil {
		return nil, err
	}

	// if a comment is propagated via the context, use it
	setCommentFromContext(ctx, q)

	// Generate the query before checking hasReturning.
	queryBytes, err := q.AppendQuery(q.db.gen, q.db.makeQueryBytes())
	if err != nil {
		return nil, err
	}

	useScan := hasDest || (q.hasReturning() && q.hasFeature(feature.InsertReturning|feature.Output))
	var model Model

	if useScan {
		var err error
		model, err = q.getModel(dest)
		if err != nil {
			return nil, err
		}
	}

	query := internal.String(queryBytes)
	var res sql.Result

	if useScan {
		res, err = q.scan(ctx, q, query, model, hasDest)
		if err != nil {
			return nil, err
		}
	} else {
		res, err = q.exec(ctx, q, query)
		if err != nil {
			return nil, err
		}

		if err := q.tryLastInsertID(res, dest); err != nil {
			return nil, err
		}
	}

	if q.table != nil {
		if err := q.afterInsertHook(ctx); err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (q *InsertQuery) beforeInsertHook(ctx context.Context) error {
	if hook, ok := q.table.ZeroIface.(BeforeInsertHook); ok {
		if err := hook.BeforeInsert(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

func (q *InsertQuery) afterInsertHook(ctx context.Context) error {
	if hook, ok := q.table.ZeroIface.(AfterInsertHook); ok {
		if err := hook.AfterInsert(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

func (q *InsertQuery) tryLastInsertID(res sql.Result, dest []any) error {
	if q.db.HasFeature(feature.Returning) ||
		q.db.HasFeature(feature.Output) ||
		q.table == nil ||
		len(q.table.PKs) != 1 ||
		!q.table.PKs[0].AutoIncrement {
		return nil
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	if id == 0 {
		return nil
	}

	model, err := q.getModel(dest)
	if err != nil {
		return err
	}

	pk := q.table.PKs[0]
	switch model := model.(type) {
	case *structTableModel:
		if err := pk.ScanValue(model.strct, id); err != nil {
			return err
		}
	case *sliceTableModel:
		sliceLen := model.slice.Len()
		for i := 0; i < sliceLen; i++ {
			strct := indirect(model.slice.Index(i))
			if err := pk.ScanValue(strct, id); err != nil {
				return err
			}
			id++
		}
	}

	return nil
}

// String returns the generated SQL query string. The InsertQuery instance must not be
// modified during query generation to ensure multiple calls to String() return identical results.
func (q *InsertQuery) String() string {
	buf, err := q.AppendQuery(q.db.QueryGen(), nil)
	if err != nil {
		panic(err)
	}
	return string(buf)
}
