package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/inngest/inngestgo"
)

type targetFlags map[string]string

func (t targetFlags) String() string { return fmt.Sprint(map[string]string(t)) }

func (t targetFlags) Set(value string) error {
	name, baseURL, ok := strings.Cut(value, "=")
	if !ok || name == "" || baseURL == "" {
		return fmt.Errorf("target must be NAME=URL")
	}
	t[name] = baseURL
	return nil
}

type manifest struct {
	SDK            sdkIdentity `json:"sdk"`
	ServePath      string      `json:"serve_path"`
	CloudServePath string      `json:"cloud_serve_path"`
	Fixtures       []fixture   `json:"fixtures"`
}

type sdkIdentity struct {
	Language string `json:"language"`
	Version  string `json:"version"`
	Commit   string `json:"commit,omitempty"`
}

type fixture struct {
	ID         string `json:"id"`
	FunctionID string `json:"function_id"`
	Event      string `json:"event"`
}

type probe struct {
	ID      string       `json:"id"`
	Fixture string       `json:"fixture"`
	Request probeRequest `json:"request"`
}

type probeRequest struct {
	Method                          string            `json:"method"`
	Path                            string            `json:"path"`
	Sign                            bool              `json:"sign,omitempty"`
	SigningKey                      string            `json:"signing_key,omitempty"`
	SignatureTimestampOffsetSeconds int               `json:"signature_timestamp_offset_seconds,omitempty"`
	Query                           map[string]string `json:"query,omitempty"`
	Headers                         map[string]string `json:"headers,omitempty"`
	JSON                            any               `json:"json,omitempty"`
}

