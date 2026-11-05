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
	"bytes"
	"cmp"
	"encoding"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"slices"
	"strconv"
	"time"
	"unsafe"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/decimal128"
	"github.com/apache/arrow-go/v18/parquet/variant"
	"github.com/google/uuid"
)

// LiteralType is a generic type constraint for the explicit Go types that we allow
// for literal values. This represents the actual primitive types that exist in Iceberg
type LiteralType interface {
	bool | int32 | int64 | float32 | float64 | Date |
		Time | Timestamp | TimestampNano | string | []byte | uuid.UUID | Decimal |
		variant.Value
}

// Comparator is a comparison function for specific literal types:
//
//	returns 0 if v1 == v2
//	returns <0 if v1 < v2
//	returns >0 if v1 > v2
type Comparator[T LiteralType] func(v1, v2 T) int

// Literal is a non-null literal value. It can be casted using To and be checked for
// equality against other literals.
type Literal interface {
	fmt.Stringer
	encoding.BinaryMarshaler

	Any() any
	Type() Type
	To(Type) (Literal, error)
	Equals(Literal) bool
}

// TypedLiteral is a generic interface for Literals so that you can retrieve the value.
// This is based on the physical representative type, which means that FixedLiteral and
// BinaryLiteral will both return []byte, etc.
type TypedLiteral[T LiteralType] interface {
	Literal

	Value() T
	Comparator() Comparator[T]
}

type NumericLiteral interface {
	Literal
	Increment() Literal
	Decrement() Literal
}

// NewLiteral provides a literal based on the type of T.
//
// The type switch is performed on &val (a pointer) rather than val itself.
// Boxing a small scalar such as bool into an interface makes the compiler
// reference runtime.staticuint64s, which emits a relocation the linker rejects
// as misaligned on big-endian platforms such as s390x. Switching on a pointer
// boxes only the pointer, never the scalar, avoiding that reference.
func NewLiteral[T LiteralType](val T) Literal {
	switch v := any(&val).(type) {
	case *bool:
		return BoolLiteral(*v)
	case *int32:
		return Int32Literal(*v)
	case *int64:
		return Int64Literal(*v)
	case *float32:
		return Float32Literal(*v)
	case *float64:
		return Float64Literal(*v)
	case *Date:
		return DateLiteral(*v)
	case *Time:
		return TimeLiteral(*v)
	case *Timestamp:
		return TimestampLiteral(*v)
	case *TimestampNano:
		return TimestampNsLiteral(*v)
	case *string:
		return StringLiteral(*v)
	case *[]byte:
		return BinaryLiteral(*v)
	case *uuid.UUID:
		return UUIDLiteral(*v)
	case *Decimal:
		return DecimalLiteral(*v)
	case *variant.Value:
		return VariantLiteral(*v)
	}
	panic("can't happen due to literal type constraint")
}

func getComparator[T LiteralType]() Comparator[T] {
	var z T

	return NewLiteral(z).(TypedLiteral[T]).Comparator()
}

// LiteralFromBytes uses the defined Iceberg spec for how to serialize a value of
// a the provided type and returns the appropriate Literal value from it.
//
// If you already have a value of the desired Literal type, you could alternatively
// call UnmarshalBinary on it yourself manually.
//
// This is primarily used for retrieving stat values.
func LiteralFromBytes(typ Type, data []byte) (Literal, error) {
	if data == nil {
		return nil, ErrInvalidBinSerialization
	}

	switch t := typ.(type) {
	case BooleanType:
		var v BoolLiteral
		err := v.UnmarshalBinary(data)

		return v, err
	case Int32Type:
		var v Int32Literal
		err := v.UnmarshalBinary(data)

		return v, err
	case Int64Type:
		var v Int64Literal
		err := v.UnmarshalBinary(data)

		return v, err
	case Float32Type:
		var v Float32Literal
		err := v.UnmarshalBinary(data)

		return v, err
	case Float64Type:
		var v Float64Literal
		err := v.UnmarshalBinary(data)

		return v, err
	case StringType:
		var v StringLiteral
		err := v.UnmarshalBinary(data)

		return v, err
	case BinaryType:
		var v BinaryLiteral
		err := v.UnmarshalBinary(data)

		return v, err
	case GeometryType, GeographyType:
		// Geometry/Geography single-value bounds use the Iceberg geospatial
		// serialization (spec Appendix D): little-endian float64 coordinates in
		// X, Y[, Z][, M] order, i.e. 16, 24, or 32 bytes — not WKB. The
		// coordinates have no total order, so they are returned as a GeoLiteral
		// (not a plain BinaryLiteral): it reports the real geo type and is not
		// comparable, so any attempt to order it errors instead of silently
		// running bytes.Compare over the coordinate bytes.
		switch len(data) {
		case 16, 24, 32: // valid coordinate lengths (XY / XYZ|XYM / XYZM)
		default:
			return nil, fmt.Errorf("%w: geometry/geography bound must be 16, 24, or 32 bytes, got %d",
				ErrInvalidBinSerialization, len(data))
		}

		return GeoLiteral{val: data, typ: typ}, nil
	case FixedType:
		if len(data) < t.Len() {
			// looks like some writers will write a prefix of the fixed length
			// for lower/upper bounds instead of the full length. so let's pad
			// it out to the full length if unpacking a fixed length literal
			padded := make([]byte, t.Len())
			copy(padded, data)
			data = padded
		} else if len(data) > t.Len() {
			return nil, fmt.Errorf("%w: %w: fixed[%d] value has %d bytes",
				ErrInvalidBinSerialization, ErrInvalidFixedLength, t.Len(), len(data))
		}
		var v FixedLiteral
		err := v.UnmarshalBinary(data)

		return v, err
	case DecimalType:
		v := DecimalLiteral{Scale: t.scale}
		err := v.UnmarshalBinary(data)

		return v, err
	case DateType:
		var v DateLiteral
		err := v.UnmarshalBinary(data)

		return v, err
	case TimeType:
		var v TimeLiteral
		err := v.UnmarshalBinary(data)

		return v, err
	case TimestampType, TimestampTzType:
		var v TimestampLiteral
		err := v.UnmarshalBinary(data)

		return v, err
	case TimestampTzNsType, TimestampNsType:
		var v TimestampNsLiteral
		err := v.UnmarshalBinary(data)

		return v, err
	case UUIDType:
		var v UUIDLiteral
		err := v.UnmarshalBinary(data)

		return v, err
	}

	return nil, fmt.Errorf("%w: unsupported type %s", ErrType, typ)
}

