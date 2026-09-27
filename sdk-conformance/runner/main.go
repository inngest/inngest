package main

import (
	"bytes"
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
)

type testCase struct {
	ID           string   `json:"id"`
	Status       string   `json:"status"`
	Profiles     []string `json:"profiles"`
	Capabilities []string `json:"capabilities"`
	Request      request  `json:"request"`
	Expect       expected `json:"expect"`
}

type targetManifest struct {
	Profiles     []string `json:"profiles"`
	Capabilities []string `json:"capabilities"`
	ServePath    string   `json:"serve_path"`
}

type request struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Query   map[string]string `json:"query"`
	Headers map[string]string `json:"headers"`
	JSON    any               `json:"json"`
}

type expected struct {
	Status     int         `json:"status"`
	Assertions []assertion `json:"assertions"`
}

type assertion struct {
	Path     string `json:"path"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
	Type     string `json:"type"`
}

func main() {
	casesDir := flag.String("cases", "sdk-conformance/cases", "directory containing case JSON files")
	target := flag.String("target", "", "fixture base URL")
	includeDisputed := flag.Bool("include-disputed", false, "execute disputed, non-blocking cases")
	flag.Parse()

	if *target == "" {
		fmt.Fprintln(os.Stderr, "--target is required")
		os.Exit(2)
	}

	cases, err := loadCases(*casesDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	manifest, err := loadTarget(http.DefaultClient, *target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load target manifest: %v\n", err)
		os.Exit(2)
	}

	failures := 0
	for _, tc := range cases {
		if tc.Status == "disputed" && !*includeDisputed {
			fmt.Printf("SKIP %s (disputed)\n", tc.ID)
			continue
		}
		if missing := unsupported(tc, manifest); len(missing) > 0 {
			fmt.Printf("UNSUPPORTED %s (%s)\n", tc.ID, strings.Join(missing, ", "))
			continue
		}
		if err := runCase(http.DefaultClient, *target, manifest, tc); err != nil {
			if tc.Status == "disputed" {
				fmt.Printf("DISPUTED %s: %v\n", tc.ID, err)
				continue
			}
			failures++
			fmt.Printf("FAIL %s: %v\n", tc.ID, err)
			continue
		}
		fmt.Printf("PASS %s\n", tc.ID)
	}
	if failures > 0 {
		os.Exit(1)
	}
}

func loadTarget(client *http.Client, target string) (targetManifest, error) {
	endpoint, err := url.JoinPath(target, "/__conformance")
	if err != nil {
		return targetManifest{}, err
	}
	resp, err := client.Get(endpoint)
	if err != nil {
		return targetManifest{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return targetManifest{}, fmt.Errorf("status %d", resp.StatusCode)
	}
	var manifest targetManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return targetManifest{}, err
	}
	if manifest.ServePath == "" {
		return targetManifest{}, fmt.Errorf("manifest has no serve_path")
	}
	return manifest, nil
}

func unsupported(tc testCase, target targetManifest) []string {
	missing := make([]string, 0)
	if !intersects(tc.Profiles, target.Profiles) {
		missing = append(missing, "profile")
	}
	for _, capability := range tc.Capabilities {
		if !contains(target.Capabilities, capability) {
			missing = append(missing, capability)
		}
	}
	return missing
}

func intersects(left, right []string) bool {
	for _, value := range left {
		if contains(right, value) {
			return true
		}
	}
	return false
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func loadCases(dir string) ([]testCase, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no case files found in %s", dir)
	}
	sort.Strings(paths)

	cases := make([]testCase, 0, len(paths))
	for _, path := range paths {
		contents, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var tc testCase
		if err := json.Unmarshal(contents, &tc); err != nil {
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
		if tc.ID == "" || tc.Request.Method == "" || tc.Request.Path == "" || tc.Expect.Status == 0 {
			return nil, fmt.Errorf("%s is missing required executable fields", path)
		}
		cases = append(cases, tc)
	}
	return cases, nil
}

func runCase(client *http.Client, target string, manifest targetManifest, tc testCase) error {
	path := strings.ReplaceAll(tc.Request.Path, "{{serve_path}}", manifest.ServePath)
	endpoint, err := url.JoinPath(target, path)
	if err != nil {
		return err
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	query := u.Query()
	for key, value := range tc.Request.Query {
		query.Set(key, value)
	}
	u.RawQuery = query.Encode()

	var body io.Reader
	if tc.Request.JSON != nil {
		payload, err := json.Marshal(tc.Request.JSON)
		if err != nil {
			return err
		}
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(tc.Request.Method, u.String(), body)
	if err != nil {
		return err
	}
	for key, value := range tc.Request.Headers {
		req.Header.Set(key, value)
	}
	if tc.Request.JSON != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != tc.Expect.Status {
		return fmt.Errorf("status %d, want %d", resp.StatusCode, tc.Expect.Status)
	}
	if len(tc.Expect.Assertions) == 0 {
		return nil
	}
	var document any
	if err := json.NewDecoder(resp.Body).Decode(&document); err != nil {
		return fmt.Errorf("decode response JSON: %w", err)
	}
	for _, assertion := range tc.Expect.Assertions {
		if err := checkAssertion(document, assertion); err != nil {
			return err
		}
	}
	return nil
}

func checkAssertion(document any, assertion assertion) error {
	value, found := lookup(document, assertion.Path)
	switch assertion.Operator {
	case "present":
		if !found {
			return fmt.Errorf("%s is absent", assertion.Path)
		}
	case "absent":
		if found {
			return fmt.Errorf("%s is present", assertion.Path)
		}
	case "equals":
		if !found || !reflect.DeepEqual(value, assertion.Value) {
			return fmt.Errorf("%s = %v, want %v", assertion.Path, value, assertion.Value)
		}
	case "type":
		if !found || jsonType(value) != assertion.Type {
			return fmt.Errorf("%s has type %s, want %s", assertion.Path, jsonType(value), assertion.Type)
		}
	default:
		return fmt.Errorf("unknown assertion operator %q", assertion.Operator)
	}
	return nil
}

func lookup(document any, path string) (any, bool) {
	if path == "$" {
		return document, true
	}
	if !strings.HasPrefix(path, "$.") {
		return nil, false
	}
	current := document
	for _, part := range strings.Split(strings.TrimPrefix(path, "$."), ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func jsonType(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	default:
		return "unknown"
	}
}
