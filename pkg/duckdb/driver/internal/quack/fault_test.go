package quack

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/result"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

// fakeQuackServer is an httptest quack endpoint whose prepare/fetch/send-data
// responses each test scripts, so transport faults and interleavings a
// cooperative real server never produces can be injected deterministically.
// It answers the handshake and CancelRequests itself, recording every
// cancelled query ID.
type fakeQuackServer struct {
	t   *testing.T
	srv *httptest.Server

	nextConn atomic.Int64

	mu        sync.Mutex
	cancelled []hugeint
	prepared  []hugeint

	onPrepare  func(w http.ResponseWriter, r *http.Request, sql string, queryID hugeint)
	onFetch    func(w http.ResponseWriter, r *http.Request, batchIndex uint64)
	onSendData func(w http.ResponseWriter, r *http.Request)
}

func newFakeQuackServer(t *testing.T) *fakeQuackServer {
	t.Helper()
	f := &fakeQuackServer{t: t}
	f.srv = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeQuackServer) handle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rd := newReader(body)
	hdr, err := decodeMessageHeader(rd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch hdr.Type {
	case msgConnectionRequest:
		_, _ = w.Write(fakeConnectionResponse(fmt.Sprintf("conn-%d", f.nextConn.Add(1))))

	case msgPrepareRequest:
		var sql string
		if ok, _ := rd.tryBeginProperty(1); ok {
			sql, _ = rd.readString()
		}
		qid := readHugeintField(f.t, rd, 2)
		f.mu.Lock()
		f.prepared = append(f.prepared, qid)
		f.mu.Unlock()
		f.onPrepare(w, r, sql, qid)

	case msgFetchRequest:
		_ = readHugeintField(f.t, rd, 1)
		require.NoError(f.t, rd.beginProperty(2))
		batch, _ := rd.readUInt64()
		f.onFetch(w, r, batch)

	case msgSendDataRequest:
		f.onSendData(w, r)

	case msgCancelRequest:
		qid := readHugeintField(f.t, rd, 1)
		f.mu.Lock()
		f.cancelled = append(f.cancelled, qid)
		f.mu.Unlock()
		_, _ = w.Write(encodeMessage(msgSuccessResponse, "", func(*writer) {}))

	default:
		http.Error(w, "unexpected message type", http.StatusBadRequest)
	}
}

func (f *fakeQuackServer) session(t *testing.T) *Session {
	t.Helper()
	s, err := NewSession(t.Context(), f.srv.URL, "token")
	require.NoError(t, err)
	return s
}

func (f *fakeQuackServer) cancelledIDs() []hugeint {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]hugeint(nil), f.cancelled...)
}

func (f *fakeQuackServer) lastPrepared() hugeint {
	f.mu.Lock()
	defer f.mu.Unlock()
	require.NotEmpty(f.t, f.prepared)
	return f.prepared[len(f.prepared)-1]
}

func readHugeintField(t *testing.T, r *reader, id uint16) hugeint {
	require.NoError(t, r.beginProperty(id))
	hi, err := r.readSignedLeb128()
	require.NoError(t, err)
	lo, err := r.readUnsignedLeb128()
	require.NoError(t, err)
	return hugeint{hi: hi, lo: lo}
}

func fakeConnectionResponse(connID string) []byte {
	w := &writer{}
	w.beginObject()
	w.writeByte(1, msgConnectionResponse)
	w.writeString(2, connID)
	w.writeUint64(3, optionalIdxInvalid)
	w.endObject()
	w.beginObject()
	w.writeStringDefault(1, "fake")
	w.writeStringDefault(2, "fake")
	w.writeUint64Default(3, supportedVersion)
	w.endObject()
	return w.bytes()
}

// fakePrepareResponse is a one-column INTEGER "v" result holding value,
// optionally announcing that more batches must be fetched.
func fakePrepareResponse(t *testing.T, connID string, value int32, needsMore bool) []byte {
	w := &writer{}
	w.beginObject()
	w.writeByte(1, msgPrepareResponse)
	w.writeStringDefault(2, connID)
	w.writeUint64(3, optionalIdxInvalid)
	w.endObject()

	w.beginObject()
	w.writeFieldID(1)
	w.beginList(1)
	w.beginObject()
	w.writeByte(100, 13) // INTEGER
	w.endObject()
	w.writeFieldID(2)
	w.beginList(1)
	w.writeUnsignedLeb128(uint64(len("v")))
	w.buf.WriteString("v")
	w.writeBool(3, needsMore)
	w.writeFieldID(4)
	w.beginList(1)
	w.buf.WriteByte(1)
	w.beginObject()
	w.writeFieldID(300)
	w.buf.Write(buildFlatIntegerChunk(t, value))
	w.endObject()
	w.writeFieldID(5)
	w.writeSignedLeb128(1)
	w.writeUnsignedLeb128(2)
	w.endObject()
	return w.bytes()
}

