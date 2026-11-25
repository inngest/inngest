package avro

import (
	"fmt"
	"math"
	"math/bits"
	"reflect"
	"sync"
	"time"
	"unsafe"
)

// userfn serializes the value at p into dst. p points directly to the Go
// field's memory; for reading only.
type userfn func(dst []byte, p unsafe.Pointer, depth int) ([]byte, error)

// udeserfn deserializes from src into the value at p.
type udeserfn func(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error)

type fastRecordSer struct {
	typ     reflect.Type
	allFast bool
	fields  []fastFieldSer
}

// fastFieldSer uses a hybrid approach: primitive fields get the unsafe
// fast path (ser != nil), complex/union fields use reflect-based
// FieldByIndex (slowFn != nil). Mixing avoids paying reflect.NewAt
// overhead for the primitive subset.
type fastFieldSer struct {
	offset   uintptr
	name     string
	ser      userfn // non-nil for unsafe-optimized fields (primitives)
	slowFn   serfn  // non-nil for reflect-based fields (complex types)
	slowIdx  []int  // field index path for FieldByIndex (used with slowFn)
	omitzero bool   // true when omitzero acts on this field (fills a default or null)
	// when omitzero fires on a zero value, the bytes to emit: the field's
	// default (ozDefault) or the null-branch index byte (ozNull). Precomputed
	// at compile from omitzeroAction so the fast path matches the reflect path.
	omitzeroBytes []byte
	// omitzeroErr carries serRecordField.defaultErr through the compile, so
	// the fast path refuses exactly where the reflect path refuses. Copying
	// the bytes without the verdict that governs them is how a fast path
	// silently emits what its slow twin rejects.
	omitzeroErr error
}

type fastRecordDeser struct {
	typ     reflect.Type
	allFast bool
	fields  []fastFieldDeser
}

type fastFieldDeser struct {
	offset  uintptr
	name    string
	deser   udeserfn // non-nil for unsafe-optimized fields (primitives)
	slowFn  deserfn  // non-nil for reflect-based fields (complex types)
	slowIdx []int    // field index path for fieldByIndex (used with slowFn)
}

func compileFastSer(fields []serRecordField, names []string, cache *sync.Map, t reflect.Type) *fastRecordSer {
	mapping, err := typeFieldMapping(names, cache, t)
	if err != nil {
		return nil
	}
	fast := &fastRecordSer{typ: t, fields: make([]fastFieldSer, len(fields))}
	allFast := true
	for i := range fields {
		f := &fields[i]
		offset, goType, ok := computeFieldOffset(t, mapping.indices[i])
		// An omitzero field that actually acts (fills a default or a null
		// branch — see omitzeroAction) needs a runtime zero check the
		// unsafe fast path can't do, so fall back to the reflect slow path.
		// A no-op omitzero (non-nullable, no default) just encodes the
		// value, so it stays on the fast path.
		oz := mapping.omitzero[i] && f.omitzeroAction() != ozNoop
		var fn userfn
		if ok && !oz {
			fn = tryCompileFieldSer(f, goType)
		}
		if fn != nil {
			fast.fields[i] = fastFieldSer{
				offset: offset,
				name:   f.name,
				ser:    fn,
			}
		} else {
			allFast = false
			ffs := fastFieldSer{
				name:     f.name,
				slowFn:   f.fn,
				slowIdx:  mapping.indices[i],
				omitzero: oz,
			}
			if oz {
				switch f.omitzeroAction() {
				case ozDefault:
					ffs.omitzeroBytes = f.defaultBytes
					ffs.omitzeroErr = f.defaultErr
				case ozNull:
					nb, _ := nullUnionBytes(f.meta != nil && f.meta.nullSecond)
					ffs.omitzeroBytes = []byte{nb}
				}
			}
			fast.fields[i] = ffs
		}
	}
	fast.allFast = allFast
	return fast
}

func compileFastDeser(fields []deserRecordField, names []string, cache *sync.Map, t reflect.Type) *fastRecordDeser {
	mapping, err := typeFieldMapping(names, cache, t)
	if err != nil {
		return nil
	}
	fast := &fastRecordDeser{typ: t, fields: make([]fastFieldDeser, len(fields))}
	allFast := true
	for i := range fields {
		f := &fields[i]
		offset, goType, ok := computeFieldOffset(t, mapping.indices[i])
		var fn udeserfn
		if ok {
			fn = tryCompileFieldDeser(f, goType)
		}
		if fn != nil {
			fast.fields[i] = fastFieldDeser{
				offset: offset,
				name:   f.name,
				deser:  fn,
			}
		} else {
			allFast = false
			fast.fields[i] = fastFieldDeser{
				name:    f.name,
				slowFn:  f.fn,
				slowIdx: mapping.indices[i],
			}
		}
	}
	fast.allFast = allFast
	return fast
}

// computeFieldOffset computes a flat byte offset for a struct field index
// path. Returns false if the path goes through a pointer (which requires
// runtime dereferencing and cannot be precomputed).
func computeFieldOffset(t reflect.Type, index []int) (uintptr, reflect.Type, bool) {
	var offset uintptr
	for _, i := range index {
		if t.Kind() == reflect.Pointer {
			return 0, nil, false
		}
		f := t.Field(i)
		offset += f.Offset
		t = f.Type
	}
	return offset, t, true
}

func serRecordFast(dst []byte, fast *fastRecordSer, v reflect.Value, depth int) ([]byte, error) {
	if depth >= maxDepth {
		return nil, errTooDeep
	}
	base := v.Addr().UnsafePointer()
	var err error
	for i := range fast.fields {
		f := &fast.fields[i]
		if f.ser != nil {
			dst, err = f.ser(dst, unsafe.Add(base, f.offset), depth+1)
		} else {
			fv := fieldByIndexZero(v, f.slowIdx)
			if f.omitzero && valueIsZero(fv) {
				// f.omitzeroBytes is populated at compile (compileFastSer)
				// from omitzeroAction — the field's default, or the
				// null-branch byte (0x00 for ["null",T], 0x02 for
				// ["T","null"]).
				if f.omitzeroErr != nil {
					return nil, recordFieldError(fast.typ, f.name, f.omitzeroErr)
				}
				dst = append(dst, f.omitzeroBytes...)
				continue
			}
			dst, err = f.slowFn(dst, fv, depth+1)
		}
		if err != nil {
			return nil, recordFieldError(fast.typ, f.name, err)
		}
	}
	return dst, nil
}

// deserRecordFast is the unsafe fast body for ONE record node. Its only
// caller is deserRecord.deser, which has already bumped sl.depth (and run
// the maxDepth guard) for this exact record node. It therefore does NOT
// bump again: a record is one schema node and must cost exactly one depth
// unit regardless of whether it is decoded via the reflect body, the fast
// body, or the reflect-body-then-fast-body chain. Double-bumping here
// would halve the bound for struct-fast records vs the reflect/map path
// and break depth uniformity (deserRecordFastPtr, the OTHER fast entry,
// is reached on child edges that bypass deserRecord.deser, so it keeps
// its own bump — it is the sole record-node entry on those paths).
func deserRecordFast(src []byte, fast *fastRecordDeser, v reflect.Value, sl *slab) ([]byte, error) {
	base := v.Addr().UnsafePointer()
	var err error
	for i := range fast.fields {
		f := &fast.fields[i]
		if f.deser != nil {
			src, err = f.deser(src, unsafe.Add(base, f.offset), sl)
		} else {
			fv, ferr := fieldByIndex(v, f.slowIdx)
			if ferr != nil {
				return nil, recordFieldError(fast.typ, f.name, ferr)
			}
			src, err = f.slowFn(src, fv, sl)
		}
		if err != nil {
			return nil, recordFieldError(fast.typ, f.name, err)
		}
	}
	return src, nil
}

// serRecordFastPtr serializes a record when all fields have unsafe ser fns.
// Only requires a raw pointer to the struct base — no reflect.Value needed.
func serRecordFastPtr(dst []byte, fast *fastRecordSer, base unsafe.Pointer, depth int) ([]byte, error) {
	if depth >= maxDepth {
		return nil, errTooDeep
	}
	var err error
	for i := range fast.fields {
		f := &fast.fields[i]
		dst, err = f.ser(dst, unsafe.Add(base, f.offset), depth+1)
		if err != nil {
			return nil, recordFieldError(fast.typ, f.name, err)
		}
	}
	return dst, nil
}

