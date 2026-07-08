package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/registration"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/util"
	"github.com/stretchr/testify/require"
)

const (
	ENV_SDK_URL     = "SDK_URL"   // eg. http://127.0.0.1:3000/api/inngest
	ENV_API_URL     = "API_URL"   // eg http://127.0.0.1:8288 or https://api.inngest.com
	ENV_EVENT_URL   = "EVENT_URL" // eg http://127.0.0.1:8288 or https://inn.gs
	ENV_SIGNING_KEY = "INNGEST_SIGNING_KEY"
	ENV_EVENT_KEY   = "INNGEST_EVENT_KEY"
	ENV_PROXY_URL   = "PROXY_URL"
)

var (
	sdkURL               url.URL
	apiURL               url.URL
	eventURL             url.URL
	signingKey, eventKey string
	proxyURL             string

	buffer = 5 * time.Second
)

func init() {
	sdkURL = parseEnvURL(ENV_SDK_URL)
	apiURL = parseEnvURL(ENV_API_URL)
	// If the ENV_EVENT_URL isn't specified default to the API url, assuming that this
	// is the dev server and runs both APIs.
	if os.Getenv(ENV_EVENT_URL) == "" {
		_ = os.Setenv(ENV_EVENT_URL, apiURL.String())
	}
	eventURL = parseEnvURL(ENV_EVENT_URL)
	signingKey = os.Getenv(ENV_SIGNING_KEY)
	eventKey = os.Getenv(ENV_EVENT_KEY)
	if eventKey == "" {
		eventKey = "eventkey"
	}

	proxyURL = os.Getenv(ENV_PROXY_URL)
	if proxyURL == "" {
		proxyURL = "http://localhost"
	}
}

func parseEnvURL(env string) url.URL {
	u, err := url.Parse(os.Getenv(env))
	if err != nil {
		panic(err.Error())
	}
	if u.Host == "" {
		panic(fmt.Sprintf("No %s env variable supplied", env))
	}
	return *u
}

func run(t *testing.T, test *Test) {
	t.Helper()

	fmt.Println("")
	fmt.Println("")
	header := fmt.Sprintf("Running test: %s", t.Name())
	fmt.Println(header)
	for i := 0; i < len(header); i++ {
		fmt.Printf("=")
	}
	fmt.Println("")
	fmt.Println("")

	test.requests = make(chan http.Request)
	test.responses = make(chan http.Response)

	// Intercept all traffic between the executor and the SDK so we can assert
	// on the messages passed in both directions (SDK responses, and the state
	// the executor injects).
	//
	// This is a reverse proxy rather than a hand-rolled forwarder: we forward
	// each request/response *verbatim* and only ever decode a copy for our own
	// assertions. Letting net/http own the HTTP framing (Content-Length,
	// chunked transfer, Content-Encoding, hop-by-hop headers) means the bytes
	// the executor receives always match their headers — hand-copying headers
	// while transforming the body is exactly how a gzipped body got forwarded
	// with a stale gzip header before.
	target := sdkURL
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			// Route to the real SDK endpoint, preserving the executor's query
			// (fnId/stepId) and ignoring the inbound "/" path.
			pr.Out.URL.Scheme = target.Scheme
			pr.Out.URL.Host = target.Host
			pr.Out.URL.Path = target.Path
			pr.Out.URL.RawQuery = pr.In.URL.RawQuery
			pr.Out.Host = target.Host
		},
		ModifyResponse: func(resp *http.Response) error {
			// Read the body so we can capture it, then hand it straight back
			// for forwarding: original bytes, original encoding, untouched.
			raw, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				return err
			}
			resp.Body = io.NopCloser(bytes.NewReader(raw))

			// Decode a copy purely for assertions/logging; never mutate resp.
			decoded := raw
			if resp.Header.Get("Content-Encoding") == "gzip" {
				if zr, zerr := gzip.NewReader(bytes.NewReader(raw)); zerr == nil {
					if d, derr := io.ReadAll(zr); derr == nil {
						decoded = d
					}
					_ = zr.Close()
				}
			}

			fmt.Printf(" ==> Received SDK response:\n\t%s\n\n", string(decoded))

			respCopy := *resp
			respCopy.Body = io.NopCloser(bytes.NewReader(decoded))
			select {
			case test.responses <- respCopy:
			case <-time.After(time.Second):
				// Fail from the test goroutine's perspective; calling
				// require.FailNow here (a non-test goroutine) would only
				// Goexit this handler, not fail the test.
				t.Errorf("unexpected sdk response")
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			t.Errorf("error forwarding request to SDK: %v", err)
			w.WriteHeader(http.StatusBadGateway)
		},
	}

	// Create a new server on a random port that listens on 0.0.0.0.
	// This means we cannot use httptest.NewServer
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Ignore ping requests
		if r.Method == http.MethodPut {
			_ = r.Body.Close()
			return
		}

		// Capture the executor's request body for assertions, then restore it
		// so the proxy forwards the request unchanged.
		byt, err := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if err != nil {
			t.Errorf("error reading executor request: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(byt))

		fmt.Printf(" ==> Received executor request:\n\t%s\n", string(byt))

		reqCopy := *r
		reqCopy.Body = io.NopCloser(bytes.NewReader(byt))
		select {
		case test.requests <- reqCopy:
		case <-time.After(time.Second):
			t.Errorf("unexpected executor request")
		}

		proxy.ServeHTTP(w, r)
	})
	port := rand.Int63n(10000) + 40000
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	go func() {
		_ = srv.ListenAndServe()
	}()
	defer srv.Close()

	var err error

	test.localURL, err = url.Parse(fmt.Sprintf("%s:%d", proxyURL, port))
	require.NoError(t, err)

	// Ensure that the desired function exists within the SDK.
	rr, err := introspect(test)
	require.NoError(t, err, "Introspection error")

	// Register all functions with the SDK.
	err = register(*test.localURL, *rr)
	require.NoError(t, err, "Function registration error")

	defer func() {
		// De-register the app.
		url := apiURL
		url.Path = "/fn/remove"

		fv := url.Query()
		fv.Add("url", test.localURL.String())

		req, err := http.NewRequest(http.MethodDelete, url.String()+"?"+fv.Encode(), nil)
		if err != nil {
			fmt.Println("Error removing app after test", err)
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("Error removing app after test", err)
		}
		resp.Body.Close()
	}()

	test.test = t
	for _, f := range test.chain {
		f()
	}

	fmt.Printf("\n===> Waiting for extraneous requests\n")
	<-time.After(test.Timeout + buffer)
	fmt.Printf("\n\n")
}

