package serve

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/inngest/inngest/pkg/conformance"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngestgo"
)

// Runner executes the Phase 2 serve-mode conformance flow.
//
// The runner intentionally reuses the same protocol behavior that the existing
// integration tests rely on:
// 1. ask the SDK serve endpoint for its real in-band sync payload
// 2. register the synced functions against the dev server
// 3. proxy executor callbacks back into the SDK
// 4. assert the request/response sequence for each selected case
//
// The implementation stays conservative on purpose. It supports a real set of
// portable showcase cases today and reports the rest as not implemented rather
// than pretending broader transport support already exists.
type Runner struct {
	client *http.Client
}

// NewRunner creates a serve runner backed by the provided HTTP client.
func NewRunner(client *http.Client) *Runner {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	return &Runner{client: client}
}

// Check represents a single doctor/prerequisite result.
type Check struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

// Doctor validates the serve prerequisites without executing conformance cases.
func (r *Runner) Doctor(ctx context.Context, plan conformance.RunPlan, runtime conformance.RuntimeConfig) ([]Check, error) {
	checks := make([]Check, 0, 5)

	if runtime.Transport != conformance.TransportServe {
		return nil, fmt.Errorf("serve doctor requires transport %q, got %q", conformance.TransportServe, runtime.Transport)
	}

	if runtime.SDKURL == nil {
		checks = append(checks, Check{Name: "sdk-url", Message: "sdk.url or --sdk-url is required"})
		return checks, nil
	}
	if runtime.APIURL == nil {
		checks = append(checks, Check{Name: "api-url", Message: "dev.url or dev.api_url is required"})
		return checks, nil
	}
	if runtime.EventURL == nil {
		checks = append(checks, Check{Name: "event-url", Message: "dev.url or dev.event_url is required"})
		return checks, nil
	}
	if strings.TrimSpace(runtime.SigningKey) == "" {
		checks = append(checks, Check{Name: "signing-key", Message: "dev.signing_key or --signing-key is required for serve registration"})
		return checks, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, runtime.SDKURL.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		checks = append(checks, Check{Name: "sdk-inspect", Message: err.Error()})
		return checks, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		checks = append(checks, Check{Name: "sdk-inspect", Message: fmt.Sprintf("unexpected status %d", resp.StatusCode)})
		return checks, nil
	}
	checks = append(checks, Check{Name: "sdk-inspect", Passed: true, Message: fmt.Sprintf("serve endpoint returned %d", resp.StatusCode)})

	proxy, err := newProxyServer(r.client, runtime.SDKURL)
	if err != nil {
		return nil, err
	}
	defer proxy.close(context.Background())

	registerRequest, functionsBySlug, err := r.syncSDKFunctions(ctx, runtime, proxy.URL())
	if err != nil {
		checks = append(checks, Check{Name: "sdk-sync", Message: err.Error()})
		return checks, nil
	}
	checks = append(checks, Check{Name: "sdk-sync", Passed: true, Message: fmt.Sprintf("synced %d function(s) from %s", len(registerRequest.Functions), registerRequest.URL)})

	requiredSlugs := requiredFixtureSlugs(plan.Cases)
	if len(requiredSlugs) == 0 {
		checks = append(checks, Check{Name: "fixture-functions", Passed: true, Message: "no serve fixture functions required"})
		return checks, nil
	}

	missing := make([]string, 0)
	for _, slug := range requiredSlugs {
		if !hasSyncedFunction(functionsBySlug, slug) {
			missing = append(missing, slug)
		}
	}

	if len(missing) > 0 {
		checks = append(checks, Check{
			Name:    "fixture-functions",
			Message: fmt.Sprintf("missing required fixture function(s): %s", strings.Join(missing, ", ")),
		})
		return checks, nil
	}

	checks = append(checks, Check{
		Name:    "fixture-functions",
		Passed:  true,
		Message: fmt.Sprintf("found required fixture function(s): %s", strings.Join(requiredSlugs, ", ")),
	})

	return checks, nil
}

