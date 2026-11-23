package cluster

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// Message is one value on a stream.  the first message on a stream says what
// the stream is for.
type Message any

// Transport opens streams between nodes.
type Transport interface {
	// Listen calls accept on a new goroutine for every stream dialed to
	// addr, until the returned closer is closed.
	Listen(addr string, accept func(Stream)) (io.Closer, error)
	Dial(ctx context.Context, addr string) (Stream, error)
}

// Stream is an ordered, reliable, two-way stream of messages.  after the
// first error from Send or Recv, every later call returns an error.
type Stream interface {
	Send(m Message) error
	Recv() (Message, error)
	Close() error
}

var (
	// ErrDialRefused is returned by Dial when nothing listens at the
	// address or the network refuses the connection.
	ErrDialRefused = errors.New("cluster: dial refused")
	// ErrStreamClosed is returned after one end closed the stream.
	ErrStreamClosed = errors.New("cluster: stream closed")
	// ErrStreamBroken is returned after the network broke the stream.
	ErrStreamBroken = errors.New("cluster: stream broken")
)

// memStreamBuffer is how many messages one direction of a stream queues
// before Send blocks.
const memStreamBuffer = 1024

// MemNetwork connects transports in one process and injects faults per
// unordered pair of nodes.  a node's address is its name.  messages are
// passed by reference, so the receiver must not change what the sender
// still reads.
type MemNetwork struct {
	mu        sync.Mutex
	nodes     map[string]bool
	listeners map[string]*memListener
	links     map[[2]string]*memLink
}

// NewMemNetwork returns a network with no nodes and no faults.
func NewMemNetwork() *MemNetwork {
	return &MemNetwork{
		nodes:     map[string]bool{},
		listeners: map[string]*memListener{},
		links:     map[[2]string]*memLink{},
	}
}

// Transport returns the transport of one node.
func (n *MemNetwork) Transport(node string) Transport {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.nodes[node] = true
	return &memTransport{
		net:  n,
		node: node,
	}
}

// Partition refuses new dials between a and b and breaks their open streams.
func (n *MemNetwork) Partition(a, b string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.partition(a, b)
}

// Heal undoes Partition.  broken streams stay broken.
func (n *MemNetwork) Heal(a, b string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.heal(a, b)
}

// Isolate partitions the node from every other node the network knows.
func (n *MemNetwork) Isolate(node string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for other := range n.nodes {
		if other != node {
			n.partition(node, other)
		}
	}
}

// Rejoin heals the node's pair with every other node the network knows.
func (n *MemNetwork) Rejoin(node string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for other := range n.nodes {
		if other != node {
			n.heal(node, other)
		}
	}
}

// Stall stops the delivery of messages between a and b.  streams stay open
// and Send still queues messages.
func (n *MemNetwork) Stall(a, b string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	gate := make(chan struct{})
	n.link(a, b).gate.CompareAndSwap(nil, &gate)
}

// Unstall undoes Stall.  every queued message is delivered in order.
func (n *MemNetwork) Unstall(a, b string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if gate := n.link(a, b).gate.Swap(nil); gate != nil {
		close(*gate)
	}
}

// SetDelay delays every message sent between a and b from now on by d.  the
// order of messages on a stream does not change.
func (n *MemNetwork) SetDelay(a, b string, d time.Duration) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.link(a, b).delay.Store(int64(d))
}

// Kill closes the node's listener and breaks every stream it has.  a later
// Listen with the same name works again.
func (n *MemNetwork) Kill(node string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.listeners, node)
	for pair, l := range n.links {
		if pair[0] == node || pair[1] == node {
			l.breakAll()
		}
	}
}

func (n *MemNetwork) partition(a, b string) {
	l := n.link(a, b)
	l.mu.Lock()
	l.partitioned = true
	l.mu.Unlock()
	l.breakAll()
}

func (n *MemNetwork) heal(a, b string) {
	l := n.link(a, b)
	l.mu.Lock()
	l.partitioned = false
	l.mu.Unlock()
}

// link returns the fault state of one unordered pair.  the caller has the
// network mutex.
func (n *MemNetwork) link(a, b string) *memLink {
	if a > b {
		a, b = b, a
	}
	pair := [2]string{a, b}
	l := n.links[pair]
	if l == nil {
		l = &memLink{
			conns: map[*memConn]struct{}{},
		}
		n.links[pair] = l
	}
	return l
}

// memLink is the fault state and the open streams of one pair of nodes.
type memLink struct {
	// gate is set while the pair is stalled.  Unstall closes the channel.
	gate  atomic.Pointer[chan struct{}]
	delay atomic.Int64

	mu          sync.Mutex
	partitioned bool
	conns       map[*memConn]struct{}
}

func (l *memLink) breakAll() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for c := range l.conns {
		close(c.broken)
	}
	clear(l.conns)
}

func (l *memLink) forget(c *memConn) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.conns, c)
}

