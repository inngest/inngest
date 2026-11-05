package schema

import (
	"fmt"
	"reflect"

	"github.com/uptrace/bun/dialect"
)

// Deprecated: Use List or Tuple instead.
func In(slice any) QueryAppender {
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice {
		return &inValues{
			err: fmt.Errorf("bun: In(non-slice %T)", slice),
		}
	}
	return &inValues{
		slice: v,
	}
}

type inValues struct {
	slice reflect.Value
	err   error
}

var _ QueryAppender = (*inValues)(nil)

func (in *inValues) AppendQuery(gen QueryGen, b []byte) (_ []byte, err error) {
	if in.err != nil {
		return nil, in.err
	}
	return appendIn(gen, b, in.slice), nil
}

func appendIn(gen QueryGen, b []byte, slice reflect.Value) []byte {
	sliceLen := slice.Len()

	if sliceLen == 0 {
		return dialect.AppendNull(b)
	}

	for i := 0; i < sliceLen; i++ {
		if i > 0 {
			b = append(b, ", "...)
		}

		elem := slice.Index(i)
		if elem.Kind() == reflect.Interface {
			elem = elem.Elem()
		}

		if elem.Kind() == reflect.Slice && elem.Type() != bytesType {
			b = append(b, '(')
			b = appendIn(gen, b, elem)
			b = append(b, ')')
		} else {
			b = gen.AppendValue(b, elem)
		}
	}
	return b
}

//------------------------------------------------------------------------------

func NullZero(value any) QueryAppender {
	return nullZero{
		value: value,
	}
}

type nullZero struct {
	value any
}

func (nz nullZero) AppendQuery(gen QueryGen, b []byte) (_ []byte, err error) {
	if isZero(nz.value) {
		return dialect.AppendNull(b), nil
	}
	return gen.AppendValue(b, reflect.ValueOf(nz.value)), nil
}
