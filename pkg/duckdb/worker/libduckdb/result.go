package libduckdb

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

// Result wraps a duckdb_result and streams decoded chunks.
type Result struct {
	lib    *Lib
	mu     sync.Mutex
	res    cResult
	cols   []ColumnInfo
	closed bool
}

// Columns returns metadata for each column in the result set.
func (r *Result) Columns() []ColumnInfo {
	out := make([]ColumnInfo, len(r.cols))
	copy(out, r.cols)
	return out
}

// Close destroys the underlying duckdb_result. Safe to call multiple times.
func (r *Result) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.closed = true
	r.lib.destroyResult(&r.res)
}

// NextChunk fetches and decodes the next data chunk from the result.
// It returns (nil, nil) when all chunks have been consumed.
func (r *Result) NextChunk() ([][]any, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, errors.New("libduckdb: result is closed")
	}

	chunk := r.lib.fetchChunk(r.res)
	if chunk == 0 {
		// Check whether NULL chunk indicates a streaming error vs normal EOF.
		if errPtr := r.lib.resultError(&r.res); errPtr != nil {
			if msg := cString(errPtr); msg != "" {
				return nil, fmt.Errorf("libduckdb: %s", msg)
			}
		}
		return nil, nil
	}
	defer r.lib.destroyDataChunk(&chunk)

	rowCount := r.lib.dataChunkGetSize(chunk)
	if rowCount == 0 {
		return nil, nil
	}
	colCount := uint64(len(r.cols))
	if cc := r.lib.dataChunkGetColumnCount(chunk); cc < colCount {
		colCount = cc
	}

	rows := make([][]any, rowCount)
	for i := uint64(0); i < rowCount; i++ {
		rows[i] = make([]any, colCount)
	}

	for colIdx := uint64(0); colIdx < colCount; colIdx++ {
		vec := r.lib.dataChunkGetVector(chunk, colIdx)
		lt := r.lib.vectorGetColumnType(vec)
		vals, err := decodeVector(r.lib, vec, lt, rowCount)
		if lt != 0 {
			r.lib.destroyLogicalType(&lt)
		}
		if err != nil {
			return nil, fmt.Errorf("libduckdb: column %q (type %s): %w",
				r.cols[colIdx].Name, r.cols[colIdx].TypeName, err)
		}
		for rowIdx := uint64(0); rowIdx < rowCount; rowIdx++ {
			rows[rowIdx][colIdx] = vals[rowIdx]
		}
	}

	return rows, nil
}

// formatLogicalType renders a duckdb_logical_type into its canonical DuckDB SQL
// type string (matching DESCRIBE / typeof()), e.g. "DECIMAL(4,1)", "INTEGER[3]",
// "MAP(VARCHAR, INTEGER)", "ENUM('sad', 'ok', 'happy')", "JSON", "VARIANT".
func formatLogicalType(lib *Lib, lt uintptr) string {
	if lt == 0 {
		return ""
	}
	if aliasPtr := lib.logicalTypeGetAlias(lt); aliasPtr != nil {
		alias := cString(aliasPtr)
		lib.free(unsafe.Pointer(aliasPtr))
		if alias == "JSON" {
			return "JSON"
		}
	}

	tid := Type(lib.getTypeID(lt))
	switch tid {
	case TypeDecimal:
		return fmt.Sprintf("DECIMAL(%d,%d)", lib.decimalWidth(lt), lib.decimalScale(lt))
	case TypeEnum:
		n := lib.enumDictionarySize(lt)
		parts := make([]string, n)
		for i := uint32(0); i < n; i++ {
			p := lib.enumDictionaryValue(lt, uint64(i))
			s := cString(p)
			if p != nil {
				lib.free(unsafe.Pointer(p))
			}
			parts[i] = "'" + strings.ReplaceAll(s, "'", "''") + "'"
		}
		return "ENUM(" + strings.Join(parts, ", ") + ")"
	case TypeList:
		child := lib.listTypeChildType(lt)
		if child == 0 {
			return "LIST"
		}
		s := formatLogicalType(lib, child)
		lib.destroyLogicalType(&child)
		return s + "[]"
	case TypeArray:
		size := lib.arrayTypeArraySize(lt)
		child := lib.arrayTypeChildType(lt)
		if child == 0 {
			return fmt.Sprintf("ARRAY[%d]", size)
		}
		// Match DuckDB's rendering for nested arrays: INTEGER[2][2] (inner size first).
		var dims []uint64
		dims = append(dims, size)
		cur := child
		for cur != 0 && Type(lib.getTypeID(cur)) == TypeArray {
			dims = append(dims, lib.arrayTypeArraySize(cur))
			next := lib.arrayTypeChildType(cur)
			lib.destroyLogicalType(&cur)
			cur = next
		}
		base := formatLogicalType(lib, cur)
		if cur != 0 {
			lib.destroyLogicalType(&cur)
		}
		var sb strings.Builder
		sb.WriteString(base)
		for i := len(dims) - 1; i >= 0; i-- {
			fmt.Fprintf(&sb, "[%d]", dims[i])
		}
		return sb.String()
	case TypeMap:
		kt := lib.mapTypeKeyType(lt)
		vt := lib.mapTypeValueType(lt)
		ks := formatLogicalType(lib, kt)
		vs := formatLogicalType(lib, vt)
		if kt != 0 {
			lib.destroyLogicalType(&kt)
		}
		if vt != 0 {
			lib.destroyLogicalType(&vt)
		}
		return fmt.Sprintf("MAP(%s, %s)", ks, vs)
	case TypeStruct:
		n := lib.structTypeChildCount(lt)
		parts := make([]string, n)
		for i := uint64(0); i < n; i++ {
			np := lib.structTypeChildName(lt, i)
			name := cString(np)
			if np != nil {
				lib.free(unsafe.Pointer(np))
			}
			ct := lib.structTypeChildType(lt, i)
			ts := formatLogicalType(lib, ct)
			if ct != 0 {
				lib.destroyLogicalType(&ct)
			}
			if name != "" {
				parts[i] = formatStructFieldName(name) + " " + ts
			} else {
				parts[i] = ts
			}
		}
		return "STRUCT(" + strings.Join(parts, ", ") + ")"
	case TypeUnion:
		n := lib.unionTypeMemberCount(lt)
		parts := make([]string, n)
		for i := uint64(0); i < n; i++ {
			np := lib.unionTypeMemberName(lt, i)
			name := cString(np)
			if np != nil {
				lib.free(unsafe.Pointer(np))
			}
			ct := lib.unionTypeMemberType(lt, i)
			ts := formatLogicalType(lib, ct)
			if ct != 0 {
				lib.destroyLogicalType(&ct)
			}
			parts[i] = formatStructFieldName(name) + " " + ts
		}
		return "UNION(" + strings.Join(parts, ", ") + ")"
	default:
		return tid.String()
	}
}

