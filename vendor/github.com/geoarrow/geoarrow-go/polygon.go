package geoarrow

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	json "github.com/goccy/go-json"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

// PolygonValue represents a polygon geometry with zero or more rings.
// The first ring is the exterior boundary; subsequent rings are holes.
// Each ring is stored as interleaved flat coordinates [x1,y1,x2,y2,...].
type PolygonValue struct {
	rings [][]float64
	dim   Dimension
}

func NewPolygonValue(dim Dimension, rings [][]float64) PolygonValue {
	return PolygonValue{rings: rings, dim: dim}
}

func (v PolygonValue) NumRings() int {
	return len(v.rings)
}

// Ring returns the flat interleaved coordinates for ring i.
func (v PolygonValue) Ring(i int) []float64 {
	return v.rings[i]
}

// NumVertices returns the number of vertices in ring i.
func (v PolygonValue) NumVertices(i int) int {
	return len(v.rings[i]) / v.dim.NDim()
}

func (v PolygonValue) Dimension() Dimension {
	return v.dim
}

func (v PolygonValue) GeometryType() GeometryTypeID {
	switch v.dim {
	case XY:
		return PolygonID
	case XYZ:
		return PolygonZID
	case XYM:
		return PolygonMID
	case XYZM:
		return PolygonZMID
	default:
		panic("invalid coordinate dimension for PolygonValue")
	}
}

func (v PolygonValue) IsEmpty() bool {
	return len(v.rings) == 0
}

func (v PolygonValue) String() string {
	if v.IsEmpty() {
		return "POLYGON EMPTY"
	}

	b := strings.Builder{}
	b.WriteString("POLYGON")
	switch v.dim {
	case XYZ:
		b.WriteString(" Z")
	case XYM:
		b.WriteString(" M")
	case XYZM:
		b.WriteString(" ZM")
	}
	b.WriteString("(")

	stride := v.dim.NDim()
	for i, ring := range v.rings {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("(")
		nVerts := len(ring) / stride
		for vi := 0; vi < nVerts; vi++ {
			if vi > 0 {
				b.WriteString(", ")
			}
			for d := 0; d < stride; d++ {
				if d > 0 {
					b.WriteString(" ")
				}
				b.WriteString(strconv.FormatFloat(ring[vi*stride+d], 'f', 6, 64))
			}
		}
		b.WriteString(")")
	}
	b.WriteString(")")
	return b.String()
}

func (v PolygonValue) MarshalJSON() ([]byte, error) {
	stride := v.dim.NDim()
	rings := make([][][]float64, len(v.rings))
	for i, ring := range v.rings {
		nVerts := len(ring) / stride
		verts := make([][]float64, nVerts)
		for vi := 0; vi < nVerts; vi++ {
			verts[vi] = ring[vi*stride : (vi+1)*stride]
		}
		rings[i] = verts
	}
	return json.Marshal(rings)
}

// PolygonType is the extension type for Polygon geometries.
// Storage: List<List<Struct<x: double, y: double, [z: double, [m: double]]>>>
type PolygonType struct {
	arrow.ExtensionBase
	Extension
}

type polygonOption func(*PolygonType)

func PolygonWithStorage(storage arrow.DataType) polygonOption {
	return func(pt *PolygonType) {
		pt.Storage = storage
	}
}

func PolygonWithMetadata(metadata Metadata) polygonOption {
	return func(pt *PolygonType) {
		pt.meta = metadata
	}
}

func polygonStorage(coordType arrow.DataType) arrow.DataType {
	verticesList := arrow.ListOfField(arrow.Field{Name: "vertices", Type: coordType, Nullable: false})
	return arrow.ListOfField(arrow.Field{Name: "rings", Type: verticesList, Nullable: false})
}

func defaultPolygonStorage() arrow.DataType {
	coordStruct := arrow.StructOf(
		arrow.Field{Name: "x", Type: arrow.PrimitiveTypes.Float64, Nullable: false},
		arrow.Field{Name: "y", Type: arrow.PrimitiveTypes.Float64, Nullable: false},
	)
	return polygonStorage(coordStruct)
}

func NewPolygonType(opts ...polygonOption) *PolygonType {
	pt := &PolygonType{
		ExtensionBase: arrow.ExtensionBase{Storage: defaultPolygonStorage()},
		Extension:     Extension{meta: NewMetadata()},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(pt)
		}
	}
	return pt
}

