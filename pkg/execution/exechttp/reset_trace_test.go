package exechttp

import (
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/http/httptrace"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

type resetRoundTripper func(*http.Request) (*http.Response, error)

func (f resetRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type resetBody struct{}

func (resetBody) Read([]byte) (int, error) { return 0, syscall.ECONNRESET }
func (resetBody) Close() error             { return nil }

func TestResetTracePreservesError(t *testing.T) {
	for _, tc := range []struct {
		name string
		body io.ReadCloser
		err  error
		want error
	}{
		{name: "transport reset", err: syscall.ECONNRESET, want: syscall.ECONNRESET},
		{name: "response body reset", body: resetBody{}, want: ErrUnexpectedEnd},
		{name: "unexpected EOF", err: io.ErrUnexpectedEOF, want: io.ErrUnexpectedEOF},
	} {
		t.Run(tc.name, func(t *testing.T) {
			transport := resetRoundTripper(func(req *http.Request) (*http.Response, error) {
				trace := httptrace.ContextClientTrace(req.Context())
				trace.TLSHandshakeStart()
				trace.TLSHandshakeDone(tls.ConnectionState{}, nil)
				trace.WroteRequest(httptrace.WroteRequestInfo{})
				if tc.body != nil {
					trace.GotFirstResponseByte()
					return &http.Response{StatusCode: 200, Body: tc.body}, nil
				}
				return nil, tc.err
			})
			req, err := NewRequest(http.MethodPost, "https://example.com", nil)
			require.NoError(t, err)
			client := ExtendedClient{Client: &http.Client{Transport: transport}}
			resp, original := client.DoRequest(t.Context(), req)
			require.Nil(t, resp)
			require.ErrorIs(t, original, tc.want)

			WithResetTrace()(&client)
			resp, err = client.DoRequest(t.Context(), req)
			require.Nil(t, resp)
			require.ErrorIs(t, err, tc.want)
			require.EqualError(t, err, original.Error())
			var reset *ResetTraceError
			if tc.want == io.ErrUnexpectedEOF {
				require.False(t, errors.As(err, &reset))
				return
			}
			require.ErrorAs(t, err, &reset)
			require.ErrorIs(t, reset.Cause, syscall.ECONNRESET)
			require.True(t, reset.Snapshot.TLSCompleted)
			require.True(t, reset.Snapshot.RequestWritten)
			require.Equal(t, tc.body != nil, reset.Snapshot.FirstByte)
			require.Equal(t, errors.Is(original, syscall.ECONNRESET), errors.Is(err, syscall.ECONNRESET))
		})
	}
}
