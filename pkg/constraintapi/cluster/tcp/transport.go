// Package tcp is the TCP implementation of cluster.Transport.  the wire
// encoding of every message is in the package that owns the message.
// connections have no TLS and no authentication.  the network must let only nodes of the cluster reach a
// listener.
package tcp

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi/cluster"
)

const (
	// maxFrame is the largest message body in bytes.  Send and Recv refuse a
	// larger one and the stream ends.
	maxFrame = 64 << 20
	// maxKept is the largest buffer a stream or the pool keeps for the next
	// message.  one large state part then does not pin its memory.
	maxKept = 1 << 20
	// headerSize is the big endian body length in front of every message.
	headerSize = 4
)

var sendBufs = sync.Pool{
	New: func() any { return new([]byte) },
}

// Transport opens one TCP connection per stream.
type Transport struct{}

// New returns a transport.  it has no state and is safe for concurrent use.
func New() *Transport {
	return &Transport{}
}

var _ cluster.Transport = (*Transport)(nil)

// Listener is the closer that Listen returns.
type Listener struct {
	ln   net.Listener
	done chan struct{}
}

// Listen accepts TCP connections at addr and calls accept on a new goroutine
// for each one.  the result is a *Listener.
func (t *Transport) Listen(addr string, accept func(cluster.Stream)) (io.Closer, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	l := &Listener{
		ln:   ln,
		done: make(chan struct{}),
	}
	go l.serve(accept)
	return l, nil
}

func (l *Listener) serve(accept func(cluster.Stream)) {
	defer close(l.done)
	for {
		conn, err := l.ln.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			// the process is out of file descriptors or the peer went away
			// during the handshake.  the listener stays up.
			time.Sleep(10 * time.Millisecond)
			continue
		}
		go accept(newStream(conn))
	}
}

// Addr returns the address the listener is bound to.  it carries the real
// port when the listen address had port 0.
func (l *Listener) Addr() net.Addr {
	return l.ln.Addr()
}

// Close stops new dials.  open streams stay open.
func (l *Listener) Close() error {
	err := l.ln.Close()
	<-l.done
	return err
}

// Dial opens a connection to addr.  the error wraps cluster.ErrDialRefused
// unless ctx ended first.
func (t *Transport) Dial(ctx context.Context, addr string) (cluster.Stream, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("%w: %v", cluster.ErrDialRefused, err)
	}
	return newStream(conn), nil
}

// stream is one end of a connection.
type stream struct {
	conn net.Conn
	// failed is the first error of the stream.  the connection is closed
	// when it is set, and every later Send and Recv returns it.
	failed atomic.Pointer[error]

	wmu sync.Mutex

	rmu sync.Mutex
	br  *bufio.Reader
	buf []byte
}

func newStream(conn net.Conn) *stream {
	if tc, ok := conn.(*net.TCPConn); ok {
		// a small message such as an Ack goes out at once and does not wait
		// for more bytes.
		_ = tc.SetNoDelay(true)
	}
	return &stream{
		conn: conn,
		br:   bufio.NewReaderSize(conn, 64<<10),
	}
}

// fail ends the stream with err unless it has ended already.  it returns
// the error the stream ended with.
func (s *stream) fail(err error) error {
	if s.failed.CompareAndSwap(nil, &err) {
		_ = s.conn.Close()
	}
	return *s.failed.Load()
}

func broken(err error) error {
	return fmt.Errorf("%w: %v", cluster.ErrStreamBroken, err)
}

// Send writes one message with one write call, so messages of concurrent
// senders do not mix.  it blocks while the peer does not read.
func (s *stream) Send(m cluster.Message) error {
	if err := s.failed.Load(); err != nil {
		return *err
	}
	bp := sendBufs.Get().(*[]byte)
	b, err := cluster.EncodeMessage(append((*bp)[:0], make([]byte, headerSize)...), m)
	if n := len(b) - headerSize; err == nil && n > maxFrame {
		err = fmt.Errorf("tcp: message of %d bytes is above the limit of %d", n, maxFrame)
	}
	if err == nil {
		binary.BigEndian.PutUint32(b, uint32(len(b)-headerSize))
		s.wmu.Lock()
		_, err = s.conn.Write(b)
		s.wmu.Unlock()
		if err != nil {
			err = broken(err)
		}
	}
	if cap(b) <= maxKept {
		*bp = b
		sendBufs.Put(bp)
	}
	if err != nil {
		return s.fail(err)
	}
	return nil
}

// Recv reads one message.  the message does not alias the read buffer.
// after the peer closed, it returns what the peer sent before the close and
// then cluster.ErrStreamClosed.
func (s *stream) Recv() (cluster.Message, error) {
	s.rmu.Lock()
	defer s.rmu.Unlock()
	if err := s.failed.Load(); err != nil {
		return nil, *err
	}
	m, err := s.read()
	if err != nil {
		return nil, s.fail(err)
	}
	return m, nil
}

func (s *stream) read() (cluster.Message, error) {
	head, err := s.br.Peek(headerSize)
	if err == io.EOF && len(head) == 0 {
		return nil, cluster.ErrStreamClosed
	}
	if err != nil {
		return nil, broken(err)
	}
	n := int(binary.BigEndian.Uint32(head))
	if n > maxFrame {
		return nil, broken(fmt.Errorf("tcp: message of %d bytes is above the limit of %d", n, maxFrame))
	}
	_, _ = s.br.Discard(headerSize)
	if cap(s.buf) < n {
		s.buf = make([]byte, n)
	}
	b := s.buf[:n]
	if _, err := io.ReadFull(s.br, b); err != nil {
		return nil, broken(err)
	}
	if cap(s.buf) > maxKept {
		s.buf = nil
	}
	m, err := cluster.DecodeMessage(b)
	if err != nil {
		return nil, broken(err)
	}
	return m, nil
}

// Close closes the connection.  a Send or Recv that is blocked on this end
// returns cluster.ErrStreamClosed.
func (s *stream) Close() error {
	_ = s.fail(cluster.ErrStreamClosed)
	return nil
}
