package cluster

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/util/errs"
)

// ClientConfig configures a Client.
type ClientConfig struct {
	Transport Transport
	// Seeds are node addresses the client asks for the table.  one that
	// answers is enough.
	Seeds []string
	// StreamsPerNode is how many streams the client keeps to each node.
	// one stream is served by one goroutine on the node.  default 4.
	StreamsPerNode int
	// MaxFrame is the most requests one frame carries.  default 512.
	MaxFrame int
	// MaxInFlight is the most frames one stream has sent and not yet got an
	// answer for.  while a stream is at this number its requests queue, and
	// the next frame carries all of them.  default 2.
	MaxInFlight int
	// RetryBackoff is the wait before a request is sent again after a
	// not-primary answer or a broken stream.  default 10ms.
	RetryBackoff time.Duration
}

var (
	_ constraintapi.CapacityManager  = (*Client)(nil)
	_ constraintapi.SemaphoreManager = (*Client)(nil)
)

var (
	errStreamDown   = errors.New("cluster: client stream is down")
	errClientClosed = errors.New("cluster: client is closed")
)

// Client sends constraint calls to the primary of each account's shard.  it
// is safe for concurrent use, and concurrent calls to one node share frames.
type Client struct {
	cfg ClientConfig

	table       atomic.Pointer[Table]
	refreshMu   sync.Mutex
	lastRefresh time.Time

	// conns is replaced, never changed, so a request reads it with no lock.
	conns  atomic.Pointer[map[string]*clientConn]
	connMu sync.Mutex
	closed atomic.Bool

	next  atomic.Uint64
	calls sync.Pool
}

// NewClient returns a client.  it dials nothing until the first call.
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.Transport == nil || len(cfg.Seeds) == 0 {
		return nil, errors.New("cluster: client needs a transport and a seed")
	}
	if cfg.StreamsPerNode <= 0 {
		cfg.StreamsPerNode = 4
	}
	if cfg.MaxFrame <= 0 {
		cfg.MaxFrame = 512
	}
	if cfg.MaxInFlight <= 0 {
		cfg.MaxInFlight = 2
	}
	if cfg.RetryBackoff <= 0 {
		cfg.RetryBackoff = 10 * time.Millisecond
	}
	c := &Client{cfg: cfg}
	c.conns.Store(&map[string]*clientConn{})
	c.calls.New = func() any {
		return &call{done: make(chan struct{}, 1)}
	}
	return c, nil
}

// Close breaks every stream.  calls in flight return a retryable error.
func (c *Client) Close() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	c.closed.Store(true)
	for _, cn := range *c.conns.Load() {
		for i := range cn.slots {
			if st := cn.slots[i].st.Load(); st != nil {
				st.fail()
			}
		}
	}
	return nil
}

// call is one request waiting for its answer.  calls are pooled.  a call
// whose caller gave up is not put back, because the stream may still write
// to it.
type call struct {
	req  Request
	resp Response
	// down is true when the stream broke before the answer arrived.
	down bool
	// done gets one signal, after resp or down is set.
	done chan struct{}
}

// clientConn is the streams to one node.
type clientConn struct {
	slots []clientSlot
}

type clientSlot struct {
	mu sync.Mutex
	st atomic.Pointer[clientStream]
}

// clientStream is one stream to a node.  callers append to queue, and the
// writer sends everything queued as one frame of at most max requests.  the
// writer sends only while fewer than maxInFlight frames are unanswered.
// under load a frame then carries every request that arrived during a round
// trip, and an idle stream sends the first request at once.  frames are
// answered in the order sent, so the reader matches an answer frame to the
// oldest frame in flight.
type clientStream struct {
	s           Stream
	max         int
	maxInFlight int
	// wake has room for one signal.  only the goroutine that set waiting to
	// false sends on it, so the send never blocks.
	wake chan struct{}
	// dead is set once, with mu locked.  after that no call enters queue.
	dead atomic.Bool

	mu sync.Mutex
	// queue is the calls that no frame carries yet, oldest first.
	queue []*call
	// inflight is the frames sent and not answered, oldest first.
	inflight [][]*call
	// waiting is true while the writer sleeps on wake.
	waiting bool
	// spare is the call slices of answered frames.  the writer uses them
	// again, so a frame under steady load allocates no call slice.
	spare [][]*call
}

