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
	"sort"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/decimal"
	"github.com/apache/arrow-go/v18/arrow/extensions"
	"github.com/apache/arrow-go/v18/parquet/variant"
)

// Variant shredding inference: a field shreds only when all its non-null values
// share one type family (the integer and decimal families widen to their widest
// observed member); a field mixing more than one family is not shreddable and
// stays in the residual value. Frequency/depth/field-count bounds and per-object-
// node caps match Java; no global budget.
const (
	minFieldFrequency     = 0.10
	maxShreddedFields     = 300
	maxShreddingDepth     = 50
	maxIntermediateFields = 1000
)

// integerPriority and decimalPriority pick the widest observed family member.
var integerPriority = map[variant.Type]int{
	variant.Int8: 0, variant.Int16: 1, variant.Int32: 2, variant.Int64: 3,
}

var decimalPriority = map[variant.Type]int{
	variant.Decimal4: 0, variant.Decimal8: 1, variant.Decimal16: 2,
}

type fieldInfo struct {
	admitted            variant.Type
	hasAdmitted         bool
	mixed               bool
	observationCount    int
	maxDecimalScale     int
	maxDecimalIntDigits int
}

func newFieldInfo() *fieldInfo {
	return &fieldInfo{}
}

// observe records one value at this node, folding its type into the admitted type.
func (f *fieldInfo) observe(v variant.Value) {
	f.observationCount++

	t := v.Type()
	// arrow-go has a single Bool type, so no false-folds-to-true step.
	f.admit(t)

	if isDecimalType(t) {
		intDigits, scale := decimalDigits(v.Value())
		if scale > f.maxDecimalScale {
			f.maxDecimalScale = scale
		}
		if intDigits > f.maxDecimalIntDigits {
			f.maxDecimalIntDigits = intDigits
		}
	}
}

// admit widens the admitted type with t, or marks the node mixed when t belongs to
// a different family.
func (f *fieldInfo) admit(t variant.Type) {
	if f.mixed {
		return
	}
	if !f.hasAdmitted {
		f.admitted, f.hasAdmitted = t, true

		return
	}
	merged, ok := mergeFamily(f.admitted, t)
	if !ok {
		f.mixed = true

		return
	}
	f.admitted = merged
}

// mergeFamily widens current with candidate within the integer or decimal family,
// returning ok=false when they belong to different families (mirroring Java's
// VariantShreddingAnalyzer.mergeFamily).
func mergeFamily(current, candidate variant.Type) (variant.Type, bool) {
	if current == candidate {
		return current, true
	}
	if isIntegerType(current) && isIntegerType(candidate) {
		if integerPriority[candidate] > integerPriority[current] {
			return candidate, true
		}

		return current, true
	}
	if isDecimalType(current) && isDecimalType(candidate) {
		if decimalPriority[candidate] > decimalPriority[current] {
			return candidate, true
		}

		return current, true
	}

	return 0, false
}

// admittedType returns the single type this node shreds to, or ok=false when the
// node saw no values or mixed more than one type family.
func (f *fieldInfo) admittedType() (variant.Type, bool) {
	if !f.hasAdmitted || f.mixed {
		return 0, false
	}

	return f.admitted, true
}

func isIntegerType(t variant.Type) bool {
	switch t {
	case variant.Int8, variant.Int16, variant.Int32, variant.Int64:
		return true
	}

	return false
}

func isDecimalType(t variant.Type) bool {
	switch t {
	case variant.Decimal4, variant.Decimal8, variant.Decimal16:
		return true
	}

	return false
}

// decimalDigits returns the integer-digit count and scale of a variant decimal.
func decimalDigits(val any) (intDigits, scale int) {
	switch d := val.(type) {
	case variant.DecimalValue[decimal.Decimal32]:
		return coefficientDigits(d.Value.ToString(0)) - int(d.Scale), int(d.Scale)
	case variant.DecimalValue[decimal.Decimal64]:
		return coefficientDigits(d.Value.ToString(0)) - int(d.Scale), int(d.Scale)
	case variant.DecimalValue[decimal.Decimal128]:
		return coefficientDigits(d.Value.ToString(0)) - int(d.Scale), int(d.Scale)
	}

	return 0, 0
}

func coefficientDigits(s string) int {
	if len(s) > 0 && s[0] == '-' {
		s = s[1:]
	}

	return len(s)
}

type pathNode struct {
	info           *fieldInfo
	objectChildren map[string]*pathNode
	arrayElement   *pathNode
}

func newPathNode() *pathNode {
	return &pathNode{info: newFieldInfo(), objectChildren: make(map[string]*pathNode)}
}

// AnalyzeVariantShredding infers the inner Arrow type to shred the sample by, or
// ok=false to not shred. The result is the INNER type for NewShreddedVariantType.
func AnalyzeVariantShredding(sample []variant.Value) (arrow.DataType, bool) {
	if len(sample) == 0 {
		return nil, false
	}

	root := newPathNode()
	for _, v := range sample {
		traverseVariant(root, v, 0)
	}

	rootType, ok := root.info.admittedType()
	if !ok {
		return nil, false
	}

	pruneInfrequent(root, root.info.observationCount)

	dt := buildInnerType(root, rootType)
	if dt == nil {
		return nil, false
	}

	return dt, true
}

