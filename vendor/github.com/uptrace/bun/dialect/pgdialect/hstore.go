package pgdialect

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/uptrace/bun/schema"
)

type HStoreValue struct {
	v reflect.Value

	append schema.AppenderFunc
	scan   schema.ScannerFunc
}

// HStore accepts a map[string]string and returns a wrapper for working with PostgreSQL
// hstore data type.
//
// For struct fields you can use hstore tag:
//
//	Attrs  map[string]string `bun:",hstore"`
func HStore(vi any) *HStoreValue {
	v := reflect.ValueOf(vi)
	if !v.IsValid() {
		panic(fmt.Errorf("bun: HStore(nil)"))
	}

	typ := v.Type()
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Map {
		panic(fmt.Errorf("bun: Hstore(unsupported %s)", typ))
	}

	return &HStoreValue{
		v: v,

		append: pgDialect.hstoreAppender(v.Type()),
		scan:   hstoreScanner(v.Type()),
	}
}

var (
	_ schema.QueryAppender = (*HStoreValue)(nil)
	_ sql.Scanner          = (*HStoreValue)(nil)
)

func (h *HStoreValue) AppendQuery(gen schema.QueryGen, b []byte) ([]byte, error) {
	if h.append == nil {
		panic(fmt.Errorf("bun: HStore(unsupported %s)", h.v.Type()))
	}
	return h.append(gen, b, h.v), nil
}

func (h *HStoreValue) Scan(src any) error {
	if h.scan == nil {
		return fmt.Errorf("bun: HStore(unsupported %s)", h.v.Type())
	}
	if h.v.Kind() != reflect.Ptr {
		return fmt.Errorf("bun: HStore(non-pointer %s)", h.v.Type())
	}
	return h.scan(h.v.Elem(), src)
}

func (h *HStoreValue) Value() any {
	if h.v.IsValid() {
		return h.v.Interface()
	}
	return nil
}
