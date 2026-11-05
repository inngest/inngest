package pgdialect

import (
	"context"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate/sqlschema"
)

type (
	Schema = sqlschema.BaseDatabase
	Table  = sqlschema.BaseTable
	Column = sqlschema.BaseColumn
)

func (d *Dialect) NewInspector(db *bun.DB, options ...sqlschema.InspectorOption) sqlschema.Inspector {
	return newInspector(db, options...)
}

type Inspector struct {
	sqlschema.InspectorConfig
	db *bun.DB
}

var _ sqlschema.Inspector = (*Inspector)(nil)

func newInspector(db *bun.DB, options ...sqlschema.InspectorOption) *Inspector {
	i := &Inspector{db: db}
	sqlschema.ApplyInspectorOptions(&i.InspectorConfig, options...)
	return i
}

func (in *Inspector) Inspect(ctx context.Context) (sqlschema.Database, error) {
	dbSchema := Schema{
		ForeignKeys: make(map[sqlschema.ForeignKey]string),
	}

	exclude := in.ExcludeTables
	if len(exclude) == 0 {
		// Avoid getting NOT LIKE ALL (ARRAY[NULL]) if bun.List() is called with an empty slice.
		exclude = []string{""}
	}

	var tables []*InformationSchemaTable
	if err := in.db.NewRaw(sqlInspectTables, in.SchemaName, bun.List(exclude)).Scan(ctx, &tables); err != nil {
		return dbSchema, err
	}

	var fks []*ForeignKey
	if err := in.db.NewRaw(sqlInspectForeignKeys, in.SchemaName, bun.List(exclude), bun.List(exclude)).Scan(ctx, &fks); err != nil {
		return dbSchema, err
	}
	dbSchema.ForeignKeys = make(map[sqlschema.ForeignKey]string, len(fks))

	for _, table := range tables {
		var columns []*InformationSchemaColumn
		if err := in.db.NewRaw(sqlInspectColumnsQuery, table.Schema, table.Name).Scan(ctx, &columns); err != nil {
			return dbSchema, err
		}

		var colDefs []sqlschema.Column
		uniqueGroups := make(map[string][]string)

		for _, c := range columns {
			def := c.Default
			if c.IsSerial || c.IsIdentity {
				def = ""
			} else if !c.IsDefaultLiteral {
				def = strings.ToLower(def)
			}

			colDefs = append(colDefs, &Column{
				Name:            c.Name,
				SQLType:         c.DataType,
				VarcharLen:      c.VarcharLen,
				DefaultValue:    def,
				IsNullable:      c.IsNullable,
				IsAutoIncrement: c.IsSerial,
				IsIdentity:      c.IsIdentity,
			})

			for _, group := range c.UniqueGroups {
				uniqueGroups[group] = append(uniqueGroups[group], c.Name)
			}
		}

		var unique []sqlschema.Unique
		for name, columns := range uniqueGroups {
			unique = append(unique, sqlschema.Unique{
				Name:    name,
				Columns: sqlschema.NewColumns(columns...),
			})
		}

		var pk *sqlschema.PrimaryKey
		if len(table.PrimaryKey.Columns) > 0 {
			pk = &sqlschema.PrimaryKey{
				Name:    table.PrimaryKey.ConstraintName,
				Columns: sqlschema.NewColumns(table.PrimaryKey.Columns...),
			}
		}

		dbSchema.Tables = append(dbSchema.Tables, &Table{
			Schema:            table.Schema,
			Name:              table.Name,
			Columns:           colDefs,
			PrimaryKey:        pk,
			UniqueConstraints: unique,
		})
	}

	for _, fk := range fks {
		dbFK := sqlschema.ForeignKey{
			From: sqlschema.NewColumnReference(fk.SourceTable, fk.SourceColumns...),
			To:   sqlschema.NewColumnReference(fk.TargetTable, fk.TargetColumns...),
		}
		if _, exclude := in.ExcludeForeignKeys[dbFK]; exclude {
			continue
		}
		dbSchema.ForeignKeys[dbFK] = fk.ConstraintName
	}
	return dbSchema, nil
}