func (*PolygonType) ExtensionName() string {
	return ExtensionNamePolygon
}

func (*PolygonType) Deserialize(storageType arrow.DataType, data string) (arrow.ExtensionType, error) {
	var meta Metadata
	if err := json.Unmarshal([]byte(data), &meta); err != nil {
		return nil, err
	}
	return NewPolygonType(PolygonWithStorage(storageType), PolygonWithMetadata(meta)), nil
}

func (pt *PolygonType) ExtensionEquals(other arrow.ExtensionType) bool {
	otherPt, ok := other.(*PolygonType)
	if !ok {
		return false
	}
	if pt == nil && otherPt == nil {
		return true
	}
	if pt == nil || otherPt == nil {
		return false
	}
	return pt.Storage.ID() == otherPt.Storage.ID() && pt.Equal(&otherPt.Extension)
}

func (*PolygonType) ArrayType() reflect.Type {
	return reflect.TypeOf(PolygonArray{})
}

// coordStructFromStorage extracts the coordinate struct type from the
// nested List<List<Struct>> polygon storage type.
func coordStructFromStorage(storage arrow.DataType) *arrow.StructType {
	outerList := storage.(*arrow.ListType)
	innerList := outerList.ElemField().Type.(*arrow.ListType)
	return innerList.ElemField().Type.(*arrow.StructType)
}

func (pt *PolygonType) valueFromArray(a array.ExtensionArray, i int) PolygonValue {
	if a.IsNull(i) {
		return PolygonValue{}
	}

	outerList := a.Storage().(*array.List)
	ringStart, ringEnd := outerList.ValueOffsets(i)

	innerList := outerList.ListValues().(*array.List)
	structArr := innerList.ListValues().(*array.Struct)

	coordStruct := coordStructFromStorage(pt.StorageType())
	nFields := coordStruct.NumFields()
	dim := DimensionFromStructType(coordStruct)
	stride := dim.NDim()

	rings := make([][]float64, ringEnd-ringStart)
	for r := ringStart; r < ringEnd; r++ {
		vertStart, vertEnd := innerList.ValueOffsets(int(r))
		nVerts := int(vertEnd - vertStart)
		coords := make([]float64, nVerts*stride)
		for v := 0; v < nVerts; v++ {
			idx := int(vertStart) + v
			for f := 0; f < nFields; f++ {
				coords[v*stride+f] = structArr.Field(f).(*array.Float64).Value(idx)
			}
		}
		rings[r-ringStart] = coords
	}

	return PolygonValue{rings: rings, dim: dim}
}

func (pt *PolygonType) appendValueToBuilder(b array.Builder, v PolygonValue) {
	outerListBuilder := b.(*array.ListBuilder)
	outerListBuilder.Append(true)

	innerListBuilder := outerListBuilder.ValueBuilder().(*array.ListBuilder)
	structBuilder := innerListBuilder.ValueBuilder().(*array.StructBuilder)

	coordStruct := coordStructFromStorage(pt.StorageType())
	nFields := coordStruct.NumFields()
	stride := v.dim.NDim()

	for _, ring := range v.rings {
		innerListBuilder.Append(true)
		nVerts := len(ring) / stride
		for vi := 0; vi < nVerts; vi++ {
			structBuilder.Append(true)
			for f := 0; f < nFields; f++ {
				structBuilder.FieldBuilder(f).(*array.Float64Builder).Append(ring[vi*stride+f])
			}
		}
	}
}