type observation struct {
	Status  int               `json:"status,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	JSON    any               `json:"json,omitempty"`
	Body    string            `json:"body,omitempty"`
	Error   string            `json:"error,omitempty"`
}

type caseReport struct {
	ID               string                 `json:"id"`
	Observations     map[string]observation `json:"observations"`
	MatchesReference map[string]bool        `json:"matches_reference"`
}

type report struct {
	SchemaVersion string              `json:"schema_version"`
	Reference     string              `json:"reference"`
	Targets       map[string]manifest `json:"targets"`
	Cases         []caseReport        `json:"cases"`
}

var comparedHeaders = []string{
	"content-type",
	"retry-after",
	"x-inngest-no-retry",
	"x-inngest-req-version",
	"x-inngest-sync-kind",
}

var signingKeys = map[string][]byte{
	"primary":  []byte("7468697320697320612074657374206b6579"),
	"fallback": []byte("66616c6c6261636b20636f6e666f726d616e6365"),
	"wrong":    []byte("77726f6e6720636f6e666f726d616e6365206b6579"),
}

func main() {
	probesDir := flag.String("probes", "sdk-conformance/probes", "directory containing observation probes")
	reference := flag.String("reference", "typescript", "target used as the reference observation")
	output := flag.String("output", "", "write the JSON report to this path; defaults to stdout")
	targets := targetFlags{}
	flag.Var(targets, "target", "fixture target as NAME=URL; repeat for every SDK")
	flag.Parse()

	if len(targets) == 0 {
		fatalf("at least one --target is required")
	}
	if _, ok := targets[*reference]; !ok {
		fatalf("reference target %q was not provided", *reference)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	manifests := make(map[string]manifest, len(targets))
	for _, name := range sortedKeys(targets) {
		loaded, err := loadManifest(client, targets[name])
		if err != nil {
			fatalf("load %s manifest: %v", name, err)
		}
		manifests[name] = loaded
	}

	probes, err := loadProbes(*probesDir)
	if err != nil {
		fatalf("load probes: %v", err)
	}

	result := report{
		SchemaVersion: "0.1",
		Reference:     *reference,
		Targets:       manifests,
		Cases:         make([]caseReport, 0, len(probes)),
	}
	for _, current := range probes {
		caseResult := caseReport{
			ID:               current.ID,
			Observations:     map[string]observation{},
			MatchesReference: map[string]bool{},
		}
		for _, name := range sortedKeys(targets) {
			if err := resetTarget(client, targets[name]); err != nil {
				caseResult.Observations[name] = observation{Error: "reset fixture: " + err.Error()}
				continue
			}
			caseResult.Observations[name] = runProbe(client, targets[name], manifests[name], current)
		}
		referenceObservation := normalize(caseResult.Observations[*reference])
		for _, name := range sortedKeys(targets) {
			caseResult.MatchesReference[name] = reflect.DeepEqual(referenceObservation, normalize(caseResult.Observations[name]))
		}
		result.Cases = append(result.Cases, caseResult)
	}

	payload, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fatalf("encode report: %v", err)
	}
	payload = append(payload, '\n')
	if *output == "" {
		_, _ = os.Stdout.Write(payload)
		return
	}
	if err := os.WriteFile(*output, payload, 0o644); err != nil {
		fatalf("write report: %v", err)
	}
	for _, current := range result.Cases {
		different := make([]string, 0)
		for name, matches := range current.MatchesReference {
			if name != *reference && !matches {
				different = append(different, name)
			}
		}
		sort.Strings(different)
		if len(different) == 0 {
			fmt.Printf("MATCH %s\n", current.ID)
		} else {
			fmt.Printf("DIFF  %s (%s)\n", current.ID, strings.Join(different, ", "))
		}
	}
}

func loadManifest(client *http.Client, baseURL string) (manifest, error) {
	endpoint, err := url.JoinPath(baseURL, "/__conformance")
	if err != nil {
		return manifest{}, err
	}
	resp, err := client.Get(endpoint)
	if err != nil {
		return manifest{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return manifest{}, fmt.Errorf("status %d", resp.StatusCode)
	}
	var value manifest
	if err := json.NewDecoder(resp.Body).Decode(&value); err != nil {
		return manifest{}, err
	}
	if value.SDK.Language == "" || value.ServePath == "" {
		return manifest{}, fmt.Errorf("manifest is missing SDK language or serve path")
	}
	return value, nil
}

func resetTarget(client *http.Client, baseURL string) error {
	endpoint, err := url.JoinPath(baseURL, "/__conformance/reset")
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func loadProbes(dir string) ([]probe, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no probes found in %s", dir)
	}
	sort.Strings(paths)
	result := make([]probe, 0, len(paths))
	for _, path := range paths {
		contents, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var value probe
		if err := json.Unmarshal(contents, &value); err != nil {
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
		if value.ID == "" || value.Request.Method == "" || value.Request.Path == "" {
			return nil, fmt.Errorf("%s is missing required fields", path)
		}
		result = append(result, value)
	}
	return result, nil
}

func runProbe(client *http.Client, baseURL string, target manifest, current probe) observation {
	values := map[string]string{
		"serve_path":       target.ServePath,
		"cloud_serve_path": target.CloudServePath,
	}
	if current.Fixture != "" {
		found, ok := findFixture(target, current.Fixture)
		if !ok {
			return observation{Error: fmt.Sprintf("fixture %q is not declared", current.Fixture)}
		}
		values["function_id"] = found.FunctionID
		values["event"] = found.Event
	}
	resolved, err := substitute(current.Request, values)
	if err != nil {
		return observation{Error: err.Error()}
	}

	endpoint, err := url.JoinPath(baseURL, resolved.Path)
	if err != nil {
		return observation{Error: err.Error()}
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return observation{Error: err.Error()}
	}
	query := u.Query()
	for key, value := range resolved.Query {
		query.Set(key, value)
	}
	u.RawQuery = query.Encode()

	var body io.Reader
	if resolved.JSON != nil {
		payload, err := json.Marshal(resolved.JSON)
		if err != nil {
			return observation{Error: err.Error()}
		}
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(resolved.Method, u.String(), body)
	if err != nil {
		return observation{Error: err.Error()}
	}
	for key, value := range resolved.Headers {
		req.Header.Set(key, value)
	}
	if resolved.JSON != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if resolved.Sign {
		var payload []byte
		if req.Body != nil {
			payload, err = io.ReadAll(req.Body)
			if err != nil {
				return observation{Error: err.Error()}
			}
			req.Body = io.NopCloser(bytes.NewReader(payload))
		}
		keyName := resolved.SigningKey
		if keyName == "" {
			keyName = "primary"
		}
		key, ok := signingKeys[keyName]
		if !ok {
			return observation{Error: fmt.Sprintf("unknown signing key %q", keyName)}
		}
		at := time.Now().Add(time.Duration(resolved.SignatureTimestampOffsetSeconds) * time.Second)
		signature, err := inngestgo.Sign(context.Background(), at, key, payload)
		if err != nil {
			return observation{Error: err.Error()}
		}
		req.Header.Set("X-Inngest-Signature", signature)
	}
	resp, err := client.Do(req)
	if err != nil {
		return observation{Error: err.Error()}
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return observation{Status: resp.StatusCode, Error: err.Error()}
	}

	result := observation{Status: resp.StatusCode, Headers: map[string]string{}}
	for _, key := range comparedHeaders {
		if value := resp.Header.Get(key); value != "" {
			result.Headers[key] = value
		}
	}
	if len(result.Headers) == 0 {
		result.Headers = nil
	}
	if len(bytes.TrimSpace(payload)) == 0 {
		return result
	}
	if err := json.Unmarshal(payload, &result.JSON); err != nil {
		result.Body = string(payload)
	}
	return result
}

func substitute(input probeRequest, values map[string]string) (probeRequest, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return probeRequest{}, err
	}
	text := string(payload)
	for key, value := range values {
		text = strings.ReplaceAll(text, "{{"+key+"}}", value)
	}
	if strings.Contains(text, "{{") {
		return probeRequest{}, fmt.Errorf("unresolved template in request")
	}
	var result probeRequest
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return probeRequest{}, err
	}
	return result, nil
}

func findFixture(target manifest, id string) (fixture, bool) {
	for _, value := range target.Fixtures {
		if value.ID == id {
			return value, true
		}
	}
	return fixture{}, false
}

func normalize(input observation) observation {
	input.JSON = normalizeJSON(input.JSON)
	if input.Body != "" {
		input.Body = strings.TrimSpace(input.Body)
	}
	return input
}

func normalizeJSON(input any) any {
	switch value := input.(type) {
	case map[string]any:
		result := make(map[string]any, len(value))
		for key, child := range value {
			switch strings.ToLower(key) {
			case "stack", "trace", "sdk_language", "sdk_version":
				continue
			default:
				result[key] = normalizeJSON(child)
			}
		}
		return result
	case []any:
		result := make([]any, len(value))
		for index, child := range value {
			result[index] = normalizeJSON(child)
		}
		return result
	default:
		return input
	}
}

func sortedKeys[V any](values map[string]V) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}