// Run executes the selected serve cases and returns a normalized report.
func (r *Runner) Run(ctx context.Context, plan conformance.RunPlan, runtime conformance.RuntimeConfig) (conformance.Report, error) {
	if runtime.Transport != conformance.TransportServe {
		return conformance.Report{}, fmt.Errorf("serve runner requires transport %q, got %q", conformance.TransportServe, runtime.Transport)
	}

	if runtime.SDKURL == nil || runtime.APIURL == nil || runtime.EventURL == nil {
		return conformance.Report{}, fmt.Errorf("serve runner requires sdk, api, and event URLs")
	}
	if strings.TrimSpace(runtime.SigningKey) == "" {
		return conformance.Report{}, fmt.Errorf("serve runner requires a signing key")
	}

	env, err := r.prepare(ctx, runtime)
	if err != nil {
		caseResults := markTransportSetupFailed(plan.Cases, err)
		return conformance.NewReport(plan, caseResults), nil
	}
	defer env.close(context.Background())

	caseResults := make([]conformance.CaseResult, 0, len(plan.Cases))
	for _, testCase := range plan.Cases {
		result := env.runCase(ctx, testCase)
		caseResults = append(caseResults, result)
	}

	return conformance.NewReport(plan, caseResults), nil
}

type runtimeEnv struct {
	runtime conformance.RuntimeConfig
	client  *http.Client

	proxy *proxyServer

	registerRequest sdk.RegisterRequest
	functionsBySlug map[string]sdk.SDKFunction
}

func (r *Runner) prepare(ctx context.Context, runtime conformance.RuntimeConfig) (*runtimeEnv, error) {
	proxy, err := newProxyServer(r.client, runtime.SDKURL)
	if err != nil {
		return nil, err
	}

	registerRequest, functionsBySlug, err := r.syncSDKFunctions(ctx, runtime, proxy.URL())
	if err != nil {
		_ = proxy.close(context.Background())
		return nil, err
	}

	// Registration rewrites every step URL to point at the local proxy so the
	// runner can inspect executor traffic while still forwarding to the real SDK.
	if err := registerServeFunctions(ctx, r.client, runtime, proxy.URL(), registerRequest); err != nil {
		_ = proxy.close(context.Background())
		return nil, err
	}

	return &runtimeEnv{
		runtime:         runtime,
		client:          r.client,
		proxy:           proxy,
		registerRequest: registerRequest,
		functionsBySlug: functionsBySlug,
	}, nil
}

type inBandSyncResponse struct {
	AppID       string            `json:"app_id"`
	Functions   []sdk.SDKFunction `json:"functions"`
	Platform    *string           `json:"platform"`
	SDKLanguage string            `json:"sdk_language"`
	SDKVersion  string            `json:"sdk_version"`
	URL         string            `json:"url"`
}

