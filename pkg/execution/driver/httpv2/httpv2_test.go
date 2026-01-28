package httpv2

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	apiv1 "github.com/inngest/inngest/pkg/api/apiv1"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/exechttp"
	sv1 "github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/inngest"
	inngestgo "github.com/inngest/inngestgo"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestNewDriver(t *testing.T) {
	client := exechttp.Client(exechttp.SecureDialerOpts{})
	driver := NewDriver(client)
	require.NotNil(t, driver)
	require.Equal(t, "httpv2", driver.Name())
}

func TestSyncMethod(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedMethod string
	}{
		{
			name:           "default POST method",
			method:         "",
			expectedMethod: "POST",
		},
		{
			name:           "custom GET method",
			method:         "GET",
			expectedMethod: "GET",
		},
		{
			name:           "custom PUT method",
			method:         "PUT",
			expectedMethod: "PUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedMethod string
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedMethod = r.Method
				w.Header().Set(headers.HeaderKeySDK, "test-sdk")
				opcodes := []*sv1.GeneratorOpcode{{Op: enums.OpcodeNone}}
				w.WriteHeader(200)
				_ = json.NewEncoder(w).Encode(opcodes)
			}))
			defer ts.Close()

			client := exechttp.Client(exechttp.SecureDialerOpts{AllowPrivate: true})
			d := &httpv2{Client: client}

			u, _ := url.Parse(ts.URL)
			fn := inngest.Function{
				Driver: inngest.FunctionDriver{
					URI: u.String(),
					Metadata: map[string]any{
						"type": "sync",
					},
				},
			}

			if tt.method != "" {
				fn.Driver.Metadata["method"] = tt.method
			}

			opts := driver.V2RequestOpts{
				Fn:         fn,
				SigningKey: []byte("test-key"),
				Metadata: sv2.Metadata{
					ID: sv2.ID{
						RunID: ulid.MustNew(ulid.Now(), rand.Reader),
					},
				},
				URL: u.String(),
			}

			resp, userErr, internalErr := d.Do(context.Background(), nil, opts)
			require.NoError(t, userErr)
			require.NoError(t, internalErr)
			require.NotNil(t, resp)
			require.Equal(t, tt.expectedMethod, receivedMethod)
		})
	}
}

func TestSyncHeaders(t *testing.T) {
	var receivedHeaders http.Header
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.Header().Set(headers.HeaderKeySDK, "test-sdk")
		opcodes := []*sv1.GeneratorOpcode{{Op: enums.OpcodeNone}}
		w.WriteHeader(200)
		_ = json.NewEncoder(w).Encode(opcodes)
	}))
	defer ts.Close()

	client := exechttp.Client(exechttp.SecureDialerOpts{AllowPrivate: true})
	d := &httpv2{Client: client}

	u, _ := url.Parse(ts.URL)
	runID := ulid.MustNew(ulid.Now(), rand.Reader)
	fn := inngest.Function{
		Driver: inngest.FunctionDriver{
			URI: u.String(),
			Metadata: map[string]any{
				"type": "sync",
			},
		},
	}

	opts := driver.V2RequestOpts{
		Fn:         fn,
		SigningKey: []byte("test-signing-key"),
		Metadata: sv2.Metadata{
			ID: sv2.ID{
				RunID: runID,
			},
		},
		URL: u.String(),
	}

	resp, userErr, internalErr := d.Do(context.Background(), nil, opts)
	require.NoError(t, userErr)
	require.NoError(t, internalErr)
	require.NotNil(t, resp)

	require.Equal(t, "application/json", receivedHeaders.Get("Content-Type"))
	require.Contains(t, receivedHeaders.Get("X-Inngest-Signature"), "t=")
	require.Contains(t, receivedHeaders.Get("X-Inngest-Signature"), "s=")
	require.Equal(t, runID.String(), receivedHeaders.Get("X-Run-ID"))
}

func TestSyncNonSDKResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("not an SDK response"))
	}))
	defer ts.Close()

	client := exechttp.Client(exechttp.SecureDialerOpts{AllowPrivate: true})
	d := &httpv2{Client: client}

	u, _ := url.Parse(ts.URL)
	fn := inngest.Function{
		Driver: inngest.FunctionDriver{
			URI: u.String(),
			Metadata: map[string]any{
				"type": "sync",
			},
		},
	}

	opts := driver.V2RequestOpts{
		Fn:         fn,
		SigningKey: []byte("test-key"),
		Metadata: sv2.Metadata{
			ID: sv2.ID{
				RunID: ulid.MustNew(ulid.Now(), rand.Reader),
			},
		},
		URL: u.String(),
	}

	resp, userErr, internalErr := d.Do(context.Background(), nil, opts)
	require.Nil(t, resp)
	require.NotNil(t, userErr)
	require.NoError(t, internalErr)
	require.Contains(t, userErr.Error(), "didn't receive SDK response")
}

func TestSyncRequestErrors(t *testing.T) {
	tests := []struct {
		name          string
		serverHandler func(w http.ResponseWriter, r *http.Request)
		expectedError string
		isUserError   bool
	}{
		{
			name: "body too large",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(headers.HeaderKeySDK, "test-sdk")
				data := strings.Repeat("a", 10*1024*1024) // Large response
				w.WriteHeader(200)
				_, _ = w.Write([]byte(data))
			},
			expectedError: "SDK response too large",
			isUserError:   true,
		},
		{
			name: "server error",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(500)
			},
			expectedError: "didn't receive SDK response",
			isUserError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(tt.serverHandler))
			defer ts.Close()

			client := exechttp.Client(exechttp.SecureDialerOpts{AllowPrivate: true})
			d := &httpv2{Client: client}

			u, _ := url.Parse(ts.URL)
			fn := inngest.Function{
				Driver: inngest.FunctionDriver{
					URI: u.String(),
					Metadata: map[string]any{
						"type": "sync",
					},
				},
			}

			opts := driver.V2RequestOpts{
				Fn:         fn,
				SigningKey: []byte("test-key"),
				Metadata: sv2.Metadata{
					ID: sv2.ID{
						RunID: ulid.MustNew(ulid.Now(), rand.Reader),
					},
				},
				URL: u.String(),
			}

			resp, userErr, internalErr := d.Do(context.Background(), nil, opts)

			if tt.isUserError {
				require.Nil(t, resp)
				require.NotNil(t, userErr)
				require.NoError(t, internalErr)
				require.Contains(t, userErr.Error(), tt.expectedError)
			} else {
				require.Nil(t, resp)
				require.NoError(t, userErr)
				require.NotNil(t, internalErr)
				require.Contains(t, internalErr.Error(), tt.expectedError)
			}
		})
	}
}

func TestAsync(t *testing.T) {
	client := exechttp.Client(exechttp.SecureDialerOpts{})
	d := &httpv2{Client: client}

	fn := inngest.Function{
		Driver: inngest.FunctionDriver{
			Metadata: map[string]any{
				"type": "async",
			},
		},
	}

	opts := driver.V2RequestOpts{
		Fn:         fn,
		SigningKey: []byte("test-key"),
		Metadata: sv2.Metadata{
			ID: sv2.ID{
				RunID: ulid.MustNew(ulid.Now(), rand.Reader),
			},
		},
	}

	resp, userErr, internalErr := d.Do(context.Background(), nil, opts)
	require.Nil(t, resp)
	require.NoError(t, userErr)
	require.NotNil(t, internalErr)
	require.Contains(t, internalErr.Error(), "async v2 http driver not implemneted")
}

func TestParseOpcodes(t *testing.T) {
	tests := []struct {
		name          string
		input         []byte
		expectedOps   []*sv1.GeneratorOpcode
		expectedError string
	}{
		{
			name:  "valid opcodes",
			input: []byte(`[{"op":"Step","id":"test-id","name":"test-step","data":"dGVzdA=="}]`),
			expectedOps: []*sv1.GeneratorOpcode{
				{
					Op:   enums.OpcodeStep,
					ID:   "test-id",
					Name: "test-step",
					Data: []byte(`"dGVzdA=="`),
				},
			},
		},
		{
			name:        "empty array becomes OpcodeNone",
			input:       []byte(`[]`),
			expectedOps: []*sv1.GeneratorOpcode{{Op: enums.OpcodeNone}},
		},
		{
			name:          "invalid JSON",
			input:         []byte(`invalid json`),
			expectedError: "SDK returned an unexpected response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ops, err := parseOpcodes(tt.input, 200)

			if tt.expectedError != "" {
				require.NotNil(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				require.Nil(t, ops)
			} else {
				require.NoError(t, err)
				require.Equal(t, len(tt.expectedOps), len(ops))
				for i, expected := range tt.expectedOps {
					require.Equal(t, expected.Op, ops[i].Op)
					require.Equal(t, expected.ID, ops[i].ID)
					require.Equal(t, expected.Name, ops[i].Name)
				}
			}
		})
	}
}

