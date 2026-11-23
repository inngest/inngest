//go:build darwin || linux || windows

package libduckdb

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ebitengine/purego"
)

// StreamColumnKind identifies a column's physical type inside a StreamBatch.
type StreamColumnKind uint8

const (
	StreamColumnUUID        StreamColumnKind = 0
	StreamColumnVarchar     StreamColumnKind = 1
	StreamColumnJSON        StreamColumnKind = 2
	StreamColumnTimestampTZ StreamColumnKind = 3
	StreamColumnBool        StreamColumnKind = 4
	StreamColumnBigint      StreamColumnKind = 5
)

// StreamColumn holds one column of a single StreamChunk (<= 2048 rows) in
// column-major form, ready to be copied directly into a DuckDB output vector.
type StreamColumn struct {
	Kind   StreamColumnKind
	Nulls  []bool
	UUIDs  [][16]byte // big-endian 16-byte UUIDs
	Int64s []int64    // BIGINT values or TIMESTAMPTZ microseconds
	Bools  []bool
	Bytes  []byte
	Ends   []uint32
}

// StreamChunk is one DuckDB-sized chunk (<= 2048 rows) pushed into a Stream.
type StreamChunk struct {
	RowCount int
	Columns  []StreamColumn
	Err      error
}

// DefaultMaxStreamQueueBytes bounds how much decoded chunk data a single Stream
// may buffer ahead of DuckDB consumption before failing the stream with
// ErrStreamQueueOverflow. Keeping the queue bounded prevents unbounded Go-heap
// growth outside DuckDB's memory_limit without blocking the multiplexed reader.
const DefaultMaxStreamQueueBytes int64 = 256 << 20

const maxStreamChunkRows = 2048

var (
	// ErrStreamClosed is returned by Stream.Push when the stream was already closed.
	ErrStreamClosed = errors.New("libduckdb: stream closed")
	// ErrStreamQueueOverflow is returned by Stream.Push when a stream's queued
	// bytes would exceed its configured limit.
	ErrStreamQueueOverflow = errors.New("libduckdb: stream queue memory limit exceeded")
)

// approxBytes estimates the heap memory retained by a StreamChunk.
func (c StreamChunk) approxBytes() int64 {
	total := int64(64)
	for _, col := range c.Columns {
		total += int64(len(col.Nulls))
		total += int64(len(col.UUIDs)) * 16
		total += int64(len(col.Int64s)) * 8
		total += int64(len(col.Bools))
		total += int64(len(col.Bytes))
		total += int64(len(col.Ends)) * 4
	}
	return total
}

// Stream feeds chunks from a Go producer into a running DuckDB query via the
// registered `scan_data_from_quack_client` table function.
//
// It uses a mutex+cond FIFO queue rather than a bounded channel so:
//   - closing early (e.g. LIMIT or query cancel) can never race Push into a
//     send-on-closed-channel panic, and
//   - pushing frames for stream A on a multiplexed socket reader can never
//     block dispatching frames for stream B.
//
// Total queued bytes are capped at maxQueueBytes (default
// DefaultMaxStreamQueueBytes) so an unconsumed stream fails cleanly instead of
// growing Go heap memory without bound.
type Stream struct {
	id            string
	mu            sync.Mutex
	cond          *sync.Cond
	queue         []StreamChunk
	queuedBytes   int64
	maxQueueBytes int64
	closed        bool
	err           error
}

// SetMaxQueueBytes overrides the queued-byte limit for this stream (<= 0 disables).
func (s *Stream) SetMaxQueueBytes(limit int64) {
	s.mu.Lock()
	s.maxQueueBytes = limit
	s.mu.Unlock()
}

// Push appends a chunk to the stream queue without blocking. Returns
// ErrStreamClosed if the stream was closed early (e.g., query finished on LIMIT
// or cancelled), or ErrStreamQueueOverflow if buffering the chunk would exceed
// the stream's memory limit.
func (s *Stream) Push(chunk StreamChunk) error {
	sz := chunk.approxBytes()
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrStreamClosed
	}
	if s.maxQueueBytes > 0 && s.queuedBytes+sz > s.maxQueueBytes {
		s.queue = nil
		s.queuedBytes = 0
		s.err = fmt.Errorf("%w: stream %q exceeded %d bytes", ErrStreamQueueOverflow, s.id, s.maxQueueBytes)
		s.closed = true
		s.cond.Broadcast()
		s.mu.Unlock()
		return ErrStreamQueueOverflow
	}
	s.queue = append(s.queue, chunk)
	s.queuedBytes += sz
	s.cond.Signal()
	s.mu.Unlock()
	return nil
}

