package exechttp

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/stretchr/testify/require"
)

func silenceLogger(ctx context.Context) context.Context {
	return logger.WithStdlib(
		ctx,
		logger.StdlibLogger(ctx, logger.WithLoggerLevel(logger.LevelEmergency)),
	)
}

func TestDoRequest_PublishFailure_PreservesBody(t *testing.T) {
	r := require.New(t)
	ctx := silenceLogger(t.Context())

	expectedBody := `{"result":"hello"}`
	mockSDK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(expectedBody))
	}))
	defer mockSDK.Close()

	mockAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockAPI.Close()

	client := ExtendedClient{
		Client:  mockSDK.Client(),
		publish: true,
	}
	req := SerializableRequest{
		Method: http.MethodPost,
		URL:    mockSDK.URL,
		Body:   json.RawMessage(`{}`),
		Header: http.Header{},
		Publish: RequestPublishOpts{
			Channel:    "test-channel",
			Topic:      "test-topic",
			Token:      "test-token",
			PublishURL: mockAPI.URL,
			RequestID:  "step-1",
		},
	}
	resp, err := client.DoRequest(ctx, req)
	r.NoError(err)
	r.NotNil(resp)
	r.Equal(http.StatusOK, resp.StatusCode)
	r.Equal(expectedBody, string(resp.Body))
}

func TestDoRequest_PublishSuccess_PreservesBody(t *testing.T) {
	r := require.New(t)
	ctx := silenceLogger(t.Context())

	expectedBody := `{"result":"hello"}`
	mockSDK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(expectedBody))
	}))
	defer mockSDK.Close()

	var publishedChannel, publishedTopic string
	mockAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		publishedChannel = r.URL.Query().Get("channel")
		publishedTopic = r.URL.Query().Get("topic")
		w.WriteHeader(http.StatusOK)
	}))
	defer mockAPI.Close()

	client := ExtendedClient{
		Client:  mockSDK.Client(),
		publish: true,
	}
	req := SerializableRequest{
		Method: http.MethodPost,
		URL:    mockSDK.URL,
		Body:   json.RawMessage(`{}`),
		Header: http.Header{},
		Publish: RequestPublishOpts{
			Channel:    "test-channel",
			Topic:      "test-topic",
			Token:      "test-token",
			PublishURL: mockAPI.URL,
			RequestID:  "step-1",
		},
	}
	resp, err := client.DoRequest(ctx, req)
	r.NoError(err)
	r.NotNil(resp)
	r.Equal(http.StatusOK, resp.StatusCode)
	r.Equal(expectedBody, string(resp.Body))
	r.Equal("test-channel", publishedChannel)
	r.Equal("test-topic", publishedTopic)
}

func TestDoRequest_PublishDisabled_PreservesBody(t *testing.T) {
	r := require.New(t)
	ctx := silenceLogger(t.Context())

	expectedBody := `{"result":"hello"}`
	mockSDK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(expectedBody))
	}))
	defer mockSDK.Close()

	// publish is false, so publish opts should be ignored entirely.
	client := ExtendedClient{
		Client:  mockSDK.Client(),
		publish: false,
	}
	req := SerializableRequest{
		Method: http.MethodPost,
		URL:    mockSDK.URL,
		Body:   json.RawMessage(`{}`),
		Header: http.Header{},
		Publish: RequestPublishOpts{
			Channel:    "test-channel",
			Topic:      "test-topic",
			Token:      "test-token",
			PublishURL: "http://should-not-be-called",
			RequestID:  "step-1",
		},
	}
	resp, err := client.DoRequest(ctx, req)
	r.NoError(err)
	r.NotNil(resp)
	r.Equal(http.StatusOK, resp.StatusCode)
	r.Equal(expectedBody, string(resp.Body))
}

func TestDoRequest_PublishEndpointUnreachable_PreservesBody(t *testing.T) {
	r := require.New(t)
	ctx := silenceLogger(t.Context())

	expectedBody := `{"result":"hello"}`
	sdk := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(expectedBody))
	}))
	defer sdk.Close()

	client := ExtendedClient{
		Client:  sdk.Client(),
		publish: true,
	}

	// Point to a closed server so the connection fails.
	mockAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	mockAPI.Close()

	req := SerializableRequest{
		Method: http.MethodPost,
		URL:    sdk.URL,
		Body:   json.RawMessage(`{}`),
		Header: http.Header{},
		Publish: RequestPublishOpts{
			Channel:    "test-channel",
			Topic:      "test-topic",
			Token:      "test-token",
			PublishURL: mockAPI.URL,
			RequestID:  "step-1",
		},
	}
	resp, err := client.DoRequest(ctx, req)
	r.NoError(err)
	r.NotNil(resp)
	r.Equal(http.StatusOK, resp.StatusCode)
	r.Equal(expectedBody, string(resp.Body))
}

