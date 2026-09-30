// Package cloudsandboxes connects the local dev server to Cloud compute without
// moving function execution, events, or run state out of the dev server.
package cloudsandboxes

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/flock"
	cliauth "github.com/inngest/inngest/cmd/internal/auth"
	"github.com/inngest/inngest/pkg/api/v2/apiv2endpoint"
)

const loginRequiredMessage = "Run `inngest login` to use Cloud sandboxes from the dev server"

type status struct {
	AccountName     string   `json:"accountName,omitempty"`
	EnvironmentName string   `json:"environmentName,omitempty"`
	EnvironmentID   string   `json:"environmentId,omitempty"`
	SandboxIDs      []string `json:"sandboxIds"`
	Warning         string   `json:"warning,omitempty"`
}

type Bridge struct {
	ctx       context.Context
	auth      *cliauth.Manager
	resource  *url.URL
	port      string
	statePath string
	router    http.Handler
	transport http.RoundTripper

	mu      sync.Mutex
	warning string
	// The journal contains resource IDs only, partitioned by Cloud identity.
	// It survives dev-server restarts independently of local run persistence.
	sandboxes map[string][]string
}

func New(ctx context.Context, port int) (*Bridge, error) {
	manager, err := cliauth.NewManager()
	if err != nil {
		return nil, err
	}
	issuer, err := cliauth.Issuer()
	if err != nil {
		return nil, err
	}
	resource, err := url.Parse(cliauth.Resource(issuer))
	if err != nil {
		return nil, err
	}
	project, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	dir := os.Getenv("INNGEST_CONFIG_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(home, ".config", "inngest")
	}
	b := &Bridge{
		ctx: ctx, auth: manager,
		resource: resource, port: fmt.Sprint(port),
		statePath: filepath.Join(dir, "dev-sandboxes", fmt.Sprintf("%x.json", sha256.Sum256([]byte(project)))),
		transport: http.DefaultTransport.(*http.Transport).Clone(),
	}
	b.sandboxes, err = b.loadJournal()
	if err != nil {
		b.sandboxes = map[string][]string{}
		b.warning = "Could not load saved sandbox IDs. Existing sandboxes remain accessible through the SDK or Cloud dashboard."
	}
	b.routes()
	return b, nil
}

func (b *Bridge) routes() {
	r := chi.NewRouter()
	r.Get("/dev/cloud/status", b.getStatus)
	// Derive the allowlist from the public API contract, including snapshots
	// and streaming routes. No other Cloud API is exposed through this bridge.
	for _, endpoint := range apiv2endpoint.Discover() {
		if strings.HasPrefix(endpoint.AuthzPermission, "sandboxes:") {
			for _, prefix := range []string{"/v2", "/api/v2"} {
				r.MethodFunc(endpoint.HTTPMethod, prefix+endpoint.Path, b.proxy)
			}
		}
	}
	b.router = r
}

func (b *Bridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	// Existing dev-server CORS is deliberately permissive. This privileged
	// surface only accepts loopback connections and same-origin browser
	// requests. Remote development uses a local port forward.
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || !net.ParseIP(remoteHost).IsLoopback() {
		writeError(w, http.StatusForbidden, "invalid_local_address", "Cloud sandboxes require a local connection to the dev server")
		return
	}
	host, port, err := net.SplitHostPort(r.Host)
	ip := net.ParseIP(host)
	if err != nil || port != b.port || (host != "localhost" && (ip == nil || !ip.IsLoopback())) {
		writeError(w, http.StatusForbidden, "invalid_local_host", "Cloud sandboxes require a loopback dev-server URL")
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+r.Host {
		writeError(w, http.StatusForbidden, "invalid_origin", "Open sandboxes from the dev server's own UI")
		return
	}
	// Reject escaped separators/dot paths rather than allowing the upstream
	// router to normalize an allowlisted URL into another API.
	if r.URL.RawPath != "" || strings.Contains(r.URL.Path, "..") {
		writeError(w, http.StatusBadRequest, "invalid_path", "Invalid sandbox API path")
		return
	}
	// Mounts share this handler; route against the original full request path.
	b.router.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chi.NewRouteContext())))
}

func (b *Bridge) getStatus(w http.ResponseWriter, r *http.Request) {
	_, metadata := b.accessToken(w, r)
	if metadata == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	writeData(w, status{
		AccountName: metadata.AccountName, EnvironmentName: metadata.WorkspaceName,
		EnvironmentID: *metadata.WorkspaceID, Warning: b.warning,
		SandboxIDs: append([]string{}, b.sandboxes[b.identity(metadata)]...),
	})
}

func (b *Bridge) accessToken(w http.ResponseWriter, r *http.Request) (string, *cliauth.Metadata) {
	token, metadata, err := b.auth.AccessToken(r.Context(), b.resource.String())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "cloud_login_required", loginRequiredMessage)
		return "", nil
	}
	if metadata.WorkspaceID == nil || *metadata.WorkspaceID == "" {
		writeError(w, http.StatusBadRequest, "environment_required", "Run `inngest login --force` and select a single development environment to use Cloud sandboxes")
		return "", nil
	}
	return token, metadata
}