// convenience to avoid repreating this pattern for primitive types
func literalEq[L interface {
	comparable
	LiteralType
}, T TypedLiteral[L]](lhs T, other Literal) bool {
	rhs, ok := other.(T)
	if !ok {
		return false
	}

	return lhs.Value() == rhs.Value()
}

// AboveMaxLiteral represents values that are above the maximum for their type
// such as values > math.MaxInt32 for an Int32Literal
type AboveMaxLiteral interface {
	Literal

	aboveMax()
}

// BelowMinLiteral represents values that are below the minimum for their type
// such as values < math.MinInt32 for an Int32Literal
type BelowMinLiteral interface {
	Literal

	belowMin()
}

type aboveMaxLiteral[T int32 | int64 | float32 | float64] struct {
	value T
}

func (ab aboveMaxLiteral[T]) MarshalBinary() (data []byte, err error) {
	return nil, fmt.Errorf("%w: cannot marshal above max literal",
		ErrInvalidBinSerialization)
}

func (ab aboveMaxLiteral[T]) aboveMax() {}

func (ab aboveMaxLiteral[T]) Type() Type {
	var z T
	switch any(z).(type) {
	case int32:
		return PrimitiveTypes.Int32
	case int64:
		return PrimitiveTypes.Int64
	case float32:
		return PrimitiveTypes.Float32
	case float64:
		return PrimitiveTypes.Float64
	default:
		panic("should never happen")
	}
}

func (ab aboveMaxLiteral[T]) To(t Type) (Literal, error) {
	if ab.Type().Equals(t) {
		return ab, nil
	}

	return nil, fmt.Errorf("%w: cannot change type of AboveMax%sLiteral",
		ErrBadCast, reflect.TypeOf(T(0)).String())
}

func (ab aboveMaxLiteral[T]) Value() T       { return ab.value }
func (ab aboveMaxLiteral[T]) Any() any       { return ab.Value() }
func (ab aboveMaxLiteral[T]) String() string { return "AboveMax" }
func (ab aboveMaxLiteral[T]) Equals(other Literal) bool {
	// AboveMaxLiteral isn't comparable and thus isn't even equal to itself
	return false
}

type belowMinLiteral[T int32 | int64 | float32 | float64] struct {
	value T
}

func (bm belowMinLiteral[T]) MarshalBinary() (data []byte, err error) {
	return nil, fmt.Errorf("%w: cannot marshal below min literal",
		ErrInvalidBinSerialization)
}

func (bm belowMinLiteral[T]) belowMin() {}

func (bm belowMinLiteral[T]) Type() Type {
	var z T
	switch any(z).(type) {
	case int32:
		return PrimitiveTypes.Int32
	case int64:
		return PrimitiveTypes.Int64
	case float32:
		return PrimitiveTypes.Float32
	case float64:
		return PrimitiveTypes.Float64
	default:
		panic("should never happen")
	}
}

func (bm belowMinLiteral[T]) To(t Type) (Literal, error) {
	if bm.Type().Equals(t) {
		return bm, nil
	}

	return nil, fmt.Errorf("%w: cannot change type of BelowMin%sLiteral",
		ErrBadCast, reflect.TypeOf(T(0)).String())
}

func (bm belowMinLiteral[T]) Value() T       { return bm.value }
func (bm belowMinLiteral[T]) Any() any       { return bm.Value() }
func (bm belowMinLiteral[T]) String() string { return "BelowMin" }
func (bm belowMinLiteral[T]) Equals(other Literal) bool {
	// BelowMinLiteral isn't comparable and thus isn't even equal to itself
	return false
}

func Int32AboveMaxLiteral() Literal {
	return aboveMaxLiteral[int32]{value: math.MaxInt32}
}

func Int64AboveMaxLiteral() Literal {
	return aboveMaxLiteral[int64]{value: math.MaxInt64}
}

func Float32AboveMaxLiteral() Literal {
	return aboveMaxLiteral[float32]{value: math.MaxFloat32}
}

func Float64AboveMaxLiteral() Literal {
	return aboveMaxLiteral[float64]{value: math.MaxFloat64}
}

func Int32BelowMinLiteral() Literal {
	return belowMinLiteral[int32]{value: math.MinInt32}
}

func Int64BelowMinLiteral() Literal {
	return belowMinLiteral[int64]{value: math.MinInt64}
}

func Float32BelowMinLiteral() Literal {
	return belowMinLiteral[float32]{value: -math.MaxFloat32}
}

func Float64BelowMinLiteral() Literal {
	return belowMinLiteral[float64]{value: -math.MaxFloat64}
}

type BoolLiteral bool

func (BoolLiteral) Comparator() Comparator[bool] {
	return func(v1, v2 bool) int {
		if v1 == v2 {
			return 0
		}
		if v1 {
			return 1
		}

		return -1
	}
}

func (b BoolLiteral) Any() any       { return b.Value() }
func (b BoolLiteral) Type() Type     { return PrimitiveTypes.Bool }
func (b BoolLiteral) Value() bool    { return bool(b) }
func (b BoolLiteral) String() string { return strconv.FormatBool(bool(b)) }
func (b BoolLiteral) To(t Type) (Literal, error) {
	switch t.(type) {
	case BooleanType:
		return b, nil
	}

	return nil, fmt.Errorf("%w: BoolLiteral to %s", ErrBadCast, t)
}

func (b BoolLiteral) Equals(l Literal) bool {
	return literalEq(b, l)
}

func (b BoolLiteral) MarshalBinary() (data []byte, err error) {
	// stored as 0x00 for false, and anything non-zero for True
	if b {
		return []byte{0x1}, nil
	}

	return []byte{0x0}, nil
}

func (b *BoolLiteral) UnmarshalBinary(data []byte) error {
	// stored as 0x00 for false and anything non-zero for True
	if len(data) < 1 {
		return fmt.Errorf("%w: expected at least 1 byte for bool", ErrInvalidBinSerialization)
	}
	*b = data[0] != 0

	return nil
}