// fakeFetchResponse delivers batchIndex with one chunk holding value, or,
// when done, the zero-chunk response that ends the fetch loop.
func fakeFetchResponse(t *testing.T, batchIndex uint64, value int32, done bool) []byte {
	w := &writer{}
	w.beginObject()
	w.writeByte(1, msgFetchResponse)
	w.writeUint64(3, optionalIdxInvalid)
	w.endObject()
	w.beginObject()
	if !done {
		w.writeUint64(1, 1)
	}
	w.writeUint64(3, batchIndex)
	w.endObject()
	if !done {
		w.buf.Write(buildFlatIntegerChunk(t, value))
	}
	return w.bytes()
}

// blockUntilClientGone parks a handler until the client abandons the request
// (or a generous safety timeout), simulating a statement still running.
func blockUntilClientGone(r *http.Request) {
	select {
	case <-r.Context().Done():
	case <-time.After(10 * time.Second):
	}
}

// verifyNoQuackLeaks asserts query()'s cancel watcher and HTTP plumbing
// have all exited, once idle keep-alive connections are dropped.
func verifyNoQuackLeaks(t *testing.T, opts ...goleak.Option) {
	t.Helper()
	http.DefaultTransport.(*http.Transport).CloseIdleConnections()
	goleak.VerifyNone(t, opts...)
}

func requireTransportError(t *testing.T, err error, wantSubstrings ...string) {
	t.Helper()
	require.Error(t, err)
	require.NotErrorIs(t, err, result.ErrStatementFailed, "a transport fault must not be classified as a statement error")
	for _, s := range wantSubstrings {
		require.Contains(t, err.Error(), s)
	}
}

func TestQuackFaultTruncatedPrepareResponse(t *testing.T) {
	f := newFakeQuackServer(t)
	f.onPrepare = func(w http.ResponseWriter, _ *http.Request, _ string, _ hugeint) {
		full := fakePrepareResponse(t, "", 1, false)
		_, _ = w.Write(full[:len(full)/2])
	}
	s := f.session(t)

	_, _, _, err := s.Query(t.Context(), "SELECT 1")
	requireTransportError(t, err, "prepare", "connection_id="+s.connectionID, "query_id="+f.lastPrepared().String())
}

func TestQuackFaultUnexpectedPrepareMessageType(t *testing.T) {
	f := newFakeQuackServer(t)
	f.onPrepare = func(w http.ResponseWriter, _ *http.Request, _ string, _ hugeint) {
		_, _ = w.Write(fakeFetchResponse(t, 1, 1, false))
	}
	s := f.session(t)

	_, _, _, err := s.Query(t.Context(), "SELECT 1")
	requireTransportError(t, err, "prepare", "unexpected response message type")
}

func TestQuackFaultHTTPErrorOnPrepare(t *testing.T) {
	f := newFakeQuackServer(t)
	f.onPrepare = func(w http.ResponseWriter, _ *http.Request, _ string, _ hugeint) {
		w.WriteHeader(http.StatusInternalServerError)
	}
	s := f.session(t)

	_, _, _, err := s.Query(t.Context(), "SELECT 1")
	requireTransportError(t, err, "prepare", "HTTP 500")
}

func TestQuackFaultResponseForAnotherConnection(t *testing.T) {
	f := newFakeQuackServer(t)
	f.onPrepare = func(w http.ResponseWriter, _ *http.Request, _ string, _ hugeint) {
		_, _ = w.Write(fakePrepareResponse(t, "someone-else", 1, false))
	}
	s := f.session(t)

	_, _, _, err := s.Query(t.Context(), "SELECT 1")
	requireTransportError(t, err, "received on connection "+s.connectionID)
}

func TestQuackFaultTruncatedFetchResponse(t *testing.T) {
	f := newFakeQuackServer(t)
	f.onPrepare = func(w http.ResponseWriter, _ *http.Request, _ string, _ hugeint) {
		_, _ = w.Write(fakePrepareResponse(t, "", 1, true))
	}
	f.onFetch = func(w http.ResponseWriter, _ *http.Request, batch uint64) {
		full := fakeFetchResponse(t, batch, 2, false)
		_, _ = w.Write(full[:len(full)-3])
	}
	s := f.session(t)

	_, _, _, err := s.Query(t.Context(), "SELECT 1")
	requireTransportError(t, err, "fetch", "batch_index=1")
}

func TestQuackFaultMultiBatchFetchHappyPath(t *testing.T) {
	f := newFakeQuackServer(t)
	f.onPrepare = func(w http.ResponseWriter, _ *http.Request, _ string, _ hugeint) {
		_, _ = w.Write(fakePrepareResponse(t, "", 1, true))
	}
	f.onFetch = func(w http.ResponseWriter, _ *http.Request, batch uint64) {
		_, _ = w.Write(fakeFetchResponse(t, batch, int32(batch+1), batch >= 3))
	}
	s := f.session(t)

	_, _, rows, err := s.Query(t.Context(), "SELECT 1")
	require.NoError(t, err)
	var got []any
	for _, r := range rows {
		got = append(got, r.Get("v"))
	}
	require.Equal(t, []any{int64(1), int64(2), int64(3)}, got)
}