type InformationSchemaTable struct {
	Schema     string     `bun:"table_schema,pk"`
	Name       string     `bun:"table_name,pk"`
	PrimaryKey PrimaryKey `bun:"embed:primary_key_"`

	Columns []*InformationSchemaColumn `bun:"rel:has-many,join:table_schema=table_schema,join:table_name=table_name"`
}

type InformationSchemaColumn struct {
	Schema           string   `bun:"table_schema"`
	Table            string   `bun:"table_name"`
	Name             string   `bun:"column_name"`
	DataType         string   `bun:"data_type"`
	VarcharLen       int      `bun:"varchar_len"`
	IsArray          bool     `bun:"is_array"`
	ArrayDims        int      `bun:"array_dims"`
	Default          string   `bun:"default"`
	IsDefaultLiteral bool     `bun:"default_is_literal_expr"`
	IsIdentity       bool     `bun:"is_identity"`
	IndentityType    string   `bun:"identity_type"`
	IsSerial         bool     `bun:"is_serial"`
	IsNullable       bool     `bun:"is_nullable"`
	UniqueGroups     []string `bun:"unique_groups,array"`
}

type ForeignKey struct {
	ConstraintName string   `bun:"constraint_name"`
	SourceSchema   string   `bun:"schema_name"`
	SourceTable    string   `bun:"table_name"`
	SourceColumns  []string `bun:"columns,array"`
	TargetSchema   string   `bun:"target_schema"`
	TargetTable    string   `bun:"target_table"`
	TargetColumns  []string `bun:"target_columns,array"`
}

type PrimaryKey struct {
	ConstraintName string   `bun:"name"`
	Columns        []string `bun:"columns,array"`
}

