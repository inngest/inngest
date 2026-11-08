package quack

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// terminatorFieldID marks the end of an object on the wire: a raw
// uint16 0xFFFF written where a field id would otherwise appear.
const terminatorFieldID = 0xFFFF

// writer and reader implement DuckDB's BinarySerializer wire
// codec: signed ints are sign-extending LEB128 (not ZigZag), unsigned ints
// are standard LEB128, strings/blobs are an unsigned-LEB128 length prefix
// plus raw bytes, and every object ends with a raw uint16 0xFFFF terminator.
type writer struct {
	buf bytes.Buffer
}

func (w *writer) bytes() []byte { return w.buf.Bytes() }

func (w *writer) beginObject() {}

func (w *writer) endObject() { w.writeFieldID(terminatorFieldID) }

func (w *writer) writeFieldID(id uint16) {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], id)
	w.buf.Write(b[:])
}

func (w *writer) writeUnsignedLeb128(v uint64) {
	for {
		b := byte(v & 0x7F)
		v >>= 7
		if v != 0 {
			b |= 0x80
		}
		w.buf.WriteByte(b)
		if v == 0 {
			return
		}
	}
}

func (w *writer) writeSignedLeb128(v int64) {
	for {
		b := byte(v & 0x7F)
		v >>= 7
		done := (v == 0 && b&0x40 == 0) || (v == -1 && b&0x40 != 0)
		if !done {
			b |= 0x80
		}
		w.buf.WriteByte(b)
		if done {
			return
		}
	}
}

func (w *writer) writeBool(id uint16, v bool) {
	w.writeFieldID(id)
	if v {
		w.buf.WriteByte(1)
	} else {
		w.buf.WriteByte(0)
	}
}

func (w *writer) writeByte(id uint16, v byte) {
	w.writeFieldID(id)
	w.writeUnsignedLeb128(uint64(v))
}

func (w *writer) writeUint64(id uint16, v uint64) {
	w.writeFieldID(id)
	w.writeUnsignedLeb128(v)
}

// writeHugeint writes hi as signed LEB128 and lo as unsigned LEB128; always
// present on the wire, never default-omitted.
func (w *writer) writeHugeint(id uint16, v hugeint) {
	w.writeFieldID(id)
	w.writeSignedLeb128(v.hi)
	w.writeUnsignedLeb128(v.lo)
}

func (w *writer) writeUint64Default(id uint16, v uint64) {
	if v == 0 {
		return
	}
	w.writeUint64(id, v)
}

func (w *writer) writeString(id uint16, v string) {
	w.writeFieldID(id)
	w.writeUnsignedLeb128(uint64(len(v)))
	w.buf.WriteString(v)
}

func (w *writer) writeStringDefault(id uint16, v string) {
	if v == "" {
		return
	}
	w.writeString(id, v)
}

// writeData writes an unsigned-LEB128 length prefix followed by data. Used
// for the top-level ConnectionRequest body; this client never writes a
// DataChunk payload, only reads them.
func (w *writer) writeData(data []byte) {
	w.writeUnsignedLeb128(uint64(len(data)))
	w.buf.Write(data)
}

// beginList writes a list's element count; there is no endList since the
// wire format has no list terminator.
func (w *writer) beginList(count uint64) {
	w.writeUnsignedLeb128(count)
}

// reader parses messages from the duckdb quack server. Field ids are
// read one token ahead (peek/consume) so tryBeginProperty can detect a
// default-omitted optional field (next id greater than expected) without
// consuming it.
type reader struct {
	data          []byte
	pos           int
	hasBuffered   bool
	bufferedField uint16
}

func newReader(data []byte) *reader {
	return &reader{data: data}
}

var errTruncated = errors.New("duckdb: quack message truncated")

func (r *reader) readRawByte() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, errTruncated
	}
	b := r.data[r.pos]
	r.pos++
	return b, nil
}