// deserRecordFastPtr deserializes a record when all fields have unsafe deser fns.
func deserRecordFastPtr(src []byte, fast *fastRecordDeser, base unsafe.Pointer, sl *slab) ([]byte, error) {
	if sl.depth >= maxDepth {
		return nil, errTooDeep
	}
	sl.depth++
	defer func() { sl.depth-- }()
	var err error
	for i := range fast.fields {
		f := &fast.fields[i]
		src, err = f.deser(src, unsafe.Add(base, f.offset), sl)
		if err != nil {
			return nil, recordFieldError(fast.typ, f.name, err)
		}
	}
	return src, nil
}

// serRecordVia dispatches one record encode via the all-fast unsafe
// path when available, falling back to the reflect-based path.
// Callers pass pre-resolved fast (from rec.fastFor) so per-element
// loops compute the lookup once outside the loop.
//
// Do NOT inline this into per-element array loops: the helper exceeds
// the Go inline budget (cost ~290 vs 80) because reflect.NewAt+Elem
// counts as interface method calls, and the t/rec parameters escape
// to heap. The four array sites in this file (usArrayRecord,
// usArrayPtrRecord, usArrayNullUnionRecord, udArrayPtrRecord) keep
// the if/else open-coded with useFast hoisted outside the loop.
func serRecordVia(dst []byte, fast *fastRecordSer, rec *serRecord, t reflect.Type, p unsafe.Pointer, depth int) ([]byte, error) {
	if fast != nil && fast.allFast {
		return serRecordFastPtr(dst, fast, p, depth)
	}
	return rec.ser(dst, reflect.NewAt(t, p).Elem(), depth)
}

// deserRecordVia is serRecordVia's decode-side mirror. The same
// inline-budget caveat applies: do not use in per-element loops.
func deserRecordVia(src []byte, fast *fastRecordDeser, rec *deserRecord, t reflect.Type, p unsafe.Pointer, sl *slab) ([]byte, error) {
	if fast != nil && fast.allFast {
		return deserRecordFastPtr(src, fast, p, sl)
	}
	return rec.deser(src, reflect.NewAt(t, p).Elem(), sl)
}

// tryCompileFieldSer returns a userfn for fields that can be fully handled
// via unsafe pointer access. Returns nil for complex types that must use
// the reflect-based slow path.
func tryCompileFieldSer(f *serRecordField, goType reflect.Type) userfn {
	// Custom types need the reflect slow path for the conversion wrapper.
	if f.meta != nil && (f.meta.hasCustomType || (f.meta.inner != nil && f.meta.inner.hasCustomType)) {
		return nil
	}
	k := goType.Kind()

	// Regular unions need the reflect slow path.
	if f.avroType() == "union" {
		return nil
	}

	// Null-union: *T mapped to ["null", T] or [T, "null"].
	if f.avroType() == "nullunion" {
		if k != reflect.Pointer {
			return nil
		}
		if f.meta.inner == nil {
			return nil
		}
		nullByte, valByte := nullUnionBytes(f.meta.nullSecond)
		inner := f.meta.inner
		if inner.hasCustomType {
			return nil
		}
		innerGoType := goType.Elem()
		// The inner is nil-able beyond the outer pointer: a non-nil *T
		// wrapping a nil pointer (**T), a nil slice (*[]E), a nil map
		// (*map[K]V), or a nil interface (*iface) is nil-equivalent per
		// isNilValue, which peels the outer pointer then nil-checks the
		// bottom kind — so the reflect path encodes the null branch. The
		// unsafe enter (usNullUnionEnter) tests ONLY the outer pointer and
		// would commit to the value branch (then either emit an empty
		// slice/map or fault on the nil inner). Decline the whole nilable
		// inner set to the reflect path, which consults isNilValue; this
		// gate is the fifth dispatch site isNilValue's contract names, kept
		// in lockstep via the shared isNilableKind predicate.
		if isNilableKind(innerGoType.Kind()) {
			return nil
		}
		if inner.serRecord != nil {
			return usNullUnionRecord(inner.serRecord, innerGoType, nullByte, valByte)
		}
		innerFn := tryCompileFieldSer(&serRecordField{meta: inner}, innerGoType)
		if innerFn != nil {
			return usNullUnionPtr(innerFn, nullByte, valByte)
		}
		return nil
	}

	// Array: []T or []*T.
	if f.avroType() == "array" {
		if k != reflect.Slice {
			return nil
		}
		if f.meta.inner == nil {
			return nil
		}
		inner := f.meta.inner
		elemGoType := goType.Elem()
		switch inner.avroType {
		case "nullunion":
			if elemGoType.Kind() != reflect.Pointer {
				return nil
			}
			// Element inner nil-able beyond the outer pointer ([]**T,
			// []*[]E, []*map[K]V, []*iface): &(nilInner) is nil-equivalent
			// per isNilValue (peels the outer pointer then nil-checks the
			// bottom kind → null branch on reflect/JSON); the unsafe array
			// enter tests only the outer pointer and would emit the value
			// branch. Decline the whole nilable inner set to the reflect
			// path. (See the field nullunion case; shared isNilableKind.)
			if isNilableKind(elemGoType.Elem().Kind()) {
				return nil
			}
			nullByte, valByte := nullUnionBytes(inner.nullSecond)
			if inner.inner != nil && inner.inner.hasCustomType {
				return nil
			}
			if inner.inner != nil && inner.inner.serRecord != nil {
				return usArrayNullUnionRecord(inner.inner.serRecord, elemGoType.Elem(), nullByte, valByte)
			}
			innerFn := tryCompileFieldSer(&serRecordField{meta: inner.inner}, elemGoType.Elem())
			if innerFn != nil {
				return usArrayNullUnionPtr(innerFn, nullByte, valByte)
			}
		case "record":
			if inner.serRecord != nil {
				return usArrayRecord(inner.serRecord, elemGoType)
			}
		default:
			innerFn := tryCompileFieldSer(&serRecordField{meta: inner}, elemGoType)
			if innerFn != nil {
				return usArrayDirect(innerFn, elemGoType.Size())
			}
		}
		return nil
	}

	// Record: struct T.
	if f.avroType() == "record" {
		if k != reflect.Struct {
			return nil
		}
		if f.meta.serRecord == nil {
			return nil
		}
		rec := f.meta.serRecord
		// depth (not depth+1): the field-pass call in serRecordFast/Ptr
		// already incremented for the parent-record→child-record edge.
		// The field IS the child record node, so its node depth equals the
		// field-pass depth; serRecordVia treats its depth arg as the
		// record's node depth. Adding +1 here would charge the single edge
		// twice (the reflect path and every decode path charge it once),
		// breaking depth uniformity for a directly-nested struct record.
		return func(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
			return serRecordVia(dst, rec.fastFor(goType), rec, goType, p, depth)
		}
	}

	if k == reflect.Pointer {
		// Bound the pointer-chain peel before recursing on the element type.
		// A cyclic pointer type (type P *P, where P.Elem() == P) recurses here
		// forever AT COMPILE TIME and overflows the stack, unrecoverably. Depth
		// matters too: the reflect encoder peels at most maxIndirectDepth
		// levels, so an unbounded fast path would accept chains it rejects and
		// stop mirroring it byte-for-byte. Counting levels and declining beyond
		// the cap ends the cycle and keeps both accept sets identical — within
		// the cap compiles, deeper routes to the reflect path, which errors
		// uniformly.
		levels := 1
		for e := goType.Elem(); e.Kind() == reflect.Pointer; e = e.Elem() {
			levels++
			if levels > maxIndirectDepth {
				return nil
			}
		}
		inner := tryCompileFieldSer(f, goType.Elem())
		if inner == nil {
			return nil
		}
		return func(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
			pp := *(*unsafe.Pointer)(p)
			if pp == nil {
				return nil, errIndirectNil
			}
			return inner(dst, pp, depth+1)
		}
	}

	// Logical type fast paths for time.Time and time.Duration.
	if f.meta != nil && f.meta.logical != "" {
		return tryCompileLogicalSer(f.meta.logical, f.avroType(), goType)
	}

	switch f.avroType() {
	case "boolean":
		if k == reflect.Bool {
			return usBool
		}
	case "int":
		return usInt(goType)
	case "long":
		return usLong(goType)
	case "float":
		return usFloat(k)
	case "double":
		return usDouble(k)
	case "string":
		if k == reflect.String {
			// json.Number and text-method string-kind types take the safe
			// path: the unsafe *(*string)(p) read would write the raw
			// underlying string, bypassing appendAvroString's json.Number
			// reject and text-out arm — diverging a struct field from the
			// scalar encode of the same value. See stringFastPathEligibleEncode.
			if !stringFastPathEligibleEncode(goType) {
				return nil
			}
			return usString
		}
	case "bytes":
		if k == reflect.Slice && goType.Elem().Kind() == reflect.Uint8 {
			return usBytes
		}
	}

	return nil
}

