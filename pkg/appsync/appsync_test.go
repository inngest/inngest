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

// signResponse mirrors the SDK's signWithoutJCS.
func signResponse(t *testing.T, body []byte, key string) string {
	t.Helper()
	keyBytes := regexp.MustCompile(`^signkey-\w+-`).ReplaceAll([]byte(key), nil)
	ts := time.Now().Unix()
	mac := hmac.New(sha256.New, keyBytes)
	_, _ = mac.Write(body)
	_, _ = fmt.Fprintf(mac, "%d", ts)
	return fmt.Sprintf("t=%d&s=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

func sampleResponseBody(t *testing.T) []byte {
	t.Helper()
	env := "production"
	platform := "vercel"
	resp := Response{
		AppID:       "my-app",
		Env:         &env,
		Platform:    &platform,
		SDKAuthor:   "inngest",
		SDKLanguage: "go",
		SDKVersion:  "0.7.0",
		URL:         "https://example.com/api/inngest",
		Functions:   []sdk.SDKFunction{},
		Inspection: map[string]any{
			"capabilities": map[string]any{
				"in_band_sync": "v1",
				"trust_probe":  "v1",
				"connect":      "v1",
			},
		},
	}
	byt, err := json.Marshal(resp)
	require.NoError(t, err, "marshal sample response")
	return byt
}

// inBandHandler returns a fake SDK server. customize runs after defaults are
// set, so it can also override (including with empty values).
func inBandHandler(t *testing.T, customize func(w http.ResponseWriter, body *[]byte)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		body := sampleResponseBody(t)
		w.Header().Set(headers.HeaderKeySignature, signResponse(t, body, testSigningKey))
		w.Header().Set(inngestgo.HeaderKeySyncKind, inngestgo.SyncKindInBand)
		w.Header().Set(headers.HeaderKeySDK, "go:0.7.0")
		if customize != nil {
			customize(w, &body)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
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
		srv := inBandHandler(t, nil)
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
		r.Equal("go", resp.SDKLanguage)
		r.Equal("0.7.0", resp.SDKVersion)
	})

	t.Run("sends required request headers", func(t *testing.T) {
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

			body := sampleResponseBody(t)
			w.Header().Set(headers.HeaderKeySignature, signResponse(t, body, testSigningKey))
			w.Header().Set(inngestgo.HeaderKeySyncKind, inngestgo.SyncKindInBand)
			w.Header().Set(headers.HeaderKeySDK, "go:0.7.0")
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
		r.NoError(err)
		r.Nil(syscodeErr)

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
			body := sampleResponseBody(t)
			w.Header().Set(headers.HeaderKeySignature, signResponse(t, body, testSigningKey))
			w.Header().Set(inngestgo.HeaderKeySyncKind, inngestgo.SyncKindInBand)
			_, _ = w.Write(body)
		}))
		defer srv.Close()

		_, syscodeErr, err := Sync(context.Background(), Opts{
			URL:               srv.URL,
			SigningKey:        testSigningKey,
			ServerKind:        headers.ServerKindDev,
			ExpectedAppID:     "my-app",
			AllowInsecureHTTP: true,
			HTTPClient:        srv.Client(),
		})
		r.NoError(err)
		r.Nil(syscodeErr)
		r.Equal(headers.ServerKindDev, got)
	})

	t.Run("missing response signature", func(t *testing.T) {
		srv := inBandHandler(t, func(w http.ResponseWriter, _ *[]byte) {
			w.Header().Set(headers.HeaderKeySignature, "")
		})
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
		srv := inBandHandler(t, func(w http.ResponseWriter, _ *[]byte) {
			w.Header().Set(headers.HeaderKeySignature, fmt.Sprintf("t=%d&s=cafebabe", time.Now().Unix()))
		})
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
		otherKey := "signkey-test-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		srv := inBandHandler(t, func(w http.ResponseWriter, body *[]byte) {
			w.Header().Set(headers.HeaderKeySignature, signResponse(t, *body, otherKey))
		})
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
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			body := []byte(`{not valid json`)
			w.Header().Set(headers.HeaderKeySignature, signResponse(t, body, testSigningKey))
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
			w.Header().Set(inngestgo.HeaderKeySyncKind, inngestgo.SyncKindInBand)
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
		srv := inBandHandler(t, nil)
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
		// sampleResponseBody reports AppID "my-app"; expect "other-app".
		srv := inBandHandler(t, nil)
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
		// 127.0.0.1 → localhost, :80 stripped, step URLs same.
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
