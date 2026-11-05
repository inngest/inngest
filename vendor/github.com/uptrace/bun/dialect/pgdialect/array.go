package pgdialect

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/schema"
)

type ArrayValue struct {
	v reflect.Value

	append schema.AppenderFunc
	scan   schema.ScannerFunc
}

// Array accepts a slice and returns a wrapper for working with PostgreSQL
// array data type.
//
// For struct fields you can use array tag:
//
//	Emails  []string `bun:",array"`
func Array(vi any) *ArrayValue {
	v := reflect.ValueOf(vi)
	if !v.IsValid() {
		panic(fmt.Errorf("bun: Array(nil)"))
	}

	return &ArrayValue{
		v: v,

		append: pgDialect.arrayAppender(v.Type()),
		scan:   arrayScanner(v.Type()),
	}
}

var (
	_ schema.QueryAppender = (*ArrayValue)(nil)
	_ sql.Scanner          = (*ArrayValue)(nil)
)

func (a *ArrayValue) AppendQuery(gen schema.QueryGen, b []byte) ([]byte, error) {
	if a.append == nil {
		panic(fmt.Errorf("bun: Array(unsupported %s)", a.v.Type()))
	}
	return a.append(gen, b, a.v), nil
}

func (a *ArrayValue) Scan(src any) error {
	if a.scan == nil {
		return fmt.Errorf("bun: Array(unsupported %s)", a.v.Type())
	}
	if a.v.Kind() != reflect.Ptr {
		return fmt.Errorf("bun: Array(non-pointer %s)", a.v.Type())
	}
	return a.scan(a.v, src)
}

func (a *ArrayValue) Value() any {
	if a.v.IsValid() {
		return a.v.Interface()
	}
	return nil
}

//------------------------------------------------------------------------------

func (d *Dialect) arrayAppender(typ reflect.Type) schema.AppenderFunc {
	kind := typ.Kind()

	switch kind {
	case reflect.Ptr:
		if fn := d.arrayAppender(typ.Elem()); fn != nil {
			return schema.PtrAppender(fn)
		}
	case reflect.Slice, reflect.Array:
		// continue below
	default:
		return nil
	}

	elemType := typ.Elem()

	if kind == reflect.Slice {
		switch elemType {
		case stringType:
			return appendStringSliceValue
		case intType:
			return appendIntSliceValue
		case int64Type:
			return appendInt64SliceValue
		case float64Type:
			return appendFloat64SliceValue
		case timeType:
			return appendTimeSliceValue
		}
	}

	appendElem := d.arrayElemAppender(elemType)
	if appendElem == nil {
		panic(fmt.Errorf("pgdialect: %s is not supported", typ))
	}

	return func(gen schema.QueryGen, b []byte, v reflect.Value) []byte {
		kind := v.Kind()
		switch kind {
		case reflect.Ptr, reflect.Slice:
			if v.IsNil() {
				return dialect.AppendNull(b)
			}
		}

		if kind == reflect.Ptr {
			v = v.Elem()
		}

		b = append(b, "'{"...)

		ln := v.Len()
		for i := 0; i < ln; i++ {
			elem := v.Index(i)
			if i > 0 {
				b = append(b, ',')
			}
			b = appendElem(gen, b, elem)
		}

		b = append(b, "}'"...)

		return b
	}
}

func (d *Dialect) arrayElemAppender(typ reflect.Type) schema.AppenderFunc {
	if typ.Implements(driverValuerType) {
		return arrayAppendDriverValue
	}
	if typ == timeType {
		return appendTimeElemValue
	}

	switch typ.Kind() {
	case reflect.String:
		return appendStringElemValue
	case reflect.Slice:
		if typ.Elem().Kind() == reflect.Uint8 {
			return appendBytesElemValue
		}
	case reflect.Ptr:
		return schema.PtrAppender(d.arrayElemAppender(typ.Elem()))
	}
	return schema.Appender(d, typ)
}

func appendTimeElemValue(gen schema.QueryGen, b []byte, v reflect.Value) []byte {
	ts := v.Convert(timeType).Interface().(time.Time)

	b = append(b, '"')
	b = appendTime(b, ts)
	return append(b, '"')
}

func appendStringElemValue(gen schema.QueryGen, b []byte, v reflect.Value) []byte {
	return appendStringElem(b, v.String())
}

func appendBytesElemValue(gen schema.QueryGen, b []byte, v reflect.Value) []byte {
	return appendBytesElem(b, v.Bytes())
}

func arrayAppendDriverValue(gen schema.QueryGen, b []byte, v reflect.Value) []byte {
	iface, err := v.Interface().(driver.Valuer).Value()
	if err != nil {
		return dialect.AppendError(b, err)
	}
	return appendElem(b, iface)
}

func appendStringSliceValue(gen schema.QueryGen, b []byte, v reflect.Value) []byte {
	ss := v.Convert(sliceStringType).Interface().([]string)
	return appendStringSlice(b, ss)
}

