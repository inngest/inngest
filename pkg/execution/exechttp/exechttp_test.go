package exechttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
