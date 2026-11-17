package bun

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"

	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

type hasManyModel struct {
	*sliceTableModel
	baseTable *schema.Table
	rel       *schema.Relation

	baseValues map[internal.MapKey][]reflect.Value
	structKey  []any
}

var _ TableModel = (*hasManyModel)(nil)

func newHasManyModel(j *relationJoin) *hasManyModel {
	baseTable := j.BaseModel.Table()
	joinModel := j.JoinModel.(*sliceTableModel)
	baseValues := baseValues(joinModel, j.Relation.BasePKs)
	if len(baseValues) == 0 {
		return nil
	}
	m := hasManyModel{
		sliceTableModel: joinModel,
		baseTable:       baseTable,
		rel:             j.Relation,

		baseValues: baseValues,
	}
	if !m.sliceOfPtr {
		m.strct = reflect.New(m.table.Type).Elem()
	}
	return &m
}

func (m *hasManyModel) ScanRows(ctx context.Context, rows *sql.Rows) (int, error) {
	columns, err := rows.Columns()
	if err != nil {
		return 0, err
	}

	m.columns = columns
	dest := makeDest(m, len(columns))

	var n int
	m.structKey = make([]any, len(m.rel.JoinPKs))
	for rows.Next() {
		if m.sliceOfPtr {
			m.strct = reflect.New(m.table.Type).Elem()
		} else {
			m.strct.Set(m.table.ZeroValue)
		}
		m.structInited = false
		m.scanIndex = 0

		if err := rows.Scan(dest...); err != nil {
			return 0, err
		}

		if err := m.parkStruct(); err != nil {
			return 0, err
		}

		n++
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	return n, nil
}

func (m *hasManyModel) Scan(src any) error {
	column := m.columns[m.scanIndex]
	m.scanIndex++

	field := m.table.LookupField(column)
	if field == nil {
		return fmt.Errorf("bun: %s does not have column %q", m.table.TypeName, column)
	}

	if err := field.ScanValue(m.strct, src); err != nil {
		return err
	}

	for i, f := range m.rel.JoinPKs {
		if f.Name == column {
			m.structKey[i] = indirectAsKey(field.Value(m.strct))
			break
		}
	}

	return nil
}

func (m *hasManyModel) parkStruct() error {

	baseValues, ok := m.baseValues[internal.NewMapKey(m.structKey)]
	if !ok {
		return fmt.Errorf(
			"bun: has-many relation=%s does not have base %s with id=%q (check join conditions)",
			m.rel.Field.GoName, m.baseTable, m.structKey)
	}

	for i, v := range baseValues {
		if !m.sliceOfPtr {
			v.Set(reflect.Append(v, m.strct))
			continue
		}

		if i == 0 {
			v.Set(reflect.Append(v, m.strct.Addr()))
			continue
		}

		clone := reflect.New(m.strct.Type()).Elem()
		clone.Set(m.strct)
		v.Set(reflect.Append(v, clone.Addr()))
	}

	return nil
}

func (m *hasManyModel) clone() TableModel {
	return &hasManyModel{
		sliceTableModel: m.sliceTableModel.clone().(*sliceTableModel),
		baseTable:       m.baseTable,
		rel:             m.rel,
		baseValues:      m.baseValues,
		structKey:       m.structKey,
	}
}

func baseValues(model TableModel, fields []*schema.Field) map[internal.MapKey][]reflect.Value {
	fieldIndex := model.Relation().Field.Index
	m := make(map[internal.MapKey][]reflect.Value)
	key := make([]any, 0, len(fields))
	walk(model.rootValue(), model.parentIndex(), func(v reflect.Value) {
		key = modelKey(key[:0], v, fields)
		mapKey := internal.NewMapKey(key)
		m[mapKey] = append(m[mapKey], v.FieldByIndex(fieldIndex))
	})
	return m
}

func modelKey(key []any, strct reflect.Value, fields []*schema.Field) []any {
	for _, f := range fields {
		key = append(key, indirectAsKey(f.Value(strct)))
	}
	return key
}

// indirectAsKey return the field value dereferencing the pointer if necessary.
// The value is then used as a map key.
func indirectAsKey(field reflect.Value) any {
	if field.Kind() == reflect.Pointer && field.IsNil() {
		return nil
	}

	i := field.Interface()
	if valuer, ok := i.(driver.Valuer); ok {
		if v, err := valuer.Value(); err == nil && v != nil {
			switch reflect.TypeOf(v).Kind() {
			case reflect.Array, reflect.Chan, reflect.Func,
				reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
				// NOTE #1107, these types cannot be used as map key,
				// let us use original logic.
				return i
			default:
				return v
			}
		}
	}

	return reflect.Indirect(field).Interface()
}
