// Package fakesandboxes is a development and test stand-in for the sandbox API.
// It exists so the signalId flow can be exercised end to end on a local Dev
// Server, which has no sandbox runtime.
//
// It is not a sandbox. It never executes commands: a process "runs" a tiny
// script made of sleep, echo, err, exit, kill, flood and probe statements, and
// its supervisor then reports the exit the way the real in-sandbox supervisor
// would, by posting the signal with the scoped token. The Dev Server only
// mounts it when INNGEST_DEV_FAKE_SANDBOXES is set.
package fakesandboxes

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
)

const (
	// tailBytes is how much of each output stream the exit signal carries.
	tailBytes = 8 * 1024
	// maxLifetime stands in for the sandbox's maximum lifetime, which bounds
	// how long the signal token is valid.
	maxLifetime = 24 * time.Hour
	// retryWindow is how long the supervisor keeps retrying a failed send.
	retryWindow = time.Hour
	// maxRetryDelay caps the supervisor's backoff.
	maxRetryDelay = 2 * time.Second
	// maxSleepSeconds and maxFloodKiB bound what a script statement may do.
	maxSleepSeconds = 3600
	maxFloodKiB     = 1024
)

// Server serves the sandbox routes the SDK uses.
type Server struct {
	apiBase string
	secret  []byte
	envID   uuid.UUID
	mux     *http.ServeMux

	mu        sync.Mutex
	sandboxes map[uuid.UUID]*sandbox
	requests  map[string]int
	probes    []string
}

type sandbox struct {
	id        uuid.UUID
	name      string
	createdAt time.Time
	processes map[uuid.UUID]*process
	nextPID   int
}

type process struct {
	id         uuid.UUID
	sandboxID  uuid.UUID
	command    []string
	pid        int
	state      string
	exitCode   *int
	termSignal *int
	startedAt  time.Time
	endedAt    time.Time
	stdout     []byte
	stderr     []byte
	done       chan struct{}
}

// New returns a server whose supervisors post exit signals to apiBase, for
// example "http://127.0.0.1:8288", authenticated with tokens signed by secret
// for envID.
func New(apiBase string, secret []byte, envID uuid.UUID) *Server {
	s := &Server{
		apiBase:   strings.TrimRight(apiBase, "/"),
		secret:    secret,
		envID:     envID,
		sandboxes: map[uuid.UUID]*sandbox{},
		requests:  map[string]int{},
	}
	mux := http.NewServeMux()
	for _, prefix := range []string{"/v2", "/api/v2"} {
		mux.HandleFunc("POST "+prefix+"/sandboxes", s.create)
		mux.HandleFunc("GET "+prefix+"/sandboxes/{id}", s.get)
		mux.HandleFunc("DELETE "+prefix+"/sandboxes/{id}", s.destroy)
		mux.HandleFunc("POST "+prefix+"/sandboxes/{id}/processes", s.startProcess)
		mux.HandleFunc("GET "+prefix+"/sandboxes/{id}/processes/{pid}", s.getProcess)
		mux.HandleFunc("POST "+prefix+"/sandboxes/{id}/processes/{pid}/wait", s.waitProcess)
		mux.HandleFunc("GET "+prefix+"/sandboxes/{id}/processes/{pid}/output", s.processOutput)
		mux.HandleFunc("GET "+prefix+"/sandboxes/_fake/requests", s.stats)
		mux.HandleFunc("DELETE "+prefix+"/sandboxes/_fake/requests", s.resetStats)
	}
	s.mux = mux
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || !net.ParseIP(host).IsLoopback() {
		writeError(w, http.StatusForbidden, "invalid_local_address", "The fake sandbox API only accepts local connections")
		return
	}
	s.mux.ServeHTTP(w, r)
	if !strings.Contains(r.URL.Path, "/_fake/") {
		s.mu.Lock()
		s.requests[r.Method+" "+strings.TrimPrefix(r.Pattern, r.Method+" ")]++
		s.mu.Unlock()
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data":     data,
		"metadata": map[string]any{"fetchedAt": time.Now().UTC().Format(time.RFC3339Nano)},
	})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"errors": []map[string]any{{"code": code, "message": message}},
	})
}

func timestamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func (s *Server) sandboxResource(sb *sandbox) map[string]any {
	return map[string]any{
		"id":        sb.id.String(),
		"name":      sb.name,
		"status":    "RUNNING",
		"vpcId":     uuid.NewSHA1(uuid.NameSpaceOID, []byte("fake-vpc")).String(),
		"imageRef":  "fake",
		"resources": map[string]any{"vcpu": 1, "memoryMb": 512},
		"createdAt": timestamp(sb.createdAt),
		"startedAt": timestamp(sb.createdAt),
	}
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "name is required")
		return
	}
	sb := &sandbox{id: uuid.New(), name: body.Name, createdAt: time.Now(), processes: map[uuid.UUID]*process{}, nextPID: 100}
	s.mu.Lock()
	s.sandboxes[sb.id] = sb
	s.mu.Unlock()
	writeJSON(w, http.StatusCreated, s.sandboxResource(sb))
}

