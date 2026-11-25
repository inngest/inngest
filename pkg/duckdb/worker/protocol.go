package worker

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"time"
)

const (
	// maxFrameBytes caps a single RPC message frame at 64 MiB.
	maxFrameBytes = 64 << 20
)

// ColumnType mirrors DuckDB's DUCKDB_TYPE enum without importing purego or
// libduckdb, keeping pkg/duckdb/worker 100% pure Go (safe for static Alpine/musl
// binaries).
type ColumnType int32

const (
	TypeInvalid       ColumnType = 0
	TypeBoolean       ColumnType = 1
	TypeTinyInt       ColumnType = 2
	TypeSmallInt      ColumnType = 3
	TypeInteger       ColumnType = 4
	TypeBigInt        ColumnType = 5
	TypeUTinyInt      ColumnType = 6
	TypeUSmallInt     ColumnType = 7
	TypeUInteger      ColumnType = 8
	TypeUBigInt       ColumnType = 9
	TypeFloat         ColumnType = 10
	TypeDouble        ColumnType = 11
	TypeTimestamp     ColumnType = 12
	TypeDate          ColumnType = 13
	TypeTime          ColumnType = 14
	TypeInterval      ColumnType = 15
	TypeHugeInt       ColumnType = 16
	TypeVarchar       ColumnType = 17
	TypeBlob          ColumnType = 18
	TypeDecimal       ColumnType = 19
	TypeTimestampS    ColumnType = 20
	TypeTimestampMS   ColumnType = 21
	TypeTimestampNS   ColumnType = 22
	TypeUUID          ColumnType = 27
	TypeTimestampTZ   ColumnType = 31
	TypeUHugeInt      ColumnType = 32
	TypeSQLNull       ColumnType = 36
	TypeTimestampTZNS ColumnType = 42
)

// ColumnInfo describes a result column on the wire.
type ColumnInfo struct {
	Name string     `json:"name"`
	Type ColumnType `json:"type"`
}

// Op identifies a worker RPC operation.
type Op string

const (
	OpPing         Op = "ping"
	OpOpenSession  Op = "open_session"
	OpCloseSession Op = "close_session"
	OpExec         Op = "exec"
	OpQuery        Op = "query"
	OpInterrupt    Op = "interrupt"
	OpShutdown     Op = "shutdown"
)

// ResponseKind identifies a frame in a worker response stream.
type ResponseKind string

const (
	KindOK    ResponseKind = "ok"
	KindMeta  ResponseKind = "meta"
	KindChunk ResponseKind = "chunk"
	KindDone  ResponseKind = "done"
	KindError ResponseKind = "error"
)