type Int32Literal int32

func (Int32Literal) Comparator() Comparator[int32] { return cmp.Compare[int32] }
func (i Int32Literal) Type() Type                  { return PrimitiveTypes.Int32 }
func (i Int32Literal) Value() int32                { return int32(i) }
func (i Int32Literal) Any() any                    { return i.Value() }
func (i Int32Literal) String() string              { return strconv.FormatInt(int64(i), 10) }
func (i Int32Literal) To(t Type) (Literal, error) {
	switch t := t.(type) {
	case Int32Type:
		return i, nil
	case Int64Type:
		return Int64Literal(i), nil
	case Float32Type:
		return Float32Literal(i), nil
	case Float64Type:
		return Float64Literal(i), nil
	case DateType:
		return DateLiteral(i), nil
	case TimeType:
		return TimeLiteral(i), nil
	case TimestampType:
		return TimestampLiteral(i), nil
	case TimestampTzType:
		return TimestampLiteral(i), nil
	case DecimalType:
		unscaled := Decimal{Val: decimal128.FromI64(int64(i)), Scale: 0}
		if t.scale == 0 {
			return DecimalLiteral(unscaled), nil
		}
		out, err := unscaled.Val.Rescale(0, int32(t.scale))
		if err != nil {
			return nil, fmt.Errorf("%w: failed to cast to DecimalType: %s", ErrBadCast, err.Error())
		}

		return DecimalLiteral{Val: out, Scale: t.scale}, nil
	}

	return nil, fmt.Errorf("%w: Int32Literal to %s", ErrBadCast, t)
}

func (i Int32Literal) Equals(other Literal) bool {
	return literalEq(i, other)
}

func (i Int32Literal) Increment() Literal {
	if i == math.MaxInt32 {
		return Int32AboveMaxLiteral()
	}

	return Int32Literal(i + 1)
}

func (i Int32Literal) Decrement() Literal {
	if i == math.MinInt32 {
		return Int32BelowMinLiteral()
	}

	return Int32Literal(i - 1)
}

func (i Int32Literal) MarshalBinary() (data []byte, err error) {
	// stored as 4 bytes in little endian order
	data = make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(i))

	return data, err
}

func (i *Int32Literal) UnmarshalBinary(data []byte) error {
	// stored as 4 bytes little endian
	if len(data) != 4 {
		return fmt.Errorf("%w: expected 4 bytes for int32 value, got %d",
			ErrInvalidBinSerialization, len(data))
	}

	*i = Int32Literal(binary.LittleEndian.Uint32(data))

	return nil
}

type Int64Literal int64

func (Int64Literal) Comparator() Comparator[int64] { return cmp.Compare[int64] }
func (i Int64Literal) Type() Type                  { return PrimitiveTypes.Int64 }
func (i Int64Literal) Value() int64                { return int64(i) }
func (i Int64Literal) Any() any                    { return i.Value() }
func (i Int64Literal) String() string              { return strconv.FormatInt(int64(i), 10) }
func (i Int64Literal) To(t Type) (Literal, error) {
	switch t := t.(type) {
	case Int32Type:
		if math.MaxInt32 < i {
			return Int32AboveMaxLiteral(), nil
		} else if math.MinInt32 > i {
			return Int32BelowMinLiteral(), nil
		}

		return Int32Literal(i), nil
	case Int64Type:
		return i, nil
	case Float32Type:
		return Float32Literal(i), nil
	case Float64Type:
		return Float64Literal(i), nil
	case DateType:
		if math.MaxInt32 < i {
			return Int32AboveMaxLiteral(), nil
		} else if math.MinInt32 > i {
			return Int32BelowMinLiteral(), nil
		}

		return DateLiteral(i), nil
	case TimeType:
		return TimeLiteral(i), nil
	case TimestampType:
		return TimestampLiteral(i), nil
	case TimestampTzType:
		return TimestampLiteral(i), nil
	case TimestampNsType:
		return TimestampNsLiteral(i), nil
	case TimestampTzNsType:
		return TimestampNsLiteral(i), nil
	case DecimalType:
		unscaled := Decimal{Val: decimal128.FromI64(int64(i)), Scale: 0}
		if t.scale == 0 {
			return DecimalLiteral(unscaled), nil
		}
		out, err := unscaled.Val.Rescale(0, int32(t.scale))
		if err != nil {
			return nil, fmt.Errorf("%w: failed to cast to DecimalType: %s", ErrBadCast, err.Error())
		}

		return DecimalLiteral{Val: out, Scale: t.scale}, nil
	}

	return nil, fmt.Errorf("%w: Int64Literal to %s", ErrBadCast, t)
}

func (i Int64Literal) Equals(other Literal) bool {
	return literalEq(i, other)
}

func (i Int64Literal) Increment() Literal {
	if i == math.MaxInt64 {
		return Int64AboveMaxLiteral()
	}

	return Int64Literal(i + 1)
}

func (i Int64Literal) Decrement() Literal {
	if i == math.MinInt64 {
		return Int64BelowMinLiteral()
	}

	return Int64Literal(i - 1)
}

func (i Int64Literal) MarshalBinary() (data []byte, err error) {
	// stored as 8 byte little-endian
	data = make([]byte, 8)
	binary.LittleEndian.PutUint64(data, uint64(i))

	return data, err
}

func (i *Int64Literal) UnmarshalBinary(data []byte) error {
	// stored as 8 byte little-endian
	if len(data) != 8 {
		return fmt.Errorf("%w: expected 8 bytes for int64 value, got %d",
			ErrInvalidBinSerialization, len(data))
	}
	*i = Int64Literal(binary.LittleEndian.Uint64(data))

	return nil
}

type Float32Literal float32