func (s *Server) lookup(w http.ResponseWriter, r *http.Request) *sandbox {
	id, err := uuid.Parse(r.PathValue("id"))
	s.mu.Lock()
	sb := s.sandboxes[id]
	s.mu.Unlock()
	if err != nil || sb == nil {
		writeError(w, http.StatusNotFound, "sandbox_not_found", "Sandbox not found")
		return nil
	}
	return sb
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	if sb := s.lookup(w, r); sb != nil {
		writeJSON(w, http.StatusOK, s.sandboxResource(sb))
	}
}

func (s *Server) destroy(w http.ResponseWriter, r *http.Request) {
	sb := s.lookup(w, r)
	if sb == nil {
		return
	}
	s.mu.Lock()
	delete(s.sandboxes, sb.id)
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) lookupProcess(w http.ResponseWriter, r *http.Request) *process {
	sb := s.lookup(w, r)
	if sb == nil {
		return nil
	}
	pid, err := uuid.Parse(r.PathValue("pid"))
	s.mu.Lock()
	p := sb.processes[pid]
	s.mu.Unlock()
	if err != nil || p == nil {
		writeError(w, http.StatusNotFound, "sandbox_process_not_found", "Process not found")
		return nil
	}
	return p
}

// resource must be called with s.mu held.
func (p *process) resource() map[string]any {
	res := map[string]any{
		"id":        p.id.String(),
		"command":   p.command,
		"state":     p.state,
		"startedAt": timestamp(p.startedAt),
	}
	if p.state == "RUNNING" {
		res["pid"] = p.pid
	}
	if p.exitCode != nil {
		res["exitCode"] = *p.exitCode
	}
	if p.termSignal != nil {
		res["terminationSignal"] = *p.termSignal
	}
	if !p.endedAt.IsZero() {
		res["endedAt"] = timestamp(p.endedAt)
	}
	return res
}