var sqlReservedKeywords = map[string]struct{}{
	"all": {}, "analyse": {}, "analyze": {}, "and": {}, "any": {}, "array": {}, "as": {},
	"asc": {}, "asymmetric": {}, "both": {}, "case": {}, "cast": {}, "check": {},
	"collate": {}, "column": {}, "constraint": {}, "create": {}, "current_catalog": {},
	"current_date": {}, "current_role": {}, "current_time": {}, "current_timestamp": {},
	"current_user": {}, "default": {}, "deferrable": {}, "desc": {}, "describe": {},
	"distinct": {}, "do": {}, "else": {}, "end": {}, "except": {}, "false": {}, "fetch": {},
	"for": {}, "foreign": {}, "from": {}, "grant": {}, "group": {}, "having": {}, "in": {},
	"initially": {}, "intersect": {}, "into": {}, "lateral": {}, "leading": {}, "limit": {},
	"localtime": {}, "localtimestamp": {}, "not": {}, "null": {}, "offset": {}, "on": {},
	"only": {}, "or": {}, "order": {}, "pivot": {}, "placing": {}, "primary": {},
	"qualify": {}, "references": {}, "returning": {}, "select": {}, "session_user": {},
	"show": {}, "some": {}, "summarize": {}, "symmetric": {}, "table": {}, "then": {},
	"to": {}, "trailing": {}, "true": {}, "union": {}, "unique": {}, "unpivot": {},
	"user": {}, "using": {}, "variadic": {}, "when": {}, "where": {}, "window": {}, "with": {},
}

func formatStructFieldName(name string) string {
	if name == "" {
		return `""`
	}
	if _, reserved := sqlReservedKeywords[strings.ToLower(name)]; reserved {
		return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		ok := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || (i > 0 && c >= '0' && c <= '9')
		if !ok {
			return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
		}
	}
	return name
}

