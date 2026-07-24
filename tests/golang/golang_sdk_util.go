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
	"runtime"
	"sync"
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
		req.Close = true
		// Registration updates are infrequent and CI occasionally reuses a stale
		// idle connection here. Use a one-off transport to avoid surfacing that
		// socket reuse as a flaky test failure.
		resp, err := (&http.Client{
			Transport: &http.Transport{DisableKeepAlives: true},
		}).Do(req)
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

	s := &http.Server{
		Handler:        f,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go func() {
		err := s.Serve(l)
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	// Give it time to start.
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

// RunID provides a safe, one-shot mechanism for capturing a run ID from inside
// a function handler and waiting for it in the test body with a timeout.
type RunID struct {
	ch chan string
}

func NewRunID() *RunID {
	return &RunID{ch: make(chan string, 1)}
}

// Send captures the run ID. Only the first call has any effect (non-blocking).
// Call this inside the function handler.
func (r *RunID) Send(id string) {
	select {
	case r.ch <- id:
	default:
	}
}

// Wait blocks until the run ID is received or 20s elapses (test fails on timeout).
func (r *RunID) Wait(t require.TestingT) string {
	select {
	case id := <-r.ch:
		return id
	case <-time.After(20 * time.Second):
		require.Fail(t, "timed out after 20s waiting for run ID")
		return ""
	}
}

// goroutineT is a require.TestingT that is safe to use from goroutines spawned
// during a test.  require.* and helpers like WaitForRunStatus call FailNow,
// which testing.T only permits from the test goroutine.  goroutineT records
// failures and surfaces them on t via check once the goroutines have finished.
type goroutineT struct {
	mu   sync.Mutex
	msgs []string
}

func (g *goroutineT) Errorf(format string, args ...any) {
	g.mu.Lock()
	g.msgs = append(g.msgs, fmt.Sprintf(format, args...))
	g.mu.Unlock()
}

func (g *goroutineT) FailNow() {
	// stop only this goroutine; the deferred wg.Done still runs.
	runtime.Goexit()
}

// check reports recorded failures on t.  call it from the test goroutine after
// the spawned goroutines finish (e.g. after wg.Wait).
func (g *goroutineT) check(t *testing.T) {
	t.Helper()
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, m := range g.msgs {
		t.Error(m)
	}
}
