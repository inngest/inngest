package avro

import (
	"fmt"
	"reflect"
	"sync"
)

// skipfn advances past an encoded Avro value without storing it. It threads
// the slab through purely to share the recursion-depth bound with deserfn
// (sl.depth); skips otherwise don't need it.
type skipfn func(src []byte, sl *slab) ([]byte, error)

func skipNull(src []byte, _ *slab) ([]byte, error) {
	return src, nil
}

// needLen reports a ShortBufferError when src is too short to read n bytes.
// Single constructor of the fixed-size {Type + Need + Have} short-buffer
// shape so every fixed-length read/skip agrees on it (skip paths via
// skipBytesN; decode paths — deserFixed, deserDuration, deserFixedDecimal,
// deserFixedUUIDReflect, udDuration, readFixedUUID — call it directly).
func needLen(src []byte, n int, typ string) error {
	if len(src) < n {
		return &ShortBufferError{Type: typ, Need: n, Have: len(src)}
	}
	return nil
}

// skipBytesN advances past n bytes (boolean/float/double/fixed). One
// helper for every fixed-size skip so they agree on the ShortBufferError
// shape (Type + Need + Have).
func skipBytesN(src []byte, n int, typ string) ([]byte, error) {
	if err := needLen(src, n, typ); err != nil {
		return nil, err
	}
	return src[n:], nil
}

func skipBoolean(src []byte, _ *slab) ([]byte, error) { return skipBytesN(src, 1, "boolean") }
func skipFloat(src []byte, _ *slab) ([]byte, error)   { return skipBytesN(src, 4, "float") }
func skipDouble(src []byte, _ *slab) ([]byte, error)  { return skipBytesN(src, 8, "double") }

func skipInt(src []byte, _ *slab) ([]byte, error) {
	_, src, err := readVarint(src)
	return src, err
}

func skipLong(src []byte, _ *slab) ([]byte, error) {
	_, src, err := readVarlong(src)
	return src, err
}

func skipBytes(src []byte, _ *slab) ([]byte, error) {
	n, src, err := readLength(src, "bytes")
	if err != nil {
		return nil, err
	}
	return src[n:], nil
}

func skipFixed(size int) skipfn {
	return func(src []byte, _ *slab) ([]byte, error) { return skipBytesN(src, size, "fixed") }
}

type skipRecordFields struct {
	once   sync.Once
	fields []skipfn
	node   *schemaNode
	mbw    *minBytesWalk
}

func skipRecord(w *schemaNode, mbw *minBytesWalk) skipfn {
	s := &skipRecordFields{node: w, mbw: mbw}
	return func(src []byte, sl *slab) ([]byte, error) {
		if sl.depth >= maxDepth {
			return nil, errTooDeep
		}
		sl.depth++
		defer func() { sl.depth-- }()
		s.once.Do(func() {
			// The OPERATION's walk, threaded down from Resolve — not a fresh
			// one. A record is not a cost boundary: the schema chooses how many
			// records a dropped subtree references, each reference compiles its
			// own skipRecordFields, and a per-record walk therefore multiplied
			// the per-walk allowance by a count the schema picks. Reaching each
			// one costs O(1) wire bytes, so "the wire bounds how many compile"
			// bounds the COUNT and not the WORK.
			//
			// This runs at decode time and the walk is shared, so minBytesOf
			// locks; see minBytesWalk.mu for why that is uncontended and why a
			// nondeterministic drain order is safe here.
			s.fields = make([]skipfn, len(s.node.fields))
			for i := range s.node.fields {
				s.fields[i] = buildSkip(s.node.fields[i].node, s.mbw)
			}
		})
		var err error
		for _, f := range s.fields {
			if src, err = f(src, sl); err != nil {
				return nil, err
			}
		}
		return src, nil
	}
}

// skipBlocks iterates Avro block-framed data, invoking inner once per
// item/entry. blockType labels readBlockHeader errors ("array" / "map",
// the same labels the value-path block walkers pass, so the two paths
// report identical text for identical corruption — readBlockHeader's
// formats append their own "block"). When negative-count byte-size
// framing is present the
// block is fast-skipped via byteSize; otherwise totalGuard (when non-nil)
// gates the per-block count, and inner is called for each item/entry.
// Shared by skipArray and skipMap.
func skipBlocks(src []byte, sl *slab, blockType string,
	totalGuard func(count, total int64, srcLen int) error,
	inner func(src []byte, sl *slab) ([]byte, error),
) ([]byte, error) {
	if sl.depth >= maxDepth {
		return nil, errTooDeep
	}
	sl.depth++
	defer func() { sl.depth-- }()
	var total int64
	for {
		count, byteSize, rest, end, err := readBlockHeader(src, blockType, true)
		if err != nil {
			return nil, err
		}
		src = rest
		if end {
			return src, nil
		}
		if byteSize > 0 {
			// Negative-count framing: skip the whole block by byte-size.
			src = src[byteSize:]
			continue
		}
		if totalGuard != nil {
			if err := totalGuard(count, total, len(src)); err != nil {
				return nil, err
			}
			total += count
		}
		for range int(count) {
			if src, err = inner(src, sl); err != nil {
				return nil, err
			}
		}
	}
}