// failCalls ends every call with a broken stream.
func failCalls(calls []*call) {
	for _, cl := range calls {
		cl.down = true
		cl.done <- struct{}{}
	}
}

// fail closes the stream and ends every call that has no answer.  without
// it a caller with a context that never ends waits forever.
func (st *clientStream) fail() {
	st.mu.Lock()
	if st.dead.Load() {
		st.mu.Unlock()
		return
	}
	st.dead.Store(true)
	queue, inflight := st.queue, st.inflight
	st.queue, st.inflight = nil, nil
	wake := st.waiting
	st.waiting = false
	st.mu.Unlock()
	if wake {
		st.wake <- struct{}{}
	}
	_ = st.s.Close()
	failCalls(queue)
	for _, calls := range inflight {
		failCalls(calls)
	}
}

// enqueue adds a call to the next frame.  it is false when the stream is
// down.  the call is then not queued and gets no signal.
func (st *clientStream) enqueue(cl *call) bool {
	st.mu.Lock()
	if st.dead.Load() {
		st.mu.Unlock()
		return false
	}
	st.queue = append(st.queue, cl)
	wake := st.waiting && len(st.inflight) < st.maxInFlight
	if wake {
		st.waiting = false
	}
	st.mu.Unlock()
	if wake {
		st.wake <- struct{}{}
	}
	return true
}

// take removes the calls of the next frame from queue.  the caller has mu.
func (st *clientStream) take() []*call {
	var spare []*call
	if n := len(st.spare); n > 0 {
		spare = st.spare[n-1]
		st.spare[n-1] = nil
		st.spare = st.spare[:n-1]
	}
	if len(st.queue) <= st.max {
		calls := st.queue
		st.queue = spare
		return calls
	}
	calls := append(spare, st.queue[:st.max]...)
	n := copy(st.queue, st.queue[st.max:])
	clear(st.queue[n:])
	st.queue = st.queue[:n]
	return calls
}

func (st *clientStream) write() {
	var id uint64
	for {
		st.mu.Lock()
		for len(st.queue) == 0 || len(st.inflight) >= st.maxInFlight {
			if st.dead.Load() {
				st.mu.Unlock()
				return
			}
			st.waiting = true
			st.mu.Unlock()
			<-st.wake
			st.mu.Lock()
		}
		// the requests are copied with mu locked.  after the frame is in
		// inflight, fail can end its calls, and their callers put them back
		// in the pool.
		calls := st.take()
		reqs := make([]Request, len(calls))
		for i, cl := range calls {
			id++
			cl.req.ID = id
			reqs[i] = cl.req
		}
		st.inflight = append(st.inflight, calls)
		st.mu.Unlock()
		if err := st.s.Send(RequestFrame{Reqs: reqs}); err != nil {
			st.fail()
			return
		}
	}
}

func (st *clientStream) read() {
	defer st.fail()
	var answered []*call
	for {
		m, err := st.s.Recv()
		if err != nil {
			return
		}
		f, ok := m.(ResponseFrame)
		if !ok {
			return
		}
		st.mu.Lock()
		if answered != nil {
			st.spare = append(st.spare, answered)
			answered = nil
		}
		if len(st.inflight) == 0 {
			st.mu.Unlock()
			return
		}
		calls := st.inflight[0]
		n := copy(st.inflight, st.inflight[1:])
		st.inflight[n] = nil
		st.inflight = st.inflight[:n]
		// the writer goes first, so the next frame is on its way while this
		// one is handed to its callers.
		wake := st.waiting && len(st.queue) > 0
		if wake {
			st.waiting = false
		}
		st.mu.Unlock()
		if wake {
			st.wake <- struct{}{}
		}
		if len(calls) != len(f.Resps) {
			failCalls(calls)
			return
		}
		for i, cl := range calls {
			if f.Resps[i].ID != cl.req.ID {
				failCalls(calls[i:])
				return
			}
			cl.resp = f.Resps[i]
			cl.done <- struct{}{}
		}
		clear(calls)
		answered = calls[:0]
	}
}