// Request is a single client-to-worker RPC message.
type Request struct {
	Op        Op     `json:"op"`
	AuthToken string `json:"auth_token,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	RequestID uint64 `json:"request_id,omitempty"`
	SQL       string `json:"sql,omitempty"`
}

// Response is a single worker-to-client RPC message frame.
type Response struct {
	Kind      ResponseKind `json:"kind"`
	Version   string       `json:"version,omitempty"`
	PID       int          `json:"pid,omitempty"`
	SessionID string       `json:"session_id,omitempty"`
	Columns   []ColumnInfo `json:"columns,omitempty"`
	Rows      [][]any      `json:"rows,omitempty"`
	Error     string       `json:"error,omitempty"`
}

// WriteFrame writes a 4-byte length-prefixed JSON frame to w.
func WriteFrame(w io.Writer, v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("worker: marshal frame: %w", err)
	}
	if len(payload) > maxFrameBytes {
		return fmt.Errorf("worker: frame size %d exceeds limit %d", len(payload), maxFrameBytes)
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err = w.Write(payload)
	return err
}

// ReadFrame reads a 4-byte length-prefixed JSON frame from r into dst.
func ReadFrame(r io.Reader, dst any) error {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n == 0 || n > maxFrameBytes {
		return fmt.Errorf("worker: invalid frame length %d", n)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	if err := json.Unmarshal(buf, dst); err != nil {
		return fmt.Errorf("worker: unmarshal frame: %w", err)
	}
	return nil
}

// EncodeRows converts decoded Go values into JSON-safe lossless wire values
// using column type metadata so int64/uint64/time.Time/[]byte/NaN round-trip
// without precision loss.
func EncodeRows(cols []ColumnInfo, rows [][]any) [][]any {
	out := make([][]any, len(rows))
	for rIdx, row := range rows {
		enc := make([]any, len(row))
		for cIdx, val := range row {
			if val == nil {
				enc[cIdx] = nil
				continue
			}
			colType := TypeInvalid
			if cIdx < len(cols) {
				colType = cols[cIdx].Type
			}
			enc[cIdx] = encodeCell(colType, val)
		}
		out[rIdx] = enc
	}
	return out
}

func encodeCell(colType ColumnType, val any) any {
	switch colType {
	case TypeTinyInt:
		return strconv.FormatInt(int64(val.(int8)), 10)
	case TypeSmallInt:
		return strconv.FormatInt(int64(val.(int16)), 10)
	case TypeInteger:
		return strconv.FormatInt(int64(val.(int32)), 10)
	case TypeBigInt:
		return strconv.FormatInt(val.(int64), 10)
	case TypeUTinyInt:
		return strconv.FormatUint(uint64(val.(uint8)), 10)
	case TypeUSmallInt:
		return strconv.FormatUint(uint64(val.(uint16)), 10)
	case TypeUInteger:
		return strconv.FormatUint(uint64(val.(uint32)), 10)
	case TypeUBigInt:
		return strconv.FormatUint(val.(uint64), 10)
	case TypeFloat:
		f := float64(val.(float32))
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return strconv.FormatFloat(f, 'g', -1, 32)
		}
		return f
	case TypeDouble:
		f := val.(float64)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return strconv.FormatFloat(f, 'g', -1, 64)
		}
		return f
	case TypeBlob:
		return base64.StdEncoding.EncodeToString(val.([]byte))
	case TypeTimestamp, TypeTimestampTZ,
		TypeTimestampS, TypeTimestampMS,
		TypeTimestampNS, TypeTimestampTZNS,
		TypeDate:
		return val.(time.Time).UTC().Format(time.RFC3339Nano)
	default:
		return val
	}
}

// DecodeRows reconstructs typed Go values from wire JSON rows using column metadata.
func DecodeRows(cols []ColumnInfo, rows [][]any) ([][]any, error) {
	out := make([][]any, len(rows))
	for rIdx, row := range rows {
		dec := make([]any, len(row))
		for cIdx, val := range row {
			if val == nil {
				dec[cIdx] = nil
				continue
			}
			colType := TypeInvalid
			if cIdx < len(cols) {
				colType = cols[cIdx].Type
			}
			v, err := decodeCell(colType, val)
			if err != nil {
				return nil, fmt.Errorf("row %d col %d (type %d): %w", rIdx, cIdx, colType, err)
			}
			dec[cIdx] = v
		}
		out[rIdx] = dec
	}
	return out, nil
}

func decodeCell(colType ColumnType, val any) (any, error) {
	switch colType {
	case TypeBoolean:
		b, ok := val.(bool)
		if !ok {
			return nil, fmt.Errorf("expected bool, got %T", val)
		}
		return b, nil
	case TypeTinyInt:
		s, err := asString(val)
		if err != nil {
			return nil, err
		}
		n, err := strconv.ParseInt(s, 10, 8)
		return int8(n), err
	case TypeSmallInt:
		s, err := asString(val)
		if err != nil {
			return nil, err
		}
		n, err := strconv.ParseInt(s, 10, 16)
		return int16(n), err
	case TypeInteger:
		s, err := asString(val)
		if err != nil {
			return nil, err
		}
		n, err := strconv.ParseInt(s, 10, 32)
		return int32(n), err
	case TypeBigInt:
		s, err := asString(val)
		if err != nil {
			return nil, err
		}
		return strconv.ParseInt(s, 10, 64)
	case TypeUTinyInt:
		s, err := asString(val)
		if err != nil {
			return nil, err
		}
		n, err := strconv.ParseUint(s, 10, 8)
		return uint8(n), err
	case TypeUSmallInt:
		s, err := asString(val)
		if err != nil {
			return nil, err
		}
		n, err := strconv.ParseUint(s, 10, 16)
		return uint16(n), err
	case TypeUInteger:
		s, err := asString(val)
		if err != nil {
			return nil, err
		}
		n, err := strconv.ParseUint(s, 10, 32)
		return uint32(n), err
	case TypeUBigInt:
		s, err := asString(val)
		if err != nil {
			return nil, err
		}
		return strconv.ParseUint(s, 10, 64)
	case TypeFloat:
		switch v := val.(type) {
		case float64:
			return float32(v), nil
		case string:
			f, err := strconv.ParseFloat(v, 32)
			return float32(f), err
		default:
			return nil, fmt.Errorf("expected float, got %T", val)
		}
	case TypeDouble:
		switch v := val.(type) {
		case float64:
			return v, nil
		case string:
			return strconv.ParseFloat(v, 64)
		default:
			return nil, fmt.Errorf("expected double, got %T", val)
		}
	case TypeBlob:
		s, err := asString(val)
		if err != nil {
			return nil, err
		}
		return base64.StdEncoding.DecodeString(s)
	case TypeTimestamp, TypeTimestampTZ,
		TypeTimestampS, TypeTimestampMS,
		TypeTimestampNS, TypeTimestampTZNS,
		TypeDate:
		s, err := asString(val)
		if err != nil {
			return nil, err
		}
		return time.Parse(time.RFC3339Nano, s)
	default:
		return val, nil
	}
}

func asString(v any) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string, got %T", v)
	}
	return s, nil
}