func skipArray(w *schemaNode, mbw *minBytesWalk) skipfn {
	itemSkip := buildSkip(w.items, mbw)
	minItemBytes := mbw.minBytesOf(w.items)
	return func(src []byte, sl *slab) ([]byte, error) {
		return skipBlocks(src, sl, "array",
			func(count, total int64, srcLen int) error {
				return checkArrayBlockBounds(count, total, srcLen, minItemBytes)
			},
			itemSkip)
	}
}

func skipMap(w *schemaNode, mbw *minBytesWalk) skipfn {
	valueSkip := buildSkip(w.values, mbw)
	// minEntryBytes = 1 (key length varint, ≥1 byte for an empty key) +
	// the value's minimum wire bytes — identical to deserMap.minEntryBytes.
	minEntryBytes := mapEntryMinBytes(mbw.minBytesOf(w.values))
	return func(src []byte, sl *slab) ([]byte, error) {
		return skipBlocks(src, sl, "map",
			// Bound the block count against the remaining buffer, matching
			// deserMap (deser.go) and skipArray's checkArrayBlockBounds.
			// Without it, skipBlocks' `for range int(count)` loop truncates
			// a count above 2^31 on a 32-bit build, mis-framing the skip of
			// subsequent bytes; on 64-bit the loop is buffer-bounded already,
			// but this keeps deserMap, skipArray, and skipMap on one rule.
			// minEntryBytes ≥ 1, so this is always the buffer-relative bound
			// and never false-rejects a legitimate block (each entry occupies
			// at least minEntryBytes wire bytes).
			func(count, _ int64, srcLen int) error {
				return checkMapBlockBounds(count, srcLen, minEntryBytes)
			},
			func(src []byte, sl *slab) ([]byte, error) {
				src, err := skipBytes(src, sl) // the entry key
				if err != nil {
					return nil, err
				}
				return valueSkip(src, sl)
			})
	}
}

func skipUnion(w *schemaNode, mbw *minBytesWalk) skipfn {
	branchSkips := make([]skipfn, len(w.branches))
	for i, br := range w.branches {
		branchSkips[i] = buildSkip(br, mbw)
	}
	return func(src []byte, sl *slab) ([]byte, error) {
		if sl.depth >= maxDepth {
			return nil, errTooDeep
		}
		sl.depth++
		defer func() { sl.depth-- }()
		idx, src, err := readVarint(src)
		if err != nil {
			return nil, err
		}
		if idx < 0 || int(idx) >= len(branchSkips) {
			return nil, fmt.Errorf("union index %d out of range [0, %d)", idx, len(branchSkips))
		}
		return branchSkips[idx](src, sl)
	}
}

var primitiveSkips = map[string]skipfn{
	"null":    skipNull,
	"boolean": skipBoolean,
	"int":     skipInt,
	"long":    skipLong,
	"float":   skipFloat,
	"double":  skipDouble,
	"bytes":   skipBytes,
	// A string is a length-prefixed byte run on the wire, so skipping one
	// IS skipping bytes — the shared reader already labels its
	// short-buffer error "bytes" for both.
	"string": skipBytes,
}

// buildSkip compiles a skipper for writer node w. mbw is the ONE min-bytes walk
// of the operation that reached here (resolveCtx.minBytes), and every arm passes
// it down — including the record arm, whose compile is deferred to decode time
// but still joins this walk rather than starting a fresh one. Nothing along the
// way is a cost boundary: containers, union branches and records are all counted
// by the schema, so any per-unit allowance is the per-walk bound multiplied by a
// number the schema author picks.
func buildSkip(w *schemaNode, mbw *minBytesWalk) skipfn {
	if f, ok := primitiveSkips[w.kind]; ok {
		return f
	}
	switch w.kind {
	case "record":
		return skipRecord(w, mbw)
	case "enum":
		return skipInt // an enum is its symbol index, encoded as an int
	case "array":
		return skipArray(w, mbw)
	case "map":
		return skipMap(w, mbw)
	case "union":
		return skipUnion(w, mbw)
	case "fixed":
		return skipFixed(w.size)
	default:
		return func(src []byte, _ *slab) ([]byte, error) {
			return nil, fmt.Errorf("cannot skip unknown type %q", w.kind)
		}
	}
}

// skipToDeser wraps a skipfn as a deserfn that ignores the reflect.Value.
func skipToDeser(skip skipfn) deserfn {
	return func(src []byte, _ reflect.Value, sl *slab) ([]byte, error) {
		return skip(src, sl)
	}
}
