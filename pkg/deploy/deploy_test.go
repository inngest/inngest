package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPing(t *testing.T) {
	newFakeSDK := func(handler func(w http.ResponseWriter, r *http.Request)) (string, func(), error) {
		mux := http.NewServeMux()

		mux.HandleFunc("/", handler)

		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			return "", nil, err
		}

		port := listener.Addr().(*net.TCPAddr).Port
		url := fmt.Sprintf("http://localhost:%d", port)

		server := &http.Server{
			Handler: mux,
		}

		go func() {
			_ = server.Serve(listener)
		}()

		close := func() {
			_ = server.Close()
		}

		return url, close, nil
	}

	t.Run("headers and body are correct", func(t *testing.T) {
		ctx := context.Background()
		r := require.New(t)

		var reqHeader http.Header
		var reqBody map[string]any
		url, close, err := newFakeSDK(func(w http.ResponseWriter, r *http.Request) {
			reqHeader = r.Header
			byt, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(byt, &reqBody)
			w.WriteHeader(http.StatusOK)
		})
		r.NoError(err)
		defer close()

		Ping(ctx, url, "my-server-kind", "deadbeef", true)
		r.Equal(map[string]any{"url": url}, reqBody)

		// We need this check to prevent a regression. Apparently using
		// io.NopCloser with a request causes a missing content-length header
		r.Equal("32", reqHeader.Get("content-length"))

		r.Equal("application/json", reqHeader.Get("content-type"))
		r.Equal("my-server-kind", reqHeader.Get("x-inngest-server-kind"))
		r.NotEmpty(reqHeader.Get("x-inngest-signature"))
	})
}