func (Float32Literal) Comparator() Comparator[float32] { return cmp.Compare[float32] }
func (f Float32Literal) Type() Type                    { return PrimitiveTypes.Float32 }
func (f Float32Literal) Value() float32                { return float32(f) }
func (f Float32Literal) Any() any                      { return f.Value() }
func (f Float32Literal) String() string                { return strconv.FormatFloat(float64(f), 'g', -1, 32) }
func (f Float32Literal) To(t Type) (Literal, error) {
	switch t := t.(type) {
	case Float32Type:
		return f, nil
	case Float64Type:
		return Float64Literal(f), nil
	case DecimalType:
		v, err := decimal128.FromFloat32(float32(f), int32(t.precision), int32(t.scale))
		if err != nil {
			return nil, err
		}

		return DecimalLiteral{Val: v, Scale: t.scale}, nil
	}

	return nil, fmt.Errorf("%w: Float32Literal to %s", ErrBadCast, t)
}

func (f Float32Literal) Equals(other Literal) bool {
	return literalEq(f, other)
}

func (f Float32Literal) MarshalBinary() (data []byte, err error) {
	// stored as 4 bytes little endian
	data = make([]byte, 4)
	binary.LittleEndian.PutUint32(data, math.Float32bits(float32(f)))

	return data, err
}

func (f *Float32Literal) UnmarshalBinary(data []byte) error {
	// stored as 4 bytes little endian
	if len(data) != 4 {
		return fmt.Errorf("%w: expected 4 bytes for float32 value, got %d",
			ErrInvalidBinSerialization, len(data))
	}
	*f = Float32Literal(math.Float32frombits(binary.LittleEndian.Uint32(data)))

	return nil
}

type Float64Literal float64

func (Float64Literal) Comparator() Comparator[float64] { return cmp.Compare[float64] }
func (f Float64Literal) Type() Type                    { return PrimitiveTypes.Float64 }
func (f Float64Literal) Value() float64                { return float64(f) }
func (f Float64Literal) Any() any                      { return f.Value() }
func (f Float64Literal) String() string                { return strconv.FormatFloat(float64(f), 'g', -1, 64) }
func (f Float64Literal) To(t Type) (Literal, error) {
	switch t := t.(type) {
	case Float32Type:
		if math.MaxFloat32 < f {
			return Float32AboveMaxLiteral(), nil
		} else if -math.MaxFloat32 > f {
			return Float32BelowMinLiteral(), nil
		}

		return Float32Literal(f), nil
	case Float64Type:
		return f, nil
	case DecimalType:
		v, err := decimal128.FromFloat64(float64(f), int32(t.precision), int32(t.scale))
		if err != nil {
			return nil, err
		}

		return DecimalLiteral{Val: v, Scale: t.scale}, nil
	}

	return nil, fmt.Errorf("%w: Float64Literal to %s", ErrBadCast, t)
}

func (f Float64Literal) Equals(other Literal) bool {
	return literalEq(f, other)
}

func (f Float64Literal) MarshalBinary() (data []byte, err error) {
	// stored as 8 bytes little endian
	data = make([]byte, 8)
	binary.LittleEndian.PutUint64(data, math.Float64bits(float64(f)))

	return data, err
}

func (f *Float64Literal) UnmarshalBinary(data []byte) error {
	// stored as 8 bytes in little endian
	if len(data) != 8 {
		return fmt.Errorf("%w: expected 8 bytes for float64 value, got %d",
			ErrInvalidBinSerialization, len(data))
	}
	*f = Float64Literal(math.Float64frombits(binary.LittleEndian.Uint64(data)))

	return nil
}

type DateLiteral Date

func (DateLiteral) Comparator() Comparator[Date] { return cmp.Compare[Date] }
func (d DateLiteral) Type() Type                 { return PrimitiveTypes.Date }
func (d DateLiteral) Value() Date                { return Date(d) }
func (d DateLiteral) Any() any                   { return d.Value() }
func (d DateLiteral) String() string {
	t := Date(d).ToTime()

	return t.Format("2006-01-02")
}

func (d DateLiteral) To(t Type) (Literal, error) {
	switch t.(type) {
	case DateType:
		return d, nil
	}

	return nil, fmt.Errorf("%w: DateLiteral to %s", ErrBadCast, t)
}

func (d DateLiteral) Equals(other Literal) bool {
	return literalEq(d, other)
}

func (d DateLiteral) Increment() Literal { return DateLiteral(d + 1) }
func (d DateLiteral) Decrement() Literal { return DateLiteral(d - 1) }

func (d DateLiteral) MarshalBinary() (data []byte, err error) {
	// stored as 4 byte little endian
	data = make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(d))

	return data, err
}

func (d *DateLiteral) UnmarshalBinary(data []byte) error {
	// stored as 4 byte little endian
	if len(data) != 4 {
		return fmt.Errorf("%w: expected 4 bytes for date value, got %d",
			ErrInvalidBinSerialization, len(data))
	}
	*d = DateLiteral(binary.LittleEndian.Uint32(data))

	return nil
}

type TimeLiteral Time

func (TimeLiteral) Comparator() Comparator[Time] { return cmp.Compare[Time] }
func (t TimeLiteral) Type() Type                 { return PrimitiveTypes.Time }
func (t TimeLiteral) Value() Time                { return Time(t) }
func (t TimeLiteral) Any() any                   { return t.Value() }
func (t TimeLiteral) String() string {
	tm := time.UnixMicro(int64(t)).UTC()

	return tm.Format("15:04:05.000000")
}

func (t TimeLiteral) To(typ Type) (Literal, error) {
	switch typ.(type) {
	case TimeType:
		return t, nil
	}

	return nil, fmt.Errorf("%w: TimeLiteral to %s", ErrBadCast, typ)
}

func (t TimeLiteral) Equals(other Literal) bool {
	return literalEq(t, other)
}

func (t TimeLiteral) MarshalBinary() (data []byte, err error) {
	// stored as 8 byte little-endian
	data = make([]byte, 8)
	binary.LittleEndian.PutUint64(data, uint64(t))

	return data, err
}

func (t *TimeLiteral) UnmarshalBinary(data []byte) error {
	// stored as 8 byte little-endian representing microseconds from midnight
	if len(data) != 8 {
		return fmt.Errorf("%w: expected 8 bytes for time value, got %d",
			ErrInvalidBinSerialization, len(data))
	}
	*t = TimeLiteral(binary.LittleEndian.Uint64(data))

	return nil
}

type TimestampLiteral Timestamp

