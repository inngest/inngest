package cluster

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/constraintapi/memory"
	pb "github.com/inngest/inngest/proto/gen/constraintapi/v1"
	"google.golang.org/protobuf/proto"
)

const (
	// forceBit is set in the kind byte of a record whose Force is true.
	forceBit = 0x80
	// idemBit is set in the kind byte of a grant or extend that carries an
	// idempotency record.  the key hash and its expiry follow the other
	// fields.  a record with no idempotency record costs no byte for it.
	idemBit = 0x40
	// seqShift splits a lease sequence into two varints.  the slab puts its
	// shard number above this bit, so both parts are small numbers.
	seqShift = 44
	// maxPrealloc is the most records a receiver allocates for a count it
	// was sent, before it has decoded them.
	maxPrealloc = 1 << 14
)

var errShort = errors.New("cluster: the encoding is cut short")

// AppendRecords appends the encoding of recs to dst.  each record is a kind
// byte and only the fields that kind uses.  there is no count, so two
// encodings joined end to end are the encoding of both lists.  a record of
// an unknown kind is written as its kind byte alone, and DecodeRecords
// returns an error for it.
//
// a grant is the set ID, the lease expiry, the requested and granted
// amounts, the maximum lifetime, the three source fields and the seqs.  an
// extend is the old seq, the new seq and the lease expiry.  when either has
// an idempotency record, idemBit is set in its kind byte and the record ends
// with the 8 byte key hash and the record expiry as a varint distance from
// the lease expiry.
func AppendRecords(dst []byte, recs []memory.Record) []byte {
	for i := range recs {
		r := &recs[i]
		kind := byte(r.Kind)
		if r.Force {
			kind |= forceBit
		}
		idem := r.Value != 0 && (r.Kind == memory.RecGrant || r.Kind == memory.RecExtend)
		if idem {
			kind |= idemBit
		}
		dst = append(dst, kind)
		switch r.Kind {
		case memory.RecSet:
			dst = binary.AppendUvarint(dst, r.SetID)
			dst = appendSet(dst, r.Set)
		case memory.RecSetDrop:
			dst = binary.AppendUvarint(dst, r.SetID)
		case memory.RecGrant:
			dst = binary.AppendUvarint(dst, r.SetID)
			dst = binary.AppendVarint(dst, r.ExpiresAtMS)
			dst = binary.AppendVarint(dst, int64(r.Requested))
			dst = binary.AppendVarint(dst, int64(r.Granted))
			dst = binary.AppendVarint(dst, int64(r.MaximumLifetime))
			dst = binary.AppendVarint(dst, int64(r.Source.Service))
			dst = binary.AppendVarint(dst, int64(r.Source.Location))
			dst = binary.AppendVarint(dst, int64(r.Source.RunProcessingMode))
			dst = appendSeqs(dst, r.Seqs)
			if idem {
				dst = appendIdem(dst, r)
			}
		case memory.RecTake:
			dst = appendSeq(dst, r.Seq)
		case memory.RecExtend:
			dst = appendSeq(dst, r.Seq)
			dst = appendSeq(dst, r.NewSeq)
			dst = binary.AppendVarint(dst, r.ExpiresAtMS)
			if idem {
				dst = appendIdem(dst, r)
			}
		case memory.RecTAT:
			dst = binary.BigEndian.AppendUint64(dst, r.KeyA)
			dst = binary.BigEndian.AppendUint64(dst, r.KeyB)
			dst = binary.BigEndian.AppendUint64(dst, math.Float64bits(r.TAT))
			dst = binary.BigEndian.AppendUint64(dst, r.Version)
			dst = binary.AppendVarint(dst, r.ExpiresAtMS)
		case memory.RecCell, memory.RecDelta:
			dst = binary.BigEndian.AppendUint64(dst, r.KeyA)
			dst = binary.BigEndian.AppendUint64(dst, r.KeyB)
			dst = binary.AppendVarint(dst, r.Value)
		case memory.RecIdem:
			dst = binary.BigEndian.AppendUint64(dst, r.KeyA)
			dst = binary.AppendVarint(dst, r.Value)
			dst = binary.AppendVarint(dst, r.ExpiresAtMS)
		case memory.RecSeqs:
			dst = appendSeqs(dst, r.Seqs)
		}
	}
	return dst
}

