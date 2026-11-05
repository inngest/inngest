package migrate

import (
	"fmt"

	"github.com/uptrace/bun/migrate/sqlschema"
)

// Operation encapsulates the request to change a database definition
// and knowns which operation can revert it.
//
// It is useful to define "monolith" Operations whenever possible,
// even though they a dialect may require several distinct steps to apply them.
// For example, changing a primary key involves first dropping the old constraint
// before generating the new one. Yet, this is only an implementation detail and
// passing a higher-level ChangePrimaryKeyOp will give the dialect more information
// about the applied change.
//
// Some operations might be irreversible due to technical limitations. Returning
// a *comment from GetReverse() will add an explanatory note to the generate migration file.
//
// To declare dependency on another Operation, operations should implement
// { DependsOn(Operation) bool } interface, which Changeset will use to resolve dependencies.
type Operation interface {
	GetReverse() Operation
}

// CreateTableOp creates a new table in the schema.
//
// It does not report dependency on any other migration and may be executed first.
// Make sure the dialect does not include FOREIGN KEY constraints in the CREATE TABLE
// statement, as those may potentially reference not-yet-existing columns/tables.
type CreateTableOp struct {
	TableName string
	Model     any
}

var _ Operation = (*CreateTableOp)(nil)

func (op *CreateTableOp) GetReverse() Operation {
	return &DropTableOp{TableName: op.TableName}
}

// DropTableOp drops a database table. This operation is not reversible.
type DropTableOp struct {
	TableName string
}

var _ Operation = (*DropTableOp)(nil)

func (op *DropTableOp) DependsOn(another Operation) bool {
	drop, ok := another.(*DropForeignKeyOp)
	return ok && drop.ForeignKey.DependsOnTable(op.TableName)
}

// GetReverse for a DropTable returns a no-op migration. Logically, CreateTable is the reverse,
// but DropTable does not have the table's definition to create one.
func (op *DropTableOp) GetReverse() Operation {
	c := Unimplemented(fmt.Sprintf("WARNING: \"DROP TABLE %s\" cannot be reversed automatically because table definition is not available", op.TableName))
	return &c
}

// RenameTableOp renames the table. Changing the "schema" part of the table's FQN (moving tables between schemas) is not allowed.
type RenameTableOp struct {
	TableName string
	NewName   string
}

var _ Operation = (*RenameTableOp)(nil)

func (op *RenameTableOp) GetReverse() Operation {
	return &RenameTableOp{
		TableName: op.NewName,
		NewName:   op.TableName,
	}
}

// RenameColumnOp renames a column in the table. If the changeset includes a rename operation
// for the column's table, it should be executed first.
type RenameColumnOp struct {
	TableName string
	OldName   string
	NewName   string
}

var _ Operation = (*RenameColumnOp)(nil)

func (op *RenameColumnOp) GetReverse() Operation {
	return &RenameColumnOp{
		TableName: op.TableName,
		OldName:   op.NewName,
		NewName:   op.OldName,
	}
}

func (op *RenameColumnOp) DependsOn(another Operation) bool {
	rename, ok := another.(*RenameTableOp)
	return ok && op.TableName == rename.NewName
}

// AddColumnOp adds a new column to the table.
type AddColumnOp struct {
	TableName  string
	ColumnName string
	Column     sqlschema.Column
}

var _ Operation = (*AddColumnOp)(nil)

func (op *AddColumnOp) GetReverse() Operation {
	return &DropColumnOp{
		TableName:  op.TableName,
		ColumnName: op.ColumnName,
		Column:     op.Column,
	}
}

// DropColumnOp drop a column from the table.
//
// While some dialects allow DROP CASCADE to drop dependent constraints,
// explicit handling on constraints is preferred for transparency and debugging.
// DropColumnOp depends on DropForeignKeyOp, DropPrimaryKeyOp, and ChangePrimaryKeyOp
// if any of the constraints is defined on this table.
type DropColumnOp struct {
	TableName  string
	ColumnName string
	Column     sqlschema.Column
}

var _ Operation = (*DropColumnOp)(nil)

func (op *DropColumnOp) GetReverse() Operation {
	return &AddColumnOp{
		TableName:  op.TableName,
		ColumnName: op.ColumnName,
		Column:     op.Column,
	}
}

func (op *DropColumnOp) DependsOn(another Operation) bool {
	switch drop := another.(type) {
	case *DropForeignKeyOp:
		return drop.ForeignKey.DependsOnColumn(op.TableName, op.ColumnName)
	case *DropPrimaryKeyOp:
		return op.TableName == drop.TableName && drop.PrimaryKey.Columns.Contains(op.ColumnName)
	case *ChangePrimaryKeyOp:
		return op.TableName == drop.TableName && drop.Old.Columns.Contains(op.ColumnName)
	}
	return false
}

// AddForeignKey adds a new FOREIGN KEY constraint.
type AddForeignKeyOp struct {
	ForeignKey     sqlschema.ForeignKey
	ConstraintName string
}

var _ Operation = (*AddForeignKeyOp)(nil)

func (op *AddForeignKeyOp) TableName() string {
	return op.ForeignKey.From.TableName
}

func (op *AddForeignKeyOp) DependsOn(another Operation) bool {
	switch another := another.(type) {
	case *RenameTableOp:
		return op.ForeignKey.DependsOnTable(another.TableName) || op.ForeignKey.DependsOnTable(another.NewName)
	case *CreateTableOp:
		return op.ForeignKey.DependsOnTable(another.TableName)
	}
	return false
}