// decodeVector decodes rowCount elements from vec according to its logical type lt.
func decodeVector(lib *Lib, vec uintptr, lt uintptr, rowCount uint64) ([]any, error) {
	out := make([]any, rowCount)
	if rowCount == 0 {
		return out, nil
	}
	if lt == 0 {
		return nil, errors.New("null logical type")
	}

	colType := Type(lib.getTypeID(lt))
	validity := lib.vectorGetValidity(vec)

	switch colType {
	case TypeSQLNull:
		return out, nil

	case TypeList:
		entriesPtr := lib.vectorGetData(vec)
		if entriesPtr == nil {
			return nil, errors.New("null list vector data pointer")
		}
		childVec := lib.listVectorGetChild(vec)
		childSize := lib.listVectorGetSize(vec)
		childLT := lib.listTypeChildType(lt)
		childVals, err := decodeVector(lib, childVec, childLT, childSize)
		if childLT != 0 {
			lib.destroyLogicalType(&childLT)
		}
		if err != nil {
			return nil, fmt.Errorf("list child: %w", err)
		}
		for i := uint64(0); i < rowCount; i++ {
			if !isValid(validity, i) {
				out[i] = nil
				continue
			}
			entry := *(*cListEntry)(unsafe.Add(entriesPtr, uintptr(i)*unsafe.Sizeof(cListEntry{})))
			end := entry.offset + entry.length
			if end > childSize {
				return nil, fmt.Errorf("list row %d slice [%d:%d] out of range (%d)", i, entry.offset, end, childSize)
			}
			slice := make([]any, entry.length)
			copy(slice, childVals[entry.offset:end])
			out[i] = slice
		}
		return out, nil

	case TypeMap:
		// Physically MAP is LIST<STRUCT(key K, value V)>.
		entriesPtr := lib.vectorGetData(vec)
		if entriesPtr == nil {
			return nil, errors.New("null map vector data pointer")
		}
		childVec := lib.listVectorGetChild(vec)
		childSize := lib.listVectorGetSize(vec)
		childLT := lib.vectorGetColumnType(childVec)
		childVals, err := decodeVector(lib, childVec, childLT, childSize)
		if childLT != 0 {
			lib.destroyLogicalType(&childLT)
		}
		if err != nil {
			return nil, fmt.Errorf("map child: %w", err)
		}
		for i := uint64(0); i < rowCount; i++ {
			if !isValid(validity, i) {
				out[i] = nil
				continue
			}
			entry := *(*cListEntry)(unsafe.Add(entriesPtr, uintptr(i)*unsafe.Sizeof(cListEntry{})))
			end := entry.offset + entry.length
			if end > childSize {
				return nil, fmt.Errorf("map row %d slice [%d:%d] out of range (%d)", i, entry.offset, end, childSize)
			}
			slice := make([]any, entry.length)
			copy(slice, childVals[entry.offset:end])
			out[i] = slice
		}
		return out, nil

	case TypeArray:
		arraySize := lib.arrayTypeArraySize(lt)
		childVec := lib.arrayVectorGetChild(vec)
		childLT := lib.arrayTypeChildType(lt)
		childVals, err := decodeVector(lib, childVec, childLT, rowCount*arraySize)
		if childLT != 0 {
			lib.destroyLogicalType(&childLT)
		}
		if err != nil {
			return nil, fmt.Errorf("array child: %w", err)
		}
		for i := uint64(0); i < rowCount; i++ {
			if !isValid(validity, i) {
				out[i] = nil
				continue
			}
			start := i * arraySize
			slice := make([]any, arraySize)
			copy(slice, childVals[start:start+arraySize])
			out[i] = slice
		}
		return out, nil

	case TypeStruct:
		childCount := lib.structTypeChildCount(lt)
		names := make([]string, childCount)
		childCols := make([][]any, childCount)
		for j := uint64(0); j < childCount; j++ {
			np := lib.structTypeChildName(lt, j)
			names[j] = cString(np)
			if np != nil {
				lib.free(unsafe.Pointer(np))
			}
			childLT := lib.structTypeChildType(lt, j)
			childVec := lib.structVectorGetChild(vec, j)
			vals, err := decodeVector(lib, childVec, childLT, rowCount)
			if childLT != 0 {
				lib.destroyLogicalType(&childLT)
			}
			if err != nil {
				return nil, fmt.Errorf("struct field %q: %w", names[j], err)
			}
			childCols[j] = vals
		}
		for i := uint64(0); i < rowCount; i++ {
			if !isValid(validity, i) {
				out[i] = nil
				continue
			}
			m := make(map[string]any, childCount)
			for j := uint64(0); j < childCount; j++ {
				m[names[j]] = childCols[j][i]
			}
			out[i] = m
		}
		return out, nil

	case TypeUnion:
		// Physically UNION is STRUCT(tag UTINYINT, member0, member1, ...).
		memberCount := lib.unionTypeMemberCount(lt)
		tagVec := lib.structVectorGetChild(vec, 0)
		tagLT := lib.vectorGetColumnType(tagVec)
		tags, err := decodeVector(lib, tagVec, tagLT, rowCount)
		if tagLT != 0 {
			lib.destroyLogicalType(&tagLT)
		}
		if err != nil {
			return nil, fmt.Errorf("union tag: %w", err)
		}
		names := make([]string, memberCount)
		memberCols := make([][]any, memberCount)
		for j := uint64(0); j < memberCount; j++ {
			np := lib.unionTypeMemberName(lt, j)
			names[j] = cString(np)
			if np != nil {
				lib.free(unsafe.Pointer(np))
			}
			memberLT := lib.unionTypeMemberType(lt, j)
			memberVec := lib.structVectorGetChild(vec, j+1)
			vals, err := decodeVector(lib, memberVec, memberLT, rowCount)
			if memberLT != 0 {
				lib.destroyLogicalType(&memberLT)
			}
			if err != nil {
				return nil, fmt.Errorf("union member %q: %w", names[j], err)
			}
			memberCols[j] = vals
		}
		for i := uint64(0); i < rowCount; i++ {
			if !isValid(validity, i) {
				out[i] = nil
				continue
			}
			tag := asInt64(tags[i])
			if tag < 0 || uint64(tag) >= memberCount {
				return nil, fmt.Errorf("union row %d tag %d out of range (%d)", i, tag, memberCount)
			}
			out[i] = map[string]any{names[tag]: memberCols[tag][i]}
		}
		return out, nil

	case TypeVariant:
		return decodeVariantVector(lib, vec, validity, rowCount)

	case TypeEnum:
		dataPtr := lib.vectorGetData(vec)
		if dataPtr == nil {
			return nil, errors.New("null enum vector data pointer")
		}
		internal := Type(lib.enumInternalType(lt))
		dictSize := lib.enumDictionarySize(lt)
		dict := make([]string, dictSize)
		for i := uint32(0); i < dictSize; i++ {
			p := lib.enumDictionaryValue(lt, uint64(i))
			dict[i] = cString(p)
			if p != nil {
				lib.free(unsafe.Pointer(p))
			}
		}
		for i := uint64(0); i < rowCount; i++ {
			if !isValid(validity, i) {
				out[i] = nil
				continue
			}
			var idx uint32
			switch internal {
			case TypeUTinyInt:
				idx = uint32(*(*uint8)(unsafe.Add(dataPtr, uintptr(i))))
			case TypeUSmallInt:
				idx = uint32(*(*uint16)(unsafe.Add(dataPtr, uintptr(i)*2)))
			case TypeUInteger:
				idx = *(*uint32)(unsafe.Add(dataPtr, uintptr(i)*4))
			default:
				return nil, fmt.Errorf("unsupported enum internal type %s", internal)
			}
			if idx >= dictSize {
				return nil, fmt.Errorf("enum row %d index %d out of range (%d)", i, idx, dictSize)
			}
			out[i] = dict[idx]
		}
		return out, nil
	}

	// Scalar vector types.
	dataPtr := lib.vectorGetData(vec)
	var decScale uint8
	var decInternal Type
	if colType == TypeDecimal {
		decScale = lib.decimalScale(lt)
		decInternal = Type(lib.decimalInternal(lt))
	}
	isJSON := false
	if colType == TypeVarchar {
		if aliasPtr := lib.logicalTypeGetAlias(lt); aliasPtr != nil {
			alias := cString(aliasPtr)
			lib.free(unsafe.Pointer(aliasPtr))
			isJSON = (alias == "JSON")
		}
	}

	for rowIdx := uint64(0); rowIdx < rowCount; rowIdx++ {
		if !isValid(validity, rowIdx) {
			out[rowIdx] = nil
			continue
		}
		val, err := decodeCell(colType, dataPtr, rowIdx, decScale, decInternal, isJSON)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowIdx, err)
		}
		out[rowIdx] = val
	}
	return out, nil
}