// tryCompileFieldDeser returns a udeserfn for fields that can be written
// directly via unsafe. Returns nil for complex types that must use the
// reflect-based slow path.
func tryCompileFieldDeser(f *deserRecordField, goType reflect.Type) udeserfn {
	if f.meta != nil && (f.meta.hasCustomType || (f.meta.inner != nil && f.meta.inner.hasCustomType)) {
		return nil
	}
	k := goType.Kind()

	if f.avroType() == "union" {
		return nil
	}

	// Null-union: *T mapped to ["null", T] or [T, "null"].
	if f.avroType() == "nullunion" {
		if k != reflect.Pointer {
			return nil
		}
		if f.meta.inner == nil {
			return nil
		}
		nullByte, valByte := nullUnionBytes(f.meta.nullSecond)
		valIdx := 1
		if f.meta.nullSecond {
			valIdx = 0
		}
		inner := f.meta.inner
		if inner.hasCustomType {
			return nil
		}
		innerGoType := goType.Elem()
		// A multi-level-pointer target (**…T) declines to the reflect path,
		// mirroring the encode field-nullunion decline. udNullUnionRecord/
		// udNullUnionPtr consume the outer pointer (the nullunion optional) via
		// goType.Elem() and then indirectAlloc the remainder — peeling one level
		// past the cap, so a **…T target would accept 1+maxIndirectDepth levels
		// where the reflect deser (deserNullUnionAt hands the un-indirected
		// target to the branch fn, whose single indirectAlloc caps the whole
		// chain at maxIndirectDepth), the encode side, and every other context
		// reject. (A primitive inner already declines below when
		// tryCompileFieldDeser refuses the *…T innerGoType; the record inner has
		// its own branch, so the guard must precede it.)
		if innerGoType.Kind() == reflect.Pointer {
			return nil
		}
		if inner.deserRecord != nil {
			return udNullUnionRecord(inner.deserRecord, innerGoType, valIdx, nullByte, valByte)
		}
		innerFn := tryCompileFieldDeser(&deserRecordField{meta: inner}, innerGoType)
		if innerFn != nil {
			return udNullUnionPtr(innerFn, innerGoType, valIdx, nullByte, valByte)
		}
		return nil
	}

	// Array: []T or []*T.
	if f.avroType() == "array" {
		if k != reflect.Slice {
			return nil
		}
		if f.meta.inner == nil {
			return nil
		}
		inner := f.meta.inner
		if inner.hasCustomType {
			return nil
		}
		elemGoType := goType.Elem()
		switch inner.avroType {
		case "record":
			// A multi-level-pointer element ([]**…record) declines to the safe
			// path: it falls to default → tryCompileFieldDeser rejects the
			// **record element (the record arm needs a struct, the generic
			// pointer arm declines all pointers), so the field's reflect deser
			// handles it with a single indirectAlloc budget. The fast
			// udArrayPtrRecord peels one pointer level inline and rec.deser peels
			// a FURTHER maxIndirectDepth, accepting 1+maxIndirectDepth — one past
			// the cap every other path enforces. Mirrors the encode usArrayRecord
			// decline; the fast path stays single-pointer only ([]*record).
			if inner.deserRecord != nil && elemGoType.Kind() == reflect.Pointer && elemGoType.Elem().Kind() != reflect.Pointer {
				return udArrayPtrRecord(inner.deserRecord, elemGoType.Elem(), goType, inner.minBytes)
			}
		default:
			innerFn := tryCompileFieldDeser(&deserRecordField{meta: inner}, elemGoType)
			if innerFn != nil {
				return udArrayDirect(innerFn, elemGoType.Size(), goType, inner.minBytes)
			}
		}
		return nil
	}

	// Record: struct T.
	if f.avroType() == "record" {
		if k != reflect.Struct {
			return nil
		}
		if f.meta.deserRecord == nil {
			return nil
		}
		rec := f.meta.deserRecord
		return func(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
			return deserRecordVia(src, rec.fastFor(goType), rec, goType, p, sl)
		}
	}

	// Pointer fields need GC write barriers for allocation; use slow path.
	if k == reflect.Pointer {
		return nil
	}

	// Logical type fast paths for time.Time and time.Duration.
	if f.meta != nil && f.meta.logical != "" {
		return tryCompileLogicalDeser(f.meta.logical, f.avroType(), goType)
	}

	switch f.avroType() {
	case "boolean":
		if k == reflect.Bool {
			return udBool
		}
	case "int":
		return udInt(goType)
	case "long":
		return udLong(goType)
	case "float":
		return udFloat(k)
	case "double":
		return udDouble(goType)
	case "string":
		if k == reflect.String {
			// json.Number and TextUnmarshaler string-kind types take the
			// safe path: the unsafe *(*string)(p)=... store would bypass
			// setStringValue's json.Number RFC 8259 guard and UnmarshalText
			// arm — diverging a struct field from the scalar decode of the
			// same target. See stringFastPathEligibleDecode.
			if !stringFastPathEligibleDecode(goType) {
				return nil
			}
			return udStringDeser
		}
	case "bytes":
		if k == reflect.Slice && goType.Elem().Kind() == reflect.Uint8 {
			return udBytesDeser
		}
	}

	return nil
}

// ---- Logical type unsafe serializers ----

// usTimestampLogicals maps the six long-typed timestamp logicals to
// their time.Time-target unsafe serializer. Non-time.Time targets fall
// back to usLong(kind) at the dispatch site below.
var usTimestampLogicals = map[string]userfn{
	"timestamp-millis":       usTimestampMillis,
	"timestamp-micros":       usTimestampMicros,
	"timestamp-nanos":        usTimestampNanos,
	"local-timestamp-millis": usLocalTimestampMillis,
	"local-timestamp-micros": usLocalTimestampMicros,
	"local-timestamp-nanos":  usLocalTimestampNanos,
}

func tryCompileLogicalSer(logical, avroType string, goType reflect.Type) userfn {
	if fn, ok := usTimestampLogicals[logical]; ok {
		if goType == timeType {
			return fn
		}
		return usLong(goType)
	}
	switch logical {
	case "date":
		if goType == timeType {
			return usDate
		}
		return usInt(goType)
	case "time-millis":
		if goType == durationType {
			return usTimeMillis
		}
		if goType == timeType {
			return usTimeMillisTime
		}
		return usInt(goType)
	case "time-micros":
		if goType == durationType {
			return usTimeMicros
		}
		if goType == timeType {
			return usTimeMicrosTime
		}
		return usLong(goType)
	case "duration":
		if goType == avroDurationType {
			return usDuration
		}
		return nil
	case "uuid":
		if avroType == "fixed" {
			if goType.Kind() == reflect.String {
				// text-method (and json.Number) string-kind types take the
				// safe path so serFixedUUIDReflect's text-before-string-kind
				// order applies; json.Number reaches the same parseUUID
				// outcome on either path. See stringFastPathEligibleEncode.
				if !stringFastPathEligibleEncode(goType) {
					return nil
				}
				return usFixedUUIDString
			}
			return nil // [16]byte, []byte handled by default fixed ser
		}
		if isUUIDType(goType) {
			return usUUID
		}
		if goType.Kind() == reflect.String {
			// json.Number/text string+uuid: usString would emit the raw
			// underlying string as the wire UUID with no validation / no
			// text-out; the safe path's serUUID → appendAvroString applies
			// both. See stringFastPathEligibleEncode.
			if !stringFastPathEligibleEncode(goType) {
				return nil
			}
			return usString
		}
	}
	return nil
}