func (r *reader) readRawBytes(n int) ([]byte, error) {
	if r.pos+n > len(r.data) {
		return nil, errTruncated
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b, nil
}

func (r *reader) readRawUint16() (uint16, error) {
	b, err := r.readRawBytes(2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(b), nil
}

func (r *reader) readUnsignedLeb128() (uint64, error) {
	var result uint64
	var shift uint
	for {
		if shift >= 64 {
			return 0, fmt.Errorf("duckdb: unsigned LEB128 value exceeds 64 bits")
		}
		b, err := r.readRawByte()
		if err != nil {
			return 0, fmt.Errorf("duckdb: truncated unsigned LEB128: %w", err)
		}
		result |= uint64(b&0x7F) << shift
		shift += 7
		if b&0x80 == 0 {
			return result, nil
		}
	}
}

func (r *reader) readSignedLeb128() (int64, error) {
	var result int64
	var shift uint
	var b byte
	for {
		if shift >= 64 {
			return 0, fmt.Errorf("duckdb: signed LEB128 value exceeds 64 bits")
		}
		var err error
		b, err = r.readRawByte()
		if err != nil {
			return 0, fmt.Errorf("duckdb: truncated signed LEB128: %w", err)
		}
		result |= int64(b&0x7F) << shift
		shift += 7
		if b&0x80 == 0 {
			break
		}
	}
	if shift < 64 && b&0x40 != 0 {
		result |= -(1 << shift)
	}
	return result, nil
}

func (r *reader) peekField() (uint16, error) {
	if !r.hasBuffered {
		f, err := r.readRawUint16()
		if err != nil {
			return 0, err
		}
		r.bufferedField = f
		r.hasBuffered = true
	}
	return r.bufferedField, nil
}

func (r *reader) consumeField() { r.hasBuffered = false }

func (r *reader) nextField() (uint16, error) {
	if r.hasBuffered {
		r.hasBuffered = false
		return r.bufferedField, nil
	}
	return r.readRawUint16()
}

func (r *reader) beginObject() {}

func (r *reader) endObject() error {
	next, err := r.nextField()
	if err != nil {
		return err
	}
	if next != terminatorFieldID {
		return fmt.Errorf("duckdb: expected end-of-object terminator (0x%04x) but found field id 0x%04x", terminatorFieldID, next)
	}
	return nil
}

func (r *reader) beginProperty(id uint16) error {
	actual, err := r.nextField()
	if err != nil {
		return err
	}
	if actual != id {
		return fmt.Errorf("duckdb: expected field id 0x%04x but found 0x%04x", id, actual)
	}
	return nil
}

// tryBeginProperty reports whether the next field matches id, consuming it
// if so. Returns false without consuming if the field is past id (optional
// field omitted) or is the terminator; errors if it's below id (corrupt stream).
func (r *reader) tryBeginProperty(id uint16) (bool, error) {
	next, err := r.peekField()
	if err != nil {
		return false, err
	}
	if next == id {
		r.consumeField()
		return true, nil
	}
	if next == terminatorFieldID || next > id {
		return false, nil
	}
	return false, fmt.Errorf("duckdb: out-of-order field id 0x%04x (expected >= 0x%04x)", next, id)
}

func (r *reader) readBool() (bool, error) {
	b, err := r.readRawByte()
	if err != nil {
		return false, err
	}
	return b != 0, nil
}

func (r *reader) readByte() (byte, error) {
	v, err := r.readUnsignedLeb128()
	return byte(v), err
}

func (r *reader) readUInt32() (uint32, error) {
	v, err := r.readUnsignedLeb128()
	return uint32(v), err
}

func (r *reader) readUInt64() (uint64, error) {
	return r.readUnsignedLeb128()
}

func (r *reader) readString() (string, error) {
	n, err := r.readUnsignedLeb128()
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", nil
	}
	b, err := r.readRawBytes(int(n))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *reader) readData() ([]byte, error) {
	n, err := r.readUnsignedLeb128()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}
	return r.readRawBytes(int(n))
}

func (r *reader) beginList() (uint64, error) {
	return r.readUnsignedLeb128()
}

func (r *reader) beginNullable() (bool, error) {
	return r.readBool()
}