// introspect asserts that the SDK is live and the expected function exists when calling
// the introspect handler.
func introspect(test *Test) (*sdk.RegisterRequest, error) {
	url := sdkURL
	// A custom URL
	url.Path = "/api/introspect"

	resp, err := http.Get(url.String())
	if err != nil {
		return nil, fmt.Errorf("unable to call introspect on SDK: %w", err)
	}
	defer resp.Body.Close()

	fns := []sdk.SDKFunction{}
	if err := json.NewDecoder(resp.Body).Decode(&fns); err != nil {
		return nil, fmt.Errorf("invalid introspect response: unable to decode introspect response: %w", err)
	}

	rr := &sdk.RegisterRequest{
		URL:       "http://127.0.0.1:3000/api/inngest",
		AppName:   "test-suite",
		Functions: fns,
	}

	result, err := registration.ProcessFunctions(context.Background(), *rr, registration.ProcessOpts{})
	if err != nil {
		return nil, err
	}

	found := false
	for _, df := range result.Functions {
		f := &df.Function
		for i := range f.Steps {
			forceHTTPS := false
			f.Steps[i].URI = util.NormalizeAppURL(f.Steps[i].URI, forceHTTPS)
		}
		if f.Slug == test.ID {
			found = true
		}
	}

	response, _ := json.MarshalIndent(result.Functions, "", "  ")
	if !found {
		return nil, fmt.Errorf("Expected function not found:\n%s\n\nIntrospection:\n%s", test.ID, string(response))
	}

	return rr, nil
}

func replaceURL(nodeURL, proxyURL string) string {
	// Take the URL and replace the host with our server's URL.
	node, err := url.Parse(nodeURL)
	if err != nil {
		return ""
	}
	proxy, _ := url.Parse(proxyURL)
	proxy.Path = "/"
	proxy.RawQuery = node.RawQuery
	return proxy.String()
}

func register(serverURL url.URL, rr sdk.RegisterRequest) error {
	// Register functions using _this_ host and the introspection request
	for n, fn := range rr.Functions {
		for key, step := range fn.Steps {
			nodeURL, _ := step.Runtime["url"].(string)
			step.Runtime["url"] = replaceURL(nodeURL, serverURL.String())
			fn.Steps[key] = step
		}
		rr.Functions[n] = fn
	}

	rr.URL = serverURL.String()

	byt, err := json.Marshal(rr)
	if err != nil {
		return err
	}

	url := apiURL
	url.Path = "/fn/register"
	req, err := http.NewRequest(http.MethodPost, url.String(), bytes.NewReader(byt))
	if err != nil {
		return err
	}

	key := regexp.MustCompile(`^signkey-[\w]+-`).ReplaceAllString(signingKey, "")
	byt, _ = hex.DecodeString(key)
	sum := sha256.Sum256(byt)
	keyHash := hex.EncodeToString(sum[:])

	req.Header.Add("Authorization", fmt.Sprintf("Bearer signkey-test-%s", keyHash))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error registering: %w", err)
	}
	if resp.StatusCode > 299 {
		byt, _ := httputil.DumpResponse(resp, true)
		return fmt.Errorf("Error when registering functions: %s", string(byt))
	}

	return nil
}
