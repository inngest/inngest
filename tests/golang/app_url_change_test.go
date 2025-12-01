package golang

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/inngest/inngestgo"
)

func TestAppURLChange(t *testing.T) {
	t.Setenv("INNGEST_DEV", "1")

	sync := func(t *testing.T, u string) {
		r := require.New(t)

		r.EventuallyWithT(func(t *assert.CollectT) {
			r := require.New(t)
			req, err := http.NewRequest(http.MethodPut, u, nil)
			r.NoError(err)
			resp, err := http.DefaultClient.Do(req)
			r.NoError(err)
			r.Equal(200, resp.StatusCode)
			_ = resp.Body.Close()
		}, 5*time.Second, 100*time.Millisecond)
	}

	t.Run("resync with new URL", func(t *testing.T) {
		// Resyncing with a new URL changes the app URL

		r := require.New(t)
		ctx := context.Background()

		// Create an app
		ic, err := inngestgo.NewClient(
			inngestgo.ClientOpts{
				AppID:       randomSuffix("app"),
				Dev:         inngestgo.BoolPtr(true),
				RegisterURL: inngestgo.StrPtr(fmt.Sprintf("%s/fn/register", DEV_URL)),
			},
		)
		r.NoError(err)
		eventName := randomSuffix("event")
		_, err = inngestgo.CreateFunction(
			ic,
			inngestgo.FunctionOpts{
				ID:      "fn",
				Retries: inngestgo.IntPtr(0),
			},
			inngestgo.EventTrigger(eventName, nil),
			func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
				return nil, nil
			},
		)
		r.NoError(err)
		server := NewHTTPServer(ic.Serve())
		defer server.Close()
		appURL, err := url.Parse(server.LocalURL())
		r.NoError(err)

		// Create 2 proxies and a slice that tracks their POST request hosts.
		// This is how we'll assert that the app URL changes
		var postRequestHosts []string
		proxy1URL, cleanup := createProxy(t, appURL, func(req *http.Request) {
			fmt.Println("req", req.Method, req.Host)
			if req.Method == http.MethodPost {
				postRequestHosts = append(postRequestHosts, req.Host)
			}
		})
		defer cleanup()
		proxy2URL, cleanup := createProxy(t, appURL, func(req *http.Request) {
			fmt.Println("req", req.Method, req.Host)
			if req.Method == http.MethodPost {
				postRequestHosts = append(postRequestHosts, req.Host)
			}
		})
		defer cleanup()

		// Sync via proxy 1. Execution requests go via proxy 1
		sync(t, proxy1URL.String())
		ic.Send(ctx, inngestgo.Event{Name: eventName})
		r.EventuallyWithT(func(t *assert.CollectT) {
			require.Equal(t,
				[]string{proxy1URL.Host},
				postRequestHosts,
			)
		}, 5*time.Second, 100*time.Millisecond)

		// Sync via proxy 2. Execution requests go via proxy 2. This proves that
		// resyncing with a new URL changes the app URL
		sync(t, proxy2URL.String())
		ic.Send(ctx, inngestgo.Event{Name: eventName})
		r.EventuallyWithT(func(t *assert.CollectT) {
			require.Equal(t,
				[]string{proxy1URL.Host, proxy2URL.Host},
				postRequestHosts,
			)
		}, 5*time.Second, 100*time.Millisecond)
	})
}

func createProxy(
	t *testing.T,
	targetURL *url.URL,
	onRequest func(req *http.Request),
) (*url.URL, func()) {
	r := require.New(t)

	var proxyURL *url.URL
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		onRequest(req)
		originalDirector(req)
		req.Host = proxyURL.Host
	}
	listener, err := net.Listen("tcp", ":0")
	r.NoError(err)
	proxyServer := &http.Server{
		Handler: proxy,
	}
	go func() {
		_ = proxyServer.Serve(listener)
	}()

	// Give it time to start
	time.Sleep(20 * time.Millisecond)

	proxyURL, err = url.Parse(fmt.Sprintf("http://localhost:%d", listener.Addr().(*net.TCPAddr).Port))
	r.NoError(err)

	return proxyURL, func() {
		_ = proxyServer.Shutdown(context.Background())
	}
}