// isValid checks DuckDB's validity bitmask for rowIdx. A nil validity pointer
// means all rows in the vector are non-NULL.
func isValid(validity *uint64, rowIdx uint64) bool {
	if validity == nil {
		return true
	}
	word := *(*uint64)(unsafe.Add(unsafe.Pointer(validity), uintptr(rowIdx/64)*8))
	return (word & (uint64(1) << (rowIdx % 64))) != 0
}

func decodeCell(colType Type, data unsafe.Pointer, row uint64, decScale uint8, decInternal Type, isJSON bool) (any, error) {
	if colType == TypeSQLNull {
		return nil, nil
	}
	if data == nil {
		return nil, errors.New("null vector data pointer")
	}

	switch colType {
	case TypeBoolean:
		b := *(*byte)(unsafe.Add(data, uintptr(row)))
		return b != 0, nil
	case TypeTinyInt:
		return *(*int8)(unsafe.Add(data, uintptr(row))), nil
	case TypeSmallInt:
		return *(*int16)(unsafe.Add(data, uintptr(row)*2)), nil
	case TypeInteger:
		return *(*int32)(unsafe.Add(data, uintptr(row)*4)), nil
	case TypeBigInt:
		return *(*int64)(unsafe.Add(data, uintptr(row)*8)), nil
	case TypeUTinyInt:
		return *(*uint8)(unsafe.Add(data, uintptr(row))), nil
	case TypeUSmallInt:
		return *(*uint16)(unsafe.Add(data, uintptr(row)*2)), nil
	case TypeUInteger:
		return *(*uint32)(unsafe.Add(data, uintptr(row)*4)), nil
	case TypeUBigInt:
		return *(*uint64)(unsafe.Add(data, uintptr(row)*8)), nil
	case TypeFloat:
		bits := *(*uint32)(unsafe.Add(data, uintptr(row)*4))
		return math.Float32frombits(bits), nil
	case TypeDouble:
		bits := *(*uint64)(unsafe.Add(data, uintptr(row)*8))
		return math.Float64frombits(bits), nil
	case TypeVarchar:
		st := (*cStringT)(unsafe.Add(data, uintptr(row)*unsafe.Sizeof(cStringT{})))
		b := decodeStringT(st)
		if isJSON {
			var parsed any
			if err := json.Unmarshal(b, &parsed); err == nil {
				return parsed, nil
			}
		}
		return string(b), nil
	case TypeBlob:
		st := (*cStringT)(unsafe.Add(data, uintptr(row)*unsafe.Sizeof(cStringT{})))
		return decodeStringT(st), nil
	case TypeBit:
		st := (*cStringT)(unsafe.Add(data, uintptr(row)*unsafe.Sizeof(cStringT{})))
		return bitToString(decodeStringT(st)), nil
	case TypeUUID:
		h := *(*cHugeInt)(unsafe.Add(data, uintptr(row)*unsafe.Sizeof(cHugeInt{})))
		return decodeUUID(h), nil
	case TypeTimestamp, TypeTimestampTZ:
		us := *(*int64)(unsafe.Add(data, uintptr(row)*8))
		return decodeTimestampMicros(us), nil
	case TypeTimestampS:
		s := *(*int64)(unsafe.Add(data, uintptr(row)*8))
		return time.Unix(s, 0).UTC(), nil
	case TypeTimestampMS:
		ms := *(*int64)(unsafe.Add(data, uintptr(row)*8))
		return time.UnixMilli(ms).UTC(), nil
	case TypeTimestampNS, TypeTimestampTZNS:
		ns := *(*int64)(unsafe.Add(data, uintptr(row)*8))
		return time.Unix(0, ns).UTC(), nil
	case TypeDate:
		days := *(*int32)(unsafe.Add(data, uintptr(row)*4))
		return decodeDateDays(days), nil
	case TypeTime:
		us := *(*int64)(unsafe.Add(data, uintptr(row)*8))
		return timeToTime(us), nil
	case TypeTimeTZ:
		bits := *(*uint64)(unsafe.Add(data, uintptr(row)*8))
		return timeTZToTime(bits), nil
	case TypeTimeNS:
		ns := *(*int64)(unsafe.Add(data, uintptr(row)*8))
		return time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(ns)), nil
	case TypeInterval:
		iv := *(*cInterval)(unsafe.Add(data, uintptr(row)*unsafe.Sizeof(cInterval{})))
		return intervalToString(iv.months, iv.days, iv.micros), nil
	case TypeHugeInt:
		h := *(*cHugeInt)(unsafe.Add(data, uintptr(row)*unsafe.Sizeof(cHugeInt{})))
		return decodeHugeInt(h).String(), nil
	case TypeUHugeInt:
		uh := *(*cUHugeInt)(unsafe.Add(data, uintptr(row)*unsafe.Sizeof(cUHugeInt{})))
		return decodeUHugeInt(uh).String(), nil
	case TypeDecimal:
		return decodeDecimal(data, row, decScale, decInternal)
	default:
		return nil, fmt.Errorf("unsupported DuckDB type %s", colType)
	}
}

