package pgdialect

import (
	"context"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
	"github.com/uptrace/bun/migrate/sqlschema"
	"github.com/uptrace/bun/schema"
)

func (d *Dialect) NewMigrator(db *bun.DB, schemaName string) sqlschema.Migrator {
	return &migrator{db: db, schemaName: schemaName, BaseMigrator: sqlschema.NewBaseMigrator(db)}
}

type migrator struct {
	*sqlschema.BaseMigrator

	db         *bun.DB
	schemaName string
}

var _ sqlschema.Migrator = (*migrator)(nil)

func (m *migrator) AppendSQL(b []byte, operation any) (_ []byte, err error) {
	gen := m.db.QueryGen()

	// Append ALTER TABLE statement to the enclosed query bytes []byte.
	appendAlterTable := func(query []byte, tableName string) []byte {
		query = append(query, "ALTER TABLE "...)
		query = m.appendFQN(gen, query, tableName)
		return append(query, " "...)
	}

	switch change := operation.(type) {
	case *migrate.CreateTableOp:
		return m.AppendCreateTable(b, change.Model)
	case *migrate.DropTableOp:
		return m.AppendDropTable(b, m.schemaName, change.TableName)
	case *migrate.RenameTableOp:
		b, err = m.renameTable(gen, appendAlterTable(b, change.TableName), change)
	case *migrate.RenameColumnOp:
		b, err = m.renameColumn(gen, appendAlterTable(b, change.TableName), change)
	case *migrate.AddColumnOp:
		b, err = m.addColumn(gen, appendAlterTable(b, change.TableName), change)
	case *migrate.DropColumnOp:
		b, err = m.dropColumn(gen, appendAlterTable(b, change.TableName), change)
	case *migrate.AddPrimaryKeyOp:
		b, err = m.addPrimaryKey(gen, appendAlterTable(b, change.TableName), change.PrimaryKey)
	case *migrate.ChangePrimaryKeyOp:
		b, err = m.changePrimaryKey(gen, appendAlterTable(b, change.TableName), change)
	case *migrate.DropPrimaryKeyOp:
		b, err = m.dropConstraint(gen, appendAlterTable(b, change.TableName), change.PrimaryKey.Name)
	case *migrate.AddUniqueConstraintOp:
		b, err = m.addUnique(gen, appendAlterTable(b, change.TableName), change)
	case *migrate.DropUniqueConstraintOp:
		b, err = m.dropConstraint(gen, appendAlterTable(b, change.TableName), change.Unique.Name)
	case *migrate.ChangeColumnTypeOp:
		// If column changes to SERIAL, create sequence first.
		// https://gist.github.com/oleglomako/185df689706c5499612a0d54d3ffe856
		if !change.From.GetIsAutoIncrement() && change.To.GetIsAutoIncrement() {
			change.To, b, err = m.createDefaultSequence(gen, b, change)
		}
		b, err = m.changeColumnType(gen, appendAlterTable(b, change.TableName), change)
	case *migrate.AddForeignKeyOp:
		b, err = m.addForeignKey(gen, appendAlterTable(b, change.TableName()), change)
	case *migrate.DropForeignKeyOp:
		b, err = m.dropConstraint(gen, appendAlterTable(b, change.TableName()), change.ConstraintName)
	default:
		return nil, fmt.Errorf("append sql: unknown operation %T", change)
	}
	if err != nil {
		return nil, fmt.Errorf("append sql: %w", err)
	}
	return b, nil
}

func (m *migrator) appendFQN(gen schema.QueryGen, b []byte, tableName string) []byte {
	return gen.AppendQuery(b, "?.?", bun.Ident(m.schemaName), bun.Ident(tableName))
}

func (m *migrator) renameTable(gen schema.QueryGen, b []byte, rename *migrate.RenameTableOp) (_ []byte, err error) {
	b = append(b, "RENAME TO "...)
	b = gen.AppendName(b, rename.NewName)
	return b, nil
}

func (m *migrator) renameColumn(gen schema.QueryGen, b []byte, rename *migrate.RenameColumnOp) (_ []byte, err error) {
	b = append(b, "RENAME COLUMN "...)
	b = gen.AppendName(b, rename.OldName)

	b = append(b, " TO "...)
	b = gen.AppendName(b, rename.NewName)

	return b, nil
}

func (m *migrator) addColumn(gen schema.QueryGen, b []byte, add *migrate.AddColumnOp) (_ []byte, err error) {
	b = append(b, "ADD COLUMN "...)
	b = gen.AppendName(b, add.ColumnName)
	b = append(b, " "...)

	b, err = add.Column.AppendQuery(gen, b)
	if err != nil {
		return nil, err
	}

	if add.Column.GetDefaultValue() != "" {
		b = append(b, " DEFAULT "...)
		b = append(b, add.Column.GetDefaultValue()...)
		b = append(b, " "...)
	}

	if add.Column.GetIsIdentity() {
		b = appendGeneratedAsIdentity(b)
	}

	return b, nil
}

func (m *migrator) dropColumn(gen schema.QueryGen, b []byte, drop *migrate.DropColumnOp) (_ []byte, err error) {
	b = append(b, "DROP COLUMN "...)
	b = gen.AppendName(b, drop.ColumnName)

	return b, nil
}

func (m *migrator) addPrimaryKey(gen schema.QueryGen, b []byte, pk sqlschema.PrimaryKey) (_ []byte, err error) {
	b = append(b, "ADD PRIMARY KEY ("...)
	b, _ = pk.Columns.AppendQuery(gen, b)
	b = append(b, ")"...)

	return b, nil
}

