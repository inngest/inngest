package worker

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// StandardVectorSize is DuckDB's standard vector size (2048 rows per chunk).
const StandardVectorSize = 2048

// StreamColumnKind identifies one column's physical wire type for bulk
// streaming (QuackAppender, QuackMergeAppender, QuackStream).
type StreamColumnKind uint8

const (
	StreamColumnUUID        StreamColumnKind = 0
	StreamColumnVarchar     StreamColumnKind = 1
	StreamColumnJSON        StreamColumnKind = 2
	StreamColumnTimestampTZ StreamColumnKind = 3
	StreamColumnBool        StreamColumnKind = 4
	StreamColumnBigint      StreamColumnKind = 5
)

// StreamColumnType returns the DuckDB SQL type name for k.
func StreamColumnType(k StreamColumnKind) (string, error) {
	switch k {
	case StreamColumnUUID:
		return "UUID", nil
	case StreamColumnVarchar:
		return "VARCHAR", nil
	case StreamColumnJSON:
		return "JSON", nil
	case StreamColumnTimestampTZ:
		return "TIMESTAMPTZ", nil
	case StreamColumnBool:
		return "BOOLEAN", nil
	case StreamColumnBigint:
		return "BIGINT", nil
	default:
		return "", fmt.Errorf("worker: unsupported stream column kind %d", k)
	}
}

// Vector is one column of a batch in column-major form.
// Only the field matching Kind is read:
//   - StreamColumnUUID: UUIDs
//   - StreamColumnTimestampTZ: Int64s, as Unix milliseconds (encoded as TIMESTAMPTZ microseconds)
//   - StreamColumnBigint: Int64s
//   - StreamColumnBool: Bools
//   - StreamColumnVarchar, StreamColumnJSON: Bytes and Ends; row i's bytes are Bytes[Ends[i-1]:Ends[i]]
type Vector struct {
	Kind   StreamColumnKind
	Nulls  []bool
	UUIDs  []uuid.UUID
	Int64s []int64
	Bools  []bool
	Bytes  []byte
	Ends   []int
}

// Len returns the vector's row count.
func (v Vector) Len() int {
	switch v.Kind {
	case StreamColumnUUID:
		return len(v.UUIDs)
	case StreamColumnTimestampTZ, StreamColumnBigint:
		return len(v.Int64s)
	case StreamColumnBool:
		return len(v.Bools)
	default:
		return len(v.Ends)
	}
}

func (v Vector) isNull(i int) bool { return v.Nulls != nil && v.Nulls[i] }

func (v Vector) checkEnds() error {
	if v.Kind != StreamColumnVarchar && v.Kind != StreamColumnJSON {
		return nil
	}
	prev := 0
	for i, e := range v.Ends {
		if e < prev || e > len(v.Bytes) {
			return fmt.Errorf("row %d ends at %d, want %d..%d", i, e, prev, len(v.Bytes))
		}
		prev = e
	}
	return nil
}

// SendStream feeds one relation of a query directly from Go into the worker.
type SendStream struct {
	ID          string
	Columns     []StreamColumnKind
	Next        func(ctx context.Context) ([][]any, error)
	NextEncoded func(ctx context.Context) (blob []byte, chunks uint64, err error)
}

// MergeConfig describes a MERGE INTO statement driven by QuackMergeAppender.
type MergeConfig struct {
	On           string
	UpdateSet    string
	DedupKeys    []string
	DedupOrderBy string
}

// MergeColumn names one column of a QuackMergeAppender's source rows.
type MergeColumn struct {
	Name string
	Kind StreamColumnKind
}

// NewStreamID returns a fresh random stream identifier.
func NewStreamID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// Identifier double-quotes a DuckDB SQL identifier.
func Identifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func sqlStringLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// BuildInsertStreamSQL builds `INSERT INTO schema.table SELECT * FROM scan_data_from_quack_client(...)`.
func BuildInsertStreamSQL(schema, table string, cols []StreamColumnKind, streamID string) (string, error) {
	if len(cols) == 0 {
		return "", errors.New("duckdb: appender: no columns")
	}
	fields := make([]string, len(cols))
	for i, k := range cols {
		tn, err := StreamColumnType(k)
		if err != nil {
			return "", err
		}
		fields[i] = fmt.Sprintf("c%d %s", i, tn)
	}
	tableRef := Identifier(table)
	if schema != "" {
		tableRef = Identifier(schema) + "." + tableRef
	}
	return fmt.Sprintf("INSERT INTO %s SELECT * FROM scan_data_from_quack_client(%s, NULL::STRUCT(%s), ordered := true);",
		tableRef, sqlStringLiteral(streamID), strings.Join(fields, ", ")), nil
}