// udTimestampLogicals maps the six long-typed timestamp logicals to
// their time.Time-target unsafe deserializer (local-timestamp-* and
// timestamp-* decode identically; the wire long is interpreted the same
// way — see logical.go for the encode-side wall-clock vs instant note).
var udTimestampLogicals = map[string]udeserfn{
	"timestamp-millis":       udTimestampMillis,
	"timestamp-micros":       udTimestampMicros,
	"timestamp-nanos":        udTimestampNanos,
	"local-timestamp-millis": udTimestampMillis,
	"local-timestamp-micros": udTimestampMicros,
	"local-timestamp-nanos":  udTimestampNanos,
}

func tryCompileLogicalDeser(logical, avroType string, goType reflect.Type) udeserfn {
	if fn, ok := udTimestampLogicals[logical]; ok {
		if goType == timeType {
			return fn
		}
		return udLong(goType)
	}
	switch logical {
	case "date":
		if goType == timeType {
			return udDate
		}
		return udInt(goType)
	case "time-millis":
		if goType == durationType {
			return udTimeMillis
		}
		if goType == timeType {
			return udTimeMillisTime
		}
		return udInt(goType)
	case "time-micros":
		if goType == durationType {
			return udTimeMicros
		}
		if goType == timeType {
			return udTimeMicrosTime
		}
		return udLong(goType)
	case "duration":
		if goType == avroDurationType {
			return udDuration
		}
		return nil
	case "uuid":
		if avroType == "fixed" {
			if isUUIDType(goType) {
				return udFixedUUID
			}
			if goType.Kind() == reflect.String {
				// Safe path enforces the json.Number guard and the
				// TextUnmarshaler arm (deserFixedUUIDReflect tries
				// UnmarshalText before the string-kind setStringTarget).
				if !stringFastPathEligibleDecode(goType) {
					return nil
				}
				return udFixedUUIDString
			}
			return nil // any, []byte etc. handled by reflect path
		}
		if isUUIDType(goType) {
			return udUUID
		}
		if goType.Kind() == reflect.String {
			// Safe path enforces the json.Number guard and the
			// TextUnmarshaler arm.
			if !stringFastPathEligibleDecode(goType) {
				return nil
			}
			return udStringDeser
		}
	}
	return nil
}

// usTimeAsLong is the shared body of the six unsafe time-logical "long"
// serializers — direct *(*time.Time)(p) read, conv, SemanticError-wrap.
func usTimeAsLong(dst []byte, p unsafe.Pointer, conv func(time.Time) (int64, error)) ([]byte, error) {
	n, err := conv(*(*time.Time)(p))
	if err != nil {
		return nil, &SemanticError{GoType: timeType, AvroType: "long", Err: err}
	}
	return appendVarlong(dst, n), nil
}

func usTimestampMillis(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
	return usTimeAsLong(dst, p, timeToTimestampMillis)
}

func usTimestampMicros(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
	return usTimeAsLong(dst, p, timeToTimestampMicros)
}

func usTimestampNanos(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
	return usTimeAsLong(dst, p, timeToTimestampNanos)
}

// Local-timestamp unsafe serializers: see logical.go for rationale.

func usLocalTimestampMillis(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
	return usTimeAsLong(dst, p, timeToLocalTimestampMillis)
}

func usLocalTimestampMicros(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
	return usTimeAsLong(dst, p, timeToLocalTimestampMicros)
}

func usLocalTimestampNanos(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
	return usTimeAsLong(dst, p, timeToLocalTimestampNanos)
}

func usDate(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
	d, err := timeToDate(*(*time.Time)(p))
	if err != nil {
		return nil, &SemanticError{GoType: timeType, AvroType: "date", Err: err}
	}
	return appendVarint(dst, d), nil
}

func usTimeMillis(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
	ms, err := durationToTimeMillis(*(*time.Duration)(p))
	if err != nil {
		return nil, &SemanticError{GoType: durationType, AvroType: "time-millis", Err: err}
	}
	return appendVarint(dst, ms), nil
}

// usTimeMillisTime is the time.Time variant of usTimeMillis — extracts
// time-of-day fields and encodes as time-millis, mirroring the
// serTimeMillis(timeType) safe-path arm.
func usTimeMillisTime(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
	ms, err := durationToTimeMillis(timeOfDay(*(*time.Time)(p)))
	if err != nil {
		return nil, &SemanticError{GoType: timeType, AvroType: "time-millis", Err: err}
	}
	return appendVarint(dst, ms), nil
}

func usTimeMicros(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
	return appendVarlong(dst, (*(*time.Duration)(p)).Microseconds()), nil
}

// usTimeMicrosTime mirrors serTimeMicros(timeType).
func usTimeMicrosTime(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
	return appendVarlong(dst, timeOfDay(*(*time.Time)(p)).Microseconds()), nil
}

// udTimeFromVarint reads a varint and stores conv(val) into *T.
func udTimeFromVarint[T any](conv func(int32) T) udeserfn {
	return func(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
		val, src, err := readVarint(src)
		if err != nil {
			return nil, err
		}
		*(*T)(p) = conv(val)
		return src, nil
	}
}

// udTimeFromVarlong is udTimeFromVarint's varlong sibling.
func udTimeFromVarlong[T any](conv func(int64) T) udeserfn {
	return func(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
		val, src, err := readVarlong(src)
		if err != nil {
			return nil, err
		}
		*(*T)(p) = conv(val)
		return src, nil
	}
}

// udTimeFromVarlongChecked is udTimeFromVarlong for fallible converters.
func udTimeFromVarlongChecked[T any](conv func(int64) (T, error)) udeserfn {
	return func(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
		val, src, err := readVarlong(src)
		if err != nil {
			return nil, err
		}
		out, err := conv(val)
		if err != nil {
			return nil, err
		}
		*(*T)(p) = out
		return src, nil
	}
}

// *Time variants materialize the time-of-day count as a time.Time at
// epoch midnight (UTC) — used when a struct field is typed as time.Time
// but the schema is time-millis/time-micros.
var (
	udTimestampMillis = udTimeFromVarlong(timestampMillisToTime)
	udTimestampMicros = udTimeFromVarlong(timestampMicrosToTime)
	udTimestampNanos  = udTimeFromVarlong(timestampNanosToTime)
	udDate            = udTimeFromVarint(dateToTime)
	udTimeMillis      = udTimeFromVarint(timeMillisToDuration)
	udTimeMillisTime  = udTimeFromVarint(func(v int32) time.Time {
		return timeOfDayToTime(timeMillisToDuration(v))
	})
	udTimeMicros     = udTimeFromVarlongChecked(timeMicrosToDuration)
	udTimeMicrosTime = udTimeFromVarlongChecked(func(v int64) (time.Time, error) {
		d, err := timeMicrosToDuration(v)
		if err != nil {
			return time.Time{}, err
		}
		return timeOfDayToTime(d), nil
	})
)

func usDuration(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
	b := (*(*Duration)(p)).Bytes()
	return append(dst, b[:]...), nil
}

func udDuration(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
	if err := needLen(src, 12, "duration"); err != nil {
		return nil, err
	}
	*(*Duration)(p) = DurationFromBytes(src[:12])
	return src[12:], nil
}

func usUUID(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
	u := *(*[16]byte)(p)
	return doSerString(dst, uuidToString(u)), nil
}

func udUUID(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
	n, src, err := readLength(src, "string")
	if err != nil {
		return nil, err
	}
	u, err := parseUUIDBytes(src[:n])
	if err != nil {
		return nil, err
	}
	*(*[16]byte)(p) = u
	return src[n:], nil
}