func (TimestampLiteral) Comparator() Comparator[Timestamp] { return cmp.Compare[Timestamp] }
func (t TimestampLiteral) Type() Type                      { return PrimitiveTypes.Timestamp }
func (t TimestampLiteral) Value() Timestamp                { return Timestamp(t) }
func (t TimestampLiteral) Any() any                        { return t.Value() }
func (t TimestampLiteral) String() string {
	tm := Timestamp(t).ToTime()

	return tm.Format("2006-01-02 15:04:05.000000")
}

func (t TimestampLiteral) To(typ Type) (Literal, error) {
	switch typ.(type) {
	case TimestampType:
		return t, nil
	case TimestampTzType:
		return t, nil
	case TimestampNsType, TimestampTzNsType:
		if t > math.MaxInt64/1000 || t < math.MinInt64/1000 {
			return nil, fmt.Errorf("%w: timestamp %d is outside the nanosecond range", ErrBadCast, t)
		}

		return TimestampNsLiteral(Timestamp(t).ToNanos()), nil
	case DateType:
		return DateLiteral(Timestamp(t).ToDate()), nil
	}

	return nil, fmt.Errorf("%w: TimestampLiteral to %s", ErrBadCast, typ)
}

func (t TimestampLiteral) Equals(other Literal) bool {
	return literalEq(t, other)
}

func (t TimestampLiteral) Increment() Literal { return TimestampLiteral(t + 1) }
func (t TimestampLiteral) Decrement() Literal { return TimestampLiteral(t - 1) }

func (t TimestampLiteral) MarshalBinary() (data []byte, err error) {
	// stored as 8 byte little endian
	data = make([]byte, 8)
	binary.LittleEndian.PutUint64(data, uint64(t))

	return data, err
}

func (t *TimestampLiteral) UnmarshalBinary(data []byte) error {
	// stored as 8 byte little endian value representing microseconds since epoch
	if len(data) != 8 {
		return fmt.Errorf("%w: expected 8 bytes for timestamp value, got %d",
			ErrInvalidBinSerialization, len(data))
	}
	*t = TimestampLiteral(binary.LittleEndian.Uint64(data))

	return nil
}

type TimestampNsLiteral TimestampNano

func (TimestampNsLiteral) Comparator() Comparator[TimestampNano] { return cmp.Compare[TimestampNano] }
func (t TimestampNsLiteral) Type() Type                          { return PrimitiveTypes.TimestampNs }
func (t TimestampNsLiteral) Value() TimestampNano                { return TimestampNano(t) }
func (t TimestampNsLiteral) Any() any                            { return t.Value() }
func (t TimestampNsLiteral) String() string {
	tm := TimestampNano(t).ToTime()

	return tm.Format("2006-01-02 15:04:05.000000000")
}

func (t TimestampNsLiteral) To(typ Type) (Literal, error) {
	switch typ.(type) {
	case TimestampType:
		return TimestampLiteral(TimestampNano(t).ToMicros()), nil
	case TimestampTzType:
		return TimestampLiteral(TimestampNano(t).ToMicros()), nil
	case TimestampNsType:
		return t, nil
	case TimestampTzNsType:
		return t, nil
	case DateType:
		return DateLiteral(TimestampNano(t).ToDate()), nil
	}

	return nil, fmt.Errorf("%w: TimestampNsLiteral to %s", ErrBadCast, typ)
}

func (t TimestampNsLiteral) Equals(other Literal) bool {
	return literalEq(t, other)
}

func (t TimestampNsLiteral) Increment() Literal { return TimestampNsLiteral(t + 1) }
func (t TimestampNsLiteral) Decrement() Literal { return TimestampNsLiteral(t - 1) }

func (t TimestampNsLiteral) MarshalBinary() (data []byte, err error) {
	// stored as 8 byte little endian
	data = make([]byte, 8)
	binary.LittleEndian.PutUint64(data, uint64(t))

	return data, err
}

func (t *TimestampNsLiteral) UnmarshalBinary(data []byte) error {
	// stored as 8 byte little endian value representing nanoseconds since epoch
	if len(data) != 8 {
		return fmt.Errorf("%w: expected 8 bytes for timestamp value, got %d",
			ErrInvalidBinSerialization, len(data))
	}
	*t = TimestampNsLiteral(binary.LittleEndian.Uint64(data))

	return nil
}

type StringLiteral string

