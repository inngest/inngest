package duckdb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/inngest/inngest/pkg/logger"
)

// quackClientVersion and quackClientPlatform are sent in the ConnectionRequest
// handshake for server-side diagnostics only; the server doesn't gate on them.
const quackClientVersion = "inngest-duckdb-quack-client 0.0.1"

func quackClientPlatform() string { return runtime.GOOS + "/" + runtime.GOARCH }

// supportedQuackVersion is the only quack protocol version this client
// speaks. A mismatch fails loudly rather than risk silently misparsing a
// newer server's responses, since DuckDB's wire format may still change
// before it stabilizes.
const supportedQuackVersion = 3

// quackHeartbeatTimeoutSeconds is sent as every ConnectionRequest's
// heartbeat_timeout (required as of protocol version 3; omitting it defaults
// to 0, which the server rejects as out of range). This client has no
// keepalive of its own and a session may sit idle indefinitely between exec
// calls, so it requests the server's maximum.
const quackHeartbeatTimeoutSeconds = 9223372036854775

// newQuackHTTPClient has no Timeout of its own: every request already goes
// out via http.NewRequestWithContext(ctx, ...) in send(), so the caller's own
// context is the sole timeout authority. A fixed client-level Timeout would
// double-constrain that — and did, in practice: a 30s cap here killed a
// ducklake_merge_adjacent_files compaction call on a large (1M-row,
// 200-partition) lake with "context deadline exceeded" well before the
// caller's own context did, even though compaction is expected to take
// longer as data volume grows.
func newQuackHTTPClient() *http.Client {
	return &http.Client{}
}

// quackCancelRequestTimeout bounds the CancelRequest query.watchForCancel
// fires once a caller's ctx ends mid-statement — a fresh, short-lived
// context detached from that (already-done) ctx, since the request needs to
// go out regardless of why the caller gave up.
const quackCancelRequestTimeout = 5 * time.Second

// quackSession implements sqlExecer (see conn.go) over DuckDB's quack wire
// protocol instead of the stdio/JSON-lines transport in rows.go. It holds one
// server-assigned connection id for its lifetime; process.go recovers from a
// lost session by discarding it and re-handshaking against a fresh subprocess.
type quackSession struct {
	httpClient   *http.Client
	endpoint     string
	connectionID string
}

// newQuackSession performs the ConnectionRequest handshake against listenURL
// (as reported by `CALL quack_serve(...)`) and returns a session ready for exec.
func newQuackSession(ctx context.Context, listenURL, token string) (*quackSession, error) {
	s := &quackSession{
		httpClient: newQuackHTTPClient(),
		endpoint:   listenURL + "/quack",
	}

	req := quackConnectionRequest{
		AuthString:               token,
		ClientDuckDBVersion:      quackClientVersion,
		ClientPlatform:           quackClientPlatform(),
		MinSupportedQuackVersion: supportedQuackVersion,
		MaxSupportedQuackVersion: supportedQuackVersion,
		HeartbeatTimeoutSeconds:  quackHeartbeatTimeoutSeconds,
	}
	hdr, r, err := s.send(ctx, req.encode())
	if err != nil {
		return nil, fmt.Errorf("duckdb: quack handshake: %w", err)
	}
	if hdr.Type == quackMsgErrorResponse {
		msg, derr := decodeQuackErrorResponseBody(r)
		if derr != nil {
			return nil, fmt.Errorf("duckdb: quack handshake: server returned an error this client could not parse: %w", derr)
		}
		return nil, fmt.Errorf("duckdb: quack handshake rejected: %s", msg)
	}
	if hdr.Type != quackMsgConnectionResponse {
		return nil, fmt.Errorf("duckdb: quack handshake: unexpected response message type %d", hdr.Type)
	}
	resp, err := decodeQuackConnectionResponseBody(r)
	if err != nil {
		return nil, fmt.Errorf("duckdb: quack handshake: %w", err)
	}
	if resp.QuackVersion != supportedQuackVersion {
		return nil, fmt.Errorf("duckdb: quack server negotiated protocol version %d, but this client only supports version %d", resp.QuackVersion, supportedQuackVersion)
	}

	s.connectionID = hdr.ConnectionID
	return s, nil
}