func traverseVariant(node *pathNode, v variant.Value, depth int) {
	t := v.Type()
	if t == variant.Null {
		return
	}

	node.info.observe(v)

	switch {
	case t == variant.Object && depth < maxShreddingDepth:
		obj, ok := v.Value().(variant.ObjectValue)
		if !ok {
			return
		}
		for name, fv := range obj.Values() {
			child := node.objectChildren[name]
			if child == nil {
				if len(node.objectChildren) >= maxIntermediateFields {
					continue
				}
				child = newPathNode()
				node.objectChildren[name] = child
			}
			traverseVariant(child, fv, depth+1)
		}
	case t == variant.Array && depth < maxShreddingDepth:
		arr, ok := v.Value().(variant.ArrayValue)
		if !ok {
			return
		}
		if node.arrayElement == nil {
			node.arrayElement = newPathNode()
		}
		for ev := range arr.Values() {
			traverseVariant(node.arrayElement, ev, depth+1)
		}
	}
}

func pruneInfrequent(node *pathNode, totalRows int) {
	if len(node.objectChildren) == 0 && node.arrayElement == nil {
		return
	}

	// Frequency floor (strict <); exactly minFieldFrequency is kept.
	for name, child := range node.objectChildren {
		if float64(child.info.observationCount)/float64(totalRows) < minFieldFrequency {
			delete(node.objectChildren, name)
		}
	}

	// Cap to maxShreddedFields, keeping the highest count; on a count tie keep the
	// alphabetically-smaller name.
	if len(node.objectChildren) > maxShreddedFields {
		names := make([]string, 0, len(node.objectChildren))
		for name := range node.objectChildren {
			names = append(names, name)
		}
		sort.Slice(names, func(i, j int) bool {
			ci := node.objectChildren[names[i]].info.observationCount
			cj := node.objectChildren[names[j]].info.observationCount
			if ci != cj {
				return ci > cj
			}

			return names[i] < names[j]
		})
		for _, name := range names[maxShreddedFields:] {
			delete(node.objectChildren, name)
		}
	}

	for _, child := range node.objectChildren {
		pruneInfrequent(child, totalRows)
	}
	if node.arrayElement != nil {
		pruneInfrequent(node.arrayElement, totalRows)
	}
}

// buildInnerType emits the inner Arrow type for a node given its chosen type.
// Returns nil when nothing shreddable remains (so the caller does not shred).
func buildInnerType(node *pathNode, t variant.Type) arrow.DataType {
	switch t {
	case variant.Object:
		if len(node.objectChildren) == 0 {
			return nil
		}
		names := make([]string, 0, len(node.objectChildren))
		for name := range node.objectChildren {
			names = append(names, name)
		}
		sort.Strings(names) // alphabetical for deterministic field order
		fields := make([]arrow.Field, 0, len(names))
		for _, name := range names {
			child := node.objectChildren[name]
			ct, ok := child.info.admittedType()
			if !ok {
				continue
			}
			cdt := buildInnerType(child, ct)
			if cdt == nil {
				continue
			}
			fields = append(fields, arrow.Field{Name: name, Type: cdt, Nullable: true})
		}
		if len(fields) == 0 {
			return nil
		}

		return arrow.StructOf(fields...)
	case variant.Array:
		if node.arrayElement == nil {
			return nil
		}
		et, ok := node.arrayElement.info.admittedType()
		if !ok {
			return nil
		}
		edt := buildInnerType(node.arrayElement, et)
		if edt == nil {
			return nil
		}

		return arrow.ListOf(edt)
	default:
		return primitiveArrowType(t, node.info)
	}
}

// primitiveArrowType maps a scalar variant type to the Arrow leaf type that
// arrow-go's shredded builder accepts. Returns nil for types that cannot shred.
func primitiveArrowType(t variant.Type, info *fieldInfo) arrow.DataType {
	switch t {
	case variant.Bool:
		return arrow.FixedWidthTypes.Boolean
	case variant.Int8:
		return arrow.PrimitiveTypes.Int8
	case variant.Int16:
		return arrow.PrimitiveTypes.Int16
	case variant.Int32:
		return arrow.PrimitiveTypes.Int32
	case variant.Int64:
		return arrow.PrimitiveTypes.Int64
	case variant.Float:
		return arrow.PrimitiveTypes.Float32
	case variant.Double:
		return arrow.PrimitiveTypes.Float64
	case variant.String:
		return arrow.BinaryTypes.String
	case variant.Binary:
		return arrow.BinaryTypes.Binary
	case variant.Date:
		return arrow.FixedWidthTypes.Date32
	case variant.Time:
		return arrow.FixedWidthTypes.Time64us
	case variant.TimestampMicros:
		return &arrow.TimestampType{Unit: arrow.Microsecond, TimeZone: "UTC"}
	case variant.TimestampMicrosNTZ:
		return &arrow.TimestampType{Unit: arrow.Microsecond}
	case variant.TimestampNanos:
		return &arrow.TimestampType{Unit: arrow.Nanosecond, TimeZone: "UTC"}
	case variant.TimestampNanosNTZ:
		return &arrow.TimestampType{Unit: arrow.Nanosecond}
	case variant.UUID:
		return extensions.NewUUIDType()
	case variant.Decimal4, variant.Decimal8, variant.Decimal16:
		return decimalArrowType(info)
	}

	return nil
}

func decimalArrowType(info *fieldInfo) arrow.DataType {
	// Always Decimal128: arrow-go's pqarrow maps it to INT32/INT64/FLBA by
	// precision and cannot serialize Decimal32/Decimal64.
	intDigits := max(info.maxDecimalIntDigits, 0)
	prec := min(intDigits+info.maxDecimalScale, 38)
	if prec < 1 {
		prec = 1
	}
	scale := info.maxDecimalScale
	if maxScale := 38 - intDigits; scale > maxScale {
		if maxScale < 0 {
			maxScale = 0
		}
		scale = maxScale
	}

	return &arrow.Decimal128Type{Precision: int32(prec), Scale: int32(scale)}
}
