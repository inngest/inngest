package checkpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

type Client struct {
	primaryKey  string
	fallbackKey string
	apiBaseURL  string
	httpClient  *http.Client
	useFallback *atomic.Bool
}

func NewClient(apiURL, primaryKey, fallbackKey string) *Client {
	return &Client{
		primaryKey:  primaryKey,
		fallbackKey: fallbackKey,
		apiBaseURL:  apiURL,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		useFallback: &atomic.Bool{},
	}
}

func (c *Client) checkpointURL(runID string) string {
	return fmt.Sprintf("%s/v1/checkpoint/%s/async", c.apiBaseURL, runID)
}

func (c *Client) Checkpoint(ctx context.Context, req AsyncRequest) error {
	return c.do(ctx, req)
}

func (c *Client) do(ctx context.Context, req AsyncRequest) error {
	byt, err := json.Marshal(req)
	if err != nil {
		return err
	}

	hr, err := http.NewRequest(
		http.MethodPost,
		c.checkpointURL(req.RunID),
		bytes.NewBuffer(byt),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	hr.Header.Set("Content-Type", "application/json")
	hr.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.getCurrentSigningKey()))

	resp, err := c.httpClient.Do(hr)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	byt, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		// If we get a 401 and have a fallback key, try switching to it
		if resp.StatusCode == 401 && c.fallbackKey != "" && !c.useFallback.Load() {
			c.useFallback.Store(true)
			// Retry the request with the fallback key
			return c.do(ctx, req)
		}
		return fmt.Errorf("error checkpointing (%d): %s", resp.StatusCode, byt)
	}

	return nil
}

func (c *Client) getCurrentSigningKey() string {
	if c.useFallback.Load() {
		return c.fallbackKey
	}
	return c.primaryKey
}