// exec implements sqlExecer as query without the type-name conversion, for
// ExecContext callers (DDL/INSERT/health checks) that don't need column types.
func (s *quackSession) exec(ctx context.Context, sqlText string) (cols []string, rows []map[string]any, err error) {
	cols, _, rows, err = s.query(ctx, sqlText)
	return cols, rows, err
}

// query implements sqlExecer. A statement DuckDB itself rejected (bad SQL, a
// missing table, a constraint violation) comes back wrapped in
// errStatementFailed, matching rows.go's session.query so process.query's
// restart-vs-surface classification works identically across transports. Any
// other error (HTTP failure, malformed response) is left unwrapped, which
// process.query treats as a dead subprocess warranting a restart.
//
// A result too large for one inline PrepareResponse (needsMoreFetch) is
// paged in via a FetchRequest/FetchResponse loop, keyed off the
// PrepareResponse's result_uuid, until a response comes back with zero
// chunks — see decodeQuackFetchResponseBody for why that's the wire format's
// only "done" signal.
//
// cols and types both come from the PrepareResponse's metadata, populated
// even for zero-row results; unlike the jsonlines transport in rows.go,
// quack never needs a separate DESCRIBE statement to get types.
//
// A caller whose ctx ends mid-statement doesn't just stop waiting: a
// watchForCancel goroutine (started immediately below) fires a real
// CancelRequest for this statement's queryID, which the server turns into
// an actual DuckDB Connection::Interrupt() — see encodeQuackCancelRequest's
// doc comment for the wire format and quack_server.cpp's CANCEL_REQUEST
// handler (verified against the real extension: this is what actually
// stops the abandoned statement from continuing to burn CPU server-side,
// which simply not waiting for the response never did on its own).
func (s *quackSession) query(ctx context.Context, sqlText string) (cols []string, types []string, rows []map[string]any, err error) {
	queryID := randomQuackHugeint()
	stopCancelWatch := s.watchForCancel(ctx, queryID)
	defer stopCancelWatch()

	hdr, r, err := s.send(ctx, encodeQuackPrepareRequest(s.connectionID, sqlText, queryID))
	if err != nil {
		return nil, nil, nil, err
	}

	if hdr.Type == quackMsgErrorResponse {
		return nil, nil, nil, decodeQuackStatementError(r)
	}
	if hdr.Type != quackMsgPrepareResponse {
		return nil, nil, nil, fmt.Errorf("duckdb: quack: unexpected response message type %d", hdr.Type)
	}

	cols, quackTypes, rows, needsMoreFetch, resultUUID, err := decodeQuackPrepareResponseBody(r)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("duckdb: quack: decoding prepare response: %w", err)
	}

	// nextBatchIndex starts at 1, not 0: confirmed by reading the real quack
	// extension binary's disassembly (no source is vendored in this repo) —
	// the server deserializes batch_index as a plain optional int with no
	// separate "was it present" flag surviving past deserialization, so a
	// present-but-zero value and an absent value both collapse to the same
	// in-memory 0, and the validation that produces "FETCH_REQUEST is
	// missing its batch index" treats stored-zero as "missing" regardless
	// of which case it actually was. 0 is therefore never a usable request
	// value, confirmed empirically (still fails with a constant or
	// incrementing-from-0 index; succeeds starting from 1). Advances to
	// whatever batchIndex the previous FetchResponse says it just
	// delivered, plus one — trusting the server's own count rather than
	// assuming this loop's iteration number always matches it.
	// Function-local and never shared across goroutines: each query() call
	// (even concurrent ones on different pooled quackSessions, see
	// Options.QuackConns) runs its own independent fetch loop, so this
	// needs no synchronization.
	nextBatchIndex := uint64(1)
	for needsMoreFetch {
		hdr, r, err := s.send(ctx, encodeQuackFetchRequest(s.connectionID, resultUUID, nextBatchIndex))
		if err != nil {
			return nil, nil, nil, err
		}
		if hdr.Type == quackMsgErrorResponse {
			return nil, nil, nil, decodeQuackStatementError(r)
		}
		if hdr.Type != quackMsgFetchResponse {
			return nil, nil, nil, fmt.Errorf("duckdb: quack: unexpected response message type %d", hdr.Type)
		}

		fetchedRows, chunkCount, batchIndex, ferr := decodeQuackFetchResponseBody(r, cols)
		if ferr != nil {
			return nil, nil, nil, fmt.Errorf("duckdb: quack: decoding fetch response: %w", ferr)
		}
		rows = append(rows, fetchedRows...)
		needsMoreFetch = chunkCount > 0
		nextBatchIndex = batchIndex + 1
	}

	types = make([]string, len(quackTypes))
	for i, lt := range quackTypes {
		types[i] = lt.typeName()
	}
	return cols, types, rows, nil
}