func (r *Runner) syncSDKFunctions(ctx context.Context, runtime conformance.RuntimeConfig, proxyURL string) (sdk.RegisterRequest, map[string]sdk.SDKFunction, error) {
	payload, err := json.Marshal(map[string]string{"url": proxyURL})
	if err != nil {
		return sdk.RegisterRequest{}, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, runtime.SDKURL.String(), bytes.NewReader(payload))
	if err != nil {
		return sdk.RegisterRequest{}, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-inngest-sync-kind", "in_band")
	req.Header.Set("X-Inngest-Server-Kind", "dev")
	if strings.TrimSpace(runtime.SigningKey) != "" {
		sig, err := inngestgo.Sign(ctx, time.Now(), []byte(runtime.SigningKey), payload)
		if err != nil {
			return sdk.RegisterRequest{}, nil, fmt.Errorf("sign in-band sync request: %w", err)
		}
		req.Header.Set("X-Inngest-Signature", sig)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return sdk.RegisterRequest{}, nil, fmt.Errorf("call SDK in-band sync: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		dump, _ := httputil.DumpResponse(resp, true)
		return sdk.RegisterRequest{}, nil, fmt.Errorf("SDK in-band sync returned %d: %s", resp.StatusCode, string(dump))
	}

	syncResponse := inBandSyncResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&syncResponse); err != nil {
		return sdk.RegisterRequest{}, nil, fmt.Errorf("decode SDK in-band sync response: %w; ensure the SDK app enables in-band sync", err)
	}

	sdkVersion := ""
	if syncResponse.SDKLanguage != "" {
		sdkVersion = syncResponse.SDKLanguage
		if syncResponse.SDKVersion != "" {
			sdkVersion += ":" + syncResponse.SDKVersion
		}
	}

	requestURL := syncResponse.URL
	if requestURL == "" {
		requestURL = proxyURL
	}

	request := sdk.RegisterRequest{
		V:          "1",
		URL:        requestURL,
		DeployType: sdk.DeployTypePing,
		SDK:        sdkVersion,
		AppName:    syncResponse.AppID,
		Functions:  syncResponse.Functions,
	}
	if syncResponse.Platform != nil {
		request.Headers.Platform = *syncResponse.Platform
	}

	functionsBySlug := make(map[string]sdk.SDKFunction, len(request.Functions))
	for _, fn := range request.Functions {
		functionsBySlug[fn.Slug] = fn
	}

	return request, functionsBySlug, nil
}

func (e *runtimeEnv) close(ctx context.Context) error {
	var errs []string

	if e.proxy != nil {
		if err := e.proxy.close(ctx); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}

	return nil
}

func (e *runtimeEnv) runCase(ctx context.Context, testCase conformance.Case) conformance.CaseResult {
	executor, ok := caseExecutors[testCase.ID]
	if !ok {
		return conformance.CaseResult{
			CaseID:     testCase.ID,
			SuiteID:    testCase.SuiteID,
			Status:     conformance.CaseStatusNotImplemented,
			ReasonCode: conformance.ReasonCodeNotImplemented,
			Reason:     "no Phase 2 serve executor is defined for this case",
		}
	}

	if executor.slug != "" {
		if !hasSyncedFunction(e.functionsBySlug, executor.slug) {
			return conformance.CaseResult{
				CaseID:     testCase.ID,
				SuiteID:    testCase.SuiteID,
				Status:     conformance.CaseStatusNotImplemented,
				ReasonCode: conformance.ReasonCodeNotImplemented,
				Reason:     fmt.Sprintf("required fixture function %q was not found in SDK sync response", executor.slug),
			}
		}
	}

	if testCase.ID == "serve-introspection" {
		return conformance.CaseResult{
			CaseID:  testCase.ID,
			SuiteID: testCase.SuiteID,
			Status:  conformance.CaseStatusPassed,
		}
	}

	tap := e.proxy.newTap()
	e.proxy.setTap(tap)
	defer e.proxy.clearTap(tap)

	harness := &caseHarness{
		client:       e.client,
		runtime:      e.runtime,
		requests:     tap.requests,
		responses:    tap.responses,
		requestSteps: map[string]any{},
	}

	if err := executor.run(ctx, harness); err != nil {
		return conformance.CaseResult{
			CaseID:     testCase.ID,
			SuiteID:    testCase.SuiteID,
			Status:     conformance.CaseStatusFailed,
			ReasonCode: conformance.ReasonCodeBehaviorMismatch,
			Reason:     err.Error(),
		}
	}

	return conformance.CaseResult{
		CaseID:  testCase.ID,
		SuiteID: testCase.SuiteID,
		Status:  conformance.CaseStatusPassed,
	}
}

func markTransportSetupFailed(cases []conformance.Case, cause error) []conformance.CaseResult {
	results := make([]conformance.CaseResult, 0, len(cases))
	for _, testCase := range cases {
		results = append(results, conformance.CaseResult{
			CaseID:     testCase.ID,
			SuiteID:    testCase.SuiteID,
			Status:     conformance.CaseStatusNotEvaluable,
			ReasonCode: conformance.ReasonCodeTransportSetup,
			Reason:     cause.Error(),
		})
	}
	return results
}

type caseExecutor struct {
	slug string
	run  func(context.Context, *caseHarness) error
}

var caseExecutors = map[string]caseExecutor{
	"serve-introspection":  {},
	"basic-invoke":         {slug: "test-suite-simple-fn", run: runBasicInvoke},
	"steps-serial":         {slug: "test-suite-step-test", run: runStepsSerial},
	"retry-basic":          {slug: "test-suite-retry-test", run: runRetryBasic},
	"cancel-basic":         {slug: "test-suite-cancel-test", run: runCancelBasic},
	"wait-for-event-basic": {slug: "test-suite-wait-for-event", run: runWaitForEventBasic},
}

func requiredFixtureSlugs(cases []conformance.Case) []string {
	seen := map[string]struct{}{}
	slugs := make([]string, 0, len(cases))
	for _, testCase := range cases {
		executor, ok := caseExecutors[testCase.ID]
		if !ok || executor.slug == "" {
			continue
		}
		if _, ok := seen[executor.slug]; ok {
			continue
		}
		seen[executor.slug] = struct{}{}
		slugs = append(slugs, executor.slug)
	}
	return slugs
}

func hasSyncedFunction(functionsBySlug map[string]sdk.SDKFunction, expectedSlug string) bool {
	if _, ok := functionsBySlug[expectedSlug]; ok {
		return true
	}
	for actualSlug := range functionsBySlug {
		if strings.HasSuffix(actualSlug, "-"+expectedSlug) {
			return true
		}
	}
	return false
}

func registerServeFunctions(ctx context.Context, client *http.Client, runtime conformance.RuntimeConfig, proxyURL string, registerRequest sdk.RegisterRequest) error {
	rewritten := registerRequest

	// Every registered step URL is rewritten to the local proxy. The proxy then
	// forwards the request to the actual SDK and captures the request/response
	// pair for the conformance assertions.
	for fnIdx, fn := range rewritten.Functions {
		for stepID, step := range fn.Steps {
			nodeURL, _ := step.Runtime["url"].(string)
			step.Runtime["url"] = replaceStepURL(nodeURL, proxyURL)
			fn.Steps[stepID] = step
		}
		rewritten.Functions[fnIdx] = fn
	}
	rewritten.URL = proxyURL

	payload, err := json.Marshal(rewritten)
	if err != nil {
		return err
	}

	registerURL := cloneRuntimeURL(runtime.APIURL)
	registerURL.Path = "/fn/register"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, registerURL.String(), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", signingAuthorization(runtime.SigningKey))

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("register functions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		dump, _ := httputil.DumpResponse(resp, true)
		return fmt.Errorf("register functions returned %d: %s", resp.StatusCode, string(dump))
	}

	return nil
}

func replaceStepURL(nodeURL, proxyURL string) string {
	node, err := url.Parse(nodeURL)
	if err != nil {
		return proxyURL
	}
	proxy, err := url.Parse(proxyURL)
	if err != nil {
		return proxyURL
	}
	proxy.Path = "/"
	proxy.RawQuery = node.RawQuery
	return proxy.String()
}

func signingAuthorization(signingKey string) string {
	key := regexp.MustCompile(`^signkey-[\w]+-`).ReplaceAllString(signingKey, "")
	decoded, err := hex.DecodeString(key)
	if err != nil {
		// Fall back to the raw key when the input is not hex-encoded. The
		// conformance runner should still be usable with local/dev-only keys.
		decoded = []byte(signingKey)
	}

	sum := sha256.Sum256(decoded)
	return fmt.Sprintf("Bearer signkey-test-%s", hex.EncodeToString(sum[:]))
}

type proxyServer struct {
	server *http.Server
	url    string

	client *http.Client
	sdkURL *url.URL

	mu  sync.RWMutex
	tap *proxyTap
}

type proxyTap struct {
	requests  chan requestSnapshot
	responses chan responseSnapshot
}

type requestSnapshot struct {
	Request *http.Request
	Body    []byte
}

type responseSnapshot struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func newProxyServer(client *http.Client, sdkURL *url.URL) (*proxyServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	proxy := &proxyServer{
		client: client,
		sdkURL: sdkURL,
		url:    fmt.Sprintf("http://%s", listener.Addr().String()),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", proxy.handle)

	proxy.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		_ = proxy.server.Serve(listener)
	}()

	return proxy, nil
}

func (p *proxyServer) URL() string {
	return p.url
}

func (p *proxyServer) newTap() *proxyTap {
	return &proxyTap{
		requests:  make(chan requestSnapshot, 8),
		responses: make(chan responseSnapshot, 8),
	}
}

func (p *proxyServer) setTap(tap *proxyTap) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tap = tap
}

