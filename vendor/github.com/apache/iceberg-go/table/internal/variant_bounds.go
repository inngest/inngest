// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package internal

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/decimal"
	"github.com/apache/arrow-go/v18/parquet/variant"
	"github.com/apache/iceberg-go"
	"github.com/google/uuid"
)

// isNaNLiteral reports whether a float literal is NaN (bounds are skipped for it, per Java).
func isNaNLiteral(lit iceberg.Literal) bool {
	switch v := lit.Any().(type) {
	case float32:
		return math.IsNaN(float64(v))
	case float64:
		return math.IsNaN(v)
	}

	return false
}

// variantLeaf is one shredded typed_value primitive leaf that can carry a bound.
type variantLeaf struct {
	jsonPath    string
	typedPath   string
	ownResidual string
	icebergType iceberg.PrimitiveType
}

// enumerateVariantLeaves returns a shredded variant's bound-bearing primitive leaves.
func enumerateVariantLeaves(variantColPath []string, tv arrow.Field) []variantLeaf {
	var out []variantLeaf
	rootValue := joinPath(append(cloneStrs(variantColPath), "value"))
	walkVariantTyped(append(cloneStrs(variantColPath), "typed_value"), rootValue, nil, tv.Type, &out)

	return out
}

// walkVariantTyped collects primitive leaves from a shredded typed_value subtree, carrying each leaf's own residual value path.
func walkVariantTyped(typedPath []string, ownResidual string, fields []string, dt arrow.DataType, out *[]variantLeaf) {
	switch dt.(type) {
	case *arrow.StructType:
		t := dt.(*arrow.StructType)
		for _, f := range t.Fields() {
			grp, ok := f.Type.(*arrow.StructType)
			if !ok {
				continue
			}
			tvIdx, hasTyped := grp.FieldIdx("typed_value")
			if !hasTyped {
				continue
			}
			childResidual := ""
			if _, hasVal := grp.FieldIdx("value"); hasVal {
				childResidual = joinPath(append(cloneStrs(typedPath), f.Name, "value"))
			}
			walkVariantTyped(append(cloneStrs(typedPath), f.Name, "typed_value"), childResidual,
				append(cloneStrs(fields), f.Name), grp.Field(tvIdx).Type, out)
		}
	case *arrow.ListType, *arrow.LargeListType, *arrow.FixedSizeListType:
		return
	default:
		it, ok := arrowLeafToIceberg(dt)
		if !ok {
			return
		}
		*out = append(*out, variantLeaf{
			jsonPath:    iceberg.NormalizeVariantPath(fields),
			typedPath:   joinPath(typedPath),
			ownResidual: ownResidual,
			icebergType: it,
		})
	}
}

// arrowLeafToIceberg maps a shredded leaf's arrow type to its Iceberg type.
func arrowLeafToIceberg(dt arrow.DataType) (iceberg.PrimitiveType, bool) {
	switch t := dt.(type) {
	case *arrow.BooleanType:
		return iceberg.PrimitiveTypes.Bool, true
	case *arrow.Int8Type, *arrow.Int16Type, *arrow.Int32Type:
		return iceberg.PrimitiveTypes.Int32, true
	case *arrow.Int64Type:
		return iceberg.PrimitiveTypes.Int64, true
	case *arrow.Float32Type:
		return iceberg.PrimitiveTypes.Float32, true
	case *arrow.Float64Type:
		return iceberg.PrimitiveTypes.Float64, true
	case *arrow.StringType:
		return iceberg.PrimitiveTypes.String, true
	case *arrow.BinaryType:
		return iceberg.PrimitiveTypes.Binary, true
	case *arrow.Date32Type:
		return iceberg.PrimitiveTypes.Date, true
	case *arrow.Time64Type:
		if t.Unit == arrow.Microsecond {
			return iceberg.PrimitiveTypes.Time, true
		}
	case *arrow.TimestampType:
		switch t.Unit {
		case arrow.Microsecond:
			if t.TimeZone == "" {
				return iceberg.PrimitiveTypes.Timestamp, true
			}

			return iceberg.PrimitiveTypes.TimestampTz, true
		case arrow.Nanosecond:
			if t.TimeZone == "" {
				return iceberg.PrimitiveTypes.TimestampNs, true
			}

			return iceberg.PrimitiveTypes.TimestampTzNs, true
		}
	case *arrow.Decimal128Type:
		return iceberg.DecimalTypeOf(int(t.Precision), int(t.Scale)), true
	case arrow.ExtensionType:
		if t.ExtensionName() == "arrow.uuid" {
			return iceberg.PrimitiveTypes.UUID, true
		}
	}

	return nil, false
}