// readFixedUUID validates len(src) ≥ 16 and returns the next UUID bytes
// plus the advanced source slice. Shared body of udFixedUUID (writes
// [16]byte) and udFixedUUIDString (writes the canonical string form).
func readFixedUUID(src []byte) ([16]byte, []byte, error) {
	if err := needLen(src, 16, "uuid"); err != nil {
		return [16]byte{}, nil, err
	}
	return [16]byte(src[:16]), src[16:], nil
}

// udFixedUUID reads 16 raw bytes from a fixed(16) UUID and writes a [16]byte.
// Used when the target is [16]byte or any (interface).
func udFixedUUID(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
	u, src, err := readFixedUUID(src)
	if err == nil {
		*(*[16]byte)(p) = u
	}
	return src, err
}

// udFixedUUIDString reads 16 raw bytes from a fixed(16) UUID and writes a
// formatted UUID string (e.g. "550e8400-e29b-41d4-a716-446655440000").
func udFixedUUIDString(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
	u, src, err := readFixedUUID(src)
	if err == nil {
		*(*string)(p) = uuidToString(u)
	}
	return src, err
}

// usFixedUUIDString serializes a UUID string to 16 raw fixed bytes.
func usFixedUUIDString(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
	s := *(*string)(p)
	u, err := parseUUID(s)
	if err != nil {
		return nil, err
	}
	return append(dst, u[:]...), nil
}

// ---- Unsafe serializers ----
// These read values directly via unsafe.Pointer. No string→[]byte
// conversions; all reads go through typed pointer dereferences.

func usBool(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
	if *(*bool)(p) {
		return append(dst, 1), nil
	}
	return append(dst, 0), nil
}

// usVarintFrom is udVarintTo's serialize-side mirror: reads a typed
// integer via unsafe pointer, range-checks, writes as Avro int. t is the
// Go field type, set on the SemanticError so the unsafe struct-field path
// reports the same GoType as the reflect path (appendAvroInt in ser.go).
func usVarintFrom[T intLike](lo, hi int64, t reflect.Type) userfn {
	return func(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
		n := int64(*(*T)(p))
		if n < lo || n > hi {
			return nil, &SemanticError{GoType: t, AvroType: "int", Err: fmt.Errorf("value %d overflows int32", n)}
		}
		return appendVarint(dst, int32(n)), nil
	}
}

// usVarlongFrom is usVarintFrom's varlong (Avro long) sibling.
func usVarlongFrom[T intLike](lo, hi int64, t reflect.Type) userfn {
	return func(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
		n := int64(*(*T)(p))
		if n < lo || n > hi {
			return nil, &SemanticError{GoType: t, AvroType: "long", Err: fmt.Errorf("value %d overflows int64", n)}
		}
		return appendVarlong(dst, n), nil
	}
}

// usVarlongFromUnsigned is usVarlongFrom for unsigned T. The upper
// bound is checked in uint64 space since uint64 > MaxInt64 can't
// round-trip through int64.
func usVarlongFromUnsigned[T uint | uint8 | uint16 | uint32 | uint64](t reflect.Type) userfn {
	return func(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
		n := uint64(*(*T)(p))
		if n > math.MaxInt64 {
			return nil, &SemanticError{GoType: t, AvroType: "long", Err: fmt.Errorf("value %d overflows int64", n)}
		}
		return appendVarlong(dst, int64(n)), nil
	}
}

func usInt(t reflect.Type) userfn {
	switch t.Kind() {
	case reflect.Int:
		// On 64-bit int can exceed int32; bound check applies.
		return usVarintFrom[int](math.MinInt32, math.MaxInt32, t)
	case reflect.Int8:
		return usVarintFrom[int8](math.MinInt64, math.MaxInt64, t)
	case reflect.Int16:
		return usVarintFrom[int16](math.MinInt64, math.MaxInt64, t)
	case reflect.Int32:
		return usVarintFrom[int32](math.MinInt64, math.MaxInt64, t)
	case reflect.Int64:
		return usVarintFrom[int64](math.MinInt32, math.MaxInt32, t)
	case reflect.Uint:
		// uint as int64 max — bound applies on 64-bit too (uint > MaxInt32 possible).
		return usVarintFrom[uint](0, math.MaxInt32, t)
	case reflect.Uint8:
		return usVarintFrom[uint8](math.MinInt64, math.MaxInt64, t)
	case reflect.Uint16:
		return usVarintFrom[uint16](math.MinInt64, math.MaxInt64, t)
	case reflect.Uint32:
		return usVarintFrom[uint32](0, math.MaxInt32, t)
	case reflect.Uint64:
		// uint64 > MaxInt64 can't be represented as int64 — but the
		// usVarintFrom int64 cast already truncates negative-when-
		// reinterpreted values. Use the unsigned-aware bound directly.
		return func(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
			n := *(*uint64)(p)
			if n > math.MaxInt32 {
				return nil, &SemanticError{GoType: t, AvroType: "int", Err: fmt.Errorf("value %d overflows int32", n)}
			}
			return appendVarint(dst, int32(n)), nil
		}
	default:
		return nil
	}
}

func usLong(t reflect.Type) userfn {
	switch t.Kind() {
	case reflect.Int:
		return usVarlongFrom[int](math.MinInt64, math.MaxInt64, t)
	case reflect.Int8:
		return usVarlongFrom[int8](math.MinInt64, math.MaxInt64, t)
	case reflect.Int16:
		return usVarlongFrom[int16](math.MinInt64, math.MaxInt64, t)
	case reflect.Int32:
		return usVarlongFrom[int32](math.MinInt64, math.MaxInt64, t)
	case reflect.Int64:
		return usVarlongFrom[int64](math.MinInt64, math.MaxInt64, t)
	case reflect.Uint:
		return usVarlongFromUnsigned[uint](t)
	case reflect.Uint8:
		return usVarlongFromUnsigned[uint8](t)
	case reflect.Uint16:
		return usVarlongFromUnsigned[uint16](t)
	case reflect.Uint32:
		return usVarlongFromUnsigned[uint32](t)
	case reflect.Uint64:
		return usVarlongFromUnsigned[uint64](t)
	default:
		return nil
	}
}

func usFloat(k reflect.Kind) userfn {
	switch k {
	case reflect.Float32:
		return func(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
			return appendUint32(dst, math.Float32bits(*(*float32)(p))), nil
		}
	case reflect.Float64:
		return func(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
			// Lossy-destination: finite float64 → float32 silently narrows
			// to ±Inf when out of range (matches appendAvroFloat32 / Java).
			return appendUint32(dst, math.Float32bits(float32(*(*float64)(p)))), nil
		}
	default:
		return nil
	}
}

func usDouble(k reflect.Kind) userfn {
	switch k {
	case reflect.Float32:
		return func(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
			return appendUint64(dst, math.Float64bits(float64(*(*float32)(p)))), nil
		}
	case reflect.Float64:
		return func(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
			return appendUint64(dst, math.Float64bits(*(*float64)(p))), nil
		}
	default:
		return nil
	}
}

// usString reads the string header directly from memory.
func usString(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
	return doSerString(dst, *(*string)(p)), nil
}

// usBytes reads the slice header directly from memory.
func usBytes(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
	b := *(*[]byte)(p)
	dst = appendVarlong(dst, int64(len(b)))
	return append(dst, b...), nil
}

// ---- Unsafe deserializers ----
// For types without GC pointers (bool, ints, floats), write directly via
// unsafe. For types containing GC pointers (string, []byte), typed pointer
// stores trigger GC write barriers automatically. All decoded values are
// freshly allocated copies; no aliasing of src.

func udBool(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
	if len(src) < 1 {
		return nil, &ShortBufferError{Type: "boolean"}
	}
	*(*bool)(p) = src[0] == 1
	return src[1:], nil
}

// intLike covers all signed and unsigned integer kinds.
type intLike interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// udVarintTo reads a varint (int32 wire), range-checks against [lo, hi]
// in int64 space, and stores the narrowed result into *T. lo=MinInt64 /
// hi=MaxInt64 disables the range check.
func udVarintTo[T intLike](lo, hi int64, avroType string, t reflect.Type) udeserfn {
	return func(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
		v, src, err := readVarint(src)
		if err != nil {
			return nil, err
		}
		if int64(v) < lo || int64(v) > hi {
			return nil, &SemanticError{GoType: t, AvroType: avroType, Err: fmt.Errorf("value %d overflows %s", v, t)}
		}
		*(*T)(p) = T(v)
		return src, nil
	}
}