func (b *Bridge) identity(metadata *cliauth.Metadata) string {
	return metadata.Resource + "|" + metadata.AccountID + "|" + *metadata.WorkspaceID
}

func (b *Bridge) proxy(w http.ResponseWriter, r *http.Request) {
	token, metadata := b.accessToken(w, r)
	if metadata == nil {
		return
	}
	path := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/api"), "/v2")
	proxy := &httputil.ReverseProxy{
		Transport:     b.transport,
		FlushInterval: -1,
		Rewrite: func(p *httputil.ProxyRequest) {
			p.SetURL(b.resource)
			p.Out.URL.Path = b.resource.Path + path
			p.Out.URL.RawPath = ""
			// Do not forward caller credentials, cookies, environment overrides,
			// tracing baggage, or arbitrary user headers to Cloud.
			p.Out.Header = make(http.Header)
			for _, name := range []string{"Accept", "Content-Type", "Content-Encoding", "X-Sandbox-File-Mode"} {
				if values := r.Header.Values(name); len(values) > 0 {
					p.Out.Header[name] = values
				}
			}
			p.Out.Header.Set("Authorization", "Bearer "+token)
		},
		ModifyResponse: func(resp *http.Response) error {
			if resp.StatusCode == http.StatusUnauthorized {
				return cliauth.ErrNotLoggedIn
			}
			if resp.StatusCode >= 300 && resp.StatusCode < 400 {
				return errors.New("sandbox API redirects are not supported")
			}
			// Raw file and NDJSON responses are untouched. Only small create
			// responses need inspection to remember this project's resources.
			if r.Method == http.MethodPost && path == "/sandboxes" && resp.StatusCode >= 200 && resp.StatusCode < 300 {
				data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
				if err != nil {
					return err
				}
				resp.Body = struct {
					io.Reader
					io.Closer
				}{io.MultiReader(bytes.NewReader(data), resp.Body), resp.Body}
				var result struct {
					Data struct {
						ID string `json:"id"`
					} `json:"data"`
				}
				if json.Unmarshal(data, &result) == nil && result.Data.ID != "" {
					b.remember(metadata, result.Data.ID, false)
				}
			}
			// Cloud answers 202 while termination is in progress and 204 once it is done;
			// either way the delete has been recorded.
			if r.Method == http.MethodDelete && (resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusAccepted) && strings.Count(path, "/") == 2 && strings.HasPrefix(path, "/sandboxes/") {
				b.remember(metadata, strings.TrimPrefix(path, "/sandboxes/"), true)
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			if errors.Is(err, cliauth.ErrNotLoggedIn) {
				writeError(w, http.StatusUnauthorized, "cloud_login_required", loginRequiredMessage)
				return
			}
			writeError(w, http.StatusBadGateway, "cloud_transport_error", "Cloud connection failed. The operation may have executed; inspect its state before repeating a command")
		},
	}
	proxy.ServeHTTP(w, r)
}

func (b *Bridge) remember(metadata *cliauth.Metadata, id string, remove bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	key := b.identity(metadata)
	update := func(journal map[string][]string) {
		if remove {
			journal[key] = slices.DeleteFunc(journal[key], func(value string) bool { return value == id })
		} else if !slices.Contains(journal[key], id) {
			journal[key] = append(journal[key], id)
		}
	}
	update(b.sandboxes)
	if err := b.saveJournal(update); err != nil {
		// Do not turn a successful mutation into a retryable failure.
		b.warning = "Could not save sandbox IDs for recovery. Keep their IDs before stopping this dev server."
	}
}

func (b *Bridge) loadJournal() (map[string][]string, error) {
	journal := map[string][]string{}
	data, err := os.ReadFile(b.statePath)
	if errors.Is(err, os.ErrNotExist) {
		return journal, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &journal); err != nil || journal == nil {
		return nil, fmt.Errorf("invalid sandbox resource journal: %s", b.statePath)
	}
	return journal, nil
}

func (b *Bridge) saveJournal(update func(map[string][]string)) error {
	if err := os.MkdirAll(filepath.Dir(b.statePath), 0o700); err != nil {
		return err
	}
	// Servers in the same project share this journal. Apply only this mutation
	// to the latest file so a stale server cannot lose IDs or resurrect deletions.
	ctx, cancel := context.WithTimeout(b.ctx, 5*time.Second)
	defer cancel()
	lock := flock.New(b.statePath + ".lock")
	locked, err := lock.TryLockContext(ctx, 50*time.Millisecond)
	if err != nil {
		return err
	}
	if !locked {
		return errors.New("could not lock sandbox resource journal")
	}
	defer func() { _ = lock.Unlock() }()
	journal, err := b.loadJournal()
	if err != nil {
		return err
	}
	update(journal)
	data, err := json.Marshal(journal)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(b.statePath), ".sandboxes-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), b.statePath)
}

func writeData(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
}

func writeError(w http.ResponseWriter, code int, name, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"errors": []map[string]string{{"code": name, "message": message}}})
}
