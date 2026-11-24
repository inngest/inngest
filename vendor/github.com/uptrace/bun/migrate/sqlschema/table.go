package sqlschema

type Table interface {
	GetSchema() string
	GetName() string
	GetColumns() []Column
	GetPrimaryKey() *PrimaryKey
	GetUniqueConstraints() []Unique
}

var _ Table = (*BaseTable)(nil)

// BaseTable is a base table definition.
//
// Dialects and only dialects can use it to implement the Table interface.
// Other packages must use the Table interface.
type BaseTable struct {
	Schema string
	Name   string

	// ColumnDefinitions map each column name to the column definition.
	Columns []Column

	// PrimaryKey holds the primary key definition.
	// A nil value means that no primary key is defined for the table.
	PrimaryKey *PrimaryKey

	// UniqueConstraints defined on the table.
	UniqueConstraints []Unique
}

// PrimaryKey represents a primary key constraint defined on 1 or more columns.
type PrimaryKey struct {
	Name    string
	Columns Columns
}

func (td *BaseTable) GetSchema() string {
	return td.Schema
}

func (td *BaseTable) GetName() string {
	return td.Name
}

func (td *BaseTable) GetColumns() []Column {
	return td.Columns
}

func (td *BaseTable) GetPrimaryKey() *PrimaryKey {
	return td.PrimaryKey
}

func (td *BaseTable) GetUniqueConstraints() []Unique {
	return td.UniqueConstraints
}
