package checkpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/inngest/inngestgo/pkg/env"
)

var hc = &http.Client{Timeout: 30 * time.Second}

func checkpointURL(runID string) string {
	return fmt.Sprintf("%s/v1/checkpoint/%s/async", env.APIServerURL(), runID)
}

func checkpoint(ctx context.Context, key string, req AsyncRequest) error {
	byt, err := json.Marshal(req)
	if err != nil {
		return err
	}

	hr, err := http.NewRequest(
		http.MethodPost,
		checkpointURL(req.RunID),
		bytes.NewBuffer(byt),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	hr.Header.Set("Content-Type", "application/json")
	hr.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))

	resp, err := hc.Do(hr)
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
		// TODO: Signing key fallbacks.
		return fmt.Errorf("error checkpointing (%d): %s", resp.StatusCode, byt)
	}

	return nil
}