// BuildMergeStreamSQL builds the MERGE INTO statement using scan_data_from_quack_client.
func BuildMergeStreamSQL(schema, table string, cfg MergeConfig, columns []MergeColumn, streamID string) (string, error) {
	if len(columns) == 0 {
		return "", fmt.Errorf("duckdb: quack merge appender: no columns")
	}
	if cfg.On == "" {
		return "", fmt.Errorf("duckdb: quack merge appender: On condition is required")
	}
	if len(cfg.DedupKeys) > 0 && cfg.DedupOrderBy == "" {
		return "", fmt.Errorf("duckdb: quack merge appender: DedupOrderBy is required when DedupKeys is set")
	}

	fields := make([]string, len(columns))
	names := make([]string, len(columns))
	insertValues := make([]string, len(columns))
	knownColumns := make(map[string]bool, len(columns))
	for i, c := range columns {
		typeName, err := StreamColumnType(c.Kind)
		if err != nil {
			return "", err
		}
		ident := Identifier(c.Name)
		fields[i] = fmt.Sprintf("%s %s", ident, typeName)
		names[i] = ident
		insertValues[i] = "s." + ident
		knownColumns[c.Name] = true
	}

	dedupKeys := make([]string, len(cfg.DedupKeys))
	for i, k := range cfg.DedupKeys {
		if !knownColumns[k] {
			return "", fmt.Errorf("duckdb: quack merge appender: DedupKeys references unknown column %q", k)
		}
		dedupKeys[i] = Identifier(k)
	}

	tableRef := Identifier(table)
	if schema != "" {
		tableRef = Identifier(schema) + "." + tableRef
	}

	usingSelect := fmt.Sprintf("SELECT * FROM scan_data_from_quack_client(%s, NULL::STRUCT(%s), ordered := true)",
		sqlStringLiteral(streamID), strings.Join(fields, ", "))
	if len(dedupKeys) > 0 {
		usingSelect = fmt.Sprintf("%s\nQUALIFY ROW_NUMBER() OVER (PARTITION BY %s ORDER BY %s DESC) = 1",
			usingSelect, strings.Join(dedupKeys, ", "), cfg.DedupOrderBy)
	}

	var matchedClause string
	if cfg.UpdateSet != "" {
		matchedClause = fmt.Sprintf("WHEN MATCHED THEN UPDATE SET %s\n", cfg.UpdateSet)
	}

	return fmt.Sprintf(
		"MERGE INTO %s AS t USING (%s) AS s ON %s\n%sWHEN NOT MATCHED THEN INSERT (%s) VALUES (%s);",
		tableRef, usingSelect, cfg.On, matchedClause,
		strings.Join(names, ", "), strings.Join(insertValues, ", "),
	), nil
}

// Binary columnar wire format for stream batches:
//   [4 bytes] chunkCount (uint32 LE)
//   for each chunk:
//     [4 bytes] rowCount (uint32 LE)
//     [2 bytes] colCount (uint16 LE)
//     for each col:
//       [1 byte] kind (StreamColumnKind)
//       [1 byte] hasNulls (0 or 1)
//       if hasNulls == 1: [(rowCount+7)/8 bytes] null bitset (bit i == 1 means NULL)
//       payload:
//         UUID:        rowCount * 16 bytes (big-endian UUID bytes)
//         TimestampTZ: rowCount * 8 bytes (int64 LE microseconds)
//         Bigint:      rowCount * 8 bytes (int64 LE)
//         Bool:        rowCount bytes (0 or 1)
//         Varchar/JSON:
//           [4 bytes] totalBytes (uint32 LE)
//           [rowCount * 4 bytes] ends (uint32 LE)
//           [totalBytes bytes] raw string bytes

