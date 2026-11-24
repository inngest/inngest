package pgdialect

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"time"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/schema"
)

var (
	driverValuerType = reflect.TypeFor[driver.Valuer]()

	stringType      = reflect.TypeFor[string]()
	sliceStringType = reflect.TypeFor[[]string]()

	intType      = reflect.TypeFor[int]()
	sliceIntType = reflect.TypeFor[[]int]()

	int64Type      = reflect.TypeFor[int64]()
	sliceInt64Type = reflect.TypeFor[[]int64]()

	float64Type      = reflect.TypeFor[float64]()
	sliceFloat64Type = reflect.TypeFor[[]float64]()

	timeType      = reflect.TypeFor[time.Time]()
	sliceTimeType = reflect.TypeFor[[]time.Time]()
)

func appendTime(buf []byte, tm time.Time) []byte {
	return tm.UTC().AppendFormat(buf, "2006-01-02 15:04:05.999999-07:00")
}

var mapStringStringType = reflect.TypeOf(map[string]string(nil))

func (d *Dialect) hstoreAppender(typ reflect.Type) schema.AppenderFunc {
	kind := typ.Kind()

	switch kind {
	case reflect.Ptr:
		if fn := d.hstoreAppender(typ.Elem()); fn != nil {
			return schema.PtrAppender(fn)
		}
	case reflect.Map:
		// ok:
	default:
		return nil
	}

	if typ.Key() == stringType && typ.Elem() == stringType {
		return appendMapStringStringValue
	}

	return func(gen schema.QueryGen, b []byte, v reflect.Value) []byte {
		err := fmt.Errorf("bun: Hstore(unsupported %s)", v.Type())
		return dialect.AppendError(b, err)
	}
}

func appendMapStringString(b []byte, m map[string]string) []byte {
	if m == nil {
		return dialect.AppendNull(b)
	}

	b = append(b, '\'')

	for key, value := range m {
		b = appendStringElem(b, key)
		b = append(b, '=', '>')
		b = appendStringElem(b, value)
		b = append(b, ',')
	}
	if len(m) > 0 {
		b = b[:len(b)-1] // Strip trailing comma.
	}

	b = append(b, '\'')

	return b
}

func appendMapStringStringValue(gen schema.QueryGen, b []byte, v reflect.Value) []byte {
	m := v.Convert(mapStringStringType).Interface().(map[string]string)
	return appendMapStringString(b, m)
}