// stream returns a live stream to addr, dialing one when the slot is empty
// or its stream broke.
func (c *Client) stream(ctx context.Context, addr string) (*clientStream, error) {
	cn := (*c.conns.Load())[addr]
	if cn == nil {
		c.connMu.Lock()
		old := *c.conns.Load()
		if cn = old[addr]; cn == nil {
			cn = &clientConn{slots: make([]clientSlot, c.cfg.StreamsPerNode)}
			grown := make(map[string]*clientConn, len(old)+1)
			for k, v := range old {
				grown[k] = v
			}
			grown[addr] = cn
			c.conns.Store(&grown)
		}
		c.connMu.Unlock()
	}
	slot := &cn.slots[c.next.Add(1)%uint64(len(cn.slots))]
	if st := slot.st.Load(); st != nil && !st.dead.Load() {
		return st, nil
	}
	slot.mu.Lock()
	defer slot.mu.Unlock()
	if st := slot.st.Load(); st != nil && !st.dead.Load() {
		return st, nil
	}
	if c.closed.Load() {
		return nil, errClientClosed
	}
	s, err := c.cfg.Transport.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	st := &clientStream{
		s:           s,
		max:         c.cfg.MaxFrame,
		maxInFlight: c.cfg.MaxInFlight,
		wake:        make(chan struct{}, 1),
	}
	go st.write()
	go st.read()
	slot.st.Store(st)
	if c.closed.Load() {
		st.fail()
	}
	return st, nil
}

// send queues one request on a stream to addr and waits for its answer.
func (c *Client) send(ctx context.Context, addr string, req *Request) (Response, error) {
	// done is nil for a context that never ends.
	done := ctx.Done()
	if done != nil {
		select {
		case <-done:
			return Response{}, ctx.Err()
		default:
		}
	}
	st, err := c.stream(ctx, addr)
	if err != nil {
		return Response{}, err
	}
	cl := c.calls.Get().(*call)
	cl.req = *req
	if !st.enqueue(cl) {
		c.put(cl)
		return Response{}, errStreamDown
	}
	if done == nil {
		<-cl.done
	} else {
		select {
		case <-cl.done:
		case <-done:
			return Response{}, ctx.Err()
		}
	}
	if cl.down {
		c.put(cl)
		return Response{}, errStreamDown
	}
	resp := cl.resp
	c.put(cl)
	return resp, nil
}

func (c *Client) put(cl *call) {
	cl.req, cl.resp, cl.down = Request{}, Response{}, false
	c.calls.Put(cl)
}

// fetchTable asks one node for its table.
func (c *Client) fetchTable(ctx context.Context, addr string) (*Table, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	s, err := c.cfg.Transport.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	// the close also ends the receive when ctx ends first.
	stop := context.AfterFunc(ctx, func() { _ = s.Close() })
	defer stop()
	defer s.Close()
	if err := s.Send(TableRequest{}); err != nil {
		return nil, err
	}
	m, err := s.Recv()
	if err != nil {
		return nil, err
	}
	r, ok := m.(TableReply)
	if !ok || r.Table == nil {
		return nil, fmt.Errorf("cluster: %s sent no table", addr)
	}
	return r.Table, nil
}

