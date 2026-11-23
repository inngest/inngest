package cluster

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	pb "github.com/inngest/inngest/proto/gen/constraintapi/v1"
	"github.com/oklog/ulid/v2"
	"google.golang.org/protobuf/proto"
)

// Op names the call one Request carries.
type Op uint8

const (
	OpAcquire Op = iota + 1
	OpExtend
	OpRelease
	OpCheck
	OpSetCapacity
	OpAdjustCapacity
	OpGetCapacity
	OpReleaseSemaphore
)

// SemaphoreArgs are the arguments of one SemaphoreManager call.  Value is
// the capacity, the delta or the weight, by op.
type SemaphoreArgs struct {
	AccountID        uuid.UUID
	Name             string
	EvaluatedKeyHash string
	IdempotencyKey   string
	Value            int64
}

// Request is one call inside a RequestFrame.  the field for its Op is set
// and the others are nil.  the commit level travels inside the request.
type Request struct {
	ID        uint64
	Op        Op
	Acquire   *constraintapi.CapacityAcquireRequest
	Extend    *constraintapi.CapacityExtendLeaseRequest
	Release   *constraintapi.CapacityReleaseRequest
	Check     *constraintapi.CapacityCheckRequest
	Semaphore *SemaphoreArgs
}

// account returns the account the request is for.  ok is false when the
// field for the op is missing.
func (r *Request) account() (id uuid.UUID, ok bool) {
	switch r.Op {
	case OpAcquire:
		if r.Acquire != nil {
			return r.Acquire.AccountID, true
		}
	case OpExtend:
		if r.Extend != nil {
			return r.Extend.AccountID, true
		}
	case OpRelease:
		if r.Release != nil {
			return r.Release.AccountID, true
		}
	case OpCheck:
		if r.Check != nil {
			return r.Check.AccountID, true
		}
	case OpSetCapacity, OpAdjustCapacity, OpGetCapacity, OpReleaseSemaphore:
		if r.Semaphore != nil {
			return r.Semaphore.AccountID, true
		}
	}
	return uuid.Nil, false
}

// SemaphoreResult is the answer to one SemaphoreManager call.
type SemaphoreResult struct {
	Applied  bool
	Capacity int64
	Usage    int64
}

// ResponseError is a failed request.  NotPrimary is set when the node is not
// the open primary of the request's shard, and names the primary its table
// has.  User marks the user error of a Check.
type ResponseError struct {
	Message    string
	Retryable  bool
	User       bool
	NotPrimary *NotPrimaryError
}

// Response answers the Request with the same ID.  Err is nil on success.
type Response struct {
	ID        uint64
	Err       *ResponseError
	Acquire   *constraintapi.CapacityAcquireResponse
	Extend    *constraintapi.CapacityExtendLeaseResponse
	Release   *constraintapi.CapacityReleaseResponse
	Check     *constraintapi.CapacityCheckResponse
	Semaphore *SemaphoreResult
}

// RequestFrame is many requests sent as one message.  a client stream
// carries only these.
type RequestFrame struct {
	Reqs []Request
}

// ResponseFrame answers one RequestFrame.  Resps[i] answers Reqs[i], and
// frames come back in the order they were sent.
type ResponseFrame struct {
	Resps []Response
}

// TableRequest asks a node for the newest table it has.
type TableRequest struct{}

// TableReply answers a TableRequest.  the table is shared and read only.
type TableReply struct {
	Table *Table
}

func init() {
	RegisterMessage(MessageIDClient, RequestFrame{}, encodeRequestFrame, decodeRequestFrame)
	RegisterMessage(MessageIDClient+1, ResponseFrame{}, encodeResponseFrame, decodeResponseFrame)
	RegisterMessage(MessageIDClient+2, TableRequest{}, encodeTableRequest, decodeTableRequest)
	RegisterMessage(MessageIDClient+3, TableReply{}, encodeTableReply, decodeTableReply)
}

var errShortMessage = errors.New("cluster: message is cut short")

// wireReader reads the fields of one message.  the first failure sticks, so
// callers check err once at the end.
type wireReader struct {
	b   []byte
	err error
}

