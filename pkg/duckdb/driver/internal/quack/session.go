package quack

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/result"
	"github.com/inngest/inngest/pkg/logger"
)

// clientVersion and clientPlatform are sent in the ConnectionRequest
// handshake for server-side diagnostics only; the server doesn't gate on them.
const clientVersion = "inngest-duckdb-quack-client 0.0.1"

func clientPlatform() string { return runtime.GOOS + "/" + runtime.GOARCH }

// supportedVersion is the only quack protocol version this client
// speaks. A mismatch fails loudly rather than risk silently misparsing a
// newer server's responses, since DuckDB's wire format may still change
// before it stabilizes.
const supportedVersion = 3

// heartbeatTimeoutSeconds is sent as every ConnectionRequest's
// heartbeat_timeout (required as of protocol version 3; omitting it defaults
// to 0, which the server rejects as out of range). This client has no
// keepalive of its own and a session may sit idle indefinitely between exec
// calls, so it requests the server's maximum.
const heartbeatTimeoutSeconds = 9223372036854775

// newHTTPClient has no Timeout of its own: every request already goes
// out via http.NewRequestWithContext(ctx, ...) in send(), so the caller's own
// context is the sole timeout authority. A fixed client-level Timeout would
// double-constrain that — and did, in practice: a 30s cap here killed a
// ducklake_merge_adjacent_files compaction call on a large (1M-row,
// 200-partition) lake with "context deadline exceeded" well before the
// caller's own context did, even though compaction is expected to take
// longer as data volume grows.
func newHTTPClient() *http.Client {
	return &http.Client{}
}

// cancelRequestTimeout bounds the CancelRequest query.watchForCancel
// fires once a caller's ctx ends mid-statement — a fresh, short-lived
// context detached from that (already-done) ctx, since the request needs to
// go out regardless of why the caller gave up.
const cancelRequestTimeout = 5 * time.Second

// Session implements sqlExecer (see conn.go) over DuckDB's quack wire
// protocol instead of the stdio/JSON-lines transport in rows.go. It holds one
// server-assigned connection id for its lifetime; process.go recovers from a
// lost session by discarding it and re-handshaking against a fresh subprocess.
type Session struct {
	httpClient   *http.Client
	endpoint     string
	connectionID string
}

// NewSession performs the ConnectionRequest handshake against listenURL
// (as reported by `CALL quack_serve(...)`) and returns a session ready for exec.
func NewSession(ctx context.Context, listenURL, token string) (*Session, error) {
	s := &Session{
		httpClient: newHTTPClient(),
		endpoint:   listenURL + "/quack",
	}

	req := connectionRequest{
		AuthString:               token,
		ClientDuckDBVersion:      clientVersion,
		ClientPlatform:           clientPlatform(),
		MinSupportedQuackVersion: supportedVersion,
		MaxSupportedQuackVersion: supportedVersion,
		HeartbeatTimeoutSeconds:  heartbeatTimeoutSeconds,
	}
	hdr, r, err := s.send(ctx, req.encode())
	if err != nil {
		return nil, fmt.Errorf("duckdb: quack handshake: %w", err)
	}
	if hdr.Type == msgErrorResponse {
		msg, derr := decodeErrorResponseBody(r)
		if derr != nil {
			return nil, fmt.Errorf("duckdb: quack handshake: server returned an error this client could not parse: %w", derr)
		}
		return nil, fmt.Errorf("duckdb: quack handshake rejected: %s", msg)
	}
	if hdr.Type != msgConnectionResponse {
		return nil, fmt.Errorf("duckdb: quack handshake: unexpected response message type %d", hdr.Type)
	}
	resp, err := decodeConnectionResponseBody(r)
	if err != nil {
		return nil, fmt.Errorf("duckdb: quack handshake: %w", err)
	}
	if resp.Version != supportedVersion {
		return nil, fmt.Errorf("duckdb: quack server negotiated protocol version %d, but this client only supports version %d", resp.Version, supportedVersion)
	}

	s.connectionID = hdr.ConnectionID
	logger.StdlibLogger(ctx).Debug("duckdb: quack connection opened",
		"transport", "quack", "phase", "connect", "connection_id", s.connectionID, "endpoint", s.endpoint)
	return s, nil
}

