package sqlschema

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

type InspectorDialect interface {
	schema.Dialect

	// Inspector returns a new instance of Inspector for the dialect.
	// Dialects MAY set their default InspectorConfig values in constructor
	// but MUST apply InspectorOptions to ensure they can be overriden.
	//
	// Use ApplyInspectorOptions to reduce boilerplate.
	NewInspector(db *bun.DB, options ...InspectorOption) Inspector

	// CompareType returns true if col1 and co2 SQL types are equivalent,
	// i.e. they might use dialect-specifc type aliases (SERIAL ~ SMALLINT)
	// or specify the same VARCHAR length differently (VARCHAR(255) ~ VARCHAR).
	CompareType(Column, Column) bool
}

// InspectorConfig controls the scope of migration by limiting the objects Inspector should return.
// Inspectors SHOULD use the configuration directly instead of copying it, or MAY choose to embed it,
// to make sure options are always applied correctly.
//
// ExcludeTables and ExcludeForeignKeys are intended for database inspectors,
// to compensate for the fact that model structs may not wholly reflect the
// state of the database schema.
// Database inspectors MUST respect these exclusions to prevent relations
// from being dropped unintentionally.
type InspectorConfig struct {
	// SchemaName limits inspection to tables in a particular schema.
	SchemaName string

	// ExcludeTables from inspection. Patterns MAY make use of wildcards
	// like % and _ and dialects MUST acknowledge that by using them
	// with the SQL LIKE operator.
	ExcludeTables []string

	// ExcludeForeignKeys from inspection.
	ExcludeForeignKeys map[ForeignKey]string
}

// Inspector reads schema state.
type Inspector interface {
	Inspect(ctx context.Context) (Database, error)
}

func WithSchemaName(schemaName string) InspectorOption {
	return func(cfg *InspectorConfig) {
		cfg.SchemaName = schemaName
	}
}

// WithExcludeTables forces inspector to exclude tables from the reported schema state.
// It works in append-only mode, i.e. tables cannot be re-included.
//
// Patterns MAY make use of % and _ wildcards, as if writing a LIKE clause in SQL.
func WithExcludeTables(tables ...string) InspectorOption {
	return func(cfg *InspectorConfig) {
		cfg.ExcludeTables = append(cfg.ExcludeTables, tables...)
	}
}

// WithExcludeForeignKeys forces inspector to exclude foreign keys
// from the reported schema state.
func WithExcludeForeignKeys(fks ...ForeignKey) InspectorOption {
	return func(cfg *InspectorConfig) {
		for _, fk := range fks {
			cfg.ExcludeForeignKeys[fk] = ""
		}
	}
}

// NewInspector creates a new database inspector, if the dialect supports it.
func NewInspector(db *bun.DB, options ...InspectorOption) (Inspector, error) {
	dialect, ok := (db.Dialect()).(InspectorDialect)
	if !ok {
		return nil, fmt.Errorf("%s does not implement sqlschema.Inspector", db.Dialect().Name())
	}
	return &inspector{
		Inspector: dialect.NewInspector(db, options...),
	}, nil
}

func NewBunModelInspector(tables *schema.Tables, options ...InspectorOption) *BunModelInspector {
	bmi := &BunModelInspector{
		tables: tables,
	}
	ApplyInspectorOptions(&bmi.InspectorConfig, options...)
	return bmi
}

// InspectorOption configures an Inspector.
type InspectorOption func(*InspectorConfig)

// ApplyInspectorOptions applies the given options to an InspectorConfig.
func ApplyInspectorOptions(cfg *InspectorConfig, options ...InspectorOption) {
	if cfg.ExcludeForeignKeys == nil {
		cfg.ExcludeForeignKeys = make(map[ForeignKey]string)
	}
	for _, opt := range options {
		opt(cfg)
	}
}

// inspector is opaque pointer to a database inspector.
type inspector struct {
	Inspector
}

// BunModelInspector creates the current project state from the passed bun.Models.
// Do not recycle BunModelInspector for different sets of models, as older models will not be de-registerred before the next run.
//
// BunModelInspector does not know which the database's dialect, so it does not
// assume any default schema name. Always specify the target schema name via
// WithSchemaName option to receive meaningful results.
type BunModelInspector struct {
	InspectorConfig
	tables *schema.Tables
}

var _ Inspector = (*BunModelInspector)(nil)