func (r *wireReader) fail() {
	if r.err == nil {
		r.err = errShortMessage
	}
	r.b = nil
}

func (r *wireReader) uvarint() uint64 {
	v, n := binary.Uvarint(r.b)
	if n <= 0 {
		r.fail()
		return 0
	}
	r.b = r.b[n:]
	return v
}

func (r *wireReader) varint() int64 {
	v, n := binary.Varint(r.b)
	if n <= 0 {
		r.fail()
		return 0
	}
	r.b = r.b[n:]
	return v
}

func (r *wireReader) byte() byte {
	if len(r.b) < 1 {
		r.fail()
		return 0
	}
	v := r.b[0]
	r.b = r.b[1:]
	return v
}

// fixed returns the next n bytes.  the result aliases the message.
func (r *wireReader) fixed(n int) []byte {
	if n < 0 || len(r.b) < n {
		r.fail()
		return nil
	}
	v := r.b[:n]
	r.b = r.b[n:]
	return v
}

// bytes returns a length prefixed field.  the result aliases the message.
func (r *wireReader) bytes() []byte {
	n := r.uvarint()
	if n > uint64(len(r.b)) {
		r.fail()
		return nil
	}
	return r.fixed(int(n))
}

func (r *wireReader) str() string {
	return string(r.bytes())
}

// count returns a list length.  a length the rest of the message cannot
// fit, at one byte an entry, is a failure, so a bad length never sizes an
// allocation.
func (r *wireReader) count() int {
	n := r.uvarint()
	if n > uint64(len(r.b)) {
		r.fail()
		return 0
	}
	return int(n)
}

func appendString(dst []byte, s string) []byte {
	dst = binary.AppendUvarint(dst, uint64(len(s)))
	return append(dst, s...)
}

func appendBool(dst []byte, v bool) []byte {
	if v {
		return append(dst, 1)
	}
	return append(dst, 0)
}

// appendProto appends m with a length prefix.
func appendProto(dst []byte, m proto.Message) ([]byte, error) {
	dst = binary.AppendUvarint(dst, uint64(proto.Size(m)))
	return proto.MarshalOptions{}.MarshalAppend(dst, m)
}

func (r *wireReader) proto(m proto.Message) {
	b := r.bytes()
	if r.err != nil {
		return
	}
	if err := proto.Unmarshal(b, m); err != nil {
		r.err = err
	}
}

func encodeRequestFrame(dst []byte, m Message) ([]byte, error) {
	f := m.(RequestFrame)
	dst = binary.AppendUvarint(dst, uint64(len(f.Reqs)))
	var err error
	for i := range f.Reqs {
		if dst, err = appendRequest(dst, &f.Reqs[i]); err != nil {
			return dst, err
		}
	}
	return dst, nil
}

func appendRequest(dst []byte, q *Request) ([]byte, error) {
	if _, ok := q.account(); !ok {
		return dst, fmt.Errorf("cluster: request %d has no body for op %d", q.ID, q.Op)
	}
	dst = binary.AppendUvarint(dst, q.ID)
	dst = append(dst, byte(q.Op))
	switch q.Op {
	case OpAcquire:
		dst = append(dst, byte(q.Acquire.CommitLevel))
		return appendProto(dst, constraintapi.CapacityAcquireRequestToProto(q.Acquire))
	case OpExtend:
		dst = append(dst, byte(q.Extend.CommitLevel))
		return appendProto(dst, constraintapi.CapacityExtendLeaseRequestToProto(q.Extend))
	case OpRelease:
		dst = append(dst, byte(q.Release.CommitLevel))
		return appendProto(dst, constraintapi.CapacityReleaseRequestToProto(q.Release))
	case OpCheck:
		return appendProto(dst, constraintapi.CapacityCheckRequestToProto(q.Check))
	}
	s := q.Semaphore
	dst = append(dst, s.AccountID[:]...)
	dst = appendString(dst, s.Name)
	dst = appendString(dst, s.EvaluatedKeyHash)
	dst = appendString(dst, s.IdempotencyKey)
	return binary.AppendVarint(dst, s.Value), nil
}