// decodeStringT copies a 16-byte duckdb_string_t into a freshly allocated Go
// byte slice so it does not alias C chunk memory.
func decodeStringT(st *cStringT) []byte {
	n := int(st.length)
	if n == 0 {
		return []byte{}
	}
	if n <= 12 {
		out := make([]byte, n)
		copy(out, st.data[:n])
		return out
	}
	// Pointer layout: offset 0..3 = length, offset 4..7 = prefix, offset 8..15 = char* ptr.
	ptr := *(*unsafe.Pointer)(unsafe.Pointer(&st.data[4]))
	if ptr == nil {
		return []byte{}
	}
	src := unsafe.Slice((*byte)(ptr), n)
	out := make([]byte, n)
	copy(out, src)
	return out
}

// decodeUUID converts a DuckDB UUID (stored as duckdb_hugeint with the MSB of
// upper flipped) into its canonical lowercase hyphenated string form.
func decodeUUID(h cHugeInt) string {
	hi := uint64(h.upper) ^ (uint64(1) << 63)
	lo := h.lower

	var raw [16]byte
	binary.BigEndian.PutUint64(raw[0:8], hi)
	binary.BigEndian.PutUint64(raw[8:16], lo)

	var dst [36]byte
	hex.Encode(dst[0:8], raw[0:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], raw[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], raw[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], raw[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:36], raw[10:16])
	return string(dst[:])
}

func decodeTimestampMicros(us int64) time.Time {
	return time.UnixMicro(us).UTC()
}

func decodeDateDays(days int32) time.Time {
	return time.Unix(int64(days)*86400, 0).UTC()
}

func timeToTime(micros int64) time.Time {
	return time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(micros) * time.Microsecond)
}

const (
	timeTZOffsetBits = 24
	timeTZMaxOffset  = 16*60*60 - 1
)

func timeTZToTime(bits uint64) time.Time {
	offsetMask := uint64(1)<<timeTZOffsetBits - 1
	micros := int64(bits >> timeTZOffsetBits)
	offsetSeconds := timeTZMaxOffset - int(bits&offsetMask)
	sign := "+"
	absOff := offsetSeconds
	if absOff < 0 {
		sign = "-"
		absOff = -absOff
	}
	loc := time.FixedZone(fmt.Sprintf("UTC%s%02d:%02d", sign, absOff/3600, (absOff%3600)/60), offsetSeconds)
	return time.Date(0, 1, 1, 0, 0, 0, 0, loc).Add(time.Duration(micros) * time.Microsecond)
}

func decodeHugeInt(h cHugeInt) *big.Int {
	res := big.NewInt(h.upper)
	res.Lsh(res, 64)
	lo := new(big.Int).SetUint64(h.lower)
	res.Add(res, lo)
	return res
}

func decodeUHugeInt(h cUHugeInt) *big.Int {
	res := new(big.Int).SetUint64(h.upper)
	res.Lsh(res, 64)
	lo := new(big.Int).SetUint64(h.lower)
	res.Add(res, lo)
	return res
}