func cloneStrs(s []string) []string { return append([]string(nil), s...) }

func joinPath(s []string) string { return strings.Join(s, ".") }

// variantFieldBound is a shredded field's lower bound and upper bound; a nil upper is omitted.
type variantFieldBound struct {
	jsonPath     string
	icebergType  iceberg.PrimitiveType
	lower, upper iceberg.Literal
}

// truncateVariantBound truncates string/binary leaf bounds to truncLen; a nil upper drops that bound.
func truncateVariantBound(it iceberg.PrimitiveType, lo, hi iceberg.Literal, truncLen int) (iceberg.Literal, iceberg.Literal) {
	if truncLen <= 0 {
		return lo, hi
	}
	switch it.(type) {
	case iceberg.StringType:
		s := lo.Any().(string)
		if truncated, ok := truncateString(s, truncLen); ok {
			lo = iceberg.StringLiteral(strings.Clone(truncated))
		}

		s = hi.Any().(string)
		if up, truncated := truncateUpperBoundText(s, truncLen); truncated {
			if up != "" {
				hi = iceberg.StringLiteral(up)
			} else {
				hi = nil
			}
		}
	case iceberg.BinaryType:
		if b := lo.Any().([]byte); len(b) > truncLen {
			lo = iceberg.BinaryLiteral(slices.Clone(b[:truncLen]))
		}
		if b := hi.Any().([]byte); len(b) > truncLen {
			if up := TruncateUpperBoundBinary(b, truncLen); len(up) > 0 {
				hi = iceberg.BinaryLiteral(up)
			} else {
				hi = nil
			}
		}
	}

	return lo, hi
}

// serializeVariantBounds builds the lower and upper bound objects, per spec.
func serializeVariantBounds(fields []variantFieldBound) (lower, upper []byte, ok bool, err error) {
	if len(fields) == 0 {
		return nil, nil, false, nil
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].jsonPath < fields[j].jsonPath })

	if lower, err = boundsObjectBytes(fields, true); err != nil {
		return nil, nil, false, err
	}
	if upper, err = boundsObjectBytes(fields, false); err != nil {
		return nil, nil, false, err
	}

	return lower, upper, true, nil
}

func boundsObjectValue(fields []variantFieldBound, useLower bool) (v variant.Value, err error) {
	// Convert a malformed-literal builder panic to an error, as shredVariantColumn does.
	defer func() {
		if r := recover(); r != nil {
			v, err = variant.Value{}, fmt.Errorf("variant bounds: %v", r)
		}
	}()

	var b variant.Builder
	start := b.Offset()
	entries := make([]variant.FieldEntry, 0, len(fields))
	for _, f := range fields {
		lit := f.upper
		if useLower {
			lit = f.lower
		}
		if lit == nil {
			continue
		}
		entries = append(entries, b.NextField(start, f.jsonPath))
		if err := appendLiteralToVariant(&b, f.icebergType, lit); err != nil {
			return variant.Value{}, err
		}
	}
	if err := b.FinishObject(start, entries); err != nil {
		return variant.Value{}, err
	}

	return b.Build()
}

