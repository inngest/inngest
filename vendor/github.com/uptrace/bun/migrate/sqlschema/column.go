package sqlschema

import (
	"fmt"

	"github.com/uptrace/bun/schema"
)

type Column interface {
	GetName() string
	GetSQLType() string
	GetVarcharLen() int
	GetDefaultValue() string
	GetIsNullable() bool
	GetIsAutoIncrement() bool
	GetIsIdentity() bool
	AppendQuery(schema.QueryGen, []byte) ([]byte, error)
}

var _ Column = (*BaseColumn)(nil)

// BaseColumn is a base column definition that stores various attributes of a column.
//
// Dialects and only dialects can use it to implement the Column interface.
// Other packages must use the Column interface.
type BaseColumn struct {
	Name            string
	SQLType         string
	VarcharLen      int
	DefaultValue    string
	IsNullable      bool
	IsAutoIncrement bool
	IsIdentity      bool
	// TODO: add Precision and Cardinality for timestamps/bit-strings/floats and arrays respectively.
}

func (cd BaseColumn) GetName() string {
	return cd.Name
}

func (cd BaseColumn) GetSQLType() string {
	return cd.SQLType
}

func (cd BaseColumn) GetVarcharLen() int {
	return cd.VarcharLen
}

func (cd BaseColumn) GetDefaultValue() string {
	return cd.DefaultValue
}

func (cd BaseColumn) GetIsNullable() bool {
	return cd.IsNullable
}

func (cd BaseColumn) GetIsAutoIncrement() bool {
	return cd.IsAutoIncrement
}

func (cd BaseColumn) GetIsIdentity() bool {
	return cd.IsIdentity
}

// AppendQuery appends full SQL data type.
func (c *BaseColumn) AppendQuery(gen schema.QueryGen, b []byte) (_ []byte, err error) {
	b = append(b, c.SQLType...)
	if c.VarcharLen == 0 {
		return b, nil
	}
	b = append(b, "("...)
	b = append(b, fmt.Sprint(c.VarcharLen)...)
	b = append(b, ")"...)
	return b, nil
}
