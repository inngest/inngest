package httpdriver

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/exechttp"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/stretchr/testify/require"
)

func parseURL(s string) url.URL {
	u, _ := url.Parse(s)
	return *u
}

// TODO:
//
// Test returning a Step opcode with NonRetriableHeader semantics does NOT fill
// .err in DriverResponse.

func TestRedirect(t *testing.T) {
	input := []byte(`{"event":{"name":"hi","data":{}}}`)

	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch count {
		case 8:
			require.Equal(t, http.MethodGet, r.Method)
			byt, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Equal(t, input, byt)
			require.Equal(t, "application/json", r.Header.Get("content-type"))
			_, _ = w.Write([]byte("ok"))
		default:
			w.Header().Add("location", "/redirected/")
			w.WriteHeader(301)
		}
		count++
	}))
	defer ts.Close()

	client := exechttp.Client(exechttp.SecureDialerOpts{AllowPrivate: true})
	res, _, err := do(context.Background(), client, Request{URL: parseURL(ts.URL), Input: input})
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)
	require.Equal(t, []byte("ok"), res.Body)
}

func TestRetryAfter(t *testing.T) {
	input := []byte(`{"event":{"name":"hi","data":{}}}`)
	at := time.Now().Add(6 * time.Hour).Truncate(time.Second).UTC()
	formats := []string{
		time.RFC3339, // Standard
		time.RFC1123, // HTTP spec
	}
	for _, f := range formats {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", at.Format(f))
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"error":true}`))
		}))
		defer ts.Close()

		client := exechttp.Client(exechttp.SecureDialerOpts{AllowPrivate: true})
		res, _, err := do(context.Background(), client, Request{URL: parseURL(ts.URL), Input: input})
		require.NoError(t, err)
		require.Equal(t, 500, res.StatusCode)
		require.Equal(t, []byte(`{"error":true}`), res.Body)
		require.NotNil(t, res.RetryAt)
		require.EqualValues(t, at, *res.RetryAt)
	}
}

func TestParseRetry(t *testing.T) {
	now := time.Now().Truncate(time.Second).UTC()

	t.Run("It clips with too much time", func(t *testing.T) {
		at := now.Add(2 * consts.MaxRetryDuration)
		actual, err := ParseRetry(at.Format(time.RFC3339))
		require.NoError(t, err)
		require.Equal(t, now.Add(consts.MaxRetryDuration), actual)
	})

	t.Run("It clips with too many seconds", func(t *testing.T) {
		at := (2 * consts.MaxRetryDuration)
		actual, err := ParseRetry(strconv.Itoa(int(at.Seconds())))
		require.NoError(t, err)
		require.Equal(t, now.Add(consts.MaxRetryDuration), actual)
	})

	t.Run("It returns a minute in seconds", func(t *testing.T) {
		actual, err := ParseRetry("60")
		require.NoError(t, err)
		require.Equal(t, now.Add(time.Minute), actual)
	})

	t.Run("It uses minimums in seconds", func(t *testing.T) {
		actual, err := ParseRetry("1")
		require.NoError(t, err)
		require.Equal(t, now.Add(consts.MinRetryDuration), actual)
	})

	t.Run("It uses minimums with dates", func(t *testing.T) {
		actual, err := ParseRetry(now.Add(time.Second).Format(time.RFC1123))
		require.NoError(t, err)
		require.Equal(t, now.Add(consts.MinRetryDuration), actual)
	})
}

func TestParseStream(t *testing.T) {
	t.Run("It parses stream responses", func(t *testing.T) {
		byt := []byte(`{"status":200,"body":"hi"}`)
		resp, err := ParseStream(byt)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, StreamResponse{
			StatusCode: 200,
			Body:       []byte("hi"),
		}, *resp)
	})

	t.Run("It parses generators as a stream", func(t *testing.T) {
		gen := []state.GeneratorOpcode{
			{
				Op:   enums.OpcodeStep,
				ID:   "step-id",
				Name: "step name",
				Data: []byte(`"oh hello there"`),
			},
		}
		raw, err := json.Marshal(gen)
		require.NoError(t, err)
		r := StreamResponse{
			StatusCode: 206,
			Body:       raw,
		}
		byt, err := json.Marshal(r)
		require.NoError(t, err)

		actual, err := ParseStream(byt)
		require.NoError(t, err)
		require.NotNil(t, actual)
		require.Equal(t, r, *actual)
	})

	t.Run("It handles double encoding from old SDK releases", func(t *testing.T) {
		gen := []state.GeneratorOpcode{
			{
				Op:   enums.OpcodeStep,
				ID:   "step-id",
				Name: "step name",
				Data: []byte(`"oh hello there"`),
			},
		}

		first, err := json.Marshal(gen)
		require.NoError(t, err)

		// Encode once again
		second, err := json.Marshal(string(first))
		require.NoError(t, err)

		r := StreamResponse{
			StatusCode: 206,
			Body:       second,
		}

		byt, err := json.Marshal(r)
		require.NoError(t, err)

		// We should actually get the first encoding.
		r.Body = first

		actual, err := ParseStream(byt)
		require.NoError(t, err)
		require.NotNil(t, actual)
		require.Equal(t, r, *actual)
	})
}

func TestStreamResponseTooLarge(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, consts.MaxSDKResponseBodySize+2)
		_, err := rand.Read(data)
		require.NoError(t, err)

		// Indicate a streaming response.
		w.WriteHeader(201)
		err = json.NewEncoder(w).Encode(data)
		require.NoError(t, err)
	}))

	defer ts.Close()
	u, _ := url.Parse(ts.URL)
	client := exechttp.Client(exechttp.SecureDialerOpts{AllowPrivate: true})
	r, _, err := do(context.Background(), client, Request{
		URL: *u,
	})

	require.NotNil(t, err)
	require.NotNil(t, r, "expected some response")
	require.NotNil(t, r.SysErr)
	require.Equal(t, r.SysErr.Code, syscode.CodeOutputTooLarge)
}

func TestTiming(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay the read by 1 second.
		<-time.After(time.Second)
		_, _ = io.ReadAll(r.Body)
		r.Body.Close()
		w.WriteHeader(200)
	}))

	defer ts.Close()
	u, _ := url.Parse(ts.URL)
	client := exechttp.Client(exechttp.SecureDialerOpts{AllowPrivate: true})
	r, result, err := do(context.Background(), client, Request{
		URL:   *u,
		Input: []byte("test"),
	})

	require.NotNil(t, r, "got nil response")
	require.Nil(t, err)

	require.True(t, result.StartTransfer > time.Second)
	require.True(t, result.ServerProcessing > time.Second)
	require.True(t, result.Total > time.Second)
	require.Equal(t, strings.ReplaceAll(ts.URL, "http://", ""), fmt.Sprintf("%s:%d", result.ConnectedTo.IP, result.ConnectedTo.Port))
}