func decodeDecimal(data unsafe.Pointer, row uint64, scale uint8, internal Type) (string, error) {
	var unscaled *big.Int
	switch internal {
	case TypeSmallInt:
		unscaled = big.NewInt(int64(*(*int16)(unsafe.Add(data, uintptr(row)*2))))
	case TypeInteger:
		unscaled = big.NewInt(int64(*(*int32)(unsafe.Add(data, uintptr(row)*4))))
	case TypeBigInt:
		unscaled = big.NewInt(*(*int64)(unsafe.Add(data, uintptr(row)*8)))
	case TypeHugeInt:
		h := *(*cHugeInt)(unsafe.Add(data, uintptr(row)*unsafe.Sizeof(cHugeInt{})))
		unscaled = decodeHugeInt(h)
	default:
		return "", fmt.Errorf("unsupported decimal internal type %s", internal)
	}
	return formatDecimal(unscaled, scale), nil
}

func formatDecimal(unscaled *big.Int, scale uint8) string {
	if scale == 0 {
		return unscaled.String()
	}
	neg := unscaled.Sign() < 0
	s := new(big.Int).Abs(unscaled).String()
	sc := int(scale)
	if len(s) <= sc {
		s = strings.Repeat("0", sc-len(s)+1) + s
	}
	split := len(s) - sc
	out := s[:split] + "." + s[split:]
	if neg {
		return "-" + out
	}
	return out
}

func intervalToString(months, days int32, micros int64) string {
	var parts []string
	if months != 0 {
		years, remMonths := months/12, months%12
		if years != 0 {
			parts = append(parts, fmt.Sprintf("%d %s", years, intervalUnit("year", years)))
		}
		if remMonths != 0 {
			parts = append(parts, fmt.Sprintf("%d %s", remMonths, intervalUnit("month", remMonths)))
		}
	}
	if days != 0 {
		parts = append(parts, fmt.Sprintf("%d %s", days, intervalUnit("day", days)))
	}
	if micros != 0 || len(parts) == 0 {
		parts = append(parts, intervalTimeToString(micros))
	}
	return strings.Join(parts, " ")
}

func intervalUnit(unit string, n int32) string {
	if n == 1 || n == -1 {
		return unit
	}
	return unit + "s"
}

func intervalTimeToString(micros int64) string {
	sign := ""
	if micros < 0 {
		sign = "-"
		micros = -micros
	}
	totalSeconds := micros / 1_000_000
	fracMicros := micros % 1_000_000
	hours, minutes, seconds := totalSeconds/3600, (totalSeconds%3600)/60, totalSeconds%60
	s := fmt.Sprintf("%s%02d:%02d:%02d", sign, hours, minutes, seconds)
	if fracMicros != 0 {
		s += "." + strings.TrimRight(fmt.Sprintf("%06d", fracMicros), "0")
	}
	return s
}

func bitToString(b []byte) string {
	if len(b) < 2 {
		return ""
	}
	padding := int(b[0])
	var sb strings.Builder
	sb.Grow((8 - padding) + 8*(len(b)-2))
	for bitIdx := padding; bitIdx < 8; bitIdx++ {
		if b[1]&(1<<uint(7-bitIdx)) != 0 {
			sb.WriteByte('1')
		} else {
			sb.WriteByte('0')
		}
	}
	for byteIdx := 2; byteIdx < len(b); byteIdx++ {
		for bitIdx := 0; bitIdx < 8; bitIdx++ {
			if b[byteIdx]&(1<<uint(7-bitIdx)) != 0 {
				sb.WriteByte('1')
			} else {
				sb.WriteByte('0')
			}
		}
	}
	return sb.String()
}

// VARIANT unshredding (mirrors DuckDB's VariantLogicalType / VariantVisitor).
const (
	variantTypeNull              byte = 0
	variantTypeBoolTrue          byte = 1
	variantTypeBoolFalse         byte = 2
	variantTypeInt8              byte = 3
	variantTypeInt16             byte = 4
	variantTypeInt32             byte = 5
	variantTypeInt64             byte = 6
	variantTypeInt128            byte = 7
	variantTypeUint8             byte = 8
	variantTypeUint16            byte = 9
	variantTypeUint32            byte = 10
	variantTypeUint64            byte = 11
	variantTypeUint128           byte = 12
	variantTypeFloat             byte = 13
	variantTypeDouble            byte = 14
	variantTypeDecimal           byte = 15
	variantTypeVarchar           byte = 16
	variantTypeBlob              byte = 17
	variantTypeUUID              byte = 18
	variantTypeDate              byte = 19
	variantTypeTimeMicros        byte = 20
	variantTypeTimestampSec      byte = 22
	variantTypeTimestampMillis   byte = 23
	variantTypeTimestampMicros   byte = 24
	variantTypeTimestampNanos    byte = 25
	variantTypeTimeMicrosTZ      byte = 26
	variantTypeTimestampMicrosTZ byte = 27
	variantTypeInterval          byte = 28
	variantTypeObject            byte = 29
	variantTypeArray             byte = 30
)

