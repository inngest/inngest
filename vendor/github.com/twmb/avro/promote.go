package avro

import (
	"math"
	"reflect"
	"time"
)

// promotionDeser returns a deserfn that reads the writer's wire type and
// sets the reader's Go type, or nil if the promotion is not supported.
func promotionDeser(writerKind, readerKind string) deserfn {
	key := writerKind + ">" + readerKind
	return promotions[key]
}

var promotions = map[string]deserfn{
	"int>long":   promoteIntToLong,
	"int>float":  promoteIntToFloat,
	"int>double": promoteIntToDouble,

	"long>float":  promoteLongToFloat,
	"long>double": promoteLongToDouble,

	"float>double": promoteFloatToDouble,

	"string>bytes": promoteStringToBytes,
	"bytes>string": promoteBytesToString,
}

// promoteRead wraps a wire read + per-target setter into a deserfn.
func promoteRead[Wire any](
	read func([]byte) (Wire, []byte, error),
	apply func(reflect.Value, Wire) error,
) deserfn {
	return func(src []byte, v reflect.Value, _ *slab) ([]byte, error) {
		val, src, err := read(src)
		if err != nil {
			return nil, err
		}
		return src, apply(indirectAlloc(v), val)
	}
}

// promoteIntFloatMantissa is the int→float conversion shared by the four
// int/long→float/double promotion arms. A float/double READER schema opts into
// IEEE precision, so wire magnitudes it cannot hold exactly IEEE-round
// silently. Matches Java's ResolvingDecoder.readDouble, fastavro's
// maybe_promote, and hamba's createDoubleConverter.
//
// Asymmetric with the same-schema decode: s.Decode(wire, &f float64) against
// MustParse("long") still rejects. Lossiness is acceptable when the READER
// SCHEMA is lossy; when only the Go type is, the wire held a value the user
// should not silently lose.
//
// The float64 cast keeps the long's full value; setFloatValue's SetFloat
// narrows to float32 for the bitSize=32 arm.
func promoteIntFloatMantissa(v reflect.Value, n int64, avroType string, bitSize int) error {
	var f float64
	if bitSize == 32 {
		f = float64(float32(n))
	} else {
		f = float64(n)
	}
	return setFloatValue(v, f, avroType, bitSize)
}

var (
	// setLongValue handles the Interface arm internally, so no special-case
	// needed here.
	promoteIntToLong = promoteRead(readVarint,
		func(v reflect.Value, n int32) error { return setLongValue(v, int64(n)) })
	promoteIntToFloat = promoteRead(readVarint,
		func(v reflect.Value, n int32) error { return promoteIntFloatMantissa(v, int64(n), "float", 32) })
	promoteIntToDouble = promoteRead(readVarint,
		func(v reflect.Value, n int32) error { return promoteIntFloatMantissa(v, int64(n), "double", 64) })
	promoteLongToFloat = promoteRead(readVarlong,
		func(v reflect.Value, n int64) error { return promoteIntFloatMantissa(v, n, "float", 32) })
	promoteLongToDouble = promoteRead(readVarlong,
		func(v reflect.Value, n int64) error { return promoteIntFloatMantissa(v, n, "double", 64) })
	promoteFloatToDouble = promoteRead(readUint32,
		func(v reflect.Value, u uint32) error {
			return setFloatValue(v, float64(math.Float32frombits(u)), "double", 64)
		})
)

// readBytesPrefix reads a varlong length prefix and validates it against the
// remaining buffer, keeping the three error shapes (varlong, negative, overrun)
// in one place for every length-prefixed promotion.
//
// destAvroType labels the negative-length SemanticError; wireTypeName labels
// the overrun ShortBufferError. The two swap by direction: string→bytes tags
// negative as "bytes" (destination) and short-buffer as "string" (the writer's
// wire type).
func readBytesPrefix(src []byte, destAvroType, wireTypeName string) (n int, rest []byte, err error) {
	length, rest, err := readVarlong(src)
	if err != nil {
		return 0, nil, err
	}
	if length < 0 {
		return 0, nil, &SemanticError{AvroType: destAvroType}
	}
	if length > int64(len(rest)) {
		return 0, nil, &ShortBufferError{Type: wireTypeName, Need: int(length), Have: len(rest)}
	}
	return int(length), rest, nil
}

func promoteStringToBytes(src []byte, v reflect.Value, _ *slab) ([]byte, error) {
	n, src, err := readBytesPrefix(src, "bytes", "string")
	if err != nil {
		return nil, err
	}
	if err := setBytesValue(indirectAlloc(v), src[:n], "bytes"); err != nil {
		return nil, err
	}
	return src[n:], nil
}