// udVarlongTo is udVarintTo's varlong (int64 wire) sibling.
func udVarlongTo[T intLike](lo, hi int64, avroType string, t reflect.Type) udeserfn {
	return func(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
		v, src, err := readVarlong(src)
		if err != nil {
			return nil, err
		}
		if v < lo || v > hi {
			return nil, &SemanticError{GoType: t, AvroType: avroType, Err: fmt.Errorf("value %d overflows %s", v, t)}
		}
		*(*T)(p) = T(v)
		return src, nil
	}
}

func udInt(t reflect.Type) udeserfn {
	switch t.Kind() {
	case reflect.Int:
		// int32 wire always fits in int (int is int32 or int64).
		return udVarintTo[int](math.MinInt64, math.MaxInt64, "int", t)
	case reflect.Int8:
		return udVarintTo[int8](math.MinInt8, math.MaxInt8, "int", t)
	case reflect.Int16:
		return udVarintTo[int16](math.MinInt16, math.MaxInt16, "int", t)
	case reflect.Int32:
		return udVarintTo[int32](math.MinInt64, math.MaxInt64, "int", t)
	case reflect.Int64:
		return udVarintTo[int64](math.MinInt64, math.MaxInt64, "int", t)
	case reflect.Uint:
		// Varint wire is int32; uint is always wide enough — only
		// the lower-bound (v < 0) check matters. hi=MaxInt64 is
		// effectively unbounded.
		return udVarintTo[uint](0, math.MaxInt64, "int", t)
	case reflect.Uint8:
		return udVarintTo[uint8](0, math.MaxUint8, "int", t)
	case reflect.Uint16:
		return udVarintTo[uint16](0, math.MaxUint16, "int", t)
	case reflect.Uint32:
		return udVarintTo[uint32](0, math.MaxInt64, "int", t)
	case reflect.Uint64:
		return udVarintTo[uint64](0, math.MaxInt64, "int", t)
	default:
		return nil
	}
}

func udLong(t reflect.Type) udeserfn {
	switch t.Kind() {
	case reflect.Int:
		// On 64-bit int holds any int64; on 32-bit int is int32 so
		// the bound check is real. math.MinInt/MaxInt resolve per
		// platform.
		return udVarlongTo[int](math.MinInt, math.MaxInt, "long", t)
	case reflect.Int8:
		return udVarlongTo[int8](math.MinInt8, math.MaxInt8, "long", t)
	case reflect.Int16:
		return udVarlongTo[int16](math.MinInt16, math.MaxInt16, "long", t)
	case reflect.Int32:
		return udVarlongTo[int32](math.MinInt32, math.MaxInt32, "long", t)
	case reflect.Int64:
		return udVarlongTo[int64](math.MinInt64, math.MaxInt64, "long", t)
	case reflect.Uint:
		// On 64-bit uint = uint64 holds any non-negative int64; on
		// 32-bit uint = uint32, so cap at MaxUint32. bits.UintSize
		// is a compile-time constant so the branch resolves to
		// dead code on the non-matching platform.
		hi := int64(math.MaxInt64)
		if bits.UintSize == 32 {
			hi = math.MaxUint32
		}
		return udVarlongTo[uint](0, hi, "long", t)
	case reflect.Uint8:
		return udVarlongTo[uint8](0, math.MaxUint8, "long", t)
	case reflect.Uint16:
		return udVarlongTo[uint16](0, math.MaxUint16, "long", t)
	case reflect.Uint32:
		return udVarlongTo[uint32](0, math.MaxUint32, "long", t)
	case reflect.Uint64:
		return udVarlongTo[uint64](0, math.MaxInt64, "long", t)
	default:
		return nil
	}
}

func udFloat(k reflect.Kind) udeserfn {
	switch k {
	case reflect.Float32:
		return func(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
			u, src, err := readUint32(src)
			if err != nil {
				return nil, err
			}
			*(*float32)(p) = math.Float32frombits(u)
			return src, nil
		}
	case reflect.Float64:
		return func(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
			u, src, err := readUint32(src)
			if err != nil {
				return nil, err
			}
			*(*float64)(p) = float64(math.Float32frombits(u))
			return src, nil
		}
	default:
		return nil
	}
}

func udDouble(t reflect.Type) udeserfn {
	switch t.Kind() {
	case reflect.Float32:
		return func(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
			u, src, err := readUint64(src)
			if err != nil {
				return nil, err
			}
			f := math.Float64frombits(u)
			// Match deserDouble (safe path): reject silent narrowing of a
			// finite float64 to ±Inf when the destination is float32.
			if finiteFloat32Overflows(f) {
				return nil, &SemanticError{GoType: t, AvroType: "double", Err: fmt.Errorf("value %g overflows float32", f)}
			}
			*(*float32)(p) = float32(f)
			return src, nil
		}
	case reflect.Float64:
		return func(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
			u, src, err := readUint64(src)
			if err != nil {
				return nil, err
			}
			*(*float64)(p) = math.Float64frombits(u)
			return src, nil
		}
	default:
		return nil
	}
}

// udStringDeser writes the string directly via typed pointer store.
// *(*string)(p) = s triggers GC write barriers automatically.
// string(src[:n]) always copies, so the decoded string owns its memory.
func udStringDeser(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
	n, src, err := readLength(src, "string")
	if err != nil {
		return nil, err
	}
	*(*string)(p) = sl.string(src, n)
	return src[n:], nil
}

// udBytesDeser writes the byte slice directly via typed pointer store.
// *(*[]byte)(p) = b triggers GC write barriers automatically.
// make+copy ensures the decoded slice owns its memory.
func udBytesDeser(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
	n, src, err := readLength(src, "bytes")
	if err != nil {
		return nil, err
	}
	b := make([]byte, n)
	copy(b, src[:n])
	*(*[]byte)(p) = b
	return src[n:], nil
}

// ---- Null-union unsafe ser/deser ----

// nullUnionBytes returns the single-byte varint-encoded union index for
// the null and value branches. Zigzag varint: index 0 encodes as 0x00,
// index 1 encodes as 0x02. So ["null",T] → null=0x00, val=0x02;
// ["T","null"] → val=0x00, null=0x02.
func nullUnionBytes(nullSecond bool) (nullByte, valByte byte) {
	if nullSecond {
		return 2, 0
	}
	return 0, 2
}

// usNullUnionEnter is the encode-side mirror of udNullUnionEnter:
// emits null-branch byte when *(*unsafe.Pointer)(p) is nil, value-
// branch byte otherwise. Returns (pp, dst-after-tag, isNull). When
// isNull is true the caller returns dst directly; otherwise pp is
// the inner *T address the per-branch ser fn fills.
func usNullUnionEnter(dst []byte, p unsafe.Pointer, nullByte, valByte byte) (pp unsafe.Pointer, _ []byte, isNull bool) {
	pp = *(*unsafe.Pointer)(p)
	if pp == nil {
		return nil, append(dst, nullByte), true
	}
	return pp, append(dst, valByte), false
}

// usNullUnionPtr handles null-union ser for *T where T has a primitive unsafe serializer.
// nullByte/valByte are the varint-encoded index bytes for null and value branches.
func usNullUnionPtr(inner userfn, nullByte, valByte byte) userfn {
	return func(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
		// Guard at the union node, mirroring the decode-side udNullUnionPtr
		// and the reflect serNullUnionAt: the union is a schema node and
		// must both charge its edge (depth+1 into the branch below) AND
		// guard. See the depth-uniformity invariant in deserNullUnionAt.
		if depth >= maxDepth {
			return nil, errTooDeep
		}
		pp, dst, isNull := usNullUnionEnter(dst, p, nullByte, valByte)
		if isNull {
			return dst, nil
		}
		return inner(dst, pp, depth+1)
	}
}