// EncodeVectors encodes equal-length vectors as binary chunks of at most StandardVectorSize rows.
func EncodeVectors(vectors []Vector) (blob []byte, chunkCount uint64, err error) {
	if len(vectors) == 0 {
		return nil, 0, fmt.Errorf("duckdb: quack vectors: no columns")
	}
	n := vectors[0].Len()
	for i, v := range vectors {
		if v.Len() != n {
			return nil, 0, fmt.Errorf("duckdb: quack vectors: column %d has %d rows, want %d", i, v.Len(), n)
		}
		if v.Nulls != nil && len(v.Nulls) != n {
			return nil, 0, fmt.Errorf("duckdb: quack vectors: column %d has %d null flags, want %d", i, len(v.Nulls), n)
		}
		if err := v.checkEnds(); err != nil {
			return nil, 0, fmt.Errorf("duckdb: quack vectors: column %d: %w", i, err)
		}
	}
	if n == 0 {
		return nil, 0, nil
	}

	numChunks := (n + StandardVectorSize - 1) / StandardVectorSize
	buf := make([]byte, 4, 64)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(numChunks))

	for start := 0; start < n; start += StandardVectorSize {
		end := min(start+StandardVectorSize, n)
		rowCount := end - start
		buf = binary.LittleEndian.AppendUint32(buf, uint32(rowCount))
		buf = binary.LittleEndian.AppendUint16(buf, uint16(len(vectors)))

		for i, v := range vectors {
			var encErr error
			buf, encErr = appendVectorSlice(buf, v, start, end)
			if encErr != nil {
				return nil, 0, fmt.Errorf("duckdb: quack vectors: column %d: %w", i, encErr)
			}
		}
		chunkCount++
	}
	return buf, chunkCount, nil
}

func appendVectorSlice(buf []byte, v Vector, start, end int) ([]byte, error) {
	n := end - start
	buf = append(buf, byte(v.Kind))

	hasNulls := false
	if v.Nulls != nil {
		for i := start; i < end; i++ {
			if v.Nulls[i] {
				hasNulls = true
				break
			}
		}
	}
	if hasNulls {
		buf = append(buf, 1)
		maskLen := (n + 7) / 8
		maskOff := len(buf)
		buf = append(buf, make([]byte, maskLen)...)
		for i := 0; i < n; i++ {
			if v.Nulls[start+i] {
				buf[maskOff+i/8] |= 1 << uint(i%8)
			}
		}
	} else {
		buf = append(buf, 0)
	}

	switch v.Kind {
	case StreamColumnUUID:
		for i := start; i < end; i++ {
			if v.isNull(i) {
				buf = append(buf, make([]byte, 16)...)
			} else {
				buf = append(buf, v.UUIDs[i][:]...)
			}
		}
	case StreamColumnTimestampTZ:
		for i := start; i < end; i++ {
			if v.isNull(i) {
				buf = binary.LittleEndian.AppendUint64(buf, 0)
			} else {
				buf = binary.LittleEndian.AppendUint64(buf, uint64(v.Int64s[i]*1000))
			}
		}
	case StreamColumnBigint:
		for i := start; i < end; i++ {
			if v.isNull(i) {
				buf = binary.LittleEndian.AppendUint64(buf, 0)
			} else {
				buf = binary.LittleEndian.AppendUint64(buf, uint64(v.Int64s[i]))
			}
		}
	case StreamColumnBool:
		for i := start; i < end; i++ {
			if !v.isNull(i) && v.Bools[i] {
				buf = append(buf, 1)
			} else {
				buf = append(buf, 0)
			}
		}
	case StreamColumnVarchar, StreamColumnJSON:
		from := 0
		if start > 0 {
			from = v.Ends[start-1]
		}
		var strData []byte
		ends := make([]uint32, n)
		prev := from
		for i := 0; i < n; i++ {
			e := v.Ends[start+i]
			if !v.isNull(start + i) {
				strData = append(strData, v.Bytes[prev:e]...)
			}
			ends[i] = uint32(len(strData))
			prev = e
		}
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(strData)))
		for _, e := range ends {
			buf = binary.LittleEndian.AppendUint32(buf, e)
		}
		buf = append(buf, strData...)
	default:
		return nil, fmt.Errorf("unsupported StreamColumnKind %d", v.Kind)
	}
	return buf, nil
}