// TestQuackFaultCancelDuringPrepare: a caller's ctx ending while the server
// is still running the statement must send a CancelRequest for exactly that
// statement's query ID, return promptly with the ctx error, and leave no
// watcher goroutine or connection behind.
func TestQuackFaultCancelDuringPrepare(t *testing.T) {
	ignore := goleak.IgnoreCurrent()
	f := newFakeQuackServer(t)
	f.onPrepare = func(_ http.ResponseWriter, r *http.Request, _ string, _ hugeint) {
		blockUntilClientGone(r)
	}
	s := f.session(t)

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, _, _, err := s.Query(ctx, "SELECT slow")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(start), 5*time.Second)

	want := f.lastPrepared()
	require.Eventually(t, func() bool {
		ids := f.cancelledIDs()
		return len(ids) == 1 && ids[0] == want
	}, 5*time.Second, 10*time.Millisecond, "exactly the abandoned query must be cancelled")

	f.srv.Close()
	verifyNoQuackLeaks(t, ignore)
}

// TestQuackFaultCancelBetweenFetchesThenReuse: cancellation mid-fetch-loop
// cancels the right query, and the same session is healthy afterwards.
func TestQuackFaultCancelBetweenFetchesThenReuse(t *testing.T) {
	f := newFakeQuackServer(t)
	var blockFetch atomic.Bool
	blockFetch.Store(true)
	f.onPrepare = func(w http.ResponseWriter, _ *http.Request, sql string, _ hugeint) {
		_, _ = w.Write(fakePrepareResponse(t, "", 1, sql == "SELECT paged"))
	}
	f.onFetch = func(w http.ResponseWriter, r *http.Request, batch uint64) {
		if batch >= 2 && blockFetch.Load() {
			blockUntilClientGone(r)
			return
		}
		_, _ = w.Write(fakeFetchResponse(t, batch, 2, batch >= 2))
	}
	s := f.session(t)

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	_, _, _, err := s.Query(ctx, "SELECT paged")
	require.ErrorIs(t, err, context.Canceled)
	require.Contains(t, err.Error(), "fetch")

	want := f.lastPrepared()
	require.Eventually(t, func() bool {
		ids := f.cancelledIDs()
		return len(ids) == 1 && ids[0] == want
	}, 5*time.Second, 10*time.Millisecond)

	_, _, rows, err := s.Query(t.Context(), "SELECT 1")
	require.NoError(t, err, "the session must be reusable after a cancelled statement")
	require.Len(t, rows, 1)
	require.Len(t, f.cancelledIDs(), 1, "a completed statement must not send a cancel")
}

// TestQuackFaultConcurrentSessionsGetTheirOwnResults runs interleaved,
// out-of-order-completing statements across several sessions and checks
// every caller gets its own statement's result.
func TestQuackFaultConcurrentSessionsGetTheirOwnResults(t *testing.T) {
	f := newFakeQuackServer(t)
	f.onPrepare = func(w http.ResponseWriter, _ *http.Request, sql string, _ hugeint) {
		n, err := strconv.Atoi(strings.TrimPrefix(sql, "SELECT "))
		require.NoError(t, err)
		// Later statements finish first.
		time.Sleep(time.Duration(40-n) * time.Millisecond)
		_, _ = w.Write(fakePrepareResponse(t, "", int32(n), false))
	}
	sessions := []*Session{f.session(t), f.session(t), f.session(t)}

	var wg sync.WaitGroup
	errs := make(chan error, 40)
	for i := range 40 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, rows, err := sessions[i%len(sessions)].Query(t.Context(), fmt.Sprintf("SELECT %d", i))
			if err == nil && (len(rows) != 1 || rows[0].Get("v") != int64(i)) {
				err = fmt.Errorf("statement %d got %v", i, rows)
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.Empty(t, f.cancelledIDs())
}

func TestQuackFaultSendDataTruncatedAndHTTPError(t *testing.T) {
	f := newFakeQuackServer(t)
	s := f.session(t)
	payload := encodeSendDataRequest(s.connectionID, "stream", 0, nil, nil, nil)

	f.onSendData = func(w http.ResponseWriter, _ *http.Request) {
		full := encodeMessage(msgSendDataResponse, "", func(*writer) {})
		_, _ = w.Write(full[:len(full)-1])
	}
	requireTransportError(t, s.sendDataRequest(t.Context(), payload))

	f.onSendData = func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}
	requireTransportError(t, s.sendDataRequest(t.Context(), payload), "HTTP 502")

	f.onSendData = func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(fakeFetchResponse(t, 1, 1, true))
	}
	requireTransportError(t, s.sendDataRequest(t.Context(), payload), "unexpected response message type")
}