// appendIdem writes the idempotency key hash and when its record expires.
// the expiry is written as its distance from the lease expiry, which is a
// few seconds, so it takes one to three bytes.
func appendIdem(dst []byte, r *memory.Record) []byte {
	dst = binary.BigEndian.AppendUint64(dst, r.KeyA)
	return binary.AppendVarint(dst, r.Value-r.ExpiresAtMS)
}

func appendSeq(dst []byte, seq uint64) []byte {
	dst = binary.AppendUvarint(dst, seq>>seqShift)
	return binary.AppendUvarint(dst, seq&(1<<seqShift-1))
}

func appendSeqs(dst []byte, seqs []uint64) []byte {
	dst = binary.AppendUvarint(dst, uint64(len(seqs)))
	for _, seq := range seqs {
		dst = appendSeq(dst, seq)
	}
	return dst
}

// appendSet writes a presence byte and the definition.  an item that proto
// cannot marshal is written with no bytes and decodes as an empty item.
func appendSet(dst []byte, s *memory.SetDef) []byte {
	if s == nil {
		return append(dst, 0)
	}
	dst = append(dst, 1)
	dst = binary.AppendUvarint(dst, s.ID)
	dst = append(dst, s.AccountID[:]...)
	dst = append(dst, s.EnvID[:]...)
	dst = append(dst, s.FunctionID[:]...)
	dst = append(dst, s.AppID[:]...)
	dst = binary.AppendVarint(dst, int64(s.ConfigVersion))
	dst = binary.AppendUvarint(dst, uint64(len(s.Items)))
	for i, item := range s.Items {
		body, _ := proto.Marshal(constraintapi.ConstraintItemToProto(item))
		dst = binary.AppendUvarint(dst, uint64(len(body)))
		dst = append(dst, body...)
		var lim constraintapi.ConstraintLimits
		if i < len(s.Limits) {
			lim = s.Limits[i]
		}
		dst = binary.AppendVarint(dst, int64(lim.Limit))
		dst = binary.AppendVarint(dst, int64(lim.Burst))
		dst = binary.AppendVarint(dst, int64(lim.Period))
	}
	return dst
}

// DecodeRecords decodes every record in b and appends them to dst.  b is
// one AppendRecords encoding or several joined end to end.  the records do
// not alias b, so the caller may use dst again for the next call once it is
// done with the records.  on an error it returns nil.
func DecodeRecords(dst []memory.Record, b []byte) ([]memory.Record, error) {
	r := reader{b: b}
	for len(r.b) > 0 {
		dst = append(dst, memory.Record{})
		r.record(&dst[len(dst)-1])
		if r.err != nil {
			return nil, r.err
		}
	}
	return dst, nil
}

// reader takes fields from the front of b.  the first bad read sets err,
// and every read after it returns zero.
type reader struct {
	b   []byte
	err error
}

func (r *reader) fail(err error) {
	if r.err == nil {
		r.err = err
	}
	r.b = nil
}

func (r *reader) byte() byte {
	if len(r.b) < 1 {
		r.fail(errShort)
		return 0
	}
	v := r.b[0]
	r.b = r.b[1:]
	return v
}

func (r *reader) u64() uint64 {
	if len(r.b) < 8 {
		r.fail(errShort)
		return 0
	}
	v := binary.BigEndian.Uint64(r.b)
	r.b = r.b[8:]
	return v
}

func (r *reader) uvarint() uint64 {
	v, n := binary.Uvarint(r.b)
	if n <= 0 {
		r.fail(errShort)
		return 0
	}
	r.b = r.b[n:]
	return v
}

func (r *reader) varint() int64 {
	v, n := binary.Varint(r.b)
	if n <= 0 {
		r.fail(errShort)
		return 0
	}
	r.b = r.b[n:]
	return v
}

// bytes returns the next n bytes.  the result aliases the input.
func (r *reader) bytes(n uint64) []byte {
	if n > uint64(len(r.b)) {
		r.fail(errShort)
		return nil
	}
	v := r.b[:n]
	r.b = r.b[n:]
	return v
}

// count reads the length of a list whose entries take at least size bytes
// each.  a length the rest of the input cannot contain is an error, so a
// bad length never sizes an allocation.
func (r *reader) count(size int) int {
	n := r.uvarint()
	if n > uint64(len(r.b)/size) {
		r.fail(errShort)
		return 0
	}
	return int(n)
}

