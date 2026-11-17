package bun

import (
	"bytes"
	"context"
	"database/sql"
	"reflect"
	"slices"

	"github.com/uptrace/bun/schema"
)

type mapModel struct {
	db *DB

	dest *map[string]any
	m    map[string]any

	rows         *sql.Rows
	columns      []string
	_columnTypes []*sql.ColumnType
	scanIndex    int
}

var _ Model = (*mapModel)(nil)

func newMapModel(db *DB, dest *map[string]any) *mapModel {
	m := &mapModel{
		db:   db,
		dest: dest,
	}
	if dest != nil {
		m.m = *dest
	}
	return m
}

func (m *mapModel) Value() any {
	return m.dest
}

func (m *mapModel) ScanRows(ctx context.Context, rows *sql.Rows) (int, error) {
	if !rows.Next() {
		return 0, rows.Err()
	}

	columns, err := rows.Columns()
	if err != nil {
		return 0, err
	}

	m.rows = rows
	m.columns = columns
	dest := makeDest(m, len(columns))

	if m.m == nil {
		m.m = make(map[string]any, len(m.columns))
	}

	m.scanIndex = 0
	if err := rows.Scan(dest...); err != nil {
		return 0, err
	}

	*m.dest = m.m

	return 1, nil
}

func (m *mapModel) Scan(src any) error {
	if _, ok := src.([]byte); !ok {
		return m.scanRaw(src)
	}

	columnTypes, err := m.columnTypes()
	if err != nil {
		return err
	}

	scanType := columnTypes[m.scanIndex].ScanType()
	switch scanType.Kind() {
	case reflect.Interface:
		return m.scanRaw(src)
	case reflect.Slice:
		if scanType.Elem().Kind() == reflect.Uint8 {
			// Reference types such as []byte are only valid until the next call to Scan.
			src := bytes.Clone(src.([]byte))
			return m.scanRaw(src)
		}
	}

	dest := reflect.New(scanType).Elem()
	if err := schema.Scanner(scanType)(dest, src); err != nil {
		return err
	}

	return m.scanRaw(dest.Interface())
}

func (m *mapModel) columnTypes() ([]*sql.ColumnType, error) {
	if m._columnTypes == nil {
		columnTypes, err := m.rows.ColumnTypes()
		if err != nil {
			return nil, err
		}
		m._columnTypes = columnTypes
	}
	return m._columnTypes, nil
}

func (m *mapModel) scanRaw(src any) error {
	columnName := m.columns[m.scanIndex]
	m.scanIndex++
	m.m[columnName] = src
	return nil
}

func (m *mapModel) appendColumnsValues(gen schema.QueryGen, b []byte) []byte {
	keys := make([]string, 0, len(m.m))

	for k := range m.m {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	b = append(b, " ("...)

	for i, k := range keys {
		if i > 0 {
			b = append(b, ", "...)
		}
		b = gen.AppendIdent(b, k)
	}

	b = append(b, ") VALUES ("...)

	isTemplate := gen.IsNop()
	for i, k := range keys {
		if i > 0 {
			b = append(b, ", "...)
		}
		if isTemplate {
			b = append(b, '?')
		} else {
			b = gen.Append(b, m.m[k])
		}
	}

	b = append(b, ")"...)

	return b
}

func (m *mapModel) appendSet(gen schema.QueryGen, b []byte) []byte {
	keys := make([]string, 0, len(m.m))

	for k := range m.m {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	isTemplate := gen.IsNop()
	for i, k := range keys {
		if i > 0 {
			b = append(b, ", "...)
		}

		b = gen.AppendIdent(b, k)
		b = append(b, " = "...)
		if isTemplate {
			b = append(b, '?')
		} else {
			b = gen.Append(b, m.m[k])
		}
	}

	return b
}

func makeDest(v any, n int) []any {
	dest := make([]any, n)
	for i := range dest {
		dest[i] = v
	}
	return dest
}
