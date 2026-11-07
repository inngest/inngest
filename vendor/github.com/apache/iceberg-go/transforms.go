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
	"encoding"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
	"unsafe"

	"github.com/apache/arrow-go/v18/arrow/decimal128"
	"github.com/apache/iceberg-go/internal"
	"github.com/google/uuid"
	"github.com/twmb/murmur3"
)

// ParseTransform takes the string representation of a transform as
// defined in the iceberg spec, and produces the appropriate Transform
// object. Strings that don't name a known transform yield an
// UnknownTransform rather than an error, so that v3 tables using transforms
// this implementation doesn't recognize can still be read. This mirrors Java's
// Transforms.fromString: only a bucket/truncate whose width parses as a number
// but is out of range (e.g. bucket[0]) is an error; anything else that doesn't
// match the width pattern, like bucket[-1], is an unknown transform.
func ParseTransform(s string) (Transform, error) {
	if s == "" {
		return nil, fmt.Errorf("%w: empty transform", ErrInvalidTransform)
	}

	if transform, ok := parseSimpleTransform(s); ok {
		return transform, nil
	}

	if n, matched := parseTransformWidth(s, "bucket["); matched {
		if n == 0 {
			return nil, fmt.Errorf("%w: %s", ErrInvalidTransform, s)
		}

		return BucketTransform{NumBuckets: n}, nil
	}

	if n, matched := parseTransformWidth(s, "truncate["); matched {
		if n == 0 {
			return nil, fmt.Errorf("%w: %s", ErrInvalidTransform, s)
		}

		return TruncateTransform{Width: n}, nil
	}

	if hasASCIICaseDifference(s) {
		if transform, ok := parseSimpleTransformFold(s); ok {
			return transform, nil
		}

		if n, matched := parseTransformWidthFold(s, "bucket["); matched {
			if n == 0 {
				return nil, fmt.Errorf("%w: %s", ErrInvalidTransform, s)
			}

			return BucketTransform{NumBuckets: n}, nil
		}

		if n, matched := parseTransformWidthFold(s, "truncate["); matched {
			if n == 0 {
				return nil, fmt.Errorf("%w: %s", ErrInvalidTransform, s)
			}

			return TruncateTransform{Width: n}, nil
		}
	}

	// Unknown transform: v3 readers must load these and ignore them when
	// filtering instead of failing. Keep the original string, case and all, so it
	// round-trips; Equals stays byte-for-byte, matching Java's UnknownTransform.
	return UnknownTransform{name: s}, nil
}

func hasASCIICaseDifference(s string) bool {
	for i := range s {
		if s[i] >= 'A' && s[i] <= 'Z' {
			return true
		}
	}

	return false
}

func parseSimpleTransformFold(s string) (Transform, bool) {
	switch {
	case strings.EqualFold(s, "identity"):
		return IdentityTransform{}, true
	case strings.EqualFold(s, "void"):
		return VoidTransform{}, true
	case strings.EqualFold(s, "year"):
		return YearTransform{}, true
	case strings.EqualFold(s, "month"):
		return MonthTransform{}, true
	case strings.EqualFold(s, "day"):
		return DayTransform{}, true
	case strings.EqualFold(s, "hour"):
		return HourTransform{}, true
	default:
		return nil, false
	}
}

func parseSimpleTransform(s string) (Transform, bool) {
	switch s {
	case "identity":
		return IdentityTransform{}, true
	case "void":
		return VoidTransform{}, true
	case "year":
		return YearTransform{}, true
	case "month":
		return MonthTransform{}, true
	case "day":
		return DayTransform{}, true
	case "hour":
		return HourTransform{}, true
	default:
		return nil, false
	}
}

func parseTransformWidth(s, prefix string) (int, bool) {
	if !strings.HasPrefix(s, prefix) || len(s) <= len(prefix)+1 || s[len(s)-1] != ']' {
		return 0, false
	}

	width := s[len(prefix) : len(s)-1]
	for i := range width {
		if width[i] < '0' || width[i] > '9' {
			return 0, false
		}
	}

	n, err := strconv.Atoi(width)
	if err != nil || n <= 0 || n > math.MaxInt32 {
		return 0, true
	}

	return n, true
}

func parseTransformWidthFold(s, prefix string) (int, bool) {
	if !hasPrefixFold(s, prefix) || len(s) <= len(prefix)+1 || s[len(s)-1] != ']' {
		return 0, false
	}

	width := s[len(prefix) : len(s)-1]
	for i := range width {
		if width[i] < '0' || width[i] > '9' {
			return 0, false
		}
	}

	n, err := strconv.Atoi(width)
	if err != nil || n <= 0 || n > math.MaxInt32 {
		return 0, true
	}

	return n, true
}

func hasPrefixFold(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}

	for i := range prefix {
		got := s[i]
		if got >= 'A' && got <= 'Z' {
			got += 'a' - 'A'
		}
		if got != prefix[i] {
			return false
		}
	}

	return true
}

// Transform is an interface for the various Transformation types
// in partition specs. Currently, they do not yet provide actual
// transformation functions or implementation. That will come later as
// data reading gets implemented.
type Transform interface {
	fmt.Stringer
	encoding.TextMarshaler
	CanTransform(t Type) bool
	ResultType(t Type) Type
	PreservesOrder() bool
	Equals(Transform) bool
	Apply(Optional[Literal]) Optional[Literal]
	Project(name string, pred BoundPredicate) (UnboundPredicate, error)

	ToHumanStrType(typ Type, val any) string
	// Deprecated: ToHumanStr cannot recover source-type information; use ToHumanStrType instead.
	ToHumanStr(any) string
}

