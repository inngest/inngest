package golang

import (
	"context"
	"fmt"
	"github.com/inngest/inngest/pkg/logger"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

const DEV_URL = "http://127.0.0.1:8288"

type RegisterFunc func()

type opt func(h *inngestgo.HandlerOpts)

func NewSDKHandler(t *testing.T, appID string, hopts ...opt) (inngestgo.Handler, *HTTPServer, RegisterFunc) {
	t.Helper()

	key := "test"
	inngestgo.DefaultClient = inngestgo.NewClient(inngestgo.ClientOpts{
		EventKey: &key,
	})

	_ = os.Setenv("INNGEST_DEV", DEV_URL)

	opts := inngestgo.HandlerOpts{
		RegisterURL: inngestgo.StrPtr(fmt.Sprintf("%s/fn/register", DEV_URL)),
		Logger:      slog.Default(),
		// Env:         inngestgo.Str("test-env"),
	}

	for _, o := range hopts {
		o(&opts)
	}

	h := inngestgo.NewHandler(appID, opts)

	server := NewHTTPServer(h)

	// Update the handler's URL with this server.
	opts.URL, _ = url.Parse(server.URL())
	h.SetOptions(opts)
	<-time.After(20 * time.Millisecond)

	r := func() {
		t.Helper()
		req, err := http.NewRequest(http.MethodPut, server.LocalURL(), nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode)
		_ = resp.Body.Close()
	}
	return h, server, r
}

func WithBuildId(buildId string) opt {
	return func(h *inngestgo.HandlerOpts) {
		h.BuildId = &buildId
	}
}

func WithInstanceId(instanceId string) opt {
	return func(h *inngestgo.HandlerOpts) {
		h.InstanceId = &instanceId
	}
}

func NewSDKConnectHandler(t *testing.T, appID string, hopts ...opt) inngestgo.Handler {
	t.Helper()

	key := "test"
	inngestgo.DefaultClient = inngestgo.NewClient(inngestgo.ClientOpts{
		EventKey: &key,
	})

	_ = os.Setenv("INNGEST_DEV", DEV_URL)

	opts := inngestgo.HandlerOpts{
		RegisterURL: inngestgo.StrPtr(fmt.Sprintf("%s/fn/register", DEV_URL)),
		Logger:      logger.StdlibLogger(context.Background()),
	}

	for _, o := range hopts {
		o(&opts)
	}

	h := inngestgo.NewHandler(appID, opts)

	return h
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
	var port int32
	for port < 10000 {
		port = rand.Int31n(65335)
	}

	s := &http.Server{
		Addr:           fmt.Sprintf("0.0.0.0:%d", port),
		Handler:        f,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go func() {
		_ = s.ListenAndServe()
	}()

	return &HTTPServer{Server: s, Port: port}
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