func TestSign(t *testing.T) {
	tests := []struct {
		name     string
		key      []byte
		body     []byte
		expected string
	}{
		{
			name:     "empty key returns empty string",
			key:      []byte{},
			body:     []byte("test"),
			expected: "",
		},
		{
			name: "valid signature",
			key:  []byte("test-key"),
			body: []byte("test-body"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Sign(context.Background(), tt.key, tt.body)

			if tt.expected != "" {
				require.Equal(t, tt.expected, result)
			} else if len(tt.key) == 0 {
				require.Empty(t, result)
			} else {
				require.Contains(t, result, "t=")
				require.Contains(t, result, "s=")

				parts := strings.Split(result, "&")
				require.Len(t, parts, 2)
				require.True(t, strings.HasPrefix(parts[0], "t="))
				require.True(t, strings.HasPrefix(parts[1], "s="))

				timestampStr := strings.TrimPrefix(parts[0], "t=")
				timestamp, err := fmt.Sscanf(timestampStr, "%d", new(int64))
				require.Equal(t, 1, timestamp)
				require.NoError(t, err)
			}
		})
	}
}

func TestSyncResponseParsing(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(headers.HeaderKeySDK, "test-sdk-v1.2.3")
		w.Header().Set("retry-after", time.Now().Add(5*time.Minute).Format(time.RFC3339))
		w.Header().Set("x-inngest-no-retry", "true")
		w.Header().Set(headers.HeaderKeyRequestVersion, "20210101")

		opcodes := []*sv1.GeneratorOpcode{
			{Op: enums.OpcodeStep, ID: "test-step", Name: "Test Step"},
		}
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(opcodes)
	}))
	defer ts.Close()

	client := exechttp.Client(exechttp.SecureDialerOpts{AllowPrivate: true})
	d := &httpv2{Client: client}

	u, _ := url.Parse(ts.URL)
	fn := inngest.Function{
		Driver: inngest.FunctionDriver{
			URI: u.String(),
			Metadata: map[string]any{
				"type": "sync",
			},
		},
	}

	opts := driver.V2RequestOpts{
		Fn:         fn,
		SigningKey: []byte("test-key"),
		Metadata: sv2.Metadata{
			ID: sv2.ID{
				RunID: ulid.MustNew(ulid.Now(), rand.Reader),
			},
		},
		URL: u.String(),
	}

	resp, userErr, internalErr := d.Do(context.Background(), nil, opts)
	require.NoError(t, userErr)
	require.NoError(t, internalErr)
	require.NotNil(t, resp)

	require.Equal(t, 201, resp.StatusCode)
	require.Equal(t, "test-sdk-v1.2.3", resp.SDK)
	require.NotNil(t, resp.RetryAt)
	require.True(t, resp.NoRetry)
	require.Equal(t, 20210101, resp.RequestVersion)
	require.Len(t, resp.Generator, 1)
	require.Equal(t, enums.OpcodeStep, resp.Generator[0].Op)
	require.Equal(t, "test-step", resp.Generator[0].ID)
	require.Equal(t, "Test Step", resp.Generator[0].Name)
	require.True(t, resp.Duration > 0)
}