func decodeRequestFrame(b []byte) (Message, error) {
	r := &wireReader{b: b}
	n := r.count()
	f := RequestFrame{Reqs: make([]Request, n)}
	for i := 0; i < n && r.err == nil; i++ {
		readRequest(r, &f.Reqs[i])
	}
	if r.err != nil {
		return nil, r.err
	}
	return f, nil
}

func readRequest(r *wireReader, q *Request) {
	q.ID = r.uvarint()
	q.Op = Op(r.byte())
	var err error
	switch q.Op {
	case OpAcquire:
		level := constraintapi.CommitLevel(r.byte())
		var p pb.CapacityAcquireRequest
		r.proto(&p)
		if r.err != nil {
			return
		}
		if q.Acquire, err = constraintapi.CapacityAcquireRequestFromProto(&p); err == nil {
			q.Acquire.CommitLevel = level
		}
	case OpExtend:
		level := constraintapi.CommitLevel(r.byte())
		var p pb.CapacityExtendLeaseRequest
		r.proto(&p)
		if r.err != nil {
			return
		}
		if q.Extend, err = constraintapi.CapacityExtendLeaseRequestFromProto(&p); err == nil {
			q.Extend.CommitLevel = level
		}
	case OpRelease:
		level := constraintapi.CommitLevel(r.byte())
		var p pb.CapacityReleaseRequest
		r.proto(&p)
		if r.err != nil {
			return
		}
		if q.Release, err = constraintapi.CapacityReleaseRequestFromProto(&p); err == nil {
			q.Release.CommitLevel = level
		}
	case OpCheck:
		var p pb.CapacityCheckRequest
		r.proto(&p)
		if r.err != nil {
			return
		}
		q.Check, err = constraintapi.CapacityCheckRequestFromProto(&p)
	case OpSetCapacity, OpAdjustCapacity, OpGetCapacity, OpReleaseSemaphore:
		s := &SemaphoreArgs{}
		copy(s.AccountID[:], r.fixed(16))
		s.Name = r.str()
		s.EvaluatedKeyHash = r.str()
		s.IdempotencyKey = r.str()
		s.Value = r.varint()
		q.Semaphore = s
	default:
		err = fmt.Errorf("cluster: unknown op %d", q.Op)
	}
	if err != nil && r.err == nil {
		r.err = err
	}
}

// the body kinds of a Response on the wire.
const (
	bodyNone byte = iota
	bodyAcquire
	bodyExtend
	bodyRelease
	bodyCheck
	bodySemaphore
)

func encodeResponseFrame(dst []byte, m Message) ([]byte, error) {
	f := m.(ResponseFrame)
	dst = binary.AppendUvarint(dst, uint64(len(f.Resps)))
	var err error
	for i := range f.Resps {
		if dst, err = appendResponse(dst, &f.Resps[i]); err != nil {
			return dst, err
		}
	}
	return dst, nil
}

func appendResponse(dst []byte, p *Response) ([]byte, error) {
	dst = binary.AppendUvarint(dst, p.ID)
	dst = appendBool(dst, p.Err != nil)
	if e := p.Err; e != nil {
		dst = appendString(dst, e.Message)
		dst = appendBool(dst, e.Retryable)
		dst = appendBool(dst, e.User)
		dst = appendBool(dst, e.NotPrimary != nil)
		if np := e.NotPrimary; np != nil {
			dst = binary.AppendVarint(dst, int64(np.Shard))
			dst = appendString(dst, np.Primary)
			dst = appendString(dst, np.Addr)
			dst = binary.AppendUvarint(dst, np.Version)
		}
	}
	switch {
	case p.Acquire != nil:
		// the proto has no field for the request ID.
		dst = append(dst, bodyAcquire)
		dst = append(dst, p.Acquire.RequestID[:]...)
		return appendProto(dst, constraintapi.CapacityAcquireResponseToProto(p.Acquire))
	case p.Extend != nil:
		dst = append(dst, bodyExtend)
		return appendProto(dst, constraintapi.CapacityExtendLeaseResponseToProto(p.Extend))
	case p.Release != nil:
		// the proto has no field for the creation source.
		src := p.Release.CreationSource
		dst = append(dst, bodyRelease)
		dst = binary.AppendVarint(dst, int64(src.Service))
		dst = binary.AppendVarint(dst, int64(src.Location))
		dst = binary.AppendVarint(dst, int64(src.RunProcessingMode))
		return appendProto(dst, constraintapi.CapacityReleaseResponseToProto(p.Release))
	case p.Check != nil:
		dst = append(dst, bodyCheck)
		return appendProto(dst, constraintapi.CapacityCheckResponseToProto(p.Check))
	case p.Semaphore != nil:
		dst = append(dst, bodySemaphore)
		dst = appendBool(dst, p.Semaphore.Applied)
		dst = binary.AppendVarint(dst, p.Semaphore.Capacity)
		return binary.AppendVarint(dst, p.Semaphore.Usage), nil
	}
	return append(dst, bodyNone), nil
}