// Exec implements sqlExecer as query without the type-name conversion, for
// ExecContext callers (DDL/INSERT/health checks) that don't need column types.
func (s *Session) Exec(ctx context.Context, sqlText string) (cols []string, rows []result.Row, err error) {
	cols, _, rows, err = s.Query(ctx, sqlText)
	return cols, rows, err
}

// Query implements sqlExecer. A statement DuckDB itself rejected (bad SQL, a
// missing table, a constraint violation) comes back wrapped in
// result.ErrStatementFailed, matching jsonlines.Session.Query so process.Query's
// restart-vs-surface classification works identically across transports. Any
// other error (HTTP failure, malformed response) is left unwrapped, which
// process.Query treats as a dead subprocess warranting a restart.
//
// A result too large for one inline PrepareResponse (needsMoreFetch) is
// paged in via a FetchRequest/FetchResponse loop, keyed off the
// PrepareResponse's result_uuid, until a response comes back with zero
// chunks — see decodeFetchResponseBody for why that's the wire format's
// only "done" signal.
//
// cols and types both come from the PrepareResponse's metadata, populated
// even for zero-row results; unlike the jsonlines transport in rows.go,
// quack never needs a separate DESCRIBE statement to get types.
//
// A caller whose ctx ends mid-statement doesn't just stop waiting: a
// watchForCancel goroutine (started immediately below) fires a real
// CancelRequest for this statement's queryID, which the server turns into
// an actual DuckDB Connection::Interrupt() — see encodeCancelRequest's
// doc comment for the wire format and quack_server.cpp's CANCEL_REQUEST
// handler (verified against the real extension: this is what actually
// stops the abandoned statement from continuing to burn CPU server-side,
// which simply not waiting for the response never did on its own).
func (s *Session) Query(ctx context.Context, sqlText string) (cols []string, types []string, rows []result.Row, err error) {
	queryID := randomHugeint()
	stopCancelWatch := s.watchForCancel(ctx, queryID)
	defer stopCancelWatch()

	// Protocol-level log lines and transport errors carry correlation IDs
	// (connection, query, result, batch) and the phase, never SQL or row
	// data. Statement errors are left as DuckDB reported them, since those
	// are shown to users (Insights) as-is.
	l := logger.StdlibLogger(ctx).With("transport", "quack", "connection_id", s.connectionID, "query_id", queryID.String())
	phaseErr := func(phase string, err error) error {
		return fmt.Errorf("duckdb: quack %s (connection_id=%s query_id=%s): %w", phase, s.connectionID, queryID, err)
	}
	start := time.Now()

	hdr, r, err := s.send(ctx, encodePrepareRequest(s.connectionID, sqlText, queryID))
	if err != nil {
		return nil, nil, nil, phaseErr("prepare", err)
	}

	if hdr.Type == msgErrorResponse {
		return nil, nil, nil, decodeStatementError(r)
	}
	if hdr.Type != msgPrepareResponse {
		return nil, nil, nil, phaseErr("prepare", fmt.Errorf("unexpected response message type %d", hdr.Type))
	}

	cols, logicalTypes, rows, needsMoreFetch, resultUUID, err := decodePrepareResponseBody(r)
	if err != nil {
		return nil, nil, nil, phaseErr("prepare", fmt.Errorf("decoding prepare response: %w", err))
	}
	l.Debug("duckdb: quack prepare complete",
		"phase", "prepare", "rows", len(rows), "needs_more_fetch", needsMoreFetch,
		"result_uuid", resultUUID.String(), "duration", time.Since(start))

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
	// (even concurrent ones on different pooled sessions, see
	// driver.Options.QuackConns) runs its own independent fetch loop, so this
	// needs no synchronization.
	nextBatchIndex := uint64(1)
	fetchCols := result.NewColumns(cols)
	for needsMoreFetch {
		fetchPhase := fmt.Sprintf("fetch (result_uuid=%s batch_index=%d)", resultUUID, nextBatchIndex)
		hdr, r, err := s.send(ctx, encodeFetchRequest(s.connectionID, resultUUID, nextBatchIndex))
		if err != nil {
			return nil, nil, nil, phaseErr(fetchPhase, err)
		}
		if hdr.Type == msgErrorResponse {
			return nil, nil, nil, decodeStatementError(r)
		}
		if hdr.Type != msgFetchResponse {
			return nil, nil, nil, phaseErr(fetchPhase, fmt.Errorf("unexpected response message type %d", hdr.Type))
		}

		fetchedRows, chunkCount, batchIndex, ferr := decodeFetchResponseBody(r, fetchCols)
		if ferr != nil {
			return nil, nil, nil, phaseErr(fetchPhase, fmt.Errorf("decoding fetch response: %w", ferr))
		}
		l.Debug("duckdb: quack fetch complete",
			"phase", "fetch", "result_uuid", resultUUID.String(), "batch_index", batchIndex,
			"rows", len(fetchedRows), "chunks", chunkCount)
		rows = append(rows, fetchedRows...)
		needsMoreFetch = chunkCount > 0
		nextBatchIndex = batchIndex + 1
	}

	types = make([]string, len(logicalTypes))
	for i, lt := range logicalTypes {
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
// encodeCancelRequest), so a stray cancel racing the statement's own
// natural completion — or racing a *new* statement already started on the
// same connection by the time it arrives — can never interrupt the wrong
// query. stop exists purely to avoid the noise/latency of sending a cancel
// nobody needs, not for correctness.
//
// The cancel request itself runs under cancelRequestTimeout, detached
// from ctx (already done by the time this fires) — its own result is
// logged, not returned: query()'s caller already gets ctx's own error back
// regardless of whether the server managed to actually interrupt anything.
func (s *Session) watchForCancel(ctx context.Context, queryID hugeint) (stop func()) {
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			cancelCtx, cancel := context.WithTimeout(context.Background(), cancelRequestTimeout)
			defer cancel()
			hdr, r, err := s.send(cancelCtx, encodeCancelRequest(s.connectionID, queryID))
			l := logger.StdlibLogger(context.WithoutCancel(ctx)).With(
				"transport", "quack", "phase", "cancel", "connection_id", s.connectionID, "query_id", queryID.String())
			switch {
			case err != nil:
				l.Warn("duckdb: quack: failed to send cancel request for an abandoned query", "error", err)
			case hdr.Type == msgErrorResponse:
				msg, _ := decodeErrorResponseBody(r)
				l.Debug("duckdb: quack: cancel request rejected (statement likely already finished)", "message", msg)
			}
		case <-done:
		}
	}()
	return func() { close(done) }
}

