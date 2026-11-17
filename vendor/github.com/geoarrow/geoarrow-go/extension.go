package geoarrow

import (
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	json "github.com/goccy/go-json"
)

const (
	ExtensionNameWKB     = "geoarrow.wkb"
	ExtensionNamePoint   = "geoarrow.point"
	ExtensionNamePolygon = "geoarrow.polygon"
	// TODO ExtensionNameWKT = "geoarrow.wkt"
	// TODO ExtensionNameLineString = "geoarrow.linestring"
	// TODO ExtensionNameMultiPoint = "geoarrow.multipoint"
	// TODO ExtensionNameMultiLineString = "geoarrow.multilinestring"
	// TODO ExtensionNameMultiPolygon = "geoarrow.multipolygon"
	// TODO ExtensionNameGeometry = "geoarrow.geometry"
	// TODO ExtensionNameGeometryCollection = "geoarrow.geometrycollection"
	// TODO ExtensionNameBox = "geoarrow.box"
)

func init() {
	arrow.RegisterExtensionType(NewWKBType())
	arrow.RegisterExtensionType(NewPointType())
	arrow.RegisterExtensionType(NewPolygonType())
}

// GeometryType represents a GeoArrow geometry type with arbitrary
// format and encoding.
type GeometryType[V GeometryValue] interface {
	arrow.ExtensionType
	valueFromArray(a array.ExtensionArray, i int) V
	appendValueToBuilder(b array.Builder, v V)
	valueFromString(s string) (V, error)
	unmarshalJSONOne(dec *json.Decoder) (V, bool, error)
}

// Extension is a base struct that can be embedded in a GeoArrow extension
//
//	type to provide common metadata and serialization logic.
type Extension struct {
	meta Metadata
}

// Metadata returns the GeoArrow metadata associated with this extension type.
func (e *Extension) Metadata() Metadata {
	return e.meta
}

// Serialize returns the JSON string representation of the GeoArrow metadata for
// this extension type.
func (e *Extension) Serialize() string {
	if e == nil {
		return ""
	}

	// Ignore errors since Metadata fields are well-defined and should always
	// serialize successfully
	serialized, _ := json.Marshal(e.meta)
	return string(serialized)
}

// Equal compares two Extension instances for equality based on their metadata.
func (e *Extension) Equal(other *Extension) bool {
	if e == nil && other == nil {
		return true
	}
	if e == nil || other == nil {
		return false
	}
	if e.meta.CRSType != other.meta.CRSType {
		return false
	}
	if e.meta.Edges != other.meta.Edges {
		return false
	}

	if len(e.meta.CRS) != len(other.meta.CRS) {
		return false
	}
	for i, r := range e.meta.CRS {
		if r != other.meta.CRS[i] {
			return false
		}
	}

	return true
}