// refresh looks for a table newer than version have, at hint first, then at
// the seeds, then at every node the current table names.  callers that
// arrive while one is looking wait for it and reuse what it found.
func (c *Client) refresh(ctx context.Context, hint string, have uint64) {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()
	cur := c.table.Load()
	if cur != nil && cur.Version > have {
		return
	}
	if time.Since(c.lastRefresh) < c.cfg.RetryBackoff {
		return
	}
	defer func() { c.lastRefresh = time.Now() }()

	addrs := make([]string, 0, len(c.cfg.Seeds)+4)
	if hint != "" {
		addrs = append(addrs, hint)
	}
	addrs = append(addrs, c.cfg.Seeds...)
	if cur != nil {
		for _, n := range cur.Nodes {
			addrs = append(addrs, n.Addr)
		}
	}
	asked := make(map[string]bool, len(addrs))
	for _, addr := range addrs {
		if addr == "" || asked[addr] {
			continue
		}
		asked[addr] = true
		if ctx.Err() != nil {
			return
		}
		tbl, err := c.fetchTable(ctx, addr)
		if err != nil || len(tbl.Shards) == 0 {
			continue
		}
		if cur == nil || tbl.Version > cur.Version {
			c.table.Store(tbl)
			return
		}
	}
}

// do sends one request to the primary of the account's shard.  a broken
// stream, a refused dial and a not-primary answer make it look for a newer
// table and send again, until ctx ends.  every other answer is returned.
func (c *Client) do(ctx context.Context, account uuid.UUID, req *Request) (Response, errs.InternalError) {
	var last error = errors.New("no table")
	for {
		if c.closed.Load() {
			return Response{}, errs.Wrap(0, true, "%w", errClientClosed)
		}
		var have uint64
		hint := ""
		if tbl := c.table.Load(); tbl != nil {
			have = tbl.Version
			if shard := tbl.ShardOf(account); shard >= 0 {
				row := tbl.Shards[shard]
				if addr := tbl.Nodes[row.Primary].Addr; addr != "" {
					resp, err := c.send(ctx, addr, req)
					switch {
					case err != nil:
						last = err
					case resp.Err == nil || resp.Err.NotPrimary == nil:
						return resp, nil
					default:
						last = resp.Err.NotPrimary
						if resp.Err.NotPrimary.Addr != addr {
							hint = resp.Err.NotPrimary.Addr
						}
					}
				} else {
					last = fmt.Errorf("shard %d has no primary", shard)
				}
			}
		}
		if ctx.Err() == nil {
			c.refresh(ctx, hint, have)
		}
		if tbl := c.table.Load(); tbl != nil && tbl.Version > have {
			continue
		}
		select {
		case <-time.After(c.cfg.RetryBackoff):
		case <-ctx.Done():
			return Response{}, errs.Wrap(0, true, "cluster: no answer from a primary: %w: last error: %v", ctx.Err(), last)
		}
	}
}

// internalError is the error of an answered request.
func internalError(e *ResponseError) errs.InternalError {
	return errs.Wrap(0, e.Retryable, "%s", e.Message)
}

// Check implements constraintapi.CapacityManager.
func (c *Client) Check(ctx context.Context, req *constraintapi.CapacityCheckRequest) (*constraintapi.CapacityCheckResponse, errs.UserError, errs.InternalError) {
	resp, err := c.do(ctx, req.AccountID, &Request{
		Op:    OpCheck,
		Check: req,
	})
	if err != nil {
		return nil, nil, err
	}
	if e := resp.Err; e != nil {
		if e.User {
			return nil, errs.WrapUser(0, e.Retryable, "%s", e.Message), nil
		}
		return nil, nil, internalError(e)
	}
	return resp.Check, nil, nil
}

// Acquire implements constraintapi.CapacityManager.
func (c *Client) Acquire(ctx context.Context, req *constraintapi.CapacityAcquireRequest) (*constraintapi.CapacityAcquireResponse, errs.InternalError) {
	resp, err := c.do(ctx, req.AccountID, &Request{
		Op:      OpAcquire,
		Acquire: req,
	})
	if err != nil {
		return nil, err
	}
	if resp.Err != nil {
		return nil, internalError(resp.Err)
	}
	return resp.Acquire, nil
}

