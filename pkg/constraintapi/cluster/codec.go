package cluster

import (
	"fmt"
	"reflect"
	"sync"
)

// message IDs are one byte on the wire.  each owner registers inside its own
// range so two packages never pick the same ID.
const (
	// MessageIDReplication is the first ID of the replication messages in
	// messages.go.
	MessageIDReplication uint8 = 1
	// MessageIDClient is the first ID of the client API messages.
	MessageIDClient uint8 = 32
	// MessageIDCoordinator is the first ID of the raftcoord messages.
	MessageIDCoordinator uint8 = 64
)

type messageCodec struct {
	id  uint8
	enc func(dst []byte, m Message) ([]byte, error)
	dec func(b []byte) (Message, error)
}

var (
	codecMu   sync.RWMutex
	codecByID [256]*messageCodec
	codecByGo = map[reflect.Type]*messageCodec{}
)

// RegisterMessage makes one message type known to the wire.  enc appends the
// body to dst.  dec returns the same Go type as example, a value and not a
// pointer, because Mux routes by that type.  it panics on a duplicate ID or
// type, which is a programming error found at start.
func RegisterMessage(id uint8, example Message, enc func(dst []byte, m Message) ([]byte, error), dec func(b []byte) (Message, error)) {
	t := reflect.TypeOf(example)
	codecMu.Lock()
	defer codecMu.Unlock()
	if codecByID[id] != nil {
		panic(fmt.Sprintf("cluster: message id %d registered twice", id))
	}
	if codecByGo[t] != nil {
		panic(fmt.Sprintf("cluster: message type %s registered twice", t))
	}
	c := &messageCodec{
		id:  id,
		enc: enc,
		dec: dec,
	}
	codecByID[id] = c
	codecByGo[t] = c
}

// EncodeMessage appends the ID byte and the body of m to dst.
func EncodeMessage(dst []byte, m Message) ([]byte, error) {
	codecMu.RLock()
	c := codecByGo[reflect.TypeOf(m)]
	codecMu.RUnlock()
	if c == nil {
		return dst, fmt.Errorf("cluster: no codec for message type %T", m)
	}
	return c.enc(append(dst, c.id), m)
}

// DecodeMessage reads one message written by EncodeMessage.  the result does
// not alias b.
func DecodeMessage(b []byte) (Message, error) {
	if len(b) == 0 {
		return nil, fmt.Errorf("cluster: empty message")
	}
	codecMu.RLock()
	c := codecByID[b[0]]
	codecMu.RUnlock()
	if c == nil {
		return nil, fmt.Errorf("cluster: unknown message id %d", b[0])
	}
	return c.dec(b[1:])
}
