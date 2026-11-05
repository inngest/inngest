package sqlschema

import (
	"slices"
	"strings"

	"github.com/uptrace/bun/schema"
)

type Database interface {
	GetTables() []Table
	GetForeignKeys() map[ForeignKey]string
}

var _ Database = (*BaseDatabase)(nil)

// BaseDatabase is a base database definition.
//
// Dialects and only dialects can use it to implement the Database interface.
// Other packages must use the Database interface.
type BaseDatabase struct {
	Tables      []Table
	ForeignKeys map[ForeignKey]string
}

func (ds BaseDatabase) GetTables() []Table {
	return ds.Tables
}

func (ds BaseDatabase) GetForeignKeys() map[ForeignKey]string {
	return ds.ForeignKeys
}

// ForeignKey represents a foreign key constraint between two tables.
type ForeignKey struct {
	From ColumnReference
	To   ColumnReference
}

func NewColumnReference(tableName string, columns ...string) ColumnReference {
	return ColumnReference{
		TableName: tableName,
		Column:    NewColumns(columns...),
	}
}

func (fk ForeignKey) DependsOnTable(tableName string) bool {
	return fk.From.TableName == tableName || fk.To.TableName == tableName
}

func (fk ForeignKey) DependsOnColumn(tableName string, column string) bool {
	return fk.DependsOnTable(tableName) &&
		(fk.From.Column.Contains(column) || fk.To.Column.Contains(column))
}

// Columns is a hashable representation of []string used to define schema constraints that depend on multiple columns.
// Although having duplicated column references in these constraints is illegal, Columns neither validates nor enforces this constraint on the caller.
type Columns string

// NewColumns creates a composite column from a slice of column names.
func NewColumns(columns ...string) Columns {
	slices.Sort(columns)
	return Columns(strings.Join(columns, ","))
}

func (c *Columns) String() string {
	return string(*c)
}

func (c *Columns) AppendQuery(gen schema.QueryGen, b []byte) ([]byte, error) {
	return schema.Safe(*c).AppendQuery(gen, b)
}

// Split returns a slice of column names that make up the composite.
func (c *Columns) Split() []string {
	return strings.Split(c.String(), ",")
}

// ContainsColumns checks that columns in "other" are a subset of current colums.
func (c *Columns) ContainsColumns(other Columns) bool {
	columns := c.Split()
Outer:
	for _, check := range other.Split() {
		for _, column := range columns {
			if check == column {
				continue Outer
			}
		}
		return false
	}
	return true
}

// Contains checks that a composite column contains the current column.
func (c *Columns) Contains(other string) bool {
	return c.ContainsColumns(Columns(other))
}

// Replace renames a column if it is part of the composite.
// If a composite consists of multiple columns, only one column will be renamed.
func (c *Columns) Replace(oldColumn, newColumn string) bool {
	columns := c.Split()
	for i, column := range columns {
		if column == oldColumn {
			columns[i] = newColumn
			*c = NewColumns(columns...)
			return true
		}
	}
	return false
}

// Unique represents a unique constraint defined on 1 or more columns.
type Unique struct {
	Name    string
	Columns Columns
}

// Equals checks that two unique constraint are the same, assuming both are defined for the same table.
func (u Unique) Equals(other Unique) bool {
	return u.Columns == other.Columns
}

// ColumnReference identifies a column or set of columns in a table.
type ColumnReference struct {
	TableName string
	Column    Columns
}