// IdentityTransform uses the identity function, performing no transformation
// but instead partitioning on the value itself.
type IdentityTransform struct{}

func (t IdentityTransform) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (IdentityTransform) String() string { return "identity" }

func (IdentityTransform) CanTransform(t Type) bool {
	switch t.(type) {
	case GeometryType, GeographyType, VariantType:
		return false
	}
	_, ok := t.(PrimitiveType)

	return ok
}
func (IdentityTransform) ResultType(t Type) Type { return t }
func (IdentityTransform) PreservesOrder() bool   { return true }

func (IdentityTransform) Equals(other Transform) bool {
	switch rhs := other.(type) {
	case IdentityTransform:
		return true
	case *IdentityTransform:
		return rhs != nil
	default:
		return false
	}
}

func (IdentityTransform) Apply(value Optional[Literal]) Optional[Literal] {
	return value
}

func (IdentityTransform) ToHumanStr(val any) string {
	switch v := val.(type) {
	case nil:
		return "null"
	case string:
		return v
	case []byte:
		return base64.StdEncoding.EncodeToString(v)
	case bool:
		return strconv.FormatBool(v)
	case Date:
		return v.ToTime().Format("2006-01-02")
	case Time:
		return v.ToTime().Format("15:04:05.999999")
	case Timestamp:
		return v.ToTime().Format("2006-01-02T15:04:05.999999")
	case TimestampNano:
		return v.ToTime().Format("2006-01-02T15:04:05.999999999")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ToHumanStrType is the type-aware form invoked by PartitionToPath.
// It appends "+00:00" for TimestampTz/TimestampTzNs — information the value-only ToHumanStr(any) cannot recover by design
func (t IdentityTransform) ToHumanStrType(typ Type, val any) string {
	if val == nil {
		return "null"
	}
	switch typ.(type) {
	case TimestampType:
		if v, ok := val.(Timestamp); ok {
			return v.ToTime().Format("2006-01-02T15:04:05.999999")
		}
	case TimestampTzType:
		if v, ok := val.(Timestamp); ok {
			return v.ToTime().Format("2006-01-02T15:04:05.999999") + "+00:00"
		}
	case TimestampNsType:
		if v, ok := val.(TimestampNano); ok {
			return v.ToTime().Format("2006-01-02T15:04:05.999999999")
		}
	case TimestampTzNsType:
		if v, ok := val.(TimestampNano); ok {
			return v.ToTime().Format("2006-01-02T15:04:05.999999999") + "+00:00"
		}
	}

	return t.ToHumanStr(val)
}

func (t IdentityTransform) Project(name string, pred BoundPredicate) (UnboundPredicate, error) {
	if _, ok := pred.Term().(*BoundTransform); ok {
		return projectTransformPredicate(t, name, pred)
	}

	switch p := pred.(type) {
	case BoundUnaryPredicate:
		return p.AsUnbound(Reference(name)), nil
	case BoundLiteralPredicate:
		return p.AsUnbound(Reference(name), p.Literal()), nil
	case BoundSetPredicate:
		return p.AsUnbound(Reference(name), boundSetLiteralsForVisit(p).Members()), nil
	}

	return nil, nil
}

// VoidTransform is a transformation that always returns nil.
type VoidTransform struct{}

func (t VoidTransform) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (VoidTransform) String() string { return "void" }

func (VoidTransform) CanTransform(Type) bool { return true }
func (VoidTransform) ResultType(t Type) Type { return t }
func (VoidTransform) PreservesOrder() bool   { return false }

func (VoidTransform) Equals(other Transform) bool {
	switch rhs := other.(type) {
	case VoidTransform:
		return true
	case *VoidTransform:
		return rhs != nil
	default:
		return false
	}
}

func (VoidTransform) Apply(value Optional[Literal]) Optional[Literal] {
	return Optional[Literal]{}
}

func (VoidTransform) ToHumanStr(any) string { return "null" }

func (VoidTransform) ToHumanStrType(Type, any) string { return "null" }

func (VoidTransform) Project(string, BoundPredicate) (UnboundPredicate, error) {
	return nil, nil
}

// UnknownTransform is a placeholder for a partition or sort transform that
// this implementation doesn't recognize. The v3 spec requires readers to load
// tables that use unknown transforms and to ignore those fields when
// filtering; writers must not commit a partition spec that uses one.
type UnknownTransform struct {
	name string
}

// MarshalText rejects the zero value. UnknownTransform{} is a legal composite
// literal outside this package, and an empty name would serialize to
// "transform": "" -- metadata that can't be read back.
func (t UnknownTransform) MarshalText() ([]byte, error) {
	if t.name == "" {
		return nil, fmt.Errorf("%w: unknown transform has no name", ErrInvalidTransform)
	}

	return []byte(t.name), nil
}

func (t UnknownTransform) String() string { return t.name }

// CanTransform always returns true: compatibility with a source type is
// unverifiable for an unknown transform, not verified. Callers such as
// SortOrder.CheckCompatibility therefore accept any source type here.
func (UnknownTransform) CanTransform(Type) bool { return true }

// ResultType is unknown, so report string, matching the Java reference.
func (UnknownTransform) ResultType(Type) Type { return StringType{} }

func (UnknownTransform) PreservesOrder() bool { return false }

func (t UnknownTransform) Equals(other Transform) bool {
	switch rhs := other.(type) {
	case UnknownTransform:
		return t.name == rhs.name
	case *UnknownTransform:
		return rhs != nil && t.name == rhs.name
	default:
		return false
	}
}

// Apply can't be evaluated for an unknown transform.
func (UnknownTransform) Apply(Optional[Literal]) Optional[Literal] {
	return Optional[Literal]{}
}

// ToHumanStr renders the value, not the transform name. Java's
// Transform#toHumanString default does the same and UnknownTransform doesn't
// override it. Returning the name would be constant for a spec field, which
// collapses distinct partitions into one PartitionToPath key.
func (UnknownTransform) ToHumanStr(val any) string {
	return IdentityTransform{}.ToHumanStr(val)
}

func (UnknownTransform) ToHumanStrType(typ Type, val any) string {
	return IdentityTransform{}.ToHumanStrType(typ, val)
}

// Project returns nil so scans don't prune on an unknown partition field.
func (UnknownTransform) Project(string, BoundPredicate) (UnboundPredicate, error) {
	return nil, nil
}

// BucketTransform transforms values into a bucket partition value. It is
// parameterized by a number of buckets. Bucket partition transforms use
// a 32-bit hash of the source value to produce a positive value by mod
// the bucket number.
type BucketTransform struct {
	NumBuckets int
}

func (t BucketTransform) MarshalText() ([]byte, error) {
	if err := t.validateNumBuckets(); err != nil {
		return nil, err
	}

	return []byte(t.String()), nil
}

func (t BucketTransform) String() string { return fmt.Sprintf("bucket[%d]", t.NumBuckets) }

func (t BucketTransform) validateNumBuckets() error {
	if t.NumBuckets <= 0 {
		return fmt.Errorf("%w: bucket transform requires numBuckets > 0", ErrInvalidArgument)
	}
	if t.NumBuckets > math.MaxInt32 {
		return fmt.Errorf("%w: bucket transform requires numBuckets <= %d", ErrInvalidArgument, math.MaxInt32)
	}

	return nil
}

func (BucketTransform) CanTransform(t Type) bool {
	switch t.(type) {
	case Int32Type,
		DateType,
		Int64Type,
		TimeType,
		TimestampNsType,
		TimestampType,
		TimestampTzNsType,
		TimestampTzType,
		DecimalType,
		StringType,
		FixedType,
		BinaryType,
		UUIDType:
		return true
	default:
		return false
	}
}
func (BucketTransform) ResultType(Type) Type { return PrimitiveTypes.Int32 }
func (BucketTransform) PreservesOrder() bool { return false }

func hashHelperInt[T ~int32 | ~int64](v any) uint32 {
	var (
		val = uint64(v.(T))
		buf [8]byte
		b   = buf[:]
	)

	binary.LittleEndian.PutUint64(b, val)

	return murmur3.Sum32(b)
}

func hashTimestampNano(v any) uint32 {
	micros := internal.FloorDiv(int64(v.(TimestampNano)), int64(time.Microsecond))

	return hashHelperInt[int64](micros)
}

func hashDecimal(v Decimal) uint32 {
	// Decimal bucket transforms hash the minimum two's-complement
	// big-endian representation of the unscaled value. Keep the full
	// decimal128 value on the stack and trim only redundant sign bytes.
	var buf [16]byte
	binary.BigEndian.PutUint64(buf[:8], uint64(v.Val.HighBits()))
	binary.BigEndian.PutUint64(buf[8:], v.Val.LowBits())

	start := 0
	if v.Val.Sign() >= 0 {
		for start < len(buf)-1 && buf[start] == 0 && buf[start+1]&0x80 == 0 {
			start++
		}
	} else {
		for start < len(buf)-1 && buf[start] == 0xff && buf[start+1]&0x80 != 0 {
			start++
		}
	}

	return murmur3.Sum32(buf[start:])
}

func (t BucketTransform) Equals(other Transform) bool {
	switch rhs := other.(type) {
	case BucketTransform:
		return t.NumBuckets == rhs.NumBuckets
	case *BucketTransform:
		return rhs != nil && t.NumBuckets == rhs.NumBuckets
	default:
		return false
	}
}

func (t BucketTransform) Apply(value Optional[Literal]) Optional[Literal] {
	if !value.Valid {
		return Optional[Literal]{}
	}
	if err := t.validateNumBuckets(); err != nil {
		return Optional[Literal]{}
	}

	var hash uint32
	switch v := value.Val.(type) {
	case TypedLiteral[[]byte]:
		hash = murmur3.Sum32(v.Value())
	case StringLiteral:
		hash = murmur3.Sum32(unsafe.Slice(unsafe.StringData(string(v)), len(v)))
	case UUIDLiteral:
		hash = murmur3.Sum32(v[:])
	case DecimalLiteral:
		hash = hashDecimal(Decimal(v))
	case Int32Literal:
		hash = hashHelperInt[int64](int64(v))
	case Int64Literal:
		hash = hashHelperInt[int64](int64(v))
	case DateLiteral:
		hash = hashHelperInt[int64](int64(v))
	case TimeLiteral:
		hash = hashHelperInt[int64](int64(v))
	case TimestampLiteral:
		hash = hashHelperInt[int64](int64(v))
	case TimestampNsLiteral:
		hash = hashTimestampNano(TimestampNano(v))
	default:
		return Optional[Literal]{}
	}

	return Optional[Literal]{
		Valid: true,
		Val:   Int32Literal((int32(hash) & math.MaxInt32) % int32(t.NumBuckets)),
	}
}

func (t BucketTransform) Transformer(src Type) func(any) Optional[int32] {
	if err := t.validateNumBuckets(); err != nil {
		return func(_ any) Optional[int32] {
			return Optional[int32]{}
		}
	}

	var h func(any) uint32

	switch src.(type) {
	case Int32Type:
		h = hashHelperInt[int32]
	case DateType:
		h = hashHelperInt[Date]
	case Int64Type:
		h = hashHelperInt[int64]
	case TimeType:
		h = hashHelperInt[Time]
	case TimestampType:
		h = hashHelperInt[Timestamp]
	case TimestampTzType:
		h = hashHelperInt[Timestamp]
	case TimestampNsType, TimestampTzNsType:
		h = hashTimestampNano
	case DecimalType:
		h = func(v any) uint32 {
			return hashDecimal(v.(Decimal))
		}
	case StringType, FixedType, BinaryType:
		h = func(v any) uint32 {
			if v, ok := v.([]byte); ok {
				return murmur3.Sum32(v)
			}

			str := v.(string)

			return murmur3.Sum32(unsafe.Slice(unsafe.StringData(str), len(str)))
		}
	case UUIDType:
		h = func(v any) uint32 {
			if v, ok := v.([]byte); ok {
				return murmur3.Sum32(v)
			}

			u := v.(uuid.UUID)

			return murmur3.Sum32(u[:])
		}
	}

	return func(v any) Optional[int32] {
		if v == nil || h == nil {
			return Optional[int32]{}
		}

		return Optional[int32]{
			Valid: true,
			Val:   int32((int32(h(v)) & math.MaxInt32) % int32(t.NumBuckets)),
		}
	}
}

func (BucketTransform) ToHumanStr(val any) string {
	if val == nil {
		return "null"
	}

	return fmt.Sprintf("%v", val)
}

func (t BucketTransform) ToHumanStrType(_ Type, val any) string {
	return t.ToHumanStr(val)
}

func (t BucketTransform) Project(name string, pred BoundPredicate) (UnboundPredicate, error) {
	if err := t.validateNumBuckets(); err != nil {
		return nil, err
	}
	if _, ok := pred.Term().(*BoundTransform); ok {
		return projectTransformPredicate(t, name, pred)
	}

	if !t.CanTransform(pred.Term().Type()) {
		return nil, nil
	}

	transformer := t.Transformer(pred.Term().Type())
	switch p := pred.(type) {
	case BoundUnaryPredicate:
		return p.AsUnbound(Reference(name)), nil
	case BoundLiteralPredicate:
		if p.Op() != OpEQ {
			break
		}

		return p.AsUnbound(Reference(name), transformLiteral(transformer, p.Literal())), nil
	case BoundSetPredicate:
		if p.Op() != OpIn {
			break
		}

		return setApplyTransform(name, p, transformer), nil
	}

	return nil, nil
}

// TruncateTransform is a transformation for truncating a value to a specified width.
type TruncateTransform struct {
	Width int
}

func (t TruncateTransform) MarshalText() ([]byte, error) {
	if err := t.validateWidth(); err != nil {
		return nil, err
	}

	return []byte(t.String()), nil
}

func (t TruncateTransform) String() string { return fmt.Sprintf("truncate[%d]", t.Width) }

func (t TruncateTransform) validateWidth() error {
	if t.Width <= 0 {
		return fmt.Errorf("%w: truncate transform requires width > 0", ErrInvalidArgument)
	}
	if t.Width > math.MaxInt32 {
		return fmt.Errorf("%w: truncate transform requires width <= %d", ErrInvalidArgument, math.MaxInt32)
	}

	return nil
}

func (TruncateTransform) CanTransform(t Type) bool {
	switch t.(type) {
	case Int32Type,
		Int64Type,
		StringType,
		BinaryType,
		DecimalType:
		return true
	default:
		return false
	}
}
func (TruncateTransform) ResultType(t Type) Type { return t }
func (TruncateTransform) PreservesOrder() bool   { return true }
func (t TruncateTransform) Equals(other Transform) bool {
	switch rhs := other.(type) {
	case TruncateTransform:
		return t.Width == rhs.Width
	case *TruncateTransform:
		return rhs != nil && t.Width == rhs.Width
	default:
		return false
	}
}

func (t TruncateTransform) Transformer(src Type) (func(any) any, error) {
	if err := t.validateWidth(); err != nil {
		return nil, err
	}

	switch src.(type) {
	case Int32Type:
		return func(v any) any {
			if v == nil {
				return nil
			}

			val := v.(int32)
			width := int32(t.Width)

			return val - (((val % width) + width) % width)
		}, nil
	case Int64Type:
		return func(v any) any {
			if v == nil {
				return nil
			}

			val := v.(int64)
			width := int64(t.Width)

			return val - (((val % width) + width) % width)
		}, nil
	case StringType:
		return func(v any) any {
			str, ok := v.(string)
			if !ok {
				return nil
			}

			return truncateString(str, t.Width)
		}, nil
	case BinaryType:
		return func(v any) any {
			b, ok := v.([]byte)
			if !ok {
				return nil
			}

			return b[:min(len(b), t.Width)]
		}, nil
	case DecimalType:
		bigWidth := big.NewInt(int64(t.Width))

		return func(v any) any {
			if v == nil {
				return nil
			}

			val := v.(Decimal)
			unscaled := val.Val.BigInt()
			// unscaled - (((unscaled % width) + width) % width)
			applied := (&big.Int{}).Mod(unscaled, bigWidth)
			applied.Add(applied, bigWidth).Mod(applied, bigWidth)
			val.Val = decimal128.FromBigInt(unscaled.Sub(unscaled, applied))

			return val
		}, nil
	}

	return nil, fmt.Errorf("%w: cannot truncate for type %s",
		ErrInvalidArgument, src)
}

// Range yields byte indexes at UTF-8 code point boundaries, so slicing on idx
// truncates by Unicode characters without copying the whole string to []rune.
func truncateString(s string, width int) string {
	for idx := range s {
		if width == 0 {
			return s[:idx]
		}

		width--
	}

	return s
}

func (t TruncateTransform) Apply(value Optional[Literal]) (out Optional[Literal]) {
	if !value.Valid {
		return out
	}

	fn, err := t.Transformer(value.Val.Type())
	if err != nil {
		return out
	}

	out.Valid = true
	switch v := value.Val.(type) {
	case Int32Literal:
		out.Val = Int32Literal(fn(int32(v)).(int32))
	case Int64Literal:
		out.Val = Int64Literal(fn(int64(v)).(int64))
	case DecimalLiteral:
		out.Val = DecimalLiteral(fn(Decimal(v)).(Decimal))
	case StringLiteral:
		out.Val = StringLiteral(fn(string(v)).(string))
	case BinaryLiteral:
		out.Val = BinaryLiteral(fn([]byte(v)).([]byte))
	}

	return out
}

func (TruncateTransform) ToHumanStr(val any) string {
	switch v := val.(type) {
	case nil:
		return "null"
	case []byte:
		return base64.StdEncoding.EncodeToString(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (t TruncateTransform) ToHumanStrType(_ Type, val any) string {
	return t.ToHumanStr(val)
}

func (t TruncateTransform) Project(name string, pred BoundPredicate) (UnboundPredicate, error) {
	if err := t.validateWidth(); err != nil {
		return nil, err
	}

	if _, ok := pred.Term().(*BoundTransform); ok {
		return projectTransformPredicate(t, name, pred)
	}

	fieldType := pred.Term().Ref().Field().Type

	transformer, err := t.Transformer(fieldType)
	if err != nil {
		return nil, err
	}

	switch p := pred.(type) {
	case BoundUnaryPredicate:
		return p.AsUnbound(Reference(name)), nil
	case BoundSetPredicate:
		if p.Op() != OpIn {
			break
		}

		switch fieldType.(type) {
		case Int32Type:
			return setApplyTransform(name, p, wrapTransformFn[int32](transformer)), nil
		case Int64Type:
			return setApplyTransform(name, p, wrapTransformFn[int64](transformer)), nil
		case DecimalType:
			return setApplyTransform(name, p, wrapTransformFn[Decimal](transformer)), nil
		case StringType:
			return setApplyTransform(name, p, wrapTransformFn[string](transformer)), nil
		case BinaryType:
			return setApplyTransform(name, p, wrapTransformFn[[]byte](transformer)), nil
		}
	case BoundLiteralPredicate:
		// notStartsWith is only projectable when the prefix fits within the
		// truncation width. If the prefix is longer, the truncated partition
		// value may itself satisfy the predicate, so projecting the truncated
		// prefix would incorrectly prune matching files.
		if p.Op() == OpNotStartsWith {
			l := literalLen(p.Literal())
			switch {
			case l < 0:
				return nil, nil
			case l < t.Width:
				return LiteralPredicate(OpNotStartsWith, Reference(name), p.Literal()), nil
			case l == t.Width:
				return LiteralPredicate(OpNEQ, Reference(name), p.Literal()), nil
			}

			return nil, nil
		}

		switch fieldType.(type) {
		case Int32Type:
			return truncateNumber(name, p, wrapTransformFn[int32](transformer))
		case Int64Type:
			return truncateNumber(name, p, wrapTransformFn[int64](transformer))
		case DecimalType:
			return truncateNumber(name, p, wrapTransformFn[Decimal](transformer))
		case StringType:
			return truncateArray(name, p, wrapTransformFn[string](transformer))
		case BinaryType:
			return truncateArray(name, p, wrapTransformFn[[]byte](transformer))
		}
	}

	return nil, nil
}

var epochTM = time.Unix(0, 0).UTC()

type TimeTransform interface {
	Transform
	Transformer(Type) (func(any) Optional[int32], error)
}

func canTransformTime(t TimeTransform, sourceType Type) bool {
	switch sourceType.(type) {
	case DateType, TimestampType, TimestampTzType, TimestampNsType, TimestampTzNsType:
		return true
	default:
		return false
	}
}

func projectTimeTransform(t TimeTransform, name string, pred BoundPredicate) (UnboundPredicate, error) {
	if _, ok := pred.Term().(*BoundTransform); ok {
		return projectTransformPredicate(t, name, pred)
	}

	transformer, err := t.Transformer(pred.Term().Ref().Type())
	if err != nil {
		return nil, err
	}

	switch p := pred.(type) {
	case BoundUnaryPredicate:
		return p.AsUnbound(Reference(name)), nil
	case BoundLiteralPredicate:
		return truncateNumber(name, p, transformer)
	case BoundSetPredicate:
		if p.Op() != OpIn {
			break
		}

		return setApplyTransform(name, p, transformer), nil
	}

	return nil, nil
}

// YearTransform transforms a datetime value into a year value.
type YearTransform struct{}

func (t YearTransform) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (YearTransform) String() string { return "year" }

func (t YearTransform) CanTransform(sourceType Type) bool { return canTransformTime(t, sourceType) }
func (YearTransform) ResultType(Type) Type                { return PrimitiveTypes.Int32 }
func (YearTransform) PreservesOrder() bool                { return true }

func (YearTransform) Equals(other Transform) bool {
	switch rhs := other.(type) {
	case YearTransform:
		return true
	case *YearTransform:
		return rhs != nil
	default:
		return false
	}
}

func (YearTransform) Transformer(src Type) (func(any) Optional[int32], error) {
	switch src.(type) {
	case DateType:
		return func(v any) Optional[int32] {
			if v == nil {
				return Optional[int32]{}
			}

			return Optional[int32]{
				Valid: true,
				Val:   int32(v.(Date).ToTime().Year() - epochTM.Year()),
			}
		}, nil
	case TimestampType, TimestampTzType:
		return func(v any) Optional[int32] {
			if v == nil {
				return Optional[int32]{}
			}

			return Optional[int32]{
				Valid: true,
				Val:   int32(v.(Timestamp).ToTime().Year() - epochTM.Year()),
			}
		}, nil
	case TimestampNsType, TimestampTzNsType:
		return func(v any) Optional[int32] {
			if v == nil {
				return Optional[int32]{}
			}

			return Optional[int32]{
				Valid: true,
				Val:   int32(v.(TimestampNano).ToTime().Year() - epochTM.Year()),
			}
		}, nil
	}

	return nil, fmt.Errorf("%w: cannot apply year transform for type %s",
		ErrInvalidArgument, src)
}

func (YearTransform) Apply(value Optional[Literal]) (out Optional[Literal]) {
	if !value.Valid {
		return out
	}

	switch v := value.Val.(type) {
	case DateLiteral:
		out.Valid = true
		out.Val = Int32Literal(Date(v).ToTime().Year() - epochTM.Year())
	case TimestampLiteral:
		out.Valid = true
		out.Val = Int32Literal(Timestamp(v).ToTime().Year() - epochTM.Year())
	case TimestampNsLiteral:
		out.Valid = true
		out.Val = Int32Literal(TimestampNano(v).ToTime().Year() - epochTM.Year())
	}

	return out
}

func (YearTransform) ToHumanStr(val any) string {
	switch v := val.(type) {
	case int32:
		return strconv.Itoa(int(v) + epochTM.Year())
	default:
		return "null"
	}
}

func (t YearTransform) ToHumanStrType(_ Type, val any) string {
	return t.ToHumanStr(val)
}

func (t YearTransform) Project(name string, pred BoundPredicate) (UnboundPredicate, error) {
	return projectTimeTransform(t, name, pred)
}

// MonthTransform transforms a datetime value into a month value.
type MonthTransform struct{}

func (t MonthTransform) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (MonthTransform) String() string { return "month" }

func (t MonthTransform) CanTransform(sourceType Type) bool { return canTransformTime(t, sourceType) }
func (MonthTransform) ResultType(Type) Type                { return PrimitiveTypes.Int32 }
func (MonthTransform) PreservesOrder() bool                { return true }

func (MonthTransform) Equals(other Transform) bool {
	switch rhs := other.(type) {
	case MonthTransform:
		return true
	case *MonthTransform:
		return rhs != nil
	default:
		return false
	}
}

func (MonthTransform) Transformer(src Type) (func(any) Optional[int32], error) {
	switch src.(type) {
	case DateType:
		return func(v any) Optional[int32] {
			if v == nil {
				return Optional[int32]{}
			}

			d := v.(Date).ToTime()

			return Optional[int32]{
				Valid: true,
				Val:   int32((d.Year()-epochTM.Year())*12 + (int(d.Month()) - int(epochTM.Month()))),
			}
		}, nil
	case TimestampType, TimestampTzType:
		return func(v any) Optional[int32] {
			if v == nil {
				return Optional[int32]{}
			}

			d := v.(Timestamp).ToTime()

			return Optional[int32]{
				Valid: true,
				Val:   int32((d.Year()-epochTM.Year())*12 + (int(d.Month()) - int(epochTM.Month()))),
			}
		}, nil
	case TimestampNsType, TimestampTzNsType:
		return func(v any) Optional[int32] {
			if v == nil {
				return Optional[int32]{}
			}

			d := v.(TimestampNano).ToTime()

			return Optional[int32]{
				Valid: true,
				Val:   int32((d.Year()-epochTM.Year())*12 + (int(d.Month()) - int(epochTM.Month()))),
			}
		}, nil

	}

	return nil, fmt.Errorf("%w: cannot apply month transform for type %s",
		ErrInvalidArgument, src)
}

func (MonthTransform) Apply(value Optional[Literal]) (out Optional[Literal]) {
	if !value.Valid {
		return out
	}

	var tm time.Time
	switch v := value.Val.(type) {
	case DateLiteral:
		tm = Date(v).ToTime()
	case TimestampLiteral:
		tm = Timestamp(v).ToTime()
	case TimestampNsLiteral:
		tm = TimestampNano(v).ToTime()
	default:
		return out
	}

	out.Valid = true
	out.Val = Int32Literal(int32((tm.Year()-epochTM.Year())*12 + (int(tm.Month()) - int(epochTM.Month()))))

	return out
}

func (t MonthTransform) ToHumanStr(val any) string {
	switch v := val.(type) {
	case int32:
		tm := epochTM.AddDate(0, int(v), 0)

		return tm.Format("2006-01")
	default:
		return "null"
	}
}

func (t MonthTransform) ToHumanStrType(_ Type, val any) string {
	return t.ToHumanStr(val)
}

func (t MonthTransform) Project(name string, pred BoundPredicate) (UnboundPredicate, error) {
	return projectTimeTransform(t, name, pred)
}

// DayTransform transforms a datetime value into a date value.
type DayTransform struct{}

func (t DayTransform) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (DayTransform) String() string { return "day" }

func (t DayTransform) CanTransform(sourceType Type) bool { return canTransformTime(t, sourceType) }
func (DayTransform) ResultType(Type) Type                { return PrimitiveTypes.Date }
func (DayTransform) PreservesOrder() bool                { return true }

func (DayTransform) Equals(other Transform) bool {
	switch rhs := other.(type) {
	case DayTransform:
		return true
	case *DayTransform:
		return rhs != nil
	default:
		return false
	}
}

func (DayTransform) Transformer(src Type) (func(any) Optional[int32], error) {
	switch src.(type) {
	case DateType:
		return func(v any) Optional[int32] {
			if v == nil {
				return Optional[int32]{}
			}

			return Optional[int32]{
				Valid: true,
				Val:   int32(v.(Date)),
			}
		}, nil
	case TimestampType, TimestampTzType:
		return func(v any) Optional[int32] {
			if v == nil {
				return Optional[int32]{}
			}

			return Optional[int32]{
				Valid: true,
				Val:   int32(v.(Timestamp).ToDate()),
			}
		}, nil
	case TimestampNsType, TimestampTzNsType:
		return func(v any) Optional[int32] {
			if v == nil {
				return Optional[int32]{}
			}

			return Optional[int32]{
				Valid: true,
				Val:   int32(v.(TimestampNano).ToDate()),
			}
		}, nil
	}

	return nil, fmt.Errorf("%w: cannot apply day transform for type %s",
		ErrInvalidArgument, src)
}

func (DayTransform) Apply(value Optional[Literal]) (out Optional[Literal]) {
	if !value.Valid {
		return out
	}

	switch v := value.Val.(type) {
	case DateLiteral:
		out.Valid, out.Val = true, v
	case TimestampLiteral:
		out.Valid, out.Val = true, DateLiteral(Timestamp(v).ToDate())
	case TimestampNsLiteral:
		out.Valid, out.Val = true, DateLiteral(TimestampNano(v).ToDate())
	}

	return out
}

func (DayTransform) ToHumanStr(val any) string {
	var days int32
	switch v := val.(type) {
	case Date:
		days = int32(v)
	case int32:
		days = v
	default:
		return "null"
	}

	tm := epochTM.AddDate(0, 0, int(days))

	return tm.Format("2006-01-02")
}

func (t DayTransform) ToHumanStrType(_ Type, val any) string {
	return t.ToHumanStr(val)
}

func (t DayTransform) Project(name string, pred BoundPredicate) (UnboundPredicate, error) {
	return projectTimeTransform(t, name, pred)
}

// HourTransform transforms a datetime value into an hour value.
type HourTransform struct{}

func (t HourTransform) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (HourTransform) String() string { return "hour" }

func (t HourTransform) CanTransform(sourceType Type) bool {
	switch sourceType.(type) {
	case TimestampType, TimestampTzType, TimestampTzNsType, TimestampNsType:
		return true
	default:
		return false
	}
}
func (HourTransform) ResultType(Type) Type { return PrimitiveTypes.Int32 }
func (HourTransform) PreservesOrder() bool { return true }

func (HourTransform) Equals(other Transform) bool {
	switch rhs := other.(type) {
	case HourTransform:
		return true
	case *HourTransform:
		return rhs != nil
	default:
		return false
	}
}

func (HourTransform) Transformer(src Type) (func(any) Optional[int32], error) {
	switch src.(type) {
	case TimestampType, TimestampTzType:
		const factor = int64(time.Hour / time.Microsecond)

		return func(v any) Optional[int32] {
			if v == nil {
				return Optional[int32]{}
			}

			return Optional[int32]{
				Valid: true,
				Val:   int32(internal.FloorDiv(int64(v.(Timestamp)), factor)),
			}
		}, nil
	case TimestampNsType, TimestampTzNsType:
		const factor = int64(time.Hour)

		return func(v any) Optional[int32] {
			if v == nil {
				return Optional[int32]{}
			}

			return Optional[int32]{
				Valid: true,
				Val:   int32(internal.FloorDiv(int64(v.(TimestampNano)), factor)),
			}
		}, nil
	}

	return nil, fmt.Errorf("%w: cannot apply hour transform for type %s",
		ErrInvalidArgument, src)
}

func (HourTransform) Apply(value Optional[Literal]) (out Optional[Literal]) {
	if !value.Valid {
		return out
	}

	switch v := value.Val.(type) {
	case TimestampLiteral:
		const factor = int64(time.Hour / time.Microsecond)
		out.Valid, out.Val = true, Int32Literal(int32(internal.FloorDiv(int64(v), factor)))
	case TimestampNsLiteral:
		const factor = int64(time.Hour)
		out.Valid, out.Val = true, Int32Literal(int32(internal.FloorDiv(int64(v), factor)))
	}

	return out
}

func (HourTransform) ToHumanStr(val any) string {
	switch v := val.(type) {
	case int32:
		seconds := int64(v) * int64(time.Hour/time.Second)
		tm := time.Unix(seconds, 0).UTC()

		return tm.Format("2006-01-02-15")
	default:
		return "null"
	}
}

func (t HourTransform) ToHumanStrType(_ Type, val any) string {
	return t.ToHumanStr(val)
}

func (t HourTransform) Project(name string, pred BoundPredicate) (UnboundPredicate, error) {
	return projectTimeTransform(t, name, pred)
}

func removeTransform(partName string, pred BoundPredicate) (UnboundPredicate, error) {
	ref := Reference(partName)
	switch p := pred.(type) {
	case BoundUnaryPredicate:
		return UnaryPredicate(p.Op(), ref), nil
	case BoundLiteralPredicate:
		return LiteralPredicate(p.Op(), ref, p.Literal()), nil
	case BoundSetPredicate:
		return SetPredicate(p.Op(), ref, boundSetLiteralsForVisit(p).Members()).(UnboundPredicate), nil
	}

	return nil, fmt.Errorf("%w: cannot replace transform in unknown predicate: %s",
		ErrInvalidArgument, pred)
}

func projectTransformPredicate(t Transform, partitionName string, pred BoundPredicate) (UnboundPredicate, error) {
	term := pred.Term()
	bt, ok := term.(*BoundTransform)
	if !ok || !t.Equals(bt.transform) {
		return nil, nil
	}

	return removeTransform(partitionName, pred)
}

func transformLiteral[T LiteralType](fn func(any) Optional[T], lit Literal) Literal {
	switch l := lit.(type) {
	case BoolLiteral:
		return NewLiteral(fn(bool(l)).Val)
	case Int32Literal:
		return NewLiteral(fn(int32(l)).Val)
	case Int64Literal:
		return NewLiteral(fn(int64(l)).Val)
	case Float32Literal:
		return NewLiteral(fn(float32(l)).Val)
	case Float64Literal:
		return NewLiteral(fn(float64(l)).Val)
	case DateLiteral:
		return NewLiteral(fn(Date(l)).Val)
	case TimeLiteral:
		return NewLiteral(fn(Time(l)).Val)
	case TimestampLiteral:
		return NewLiteral(fn(Timestamp(l)).Val)
	case TimestampNsLiteral:
		return NewLiteral(fn(TimestampNano(l)).Val)
	case StringLiteral:
		return NewLiteral(fn(string(l)).Val)
	case FixedLiteral:
		return NewLiteral(fn([]byte(l)).Val)
	case BinaryLiteral:
		return NewLiteral(fn([]byte(l)).Val)
	case UUIDLiteral:
		return NewLiteral(fn(uuid.UUID(l)).Val)
	case DecimalLiteral:
		return NewLiteral(fn(Decimal(l)).Val)
	}

	panic("invalid literal type")
}

func wrapTransformFn[T LiteralType](fn func(any) any) func(any) Optional[T] {
	return func(v any) Optional[T] {
		out := fn(v)
		if out == nil {
			return Optional[T]{}
		}

		return Optional[T]{Valid: true, Val: out.(T)}
	}
}

func truncateNumber[T LiteralType](name string, pred BoundLiteralPredicate, fn func(any) Optional[T]) (UnboundPredicate, error) {
	boundary, ok := pred.Literal().(NumericLiteral)
	if !ok {
		return nil, fmt.Errorf("%w: expected numeric literal, got %s",
			ErrInvalidArgument, pred.Literal().Type())
	}

	switch pred.Op() {
	case OpLT:
		adjusted := boundary.Decrement()
		if _, ok := adjusted.(BelowMinLiteral); ok {
			return nil, nil
		}

		return LiteralPredicate(OpLTEQ, Reference(name),
			transformLiteral(fn, adjusted)), nil
	case OpLTEQ:
		return LiteralPredicate(OpLTEQ, Reference(name),
			transformLiteral(fn, boundary)), nil
	case OpGT:
		adjusted := boundary.Increment()
		if _, ok := adjusted.(AboveMaxLiteral); ok {
			return nil, nil
		}

		return LiteralPredicate(OpGTEQ, Reference(name),
			transformLiteral(fn, adjusted)), nil
	case OpGTEQ:
		return LiteralPredicate(OpGTEQ, Reference(name),
			transformLiteral(fn, boundary)), nil
	case OpEQ:
		return LiteralPredicate(OpEQ, Reference(name),
			transformLiteral(fn, boundary)), nil
	}

	return nil, nil
}

func truncateArray[T LiteralType](name string, pred BoundLiteralPredicate, fn func(any) Optional[T]) (UnboundPredicate, error) {
	boundary := pred.Literal()

	switch pred.Op() {
	case OpLT, OpLTEQ:
		return LiteralPredicate(OpLTEQ, Reference(name),
			transformLiteral(fn, boundary)), nil
	case OpGT, OpGTEQ:
		return LiteralPredicate(OpGTEQ, Reference(name),
			transformLiteral(fn, boundary)), nil
	case OpEQ:
		return LiteralPredicate(OpEQ, Reference(name),
			transformLiteral(fn, boundary)), nil
	case OpStartsWith:
		return LiteralPredicate(OpStartsWith, Reference(name),
			transformLiteral(fn, boundary)), nil
	}

	return nil, nil
}

func literalLen(lit Literal) int {
	switch l := lit.(type) {
	case StringLiteral:
		return utf8.RuneCountInString(string(l))
	case BinaryLiteral:
		return len(l)
	case FixedLiteral:
		return len(l)
	}

	return -1
}

func setApplyTransform[T LiteralType](name string, pred BoundSetPredicate, fn func(any) Optional[T]) UnboundPredicate {
	lits := boundSetLiteralsForVisit(pred).Members()
	for i, l := range lits {
		lits[i] = transformLiteral(fn, l)
	}

	return pred.AsUnbound(Reference(name), lits)
}
