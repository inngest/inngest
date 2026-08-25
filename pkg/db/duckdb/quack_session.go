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
// discarding it and re-handshaking against a freshly bootstrapped subprocess
// — see quack_protocol.go's decodeQuackPrepareResponseBody doc comment for
// why FetchRequest/reconnect logic (which a longer-lived client would need)
// isn't implemented here.
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
// regardless of which transport is in use. Any other error (HTTP failure,
// malformed response) is left unwrapped, which process.exec treats as a dead
// subprocess warranting a restart.
func (s *quackSession) exec(ctx context.Context, sqlText string) ([]map[string]any, error) {
	hdr, r, err := s.send(ctx, encodeQuackPrepareRequest(s.connectionID, sqlText))
	if err != nil {
		return nil, err
	}

	if hdr.Type == quackMsgErrorResponse {
		msg, derr := decodeQuackErrorResponseBody(r)
		if derr != nil {
			return nil, fmt.Errorf("duckdb: quack: server returned an error this client could not parse: %w", derr)
		}
		return nil, fmt.Errorf("%w: %s", errStatementFailed, msg)
	}
	if hdr.Type != quackMsgPrepareResponse {
		return nil, fmt.Errorf("duckdb: quack: unexpected response message type %d", hdr.Type)
	}

	rows, needsMoreFetch, err := decodeQuackPrepareResponseBody(r)
	if err != nil {
		return nil, fmt.Errorf("duckdb: quack: decoding prepare response: %w", err)
	}
	if needsMoreFetch {
		return nil, fmt.Errorf("duckdb: quack: result exceeded one inline response and requires FetchRequest, which this client does not implement")
	}
	return rows, nil
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
		return quackMessageHeader{}, nil, fmt.Errorf("duckdb: quack: HTTP %d from %s", resp.StatusCode, s.endpoint)
	}

	r := newQuackReader(body)
	hdr, err := decodeQuackMessageHeader(r)
	if err != nil {
		return quackMessageHeader{}, nil, fmt.Errorf("duckdb: quack: decoding response header: %w", err)
	}
	return hdr, r, nil
}