type timeLike interface{ UnixMilli() int64 }

// EncodeRowsBatch converts row-oriented Go values into the binary columnar batch format.
func EncodeRowsBatch(cols []StreamColumnKind, rows [][]any) (blob []byte, chunks uint64, err error) {
	if len(rows) == 0 {
		return nil, 0, nil
	}
	n := len(rows)
	vecs := make([]Vector, len(cols))
	for cIdx, kind := range cols {
		v := Vector{Kind: kind}
		var nulls []bool
		switch kind {
		case StreamColumnUUID:
			v.UUIDs = make([]uuid.UUID, n)
			for rIdx, row := range rows {
				if row[cIdx] == nil {
					if nulls == nil {
						nulls = make([]bool, n)
					}
					nulls[rIdx] = true
					continue
				}
				switch val := row[cIdx].(type) {
				case string:
					u, err := uuid.Parse(val)
					if err != nil {
						return nil, 0, fmt.Errorf("col %d row %d: %w", cIdx, rIdx, err)
					}
					v.UUIDs[rIdx] = u
				case uuid.UUID:
					v.UUIDs[rIdx] = val
				default:
					return nil, 0, fmt.Errorf("col %d row %d: expected string or uuid.UUID, got %T", cIdx, rIdx, row[cIdx])
				}
			}
		case StreamColumnTimestampTZ:
			v.Int64s = make([]int64, n)
			for rIdx, row := range rows {
				if row[cIdx] == nil {
					if nulls == nil {
						nulls = make([]bool, n)
					}
					nulls[rIdx] = true
					continue
				}
				t, ok := row[cIdx].(timeLike)
				if !ok {
					return nil, 0, fmt.Errorf("col %d row %d: expected time.Time, got %T", cIdx, rIdx, row[cIdx])
				}
				v.Int64s[rIdx] = t.UnixMilli()
			}
		case StreamColumnBigint:
			v.Int64s = make([]int64, n)
			for rIdx, row := range rows {
				if row[cIdx] == nil {
					if nulls == nil {
						nulls = make([]bool, n)
					}
					nulls[rIdx] = true
					continue
				}
				val, err := anyToInt64(row[cIdx])
				if err != nil {
					return nil, 0, fmt.Errorf("col %d row %d: %w", cIdx, rIdx, err)
				}
				v.Int64s[rIdx] = val
			}
		case StreamColumnBool:
			v.Bools = make([]bool, n)
			for rIdx, row := range rows {
				if row[cIdx] == nil {
					if nulls == nil {
						nulls = make([]bool, n)
					}
					nulls[rIdx] = true
					continue
				}
				b, ok := row[cIdx].(bool)
				if !ok {
					return nil, 0, fmt.Errorf("col %d row %d: expected bool, got %T", cIdx, rIdx, row[cIdx])
				}
				v.Bools[rIdx] = b
			}
		case StreamColumnVarchar, StreamColumnJSON:
			v.Ends = make([]int, n)
			for rIdx, row := range rows {
				if row[cIdx] == nil {
					if nulls == nil {
						nulls = make([]bool, n)
					}
					nulls[rIdx] = true
					v.Ends[rIdx] = len(v.Bytes)
					continue
				}
				s, ok := row[cIdx].(string)
				if !ok {
					return nil, 0, fmt.Errorf("col %d row %d: expected string, got %T", cIdx, rIdx, row[cIdx])
				}
				v.Bytes = append(v.Bytes, s...)
				v.Ends[rIdx] = len(v.Bytes)
			}
		default:
			return nil, 0, fmt.Errorf("col %d: unsupported StreamColumnKind %d", cIdx, kind)
		}
		v.Nulls = nulls
		vecs[cIdx] = v
	}
	return EncodeVectors(vecs)
}

func anyToInt64(v any) (int64, error) {
	switch val := v.(type) {
	case int64:
		return val, nil
	case int:
		return int64(val), nil
	case int32:
		return int64(val), nil
	case int16:
		return int64(val), nil
	case int8:
		return int64(val), nil
	case uint32:
		return int64(val), nil
	case uint16:
		return int64(val), nil
	case uint8:
		return int64(val), nil
	default:
		return 0, fmt.Errorf("expected integer, got %T", v)
	}
}