func (s *Server) startProcess(w http.ResponseWriter, r *http.Request) {
	sb := s.lookup(w, r)
	if sb == nil {
		return
	}
	var body struct {
		Command  []string `json:"command"`
		SignalID *string  `json:"signalId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Command) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "command is required")
		return
	}

	var token, signalID string
	if body.SignalID != nil {
		signalID = *body.SignalID
		var err error
		token, err = apiv1auth.CreateSignalJWT(s.secret, s.envID, signalID, time.Now().Add(maxLifetime))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "signalId must not be empty")
			return
		}
	}

	s.mu.Lock()
	sb.nextPID++
	p := &process{
		id:        uuid.New(),
		sandboxID: sb.id,
		command:   body.Command,
		pid:       sb.nextPID,
		state:     "RUNNING",
		startedAt: time.Now(),
		done:      make(chan struct{}),
	}
	sb.processes[p.id] = p
	res := p.resource()
	s.mu.Unlock()

	go s.supervise(p, signalID, token)
	writeJSON(w, http.StatusCreated, res)
}

func (s *Server) getProcess(w http.ResponseWriter, r *http.Request) {
	if p := s.lookupProcess(w, r); p != nil {
		s.mu.Lock()
		res := p.resource()
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, res)
	}
}

func (s *Server) waitProcess(w http.ResponseWriter, r *http.Request) {
	p := s.lookupProcess(w, r)
	if p == nil {
		return
	}
	timeout, err := time.ParseDuration(r.URL.Query().Get("timeout"))
	if err != nil || timeout <= 0 {
		timeout = 30 * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-p.done:
	case <-timer.C:
		writeError(w, http.StatusGatewayTimeout, "sandbox_process_wait_timed_out", "Timed out waiting for sandbox process")
		return
	case <-r.Context().Done():
		return
	}
	s.mu.Lock()
	res := p.resource()
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) processOutput(w http.ResponseWriter, r *http.Request) {
	p := s.lookupProcess(w, r)
	if p == nil {
		return
	}
	s.mu.Lock()
	stdout, stderr := append([]byte(nil), p.stdout...), append([]byte(nil), p.stderr...)
	s.mu.Unlock()
	chunks := []map[string]any{}
	for _, c := range []struct {
		stream string
		data   []byte
	}{{"STDOUT", stdout}, {"STDERR", stderr}} {
		if len(c.data) > 0 {
			chunks = append(chunks, map[string]any{
				"stream":   c.stream,
				"data":     base64.StdEncoding.EncodeToString(c.data),
				"encoding": "base64",
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"chunks": chunks})
}

func (s *Server) stats(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	total := 0
	for _, n := range s.requests {
		total += n
	}
	writeJSON(w, http.StatusOK, map[string]any{"total": total, "byRoute": s.requests, "probes": s.probes})
}

func (s *Server) resetStats(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	s.requests = map[string]int{}
	s.probes = nil
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// script returns the statements a process "runs".
func script(command []string) []string {
	text := strings.Join(command, " ")
	if len(command) == 3 && command[1] == "-c" {
		text = command[2]
	}
	var statements []string
	for _, line := range strings.FieldsFunc(text, func(r rune) bool { return r == ';' || r == '\n' }) {
		if line = strings.TrimSpace(line); line != "" {
			statements = append(statements, line)
		}
	}
	return statements
}

// supervise plays the part of the in-sandbox supervisor: it runs the script,
// records the terminal state, and sends the exit signal.
func (s *Server) supervise(p *process, signalID, token string) {
	state, exit, sig := "EXITED", 0, 0
	var probe bool
run:
	for _, statement := range script(p.command) {
		verb, arg, _ := strings.Cut(statement, " ")
		arg = strings.TrimSpace(arg)
		switch verb {
		case "sleep":
			if seconds, err := strconv.ParseFloat(arg, 64); err == nil && seconds >= 0 && seconds <= maxSleepSeconds {
				time.Sleep(time.Duration(seconds * float64(time.Second)))
			}
		case "echo":
			s.append(&p.stdout, arg+"\n")
		case "err":
			s.append(&p.stderr, arg+"\n")
		case "flood":
			kib, err := strconv.Atoi(arg)
			if err != nil || kib < 0 || kib > maxFloodKiB {
				s.append(&p.stderr, "fake sandbox: invalid statement "+statement+"\n")
				exit = 127
				break run
			}
			s.append(&p.stdout, strings.Repeat("0123456789abcdef", 64*kib))
		case "probe":
			probe = true
		case "exit":
			exit, _ = strconv.Atoi(arg)
			break run
		case "kill":
			state, sig = "KILLED", 9
			if n, err := strconv.Atoi(arg); err == nil && n > 0 && n <= 64 {
				sig = n
			}
			break run
		default:
			s.append(&p.stderr, "fake sandbox: unknown statement "+statement+"\n")
			exit = 127
			break run
		}
	}

	s.mu.Lock()
	p.state, p.endedAt = state, time.Now()
	if state == "EXITED" {
		p.exitCode = &exit
	} else {
		p.termSignal = &sig
	}
	payload := s.exitPayload(p)
	s.mu.Unlock()
	close(p.done)

	if signalID == "" {
		return
	}
	if probe {
		status := s.post(token, signalID+":other", payload, false)
		s.mu.Lock()
		s.probes = append(s.probes, fmt.Sprintf("token for %q used on %q: HTTP %d", signalID, signalID+":other", status))
		s.mu.Unlock()
	}
	s.post(token, signalID, payload, true)
}

func (s *Server) append(buf *[]byte, text string) {
	s.mu.Lock()
	*buf = append(*buf, text...)
	s.mu.Unlock()
}

// tail returns the last tailBytes of data as valid UTF-8 and whether it cut
// anything.
func tail(data []byte) (string, bool) {
	if len(data) <= tailBytes {
		return strings.ToValidUTF8(string(data), "�"), false
	}
	data = data[len(data)-tailBytes:]
	for len(data) > 0 && !utf8.RuneStart(data[0]) {
		data = data[1:]
	}
	return strings.ToValidUTF8(string(data), "�"), true
}

// exitPayload must be called with s.mu held.
func (s *Server) exitPayload(p *process) map[string]any {
	stdout, cutOut := tail(p.stdout)
	stderr, cutErr := tail(p.stderr)
	payload := map[string]any{
		"sandboxId": p.sandboxID.String(),
		"processId": p.id.String(),
		"state":     p.state,
		"endedAt":   timestamp(p.endedAt),
		"stdout":    stdout,
		"stderr":    stderr,
		"truncated": cutOut || cutErr,
	}
	if p.exitCode != nil {
		payload["exitCode"] = *p.exitCode
	}
	if p.termSignal != nil {
		payload["terminationSignal"] = *p.termSignal
	}
	return payload
}

// post sends the signal with the scoped token and returns the last HTTP status.
// With retry, it retries transport errors and 5xx responses with capped
// backoff for up to retryWindow; any other status ends it.
func (s *Server) post(token, signalID string, payload map[string]any, retry bool) int {
	body, err := json.Marshal(map[string]any{"signal": signalID, "data": payload})
	if err != nil {
		return 0
	}
	deadline := time.Now().Add(retryWindow)
	delay := 100 * time.Millisecond
	for {
		status := 0
		req, err := http.NewRequest(http.MethodPost, s.apiBase+"/v1/signals/scoped", bytes.NewReader(body))
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			if res, err := http.DefaultClient.Do(req); err == nil {
				status = res.StatusCode
				_, _ = io.Copy(io.Discard, res.Body)
				_ = res.Body.Close()
			}
		}
		if status != 0 && status < 500 {
			return status
		}
		if !retry || time.Now().Add(delay).After(deadline) {
			return status
		}
		time.Sleep(delay)
		if delay *= 2; delay > maxRetryDelay {
			delay = maxRetryDelay
		}
	}
}
