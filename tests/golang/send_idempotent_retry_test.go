package golang

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

func TestSendIdempotentRetry(t *testing.T) {
	// Resending events with the same idempotency key header results in skipped
	// function runs.

	ctx := context.Background()
	r := require.New(t)

	var proxyCounter int32

	// Create a proxy that mimics a request reaching the Dev Server but the
	// client receives a 500 on the first attempt. This ensures that the Dev
	// Server's event processing logic properly handles the idempotency key
	// header.
	proxy := NewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&proxyCounter, 1)

		// Always forward requests.
		req, _ := http.NewRequest(
			r.Method,
			fmt.Sprintf("http://0.0.0.0:8288%s", r.URL.Path),
			r.Body,
		)
		req.Header = r.Header
		resp, _ := http.DefaultClient.Do(req)

		if proxyCounter == 1 {
			// Return a 500 on the first attempt.
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			// Forward the response from the Dev Server.
			for k, v := range resp.Header {
				w.Header()[k] = v
			}
			w.WriteHeader(resp.StatusCode)
			_, _ = io.Copy(w, resp.Body)
			resp.Body.Close()
		}
	}))
	defer proxy.Close()
	proxyURL, err := url.Parse(proxy.URL())
	r.NoError(err)

	ic, server, registerFuncs := NewSDKHandler(
		t,
		randomSuffix("app"),
		func(h *inngestgo.ClientOpts) {
			h.EventURL = inngestgo.Ptr(strings.TrimSuffix(proxyURL.String(), "/"))
		},
	)
	defer server.Close()

	eventName := randomSuffix("event")
	var fnCounter int32
	_, err = inngestgo.CreateFunction(
		ic,
		inngestgo.FunctionOpts{ID: "fn"},
		inngestgo.EventTrigger(eventName, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			fmt.Println("trigger")
			atomic.AddInt32(&fnCounter, 1)
			return nil, nil
		},
	)
	r.NoError(err)
	registerFuncs()

	// Send two events with the same idempotency key header. Both events trigger
	// the function.
	err = sendEvents(
		ctx,
		strings.TrimSuffix(proxyURL.String(), "/"),
		[]inngestgo.Event{
			{Name: eventName},
			{Name: eventName},
		},
	)
	r.NoError(err)

	// Sleep long enough for the Dev Server to process the events.
	time.Sleep(5 * time.Second)
	r.Equal(int32(2), atomic.LoadInt32(&proxyCounter))
	r.Equal(int32(2), atomic.LoadInt32(&fnCounter))
}

// sendEvents mimics the idempotent retry logic of the inngestgo client.
//
// TODO: Move this to the inngestgo package once we implement the idempotent
// retry logic in the Dev Server and Cloud.
func sendEvents(
	ctx context.Context,
	eventURL string,
	events []inngestgo.Event,
) error {
	byt, err := json.Marshal(events)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/e/test", eventURL),
		bytes.NewBuffer(byt),
	)
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json")

	// Create and set the idempotency key header.
	millis := time.Now().UnixMilli()
	randomByt := make([]byte, 10)
	_, err = rand.Read(randomByt)
	if err != nil {
		return err
	}
	randomBase64 := base64.StdEncoding.EncodeToString(randomByt)
	req.Header.Set(
		"x-inngest-idempotency-key",
		fmt.Sprintf("%d,%s", millis, randomBase64),
	)

	var resp *http.Response
	maxAttempts := 5
	for i := 1; i <= maxAttempts; i++ {
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode < 300 {
			break
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("status code %d", resp.StatusCode)
	}

	return nil
}