// DecodedColumn is one column of a decoded stream chunk.
type DecodedColumn struct {
	Kind   StreamColumnKind
	Nulls  []bool
	UUIDs  [][16]byte
	Int64s []int64
	Bools  []bool
	Bytes  []byte
	Ends   []uint32
}

// DecodedChunk is one decoded chunk from a binary stream batch blob.
type DecodedChunk struct {
	RowCount int
	Columns  []DecodedColumn
}

// DecodeBatch decodes a binary stream batch blob into chunks.
func DecodeBatch(blob []byte) ([]DecodedChunk, error) {
	if len(blob) == 0 {
		return nil, nil
	}
	if len(blob) < 4 {
		return nil, errors.New("worker: truncated batch header")
	}
	chunkCount := int(binary.LittleEndian.Uint32(blob[0:4]))
	off := 4
	out := make([]DecodedChunk, chunkCount)

	for ci := 0; ci < chunkCount; ci++ {
		if off+6 > len(blob) {
			return nil, errors.New("worker: truncated chunk header")
		}
		rowCount := int(binary.LittleEndian.Uint32(blob[off : off+4]))
		colCount := int(binary.LittleEndian.Uint16(blob[off+4 : off+6]))
		off += 6

		cols := make([]DecodedColumn, colCount)
		for colIdx := 0; colIdx < colCount; colIdx++ {
			if off+2 > len(blob) {
				return nil, errors.New("worker: truncated column header")
			}
			kind := StreamColumnKind(blob[off])
			hasNulls := blob[off+1] != 0
			off += 2

			var nulls []bool
			if hasNulls {
				maskLen := (rowCount + 7) / 8
				if off+maskLen > len(blob) {
					return nil, errors.New("worker: truncated null mask")
				}
				nulls = make([]bool, rowCount)
				for i := 0; i < rowCount; i++ {
					if blob[off+i/8]&(1<<uint(i%8)) != 0 {
						nulls[i] = true
					}
				}
				off += maskLen
			}

			dc := DecodedColumn{Kind: kind, Nulls: nulls}
			switch kind {
			case StreamColumnUUID:
				need := rowCount * 16
				if off+need > len(blob) {
					return nil, errors.New("worker: truncated UUID column")
				}
				dc.UUIDs = make([][16]byte, rowCount)
				for i := 0; i < rowCount; i++ {
					copy(dc.UUIDs[i][:], blob[off+i*16:off+(i+1)*16])
				}
				off += need
			case StreamColumnTimestampTZ, StreamColumnBigint:
				need := rowCount * 8
				if off+need > len(blob) {
					return nil, errors.New("worker: truncated int64 column")
				}
				dc.Int64s = make([]int64, rowCount)
				for i := 0; i < rowCount; i++ {
					dc.Int64s[i] = int64(binary.LittleEndian.Uint64(blob[off+i*8 : off+(i+1)*8]))
				}
				off += need
			case StreamColumnBool:
				if off+rowCount > len(blob) {
					return nil, errors.New("worker: truncated bool column")
				}
				dc.Bools = make([]bool, rowCount)
				for i := 0; i < rowCount; i++ {
					dc.Bools[i] = blob[off+i] != 0
				}
				off += rowCount
			case StreamColumnVarchar, StreamColumnJSON:
				if off+4 > len(blob) {
					return nil, errors.New("worker: truncated varchar length")
				}
				totalBytes := int(binary.LittleEndian.Uint32(blob[off : off+4]))
				off += 4
				endsNeed := rowCount * 4
				if off+endsNeed+totalBytes > len(blob) {
					return nil, errors.New("worker: truncated varchar data")
				}
				dc.Ends = make([]uint32, rowCount)
				for i := 0; i < rowCount; i++ {
					dc.Ends[i] = binary.LittleEndian.Uint32(blob[off+i*4 : off+(i+1)*4])
				}
				off += endsNeed
				dc.Bytes = blob[off : off+totalBytes]
				off += totalBytes
			default:
				return nil, fmt.Errorf("worker: unknown stream column kind %d", kind)
			}
			cols[colIdx] = dc
		}
		out[ci] = DecodedChunk{RowCount: rowCount, Columns: cols}
	}
	return out, nil
}