// watchForCancel starts a goroutine that sends a CancelRequest for queryID
// as soon as ctx ends, and returns a stop func the caller must defer
// immediately (before ctx can end) to retire that goroutine once query()'s
// own request(s) are done — success or any other error. Without stop, the
// goroutine would leak for the life of ctx (which, for a long-lived
// context, is far longer than this one statement).
//
// Sending the cancel late (after query() already returned) is harmless, not
// just wasteful: the server rejects a CancelRequest whose query_uuid
// doesn't match whatever is currently running on the connection (see
// encodeQuackCancelRequest), so a stray cancel racing the statement's own
// natural completion — or racing a *new* statement already started on the
// same connection by the time it arrives — can never interrupt the wrong
// query. stop exists purely to avoid the noise/latency of sending a cancel
// nobody needs, not for correctness.
//
// The cancel request itself runs under quackCancelRequestTimeout, detached
// from ctx (already done by the time this fires) — its own result is
// logged, not returned: query()'s caller already gets ctx's own error back
// regardless of whether the server managed to actually interrupt anything.
func (s *quackSession) watchForCancel(ctx context.Context, queryID quackHugeint) (stop func()) {
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			cancelCtx, cancel := context.WithTimeout(context.Background(), quackCancelRequestTimeout)
			defer cancel()
			hdr, r, err := s.send(cancelCtx, encodeQuackCancelRequest(s.connectionID, queryID))
			l := logger.StdlibLogger(context.WithoutCancel(ctx))
			switch {
			case err != nil:
				l.Warn("duckdb: quack: failed to send cancel request for an abandoned query", "error", err)
			case hdr.Type == quackMsgErrorResponse:
				msg, _ := decodeQuackErrorResponseBody(r)
				l.Debug("duckdb: quack: cancel request rejected (statement likely already finished)", "message", msg)
			}
		case <-done:
		}
	}()
	return func() { close(done) }
}

// decodeQuackStatementError decodes an ErrorResponse body into an error
// wrapped in errStatementFailed: the server responded, it just rejected the
// statement — not a transport or subprocess failure.
func decodeQuackStatementError(r *quackReader) error {
	msg, derr := decodeQuackErrorResponseBody(r)
	if derr != nil {
		return fmt.Errorf("duckdb: quack: server returned an error this client could not parse: %w", derr)
	}
	return fmt.Errorf("%w: %s", errStatementFailed, msg)
}

// send POSTs one quack message and returns the decoded response header, with
// r positioned at the start of the response body object (see
// decodeQuackMessageHeader).
func (s *quackSession) send(ctx context.Context, payload []byte) (quackMessageHeader, *quackReader, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(payload))
	if err != nil {
		return quackMessageHeader{}, nil, fmt.Errorf("duckdb: quack: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/duckdb")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return quackMessageHeader{}, nil, fmt.Errorf("duckdb: quack: request to %s failed: %w", s.endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return quackMessageHeader{}, nil, fmt.Errorf("duckdb: quack: reading response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return quackMessageHeader{}, nil, fmt.Errorf("duckdb: quack: HTTP %d from %s: %s", resp.StatusCode, s.endpoint, body)
	}

	r := newQuackReader(body)
	hdr, err := decodeQuackMessageHeader(r)
	if err != nil {
		return quackMessageHeader{}, nil, fmt.Errorf("duckdb: quack: decoding response header: %w", err)
	}
	return hdr, r, nil
}