func decodeVariantVector(lib *Lib, vec uintptr, validity *uint64, rowCount uint64) ([]any, error) {
	// VARIANT is physically STRUCT(keys, children, values, data) with 4 fixed fields.
	var childCols [4][]any
	for j := uint64(0); j < 4; j++ {
		cv := lib.structVectorGetChild(vec, j)
		clt := lib.vectorGetColumnType(cv)
		vals, err := decodeVector(lib, cv, clt, rowCount)
		if clt != 0 {
			lib.destroyLogicalType(&clt)
		}
		if err != nil {
			return nil, fmt.Errorf("variant field %d: %w", j, err)
		}
		childCols[j] = vals
	}

	out := make([]any, rowCount)
	for i := uint64(0); i < rowCount; i++ {
		if !isValid(validity, i) {
			out[i] = nil
			continue
		}
		keys, _ := childCols[0][i].([]any)
		children, _ := childCols[1][i].([]any)
		values, _ := childCols[2][i].([]any)
		data, _ := childCols[3][i].([]byte)
		if len(values) == 0 {
			out[i] = nil
			continue
		}
		v, err := decodeVariantValue(keys, children, values, data, 0)
		if err != nil {
			return nil, fmt.Errorf("variant row %d: %w", i, err)
		}
		out[i] = v
	}
	return out, nil
}

func decodeVariantValue(keys, children, values []any, data []byte, valueIndex int) (any, error) {
	if valueIndex < 0 || valueIndex >= len(values) {
		return nil, fmt.Errorf("value index %d out of range (%d values)", valueIndex, len(values))
	}
	entry, ok := values[valueIndex].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("malformed values[%d] entry", valueIndex)
	}
	typeID := asInt64(entry["type_id"])
	byteOffset := asInt64(entry["byte_offset"])
	if byteOffset < 0 || byteOffset > int64(len(data)) {
		return nil, fmt.Errorf("byte_offset %d beyond %d bytes of data", byteOffset, len(data))
	}
	ptr := data[byteOffset:]

	switch byte(typeID) {
	case variantTypeNull:
		return nil, nil
	case variantTypeBoolTrue:
		return true, nil
	case variantTypeBoolFalse:
		return false, nil
	case variantTypeInt8:
		return int64(int8(ptr[0])), nil
	case variantTypeInt16:
		return int64(int16(binary.LittleEndian.Uint16(ptr))), nil
	case variantTypeInt32:
		return int64(int32(binary.LittleEndian.Uint32(ptr))), nil
	case variantTypeInt64:
		return int64(binary.LittleEndian.Uint64(ptr)), nil
	case variantTypeInt128:
		h := cHugeInt{lower: binary.LittleEndian.Uint64(ptr[0:8]), upper: int64(binary.LittleEndian.Uint64(ptr[8:16]))}
		return decodeHugeInt(h).String(), nil
	case variantTypeUint8:
		return int64(ptr[0]), nil
	case variantTypeUint16:
		return int64(binary.LittleEndian.Uint16(ptr)), nil
	case variantTypeUint32:
		return int64(binary.LittleEndian.Uint32(ptr)), nil
	case variantTypeUint64:
		u := binary.LittleEndian.Uint64(ptr)
		if u <= math.MaxInt64 {
			return int64(u), nil
		}
		return strconv.FormatUint(u, 10), nil
	case variantTypeUint128:
		uh := cUHugeInt{lower: binary.LittleEndian.Uint64(ptr[0:8]), upper: binary.LittleEndian.Uint64(ptr[8:16])}
		return decodeUHugeInt(uh).String(), nil
	case variantTypeFloat:
		return float64(math.Float32frombits(binary.LittleEndian.Uint32(ptr))), nil
	case variantTypeDouble:
		return math.Float64frombits(binary.LittleEndian.Uint64(ptr)), nil
	case variantTypeDecimal:
		width, n1, err := readVarint(ptr, 0)
		if err != nil {
			return nil, fmt.Errorf("decimal width: %w", err)
		}
		scale, n2, err := readVarint(ptr, n1)
		if err != nil {
			return nil, fmt.Errorf("decimal scale: %w", err)
		}
		valBytes := ptr[n2:]
		var v *big.Int
		switch {
		case width <= 4:
			v = big.NewInt(int64(int16(binary.LittleEndian.Uint16(valBytes))))
		case width <= 9:
			v = big.NewInt(int64(int32(binary.LittleEndian.Uint32(valBytes))))
		case width <= 18:
			v = big.NewInt(int64(binary.LittleEndian.Uint64(valBytes)))
		default:
			h := cHugeInt{lower: binary.LittleEndian.Uint64(valBytes[0:8]), upper: int64(binary.LittleEndian.Uint64(valBytes[8:16]))}
			v = decodeHugeInt(h)
		}
		return formatDecimal(v, uint8(scale)), nil
	case variantTypeVarchar:
		length, n, err := readVarint(ptr, 0)
		if err != nil {
			return nil, fmt.Errorf("varchar length: %w", err)
		}
		if length > uint64(len(ptr)-n) {
			return nil, fmt.Errorf("varchar length %d exceeds available data", length)
		}
		return string(ptr[n : n+int(length)]), nil
	case variantTypeBlob:
		length, n, err := readVarint(ptr, 0)
		if err != nil {
			return nil, fmt.Errorf("blob length: %w", err)
		}
		if length > uint64(len(ptr)-n) {
			return nil, fmt.Errorf("blob length %d exceeds available data", length)
		}
		return append([]byte(nil), ptr[n:n+int(length)]...), nil
	case variantTypeUUID:
		h := cHugeInt{lower: binary.LittleEndian.Uint64(ptr[0:8]), upper: int64(binary.LittleEndian.Uint64(ptr[8:16]))}
		return decodeUUID(h), nil
	case variantTypeDate:
		return decodeDateDays(int32(binary.LittleEndian.Uint32(ptr))), nil
	case variantTypeTimeMicros:
		return timeToTime(int64(binary.LittleEndian.Uint64(ptr))), nil
	case variantTypeTimeMicrosTZ:
		return timeTZToTime(binary.LittleEndian.Uint64(ptr)), nil
	case variantTypeTimestampSec:
		return time.Unix(int64(binary.LittleEndian.Uint64(ptr)), 0).UTC(), nil
	case variantTypeTimestampMillis:
		return time.UnixMilli(int64(binary.LittleEndian.Uint64(ptr))).UTC(), nil
	case variantTypeTimestampMicros, variantTypeTimestampMicrosTZ:
		return decodeTimestampMicros(int64(binary.LittleEndian.Uint64(ptr))), nil
	case variantTypeTimestampNanos:
		return time.Unix(0, int64(binary.LittleEndian.Uint64(ptr))).UTC(), nil
	case variantTypeInterval:
		months := int32(binary.LittleEndian.Uint32(ptr[0:4]))
		days := int32(binary.LittleEndian.Uint32(ptr[4:8]))
		micros := int64(binary.LittleEndian.Uint64(ptr[8:16]))
		return intervalToString(months, days, micros), nil
	case variantTypeObject:
		childCount, childrenIdx, err := decodeVariantNestedData(ptr)
		if err != nil {
			return nil, err
		}
		m := make(map[string]any, childCount)
		for i := 0; i < childCount; i++ {
			child, ok := children[childrenIdx+i].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("malformed children[%d] entry", childrenIdx+i)
			}
			keysIdx := asInt64(child["keys_index"])
			valuesIdx := asInt64(child["values_index"])
			if keysIdx < 0 || int(keysIdx) >= len(keys) {
				return nil, fmt.Errorf("keys index %d out of range (%d keys)", keysIdx, len(keys))
			}
			key, _ := keys[keysIdx].(string)
			val, err := decodeVariantValue(keys, children, values, data, int(valuesIdx))
			if err != nil {
				return nil, err
			}
			m[key] = val
		}
		return m, nil
	case variantTypeArray:
		childCount, childrenIdx, err := decodeVariantNestedData(ptr)
		if err != nil {
			return nil, err
		}
		a := make([]any, childCount)
		for i := 0; i < childCount; i++ {
			child, ok := children[childrenIdx+i].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("malformed children[%d] entry", childrenIdx+i)
			}
			valuesIdx := asInt64(child["values_index"])
			val, err := decodeVariantValue(keys, children, values, data, int(valuesIdx))
			if err != nil {
				return nil, err
			}
			a[i] = val
		}
		return a, nil
	default:
		return nil, fmt.Errorf("unsupported VARIANT value type %d", typeID)
	}
}

