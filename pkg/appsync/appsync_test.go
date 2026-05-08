package appsync

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

const testSigningKey = "signkey-test-deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"

// realSDKHandler returns an httptest.Server backed by a real inngestgo
// handler with in-band sync enabled. The handler signs responses with the
// production code path, so the success-path tests exercise actual signing
// rather than a hand-rolled HMAC.
func realSDKHandler(t *testing.T) *httptest.Server {
	t.Helper()
	allow := true
	dev := false
	sk := testSigningKey
	client, err := inngestgo.NewClient(inngestgo.ClientOpts{
		AppID:           "my-app",
		SigningKey:      &sk,
		AllowInBandSync: &allow,
		Dev:             &dev,
	})
	require.NoError(t, err)
	return httptest.NewServer(client.Serve())
}

// hmacSignResponse mirrors inngestgo's signWithoutJCS. Used only by tests
// that intentionally produce a signature outside the production code path:
// signing with a different key, or signing a deliberately malformed body
// (where JCS would refuse). Other tests should drive realSDKHandler.
func hmacSignResponse(t *testing.T, body []byte, key string) string {
	t.Helper()
	keyBytes := regexp.MustCompile(`^signkey-\w+-`).ReplaceAll([]byte(key), nil)
	ts := time.Now().Unix()
	mac := hmac.New(sha256.New, keyBytes)
	_, _ = mac.Write(body)
	_, _ = fmt.Fprintf(mac, "%d", ts)
	return fmt.Sprintf("t=%d&s=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

func assertSyscode(t *testing.T, syscodeErr *syscode.Error, err error, wantCode string) {
	t.Helper()
	require.NoError(t, err, "unexpected system error")
	require.NotNil(t, syscodeErr, "expected syscode.Error with code %q, got nil", wantCode)
	require.Equal(t, wantCode, syscodeErr.Code, "syscode mismatch (msg=%q)", syscodeErr.Message)
}

func TestSync(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := require.New(t)
		srv := realSDKHandler(t)
		defer srv.Close()

		resp, syscodeErr, err := Sync(context.Background(), Opts{
			URL:               srv.URL,
			SigningKey:        testSigningKey,
			ExpectedAppID:     "my-app",
			AllowInsecureHTTP: true,
			HTTPClient:        srv.Client(),
		})
		r.NoError(err)
		r.Nil(syscodeErr)
		r.NotNil(resp)
		r.Equal("my-app", resp.AppID)
		r.Equal(inngestgo.SDKLanguage, resp.SDKLanguage)
		r.Equal(inngestgo.SDKVersion, resp.SDKVersion)
	})

	t.Run("sends required request headers", func(t *testing.T) {
		// Capture the request headers/body the handler receives. We don't
		// need a real signed response — Sync's request-side behavior is what
		// we're verifying, so a minimal handler that doesn't return a body
		// is enough; we ignore Sync's resulting error.
		r := require.New(t)
		var got struct {
			serverKind string
			syncKind   string
			signature  string
			body       []byte
		}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			got.serverKind = req.Header.Get(headers.HeaderKeyServerKind)
			got.syncKind = req.Header.Get(inngestgo.HeaderKeySyncKind)
			got.signature = req.Header.Get(headers.HeaderKeySignature)
			got.body, _ = io.ReadAll(req.Body)
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		_, _, _ = Sync(context.Background(), Opts{
			URL:               srv.URL,
			SigningKey:        testSigningKey,
			ExpectedAppID:     "my-app",
			AllowInsecureHTTP: true,
			HTTPClient:        srv.Client(),
		})

		r.Equal(headers.ServerKindCloud, got.serverKind)
		r.Equal(inngestgo.SyncKindInBand, got.syncKind)
		r.NotEmpty(got.signature, "expected signed request")

		var bodyParsed map[string]string
		r.NoError(json.Unmarshal(got.body, &bodyParsed))
		r.Equal(srv.URL, bodyParsed["url"])
	})

	t.Run("respects ServerKind opt", func(t *testing.T) {
		r := require.New(t)
		var got string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			got = req.Header.Get(headers.HeaderKeyServerKind)
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		_, _, _ = Sync(context.Background(), Opts{
			URL:               srv.URL,
			SigningKey:        testSigningKey,
			ServerKind:        headers.ServerKindDev,
			ExpectedAppID:     "my-app",
			AllowInsecureHTTP: true,
			HTTPClient:        srv.Client(),
		})
		r.Equal(headers.ServerKindDev, got)
	})

	t.Run("missing response signature", func(t *testing.T) {
		// Real handler would set the sig; strip it on the way out via a
		// proxying handler that lets the request hit the SDK but rewrites
		// the response.
		srv := stripResponseSig(t, realSDKHandler(t))
		defer srv.Close()

		_, syscodeErr, err := Sync(context.Background(), Opts{
			URL:               srv.URL,
			SigningKey:        testSigningKey,
			ExpectedAppID:     "my-app",
			AllowInsecureHTTP: true,
			HTTPClient:        srv.Client(),
		})
		assertSyscode(t, syscodeErr, err, syscode.CodeHTTPMissingHeader)
	})

	t.Run("invalid response signature", func(t *testing.T) {
		srv := overrideResponseSig(t, realSDKHandler(t), fmt.Sprintf("t=%d&s=cafebabe", time.Now().Unix()))
		defer srv.Close()

		_, syscodeErr, err := Sync(context.Background(), Opts{
			URL:               srv.URL,
			SigningKey:        testSigningKey,
			ExpectedAppID:     "my-app",
			AllowInsecureHTTP: true,
			HTTPClient:        srv.Client(),
		})
		assertSyscode(t, syscodeErr, err, syscode.CodeSigVerificationFailed)
	})

	t.Run("response signature with different key", func(t *testing.T) {
		// Hand-rolled HMAC: by definition we want a sig produced under a
		// different key than the validator uses.
		otherKey := "signkey-test-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			body := []byte(`{"app_id":"my-app","sdk_language":"go","sdk_version":"0.0.0","url":"http://x/api/inngest","functions":[]}`)
			w.Header().Set(headers.HeaderKeySignature, hmacSignResponse(t, body, otherKey))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer srv.Close()

		_, syscodeErr, err := Sync(context.Background(), Opts{
			URL:               srv.URL,
			SigningKey:        testSigningKey,
			ExpectedAppID:     "my-app",
			AllowInsecureHTTP: true,
			HTTPClient:        srv.Client(),
		})
		assertSyscode(t, syscodeErr, err, syscode.CodeSigVerificationFailed)
	})

	t.Run("non-2xx response", func(t *testing.T) {
		r := require.New(t)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"secret":"do-not-leak"}`))
		}))
		defer srv.Close()

		_, syscodeErr, err := Sync(context.Background(), Opts{
			URL:               srv.URL,
			SigningKey:        testSigningKey,
			ExpectedAppID:     "my-app",
			AllowInsecureHTTP: true,
			HTTPClient:        srv.Client(),
		})
		assertSyscode(t, syscodeErr, err, syscode.CodeHTTPNotOK)

		// Upstream body must not leak into Message (callers reflect Message
		// to API responses).
		r.Equal("SDK returned non-2xx response: status=500", syscodeErr.Message)

		data, ok := syscodeErr.Data.(map[string]any)
		r.True(ok, "Data = %T, want map[string]any", syscodeErr.Data)
		r.Equal(500, data["status_code"])
	})

	t.Run("Cf-Mitigated", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Cf-Mitigated", "challenge")
			w.WriteHeader(http.StatusForbidden)
		}))
		defer srv.Close()

		_, syscodeErr, err := Sync(context.Background(), Opts{
			URL:               srv.URL,
			SigningKey:        testSigningKey,
			ExpectedAppID:     "my-app",
			AllowInsecureHTTP: true,
			HTTPClient:        srv.Client(),
		})
		assertSyscode(t, syscodeErr, err, syscode.CodeCloudflareMitigated)
	})

	t.Run("malformed body", func(t *testing.T) {
		// Hand-rolled HMAC: invalid JSON can't go through inngestgo.Sign
		// because JCS canonicalization would refuse to parse it.
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			body := []byte(`{not valid json`)
			w.Header().Set(headers.HeaderKeySignature, hmacSignResponse(t, body, testSigningKey))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer srv.Close()

		_, syscodeErr, err := Sync(context.Background(), Opts{
			URL:               srv.URL,
			SigningKey:        testSigningKey,
			ExpectedAppID:     "my-app",
			AllowInsecureHTTP: true,
			HTTPClient:        srv.Client(),
		})
		assertSyscode(t, syscodeErr, err, syscode.CodeMalformedResponse)
	})

	t.Run("body too large", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set(headers.HeaderKeySignature, "t=0&s=00")
			w.WriteHeader(http.StatusOK)
			// Size check runs before signature/parse, so content isn't validated.
			buf := make([]byte, 1024*1024)
			for i := 0; i < (maxResponseBytes/len(buf))+2; i++ {
				if _, err := w.Write(buf); err != nil {
					return
				}
			}
		}))
		defer srv.Close()

		_, syscodeErr, err := Sync(context.Background(), Opts{
			URL:               srv.URL,
			SigningKey:        testSigningKey,
			ExpectedAppID:     "my-app",
			AllowInsecureHTTP: true,
			HTTPClient:        srv.Client(),
		})
		assertSyscode(t, syscodeErr, err, syscode.CodeOutputTooLarge)
	})

	t.Run("unreachable", func(t *testing.T) {
		r := require.New(t)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
		url := srv.URL
		srv.Close() // shut it down so the dial fails

		_, syscodeErr, err := Sync(context.Background(), Opts{
			URL:               url,
			SigningKey:        testSigningKey,
			ExpectedAppID:     "my-app",
			AllowInsecureHTTP: true,
		})
		assertSyscode(t, syscodeErr, err, syscode.CodeHTTPUnreachable)
		r.Equal("unable to reach SDK URL", syscodeErr.Message)
	})

	t.Run("requires expected app ID", func(t *testing.T) {
		r := require.New(t)
		resp, syscodeErr, err := Sync(context.Background(), Opts{
			URL:        "https://example.com",
			SigningKey: testSigningKey,
		})
		r.Nil(resp)
		r.Nil(syscodeErr)
		r.ErrorIs(err, ErrMissingExpectedAppID)
	})

	t.Run("requires URL", func(t *testing.T) {
		r := require.New(t)
		resp, syscodeErr, err := Sync(context.Background(), Opts{
			SigningKey:    testSigningKey,
			ExpectedAppID: "my-app",
		})
		r.Nil(resp)
		r.Nil(syscodeErr)
		r.ErrorIs(err, ErrMissingURL)
	})

	t.Run("requires signing key", func(t *testing.T) {
		r := require.New(t)
		resp, syscodeErr, err := Sync(context.Background(), Opts{
			URL:           "https://example.com",
			ExpectedAppID: "my-app",
		})
		r.Nil(resp)
		r.Nil(syscodeErr)
		r.ErrorIs(err, ErrMissingSigningKey)
	})

	t.Run("rejects insecure HTTP by default", func(t *testing.T) {
		r := require.New(t)
		_, syscodeErr, err := Sync(context.Background(), Opts{
			URL:           "http://example.com",
			SigningKey:    testSigningKey,
			ExpectedAppID: "my-app",
		})
		assertSyscode(t, syscodeErr, err, syscode.CodeURLSchemeDenied)
		r.Equal("insecure http:// scheme not permitted", syscodeErr.Message)
	})

	t.Run("rejects unsupported scheme", func(t *testing.T) {
		_, syscodeErr, err := Sync(context.Background(), Opts{
			URL:               "ftp://example.com",
			SigningKey:        testSigningKey,
			ExpectedAppID:     "my-app",
			AllowInsecureHTTP: true, // ftp is rejected regardless
		})
		assertSyscode(t, syscodeErr, err, syscode.CodeURLSchemeDenied)
	})

	t.Run("succeeds against loopback (built-in client)", func(t *testing.T) {
		r := require.New(t)
		srv := realSDKHandler(t)
		defer srv.Close()

		resp, syscodeErr, err := Sync(context.Background(), Opts{
			URL:               srv.URL,
			SigningKey:        testSigningKey,
			ExpectedAppID:     "my-app",
			AllowInsecureHTTP: true,
		})
		r.NoError(err)
		r.Nil(syscodeErr)
		r.NotNil(resp)
		r.Equal("my-app", resp.AppID)
	})

	t.Run("refuses redirects", func(t *testing.T) {
		r := require.New(t)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Redirect(w, &http.Request{}, "https://example.com/elsewhere", http.StatusFound)
		}))
		defer srv.Close()

		_, syscodeErr, err := Sync(context.Background(), Opts{
			URL:               srv.URL,
			SigningKey:        testSigningKey,
			ExpectedAppID:     "my-app",
			AllowInsecureHTTP: true,
		})
		assertSyscode(t, syscodeErr, err, syscode.CodeRedirectDenied)
		r.Equal("redirects are not permitted", syscodeErr.Message)
	})

	t.Run("ExpectedAppID mismatch", func(t *testing.T) {
		r := require.New(t)
		// Real SDK handler reports AppID "my-app"; expect "other-app".
		srv := realSDKHandler(t)
		defer srv.Close()

		_, syscodeErr, err := Sync(context.Background(), Opts{
			URL:               srv.URL,
			SigningKey:        testSigningKey,
			ExpectedAppID:     "other-app",
			AllowInsecureHTTP: true,
			HTTPClient:        srv.Client(),
		})
		assertSyscode(t, syscodeErr, err, syscode.CodeAppIDMismatch)
		r.Contains(syscodeErr.Message, "other-app")
		r.Contains(syscodeErr.Message, "my-app")
	})

}

// stripResponseSig wraps an httptest.Server in a proxy that forwards the
// request to the inner server but removes the X-Inngest-Signature response
// header before writing it to the caller.
func stripResponseSig(t *testing.T, inner *httptest.Server) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyTo(t, inner, w, r, func(h http.Header) {
			h.Del(headers.HeaderKeySignature)
		})
	}))
}

// overrideResponseSig wraps an httptest.Server in a proxy that forwards the
// request to the inner server and replaces the X-Inngest-Signature response
// header with the given value.
func overrideResponseSig(t *testing.T, inner *httptest.Server, sig string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyTo(t, inner, w, r, func(h http.Header) {
			h.Set(headers.HeaderKeySignature, sig)
		})
	}))
}

func proxyTo(t *testing.T, inner *httptest.Server, w http.ResponseWriter, r *http.Request, mutateHeaders func(http.Header)) {
	t.Helper()
	body, _ := io.ReadAll(r.Body)
	req, err := http.NewRequest(r.Method, inner.URL+r.URL.Path, bytesReader(body))
	require.NoError(t, err)
	for k, vs := range r.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	resp, err := inner.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	mutateHeaders(w.Header())
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func bytesReader(b []byte) io.Reader {
	if len(b) == 0 {
		return http.NoBody
	}
	return &readCloserBytes{b: b}
}

type readCloserBytes struct {
	b []byte
	i int
}

func (r *readCloserBytes) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}

func TestResponse(t *testing.T) {
	t.Run("ToRegisterRequest", func(t *testing.T) {
		r := require.New(t)
		env := "staging"
		framework := "express"
		platform := "fly"
		resp := &Response{
			AppID:       "app-1",
			Env:         &env,
			Framework:   &framework,
			Platform:    &platform,
			SDKLanguage: "go",
			SDKVersion:  "1.2.3",
			URL:         "https://example.com/api/inngest",
			Functions:   []sdk.SDKFunction{{Name: "fn"}},
			Inspection: map[string]any{
				"capabilities": map[string]any{
					"in_band_sync": "v1",
					"trust_probe":  "v1",
					"connect":      "v1",
				},
			},
		}

		req := resp.ToRegisterRequest()
		r.Equal("app-1", req.AppName)
		r.Equal("go:1.2.3", req.SDK)
		r.Equal(env, req.Headers.Env)
		r.Equal(platform, req.Headers.Platform)
		r.Equal(framework, req.Framework)
		r.Equal(resp.URL, req.URL)
		r.Len(req.Functions, 1)
		r.Equal("fn", req.Functions[0].Name)
		r.Equal(sdk.Capabilities{InBandSync: "v1", TrustProbe: "v1", Connect: "v1"}, req.Capabilities)
	})

	t.Run("ToRegisterRequest with nil optionals", func(t *testing.T) {
		r := require.New(t)
		resp := &Response{
			AppID:       "app-1",
			SDKLanguage: "go",
			SDKVersion:  "1.0.0",
			URL:         "https://example.com",
		}
		req := resp.ToRegisterRequest()
		r.Empty(req.Headers.Env)
		r.Empty(req.Headers.Platform)
		r.Empty(req.Framework)
		r.Equal(sdk.Capabilities{}, req.Capabilities)
	})

	t.Run("ToRegisterRequest normalizes URLs", func(t *testing.T) {
		r := require.New(t)
		// 127.0.0.1 to localhost, :80 stripped, step URLs same.
		resp := &Response{
			AppID:       "app-1",
			SDKLanguage: "go",
			SDKVersion:  "1.0.0",
			URL:         "http://127.0.0.1:80/api/inngest",
			Functions: []sdk.SDKFunction{{
				Name: "fn",
				Steps: map[string]sdk.SDKStep{
					"step": {Runtime: map[string]any{"url": "http://127.0.0.1:80/api/inngest?fnId=1"}},
				},
			}},
		}
		req := resp.ToRegisterRequest()
		r.Equal("http://localhost/api/inngest", req.URL)
		stepURL, _ := req.Functions[0].Steps["step"].Runtime["url"].(string)
		r.Equal("http://localhost/api/inngest?fnId=1", stepURL)
	})

	t.Run("capabilities absent or malformed", func(t *testing.T) {
		cases := []struct {
			name string
			insp map[string]any
		}{
			{"nil inspection", nil},
			{"missing capabilities", map[string]any{"other": 1}},
			{"non-object capabilities", map[string]any{"capabilities": "nope"}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				resp := &Response{Inspection: tc.insp}
				require.Equal(t, sdk.Capabilities{}, resp.capabilities())
			})
		}
	})
}