func appendStringSlice(b []byte, ss []string) []byte {
	if ss == nil {
		return dialect.AppendNull(b)
	}

	b = append(b, '\'')

	b = append(b, '{')
	for _, s := range ss {
		b = appendStringElem(b, s)
		b = append(b, ',')
	}
	if len(ss) > 0 {
		b[len(b)-1] = '}' // Replace trailing comma.
	} else {
		b = append(b, '}')
	}

	b = append(b, '\'')

	return b
}

func appendIntSliceValue(gen schema.QueryGen, b []byte, v reflect.Value) []byte {
	ints := v.Convert(sliceIntType).Interface().([]int)
	return appendIntSlice(b, ints)
}

func appendIntSlice(b []byte, ints []int) []byte {
	if ints == nil {
		return dialect.AppendNull(b)
	}

	b = append(b, '\'')

	b = append(b, '{')
	for _, n := range ints {
		b = strconv.AppendInt(b, int64(n), 10)
		b = append(b, ',')
	}
	if len(ints) > 0 {
		b[len(b)-1] = '}' // Replace trailing comma.
	} else {
		b = append(b, '}')
	}

	b = append(b, '\'')

	return b
}

func appendInt64SliceValue(gen schema.QueryGen, b []byte, v reflect.Value) []byte {
	ints := v.Convert(sliceInt64Type).Interface().([]int64)
	return appendInt64Slice(b, ints)
}

func appendInt64Slice(b []byte, ints []int64) []byte {
	if ints == nil {
		return dialect.AppendNull(b)
	}

	b = append(b, '\'')

	b = append(b, '{')
	for _, n := range ints {
		b = strconv.AppendInt(b, n, 10)
		b = append(b, ',')
	}
	if len(ints) > 0 {
		b[len(b)-1] = '}' // Replace trailing comma.
	} else {
		b = append(b, '}')
	}

	b = append(b, '\'')

	return b
}

func appendFloat64SliceValue(gen schema.QueryGen, b []byte, v reflect.Value) []byte {
	floats := v.Convert(sliceFloat64Type).Interface().([]float64)
	return appendFloat64Slice(b, floats)
}

func appendFloat64Slice(b []byte, floats []float64) []byte {
	if floats == nil {
		return dialect.AppendNull(b)
	}

	b = append(b, '\'')

	b = append(b, '{')
	for _, n := range floats {
		b = arrayAppendFloat64(b, n)
		b = append(b, ',')
	}
	if len(floats) > 0 {
		b[len(b)-1] = '}' // Replace trailing comma.
	} else {
		b = append(b, '}')
	}

	b = append(b, '\'')

	return b
}

func arrayAppendFloat64(b []byte, num float64) []byte {
	switch {
	case math.IsNaN(num):
		return append(b, "NaN"...)
	case math.IsInf(num, 1):
		return append(b, "Infinity"...)
	case math.IsInf(num, -1):
		return append(b, "-Infinity"...)
	default:
		return strconv.AppendFloat(b, num, 'f', -1, 64)
	}
}

func appendTimeSliceValue(gen schema.QueryGen, b []byte, v reflect.Value) []byte {
	ts := v.Convert(sliceTimeType).Interface().([]time.Time)
	return appendTimeSlice(gen, b, ts)
}

func appendTimeSlice(gen schema.QueryGen, b []byte, ts []time.Time) []byte {
	if ts == nil {
		return dialect.AppendNull(b)
	}
	b = append(b, '\'')
	b = append(b, '{')
	for _, t := range ts {
		b = append(b, '"')
		b = appendTime(b, t)
		b = append(b, '"')
		b = append(b, ',')
	}
	if len(ts) > 0 {
		b[len(b)-1] = '}' // Replace trailing comma.
	} else {
		b = append(b, '}')
	}
	b = append(b, '\'')
	return b
}

//------------------------------------------------------------------------------

func arrayScanner(typ reflect.Type) schema.ScannerFunc {
	kind := typ.Kind()

	switch kind {
	case reflect.Ptr:
		if fn := arrayScanner(typ.Elem()); fn != nil {
			return schema.PtrScanner(fn)
		}
	case reflect.Slice, reflect.Array:
		// ok:
	default:
		return nil
	}

	elemType := typ.Elem()

	if kind == reflect.Slice {
		switch elemType {
		case stringType:
			return scanStringSliceValue
		case intType:
			return scanIntSliceValue
		case int64Type:
			return scanInt64SliceValue
		case float64Type:
			return scanFloat64SliceValue
		}
	}

	scanElem := schema.Scanner(elemType)
	return func(dest reflect.Value, src any) error {
		dest = reflect.Indirect(dest)
		if !dest.CanSet() {
			return fmt.Errorf("bun: Scan(non-settable %s)", dest.Type())
		}

		kind := dest.Kind()

		if src == nil {
			if kind != reflect.Slice || !dest.IsNil() {
				dest.Set(reflect.Zero(dest.Type()))
			}
			return nil
		}

		if kind == reflect.Slice {
			if dest.IsNil() {
				dest.Set(reflect.MakeSlice(dest.Type(), 0, 0))
			} else if dest.Len() > 0 {
				dest.Set(dest.Slice(0, 0))
			}
		}

		if src == nil {
			return nil
		}

		b, err := toBytes(src)
		if err != nil {
			return err
		}

		p := newArrayParser(b)
		nextValue := internal.MakeSliceNextElemFunc(dest)
		for p.Next() {
			elem := p.Elem()
			elemValue := nextValue()
			if err := scanElem(elemValue, elem); err != nil {
				return fmt.Errorf("scanElem failed: %w", err)
			}
		}
		return p.Err()
	}
}