// usNullUnionRecord handles null-union ser for *T where T is a record.
func usNullUnionRecord(rec *serRecord, innerType reflect.Type, nullByte, valByte byte) userfn {
	return func(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
		// Guard at the union node, mirroring the decode-side
		// udNullUnionRecord and the reflect serNullUnionAt. The union node
		// is charged here (guard) and its edge to the inner record is
		// charged by the depth+1 below; the record node self-charges on
		// entry, so a ["null", Record] link costs two depth units on every
		// path. See the depth-uniformity invariant in deserNullUnionAt.
		if depth >= maxDepth {
			return nil, errTooDeep
		}
		pp, dst, isNull := usNullUnionEnter(dst, p, nullByte, valByte)
		if isNull {
			return dst, nil
		}
		return serRecordVia(dst, rec.fastFor(innerType), rec, innerType, pp, depth+1)
	}
}

// udNullUnionEnter is the shared preamble for udNullUnionPtr / udNullUnionRecord:
// read the union index, nil-out *T on the null branch, allocate inner
// storage on the value branch. Returns (pp, src, isNull, err). When
// isNull is true the caller should return src directly (the pointer
// field has already been zeroed). Otherwise pp points at freshly-
// allocated inner storage that the caller's per-branch decoder fills.
func udNullUnionEnter(src []byte, p unsafe.Pointer, innerType reflect.Type, valIdx int, nullByte, valByte byte) (pp unsafe.Pointer, rest []byte, isNull bool, err error) {
	isVal, src, err := readNullUnionIndex(src, valIdx, nullByte, valByte)
	if err != nil {
		return nil, nil, false, err
	}
	if !isVal {
		*(*unsafe.Pointer)(p) = nil
		return nil, src, true, nil
	}
	pp = *(*unsafe.Pointer)(p)
	if pp == nil {
		v := reflect.New(innerType)
		pp = v.UnsafePointer()
		*(*unsafe.Pointer)(p) = pp
	}
	return pp, src, false, nil
}

// udNullUnionPtr handles null-union deser for *T where T has a primitive unsafe deserializer.
//
// The union is a schema node and bumps sl.depth once, exactly like the
// reflect deserNullUnionAt, the general deserUnion.deser, and the encode
// side (usNullUnionPtr passes its branch at depth+1). See the depth-
// uniformity invariant in deserNullUnionAt. The decrement is inline (not
// deferred) for the same hot-path reason as udNullUnionRecord below.
func udNullUnionPtr(inner udeserfn, innerType reflect.Type, valIdx int, nullByte, valByte byte) udeserfn {
	return func(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
		if sl.depth >= maxDepth {
			return nil, errTooDeep
		}
		sl.depth++
		pp, src, isNull, err := udNullUnionEnter(src, p, innerType, valIdx, nullByte, valByte)
		if err != nil || isNull {
			sl.depth--
			return src, err
		}
		src, err = inner(src, pp, sl)
		sl.depth--
		return src, err
	}
}

// udNullUnionRecord handles null-union deser for *T where T is a record.
//
// The union node bumps sl.depth once here and the inner record bumps on its own
// entry, so a ["null", Record] link costs two depth units on every path. See
// the depth-uniformity invariant in deserNullUnionAt.
//
// The decrement is inline rather than deferred, here and in udNullUnionPtr:
// this is the recursive-decode hot path, and benchstat showed the open-coded
// defer costs ~10% per link on BenchmarkDeserializeRecursive. Every post-bump
// return decrements first, so depth is restored on success and error alike. The
// over-bound abort returns before the bump.
func udNullUnionRecord(rec *deserRecord, innerType reflect.Type, valIdx int, nullByte, valByte byte) udeserfn {
	return func(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
		if sl.depth >= maxDepth {
			return nil, errTooDeep
		}
		sl.depth++
		pp, src, isNull, err := udNullUnionEnter(src, p, innerType, valIdx, nullByte, valByte)
		if err != nil || isNull {
			sl.depth--
			return src, err
		}
		src, err = deserRecordVia(src, rec.fastFor(innerType), rec, innerType, pp, sl)
		sl.depth--
		return src, err
	}
}

// ---- Array unsafe ser/deser ----
//
// The five per-element loops (depth-check + length-prefix + early-exit
// + per-element body + terminator) are intentionally inlined rather
// than factored into a generic helper. A usSliceFrame[Elem] factoring
// regressed BenchmarkLargeArrayEncode (the []*Record hot path) by ~16%:
// the extra closure call per slice combined with the Go compiler
// declining to inline the generic at each call site costs measurable
// per-array work.

// usArrayRecord handles array ser for []T or []*T where items are records.
func usArrayRecord(rec *serRecord, elemGoType reflect.Type) userfn {
	if elemGoType.Kind() == reflect.Pointer {
		// A multi-level-pointer element ([]**…record) declines to the reflect
		// path. usArrayPtrRecord peels one pointer level inline (the
		// []unsafe.Pointer element read) and then hands the remainder to
		// rec.ser, whose indirect peels a FURTHER maxIndirectDepth levels — so
		// the element would accept a chain 1+maxIndirectDepth deep, one past the
		// cap the leaf encoder, the JSON encoder, top-level reflect, and the
		// nullunion array arms all enforce. The reflect serArray hands the
		// element to rec.ser with a single indirect budget, capping at
		// maxIndirectDepth. (Mirrors the []**…record nullunion array decline.)
		if elemGoType.Elem().Kind() == reflect.Pointer {
			return nil
		}
		return usArrayPtrRecord(rec, elemGoType.Elem())
	}
	elemSize := elemGoType.Size()
	return func(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
		if depth >= maxDepth {
			return nil, errTooDeep
		}
		bs := *(*[]byte)(p)
		n := len(bs)
		dst = appendVarlong(dst, int64(n))
		if n == 0 {
			return dst, nil
		}
		bodyStart := len(dst)
		data := unsafe.Pointer(unsafe.SliceData(bs))
		fast := rec.fastFor(elemGoType)
		useFast := fast != nil && fast.allFast
		var err error
		for i := range n {
			elemP := unsafe.Add(data, uintptr(i)*elemSize)
			if useFast {
				dst, err = serRecordFastPtr(dst, fast, elemP, depth+1)
			} else {
				dst, err = rec.ser(dst, reflect.NewAt(elemGoType, elemP).Elem(), depth+1)
			}
			if err != nil {
				return nil, err
			}
		}
		if err := arrayZeroByteEncodeCompliance(len(dst) == bodyStart, n); err != nil {
			return nil, err
		}
		return append(dst, 0), nil
	}
}

// usArrayPtrRecord handles array ser for []*T where items are records.
func usArrayPtrRecord(rec *serRecord, innerType reflect.Type) userfn {
	return func(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
		if depth >= maxDepth {
			return nil, errTooDeep
		}
		s := *(*[]unsafe.Pointer)(p)
		n := len(s)
		dst = appendVarlong(dst, int64(n))
		if n == 0 {
			return dst, nil
		}
		bodyStart := len(dst)
		fast := rec.fastFor(innerType)
		useFast := fast != nil && fast.allFast
		var err error
		for _, pp := range s {
			if pp == nil {
				return nil, errIndirectNil
			}
			if useFast {
				dst, err = serRecordFastPtr(dst, fast, pp, depth+1)
			} else {
				dst, err = rec.ser(dst, reflect.NewAt(innerType, pp).Elem(), depth+1)
			}
			if err != nil {
				return nil, err
			}
		}
		if err := arrayZeroByteEncodeCompliance(len(dst) == bodyStart, n); err != nil {
			return nil, err
		}
		return append(dst, 0), nil
	}
}

