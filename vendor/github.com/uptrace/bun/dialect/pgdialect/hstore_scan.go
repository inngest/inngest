package pgdialect

import (
	"fmt"
	"reflect"

	"github.com/uptrace/bun/schema"
)

func hstoreScanner(typ reflect.Type) schema.ScannerFunc {
	kind := typ.Kind()

	switch kind {
	case reflect.Ptr:
		if fn := hstoreScanner(typ.Elem()); fn != nil {
			return schema.PtrScanner(fn)
		}
	case reflect.Map:
		// ok:
	default:
		return nil
	}

	if typ.Key() == stringType && typ.Elem() == stringType {
		return scanMapStringStringValue
	}
	return func(dest reflect.Value, src any) error {
		return fmt.Errorf("bun: Hstore(unsupported %s)", dest.Type())
	}
}

func scanMapStringStringValue(dest reflect.Value, src any) error {
	dest = reflect.Indirect(dest)
	if !dest.CanSet() {
		return fmt.Errorf("bun: Scan(non-settable %s)", dest.Type())
	}

	m, err := decodeMapStringString(src)
	if err != nil {
		return err
	}

	dest.Set(reflect.ValueOf(m))
	return nil
}

func decodeMapStringString(src any) (map[string]string, error) {
	if src == nil {
		return nil, nil
	}

	b, err := toBytes(src)
	if err != nil {
		return nil, err
	}

	m := make(map[string]string)

	p := newHStoreParser(b)
	for p.Next() {
		m[p.Key()] = p.Value()
	}
	if err := p.Err(); err != nil {
		return nil, err
	}
	return m, nil
}