func boundsObjectBytes(fields []variantFieldBound, useLower bool) ([]byte, error) {
	v, err := boundsObjectValue(fields, useLower)
	if err != nil {
		return nil, err
	}

	meta := v.Metadata().Bytes()
	val := v.Bytes()
	out := make([]byte, 0, len(meta)+len(val))
	out = append(out, meta...)
	out = append(out, val...)

	return out, nil
}

// appendLiteralToVariant appends a bound, reading the literal by physical type.
func appendLiteralToVariant(b *variant.Builder, it iceberg.PrimitiveType, lit iceberg.Literal) error {
	switch t := it.(type) {
	case iceberg.BooleanType:
		return b.AppendBool(lit.Any().(bool))
	case iceberg.Int32Type:
		return b.AppendInt(int64(lit.Any().(int32)))
	case iceberg.Int64Type:
		return b.AppendInt(lit.Any().(int64))
	case iceberg.Float32Type:
		return b.AppendFloat32(lit.Any().(float32))
	case iceberg.Float64Type:
		return b.AppendFloat64(lit.Any().(float64))
	case iceberg.StringType:
		return b.AppendString(lit.Any().(string))
	case iceberg.BinaryType:
		return b.AppendBinary(lit.Any().([]byte))
	case iceberg.DateType:
		return b.AppendDate(arrow.Date32(lit.Any().(int32)))
	case iceberg.TimeType:
		return b.AppendTimeMicro(arrow.Time64(lit.Any().(int64)))
	case iceberg.TimestampType:
		return b.AppendTimestamp(arrow.Timestamp(lit.Any().(int64)), true, false)
	case iceberg.TimestampTzType:
		return b.AppendTimestamp(arrow.Timestamp(lit.Any().(int64)), true, true)
	case iceberg.TimestampNsType:
		return b.AppendTimestamp(arrow.Timestamp(lit.Any().(int64)), false, false)
	case iceberg.TimestampTzNsType:
		return b.AppendTimestamp(arrow.Timestamp(lit.Any().(int64)), false, true)
	case iceberg.UUIDType:
		return b.AppendUUID(lit.Any().(uuid.UUID))
	case iceberg.DecimalType:
		return appendDecimalToVariant(b, t, lit.Any())
	}

	return fmt.Errorf("variant bounds: unsupported leaf type %s", it)
}

// appendDecimalToVariant appends a decimal bound from an int- or Decimal-typed literal.
func appendDecimalToVariant(b *variant.Builder, t iceberg.DecimalType, v any) error {
	scale := uint8(t.Scale())
	switch d := v.(type) {
	case int32:
		return b.AppendDecimal4(scale, decimal.Decimal32(d))
	case int64:
		return b.AppendDecimal8(scale, decimal.Decimal64(d))
	case iceberg.Decimal:
		switch {
		case t.Precision() <= 9:
			return b.AppendDecimal4(scale, decimal.Decimal32(int64(d.Val.LowBits())))
		case t.Precision() <= 18:
			return b.AppendDecimal8(scale, decimal.Decimal64(int64(d.Val.LowBits())))
		default:
			return b.AppendDecimal16(scale, d.Val)
		}
	}

	return fmt.Errorf("variant bounds: unsupported decimal value %T", v)
}

// VariantBoundLiteral decodes the stored variant bound object at the normalized path and casts it to typ.
func VariantBoundLiteral(raw []byte, path string, typ iceberg.PrimitiveType) (lit iceberg.Literal, ok bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			lit, ok, err = nil, false, fmt.Errorf("variant bound decode: %v", r)
		}
	}()

	meta, err := variant.NewMetadata(raw)
	if err != nil {
		return nil, false, err
	}

	v, err := variant.New(raw[:meta.SizeBytes()], raw[meta.SizeBytes():])
	if err != nil {
		return nil, false, err
	}

	obj, isObj := v.Value().(variant.ObjectValue)
	if !isObj {
		return nil, false, nil
	}

	field, err := obj.ValueByKey(path)
	if err != nil {
		return nil, false, nil
	}

	lit, ok = iceberg.CastVariantLiteral(field.Value, typ)

	return lit, ok, nil
}
