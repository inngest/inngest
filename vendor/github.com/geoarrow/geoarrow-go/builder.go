package geoarrow

import (
	"bytes"
	"fmt"

	"github.com/apache/arrow-go/v18/arrow/array"
	json "github.com/goccy/go-json"
)

type valueBuilder[V GeometryValue, G GeometryType[V]] struct {
	*array.ExtensionBuilder
}

var (
	_ array.Builder = (*valueBuilder[WKBBytes, *WKBType])(nil)
	_ array.Builder = (*valueBuilder[PointValue, *PointType])(nil)
)

func (b *valueBuilder[V, G]) Append(v V) {
	b.AppendValue(v)
}

func (b *valueBuilder[V, G]) AppendValue(v V) {
	geomType := b.Type().(G)
	geomType.appendValueToBuilder(b.Builder, v)
}

func (b *valueBuilder[V, G]) AppendValues(v []V, valid []bool) {
	if len(v) != len(valid) && len(valid) != 0 {
		panic("len(v) != len(valid) && len(valid) != 0")
	}

	for i, val := range v {
		if len(valid) > 0 && !valid[i] {
			b.AppendNull()
		} else {
			b.AppendValue(val)
		}
	}
}

func (b *valueBuilder[V, G]) AppendNull() {
	b.Builder.AppendNull()
}

func (b *valueBuilder[V, G]) AppendValueFromString(s string) error {
	if s == array.NullValueStr {
		b.AppendNull()
		return nil
	}
	geomType := b.Type().(G)
	v, err := geomType.valueFromString(s)
	if err != nil {
		return err
	}
	b.AppendValue(v)
	return nil
}

func (b *valueBuilder[V, G]) UnmarshalOne(dec *json.Decoder) error {
	geomType := b.Type().(G)
	v, isNull, err := geomType.unmarshalJSONOne(dec)
	if err != nil {
		return err
	}
	if isNull {
		b.AppendNull()
		return nil
	}
	b.AppendValue(v)
	return nil
}

func (b *valueBuilder[V, G]) Unmarshal(dec *json.Decoder) error {
	for dec.More() {
		if err := b.UnmarshalOne(dec); err != nil {
			return err
		}
	}
	return nil
}

func (b *valueBuilder[V, G]) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	t, err := dec.Token()
	if err != nil {
		return err
	}

	if delim, ok := t.(json.Delim); !ok || delim != '[' {
		return fmt.Errorf("geoarrow builder must unpack from json array, found %s", delim)
	}

	return b.Unmarshal(dec)
}
