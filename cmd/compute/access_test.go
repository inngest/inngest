package computecmd

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWaitForVerification(t *testing.T) {
	pollInterval = time.Millisecond
	t.Cleanup(func() { pollInterval = 2 * time.Second })

	const required = `{"data":{"allowed":false,"code":"verification_required","verifyUrl":"https://app.inngest.com/verify"}}`
	for _, tc := range []struct {
		name      string
		responses []string
		wantCalls int
		err       string
	}{
		{
			name:      "verified while waiting",
			responses: []string{required, required, `{"data":{"allowed":true}}`},
			wantCalls: 3,
		},
		{
			name:      "suspended while waiting",
			responses: []string{required, `{"data":{"allowed":false,"code":"compute_suspended","message":"Compute is suspended for this account."}}`},
			wantCalls: 2,
			err:       "compute_suspended: Compute is suspended for this account.",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/v2/compute/access", r.URL.Path)
				_, _ = w.Write([]byte(tc.responses[calls]))
				calls++
			}))
			defer srv.Close()
			api := &client{
				baseURL: srv.URL + "/v2",
				token:   func(context.Context) (string, error) { return "token", nil },
				http:    srv.Client(),
			}

			var stdout bytes.Buffer
			err := waitForVerification(context.Background(), &stdout, api, "https://app.inngest.com/verify")
			if tc.err != "" {
				require.EqualError(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantCalls, calls)
			require.Contains(t, stdout.String(), "An account admin can verify it at:\n\n  https://app.inngest.com/verify")
		})
	}

	t.Run("not verified in time", func(t *testing.T) {
		accessWait = 20 * time.Millisecond
		t.Cleanup(func() { accessWait = 30 * time.Minute })
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// still in flight when the wait ends.
			time.Sleep(30 * time.Millisecond)
			_, _ = w.Write([]byte(required))
		}))
		defer srv.Close()
		api := &client{
			baseURL: srv.URL + "/v2",
			token:   func(context.Context) (string, error) { return "token", nil },
			http:    srv.Client(),
		}
		err := waitForVerification(context.Background(), io.Discard, api, "https://app.inngest.com/verify")
		require.EqualError(t, err, "the account wasn't verified within 20ms; run inngest deploy again once it is")
	})
}

// a 404 is a server without the check, unless the environment is missing.
func TestRequireComputeAccessNotFound(t *testing.T) {
	for body, wantErr := range map[string]bool{
		"404 page not found": false,
		`{"errors":[{"code":"env_not_found","message":"Environment not found"}]}`: true,
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(body))
		}))
		api := &client{
			baseURL: srv.URL + "/v2",
			token:   func(context.Context) (string, error) { return "token", nil },
			http:    srv.Client(),
		}
		err := requireComputeAccess(context.Background(), nil, io.Discard, api)
		srv.Close()
		require.Equal(t, wantErr, err != nil, body)
	}
}
