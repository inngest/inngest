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

package iceberg

import (
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/decimal"
	"github.com/apache/arrow-go/v18/arrow/decimal128"
	"github.com/apache/arrow-go/v18/parquet/variant"
	"github.com/apache/iceberg-go/internal"
	"github.com/google/uuid"
)

const (
	microsPerDay  = int64(86_400_000_000)
	nanosPerDay   = int64(86_400_000_000_000)
	nanosPerMicro = int64(1_000)
)

// CastVariantLiteral casts a leaf variant value to typ and wraps it as a Literal. Exported for table/internal; not part of the stable public API.
func CastVariantLiteral(v variant.Value, typ PrimitiveType) (Literal, bool) {
	result, ok := castVariantValue(v, typ)
	if !ok {
		return nil, false
	}

	lit := literalFromCastValue(result)
	if lit == nil {
		return nil, false
	}
	if !lit.Type().Equals(typ) {
		conv, err := lit.To(typ)
		if err != nil {
			return nil, false
		}

		lit = conv
	}

	return lit, true
}

// castVariantValue casts a leaf variant value to the Go value backing typ.
func castVariantValue(v variant.Value, typ PrimitiveType) (any, bool) {
	raw := v.Value()
	if raw == nil {
		return nil, false
	}

	if r, ok := exactVariantMatch(v.Type(), raw, typ); ok {
		return r, true
	}

	switch t := typ.(type) {
	case Int32Type:
		switch n := raw.(type) {
		case int8:
			return int32(n), true
		case int16:
			return int32(n), true
		}
	case Int64Type:
		switch n := raw.(type) {
		case int8:
			return int64(n), true
		case int16:
			return int64(n), true
		case int32:
			return int64(n), true
		}
	case Float64Type:
		if f, ok := raw.(float32); ok {
			return float64(f), true
		}
	case FixedType:
		if b, ok := raw.([]byte); ok && len(b) == t.Len() {
			return b, true
		}
	case DecimalType:
		return castVariantDecimal(raw, t)
	case BooleanType:
		if b, ok := raw.(bool); ok {
			return b, true
		}
	case TimestampType:
		return castVariantToMicros(v.Type(), raw, false)
	case TimestampTzType:
		return castVariantToMicros(v.Type(), raw, true)
	case TimestampNsType:
		return castVariantToNanos(v.Type(), raw, false)
	case TimestampTzNsType:
		return castVariantToNanos(v.Type(), raw, true)
	case DateType:
		return castVariantToDate(v.Type(), raw)
	}

	return nil, false
}

// exactVariantMatch returns raw coerced to the Go type backing typ when the variant physical type matches typ exactly.
func exactVariantMatch(pt variant.Type, raw any, typ PrimitiveType) (any, bool) {
	switch typ.(type) {
	case Int32Type:
		if pt == variant.Int32 {
			return raw.(int32), true
		}
	case Int64Type:
		if pt == variant.Int64 {
			return raw.(int64), true
		}
	case Float32Type:
		if pt == variant.Float {
			return raw.(float32), true
		}
	case Float64Type:
		if pt == variant.Double {
			return raw.(float64), true
		}
	case DateType:
		if pt == variant.Date {
			return Date(raw.(arrow.Date32)), true
		}
	case TimestampType:
		if pt == variant.TimestampMicrosNTZ {
			return Timestamp(raw.(arrow.Timestamp)), true
		}
	case TimestampTzType:
		if pt == variant.TimestampMicros {
			return Timestamp(raw.(arrow.Timestamp)), true
		}
	case TimestampNsType:
		if pt == variant.TimestampNanosNTZ {
			return TimestampNano(raw.(arrow.Timestamp)), true
		}
	case TimestampTzNsType:
		if pt == variant.TimestampNanos {
			return TimestampNano(raw.(arrow.Timestamp)), true
		}
	case TimeType:
		if pt == variant.Time {
			return Time(raw.(arrow.Time64)), true
		}
	case UUIDType:
		if pt == variant.UUID {
			return raw.(uuid.UUID), true
		}
	case StringType:
		if pt == variant.String {
			return raw.(string), true
		}
	case BinaryType:
		if pt == variant.Binary {
			return raw.([]byte), true
		}
	}

	return nil, false
}

