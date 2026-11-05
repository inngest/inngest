package sqlitedialect

import (
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/dialect/sqltype"
	"github.com/uptrace/bun/schema"
)

func init() {
	if Version() != bun.Version() {
		panic(fmt.Errorf("sqlitedialect and Bun must have the same version: v%s != v%s",
			Version(), bun.Version()))
	}
}

type Dialect struct {
	schema.BaseDialect

	tables   *schema.Tables
	features feature.Feature
}

func New(opts ...DialectOption) *Dialect {
	d := new(Dialect)
	d.tables = schema.NewTables(d)
	d.features = feature.CTE |
		feature.WithValues |
		feature.Returning |
		feature.InsertReturning |
		feature.InsertTableAlias |
		feature.UpdateTableAlias |
		feature.DeleteTableAlias |
		feature.InsertOnConflict |
		feature.TableNotExists |
		feature.SelectExists |
		feature.AutoIncrement |
		feature.CompositeIn |
		feature.FKDefaultOnAction |
		feature.DeleteReturning |
		feature.CreateIndexIfNotExists

	for _, opt := range opts {
		opt(d)
	}

	return d
}

type DialectOption func(d *Dialect)

func WithoutFeature(other feature.Feature) DialectOption {
	return func(d *Dialect) {
		d.features = d.features.Remove(other)
	}
}

func (d *Dialect) Init(*sql.DB) {}

func (d *Dialect) Name() dialect.Name {
	return dialect.SQLite
}

func (d *Dialect) Features() feature.Feature {
	return d.features
}

func (d *Dialect) Tables() *schema.Tables {
	return d.tables
}

func (d *Dialect) OnTable(table *schema.Table) {
	for _, field := range table.FieldMap {
		d.onField(field)
	}
}

func (d *Dialect) onField(field *schema.Field) {
	field.DiscoveredSQLType = fieldSQLType(field)
}

func (d *Dialect) IdentQuote() byte {
	return '"'
}

func (d *Dialect) AppendBytes(b []byte, bs []byte) []byte {
	if bs == nil {
		return dialect.AppendNull(b)
	}

	b = append(b, `X'`...)

	s := len(b)
	b = append(b, make([]byte, hex.EncodedLen(len(bs)))...)
	hex.Encode(b[s:], bs)

	b = append(b, '\'')

	return b
}

func (*Dialect) DefaultVarcharLen() int {
	return 0
}

// AppendSequence adds AUTOINCREMENT keyword to the column definition. As per [documentation],
// AUTOINCREMENT is only valid for INTEGER PRIMARY KEY, and this method will be a noop for other columns.
//
// Because this is a valid construct:
//
//	CREATE TABLE ("id" INTEGER PRIMARY KEY AUTOINCREMENT);
//
// and this is not:
//
//	CREATE TABLE ("id" INTEGER AUTOINCREMENT, PRIMARY KEY ("id"));
//
// AppendSequence adds a primary key constraint as a *side-effect*. Callers should expect it to avoid building invalid SQL.
// SQLite also [does not support] AUTOINCREMENT column in composite primary keys.
//
// [documentation]: https://www.sqlite.org/autoinc.html
// [does not support]: https://stackoverflow.com/a/6793274/14726116
func (d *Dialect) AppendSequence(b []byte, table *schema.Table, field *schema.Field) []byte {
	if field.IsPK && len(table.PKs) == 1 && field.CreateTableSQLType == sqltype.Integer {
		b = append(b, " PRIMARY KEY AUTOINCREMENT"...)
	}
	return b
}

// DefaultSchemaName is the "schema-name" of the main database.
// The details might differ from other dialects, but for all means and purposes
// "main" is the default schema in an SQLite database.
func (*Dialect) DefaultSchema() string {
	return "main"
}

func fieldSQLType(field *schema.Field) string {
	switch field.DiscoveredSQLType {
	case sqltype.SmallInt, sqltype.BigInt:
		// INTEGER PRIMARY KEY is an alias for the ROWID.
		// It is safe to convert all ints to INTEGER, because SQLite types don't have size.
		return sqltype.Integer
	default:
		return field.DiscoveredSQLType
	}
}