type memConn struct {
	link *memLink
	// broken is closed by a fault.  the link closes it once, when it
	// removes the connection from its set.
	broken chan struct{}
}

type memItem struct {
	m Message
	// due is when the receiver may deliver the message.  it is zero when
	// the pair had no delay at the time of the Send.
	due time.Time
}

type memTransport struct {
	net  *MemNetwork
	node string
}

type memListener struct {
	net    *MemNetwork
	addr   string
	accept func(Stream)
}

func (t *memTransport) Listen(addr string, accept func(Stream)) (io.Closer, error) {
	if addr != t.node {
		return nil, fmt.Errorf("cluster: node %q cannot listen at %q", t.node, addr)
	}
	n := t.net
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, ok := n.listeners[addr]; ok {
		return nil, fmt.Errorf("cluster: address %q is in use", addr)
	}
	l := &memListener{
		net:    n,
		addr:   addr,
		accept: accept,
	}
	n.listeners[addr] = l
	return l, nil
}

// Close stops new dials.  open streams stay open.
func (l *memListener) Close() error {
	n := l.net
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.listeners[l.addr] == l {
		delete(n.listeners, l.addr)
	}
	return nil
}

func (t *memTransport) Dial(ctx context.Context, addr string) (Stream, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	n := t.net
	n.mu.Lock()
	defer n.mu.Unlock()
	listener, ok := n.listeners[addr]
	if !ok {
		return nil, fmt.Errorf("%w: nothing listens at %q", ErrDialRefused, addr)
	}
	l := n.link(t.node, addr)
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.partitioned {
		return nil, fmt.Errorf("%w: %q and %q are partitioned", ErrDialRefused, t.node, addr)
	}

	conn := &memConn{
		link:   l,
		broken: make(chan struct{}),
	}
	l.conns[conn] = struct{}{}
	toServer := make(chan memItem, memStreamBuffer)
	toClient := make(chan memItem, memStreamBuffer)
	client := &memStream{
		conn:       conn,
		in:         toClient,
		out:        toServer,
		closed:     make(chan struct{}),
		peerClosed: make(chan struct{}),
	}
	server := &memStream{
		conn:       conn,
		in:         toServer,
		out:        toClient,
		closed:     client.peerClosed,
		peerClosed: client.closed,
	}
	go listener.accept(server)
	return client, nil
}

// memStream is one end of a connection.
type memStream struct {
	conn *memConn
	in   <-chan memItem
	out  chan<- memItem
	// closed is closed by this end's Close, peerClosed by the other end's.
	closed     chan struct{}
	peerClosed chan struct{}
	closeOnce  sync.Once
}

func (s *memStream) Send(m Message) error {
	// a select with several ready cases picks one at random.  this check
	// makes a Send after a break or a Close fail every time.
	select {
	case <-s.conn.broken:
		return ErrStreamBroken
	case <-s.closed:
		return ErrStreamClosed
	case <-s.peerClosed:
		return ErrStreamClosed
	default:
	}
	it := memItem{m: m}
	if d := time.Duration(s.conn.link.delay.Load()); d > 0 {
		it.due = time.Now().Add(d)
	}
	select {
	case s.out <- it:
		return nil
	case <-s.conn.broken:
		return ErrStreamBroken
	case <-s.closed:
		return ErrStreamClosed
	case <-s.peerClosed:
		return ErrStreamClosed
	}
}

func (s *memStream) Recv() (Message, error) {
	select {
	case <-s.conn.broken:
		return nil, ErrStreamBroken
	case <-s.closed:
		return nil, ErrStreamClosed
	default:
	}
	var it memItem
	select {
	case it = <-s.in:
	case <-s.conn.broken:
		return nil, ErrStreamBroken
	case <-s.closed:
		return nil, ErrStreamClosed
	case <-s.peerClosed:
		// the other end closed.  what it sent before the Close is still
		// delivered.
		select {
		case it = <-s.in:
		default:
			return nil, ErrStreamClosed
		}
	}

	// the message waits here while the pair is stalled and until its delay
	// has passed.  the next message is not taken before this one returns,
	// so the order is kept.
	for {
		var gate <-chan struct{}
		var timer <-chan time.Time
		if g := s.conn.link.gate.Load(); g != nil {
			gate = *g
		} else if it.due.IsZero() {
			return it.m, nil
		} else if d := time.Until(it.due); d > 0 {
			timer = time.After(d)
		} else {
			return it.m, nil
		}
		select {
		case <-gate:
		case <-timer:
		case <-s.conn.broken:
			return nil, ErrStreamBroken
		case <-s.closed:
			return nil, ErrStreamClosed
		}
	}
}

// Close closes this end.  the other end still receives what was sent before
// the Close and then gets ErrStreamClosed.
func (s *memStream) Close() error {
	s.closeOnce.Do(func() {
		close(s.closed)
		s.conn.link.forget(s.conn)
	})
	return nil
}