// decodeStatementError decodes an ErrorResponse body into an error
// wrapped in result.ErrStatementFailed: the server responded, it just rejected the
// statement — not a transport or subprocess failure.
func decodeStatementError(r *reader) error {
	msg, derr := decodeErrorResponseBody(r)
	if derr != nil {
		return fmt.Errorf("duckdb: quack: server returned an error this client could not parse: %w", derr)
	}
	return fmt.Errorf("%w: %s", result.ErrStatementFailed, msg)
}

// send POSTs one quack message and returns the decoded response header, with
// r positioned at the start of the response body object (see
// decodeMessageHeader).
func (s *Session) send(ctx context.Context, payload []byte) (messageHeader, *reader, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(payload))
	if err != nil {
		return messageHeader{}, nil, fmt.Errorf("duckdb: quack: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/duckdb")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return messageHeader{}, nil, fmt.Errorf("duckdb: quack: request to %s failed: %w", s.endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return messageHeader{}, nil, fmt.Errorf("duckdb: quack: reading response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return messageHeader{}, nil, fmt.Errorf("duckdb: quack: HTTP %d from %s: %s", resp.StatusCode, s.endpoint, body)
	}

	r := newReader(body)
	hdr, err := decodeMessageHeader(r)
	if err != nil {
		return messageHeader{}, nil, fmt.Errorf("duckdb: quack: decoding response header: %w", err)
	}
	// The server currently leaves connection_id empty on every response
	// after the handshake, so only a present-but-different ID is rejected:
	// that would mean this response belongs to some other session.
	if s.connectionID != "" && hdr.ConnectionID != "" && hdr.ConnectionID != s.connectionID {
		return messageHeader{}, nil, fmt.Errorf("duckdb: quack: response for connection %s received on connection %s", hdr.ConnectionID, s.connectionID)
	}
	return hdr, r, nil
}

// ConnectionID is the server-assigned connection id, for correlating logs.
func (s *Session) ConnectionID() string { return s.connectionID }
