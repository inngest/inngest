package cluster

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// quiet is how long a test waits before it concludes that nothing arrives.
const quiet = 50 * time.Millisecond

// listenOn starts a listener for the node and returns the streams it accepts.
func listenOn(t *testing.T, net *MemNetwork, node string) <-chan Stream {
	t.Helper()
	accepted := make(chan Stream, 16)
	_, err := net.Transport(node).Listen(node, func(s Stream) { accepted <- s })
	require.NoError(t, err)
	return accepted
}

// connect dials from one node to a listening node and returns both ends.
func connect(t *testing.T, net *MemNetwork, from, to string, accepted <-chan Stream) (client, server Stream) {
	t.Helper()
	client, err := net.Transport(from).Dial(context.Background(), to)
	require.NoError(t, err)
	select {
	case server = <-accepted:
	case <-time.After(5 * time.Second):
		t.Fatal("the listener did not accept the stream")
	}
	return client, server
}

type recvResult struct {
	m   Message
	err error
}

// recvAsync runs one Recv on a goroutine.
func recvAsync(s Stream) <-chan recvResult {
	out := make(chan recvResult, 1)
	go func() {
		m, err := s.Recv()
		out <- recvResult{
			m:   m,
			err: err,
		}
	}()
	return out
}

func requireNothing[T any](t *testing.T, ch <-chan T) {
	t.Helper()
	select {
	case <-ch:
		t.Fatal("got a value, want none")
	case <-time.After(quiet):
	}
}

func requireSoon[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatal("got no value")
	}
	panic("unreachable")
}

func TestMemNetworkOrderAndReference(t *testing.T) {
	net := NewMemNetwork()
	client, server := connect(t, net, "a", "b", listenOn(t, net, "b"))

	const n = 3000
	go func() {
		for i := range n {
			if err := client.Send(i); err != nil {
				return
			}
		}
	}()
	for i := range n {
		m, err := server.Recv()
		require.NoError(t, err)
		require.Equal(t, i, m)
	}

	// the other direction works on the same stream.
	type payload struct{ v int }
	p := &payload{v: 1}
	require.NoError(t, server.Send(p))
	m, err := client.Recv()
	require.NoError(t, err)
	require.Same(t, p, m)
}