func decodeResponseFrame(b []byte) (Message, error) {
	r := &wireReader{b: b}
	n := r.count()
	f := ResponseFrame{Resps: make([]Response, n)}
	for i := 0; i < n && r.err == nil; i++ {
		readResponse(r, &f.Resps[i])
	}
	if r.err != nil {
		return nil, r.err
	}
	return f, nil
}

func readResponse(r *wireReader, p *Response) {
	p.ID = r.uvarint()
	if r.byte() != 0 {
		e := &ResponseError{}
		e.Message = r.str()
		e.Retryable = r.byte() != 0
		e.User = r.byte() != 0
		if r.byte() != 0 {
			np := &NotPrimaryError{}
			np.Shard = int(r.varint())
			np.Primary = r.str()
			np.Addr = r.str()
			np.Version = r.uvarint()
			e.NotPrimary = np
		}
		p.Err = e
	}
	var err error
	switch kind := r.byte(); kind {
	case bodyNone:
	case bodyAcquire:
		var id ulid.ULID
		copy(id[:], r.fixed(len(id)))
		var m pb.CapacityAcquireResponse
		r.proto(&m)
		if r.err != nil {
			return
		}
		if p.Acquire, err = constraintapi.CapacityAcquireResponseFromProto(&m); err == nil {
			p.Acquire.RequestID = id
		}
	case bodyExtend:
		var m pb.CapacityExtendLeaseResponse
		r.proto(&m)
		if r.err != nil {
			return
		}
		p.Extend, err = constraintapi.CapacityExtendLeaseResponseFromProto(&m)
	case bodyRelease:
		src := constraintapi.LeaseSource{
			Service:           constraintapi.LeaseService(r.varint()),
			Location:          constraintapi.CallerLocation(r.varint()),
			RunProcessingMode: constraintapi.RunProcessingMode(r.varint()),
		}
		var m pb.CapacityReleaseResponse
		r.proto(&m)
		if r.err != nil {
			return
		}
		if p.Release, err = constraintapi.CapacityReleaseResponseFromProto(&m); err == nil {
			p.Release.CreationSource = src
		}
	case bodyCheck:
		var m pb.CapacityCheckResponse
		r.proto(&m)
		if r.err != nil {
			return
		}
		p.Check = constraintapi.CapacityCheckResponseFromProto(&m)
	case bodySemaphore:
		s := &SemaphoreResult{}
		s.Applied = r.byte() != 0
		s.Capacity = r.varint()
		s.Usage = r.varint()
		p.Semaphore = s
	default:
		err = fmt.Errorf("cluster: unknown response body %d", kind)
	}
	if err != nil && r.err == nil {
		r.err = err
	}
}

func encodeTableRequest(dst []byte, _ Message) ([]byte, error) {
	return dst, nil
}

func decodeTableRequest([]byte) (Message, error) {
	return TableRequest{}, nil
}

func encodeTableReply(dst []byte, m Message) ([]byte, error) {
	b, err := json.Marshal(m.(TableReply).Table)
	if err != nil {
		return dst, err
	}
	return append(dst, b...), nil
}

func decodeTableReply(b []byte) (Message, error) {
	var t *Table
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, err
	}
	return TableReply{Table: t}, nil
}