// Close marks the stream closed (signaling EOF once queued chunks are drained).
func (s *Stream) Close() {
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		s.cond.Broadcast()
	}
	s.mu.Unlock()
}

// CloseWithError delivers a source error to the table function and closes the stream.
func (s *Stream) CloseWithError(err error) {
	s.mu.Lock()
	if !s.closed {
		s.queue = nil
		s.queuedBytes = 0
		s.err = err
		s.closed = true
		s.cond.Broadcast()
	}
	s.mu.Unlock()
}

// Next pops the next chunk, waiting until one arrives or the stream closes.
func (s *Stream) Next() (StreamChunk, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for len(s.queue) == 0 && !s.closed {
		s.cond.Wait()
	}
	if len(s.queue) > 0 {
		chunk := s.queue[0]
		s.queue[0] = StreamChunk{}
		s.queue = s.queue[1:]
		s.queuedBytes -= chunk.approxBytes()
		if s.queuedBytes < 0 {
			s.queuedBytes = 0
		}
		return chunk, true
	}
	if s.err != nil {
		err := s.err
		s.err = nil
		return StreamChunk{Err: err}, true
	}
	return StreamChunk{}, false
}

// streamRegistry maps active stream IDs to their Stream channels, and active
// table-function bind handles (integer IDs cast to uintptr) to their bindState.
// Using integer handle IDs avoids passing Go pointers to C memory.
var (
	streamMu      sync.Mutex
	streamByID    = make(map[string]*Stream)
	bindNextID    atomic.Uintptr
	bindByHandle  = make(map[uintptr]*streamBindState)
	streamLibRef  atomic.Pointer[Lib]
	streamCBOnce  sync.Once
	streamBindCB  uintptr
	streamInitCB  uintptr
	streamFuncCB  uintptr
	streamFreeCB  uintptr
)

type streamBindState struct {
	lib    *Lib
	stream *Stream
	types  []Type
}

// RegisterStream creates and registers a new Stream under id. The caller must
// call UnregisterStream(id) once the query finishes.
func RegisterStream(id string) *Stream {
	s := &Stream{id: id, maxQueueBytes: DefaultMaxStreamQueueBytes}
	s.cond = sync.NewCond(&s.mu)
	streamMu.Lock()
	streamByID[id] = s
	streamMu.Unlock()
	return s
}

// UnregisterStream removes id from the registry and closes its channel.
func UnregisterStream(id string) {
	streamMu.Lock()
	s, ok := streamByID[id]
	if ok {
		delete(streamByID, id)
	}
	streamMu.Unlock()
	if ok {
		s.Close()
	}
}

func initStreamCallbacks() {
	streamCBOnce.Do(func() {
		streamBindCB = purego.NewCallback(streamBindCallback)
		streamInitCB = purego.NewCallback(streamInitCallback)
		streamFuncCB = purego.NewCallback(streamFuncCallback)
		streamFreeCB = purego.NewCallback(streamDestroyCallback)
	})
}

// RegisterStreamTableFunction registers the `scan_data_from_quack_client` table
// function on conn so SQL queries can stream Go relations directly into DuckDB.
// Safe to call once per DB (e.g. on initial connection).
func (c *Conn) RegisterStreamTableFunction() error {
	c.execMu.Lock()
	defer c.execMu.Unlock()

	c.handleMu.RLock()
	handle := c.handle
	closed := c.closed
	c.handleMu.RUnlock()
	if closed || handle == 0 {
		return errors.New("libduckdb: connection is closed")
	}

	streamLibRef.Store(c.lib)
	initStreamCallbacks()

	tf := c.lib.createTableFunction()
	if tf == 0 {
		return errors.New("libduckdb: duckdb_create_table_function failed")
	}
	defer c.lib.destroyTableFunction(&tf)

	c.lib.tableFunctionSetName(tf, "scan_data_from_quack_client")

	varcharLT := c.lib.createLogicalType(int32(TypeVarchar))
	defer c.lib.destroyLogicalType(&varcharLT)
	anyLT := c.lib.createLogicalType(int32(TypeAny))
	defer c.lib.destroyLogicalType(&anyLT)
	boolLT := c.lib.createLogicalType(int32(TypeBoolean))
	defer c.lib.destroyLogicalType(&boolLT)

	c.lib.tableFunctionAddParam(tf, varcharLT)
	c.lib.tableFunctionAddParam(tf, anyLT)
	c.lib.tableFunctionAddNamedParam(tf, "ordered", boolLT)

	c.lib.tableFunctionSetBind(tf, streamBindCB)
	c.lib.tableFunctionSetInit(tf, streamInitCB)
	c.lib.tableFunctionSetFunction(tf, streamFuncCB)

	if State(c.lib.registerTableFunction(handle, tf)) != StateSuccess {
		return errors.New("libduckdb: duckdb_register_table_function failed")
	}
	return nil
}