func (pt *PolygonType) valueFromString(s string) (PolygonValue, error) {
	s = strings.TrimSpace(s)
	upper := strings.ToUpper(s)
	if !strings.HasPrefix(upper, "POLYGON") {
		return PolygonValue{}, fmt.Errorf("invalid polygon WKT: %s", s)
	}

	if strings.Contains(upper, "EMPTY") {
		return PolygonValue{}, nil
	}

	// Determine dimension from prefix before first '('
	firstParen := strings.Index(s, "(")
	if firstParen == -1 {
		return PolygonValue{}, fmt.Errorf("invalid polygon WKT: missing '(': %s", s)
	}
	prefix := strings.ToUpper(strings.TrimSpace(s[:firstParen]))

	// Strip "POLYGON" prefix and outer parens
	body := strings.TrimSpace(s[firstParen:])
	if !strings.HasPrefix(body, "(") || !strings.HasSuffix(body, ")") {
		return PolygonValue{}, fmt.Errorf("invalid polygon WKT: %s", s)
	}
	body = body[1 : len(body)-1] // remove outer ( )

	// Split rings by "),(" pattern
	var ringStrs []string
	depth := 0
	start := 0
	for i, ch := range body {
		switch ch {
		case '(':
			if depth == 0 {
				start = i + 1
			}
			depth++
		case ')':
			depth--
			if depth == 0 {
				ringStrs = append(ringStrs, body[start:i])
			}
		}
	}

	var rings [][]float64
	var detectedDim Dimension
	for _, ringStr := range ringStrs {
		parts := strings.Split(ringStr, ",")
		var coords []float64
		for _, part := range parts {
			fields := strings.Fields(strings.TrimSpace(part))
			if detectedDim == 0 && len(rings) == 0 && len(coords) == 0 {
				switch len(fields) {
				case 2:
					detectedDim = XY
				case 3:
					if strings.Contains(prefix, "M") && !strings.Contains(prefix, "ZM") {
						detectedDim = XYM
					} else {
						detectedDim = XYZ
					}
				case 4:
					detectedDim = XYZM
				}
			}
			for _, f := range fields {
				val, err := strconv.ParseFloat(f, 64)
				if err != nil {
					return PolygonValue{}, fmt.Errorf("invalid coordinate in polygon WKT: %s", f)
				}
				coords = append(coords, val)
			}
		}
		rings = append(rings, coords)
	}

	return PolygonValue{rings: rings, dim: detectedDim}, nil
}

func (pt *PolygonType) unmarshalJSONOne(dec *json.Decoder) (PolygonValue, bool, error) {
	t, err := dec.Token()
	if err != nil {
		return PolygonValue{}, false, err
	}

	if t == nil {
		return PolygonValue{}, true, nil
	}

	// Expect '[' for array of rings
	delim, ok := t.(json.Delim)
	if !ok || delim != '[' {
		return PolygonValue{}, false, fmt.Errorf("expected '[' for Polygon value, got %T(%v)", t, t)
	}

	coordStruct := coordStructFromStorage(pt.StorageType())
	dim := DimensionFromStructType(coordStruct)
	stride := dim.NDim()

	var rings [][]float64
	for dec.More() {
		// Each ring is an array of coordinate arrays
		ringTok, err := dec.Token()
		if err != nil {
			return PolygonValue{}, false, err
		}
		if d, ok := ringTok.(json.Delim); !ok || d != '[' {
			return PolygonValue{}, false, fmt.Errorf("expected '[' for ring, got %T(%v)", ringTok, ringTok)
		}

		var coords []float64
		for dec.More() {
			// Each vertex is [x, y, ...]
			vertTok, err := dec.Token()
			if err != nil {
				return PolygonValue{}, false, err
			}
			if d, ok := vertTok.(json.Delim); !ok || d != '[' {
				return PolygonValue{}, false, fmt.Errorf("expected '[' for vertex, got %T(%v)", vertTok, vertTok)
			}
			for ci := 0; ci < stride; ci++ {
				var f float64
				if err := dec.Decode(&f); err != nil {
					return PolygonValue{}, false, err
				}
				coords = append(coords, f)
			}
			// consume vertex ']'
			if _, err := dec.Token(); err != nil {
				return PolygonValue{}, false, err
			}
		}
		// consume ring ']'
		if _, err := dec.Token(); err != nil {
			return PolygonValue{}, false, err
		}
		rings = append(rings, coords)
	}
	// consume outer ']'
	if _, err := dec.Token(); err != nil {
		return PolygonValue{}, false, err
	}

	return PolygonValue{rings: rings, dim: dim}, false, nil
}

func (pt *PolygonType) NewBuilder(mem memory.Allocator) array.Builder {
	return &valueBuilder[PolygonValue, *PolygonType]{
		ExtensionBuilder: array.NewExtensionBuilder(mem, pt),
	}
}

type PolygonArray = geometryArray[PolygonValue, *PolygonType]
type PolygonBuilder = valueBuilder[PolygonValue, *PolygonType]

var _ array.CustomExtensionBuilder = (*PolygonType)(nil)