// promotionDeserForLogical returns a deserfn that reads the writer's wire type
// AND applies the reader's logical conversion, or nil when the reader has no
// logical reachable through a promotion. The bare promotion desers know nothing
// about logicals, so without this a writer int → reader {"long",
// "logicalType":"timestamp-millis"} yields raw int64 instead of time.Time.
func promotionDeserForLogical(writerKind string, r *schemaNode) deserfn {
	if r.logical == "" {
		return nil
	}
	// Keyed on the PROMOTION, using the same "writer>reader" key the
	// promotions table itself is keyed by — not on the reader kind alone.
	// Each wrapper below reads the WRITER's wire form (a varint for
	// int→long, a varlong length prefix for string↔bytes) before applying
	// the reader's conversion, so it is correct only for the exact pair it
	// is written for. Keying on the pair is what makes that structural: a
	// promotion added to the table with no arm here falls through to the
	// bare widening deser instead of reaching a wrapper that would misread
	// the wire. The pairs absent here have no reachable logical reader —
	// long→float, long→double and float→double all land on float/double,
	// which carry no logical types.
	switch writerKind + ">" + r.kind {
	case "int>long":
		switch r.logical {
		case "timestamp-millis", "local-timestamp-millis":
			return promoteIntToLongTime(timestampMillisToTime)
		case "timestamp-micros", "local-timestamp-micros":
			return promoteIntToLongTime(timestampMicrosToTime)
		case "timestamp-nanos", "local-timestamp-nanos":
			return promoteIntToLongTime(timestampNanosToTime)
		case "time-micros":
			return promoteIntToLongTimeMicros
		}
	case "string>bytes":
		switch r.logical {
		case "decimal":
			return promoteStringToBytesDecimal(r.scale)
		case "big-decimal":
			return promoteStringToBytesBigDecimal
		}
	case "bytes>string":
		switch r.logical {
		case "uuid":
			return promoteBytesToStringUUID
		}
	}
	return nil
}

func promoteIntToLongTime(conv func(int64) time.Time) deserfn {
	return func(src []byte, v reflect.Value, _ *slab) ([]byte, error) {
		val, src, err := readVarint(src)
		if err != nil {
			return nil, err
		}
		return src, setTimeAsLongTarget(indirectAlloc(v), int64(val), conv)
	}
}

// promoteIntToLongTimeMicros mirrors deserTimeMicros but reads the
// writer's varint (int) and widens to int64 before applying the
// duration conversion. time-micros has its own overflow check inside
// timeMicrosToDuration; we preserve it here.
func promoteIntToLongTimeMicros(src []byte, v reflect.Value, _ *slab) ([]byte, error) {
	val, src, err := readVarint(src)
	if err != nil {
		return nil, err
	}
	return src, setTimeMicrosTarget(indirectAlloc(v), int64(val))
}

// promoteStringToBytesDecimal reads the writer's varlong-length-
// prefixed string bytes and applies the reader's decimal conversion
// at the given schema scale. Mirrors deserBytesDecimal but with the
// length-read shape of promoteStringToBytes.
func promoteStringToBytesDecimal(scale int) deserfn {
	return func(src []byte, v reflect.Value, _ *slab) ([]byte, error) {
		n, src, err := readBytesPrefix(src, "bytes", "string")
		if err != nil {
			return nil, err
		}
		b := src[:n]
		v = indirectAlloc(v)
		ok, err := setDecimalValue(v, b, scale)
		if err != nil {
			return nil, err
		}
		if ok {
			return src[n:], nil
		}
		// Fall through to plain bytes target.
		if err := setBytesValue(v, b, "bytes"); err != nil {
			return nil, err
		}
		return src[n:], nil
	}
}

// promoteStringToBytesBigDecimal reads the writer's varlong-length-
// prefixed bytes and dispatches to the same arms as deserBigDecimal
// (parse as structured big-decimal payload, fall back to raw bytes
// for opaque-pass-through targets).
func promoteStringToBytesBigDecimal(src []byte, v reflect.Value, _ *slab) ([]byte, error) {
	n, src, err := readBytesPrefix(src, "bytes", "string")
	if err != nil {
		return nil, err
	}
	payload := src[:n]
	v = indirectAlloc(v)
	done, err := applyBigDecimalPayload(v, payload)
	if !done {
		err = setBytesValue(v, payload, "big-decimal")
	}
	if err != nil {
		return nil, err
	}
	return src[n:], nil
}

// promoteBytesToStringUUID reads the writer's length-prefixed bytes as a
// canonical UUID string, dispatching to deserFixedUUIDReflect's target arms.
func promoteBytesToStringUUID(src []byte, v reflect.Value, sl *slab) ([]byte, error) {
	n, src, err := readBytesPrefix(src, "string", "bytes")
	if err != nil {
		return nil, err
	}
	v = indirectAlloc(v)
	// [16]byte target wants the parsed UUID bytes; everything else
	// gets the canonical-string view (interface, string, []byte).
	if isUUIDType(v.Type()) {
		s := string(src[:n])
		u, err := parseUUID(s)
		if err != nil {
			return nil, err
		}
		copyBytesToArray(v, u[:])
		return src[n:], nil
	}
	if err := setStringValue(v, src, n, sl); err != nil {
		return nil, err
	}
	return src[n:], nil
}

func promoteBytesToString(src []byte, v reflect.Value, sl *slab) ([]byte, error) {
	n, src, err := readBytesPrefix(src, "string", "bytes")
	if err != nil {
		return nil, err
	}
	if err := setStringValue(indirectAlloc(v), src, n, sl); err != nil {
		return nil, err
	}
	return src[n:], nil
}
