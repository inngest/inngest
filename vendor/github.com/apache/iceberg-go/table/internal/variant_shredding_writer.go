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

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/extensions"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

// canShredVariant reports whether an inner type is accepted by arrow-go's shredded
// builder; NewVariantBuilder panics on an unsupported leaf, so callers fall back.
func canShredVariant(dt arrow.DataType) bool {
	switch t := dt.(type) {
	case *arrow.StructType:
		for _, f := range t.Fields() {
			if !canShredVariant(f.Type) {
				return false
			}
		}

		return t.NumFields() > 0
	case *arrow.ListType:
		return canShredVariant(t.Elem())
	case *arrow.TimestampType:
		if t.Unit != arrow.Microsecond && t.Unit != arrow.Nanosecond {
			return false
		}

		return t.TimeZone == "" || t.TimeZone == "UTC"
	case *arrow.Time64Type:
		return t.Unit == arrow.Microsecond
	case arrow.ExtensionType:
		return t.ExtensionName() == "arrow.uuid"
	}

	switch dt.ID() {
	case arrow.BOOL, arrow.INT8, arrow.INT16, arrow.INT32, arrow.INT64,
		arrow.FLOAT32, arrow.FLOAT64, arrow.STRING, arrow.BINARY, arrow.DATE32,
		arrow.DECIMAL128: // pqarrow cannot serialize Decimal32/Decimal64
		return true
	}

	return false
}

// ShreddedArrowSchema swaps each top-level variant field in inferred (that passes
// canShredVariant) for a shredded type, preserving name/nullability/field-id.
func ShreddedArrowSchema(base *arrow.Schema, inferred map[int]arrow.DataType) *arrow.Schema {
	if len(inferred) == 0 {
		return base
	}

	fields := base.Fields()
	out := make([]arrow.Field, len(fields))
	copy(out, fields)
	changed := false
	for idx, inner := range inferred {
		if idx < 0 || idx >= len(out) || inner == nil || !canShredVariant(inner) {
			continue
		}
		out[idx].Type = extensions.NewShreddedVariantType(inner)
		changed = true
	}
	if !changed {
		return base
	}

	md := base.Metadata()

	return arrow.NewSchema(out, &md)
}

// ShredRecordVariants rebuilds the shredded-variant columns of rec to match
// shreddedSchema (the inverse of UnshredVariant); other columns pass through.
func ShredRecordVariants(rec arrow.RecordBatch, shreddedSchema *arrow.Schema, mem memory.Allocator) (out arrow.RecordBatch, err error) {
	shredIdx := make([]int, 0)
	for i := range shreddedSchema.NumFields() {
		if vt, ok := shreddedSchema.Field(i).Type.(*extensions.VariantType); ok && vt.TypedValue().Type != nil {
			shredIdx = append(shredIdx, i)
		}
	}
	if len(shredIdx) == 0 {
		rec.Retain()

		return rec, nil
	}

	cols := make([]arrow.Array, rec.NumCols())
	built := make([]bool, rec.NumCols())
	defer func() {
		// On error, release any arrays we created so far.
		if err != nil {
			for i, b := range built {
				if b && cols[i] != nil {
					cols[i].Release()
				}
			}
		}
	}()

	shredSet := make(map[int]bool, len(shredIdx))
	for _, i := range shredIdx {
		shredSet[i] = true
	}

	for i := range int(rec.NumCols()) {
		if !shredSet[i] {
			cols[i] = rec.Column(i)

			continue
		}
		vt := shreddedSchema.Field(i).Type.(*extensions.VariantType)
		src, ok := rec.Column(i).(*extensions.VariantArray)
		if !ok {
			return nil, fmt.Errorf("variant shredding: column %d is not a VariantArray", i)
		}
		arr, berr := shredVariantColumn(src, vt, mem)
		if berr != nil {
			return nil, berr
		}
		cols[i] = arr
		built[i] = true
	}

	out = array.NewRecordBatch(shreddedSchema, cols, rec.NumRows())
	// NewRecordBatch retains each column; drop our refs on the ones we built.
	for i, b := range built {
		if b {
			cols[i].Release()
		}
	}

	return out, nil
}

// shredVariantColumn builds the shredded form, branching on Storage().IsNull.
func shredVariantColumn(src *extensions.VariantArray, vt *extensions.VariantType, mem memory.Allocator) (arr arrow.Array, err error) {
	bldr := extensions.NewVariantBuilder(mem, vt)
	defer bldr.Release()
	bldr.Reserve(src.Len())

	// Convert an arrow-go builder panic on malformed input into an error, not a crash.
	defer func() {
		if r := recover(); r != nil {
			arr, err = nil, fmt.Errorf("variant shredding: %v", r)
		}
	}()

	storage := src.Storage()
	for i := range src.Len() {
		if storage.IsNull(i) {
			bldr.AppendNull()

			continue
		}
		v, verr := src.Value(i)
		if verr != nil {
			return nil, fmt.Errorf("variant shredding: row %d: %w", i, verr)
		}
		bldr.Append(v)
	}

	return bldr.NewArray(), nil
}
