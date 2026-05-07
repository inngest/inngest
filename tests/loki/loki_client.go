//go:build e2e_loki

package loki

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"
)

// queryRangeResponse mirrors the bits of Loki's /loki/api/v1/query_range
// response we care about. https://grafana.com/docs/loki/latest/reference/api/
type queryRangeResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Stream map[string]string `json:"stream"`
			Values [][2]string       `json:"values"` // [ts_ns_string, line]
		} `json:"result"`
	} `json:"data"`
}

// queryLoki polls /loki/api/v1/query_range with the given LogQL until at
// least min records match or the deadline expires. Returns the parsed log
// lines (each line is JSON, decoded into map[string]any) and the matching
// stream labels for each line, in the same order.
//
// Loki has chunk-flush latency at low ingestion volume; the polling loop is
// the simplest fix that keeps the test deterministic.
func queryLoki(t *testing.T, lokiBase, logql string, min int, timeout time.Duration) ([]map[string]any, []map[string]string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	delay := 250 * time.Millisecond

	var lastBody string
	for time.Now().Before(deadline) {
		bodies, labels, body := tryQuery(t, lokiBase, logql)
		lastBody = body
		if len(bodies) >= min {
			return bodies, labels
		}
		time.Sleep(delay)
		if delay < 2*time.Second {
			delay *= 2
		}
	}
	t.Fatalf("queryLoki: wanted >= %d records for %q within %s; last response:\n%s", min, logql, timeout, lastBody)
	return nil, nil
}

func tryQuery(t *testing.T, lokiBase, logql string) ([]map[string]any, []map[string]string, string) {
	t.Helper()

	now := time.Now()
	q := url.Values{}
	q.Set("query", logql)
	q.Set("start", strconv.FormatInt(now.Add(-15*time.Minute).UnixNano(), 10))
	q.Set("end", strconv.FormatInt(now.Add(1*time.Minute).UnixNano(), 10))
	q.Set("limit", "1000")
	q.Set("direction", "forward")

	endpoint := fmt.Sprintf("http://%s/loki/api/v1/query_range?%s", lokiBase, q.Encode())

	resp, err := http.Get(endpoint)
	if err != nil {
		return nil, nil, fmt.Sprintf("http error: %v", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Sprintf("read body: %v", err)
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var parsed queryRangeResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, nil, fmt.Sprintf("decode: %v\n%s", err, string(raw))
	}

	var bodies []map[string]any
	var labels []map[string]string
	for _, stream := range parsed.Data.Result {
		for _, kv := range stream.Values {
			line := kv[1]
			var m map[string]any
			if err := json.Unmarshal([]byte(line), &m); err != nil {
				// Loki returned a non-JSON line — surface it but keep going.
				m = map[string]any{"_raw": line}
			}
			bodies = append(bodies, m)
			labels = append(labels, stream.Stream)
		}
	}
	return bodies, labels, string(raw)
}
