package libduckdb

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
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
		dataPtr := r.lib.vectorGetData(vec)
		validity := r.lib.vectorGetValidity(vec)
		colType := r.cols[colIdx].Type

		var decScale uint8
		var decInternal Type
		if colType == TypeDecimal {
			lt := r.lib.vectorGetColumnType(vec)
			if lt != 0 {
				decScale = r.lib.decimalScale(lt)
				decInternal = Type(r.lib.decimalInternal(lt))
				r.lib.destroyLogicalType(&lt)
			}
		}

		for rowIdx := uint64(0); rowIdx < rowCount; rowIdx++ {
			if !isValid(validity, rowIdx) {
				rows[rowIdx][colIdx] = nil
				continue
			}
			val, err := decodeCell(colType, dataPtr, rowIdx, decScale, decInternal)
			if err != nil {
				return nil, fmt.Errorf("libduckdb: column %q (type %s) row %d: %w",
					r.cols[colIdx].Name, colType, rowIdx, err)
			}
			rows[rowIdx][colIdx] = val
		}
	}

	return rows, nil
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

func decodeCell(colType Type, data unsafe.Pointer, row uint64, decScale uint8, decInternal Type) (any, error) {
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
		return string(decodeStringT(st)), nil
	case TypeBlob:
		st := (*cStringT)(unsafe.Add(data, uintptr(row)*unsafe.Sizeof(cStringT{})))
		return decodeStringT(st), nil
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
	case TypeHugeInt:
		h := *(*cHugeInt)(unsafe.Add(data, uintptr(row)*unsafe.Sizeof(cHugeInt{})))
		return decodeHugeInt(h).String(), nil
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

func decodeHugeInt(h cHugeInt) *big.Int {
	// Value = upper * 2^64 + lower (where upper is signed int64, lower is unsigned uint64).
	res := big.NewInt(h.upper)
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