func scanStringSliceValue(dest reflect.Value, src any) error {
	dest = reflect.Indirect(dest)
	if !dest.CanSet() {
		return fmt.Errorf("bun: Scan(non-settable %s)", dest.Type())
	}

	slice, err := decodeStringSlice(src)
	if err != nil {
		return err
	}

	dest.Set(reflect.ValueOf(slice))
	return nil
}

func decodeStringSlice(src any) ([]string, error) {
	if src == nil {
		return nil, nil
	}

	b, err := toBytes(src)
	if err != nil {
		return nil, err
	}

	slice := make([]string, 0)

	p := newArrayParser(b)
	for p.Next() {
		elem := p.Elem()
		slice = append(slice, string(elem))
	}
	if err := p.Err(); err != nil {
		return nil, err
	}
	return slice, nil
}

func scanIntSliceValue(dest reflect.Value, src any) error {
	dest = reflect.Indirect(dest)
	if !dest.CanSet() {
		return fmt.Errorf("bun: Scan(non-settable %s)", dest.Type())
	}

	slice, err := decodeIntSlice(src)
	if err != nil {
		return err
	}

	dest.Set(reflect.ValueOf(slice))
	return nil
}

func decodeIntSlice(src any) ([]int, error) {
	if src == nil {
		return nil, nil
	}

	b, err := toBytes(src)
	if err != nil {
		return nil, err
	}

	slice := make([]int, 0)

	p := newArrayParser(b)
	for p.Next() {
		elem := p.Elem()

		if elem == nil {
			slice = append(slice, 0)
			continue
		}

		n, err := strconv.Atoi(internal.String(elem))
		if err != nil {
			return nil, err
		}

		slice = append(slice, n)
	}
	if err := p.Err(); err != nil {
		return nil, err
	}
	return slice, nil
}

func scanInt64SliceValue(dest reflect.Value, src any) error {
	dest = reflect.Indirect(dest)
	if !dest.CanSet() {
		return fmt.Errorf("bun: Scan(non-settable %s)", dest.Type())
	}

	slice, err := decodeInt64Slice(src)
	if err != nil {
		return err
	}

	dest.Set(reflect.ValueOf(slice))
	return nil
}

func decodeInt64Slice(src any) ([]int64, error) {
	if src == nil {
		return nil, nil
	}

	b, err := toBytes(src)
	if err != nil {
		return nil, err
	}

	slice := make([]int64, 0)

	p := newArrayParser(b)
	for p.Next() {
		elem := p.Elem()

		if elem == nil {
			slice = append(slice, 0)
			continue
		}

		n, err := strconv.ParseInt(internal.String(elem), 10, 64)
		if err != nil {
			return nil, err
		}

		slice = append(slice, n)
	}
	if err := p.Err(); err != nil {
		return nil, err
	}
	return slice, nil
}

func scanFloat64SliceValue(dest reflect.Value, src any) error {
	dest = reflect.Indirect(dest)
	if !dest.CanSet() {
		return fmt.Errorf("bun: Scan(non-settable %s)", dest.Type())
	}

	slice, err := scanFloat64Slice(src)
	if err != nil {
		return err
	}

	dest.Set(reflect.ValueOf(slice))
	return nil
}

func scanFloat64Slice(src any) ([]float64, error) {
	if src == nil {
		return nil, nil
	}

	b, err := toBytes(src)
	if err != nil {
		return nil, err
	}

	slice := make([]float64, 0)

	p := newArrayParser(b)
	for p.Next() {
		elem := p.Elem()

		if elem == nil {
			slice = append(slice, 0)
			continue
		}

		n, err := strconv.ParseFloat(internal.String(elem), 64)
		if err != nil {
			return nil, err
		}

		slice = append(slice, n)
	}
	if err := p.Err(); err != nil {
		return nil, err
	}
	return slice, nil
}

func toBytes(src any) ([]byte, error) {
	switch src := src.(type) {
	case string:
		return internal.Bytes(src), nil
	case []byte:
		return src, nil
	default:
		return nil, fmt.Errorf("pgdialect: got %T, wanted []byte or string", src)
	}
}