func (StringLiteral) Comparator() Comparator[string] { return cmp.Compare[string] }
func (s StringLiteral) Type() Type                   { return PrimitiveTypes.String }
func (s StringLiteral) Value() string                { return string(s) }
func (s StringLiteral) Any() any                     { return s.Value() }
func (s StringLiteral) String() string               { return string(s) }
func (s StringLiteral) To(typ Type) (Literal, error) {
	switch t := typ.(type) {
	case StringType:
		return s, nil
	case Int32Type:
		n, err := strconv.ParseInt(string(s), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: casting '%s' to %s",
				errors.Join(ErrBadCast, err), s, typ)
		}

		if math.MaxInt32 < n {
			return Int32AboveMaxLiteral(), nil
		} else if math.MinInt32 > n {
			return Int32BelowMinLiteral(), nil
		}

		return Int32Literal(n), nil
	case Int64Type:
		n, err := strconv.ParseInt(string(s), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: casting '%s' to %s",
				errors.Join(ErrBadCast, err), s, typ)
		}

		return Int64Literal(n), nil
	case Float32Type:
		n, err := strconv.ParseFloat(string(s), 32)
		if err != nil {
			return nil, fmt.Errorf("%w: casting '%s' to %s",
				errors.Join(ErrBadCast, err), s, typ)
		}

		return Float32Literal(n), nil
	case Float64Type:
		n, err := strconv.ParseFloat(string(s), 64)
		if err != nil {
			return nil, fmt.Errorf("%w: casting '%s' to %s",
				errors.Join(ErrBadCast, err), s, typ)
		}

		return Float64Literal(n), nil
	case DateType:
		tm, err := time.Parse("2006-01-02", string(s))
		if err != nil {
			return nil, fmt.Errorf("%w: casting '%s' to %s - %s",
				ErrBadCast, s, typ, err.Error())
		}

		return DateLiteral(tm.Truncate(24*time.Hour).Unix() / int64((time.Hour * 24).Seconds())), nil
	case TimeType:
		val, err := arrow.Time64FromString(string(s), arrow.Microsecond)
		if err != nil {
			return nil, fmt.Errorf("%w: casting '%s' to %s - %s",
				ErrBadCast, s, typ, err.Error())
		}

		return TimeLiteral(val), nil
	case TimestampType:
		// ISO date-time, no zone; fractional seconds optional.
		tm, err := time.Parse("2006-01-02T15:04:05.999999999", string(s))
		if err != nil {
			return nil, fmt.Errorf("%w: invalid Timestamp format for casting from string '%s': %s",
				ErrBadCast, s, err.Error())
		}

		return TimestampLiteral(Timestamp(tm.UTC().UnixMicro())), nil
	case TimestampTzType:
		// ISO date-time WITH zone; up to microsecond fractional seconds.
		tm, err := time.Parse("2006-01-02T15:04:05.999999Z07:00", string(s))
		if err != nil {
			return nil, fmt.Errorf("%w: invalid TimestampTz format for casting from string '%s': %s",
				ErrBadCast, s, err.Error())
		}

		return TimestampLiteral(Timestamp(tm.UTC().UnixMicro())), nil
	case TimestampNsType:
		// Same as timestamp but nanosecond precision.
		tm, err := time.Parse("2006-01-02T15:04:05.999999999", string(s))
		if err != nil {
			return nil, fmt.Errorf("%w: invalid TimestampNs format for casting from string '%s': %s",
				ErrBadCast, s, err.Error())
		}

		return TimestampNsLiteral(TimestampNano(tm.UTC().UnixNano())), nil
	case TimestampTzNsType:
		// Same as timestamptz but nanosecond precision.
		tm, err := time.Parse("2006-01-02T15:04:05.999999999Z07:00", string(s))
		if err != nil {
			return nil, fmt.Errorf("%w: invalid TimestampTzNs format for casting from string '%s': %s",
				ErrBadCast, s, err.Error())
		}

		return TimestampNsLiteral(TimestampNano(tm.UTC().UnixNano())), nil
	case UUIDType:
		val, err := uuid.Parse(string(s))
		if err != nil {
			return nil, fmt.Errorf("%w: casting '%s' to %s - %s",
				ErrBadCast, s, typ, err.Error())
		}

		return UUIDLiteral(val), nil
	case DecimalType:
		n, err := decimal128.FromString(string(s), int32(t.precision), int32(t.scale))
		if err != nil {
			return nil, fmt.Errorf("%w: casting '%s' to %s - %s",
				ErrBadCast, s, typ, err.Error())
		}

		return DecimalLiteral{Val: n, Scale: t.scale}, nil
	case BooleanType:
		val, err := strconv.ParseBool(string(s))
		if err != nil {
			return nil, fmt.Errorf("%w: casting '%s' to %s - %s",
				ErrBadCast, s, typ, err.Error())
		}

		return BoolLiteral(val), nil
	case BinaryType:
		return BinaryLiteral(s), nil
	case FixedType:
		if len(s) != t.len {
			return nil, fmt.Errorf("%w: cast '%s' to %s - wrong length",
				ErrBadCast, s, t)
		}

		return FixedLiteral(s), nil
	}

	return nil, fmt.Errorf("%w: StringLiteral to %s", ErrBadCast, typ)
}

func (s StringLiteral) Equals(other Literal) bool {
	return literalEq(s, other)
}

func (s StringLiteral) MarshalBinary() (data []byte, err error) {
	// stored as UTF-8 bytes without length
	// avoid copying by just returning a slice of the raw bytes
	data = unsafe.Slice(unsafe.StringData(string(s)), len(s))

	return data, err
}

func (s *StringLiteral) UnmarshalBinary(data []byte) error {
	// stored as UTF-8 bytes without length
	// avoid copy, but this means that the passed in slice is being given
	// to the literal for ownership
	*s = StringLiteral(unsafe.String(unsafe.SliceData(data), len(data)))

	return nil
}

type BinaryLiteral []byte

func (BinaryLiteral) Comparator() Comparator[[]byte] {
	return bytes.Compare
}
func (b BinaryLiteral) Type() Type     { return PrimitiveTypes.Binary }
func (b BinaryLiteral) Value() []byte  { return []byte(b) }
func (b BinaryLiteral) Any() any       { return b.Value() }
func (b BinaryLiteral) String() string { return string(b) }
func (b BinaryLiteral) To(typ Type) (Literal, error) {
	switch t := typ.(type) {
	case UUIDType:
		val, err := uuid.FromBytes(b)
		if err != nil {
			return nil, fmt.Errorf("%w: cannot convert BinaryLiteral to UUID",
				errors.Join(ErrBadCast, err))
		}

		return UUIDLiteral(val), nil
	case FixedType:
		if len(b) == t.len {
			return FixedLiteral(b), nil
		}

		return nil, fmt.Errorf("%w: cannot convert BinaryLiteral to %s, different length - %d <> %d",
			ErrBadCast, typ, len(b), t.len)
	case BinaryType:
		return b, nil
	}

	return nil, fmt.Errorf("%w: BinaryLiteral to %s", ErrBadCast, typ)
}

func (b BinaryLiteral) Equals(other Literal) bool {
	rhs, ok := other.(BinaryLiteral)
	if !ok {
		return false
	}

	return bytes.Equal([]byte(b), rhs)
}

func (b BinaryLiteral) MarshalBinary() (data []byte, err error) {
	// stored directly as is
	data = b

	return data, err
}

func (b *BinaryLiteral) UnmarshalBinary(data []byte) error {
	// stored directly as is
	*b = BinaryLiteral(data)

	return nil
}

// GeoLiteral is a non-null geometry or geography single-value bound. Iceberg
// serializes geospatial bounds as the concatenation of little-endian float64
// coordinates in X, Y[, Z][, M] order (spec Appendix D), not as WKB. Those
// coordinates have no total order, so - unlike every other literal - GeoLiteral
// is deliberately not a TypedLiteral: it exposes no Comparator and refuses to be
// cast to an orderable type. That keeps ordering-based pruning from silently
// running bytes.Compare over coordinate bytes (which would yield wrong answers)
// and makes it error instead. Type() reports the concrete GeometryType or
// GeographyType, so callers that key off the literal's own type - pruning, To,
// logging - see the truth rather than plain binary.
type GeoLiteral struct {
	val []byte
	typ Type
}