func (bmi *BunModelInspector) Inspect(ctx context.Context) (Database, error) {
	state := BunModelSchema{
		BaseDatabase: BaseDatabase{
			ForeignKeys: make(map[ForeignKey]string),
		},
	}
	for _, t := range bmi.tables.All() {
		if t.Schema != bmi.SchemaName {
			continue
		}

		var columns []Column
		for _, f := range t.Fields {

			sqlType, length, err := parseLen(f.CreateTableSQLType)
			if err != nil {
				return nil, fmt.Errorf("parse length in %q: %w", f.CreateTableSQLType, err)
			}

			columns = append(columns, &BaseColumn{
				Name:            f.Name,
				SQLType:         strings.ToLower(sqlType), // TODO(dyma): maybe this is not necessary after Column.Eq()
				VarcharLen:      length,
				DefaultValue:    exprOrLiteral(f.SQLDefault),
				IsNullable:      !f.NotNull,
				IsAutoIncrement: f.AutoIncrement,
				IsIdentity:      f.Identity,
			})
		}

		var unique []Unique
		for name, group := range t.Unique {
			// Create a separate unique index for single-column unique constraints
			//  let each dialect apply the default naming convention.
			if name == "" {
				for _, f := range group {
					unique = append(unique, Unique{Columns: NewColumns(f.Name)})
				}
				continue
			}

			// Set the name if it is a "unique group", in which case the user has provided the name.
			var columns []string
			for _, f := range group {
				columns = append(columns, f.Name)
			}
			unique = append(unique, Unique{Name: name, Columns: NewColumns(columns...)})
		}

		var pk *PrimaryKey
		if len(t.PKs) > 0 {
			var columns []string
			for _, f := range t.PKs {
				columns = append(columns, f.Name)
			}
			pk = &PrimaryKey{Columns: NewColumns(columns...)}
		}

		// In cases where a table is defined in a non-default schema in the `bun:table` tag,
		// schema.Table only extracts the name of the schema, but passes the entire tag value to t.Name
		// for backwads-compatibility. For example, a bun model like this:
		// 	type Model struct { bun.BaseModel `bun:"table:favourite.books` }
		// produces
		// 	schema.Table{ Schema: "favourite", Name: "favourite.books" }
		tableName := strings.TrimPrefix(t.Name, t.Schema+".")
		state.Tables = append(state.Tables, &BunTable{
			BaseTable: BaseTable{
				Schema:            t.Schema,
				Name:              tableName,
				Columns:           columns,
				UniqueConstraints: unique,
				PrimaryKey:        pk,
			},
			Model: t.ZeroIface,
		})

		for _, rel := range t.Relations {
			// These relations are nominal and do not need a foreign key to be declared in the current table.
			// They will be either expressed as N:1 relations in an m2m mapping table, or will be referenced by the other table if it's a 1:N.
			if rel.Type == schema.ManyToManyRelation ||
				rel.Type == schema.HasManyRelation {
				continue
			}

			var fromCols, toCols []string
			for _, f := range rel.BasePKs {
				fromCols = append(fromCols, f.Name)
			}
			for _, f := range rel.JoinPKs {
				toCols = append(toCols, f.Name)
			}

			target := rel.JoinTable
			state.ForeignKeys[ForeignKey{
				From: NewColumnReference(t.Name, fromCols...),
				To:   NewColumnReference(target.Name, toCols...),
			}] = ""
		}
	}
	return state, nil
}

func parseLen(typ string) (string, int, error) {
	paren := strings.Index(typ, "(")
	if paren == -1 {
		return typ, 0, nil
	}
	length, err := strconv.Atoi(typ[paren+1 : len(typ)-1])
	if err != nil {
		return typ, 0, err
	}
	return typ[:paren], length, nil
}

// exprOrLiteral converts string to lowercase, if it does not contain a string literal 'lit'
// and trims the surrounding ” otherwise.
// Use it to ensure that user-defined default values in the models are always comparable
// to those returned by the database inspector, regardless of the case convention in individual drivers.
func exprOrLiteral(s string) string {
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		return strings.Trim(s, "'")
	}
	return strings.ToLower(s)
}

// BunModelSchema is the schema state derived from bun table models.
type BunModelSchema struct {
	BaseDatabase

	Tables []Table
}

func (ms BunModelSchema) GetTables() []Table {
	return ms.Tables
}

// BunTable provides additional table metadata that is only accessible from scanning bun models.
type BunTable struct {
	BaseTable

	// Model stores the zero interface to the underlying Go struct.
	Model any
}