// ExtendLease implements constraintapi.CapacityManager.
func (c *Client) ExtendLease(ctx context.Context, req *constraintapi.CapacityExtendLeaseRequest) (*constraintapi.CapacityExtendLeaseResponse, errs.InternalError) {
	resp, err := c.do(ctx, req.AccountID, &Request{
		Op:     OpExtend,
		Extend: req,
	})
	if err != nil {
		return nil, err
	}
	if resp.Err != nil {
		return nil, internalError(resp.Err)
	}
	return resp.Extend, nil
}

// Release implements constraintapi.CapacityManager.
func (c *Client) Release(ctx context.Context, req *constraintapi.CapacityReleaseRequest) (*constraintapi.CapacityReleaseResponse, errs.InternalError) {
	resp, err := c.do(ctx, req.AccountID, &Request{
		Op:      OpRelease,
		Release: req,
	})
	if err != nil {
		return nil, err
	}
	if resp.Err != nil {
		return nil, internalError(resp.Err)
	}
	return resp.Release, nil
}

// semaphore runs one SemaphoreManager call.
func (c *Client) semaphore(ctx context.Context, op Op, args *SemaphoreArgs) (*SemaphoreResult, error) {
	resp, err := c.do(ctx, args.AccountID, &Request{
		Op:        op,
		Semaphore: args,
	})
	if err != nil {
		return nil, err
	}
	if resp.Err != nil {
		return nil, internalError(resp.Err)
	}
	if resp.Semaphore == nil {
		return nil, errs.Wrap(0, false, "cluster: semaphore answer has no body")
	}
	return resp.Semaphore, nil
}

// SetCapacity implements constraintapi.SemaphoreManager.
func (c *Client) SetCapacity(ctx context.Context, accountID uuid.UUID, name, idempotencyKey string, capacity int64) (constraintapi.SetResult, error) {
	res, err := c.semaphore(ctx, OpSetCapacity, &SemaphoreArgs{
		AccountID:      accountID,
		Name:           name,
		IdempotencyKey: idempotencyKey,
		Value:          capacity,
	})
	if err != nil {
		return constraintapi.SetResult{}, err
	}
	return constraintapi.SetResult{
		Applied:  res.Applied,
		Capacity: res.Capacity,
	}, nil
}

// AdjustCapacity implements constraintapi.SemaphoreManager.
func (c *Client) AdjustCapacity(ctx context.Context, accountID uuid.UUID, name, idempotencyKey string, delta int64) (constraintapi.AdjustResult, error) {
	res, err := c.semaphore(ctx, OpAdjustCapacity, &SemaphoreArgs{
		AccountID:      accountID,
		Name:           name,
		IdempotencyKey: idempotencyKey,
		Value:          delta,
	})
	if err != nil {
		return constraintapi.AdjustResult{}, err
	}
	return constraintapi.AdjustResult{
		Applied:  res.Applied,
		Capacity: res.Capacity,
	}, nil
}

// GetCapacity implements constraintapi.SemaphoreManager.
func (c *Client) GetCapacity(ctx context.Context, accountID uuid.UUID, name, evaluatedKeyHash string) (capacity int64, usage int64, err error) {
	res, err := c.semaphore(ctx, OpGetCapacity, &SemaphoreArgs{
		AccountID:        accountID,
		Name:             name,
		EvaluatedKeyHash: evaluatedKeyHash,
	})
	if err != nil {
		return 0, 0, err
	}
	return res.Capacity, res.Usage, nil
}

// ReleaseSemaphore implements constraintapi.SemaphoreManager.
func (c *Client) ReleaseSemaphore(ctx context.Context, accountID uuid.UUID, name, evaluatedKeyHash, idempotencyKey string, weight int64) error {
	_, err := c.semaphore(ctx, OpReleaseSemaphore, &SemaphoreArgs{
		AccountID:        accountID,
		Name:             name,
		EvaluatedKeyHash: evaluatedKeyHash,
		IdempotencyKey:   idempotencyKey,
		Value:            weight,
	})
	return err
}