func (g GeoLiteral) Type() Type    { return g.typ }
func (g GeoLiteral) Value() []byte { return g.val }
func (g GeoLiteral) Any() any      { return g.val }
func (g GeoLiteral) String() string {
	return fmt.Sprintf("%s(%x)", g.typ, g.val)
}

func (g GeoLiteral) To(typ Type) (Literal, error) {
	// Only an identity cast is meaningful. Casting to any orderable type is
	// rejected so a geo bound can never be smuggled into a comparison.
	if g.typ.Equals(typ) {
		return g, nil
	}

	return nil, fmt.Errorf("%w: GeoLiteral to %s", ErrBadCast, typ)
}

func (g GeoLiteral) Equals(other Literal) bool {
	rhs, ok := other.(GeoLiteral)
	if !ok {
		return false
	}

	return g.typ.Equals(rhs.typ) && bytes.Equal(g.val, rhs.val)
}

func (g GeoLiteral) MarshalBinary() ([]byte, error) {
	return g.val, nil
}

type FixedLiteral []byte

func (FixedLiteral) Comparator() Comparator[[]byte] { return bytes.Compare }
func (f FixedLiteral) Type() Type                   { return FixedType{len: len(f)} }
func (f FixedLiteral) Value() []byte                { return []byte(f) }
func (f FixedLiteral) Any() any                     { return f.Value() }
func (f FixedLiteral) String() string               { return string(f) }
func (f FixedLiteral) To(typ Type) (Literal, error) {
	switch t := typ.(type) {
	case UUIDType:
		val, err := uuid.FromBytes(f)
		if err != nil {
			return nil, fmt.Errorf("%w: cannot convert FixedLiteral to UUID - %s",
				ErrBadCast, err.Error())
		}

		return UUIDLiteral(val), nil
	case FixedType:
		if len(f) == t.len {
			return FixedLiteral(f), nil
		}

		return nil, fmt.Errorf("%w: cannot convert FixedLiteral to %s, different length - %d <> %d",
			ErrBadCast, typ, len(f), t.len)
	case BinaryType:
		return BinaryLiteral(slices.Clone(f)), nil
	}

	return nil, fmt.Errorf("%w: FixedLiteral[%d] to %s",
		ErrBadCast, len(f), typ)
}

func (f FixedLiteral) Equals(other Literal) bool {
	rhs, ok := other.(FixedLiteral)
	if !ok {
		return false
	}

	return bytes.Equal([]byte(f), rhs)
}

func (f FixedLiteral) MarshalBinary() (data []byte, err error) {
	// stored directly as is
	data = f

	return data, err
}

func (f *FixedLiteral) UnmarshalBinary(data []byte) error {
	// stored directly as is
	*f = FixedLiteral(data)

	return nil
}

type UUIDLiteral uuid.UUID

func (UUIDLiteral) Comparator() Comparator[uuid.UUID] {
	return func(v1, v2 uuid.UUID) int {
		return bytes.Compare(v1[:], v2[:])
	}
}

func (UUIDLiteral) Type() Type         { return PrimitiveTypes.UUID }
func (u UUIDLiteral) Value() uuid.UUID { return uuid.UUID(u) }
func (u UUIDLiteral) Any() any         { return u.Value() }
func (u UUIDLiteral) String() string   { return uuid.UUID(u).String() }
func (u UUIDLiteral) To(typ Type) (Literal, error) {
	switch t := typ.(type) {
	case UUIDType:
		return u, nil
	case FixedType:
		if len(u) == t.len {
			v, _ := uuid.UUID(u).MarshalBinary()

			return FixedLiteral(v), nil
		}

		return nil, fmt.Errorf("%w: cannot convert UUIDLiteral to %s, different length - %d <> %d",
			ErrBadCast, typ, len(u), t.len)
	case BinaryType:
		v, _ := uuid.UUID(u).MarshalBinary()

		return BinaryLiteral(v), nil
	}

	return nil, fmt.Errorf("%w: UUIDLiteral to %s", ErrBadCast, typ)
}

func (u UUIDLiteral) Equals(other Literal) bool {
	rhs, ok := other.(UUIDLiteral)
	if !ok {
		return false
	}

	return uuid.UUID(u) == uuid.UUID(rhs)
}

func (u UUIDLiteral) MarshalBinary() (data []byte, err error) {
	return uuid.UUID(u).MarshalBinary()
}

func (u *UUIDLiteral) UnmarshalBinary(data []byte) error {
	// stored as 16-byte big-endian value
	out, err := uuid.FromBytes(data)
	if err != nil {
		return err
	}
	*u = UUIDLiteral(out)

	return nil
}

type DecimalLiteral Decimal

func (DecimalLiteral) Comparator() Comparator[Decimal] {
	return func(v1, v2 Decimal) int {
		if v1.Scale == v2.Scale {
			return v1.Val.Cmp(v2.Val)
		}

		rescaled, err := v2.Val.Rescale(int32(v2.Scale), int32(v1.Scale))
		if err != nil {
			return -1
		}

		return v1.Val.Cmp(rescaled)
	}
}

// Type returns a DecimalType built from the literal's scale and a hardcoded
// precision of 9. The precision is NOT the originating column's declared
// precision; DecimalLiteral does not carry precision. Callers that need the
// real column precision must consult the bound field's type rather than
// lit.Type(). See https://github.com/apache/iceberg-go/issues/1028.
func (d DecimalLiteral) Type() Type {
	return DecimalType{precision: 9, scale: d.Scale}
}
func (d DecimalLiteral) Value() Decimal { return Decimal(d) }
func (d DecimalLiteral) Any() any       { return d.Value() }
func (d DecimalLiteral) String() string {
	return d.Val.ToString(int32(d.Scale))
}

