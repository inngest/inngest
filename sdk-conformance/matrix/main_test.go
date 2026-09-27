package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/inngest/inngestgo"
)

func TestSubstituteUsesFixtureValuesThroughoutRequest(t *testing.T) {
	input := probeRequest{
		Path:  "{{serve_path}}",
		Query: map[string]string{"fnId": "{{function_id}}"},
		JSON:  map[string]any{"event": "{{event}}"},
	}
	got, err := substitute(input, map[string]string{
		"serve_path":  "/api/inngest",
		"function_id": "app-fn",
		"event":       "test/event",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != "/api/inngest" || got.Query["fnId"] != "app-fn" {
		t.Fatalf("unexpected request: %#v", got)
	}
	if !reflect.DeepEqual(got.JSON, map[string]any{"event": "test/event"}) {
		t.Fatalf("unexpected JSON: %#v", got.JSON)
	}
}

func TestRunProbeSignsRequestWithoutBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Inngest-Signature") == "" {
			t.Error("missing signature")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	got := runProbe(server.Client(), server.URL, manifest{CloudServePath: "/cloud"}, probe{
		Request: probeRequest{Method: http.MethodGet, Path: "{{cloud_serve_path}}", Sign: true},
	})
	if got.Error != "" || got.Status != http.StatusOK {
		t.Fatalf("unexpected observation: %#v", got)
	}
}

func TestConfiguredSigningKeysAndTimestampOffsets(t *testing.T) {
	now := time.Now()
	for name, key := range signingKeys {
		signature, err := inngestgo.Sign(t.Context(), now.Add(-10*time.Minute), key, nil)
		if err != nil {
			t.Fatalf("sign with %s key: %v", name, err)
		}
		values, err := url.ParseQuery(signature)
		if err != nil {
			t.Fatalf("parse signature: %v", err)
		}
		got, err := strconv.ParseInt(values.Get("t"), 10, 64)
		if err != nil {
			t.Fatalf("parse timestamp: %v", err)
		}
		if want := now.Add(-10 * time.Minute).Unix(); got != want {
			t.Fatalf("timestamp = %d, want %d", got, want)
		}
	}
}

func TestNormalizeRemovesOnlyVolatileProtocolMetadata(t *testing.T) {
	input := map[string]any{
		"sdk_language": "typescript",
		"data": map[string]any{
			"stack": "volatile",
			"value": "preserved",
		},
	}
	want := map[string]any{"data": map[string]any{"value": "preserved"}}
	if got := normalizeJSON(input); !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeJSON() = %#v, want %#v", got, want)
	}
}
