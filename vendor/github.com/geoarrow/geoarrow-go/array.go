package geoarrow

import (
	"strings"

	json "github.com/goccy/go-json"

	"github.com/apache/arrow-go/v18/arrow/array"
)

// a generic geometry array that can be used with all geoarrow types
type geometryArray[V GeometryValue, G GeometryType[V]] struct {
	array.ExtensionArrayBase
}

func (a *geometryArray[V, G]) String() string {
	o := new(strings.Builder)
	for i := 0; i < a.Len(); i++ {
		if i > 0 {
			o.WriteString(" ")
		}
		o.WriteString(a.ValueStr(i))
	}
	return o.String()
}

func (a *geometryArray[V, G]) Value(i int) V {
	geomType := a.ExtensionType().(G)
	return geomType.valueFromArray(a, i)
}

func (a *geometryArray[V, G]) Values() []V {
	values := make([]V, a.Len())
	for i := range values {
		values[i] = a.Value(i)
	}
	return values
}

func (a *geometryArray[V, G]) ValueStr(i int) string {
	if a.IsNull(i) {
		return array.NullValueStr
	}
	return a.Value(i).String()
}

func (a *geometryArray[V, G]) GetOneForMarshal(i int) any {
	if a.IsValid(i) {
		return a.Value(i)
	}
	return nil
}

func (a *geometryArray[V, G]) MarshalJSON() ([]byte, error) {
	vals := make([]any, a.Len())
	for i := range vals {
		vals[i] = a.GetOneForMarshal(i)
	}
	return json.Marshal(vals)
}
