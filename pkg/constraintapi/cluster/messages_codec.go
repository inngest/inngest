package cluster

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var errTrailing = errors.New("cluster: bytes left after the message")

// the record bytes of a StateRecords, Batch or TailReply go last and take
// the rest of the body, so they are written to the frame as they are with no
// second encoding.
//
// the offsets below are the wire IDs of the replication messages.  a changed
// offset makes two versions of the binary unable to talk to each other.
func init() {
	id := MessageIDReplication
	register(id+0, func(dst []byte, m Hello) []byte {
		dst = binary.AppendVarint(dst, int64(m.Shard))
		dst = binary.AppendUvarint(dst, uint64(len(m.Node)))
		dst = append(dst, m.Node...)
		dst = binary.AppendUvarint(dst, m.Term)
		return binary.AppendUvarint(dst, m.LastLSN)
	}, func(r *reader) Hello {
		return Hello{
			Shard:   int(r.varint()),
			Node:    string(r.bytes(r.uvarint())),
			Term:    r.uvarint(),
			LastLSN: r.uvarint(),
		}
	})
	register(id+1, func(dst []byte, m Resume) []byte {
		return binary.AppendUvarint(dst, m.Term)
	}, func(r *reader) Resume {
		return Resume{Term: r.uvarint()}
	})
	register(id+2, func(dst []byte, m StateBegin) []byte {
		return binary.AppendUvarint(dst, m.Term)
	}, func(r *reader) StateBegin {
		return StateBegin{Term: r.uvarint()}
	})
	register(id+3, func(dst []byte, m StateRecords) []byte {
		dst = binary.AppendUvarint(dst, uint64(m.Count))
		return append(dst, m.Data...)
	}, func(r *reader) StateRecords {
		return StateRecords{
			Count: r.recordCount(),
			Data:  r.rest(),
		}
	})
	register(id+4, func(dst []byte, m StateEnd) []byte {
		return binary.AppendUvarint(dst, m.LSN)
	}, func(r *reader) StateEnd {
		return StateEnd{LSN: r.uvarint()}
	})
	register(id+5, func(dst []byte, m Batch) []byte {
		dst = binary.AppendUvarint(dst, m.Term)
		dst = binary.AppendUvarint(dst, m.FirstLSN)
		dst = binary.AppendUvarint(dst, uint64(m.Count))
		return append(dst, m.Data...)
	}, func(r *reader) Batch {
		return Batch{
			Term:     r.uvarint(),
			FirstLSN: r.uvarint(),
			Count:    r.recordCount(),
			Data:     r.rest(),
		}
	})
	register(id+6, func(dst []byte, m Ack) []byte {
		return binary.AppendUvarint(dst, m.LSN)
	}, func(r *reader) Ack {
		return Ack{LSN: r.uvarint()}
	})
	register(id+7, func(dst []byte, m Fence) []byte {
		dst = binary.AppendVarint(dst, int64(m.Shard))
		return binary.AppendUvarint(dst, m.Term)
	}, func(r *reader) Fence {
		return Fence{
			Shard: int(r.varint()),
			Term:  r.uvarint(),
		}
	})
	register(id+8, func(dst []byte, m FenceReply) []byte {
		dst = appendBool(dst, m.OK)
		return binary.AppendUvarint(dst, m.LastLSN)
	}, func(r *reader) FenceReply {
		return FenceReply{
			OK:      r.byte() != 0,
			LastLSN: r.uvarint(),
		}
	})
	register(id+9, func(dst []byte, m TailRequest) []byte {
		dst = binary.AppendVarint(dst, int64(m.Shard))
		return binary.AppendUvarint(dst, m.FromLSN)
	}, func(r *reader) TailRequest {
		return TailRequest{
			Shard:   int(r.varint()),
			FromLSN: r.uvarint(),
		}
	})
	register(id+10, func(dst []byte, m TailReply) []byte {
		dst = appendBool(dst, m.OK)
		dst = binary.AppendUvarint(dst, m.FirstLSN)
		dst = binary.AppendUvarint(dst, uint64(m.Count))
		return append(dst, m.Data...)
	}, func(r *reader) TailReply {
		return TailReply{
			OK:       r.byte() != 0,
			FirstLSN: r.uvarint(),
			Count:    r.recordCount(),
			Data:     r.rest(),
		}
	})
}

// recordCount reads how many records the bytes after it encode.  a record
// takes at least two bytes, so a larger count is an error.
func (r *reader) recordCount() int {
	return r.count(2)
}

// register adds one message type to the cluster codec.  the decoder returns
// an error when the body is cut short or has bytes left over.
func register[M any](id uint8, enc func(dst []byte, m M) []byte, dec func(r *reader) M) {
	var zero M
	RegisterMessage(id, zero, func(dst []byte, m Message) ([]byte, error) {
		return enc(dst, m.(M)), nil
	}, func(b []byte) (Message, error) {
		r := reader{b: b}
		m := dec(&r)
		if r.err == nil && len(r.b) != 0 {
			r.err = errTrailing
		}
		if r.err != nil {
			return nil, fmt.Errorf("cluster: decode %T: %w", zero, r.err)
		}
		return m, nil
	})
}