func (d DecimalLiteral) To(t Type) (Literal, error) {
	switch t := t.(type) {
	case DecimalType:
		if d.Scale == t.scale {
			return d, nil
		}

		return nil, fmt.Errorf("%w: could not convert %v to %s",
			ErrBadCast, d, t)
	case Int32Type:
		v := d.Val.BigInt()
		if !v.IsInt64() {
			if v.Sign() > 0 {
				return Int32AboveMaxLiteral(), nil
			} else if v.Sign() < 0 {
				return Int32BelowMinLiteral(), nil
			}
		}
		i := v.Int64()
		if i > math.MaxInt32 {
			return Int32AboveMaxLiteral(), nil
		} else if i < math.MinInt32 {
			return Int32BelowMinLiteral(), nil
		}

		return Int32Literal(int32(i)), nil
	case Int64Type:
		v := d.Val.BigInt()
		if !v.IsInt64() {
			if v.Sign() > 0 {
				return Int64AboveMaxLiteral(), nil
			} else if v.Sign() < 0 {
				return Int64BelowMinLiteral(), nil
			}
		}

		return Int64Literal(v.Int64()), nil
	case Float32Type:
		v := d.Val.ToFloat64(int32(d.Scale))
		if v > math.MaxFloat32 {
			return Float32AboveMaxLiteral(), nil
		} else if v < -math.MaxFloat32 {
			return Float32BelowMinLiteral(), nil
		}

		return Float32Literal(float32(v)), nil
	case Float64Type:
		return Float64Literal(d.Val.ToFloat64(int32(d.Scale))), nil
	}

	return nil, fmt.Errorf("%w: DecimalLiteral to %s", ErrBadCast, t)
}

func (d DecimalLiteral) Equals(other Literal) bool {
	rhs, ok := other.(DecimalLiteral)
	if !ok {
		return false
	}

	rescaled, err := rhs.Val.Rescale(int32(rhs.Scale), int32(d.Scale))
	if err != nil {
		return false
	}

	return d.Val == rescaled
}

func (d DecimalLiteral) Increment() Literal {
	d.Val = d.Val.Add(decimal128.FromU64(1))

	return d
}

func (d DecimalLiteral) Decrement() Literal {
	d.Val = d.Val.Sub(decimal128.FromU64(1))

	return d
}

func (d DecimalLiteral) MarshalBinary() (data []byte, err error) {
	// stored as unscaled value in two's compliment big-endian values
	// using the minimum number of bytes for the values
	n := decimal128.Num(d.Val).BigInt()

	if n.Sign() >= 0 {
		// For non-negative values, use standard encoding
		minBytes := (n.BitLen() + 8) / 8
		if minBytes == 0 {
			minBytes = 1
		}
		data = n.FillBytes(make([]byte, minBytes))

		return data, err
	}

	// For negative values, we need to compute minBytes differently
	// to handle byte boundaries correctly
	bitLen := n.BitLen()
	minBytes := (bitLen + 7) / 8

	// When BitLen is exactly at a byte boundary (multiple of 8),
	// we need to check if we need an extra byte for the sign bit
	if bitLen%8 == 0 {
		absVal := new(big.Int).Abs(n)
		// If |n| is exactly 2^(bitLen-1), it fits perfectly in bitLen/8 bytes
		// Examples: -128 (2^7), -32768 (2^15), -8388608 (2^23)
		powerOf2 := new(big.Int).Lsh(big.NewInt(1), uint(bitLen-1))
		if absVal.Cmp(powerOf2) == 0 {
			minBytes = bitLen / 8
		} else {
			// BitLen is at boundary but value isn't exactly 2^(bitLen-1)
			// Examples: -255, -32767
			// We need an extra byte to ensure the sign bit is set
			minBytes++
		}
	}

	if minBytes == 0 {
		minBytes = 1
	}

	// Convert to two's complement using proper carry propagation
	data = n.FillBytes(make([]byte, minBytes))
	// Invert all bits
	for i := range data {
		data[i] = ^data[i]
	}
	// Add 1 with proper carry propagation
	carry := byte(1)
	for i := len(data) - 1; i >= 0 && carry > 0; i-- {
		sum := uint16(data[i]) + uint16(carry)
		data[i] = byte(sum)
		carry = byte(sum >> 8)
	}

	return data, err
}

func (d *DecimalLiteral) UnmarshalBinary(data []byte) error {
	// stored as unscaled value in two's complement
	// big-endian values using the minimum number of bytes
	if len(data) == 0 {
		d.Val = decimal128.Num{}

		return nil
	}

	if int8(data[0]) >= 0 {
		// not negative
		d.Val = decimal128.FromBigInt((&big.Int{}).SetBytes(data))

		return nil
	}

	// convert two's complement and remember it's negative
	out := make([]byte, len(data))
	for i, b := range data {
		out[i] = ^b
	}

	// Add 1 with proper carry propagation
	carry := byte(1)
	for i := len(out) - 1; i >= 0 && carry > 0; i-- {
		sum := uint16(out[i]) + uint16(carry)
		out[i] = byte(sum)
		carry = byte(sum >> 8)
	}

	value := (&big.Int{}).SetBytes(out)
	d.Val = decimal128.FromBigInt(value.Neg(value))

	return nil
}

type VariantLiteral variant.Value

func (VariantLiteral) Comparator() Comparator[variant.Value] {
	return func(v1, v2 variant.Value) int {
		panic("variant values are not comparable")
	}
}

func (VariantLiteral) Type() Type             { return VariantType{} }
func (v VariantLiteral) Value() variant.Value { return variant.Value(v) }
func (v VariantLiteral) Any() any             { return variant.Value(v) }
func (v VariantLiteral) String() string       { return variant.Value(v).String() }

func (v VariantLiteral) MarshalBinary() ([]byte, error) {
	return variant.Value(v).Bytes(), nil
}

func (v VariantLiteral) To(typ Type) (Literal, error) {
	if _, ok := typ.(VariantType); ok {
		return v, nil
	}

	// TODO: improve by getting the actual value (using .Type()) and attempting
	// to convert, or returning an error if it can't.
	return nil, fmt.Errorf("%w: VariantLiteral to %s", ErrBadCast, typ)
}

func (v VariantLiteral) Equals(other Literal) bool {
	rhs, ok := other.(VariantLiteral)
	if !ok {
		return false
	}

	// TODO: use an upstream Equal method once added to parquet/variant
	lhs := variant.Value(v)
	r := variant.Value(rhs)

	return bytes.Equal(lhs.Metadata().Bytes(), r.Metadata().Bytes()) && bytes.Equal(lhs.Bytes(), r.Bytes())
}