func TestSyncDriverTypeDetection(t *testing.T) {
	client := exechttp.Client(exechttp.SecureDialerOpts{})
	d := &httpv2{Client: client}

	tests := []struct {
		name         string
		driverType   string
		expectsAsync bool
	}{
		{
			name:       "sync type",
			driverType: "sync",
		},
		{
			name:       "async type",
			driverType: "async",
		},
		{
			name:       "no type defaults to async",
			driverType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := make(map[string]any)

			if tt.driverType != "" {
				metadata["type"] = tt.driverType
			}

			fn := inngest.Function{
				Driver: inngest.FunctionDriver{
					URI:      "http://example.com",
					Metadata: metadata,
				},
			}

			opts := driver.V2RequestOpts{
				Fn:         fn,
				SigningKey: []byte("test-key"),
				Metadata: sv2.Metadata{
					ID: sv2.ID{
						RunID: ulid.MustNew(ulid.Now(), rand.Reader),
					},
				},
				URL: "http://example.com",
			}

			resp, userErr, internalErr := d.Do(context.Background(), nil, opts)

			if tt.driverType == "sync" {
				require.Nil(t, resp)
				require.Nil(t, internalErr)
			} else {
				// For sync functions that can't connect, we should get an internal error
				require.Nil(t, resp)
				require.NoError(t, userErr)
				require.NotNil(t, internalErr)
			}
		})
	}
}

// mockStateLoader implements sv2.StateLoader for testing.
type mockStateLoader struct {
	events []json.RawMessage
	err    error
}

func (m *mockStateLoader) LoadMetadata(ctx context.Context, id sv2.ID) (sv2.Metadata, error) {
	return sv2.Metadata{}, nil
}
func (m *mockStateLoader) LoadEvents(ctx context.Context, id sv2.ID) ([]json.RawMessage, error) {
	return m.events, m.err
}
func (m *mockStateLoader) LoadSteps(ctx context.Context, id sv2.ID) (map[string]json.RawMessage, error) {
	return nil, nil
}
func (m *mockStateLoader) LoadStepInputs(ctx context.Context, id sv2.ID) (map[string]json.RawMessage, error) {
	return nil, nil
}
func (m *mockStateLoader) LoadStepsWithIDs(ctx context.Context, id sv2.ID, stepIDs []string) (map[string]json.RawMessage, error) {
	return nil, nil
}
func (m *mockStateLoader) LoadStack(ctx context.Context, id sv2.ID) ([]string, error) {
	return nil, nil
}
func (m *mockStateLoader) LoadState(ctx context.Context, id sv2.ID) (sv2.State, error) {
	return sv2.State{}, nil
}

func newHTTPRequestEvent(body []byte, contentType, queryParams string) json.RawMessage {
	evt := inngestgo.GenericEvent[apiv1.NewAPIRunData]{
		Name: consts.HttpRequestName,
		Data: apiv1.NewAPIRunData{
			Body:        body,
			ContentType: contentType,
			QueryParams: queryParams,
		},
	}
	byt, _ := json.Marshal(evt)
	return byt
}

