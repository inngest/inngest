package duckdb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"
)

// quackClientVersion and quackClientPlatform are sent in the ConnectionRequest
// handshake purely for server-side diagnostics; the server doesn't gate on
// them beyond the min/max quack protocol version fields.
const quackClientVersion = "inngest-duckdb-quack-client 0.0.1"

func quackClientPlatform() string { return runtime.GOOS + "/" + runtime.GOARCH }

// supportedQuackVersion is the only quack protocol version this client
// speaks. Pinned to what shipped with DuckDB v1.5.5 ("Variegata") — DuckDB's
// own docs say the wire format may still change before it stabilizes with
// DuckDB 2.0, so a version mismatch here should fail loudly rather than
// silently misparse a newer server's responses.
const supportedQuackVersion = 1

func newQuackHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

// quackSession implements sqlExecer (see conn.go) over DuckDB's quack wire
// protocol instead of the stdio/JSON-lines transport in rows.go. It holds one
// server-assigned connection id for its lifetime; process.go's restart-on-
// failure handling (not this type) is what recovers from a lost session, by
// discarding it and re-handshaking against a freshly bootstrapped subprocess.
// exec pages a large result via FetchRequest/FetchResponse internally, but
// that's within one exec call — it doesn't survive across a lost/discarded
// session, since a fresh session has no result_uuid to resume against.
type quackSession struct {
	httpClient   *http.Client
	endpoint     string
	connectionID string
}

// newQuackSession performs the ConnectionRequest handshake against
// listenURL (as reported by `CALL quack_serve(...)`, e.g.
// "http://127.0.0.1:9494") and returns a session ready for exec.
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

// exec implements sqlExecer. A statement DuckDB itself rejected (bad SQL, a
// missing table, a constraint violation) comes back wrapped in
// errStatementFailed, matching rows.go's session.exec so process.exec's
// restart-vs-surface classification (see process.go) works identically
// regardless of which transport is in use: that's not a sign the subprocess
// is unhealthy, and an identical retry fails identically either way. Any
// other error (HTTP failure, malformed response) is left unwrapped, which
// process.exec treats as a dead subprocess warranting a restart.
//
// A result too large for one inline PrepareResponse (needsMoreFetch) is
// paged in via a FetchRequest/FetchResponse loop, keyed off the
// PrepareResponse's own result_uuid, until a response comes back with zero
// chunks — see decodeQuackFetchResponseBody's doc comment for why that's the
// only "done" signal the wire format gives.
//
// cols is the PrepareResponse's own result_names, in the query's own
// left-to-right order — unlike rows.go's session.exec, this is populated even
// when the query returns zero rows, since quack decodes it from response
// metadata rather than from a data row.
func (s *quackSession) exec(ctx context.Context, sqlText string) (cols []string, rows []map[string]any, err error) {
	hdr, r, err := s.send(ctx, encodeQuackPrepareRequest(s.connectionID, sqlText))
	if err != nil {
		return nil, nil, err
	}

	if hdr.Type == quackMsgErrorResponse {
		return nil, nil, decodeQuackStatementError(r)
	}
	if hdr.Type != quackMsgPrepareResponse {
		return nil, nil, fmt.Errorf("duckdb: quack: unexpected response message type %d", hdr.Type)
	}

	cols, rows, needsMoreFetch, resultUUID, err := decodeQuackPrepareResponseBody(r)
	if err != nil {
		return nil, nil, fmt.Errorf("duckdb: quack: decoding prepare response: %w", err)
	}

	for needsMoreFetch {
		hdr, r, err := s.send(ctx, encodeQuackFetchRequest(s.connectionID, resultUUID))
		if err != nil {
			return nil, nil, err
		}
		if hdr.Type == quackMsgErrorResponse {
			return nil, nil, decodeQuackStatementError(r)
		}
		if hdr.Type != quackMsgFetchResponse {
			return nil, nil, fmt.Errorf("duckdb: quack: unexpected response message type %d", hdr.Type)
		}

		fetchedRows, chunkCount, ferr := decodeQuackFetchResponseBody(r, cols)
		if ferr != nil {
			return nil, nil, fmt.Errorf("duckdb: quack: decoding fetch response: %w", ferr)
		}
		rows = append(rows, fetchedRows...)
		needsMoreFetch = chunkCount > 0
	}
	return cols, rows, nil
}

// decodeQuackStatementError decodes an ErrorResponse body into an error
// wrapped in errStatementFailed — the server responded successfully, just
// with a rejection (bad SQL, a closed/expired result on a stale Fetch), not
// a transport or subprocess failure.
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
