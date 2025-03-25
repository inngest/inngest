package exp

import (
	"reflect"
	"sort"

	"github.com/doug-martin/goqu/v9/internal/util"
)

// Alternative to writing map[string]interface{}. Can be used for Inserts, Updates or Deletes
type Record map[string]interface{}

func (r Record) Cols() []string {
	cols := make([]string, 0, len(r))
	for col := range r {
		cols = append(cols, col)
	}
	sort.Strings(cols)
	return cols
}

func NewRecordFromStruct(i interface{}, forInsert, forUpdate bool) (r Record, err error) {
	value := reflect.ValueOf(i)
	if value.IsValid() {
		cm, err := util.GetColumnMap(value.Interface())
		if err != nil {
			return nil, err
		}
		cols := cm.Cols()
		r = make(map[string]interface{}, len(cols))
		for _, col := range cols {
			f := cm[col]
			if !shouldSkipField(f, forInsert, forUpdate) {
				if fieldValue, isAvailable := util.SafeGetFieldByIndex(value, f.FieldIndex); isAvailable {
					if !shouldOmitField(fieldValue, f) {
						r[f.ColumnName] = getRecordValue(fieldValue, f)
					}
				}
			}
		}
	}
	return
}

func shouldSkipField(f util.ColumnData, forInsert, forUpdate bool) bool {
	shouldSkipInsert := forInsert && !f.ShouldInsert
	shouldSkipUpdate := forUpdate && !f.ShouldUpdate
	return shouldSkipInsert || shouldSkipUpdate
}

func shouldOmitField(val reflect.Value, f util.ColumnData) bool {
	if f.OmitNil && util.IsNil(val) {
		return true
	} else if f.OmitEmpty && util.IsEmptyValue(val) {
		return true
	}
	return false
}

func getRecordValue(val reflect.Value, f util.ColumnData) interface{} {
	if f.DefaultIfEmpty && util.IsEmptyValue(val) {
		return Default()
	} else if val.IsValid() {
		return val.Interface()
	} else {
		return reflect.Zero(f.GoType).Interface()
	}
}