func streamBindCallback(info uintptr) {
	lib := streamLibRef.Load()
	if lib == nil {
		return
	}

	// Param 0: stream ID (VARCHAR).
	p0 := lib.bindGetParam(info, 0)
	if p0 == 0 {
		lib.bindSetError(info, "scan_data_from_quack_client: missing stream id parameter")
		return
	}
	idPtr := lib.getVarchar(p0)
	streamID := cString(idPtr)
	if idPtr != nil {
		lib.free(unsafe.Pointer(idPtr))
	}
	lib.destroyValue(&p0)

	// Param 1: NULL::STRUCT(...) schema prototype.
	p1 := lib.bindGetParam(info, 1)
	if p1 == 0 {
		lib.bindSetError(info, "scan_data_from_quack_client: missing schema prototype parameter")
		return
	}
	structLT := lib.getValueType(p1)
	if structLT != 0 {
		defer lib.destroyLogicalType(&structLT)
	}
	if structLT == 0 || Type(lib.getTypeID(structLT)) != TypeStruct {
		lib.destroyValue(&p1)
		lib.bindSetError(info, "scan_data_from_quack_client: second parameter must be NULL::STRUCT(...)")
		return
	}

	childCount := lib.structTypeChildCount(structLT)
	colTypes := make([]Type, childCount)
	for i := uint64(0); i < childCount; i++ {
		np := lib.structTypeChildName(structLT, i)
		name := cString(np)
		if np != nil {
			lib.free(unsafe.Pointer(np))
		}
		ct := lib.structTypeChildType(structLT, i)
		colTypes[i] = Type(lib.getTypeID(ct))
		lib.bindAddResultColumn(info, name, ct)
		if ct != 0 {
			lib.destroyLogicalType(&ct)
		}
	}
	lib.destroyValue(&p1)

	streamMu.Lock()
	st, ok := streamByID[streamID]
	streamMu.Unlock()
	if !ok {
		lib.bindSetError(info, fmt.Sprintf("scan_data_from_quack_client: unknown stream %q", streamID))
		return
	}

	hid := bindNextID.Add(1)
	streamMu.Lock()
	bindByHandle[hid] = &streamBindState{
		lib:    lib,
		stream: st,
		types:  colTypes,
	}
	streamMu.Unlock()

	lib.bindSetBindData(info, hid, streamFreeCB)
}

func streamInitCallback(info uintptr) {
	lib := streamLibRef.Load()
	if lib == nil {
		return
	}
	// Preserve batch/row order by scanning single-threaded.
	lib.initSetMaxThreads(info, 1)
}

func streamDestroyCallback(data uintptr) {
	streamMu.Lock()
	delete(bindByHandle, data)
	streamMu.Unlock()
}

func streamFuncCallback(info uintptr, output uintptr) {
	lib := streamLibRef.Load()
	if lib == nil {
		return
	}
	hid := lib.functionGetBindData(info)
	streamMu.Lock()
	state := bindByHandle[hid]
	streamMu.Unlock()
	if state == nil {
		lib.functionSetError(info, "scan_data_from_quack_client: missing bind state")
		return
	}

	chunk, ok := state.stream.Next()
	if !ok {
		lib.dataChunkSetSize(output, 0)
		return
	}
	if chunk.Err != nil {
		lib.functionSetError(info, chunk.Err.Error())
		return
	}
	if chunk.RowCount == 0 {
		lib.dataChunkSetSize(output, 0)
		return
	}
	if chunk.RowCount < 0 || chunk.RowCount > maxStreamChunkRows {
		lib.functionSetError(info, fmt.Sprintf("scan_data_from_quack_client: chunk row count %d exceeds maximum %d", chunk.RowCount, maxStreamChunkRows))
		return
	}
	if len(chunk.Columns) != len(state.types) {
		lib.functionSetError(info, fmt.Sprintf("scan_data_from_quack_client: batch has %d columns, expected %d", len(chunk.Columns), len(state.types)))
		return
	}

	for colIdx, col := range chunk.Columns {
		if err := validateStreamColumnType(col.Kind, state.types[colIdx]); err != nil {
			lib.functionSetError(info, fmt.Sprintf("scan_data_from_quack_client: column %d: %v", colIdx, err))
			return
		}
		vec := lib.dataChunkGetVector(output, uint64(colIdx))
		if err := writeStreamColumn(lib, vec, col, chunk.RowCount); err != nil {
			lib.functionSetError(info, fmt.Sprintf("scan_data_from_quack_client: column %d: %v", colIdx, err))
			return
		}
	}
	lib.dataChunkSetSize(output, uint64(chunk.RowCount))
}