// usArrayNullUnionRecord handles array ser for []*T where items are ["null", Record].
//
// The element type ["null", Record] is a union schema NODE between the
// array and the record. It must cost one depth unit, exactly like the
// decode path (udArrayDirect → udNullUnionRecord → record) and every JSON
// path. The array node is entered at depth; each element's union node is
// entered at depth+1 and the inner record at depth+2. The per-element
// union guard (depth+1 >= maxDepth) is loop-invariant, so it is hoisted
// before the loop — matching decode's udNullUnionRecord, which guards
// before reading the index (so a null-branch element at the boundary
// trips too); only the empty array (no element unions entered) escapes.
func usArrayNullUnionRecord(rec *serRecord, innerType reflect.Type, nullByte, valByte byte) userfn {
	return func(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
		if depth >= maxDepth {
			return nil, errTooDeep
		}
		s := *(*[]unsafe.Pointer)(p)
		n := len(s)
		dst = appendVarlong(dst, int64(n))
		if n == 0 {
			return dst, nil
		}
		// Per-element union node entered at depth+1; charge it.
		if depth+1 >= maxDepth {
			return nil, errTooDeep
		}
		fast := rec.fastFor(innerType)
		useFast := fast != nil && fast.allFast
		var err error
		for _, pp := range s {
			if pp == nil {
				dst = append(dst, nullByte)
				continue
			}
			dst = append(dst, valByte)
			if useFast {
				dst, err = serRecordFastPtr(dst, fast, pp, depth+2)
			} else {
				dst, err = rec.ser(dst, reflect.NewAt(innerType, pp).Elem(), depth+2)
			}
			if err != nil {
				return nil, err
			}
		}
		return append(dst, 0), nil
	}
}

// usArrayNullUnionPtr handles array ser for []*T where items are ["null", primitive].
//
// Like usArrayNullUnionRecord, the ["null", primitive] element type is a
// union schema node between the array and the primitive; it costs one
// depth unit. The array node is entered at depth, each element's union
// node at depth+1, the inner primitive at depth+2. The loop-invariant
// union guard is hoisted before the loop (mirrors decode's udNullUnionPtr,
// which guards before reading the index); the empty array escapes.
func usArrayNullUnionPtr(inner userfn, nullByte, valByte byte) userfn {
	return func(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
		if depth >= maxDepth {
			return nil, errTooDeep
		}
		s := *(*[]unsafe.Pointer)(p)
		n := len(s)
		dst = appendVarlong(dst, int64(n))
		if n == 0 {
			return dst, nil
		}
		// Per-element union node entered at depth+1; charge it.
		if depth+1 >= maxDepth {
			return nil, errTooDeep
		}
		var err error
		for _, pp := range s {
			if pp == nil {
				dst = append(dst, nullByte)
				continue
			}
			dst, err = inner(append(dst, valByte), pp, depth+2)
			if err != nil {
				return nil, err
			}
		}
		return append(dst, 0), nil
	}
}

// usArrayDirect handles array ser for []T where items are primitives.
func usArrayDirect(inner userfn, elemSize uintptr) userfn {
	return func(dst []byte, p unsafe.Pointer, depth int) ([]byte, error) {
		if depth >= maxDepth {
			return nil, errTooDeep
		}
		bs := *(*[]byte)(p)
		n := len(bs)
		dst = appendVarlong(dst, int64(n))
		if n == 0 {
			return dst, nil
		}
		bodyStart := len(dst)
		data := unsafe.Pointer(unsafe.SliceData(bs))
		var err error
		for i := range n {
			dst, err = inner(dst, unsafe.Add(data, uintptr(i)*elemSize), depth+1)
			if err != nil {
				return nil, err
			}
		}
		if err := arrayZeroByteEncodeCompliance(len(dst) == bodyStart, n); err != nil {
			return nil, err
		}
		return append(dst, 0), nil
	}
}

// udArrayBlocks drives the outer block loop for the unsafe array
// deserializers. onBlock fills slots [start, start+n) of v's backing
// array; src→advancedSrc is returned. Shared by udArrayPtrRecord and
// udArrayDirect so the depth bookkeeping, length overflow guard,
// MakeSlice/SetLen growth, and block-bounds check live in one place.
//
// minItemBytes bounds block count to len(src)/minItemBytes, mirroring
// deserArray.deser. Without it the loose count > len(src) check lets a hostile
// float-array stream drive a 4x MakeSlice allocation per wire byte before the
// per-element readUint32 loop fails short.
//
// Benchstat-verified perf-neutral on BenchmarkLargeArrayDecode.
func udArrayBlocks(
	src []byte, p unsafe.Pointer, sl *slab,
	sliceType reflect.Type, minItemBytes int,
	onBlock func(src []byte, v reflect.Value, p unsafe.Pointer, start, n int, sl *slab) ([]byte, error),
) ([]byte, error) {
	if sl.depth >= maxDepth {
		return nil, errTooDeep
	}
	sl.depth++
	defer func() { sl.depth-- }()
	v := reflect.NewAt(sliceType, p).Elem()
	v.SetLen(0)
	var totalItems int64
	for {
		count, _, rest, end, err := readBlockHeader(src, "array", false)
		if err != nil {
			return nil, err
		}
		src = rest
		if end {
			// Empty array → non-nil empty slice, matching deserArray.deser
			// (the reflect path), the JSON array decoder, and the binary map
			// decoder. Only the empty case reaches here with v still nil; a
			// populated target already has a backing array. Once per array.
			if v.IsNil() {
				v.Set(reflect.MakeSlice(sliceType, 0, 0))
			}
			return src, nil
		}
		if err := checkArrayBlockBounds(count, totalItems, len(src), minItemBytes); err != nil {
			return nil, err
		}
		totalItems += count
		n := int(count)
		start := v.Len()
		if start > math.MaxInt-n {
			return nil, fmt.Errorf("array length overflows int: start=%d count=%d", start, n)
		}
		newLen := start + n
		if v.Cap() < newLen {
			ns := reflect.MakeSlice(sliceType, newLen, newLen)
			reflect.Copy(ns, v)
			v.Set(ns)
		} else {
			v.SetLen(newLen)
		}
		if src, err = onBlock(src, v, p, start, n, sl); err != nil {
			return nil, err
		}
	}
}

// udArrayPtrRecord handles array deser for []*T where items are records.
// Uses reflect for slice management, unsafe for per-element record deser.
func udArrayPtrRecord(rec *deserRecord, innerType, sliceType reflect.Type, minItemBytes int) udeserfn {
	innerSize := innerType.Size()
	return func(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
		return udArrayBlocks(src, p, sl, sliceType, minItemBytes,
			func(src []byte, _ reflect.Value, p unsafe.Pointer, start, n int, sl *slab) ([]byte, error) {
				s := *(*[]unsafe.Pointer)(p)
				// Batch-allocate backing memory for nil pointer slots in
				// one contiguous slice, then distribute pointers into the
				// individual slots. This is much cheaper than allocating
				// each element separately.
				var need int
				for i := range n {
					if s[start+i] == nil {
						need++
					}
				}
				if need > 0 {
					backing := reflect.MakeSlice(reflect.SliceOf(innerType), need, need)
					backingBase := backing.Index(0).Addr().UnsafePointer()
					j := 0
					for i := range n {
						if s[start+i] == nil {
							s[start+i] = unsafe.Add(backingBase, uintptr(j)*innerSize)
							j++
						}
					}
				}
				fast := rec.fastFor(innerType)
				useFast := fast != nil && fast.allFast
				var err error
				for i := range n {
					elemP := s[start+i]
					if useFast {
						src, err = deserRecordFastPtr(src, fast, elemP, sl)
					} else {
						src, err = rec.deser(src, reflect.NewAt(innerType, elemP).Elem(), sl)
					}
					if err != nil {
						return nil, err
					}
				}
				return src, nil
			})
	}
}

// udArrayDirect handles array deser for []T where items are primitives.
// Uses reflect for slice management, unsafe for per-element deser.
func udArrayDirect(inner udeserfn, elemSize uintptr, sliceType reflect.Type, minItemBytes int) udeserfn {
	return func(src []byte, p unsafe.Pointer, sl *slab) ([]byte, error) {
		return udArrayBlocks(src, p, sl, sliceType, minItemBytes,
			func(src []byte, v reflect.Value, _ unsafe.Pointer, start, n int, sl *slab) ([]byte, error) {
				// Access data pointer after the slice growth in
				// udArrayBlocks. v.UnsafePointer returns the slice's
				// underlying-array pointer for any element type — avoids
				// the type-pun via *(*[]byte)(p).
				data := v.UnsafePointer()
				var err error
				for i := start; i < start+n; i++ {
					src, err = inner(src, unsafe.Add(data, uintptr(i)*elemSize), sl)
					if err != nil {
						return nil, err
					}
				}
				return src, nil
			})
	}
}