func (m *migrator) changePrimaryKey(gen schema.QueryGen, b []byte, change *migrate.ChangePrimaryKeyOp) (_ []byte, err error) {
	b, _ = m.dropConstraint(gen, b, change.Old.Name)
	b = append(b, ", "...)
	b, _ = m.addPrimaryKey(gen, b, change.New)
	return b, nil
}

func (m *migrator) addUnique(gen schema.QueryGen, b []byte, change *migrate.AddUniqueConstraintOp) (_ []byte, err error) {
	b = append(b, "ADD CONSTRAINT "...)
	if change.Unique.Name != "" {
		b = gen.AppendName(b, change.Unique.Name)
	} else {
		// Default naming scheme for unique constraints in Postgres is <table>_<column>_key
		b = gen.AppendName(b, fmt.Sprintf("%s_%s_key", change.TableName, change.Unique.Columns))
	}
	b = append(b, " UNIQUE ("...)
	b, _ = change.Unique.Columns.AppendQuery(gen, b)
	b = append(b, ")"...)

	return b, nil
}

func (m *migrator) dropConstraint(gen schema.QueryGen, b []byte, name string) (_ []byte, err error) {
	b = append(b, "DROP CONSTRAINT "...)
	b = gen.AppendName(b, name)

	return b, nil
}

func (m *migrator) addForeignKey(gen schema.QueryGen, b []byte, add *migrate.AddForeignKeyOp) (_ []byte, err error) {
	b = append(b, "ADD CONSTRAINT "...)

	name := add.ConstraintName
	if name == "" {
		colRef := add.ForeignKey.From
		columns := strings.Join(colRef.Column.Split(), "_")
		name = fmt.Sprintf("%s_%s_fkey", colRef.TableName, columns)
	}
	b = gen.AppendName(b, name)

	b = append(b, " FOREIGN KEY ("...)
	if b, err = add.ForeignKey.From.Column.AppendQuery(gen, b); err != nil {
		return b, err
	}
	b = append(b, ")"...)

	b = append(b, " REFERENCES "...)
	b = m.appendFQN(gen, b, add.ForeignKey.To.TableName)

	b = append(b, " ("...)
	if b, err = add.ForeignKey.To.Column.AppendQuery(gen, b); err != nil {
		return b, err
	}
	b = append(b, ")"...)

	return b, nil
}

// createDefaultSequence creates a SEQUENCE to back a serial column.
// Having a backing sequence is necessary to change column type to SERIAL.
// The updated Column's default is  set to "nextval" of the new sequence.
func (m *migrator) createDefaultSequence(_ schema.QueryGen, b []byte, op *migrate.ChangeColumnTypeOp) (_ sqlschema.Column, _ []byte, err error) {
	var last int
	if err = m.db.NewSelect().Table(op.TableName).
		ColumnExpr("MAX(?)", op.Column).Scan(context.TODO(), &last); err != nil {
		return nil, b, err
	}
	seq := op.TableName + "_" + op.Column + "_seq"
	fqn := op.TableName + "." + op.Column

	// A sequence that is OWNED BY a table will be dropped
	// if the table is dropped with CASCADE action.
	b = append(b, "CREATE SEQUENCE "...)
	b = append(b, seq...)
	b = append(b, " START WITH "...)
	b = append(b, fmt.Sprint(last+1)...) // start with next value
	b = append(b, " OWNED BY "...)
	b = append(b, fqn...)
	b = append(b, ";\n"...)

	return &Column{
		Name:            op.To.GetName(),
		SQLType:         op.To.GetSQLType(),
		VarcharLen:      op.To.GetVarcharLen(),
		DefaultValue:    fmt.Sprintf("nextval('%s'::regclass)", seq),
		IsNullable:      op.To.GetIsNullable(),
		IsAutoIncrement: op.To.GetIsAutoIncrement(),
		IsIdentity:      op.To.GetIsIdentity(),
	}, b, nil
}

func (m *migrator) changeColumnType(gen schema.QueryGen, b []byte, colDef *migrate.ChangeColumnTypeOp) (_ []byte, err error) {
	// alterColumn never re-assigns err, so there is no need to check for err != nil after calling it
	var i int
	appendAlterColumn := func() {
		if i > 0 {
			b = append(b, ", "...)
		}
		b = append(b, "ALTER COLUMN "...)
		b = gen.AppendName(b, colDef.Column)
		i++
	}

	got, want := colDef.From, colDef.To

	inspector := m.db.Dialect().(sqlschema.InspectorDialect)
	if !inspector.CompareType(want, got) {
		appendAlterColumn()
		b = append(b, " SET DATA TYPE "...)
		if b, err = want.AppendQuery(gen, b); err != nil {
			return b, err
		}
	}

	// Column must be declared NOT NULL before identity can be added.
	// Although PG can resolve the order of operations itself, we make this explicit in the query.
	if want.GetIsNullable() != got.GetIsNullable() {
		appendAlterColumn()
		if !want.GetIsNullable() {
			b = append(b, " SET NOT NULL"...)
		} else {
			b = append(b, " DROP NOT NULL"...)
		}
	}

	if want.GetIsIdentity() != got.GetIsIdentity() {
		appendAlterColumn()
		if !want.GetIsIdentity() {
			b = append(b, " DROP IDENTITY"...)
		} else {
			b = append(b, " ADD"...)
			b = appendGeneratedAsIdentity(b)
		}
	}

	if want.GetDefaultValue() != got.GetDefaultValue() {
		appendAlterColumn()
		if want.GetDefaultValue() == "" {
			b = append(b, " DROP DEFAULT"...)
		} else {
			b = append(b, " SET DEFAULT "...)
			b = append(b, want.GetDefaultValue()...)
		}
	}

	return b, nil
}