func TestSyncHTTPRequestEvent(t *testing.T) {
	tests := []struct {
		name                string
		stateLoader         sv2.StateLoader
		urlSuffix           string
		expectedBody        string
		expectedContentType string
		expectedQuery       string
	}{
		{
			name:                "nil state loader uses defaults",
			stateLoader:         nil,
			expectedBody:        "",
			expectedContentType: "application/json",
			expectedQuery:       "",
		},
		{
			name: "non-http event uses defaults",
			stateLoader: &mockStateLoader{
				events: []json.RawMessage{
					json.RawMessage(`{"name":"user/signup","data":{}}`),
				},
			},
			expectedBody:        "",
			expectedContentType: "application/json",
			expectedQuery:       "",
		},
		{
			name: "http event with body and content type",
			stateLoader: &mockStateLoader{
				events: []json.RawMessage{
					newHTTPRequestEvent([]byte(`{"key":"value"}`), "application/json", ""),
				},
			},
			expectedBody:        `{"key":"value"}`,
			expectedContentType: "application/json",
			expectedQuery:       "",
		},
		{
			name: "http event with custom content type",
			stateLoader: &mockStateLoader{
				events: []json.RawMessage{
					newHTTPRequestEvent([]byte(`name=test`), "application/x-www-form-urlencoded", ""),
				},
			},
			expectedBody:        `name=test`,
			expectedContentType: "application/x-www-form-urlencoded",
			expectedQuery:       "",
		},
		{
			name: "http event with query params",
			stateLoader: &mockStateLoader{
				events: []json.RawMessage{
					newHTTPRequestEvent(nil, "", "foo=bar&baz=1"),
				},
			},
			expectedBody:        "",
			expectedContentType: "application/json",
			expectedQuery:       "foo=bar&baz=1",
		},
		{
			name: "http event with query params appended to existing",
			stateLoader: &mockStateLoader{
				events: []json.RawMessage{
					newHTTPRequestEvent(nil, "", "extra=yes"),
				},
			},
			urlSuffix:           "?existing=1",
			expectedBody:        "",
			expectedContentType: "application/json",
			expectedQuery:       "existing=1&extra=yes",
		},
		{
			name: "state loader error uses defaults",
			stateLoader: &mockStateLoader{
				err: fmt.Errorf("state load error"),
			},
			expectedBody:        "",
			expectedContentType: "application/json",
			expectedQuery:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				receivedBody        string
				receivedContentType string
				receivedQuery       string
			)
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				bodyBytes, _ := io.ReadAll(r.Body)
				receivedBody = string(bodyBytes)
				receivedContentType = r.Header.Get("Content-Type")
				receivedQuery = r.URL.RawQuery
				w.Header().Set(headers.HeaderKeySDK, "test-sdk")
				opcodes := []*sv1.GeneratorOpcode{{Op: enums.OpcodeNone}}
				w.WriteHeader(200)
				_ = json.NewEncoder(w).Encode(opcodes)
			}))
			defer ts.Close()

			client := exechttp.Client(exechttp.SecureDialerOpts{AllowPrivate: true})
			d := &httpv2{Client: client}

			testURL := ts.URL + tt.urlSuffix
			fn := inngest.Function{
				Driver: inngest.FunctionDriver{
					URI: ts.URL,
					Metadata: map[string]any{
						"type": "sync",
					},
				},
			}

			opts := driver.V2RequestOpts{
				Fn:         fn,
				SigningKey: []byte("test-key"),
				Metadata: sv2.Metadata{
					ID: sv2.ID{
						RunID: ulid.MustNew(ulid.Now(), rand.Reader),
					},
				},
				URL: testURL,
			}

			resp, userErr, internalErr := d.Do(context.Background(), tt.stateLoader, opts)
			require.NoError(t, userErr)
			require.NoError(t, internalErr)
			require.NotNil(t, resp)

			require.Equal(t, tt.expectedContentType, receivedContentType)
			require.Equal(t, tt.expectedBody, receivedBody)
			require.Equal(t, tt.expectedQuery, receivedQuery)
		})
	}
}

func TestLoadHTTPRequestEvent(t *testing.T) {
	id := sv2.ID{RunID: ulid.MustNew(ulid.Now(), rand.Reader)}

	t.Run("nil state loader", func(t *testing.T) {
		result := loadHTTPRequestEvent(context.Background(), nil, id)
		require.Nil(t, result)
	})

	t.Run("empty events", func(t *testing.T) {
		sl := &mockStateLoader{events: []json.RawMessage{}}
		result := loadHTTPRequestEvent(context.Background(), sl, id)
		require.Nil(t, result)
	})

	t.Run("invalid json", func(t *testing.T) {
		sl := &mockStateLoader{events: []json.RawMessage{json.RawMessage(`not json`)}}
		result := loadHTTPRequestEvent(context.Background(), sl, id)
		require.Nil(t, result)
	})

	t.Run("wrong event name", func(t *testing.T) {
		sl := &mockStateLoader{
			events: []json.RawMessage{json.RawMessage(`{"name":"other/event","data":{}}`)},
		}
		result := loadHTTPRequestEvent(context.Background(), sl, id)
		require.Nil(t, result)
	})

	t.Run("valid http request event", func(t *testing.T) {
		sl := &mockStateLoader{
			events: []json.RawMessage{
				newHTTPRequestEvent([]byte(`hello`), "text/plain", "a=1"),
			},
		}
		result := loadHTTPRequestEvent(context.Background(), sl, id)
		require.NotNil(t, result)
		require.Equal(t, consts.HttpRequestName, result.Name)
		require.Equal(t, "text/plain", result.Data.ContentType)
		require.Equal(t, "a=1", result.Data.QueryParams)
		require.Equal(t, []byte(`hello`), result.Data.Body)
	})
}