func TestMemNetworkListenAndDial(t *testing.T) {
	net := NewMemNetwork()
	a, b := net.Transport("a"), net.Transport("b")

	_, err := a.Dial(context.Background(), "b")
	require.ErrorIs(t, err, ErrDialRefused)

	_, err = b.Listen("c", func(Stream) {})
	require.Error(t, err)

	accepted := make(chan Stream, 1)
	ln, err := b.Listen("b", func(s Stream) { accepted <- s })
	require.NoError(t, err)
	_, err = b.Listen("b", func(Stream) {})
	require.Error(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = a.Dial(ctx, "b")
	require.ErrorIs(t, err, context.Canceled)

	client, err := a.Dial(context.Background(), "b")
	require.NoError(t, err)
	server := requireSoon(t, accepted)

	// a closed listener refuses dials.  its open streams stay open.
	require.NoError(t, ln.Close())
	_, err = a.Dial(context.Background(), "b")
	require.ErrorIs(t, err, ErrDialRefused)
	require.NoError(t, client.Send("still open"))
	m, err := server.Recv()
	require.NoError(t, err)
	require.Equal(t, "still open", m)

	// a node can dial its own address.
	self, selfServer := connect(t, net, "a", "a", listenOn(t, net, "a"))
	require.NoError(t, self.Send("me"))
	m, err = selfServer.Recv()
	require.NoError(t, err)
	require.Equal(t, "me", m)
}

func TestMemNetworkClose(t *testing.T) {
	net := NewMemNetwork()
	client, server := connect(t, net, "a", "b", listenOn(t, net, "b"))

	require.NoError(t, client.Send(1))
	require.NoError(t, client.Send(2))
	require.NoError(t, client.Close())
	require.NoError(t, client.Close())

	// the end that closed gets errors at once.
	require.ErrorIs(t, client.Send(3), ErrStreamClosed)
	_, err := client.Recv()
	require.ErrorIs(t, err, ErrStreamClosed)

	// the other end reads what was sent before the Close, then the error.
	require.ErrorIs(t, server.Send("late"), ErrStreamClosed)
	for _, want := range []int{1, 2} {
		m, err := server.Recv()
		require.NoError(t, err)
		require.Equal(t, want, m)
	}
	_, err = server.Recv()
	require.ErrorIs(t, err, ErrStreamClosed)
	_, err = server.Recv()
	require.ErrorIs(t, err, ErrStreamClosed)
}

func TestMemNetworkCloseWakesRecv(t *testing.T) {
	net := NewMemNetwork()
	client, server := connect(t, net, "a", "b", listenOn(t, net, "b"))

	// a Close wakes a Recv that waits on the same end or on the other end.
	own, peer := recvAsync(client), recvAsync(server)
	requireNothing(t, own)
	require.NoError(t, client.Close())
	require.ErrorIs(t, requireSoon(t, own).err, ErrStreamClosed)
	require.ErrorIs(t, requireSoon(t, peer).err, ErrStreamClosed)
}

func TestMemNetworkBackpressure(t *testing.T) {
	net := NewMemNetwork()
	client, server := connect(t, net, "a", "b", listenOn(t, net, "b"))

	for i := range memStreamBuffer {
		require.NoError(t, client.Send(i))
	}
	blocked := make(chan error, 1)
	go func() { blocked <- client.Send(memStreamBuffer) }()
	requireNothing(t, blocked)

	// one Recv makes room for the blocked Send.
	m, err := server.Recv()
	require.NoError(t, err)
	require.Equal(t, 0, m)
	require.NoError(t, requireSoon(t, blocked))

	// a Send that is blocked when the stream breaks returns an error.
	go func() { blocked <- client.Send(memStreamBuffer + 1) }()
	requireNothing(t, blocked)
	net.Partition("b", "a")
	require.ErrorIs(t, requireSoon(t, blocked), ErrStreamBroken)
}

func TestMemNetworkPartitionAndHeal(t *testing.T) {
	net := NewMemNetwork()
	acceptedB := listenOn(t, net, "b")
	acceptedC := listenOn(t, net, "c")
	ab, ba := connect(t, net, "a", "b", acceptedB)
	ac, ca := connect(t, net, "a", "c", acceptedC)

	// a queued message is dropped by the break.
	require.NoError(t, ab.Send("lost"))
	waiting := recvAsync(ab)
	requireNothing(t, waiting)

	net.Partition("a", "b")
	require.ErrorIs(t, requireSoon(t, waiting).err, ErrStreamBroken)
	require.ErrorIs(t, ab.Send(1), ErrStreamBroken)
	require.ErrorIs(t, ba.Send(1), ErrStreamBroken)
	_, err := ba.Recv()
	require.ErrorIs(t, err, ErrStreamBroken)

	// new dials are refused in both directions.
	_, err = net.Transport("a").Dial(context.Background(), "b")
	require.ErrorIs(t, err, ErrDialRefused)
	listenOn(t, net, "a")
	_, err = net.Transport("b").Dial(context.Background(), "a")
	require.ErrorIs(t, err, ErrDialRefused)

	// the pair of a and c is not touched.
	require.NoError(t, ac.Send("ok"))
	m, err := ca.Recv()
	require.NoError(t, err)
	require.Equal(t, "ok", m)

	net.Heal("b", "a")
	// a broken stream stays broken.  a new one works.
	require.ErrorIs(t, ab.Send(2), ErrStreamBroken)
	ab, ba = connect(t, net, "a", "b", acceptedB)
	require.NoError(t, ab.Send("healed"))
	m, err = ba.Recv()
	require.NoError(t, err)
	require.Equal(t, "healed", m)
}

func TestMemNetworkIsolateAndRejoin(t *testing.T) {
	net := NewMemNetwork()
	acceptedA := listenOn(t, net, "a")
	acceptedB := listenOn(t, net, "b")
	acceptedC := listenOn(t, net, "c")
	ab, _ := connect(t, net, "a", "b", acceptedB)
	ca, _ := connect(t, net, "c", "a", acceptedA)
	bc, cb := connect(t, net, "b", "c", acceptedC)

	net.Isolate("a")
	require.ErrorIs(t, ab.Send(1), ErrStreamBroken)
	require.ErrorIs(t, ca.Send(1), ErrStreamBroken)
	for _, to := range []string{"b", "c"} {
		_, err := net.Transport("a").Dial(context.Background(), to)
		require.ErrorIs(t, err, ErrDialRefused)
		_, err = net.Transport(to).Dial(context.Background(), "a")
		require.ErrorIs(t, err, ErrDialRefused)
	}
	require.NoError(t, bc.Send("ok"))
	m, err := cb.Recv()
	require.NoError(t, err)
	require.Equal(t, "ok", m)

	net.Rejoin("a")
	connect(t, net, "a", "b", acceptedB)
	connect(t, net, "c", "a", acceptedA)
}

func TestMemNetworkStallThenDeliver(t *testing.T) {
	net := NewMemNetwork()
	client, server := connect(t, net, "a", "b", listenOn(t, net, "b"))

	// the receiver already waits when the stall starts.
	got := recvAsync(server)
	requireNothing(t, got)
	net.Stall("a", "b")
	net.Stall("a", "b")
	for i := range 5 {
		require.NoError(t, client.Send(i))
	}
	require.NoError(t, server.Send("back"))
	requireNothing(t, got)
	requireNothing(t, recvAsync(client))

	net.Unstall("b", "a")
	require.Equal(t, 0, requireSoon(t, got).m)
	for i := 1; i < 5; i++ {
		m, err := server.Recv()
		require.NoError(t, err)
		require.Equal(t, i, m)
	}
	net.Unstall("a", "b")
}

func TestMemNetworkStalledStreamBreaks(t *testing.T) {
	net := NewMemNetwork()
	client, server := connect(t, net, "a", "b", listenOn(t, net, "b"))
	net.Stall("a", "b")
	require.NoError(t, client.Send(1))
	got := recvAsync(server)
	requireNothing(t, got)
	net.Partition("a", "b")
	require.ErrorIs(t, requireSoon(t, got).err, ErrStreamBroken)
}

func TestMemNetworkDelayKeepsOrder(t *testing.T) {
	net := NewMemNetwork()
	client, server := connect(t, net, "a", "b", listenOn(t, net, "b"))

	const delay = 40 * time.Millisecond
	net.SetDelay("a", "b", delay)
	start := time.Now()
	for i := range 20 {
		require.NoError(t, client.Send(i))
	}
	for i := range 20 {
		m, err := server.Recv()
		require.NoError(t, err)
		require.Equal(t, i, m)
		require.GreaterOrEqual(t, time.Since(start), delay)
	}
	// the messages travel at the same time.  the delays do not add up.
	require.Less(t, time.Since(start), 10*delay)

	// a message sent with no delay does not pass one sent with a delay.
	require.NoError(t, client.Send("slow"))
	net.SetDelay("a", "b", 0)
	require.NoError(t, client.Send("fast"))
	m, err := server.Recv()
	require.NoError(t, err)
	require.Equal(t, "slow", m)
	m, err = server.Recv()
	require.NoError(t, err)
	require.Equal(t, "fast", m)

	require.NoError(t, client.Send("now"))
	start = time.Now()
	_, err = server.Recv()
	require.NoError(t, err)
	require.Less(t, time.Since(start), delay)
}

func TestMemNetworkKillAndListenAgain(t *testing.T) {
	net := NewMemNetwork()
	acceptedA := listenOn(t, net, "a")
	acceptedB := listenOn(t, net, "b")
	acceptedC := listenOn(t, net, "c")
	ab, ba := connect(t, net, "a", "b", acceptedB)
	ca, ac := connect(t, net, "c", "a", acceptedA)
	bc, cb := connect(t, net, "b", "c", acceptedC)

	waiting := recvAsync(ba)
	net.Kill("a")

	// every stream of the node is broken, dialed by it or to it.
	require.ErrorIs(t, requireSoon(t, waiting).err, ErrStreamBroken)
	require.ErrorIs(t, ab.Send(1), ErrStreamBroken)
	require.ErrorIs(t, ca.Send(1), ErrStreamBroken)
	_, err := ac.Recv()
	require.ErrorIs(t, err, ErrStreamBroken)
	_, err = net.Transport("b").Dial(context.Background(), "a")
	require.ErrorIs(t, err, ErrDialRefused)

	// other nodes are not touched.
	require.NoError(t, bc.Send("ok"))
	m, err := cb.Recv()
	require.NoError(t, err)
	require.Equal(t, "ok", m)

	// the name can listen again and takes new streams.
	acceptedA = listenOn(t, net, "a")
	ba2, ab2 := connect(t, net, "b", "a", acceptedA)
	require.NoError(t, ba2.Send("again"))
	m, err = ab2.Recv()
	require.NoError(t, err)
	require.Equal(t, "again", m)
}