func decodeVariantNestedData(ptr []byte) (childCount int, childrenIdx int, err error) {
	count, n, err := readVarint(ptr, 0)
	if err != nil {
		return 0, 0, fmt.Errorf("nested child_count: %w", err)
	}
	if count == 0 {
		return 0, 0, nil
	}
	idx, _, err := readVarint(ptr, n)
	if err != nil {
		return 0, 0, fmt.Errorf("nested children_idx: %w", err)
	}
	return int(count), int(idx), nil
}

func readVarint(b []byte, offset int) (value uint64, newOffset int, err error) {
	shift := 0
	for {
		if offset >= len(b) {
			return 0, 0, fmt.Errorf("truncated varint")
		}
		if shift >= 64 {
			return 0, 0, fmt.Errorf("varint exceeds 64 bits")
		}
		byteVal := b[offset]
		offset++
		value |= uint64(byteVal&0x7F) << shift
		if byteVal&0x80 == 0 {
			return value, offset, nil
		}
		shift += 7
	}
}

func asInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int32:
		return int64(n)
	case int16:
		return int64(n)
	case int8:
		return int64(n)
	case uint64:
		return int64(n)
	case uint32:
		return int64(n)
	case uint16:
		return int64(n)
	case uint8:
		return int64(n)
	default:
		return 0
	}
}

// cString converts a null-terminated C string pointer into a Go string without cgo.
func cString(p *byte) string {
	if p == nil {
		return ""
	}
	n := 0
	for *(*byte)(unsafe.Add(unsafe.Pointer(p), uintptr(n))) != 0 {
		n++
	}
	if n == 0 {
		return ""
	}
	return string(unsafe.Slice(p, n))
}