func castVariantDecimal(raw any, typ DecimalType) (any, bool) {
	switch d := raw.(type) {
	case variant.DecimalValue[decimal.Decimal32]:
		if int(d.Scale) != typ.Scale() {
			return nil, false
		}

		return Decimal{Val: decimal128.FromI64(int64(d.Value.(decimal.Decimal32))), Scale: int(d.Scale)}, true
	case variant.DecimalValue[decimal.Decimal64]:
		if int(d.Scale) != typ.Scale() {
			return nil, false
		}

		return Decimal{Val: decimal128.FromI64(int64(d.Value.(decimal.Decimal64))), Scale: int(d.Scale)}, true
	case variant.DecimalValue[decimal.Decimal128]:
		if int(d.Scale) != typ.Scale() {
			return nil, false
		}

		return Decimal{Val: d.Value.(decimal.Decimal128), Scale: int(d.Scale)}, true
	}

	return nil, false
}

// castVariantToMicros converts a nanosecond or date leaf to a microsecond timestamp; the source's tz-awareness must match the target's (tz).
func castVariantToMicros(pt variant.Type, raw any, tz bool) (any, bool) {
	switch {
	case tz && pt == variant.TimestampNanos, !tz && pt == variant.TimestampNanosNTZ:
		return Timestamp(internal.FloorDiv(int64(raw.(arrow.Timestamp)), nanosPerMicro)), true
	case !tz && pt == variant.Date:
		if micros, ok := mulNoOverflow(int64(raw.(arrow.Date32)), microsPerDay); ok {
			return Timestamp(micros), true
		}
	}

	return nil, false
}

// castVariantToNanos converts a microsecond or date leaf to a nanosecond timestamp; the source's tz-awareness must match the target's (tz).
func castVariantToNanos(pt variant.Type, raw any, tz bool) (any, bool) {
	switch {
	case tz && pt == variant.TimestampMicros, !tz && pt == variant.TimestampMicrosNTZ:
		if nanos, ok := mulNoOverflow(int64(raw.(arrow.Timestamp)), nanosPerMicro); ok {
			return TimestampNano(nanos), true
		}
	case !tz && pt == variant.Date:
		if nanos, ok := mulNoOverflow(int64(raw.(arrow.Date32)), nanosPerDay); ok {
			return TimestampNano(nanos), true
		}
	}

	return nil, false
}

// castVariantToDate converts a zoneless timestamp leaf to a date; a tz-aware instant needs a zone and is rejected.
func castVariantToDate(pt variant.Type, raw any) (any, bool) {
	switch pt {
	case variant.TimestampMicrosNTZ:
		return Date(internal.FloorDiv(int64(raw.(arrow.Timestamp)), microsPerDay)), true
	case variant.TimestampNanosNTZ:
		return Date(internal.FloorDiv(int64(raw.(arrow.Timestamp)), nanosPerDay)), true
	}

	return nil, false
}

// mulNoOverflow returns a*b, reporting false if the product overflows int64.
func mulNoOverflow(a, b int64) (int64, bool) {
	p := a * b
	if a != 0 && p/a != b {
		return 0, false
	}

	return p, true
}

func literalFromCastValue(result any) Literal {
	switch r := result.(type) {
	case bool:
		return NewLiteral(r)
	case int32:
		return NewLiteral(r)
	case int64:
		return NewLiteral(r)
	case float32:
		return NewLiteral(r)
	case float64:
		return NewLiteral(r)
	case Date:
		return NewLiteral(r)
	case Time:
		return NewLiteral(r)
	case Timestamp:
		return NewLiteral(r)
	case TimestampNano:
		return NewLiteral(r)
	case string:
		return NewLiteral(r)
	case []byte:
		return NewLiteral(r)
	case uuid.UUID:
		return NewLiteral(r)
	case Decimal:
		return NewLiteral(r)
	}

	return nil
}