func (op *AddForeignKeyOp) GetReverse() Operation {
	return &DropForeignKeyOp{
		ForeignKey:     op.ForeignKey,
		ConstraintName: op.ConstraintName,
	}
}

// DropForeignKeyOp drops a FOREIGN KEY constraint.
type DropForeignKeyOp struct {
	ForeignKey     sqlschema.ForeignKey
	ConstraintName string
}

var _ Operation = (*DropForeignKeyOp)(nil)

func (op *DropForeignKeyOp) TableName() string {
	return op.ForeignKey.From.TableName
}

func (op *DropForeignKeyOp) GetReverse() Operation {
	return &AddForeignKeyOp{
		ForeignKey:     op.ForeignKey,
		ConstraintName: op.ConstraintName,
	}
}

// AddUniqueConstraintOp adds new UNIQUE constraint to the table.
type AddUniqueConstraintOp struct {
	TableName string
	Unique    sqlschema.Unique
}

var _ Operation = (*AddUniqueConstraintOp)(nil)

func (op *AddUniqueConstraintOp) GetReverse() Operation {
	return &DropUniqueConstraintOp{
		TableName: op.TableName,
		Unique:    op.Unique,
	}
}

func (op *AddUniqueConstraintOp) DependsOn(another Operation) bool {
	switch another := another.(type) {
	case *AddColumnOp:
		return op.TableName == another.TableName && op.Unique.Columns.Contains(another.ColumnName)
	case *RenameTableOp:
		return op.TableName == another.NewName
	case *DropUniqueConstraintOp:
		// We want to drop the constraint with the same name before adding this one.
		return op.TableName == another.TableName && op.Unique.Name == another.Unique.Name
	default:
		return false
	}
}

// DropUniqueConstraintOp drops a UNIQUE constraint.
type DropUniqueConstraintOp struct {
	TableName string
	Unique    sqlschema.Unique
}

var _ Operation = (*DropUniqueConstraintOp)(nil)

func (op *DropUniqueConstraintOp) DependsOn(another Operation) bool {
	if rename, ok := another.(*RenameTableOp); ok {
		return op.TableName == rename.NewName
	}
	return false
}

func (op *DropUniqueConstraintOp) GetReverse() Operation {
	return &AddUniqueConstraintOp{
		TableName: op.TableName,
		Unique:    op.Unique,
	}
}

// ChangeColumnTypeOp set a new data type for the column.
// The two types should be such that the data can be auto-casted from one to another.
// E.g. reducing VARCHAR lenght is not possible in most dialects.
// AutoMigrator does not enforce or validate these rules.
type ChangeColumnTypeOp struct {
	TableName string
	Column    string
	From      sqlschema.Column
	To        sqlschema.Column
}

var _ Operation = (*ChangeColumnTypeOp)(nil)

func (op *ChangeColumnTypeOp) GetReverse() Operation {
	return &ChangeColumnTypeOp{
		TableName: op.TableName,
		Column:    op.Column,
		From:      op.To,
		To:        op.From,
	}
}

// DropPrimaryKeyOp drops the table's PRIMARY KEY.
type DropPrimaryKeyOp struct {
	TableName  string
	PrimaryKey sqlschema.PrimaryKey
}

var _ Operation = (*DropPrimaryKeyOp)(nil)

func (op *DropPrimaryKeyOp) GetReverse() Operation {
	return &AddPrimaryKeyOp{
		TableName:  op.TableName,
		PrimaryKey: op.PrimaryKey,
	}
}

// AddPrimaryKeyOp adds a new PRIMARY KEY to the table.
type AddPrimaryKeyOp struct {
	TableName  string
	PrimaryKey sqlschema.PrimaryKey
}

var _ Operation = (*AddPrimaryKeyOp)(nil)

func (op *AddPrimaryKeyOp) GetReverse() Operation {
	return &DropPrimaryKeyOp{
		TableName:  op.TableName,
		PrimaryKey: op.PrimaryKey,
	}
}

func (op *AddPrimaryKeyOp) DependsOn(another Operation) bool {
	switch another := another.(type) {
	case *AddColumnOp:
		return op.TableName == another.TableName && op.PrimaryKey.Columns.Contains(another.ColumnName)
	}
	return false
}

// ChangePrimaryKeyOp changes the PRIMARY KEY of the table.
type ChangePrimaryKeyOp struct {
	TableName string
	Old       sqlschema.PrimaryKey
	New       sqlschema.PrimaryKey
}

var _ Operation = (*AddPrimaryKeyOp)(nil)

func (op *ChangePrimaryKeyOp) GetReverse() Operation {
	return &ChangePrimaryKeyOp{
		TableName: op.TableName,
		Old:       op.New,
		New:       op.Old,
	}
}

// Unimplemented denotes an Operation that cannot be executed.
//
// Operations, which cannot be reversed due to current technical limitations,
// may have their GetReverse() return &Unimplemented with a helpful message.
//
// When applying operations, changelog should skip it or output as a log message,
// and write it as an SQL Unimplemented when creating migration files.
type Unimplemented string

var _ Operation = (*Unimplemented)(nil)

func (reason *Unimplemented) GetReverse() Operation { return reason }