func (p *proxyServer) clearTap(tap *proxyTap) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.tap == tap {
		p.tap = nil
	}
}

func (p *proxyServer) currentTap() *proxyTap {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.tap
}

func (p *proxyServer) close(ctx context.Context) error {
	return p.server.Shutdown(ctx)
}

func (p *proxyServer) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPut {
		_ = r.Body.Close()
		w.WriteHeader(http.StatusOK)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = r.Body.Close()

	if tap := p.currentTap(); tap != nil {
		reqCopy := r.Clone(r.Context())
		reqCopy.Body = io.NopCloser(bytes.NewReader(body))
		tap.requests <- requestSnapshot{Request: reqCopy, Body: body}
	}

	target := cloneRuntimeURL(p.sdkURL)
	target.RawQuery = r.URL.RawQuery

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, target.String(), bytes.NewReader(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header = r.Header.Clone()

	resp, err := p.client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if tap := p.currentTap(); tap != nil {
		tap.responses <- responseSnapshot{
			StatusCode: resp.StatusCode,
			Header:     resp.Header.Clone(),
			Body:       respBody,
		}
	}

	for header, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(header, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)
}

// caseHarness mirrors the parts of the legacy SDK integration DSL that are
// useful for portable conformance execution, but uses explicit errors rather
// than testing helpers so it can run in the CLI and in normal Go tests.
type caseHarness struct {
	client  *http.Client
	runtime conformance.RuntimeConfig

	requests  <-chan requestSnapshot
	responses <-chan responseSnapshot

	requestEvent inngestgo.Event
	requestCtx   driver.SDKRequestContext
	requestSteps map[string]any
}

func (h *caseHarness) setRequestEvent(event inngestgo.Event) {
	h.requestEvent = event
	h.requestCtx = driver.SDKRequestContext{
		StepID: "step",
		Stack: &driver.FunctionStack{
			Current: 0,
			Stack:   []string{},
		},
	}
	if h.requestSteps == nil {
		h.requestSteps = map[string]any{}
	}
}

func (h *caseHarness) setRequestContext(ctx driver.SDKRequestContext) {
	h.requestCtx = ctx
	if h.requestCtx.Stack == nil {
		h.requestCtx.Stack = &driver.FunctionStack{Current: 0, Stack: []string{}}
	}
	if h.requestCtx.Stack.Stack == nil {
		h.requestCtx.Stack.Stack = []string{}
	}
}

func (h *caseHarness) addRequestStack(stack driver.FunctionStack) {
	if h.requestCtx.Stack == nil {
		h.requestCtx.Stack = &driver.FunctionStack{Current: 0, Stack: []string{}}
	}
	h.requestCtx.Stack.Stack = append(h.requestCtx.Stack.Stack, stack.Stack...)
	h.requestCtx.Stack.Current = stack.Current
}

func (h *caseHarness) addRequestSteps(steps map[string]any) {
	if h.requestSteps == nil {
		h.requestSteps = map[string]any{}
	}
	for key, value := range steps {
		h.requestSteps[key] = value
	}
}

func (h *caseHarness) sendEvent(ctx context.Context, event inngestgo.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	target := cloneRuntimeURL(h.runtime.EventURL)
	target.Path = strings.TrimRight(target.Path, "/") + "/e/" + h.runtime.EventKey

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("send event %q: %w", event.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send event %q returned %d: %s", event.Name, resp.StatusCode, string(body))
	}

	return nil
}

func (h *caseHarness) expectNoRequests(duration time.Duration) error {
	select {
	case req := <-h.requests:
		return fmt.Errorf("expected no executor request, but received %s %s", req.Request.Method, req.Request.URL.String())
	case <-time.After(duration):
		return nil
	}
}

func (h *caseHarness) expectRequest(timeout time.Duration, queryStepID string, modifiers ...func(*driver.SDKRequestContext)) error {
	select {
	case req := <-h.requests:
		if got := req.Request.URL.Query().Get("stepId"); got != queryStepID {
			return fmt.Errorf("expected stepId %q, got %q", queryStepID, got)
		}

		execReq := executorRequest{}
		if err := json.Unmarshal(req.Body, &execReq); err != nil {
			return fmt.Errorf("decode executor request: %w", err)
		}

		if execReq.Ctx.Stack == nil || execReq.Ctx.Stack.Stack == nil {
			return fmt.Errorf("executor request stack was nil")
		}

		expectedEvent := h.requestEvent
		actualEvent := execReq.Event
		actualTS := actualEvent.Timestamp
		actualID := actualEvent.ID
		actualEvent.Timestamp = 0
		actualEvent.ID = nil
		expectedEvent.Timestamp = 0
		expectedEvent.ID = nil
		if expectedEvent.User == nil {
			expectedEvent.User = map[string]any{}
		}
		if actualEvent.User == nil {
			actualEvent.User = map[string]any{}
		}
		if expectedEvent.Data == nil {
			expectedEvent.Data = map[string]any{}
		}
		if actualEvent.Data == nil {
			actualEvent.Data = map[string]any{}
		}
		if !reflect.DeepEqual(expectedEvent, actualEvent) {
			return fmt.Errorf("unexpected executor event:\nexpected: %s\nactual:   %s", mustJSON(expectedEvent), mustJSON(actualEvent))
		}
		actualEvent.Timestamp = actualTS
		actualEvent.ID = actualID

		expectedCtx := h.requestCtx
		for _, modifier := range modifiers {
			modifier(&expectedCtx)
		}

		expectedCtx.RunID = execReq.Ctx.RunID
		expectedCtx.FunctionID = execReq.Ctx.FunctionID
		expectedCtx.QueueItemRef = execReq.Ctx.QueueItemRef
		actualCtx := execReq.Ctx
		actualCtx.MaxAttempts = 0
		expectedCtx.MaxAttempts = 0

		if !reflect.DeepEqual(expectedCtx, actualCtx) {
			return fmt.Errorf("unexpected executor context:\nexpected: %s\nactual:   %s", mustJSON(expectedCtx), mustJSON(actualCtx))
		}

		for _, value := range execReq.Steps {
			data, ok := value.(map[string]any)
			if !ok {
				continue
			}
			if errValue, ok := data["error"].(map[string]any); ok {
				delete(errValue, "stack")
			}
		}

		if !reflect.DeepEqual(h.requestSteps, execReq.Steps) {
			return fmt.Errorf("unexpected executor steps:\nexpected: %s\nactual:   %s", mustJSON(h.requestSteps), mustJSON(execReq.Steps))
		}

		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timed out after %s waiting for executor request", timeout)
	}
}

func (h *caseHarness) expectJSONResponse(status int, expected any, timeout time.Duration) error {
	return h.expectResponse(status, timeout, func(body []byte) error {
		var actual any
		if err := json.Unmarshal(body, &actual); err != nil {
			return err
		}
		if !reflect.DeepEqual(expected, actual) {
			return fmt.Errorf("unexpected JSON response:\nexpected: %s\nactual:   %s", mustJSON(expected), mustJSON(actual))
		}
		return nil
	})
}

func (h *caseHarness) expectGeneratorResponse(expected []state.GeneratorOpcode, timeout time.Duration) error {
	return h.expectResponse(http.StatusPartialContent, timeout, func(body []byte) error {
		actual := []state.GeneratorOpcode{}
		if err := json.Unmarshal(body, &actual); err != nil {
			return err
		}

		actual = normalizeGeneratorOpcodes(actual)
		expected = normalizeGeneratorOpcodes(expected)

		if !reflect.DeepEqual(expected, actual) {
			return fmt.Errorf("unexpected generator response:\nexpected: %s\nactual:   %s", mustJSON(expected), mustJSON(actual))
		}
		return nil
	})
}

func (h *caseHarness) expectResponse(status int, timeout time.Duration, validate func([]byte) error) error {
	select {
	case resp := <-h.responses:
		if resp.StatusCode != status {
			return fmt.Errorf("expected status %d, got %d", status, resp.StatusCode)
		}
		return validate(resp.Body)
	case <-time.After(timeout):
		return fmt.Errorf("timed out after %s waiting for SDK response", timeout)
	}
}

func normalizeGeneratorOpcodes(in []state.GeneratorOpcode) []state.GeneratorOpcode {
	out := make([]state.GeneratorOpcode, len(in))
	copy(out, in)

	var zero state.GeneratorOpcode
	for idx := range out {
		out[idx].Timing = zero.Timing
		out[idx].Userland = nil
		out[idx].Metadata = nil
		out[idx].DisplayName = nil

		if out[idx].Op == enums.OpcodeSleep {
			if opts, ok := out[idx].Opts.(map[string]any); ok {
				if duration, ok := opts["duration"].(string); ok && duration != "" {
					out[idx].Name = duration
					out[idx].Opts = nil
				}
			}
		}

		if out[idx].Error != nil {
			out[idx].Error.Name = ""
			out[idx].Error.Stack = "[proxy-redact]"
			out[idx].Error.Data = nil
			out[idx].Error.Cause = nil
		}
	}

	return out
}

type executorRequest struct {
	Event   inngestgo.Event          `json:"event"`
	Steps   map[string]any           `json:"steps"`
	Ctx     driver.SDKRequestContext `json:"ctx"`
	Version int                      `json:"version"`
}

func mustJSON(v any) string {
	byt, _ := json.Marshal(v)
	return string(byt)
}

func cloneRuntimeURL(in *url.URL) *url.URL {
	if in == nil {
		return nil
	}

	out := *in
	return &out
}

func runBasicInvoke(ctx context.Context, h *caseHarness) error {
	event := inngestgo.Event{
		Name: "tests/function.test",
		Data: map[string]any{
			"test": true,
		},
		User: map[string]any{},
	}

	h.setRequestEvent(event)
	if err := h.sendEvent(ctx, event); err != nil {
		return err
	}
	if err := h.expectRequest(5*time.Second, "step"); err != nil {
		return err
	}
	return h.expectJSONResponse(http.StatusOK, map[string]any{
		"name": "tests/function.test",
		"body": "ok",
	}, 5*time.Second)
}

func runStepsSerial(ctx context.Context, h *caseHarness) error {
	event := inngestgo.Event{
		Name: "tests/step.test",
		Data: map[string]any{
			"steps": map[string]any{
				"ok": "yes",
			},
		},
		User: map[string]any{
			"email": "test@example.com",
		},
	}

	hashes := map[string]string{
		"first step":  "98bf98df193bcce7c33e6bc50927cf2ac21206cb",
		"sleep":       "c3ca5f787365eae0dea86250e27d476406956478",
		"second step": "764e20ec975d4ef820d0f42e6a5833384bd7ee36",
	}

	h.setRequestEvent(event)
	if err := h.sendEvent(ctx, event); err != nil {
		return err
	}
	if err := h.expectRequest(5*time.Second, "step"); err != nil {
		return err
	}
	if err := h.expectGeneratorResponse([]state.GeneratorOpcode{{
		Op:          enums.OpcodeStepRun,
		ID:          hashes["first step"],
		Name:        "first step",
		DisplayName: inngestgo.StrPtr("first step"),
		Data:        []byte(`"first step"`),
	}}, 5*time.Second); err != nil {
		return err
	}

	h.addRequestStack(driver.FunctionStack{Stack: []string{hashes["first step"]}, Current: 1})
	h.addRequestSteps(map[string]any{
		hashes["first step"]: map[string]any{"data": "first step"},
	})

	if err := h.expectRequest(5*time.Second, "step"); err != nil {
		return err
	}
	if err := h.expectGeneratorResponse([]state.GeneratorOpcode{{
		Op:          enums.OpcodeSleep,
		ID:          hashes["sleep"],
		Name:        "2s",
		DisplayName: inngestgo.StrPtr("for 2s"),
		Data:        json.RawMessage("null"),
	}}, 5*time.Second); err != nil {
		return err
	}

	h.addRequestStack(driver.FunctionStack{Stack: []string{hashes["sleep"]}, Current: 2})
	h.addRequestSteps(map[string]any{
		hashes["sleep"]: nil,
	})

	if err := h.expectRequest(6*time.Second, "step"); err != nil {
		return err
	}
	if err := h.expectGeneratorResponse([]state.GeneratorOpcode{{
		Op:          enums.OpcodeStepRun,
		ID:          hashes["second step"],
		Name:        "second step",
		DisplayName: inngestgo.StrPtr("second step"),
		Data:        json.RawMessage(`{"first":"first step","second":true}`),
	}}, 5*time.Second); err != nil {
		return err
	}

	h.addRequestStack(driver.FunctionStack{Stack: []string{hashes["second step"]}, Current: 3})
	h.addRequestSteps(map[string]any{
		hashes["second step"]: map[string]any{
			"data": map[string]any{
				"first":  "first step",
				"second": true,
			},
		},
	})

	if err := h.expectRequest(5*time.Second, "step"); err != nil {
		return err
	}
	return h.expectJSONResponse(http.StatusOK, map[string]any{
		"body": "ok",
		"name": "tests/step.test",
	}, 5*time.Second)
}

func runRetryBasic(ctx context.Context, h *caseHarness) error {
	event := inngestgo.Event{
		Name: "tests/retry.test",
		Data: map[string]any{
			"steps": map[string]any{
				"ok": "yes",
			},
		},
		User: map[string]any{
			"email": "test@example.com",
		},
	}

	hash := "98bf98df193bcce7c33e6bc50927cf2ac21206cb"

	h.setRequestEvent(event)
	if err := h.sendEvent(ctx, event); err != nil {
		return err
	}
	if err := h.expectRequest(5*time.Second, "step"); err != nil {
		return err
	}
	if err := h.expectGeneratorResponse([]state.GeneratorOpcode{{
		Op:          enums.OpcodeStepError,
		ID:          hash,
		Name:        "first step",
		DisplayName: inngestgo.StrPtr("first step"),
		Error: &state.UserError{
			Name:    "Error",
			Message: "broken",
		},
		Data: []byte(`null`),
	}}, 5*time.Second); err != nil {
		return err
	}

	if err := h.expectRequest(45*time.Second, "step", func(ctx *driver.SDKRequestContext) {
		ctx.Attempt = 1
	}); err != nil {
		return err
	}
	if err := h.expectGeneratorResponse([]state.GeneratorOpcode{{
		Op:          enums.OpcodeStepRun,
		ID:          hash,
		Name:        "first step",
		DisplayName: inngestgo.StrPtr("first step"),
		Data:        []byte(`"yes: 2"`),
	}}, 5*time.Second); err != nil {
		return err
	}

	h.addRequestStack(driver.FunctionStack{Stack: []string{hash}, Current: 1})
	h.addRequestSteps(map[string]any{
		hash: map[string]any{"data": "yes: 2"},
	})

	if err := h.expectRequest(5*time.Second, "step", func(ctx *driver.SDKRequestContext) {
		ctx.Attempt = 0
	}); err != nil {
		return err
	}
	if err := h.expectResponse(http.StatusInternalServerError, 5*time.Second, func(body []byte) error {
		actual := map[string]any{}
		if err := json.Unmarshal(body, &actual); err != nil {
			return err
		}
		if name, ok := actual["name"]; ok && name != "" && name != "Error" {
			return fmt.Errorf("unexpected function error response: %s", string(body))
		}
		if actual["message"] != "broken func" {
			return fmt.Errorf("unexpected function error response: %s", string(body))
		}
		return nil
	}); err != nil {
		return err
	}

	if err := h.expectRequest(45*time.Second, "step", func(ctx *driver.SDKRequestContext) {
		ctx.Attempt = 1
	}); err != nil {
		return err
	}
	return h.expectJSONResponse(http.StatusOK, map[string]any{
		"body": "ok",
		"name": "tests/retry.test",
	}, 5*time.Second)
}

func runCancelBasic(ctx context.Context, h *caseHarness) error {
	event := inngestgo.Event{
		Name: "tests/cancel.test",
		Data: map[string]any{
			"request_id": "123",
			"whatever":   "this doesn't matter my friend",
		},
		User: map[string]any{},
	}

	h.setRequestEvent(event)
	h.setRequestContext(driver.SDKRequestContext{
		StepID: "step",
		Stack: &driver.FunctionStack{
			Current: 0,
			Stack:   []string{},
		},
	})

	if err := h.sendEvent(ctx, event); err != nil {
		return err
	}
	if err := h.expectRequest(5*time.Second, "step"); err != nil {
		return err
	}
	if err := h.expectGeneratorResponse([]state.GeneratorOpcode{{
		Op:          enums.OpcodeSleep,
		ID:          "c3ca5f787365eae0dea86250e27d476406956478",
		Name:        "10s",
		DisplayName: inngestgo.StrPtr("sleep"),
		Data:        json.RawMessage("null"),
	}}, 5*time.Second); err != nil {
		return err
	}

	time.Sleep(1 * time.Second)
	if err := h.sendEvent(ctx, inngestgo.Event{
		Name: "cancel/please",
		Data: map[string]any{
			"request_id": "123",
		},
	}); err != nil {
		return err
	}

	return h.expectNoRequests(11 * time.Second)
}

func runWaitForEventBasic(ctx context.Context, h *caseHarness) error {
	event := inngestgo.Event{
		Name: "tests/wait.test",
		Data: map[string]any{
			"id": "123",
		},
		User: map[string]any{},
	}

	waitHash := "daaad336276d15594d0e765f96c17cd746bf4971"
	resumeID := "resume"
	resume := inngestgo.Event{
		ID:   &resumeID,
		Name: "test/resume",
		Data: map[string]any{
			"id":     "123",
			"resume": true,
		},
		Timestamp: time.Now().UnixMilli(),
		User:      map[string]any{},
	}

	h.setRequestEvent(event)
	h.setRequestContext(driver.SDKRequestContext{
		StepID: "step",
		Stack: &driver.FunctionStack{
			Current: 0,
			Stack:   []string{},
		},
	})

	if err := h.sendEvent(ctx, event); err != nil {
		return err
	}
	if err := h.expectRequest(5*time.Second, "step"); err != nil {
		return err
	}
	if err := h.expectGeneratorResponse([]state.GeneratorOpcode{{
		Op:          enums.OpcodeWaitForEvent,
		ID:          waitHash,
		Name:        "wait",
		DisplayName: inngestgo.StrPtr("test/resume"),
		Data:        json.RawMessage("null"),
		Opts: map[string]any{
			"event":   "test/resume",
			"if":      "async.data.resume == true && async.data.id == event.data.id",
			"timeout": "10s",
		},
	}}, 5*time.Second); err != nil {
		return err
	}

	time.Sleep(1 * time.Second)
	if err := h.sendEvent(ctx, inngestgo.Event{
		Name: "test/resume",
		Data: map[string]any{
			"id": "ignored",
		},
	}); err != nil {
		return err
	}

	time.Sleep(1 * time.Second)
	if err := h.sendEvent(ctx, resume); err != nil {
		return err
	}

	h.addRequestStack(driver.FunctionStack{Stack: []string{waitHash}, Current: 1})
	h.addRequestSteps(map[string]any{
		waitHash: resume.Map(),
	})

	if err := h.expectRequest(5*time.Second, "step"); err != nil {
		return err
	}
	return h.expectJSONResponse(http.StatusOK, map[string]any{
		"result": map[string]any{
			"id":     "123",
			"resume": true,
		},
	}, 5*time.Second)
}

// eventTrigger is a tiny helper for building fixture introspection payloads in
// tests and future serve fixture manifests.
func eventTrigger(name string) inngest.Trigger {
	return inngest.Trigger{EventTrigger: &inngest.EventTrigger{Event: name}}
}