func TestDoRequest_DecodesBrotliResponse(t *testing.T) {
	r := require.New(t)
	ctx := silenceLogger(t.Context())

	expectedBody := `{"result":"hello"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "br")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(brotliCompressed(t, expectedBody))
	}))
	defer server.Close()

	client := ExtendedClient{Client: server.Client()}
	resp, err := client.DoRequest(ctx, SerializableRequest{
		Method: http.MethodPost,
		URL:    server.URL,
		Body:   json.RawMessage(`{}`),
		Header: http.Header{"Accept-Encoding": []string{"br"}},
	})

	r.NoError(err)
	r.NotNil(resp)
	r.Equal(http.StatusOK, resp.StatusCode)
	r.Equal(expectedBody, string(resp.Body))
}

func TestDoRequest_DecodesExplicitGzipResponse(t *testing.T) {
	r := require.New(t)
	ctx := silenceLogger(t.Context())

	expectedBody := `{"result":"hello"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(gzipCompressed(t, expectedBody))
	}))
	defer server.Close()

	client := ExtendedClient{Client: server.Client()}
	resp, err := client.DoRequest(ctx, SerializableRequest{
		Method: http.MethodPost,
		URL:    server.URL,
		Body:   json.RawMessage(`{}`),
		Header: http.Header{"Accept-Encoding": []string{"gzip"}},
	})

	r.NoError(err)
	r.NotNil(resp)
	r.Equal(http.StatusOK, resp.StatusCode)
	r.Equal(expectedBody, string(resp.Body))
}

func TestDoRequest_RejectsUnsupportedContentEncoding(t *testing.T) {
	r := require.New(t)
	ctx := silenceLogger(t.Context())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "zstd")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"hello"}`))
	}))
	defer server.Close()

	client := ExtendedClient{Client: server.Client()}
	resp, err := client.DoRequest(ctx, SerializableRequest{
		Method: http.MethodPost,
		URL:    server.URL,
		Body:   json.RawMessage(`{}`),
		Header: http.Header{},
	})

	r.Nil(resp)
	r.ErrorContains(err, `unsupported content encoding: "zstd"`)
}

func TestDoRequest_RejectsInvalidCompressedResponse(t *testing.T) {
	r := require.New(t)
	ctx := silenceLogger(t.Context())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)
		compressed := gzipCompressed(t, `{"result":"hello"}`)
		_, _ = w.Write(compressed[:len(compressed)-4])
	}))
	defer server.Close()

	client := ExtendedClient{Client: server.Client()}
	resp, err := client.DoRequest(ctx, SerializableRequest{
		Method: http.MethodPost,
		URL:    server.URL,
		Body:   json.RawMessage(`{}`),
		Header: http.Header{"Accept-Encoding": []string{"gzip"}},
	})

	r.Nil(resp)
	r.ErrorContains(err, "error decoding gzip response body")
}

func TestDecompressBody_Gzip(t *testing.T) {
	r := require.New(t)
	original := `{"result":"hello"}`
	compressed := gzipCompressed(t, original)

	decoded, err := DecompressBody(compressed, "gzip")
	r.NoError(err)
	r.Equal(original, string(decoded))
}

func TestDecompressBody_Brotli(t *testing.T) {
	r := require.New(t)
	original := `{"result":"hello"}`
	compressed := brotliCompressed(t, original)

	decoded, err := DecompressBody(compressed, "br")
	r.NoError(err)
	r.Equal(original, string(decoded))
}

func TestDecompressBody_EmptyEncoding(t *testing.T) {
	r := require.New(t)
	original := []byte(`{"result":"hello"}`)

	decoded, err := DecompressBody(original, "")
	r.NoError(err)
	r.Equal(original, decoded)
}

func TestDecompressBody_Unsupported(t *testing.T) {
	r := require.New(t)
	_, err := DecompressBody([]byte("data"), "zstd")
	r.ErrorContains(err, `unsupported content encoding: "zstd"`)
}

func TestDoRequest_ClearsContentEncodingAfterDecode(t *testing.T) {
	r := require.New(t)
	ctx := silenceLogger(t.Context())

	expectedBody := `{"result":"hello"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Encoding", "br")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(brotliCompressed(t, expectedBody))
	}))
	defer server.Close()

	client := ExtendedClient{Client: server.Client()}
	resp, err := client.DoRequest(ctx, SerializableRequest{
		Method: http.MethodPost,
		URL:    server.URL,
		Body:   json.RawMessage(`{}`),
		Header: http.Header{"Accept-Encoding": []string{"br"}},
	})

	r.NoError(err)
	r.NotNil(resp)
	r.Equal(expectedBody, string(resp.Body))
	r.Empty(resp.Header.Get("Content-Encoding"), "Content-Encoding should be cleared after decompression")
}

func brotliCompressed(t *testing.T, input string) []byte {
	t.Helper()

	var buf bytes.Buffer
	writer := brotli.NewWriter(&buf)
	_, err := writer.Write([]byte(input))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	return buf.Bytes()
}

func gzipCompressed(t *testing.T, input string) []byte {
	t.Helper()

	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	_, err := writer.Write([]byte(input))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	return buf.Bytes()
}