func validateStreamColumnType(kind StreamColumnKind, expected Type) error {
	switch kind {
	case StreamColumnUUID:
		if expected == TypeUUID {
			return nil
		}
	case StreamColumnVarchar, StreamColumnJSON:
		if expected == TypeVarchar {
			return nil
		}
	case StreamColumnTimestampTZ:
		if expected == TypeTimestampTZ {
			return nil
		}
	case StreamColumnBool:
		if expected == TypeBoolean {
			return nil
		}
	case StreamColumnBigint:
		if expected == TypeBigInt {
			return nil
		}
	default:
		return fmt.Errorf("unsupported stream column kind %d", kind)
	}
	return fmt.Errorf("stream column kind %d does not match bound DuckDB type %d", kind, expected)
}

func writeStreamColumn(lib *Lib, vec uintptr, col StreamColumn, rowCount int) error {
	hasNulls := false
	for i := 0; i < rowCount && i < len(col.Nulls); i++ {
		if col.Nulls[i] {
			hasNulls = true
			break
		}
	}
	var validity *uint64
	if hasNulls {
		lib.vectorEnsureValidity(vec)
		validity = lib.vectorGetValidity(vec)
	}

	switch col.Kind {
	case StreamColumnUUID:
		data := lib.vectorGetData(vec)
		for i := 0; i < rowCount; i++ {
			if hasNulls && col.Nulls[i] {
				lib.validitySetRowInvalid(validity, uint64(i))
				continue
			}
			u := col.UUIDs[i]
			hi := binary.BigEndian.Uint64(u[0:8]) ^ (uint64(1) << 63)
			lo := binary.BigEndian.Uint64(u[8:16])
			dst := (*cHugeInt)(unsafe.Add(data, uintptr(i)*unsafe.Sizeof(cHugeInt{})))
			dst.lower = lo
			dst.upper = int64(hi)
		}
	case StreamColumnTimestampTZ, StreamColumnBigint:
		data := lib.vectorGetData(vec)
		for i := 0; i < rowCount; i++ {
			if hasNulls && col.Nulls[i] {
				lib.validitySetRowInvalid(validity, uint64(i))
				continue
			}
			*(*int64)(unsafe.Add(data, uintptr(i)*8)) = col.Int64s[i]
		}
	case StreamColumnBool:
		data := lib.vectorGetData(vec)
		for i := 0; i < rowCount; i++ {
			if hasNulls && col.Nulls[i] {
				lib.validitySetRowInvalid(validity, uint64(i))
				continue
			}
			if col.Bools[i] {
				*(*byte)(unsafe.Add(data, uintptr(i))) = 1
			} else {
				*(*byte)(unsafe.Add(data, uintptr(i))) = 0
			}
		}
	case StreamColumnVarchar, StreamColumnJSON:
		prev := uint32(0)
		for i := 0; i < rowCount; i++ {
			end := col.Ends[i]
			if hasNulls && col.Nulls[i] {
				lib.validitySetRowInvalid(validity, uint64(i))
				prev = end
				continue
			}
			n := end - prev
			if n == 0 {
				var empty byte
				lib.vectorAssignStringLen(vec, uint64(i), unsafe.Pointer(&empty), 0)
			} else {
				lib.vectorAssignStringLen(vec, uint64(i), unsafe.Pointer(&col.Bytes[prev]), uint64(n))
			}
			prev = end
		}
	default:
		return fmt.Errorf("unsupported stream column kind %d", col.Kind)
	}
	return nil
}