const (
	// sqlInspectTables retrieves all user-defined tables in the selected schema.
	// Pass bun.List([]string{...}) to exclude tables from this inspection or bun.List([]string{''}) to include all results.
	sqlInspectTables = `
SELECT
	"t".table_schema,
	"t".table_name,
	pk.name AS primary_key_name,
	pk.columns AS primary_key_columns
FROM information_schema.tables "t"
	LEFT JOIN (
		SELECT i.indrelid, "idx".relname AS "name", ARRAY_AGG("a".attname) AS "columns"
		FROM pg_index i
			JOIN pg_attribute "a"
				ON "a".attrelid = i.indrelid
				AND "a".attnum = ANY("i".indkey)
				AND i.indisprimary
			JOIN pg_class "idx" ON i.indexrelid = "idx".oid
		GROUP BY 1, 2
	) pk
	ON ("t".table_schema || '.' || "t".table_name)::regclass = pk.indrelid
WHERE table_type = 'BASE TABLE'
	AND "t".table_schema = ?
	AND "t".table_schema NOT LIKE 'pg_%'
	AND "table_name" NOT LIKE ALL (ARRAY[?])
ORDER BY "t".table_schema, "t".table_name
`

	// sqlInspectColumnsQuery retrieves column definitions for the specified table.
	// Unlike sqlInspectTables and sqlInspectSchema, it should be passed to bun.NewRaw
	// with additional args for table_schema and table_name.
	sqlInspectColumnsQuery = `
SELECT
	"c".table_schema,
	"c".table_name,
	"c".column_name,
	"c".data_type,
	"c".character_maximum_length::integer AS varchar_len,
	"c".data_type = 'ARRAY' AS is_array,
	COALESCE("c".array_dims, 0) AS array_dims,
	CASE
		WHEN "c".column_default ~ '^''.*''::.*$' THEN substring("c".column_default FROM '^''(.*)''::.*$')
		ELSE "c".column_default
	END AS "default",
	"c".column_default ~ '^''.*''::.*$' OR "c".column_default ~ '^[0-9\.]+$' AS default_is_literal_expr,
	"c".is_identity = 'YES' AS is_identity,
	"c".column_default = format('nextval(''%s_%s_seq''::regclass)', "c".table_name, "c".column_name) AS is_serial,
	COALESCE("c".identity_type, '') AS identity_type,
	"c".is_nullable = 'YES' AS is_nullable,
	"c"."unique_groups" AS unique_groups
FROM (
	SELECT
		"table_schema",
		"table_name",
		"column_name",
		"c".data_type,
		"c".character_maximum_length,
		"c".column_default,
		"c".is_identity,
		"c".is_nullable,
		att.array_dims,
		att.identity_type,
		att."unique_groups",
		att."constraint_type"
	FROM information_schema.columns "c"
		LEFT JOIN (
			SELECT
				s.nspname AS "table_schema",
				"t".relname AS "table_name",
				"c".attname AS "column_name",
				"c".attndims AS array_dims,
				"c".attidentity AS identity_type,
				ARRAY_AGG(con.conname) FILTER (WHERE con.contype = 'u') AS "unique_groups",
				ARRAY_AGG(con.contype) AS "constraint_type"
			FROM (
				SELECT
					conname,
					contype,
					connamespace,
					conrelid,
					conrelid AS attrelid,
					UNNEST(conkey) AS attnum
				FROM pg_constraint
			) con
				LEFT JOIN pg_attribute "c" USING (attrelid, attnum)
				LEFT JOIN pg_namespace s ON s.oid = con.connamespace
				LEFT JOIN pg_class "t" ON "t".oid = con.conrelid
			GROUP BY 1, 2, 3, 4, 5
		) att USING ("table_schema", "table_name", "column_name")
	) "c"
WHERE "table_schema" = ? AND "table_name" = ?
ORDER BY "table_schema", "table_name", "column_name"
`

	// sqlInspectForeignKeys get FK definitions for user-defined tables.
	// Pass bun.List([]string{...}) to exclude tables from this inspection or bun.List([]string{''}) to include all results.
	sqlInspectForeignKeys = `
WITH
	"schemas" AS (
		SELECT oid, nspname
		FROM pg_namespace
	),
	"tables" AS (
		SELECT oid, relnamespace, relname, relkind
		FROM pg_class
	),
	"columns" AS (
		SELECT attrelid, attname, attnum
		FROM pg_attribute
		WHERE attisdropped = false
	)
SELECT DISTINCT
	co.conname AS "constraint_name",
	ss.nspname AS schema_name,
	s.relname AS "table_name",
	ARRAY_AGG(sc.attname) AS "columns",
	ts.nspname AS target_schema,
	"t".relname AS target_table,
	ARRAY_AGG(tc.attname) AS target_columns
FROM pg_constraint co
	LEFT JOIN "tables" s ON s.oid = co.conrelid
	LEFT JOIN "schemas" ss ON ss.oid = s.relnamespace
	LEFT JOIN "columns" sc ON sc.attrelid = s.oid AND sc.attnum = ANY(co.conkey)
	LEFT JOIN "tables" t ON t.oid = co.confrelid
	LEFT JOIN "schemas" ts ON ts.oid = "t".relnamespace
	LEFT JOIN "columns" tc ON tc.attrelid = "t".oid AND tc.attnum = ANY(co.confkey)
WHERE co.contype = 'f'
	AND co.conrelid IN (SELECT oid FROM pg_class WHERE relkind = 'r')
	AND ARRAY_POSITION(co.conkey, sc.attnum) = ARRAY_POSITION(co.confkey, tc.attnum)
	AND ss.nspname = ?
	AND s.relname NOT LIKE ALL (ARRAY[?])
	AND "t".relname NOT LIKE ALL (ARRAY[?])
GROUP BY "constraint_name", "schema_name", "table_name", target_schema, target_table
`
)
