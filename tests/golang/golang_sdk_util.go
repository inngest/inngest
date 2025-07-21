package golang

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"

	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

const DEV_URL = "http://127.0.0.1:8288"

type RegisterFunc func()

type opt func(h *inngestgo.ClientOpts)

func NewSDKHandler(t *testing.T, appID string, copts ...opt) (inngestgo.Client, *HTTPServer, RegisterFunc) {
	t.Helper()
	_ = os.Setenv("INNGEST_DEV", DEV_URL)

	key := "test"
	opts := inngestgo.ClientOpts{
		AppID:       appID,
		EventKey:    &key,
		Logger:      slog.New(slog.DiscardHandler),
		RegisterURL: inngestgo.StrPtr(fmt.Sprintf("%s/fn/register", DEV_URL)),
	}

	for _, o := range copts {
		o(&opts)
	}

	client, err := inngestgo.NewClient(opts)
	require.NoError(t, err)

	server := NewHTTPServer(client.Serve())

	// Update the handler's URL with this server.
	opts.URL, _ = url.Parse(server.URL())
	client.SetURL(opts.URL)
	<-time.After(20 * time.Millisecond)

	r := func() {
		t.Helper()
		req, err := http.NewRequest(http.MethodPut, server.LocalURL(), nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		fmt.Println(string(body))
		require.Equal(t, 200, resp.StatusCode)
		_ = resp.Body.Close()
	}
	return client, server, r
}

func NewSDKConnectHandler(t *testing.T, appID string, copts ...opt) inngestgo.Client {
	t.Helper()
	_ = os.Setenv("INNGEST_DEV", DEV_URL)

	key := "test"
	opts := inngestgo.ClientOpts{
		AppID:       appID,
		EventKey:    &key,
		Logger:      logger.StdlibLogger(context.Background()).SLog(),
		RegisterURL: inngestgo.StrPtr(fmt.Sprintf("%s/fn/register", DEV_URL)),
	}

	for _, o := range copts {
		o(&opts)
	}

	client, err := inngestgo.NewClient(opts)
	require.NoError(t, err)

	return client
}

type HTTPServer struct {
	*http.Server

	Port int32
}

func (h HTTPServer) URL() string {
	// This can be accessed by any docker container with the added network of
	// `host.docker.internal:host-gateway`.
	//
	// We need this for testing actions, as the executor runs in a docker container
	// and cannot access 127.0.0.1 (which httptest listens on).
	// return fmt.Sprintf("http://host.docker.internal:%d/", h.Port)
	return fmt.Sprintf("http://127.0.0.1:%d/", h.Port)
}

func (h HTTPServer) LocalURL() string {
	// This can be accessed by any docker container with the added network of
	// `host.docker.internal:host-gateway`.
	//
	// We need this for testing actions, as the executor runs in a docker container
	// and cannot access 127.0.0.1 (which httptest listens on).
	return fmt.Sprintf("http://127.0.0.1:%d/", h.Port)
}

// NewHTTPServer returns a new HTTP server with the given handler, listening on a
// random port between 10_000 and 65_535 which can be accessed by any host.
//
// This is a copy of httptest, but listens on the docker gateway instead of 127.0.0.1
// only - which doesn't work in tests as the executor's localhost is different.
func NewHTTPServer(f http.Handler) *HTTPServer {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	s := &http.Server{
		Addr:           fmt.Sprintf("0.0.0.0:%d", port),
		Handler:        f,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go func() {
		err := s.ListenAndServe()
		// Check if server is closed error
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	// Give it ime to start.
	<-time.After(20 * time.Millisecond)

	return &HTTPServer{Server: s, Port: int32(port)}
}

func NewHTTPSServer(f http.Handler) *HTTPServer {
	var port int32
	for port < 10000 {
		port = rand.Int31n(65335)
	}

	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        f,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go func() {
		err := s.ListenAndServe()
		if err != nil {
			fmt.Println(err)
		}
	}()

	return &HTTPServer{Server: s, Port: port}
}

func randomSuffix(s string) string {
	return fmt.Sprintf("%s-%s", s, uuid.NewString())
}