func (r *reader) seq() uint64 {
	hi := r.uvarint()
	return hi<<seqShift | r.uvarint()
}

// idem reads what appendIdem wrote.  the lease expiry of rec is already
// read.
func (r *reader) idem(rec *memory.Record) {
	rec.KeyA = r.u64()
	rec.Value = r.varint() + rec.ExpiresAtMS
}

func (r *reader) seqs() []uint64 {
	n := r.count(2)
	if n == 0 {
		return nil
	}
	seqs := make([]uint64, n)
	for i := range seqs {
		seqs[i] = r.seq()
	}
	return seqs
}

func (r *reader) uuid() (id uuid.UUID) {
	copy(id[:], r.bytes(uint64(len(id))))
	return id
}

// rest returns a copy of what is left and ends the input.  nothing left is
// nil.
func (r *reader) rest() []byte {
	if len(r.b) == 0 {
		return nil
	}
	v := append([]byte(nil), r.b...)
	r.b = nil
	return v
}

// record reads one record into rec, which is zero.
func (r *reader) record(rec *memory.Record) {
	kind := r.byte()
	rec.Kind = memory.RecordKind(kind &^ (forceBit | idemBit))
	rec.Force = kind&forceBit != 0
	idem := kind&idemBit != 0
	switch rec.Kind {
	case memory.RecSet:
		rec.SetID = r.uvarint()
		rec.Set = r.set()
	case memory.RecSetDrop:
		rec.SetID = r.uvarint()
	case memory.RecGrant:
		rec.SetID = r.uvarint()
		rec.ExpiresAtMS = r.varint()
		rec.Requested = int32(r.varint())
		rec.Granted = int32(r.varint())
		rec.MaximumLifetime = time.Duration(r.varint())
		rec.Source.Service = constraintapi.LeaseService(r.varint())
		rec.Source.Location = constraintapi.CallerLocation(r.varint())
		rec.Source.RunProcessingMode = constraintapi.RunProcessingMode(r.varint())
		rec.Seqs = r.seqs()
		if idem {
			r.idem(rec)
		}
	case memory.RecTake:
		rec.Seq = r.seq()
	case memory.RecExtend:
		rec.Seq = r.seq()
		rec.NewSeq = r.seq()
		rec.ExpiresAtMS = r.varint()
		if idem {
			r.idem(rec)
		}
	case memory.RecTAT:
		rec.KeyA = r.u64()
		rec.KeyB = r.u64()
		rec.TAT = math.Float64frombits(r.u64())
		rec.Version = r.u64()
		rec.ExpiresAtMS = r.varint()
	case memory.RecCell, memory.RecDelta:
		rec.KeyA = r.u64()
		rec.KeyB = r.u64()
		rec.Value = r.varint()
	case memory.RecIdem:
		rec.KeyA = r.u64()
		rec.Value = r.varint()
		rec.ExpiresAtMS = r.varint()
	case memory.RecSeqs:
		rec.Seqs = r.seqs()
	default:
		r.fail(fmt.Errorf("cluster: unknown record kind %d", rec.Kind))
	}
	if idem && rec.Kind != memory.RecGrant && rec.Kind != memory.RecExtend {
		r.fail(fmt.Errorf("cluster: record kind %d cannot carry an idempotency record", rec.Kind))
	}
}

func (r *reader) set() *memory.SetDef {
	if r.byte() == 0 {
		return nil
	}
	s := &memory.SetDef{
		ID:            r.uvarint(),
		AccountID:     r.uuid(),
		EnvID:         r.uuid(),
		FunctionID:    r.uuid(),
		AppID:         r.uuid(),
		ConfigVersion: int(r.varint()),
	}
	n := r.count(4)
	if n == 0 {
		return s
	}
	s.Items = make([]constraintapi.ConstraintItem, n)
	s.Limits = make([]constraintapi.ConstraintLimits, n)
	for i := range s.Items {
		var item pb.ConstraintItem
		if err := proto.Unmarshal(r.bytes(r.uvarint()), &item); err != nil {
			r.fail(fmt.Errorf("cluster: constraint item: %w", err))
		}
		s.Items[i] = constraintapi.ConstraintItemFromProto(&item)
		s.Limits[i] = constraintapi.ConstraintLimits{
			Limit:  int(r.varint()),
			Burst:  int(r.varint()),
			Period: int(r.varint()),
		}
		if r.err != nil {
			return nil
		}
	}
	return s
}
