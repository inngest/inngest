package sdkrequest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/sync/errgroup"
)

// LoadFromAPIOpts configures loading request data omitted from an executor
// request because it exceeded the SDK request body limit.
type LoadFromAPIOpts struct {
	APIBaseURL        string
	AuthToken         string
	AuthTokenFallback string
	HTTPClient        *http.Client
}

// LoadFromAPI hydrates the event batch and memoized step state for requests
// with use_api enabled.
func LoadFromAPI(ctx context.Context, request *Request, opts LoadFromAPIOpts) error {
	if !request.UsesAPI() {
		return nil
	}
	if request.CallCtx.RunID == "" {
		return fmt.Errorf("cannot retrieve request data from API without a run ID")
	}

	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	var (
		events []json.RawMessage
		steps  map[string]json.RawMessage
	)

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		path := fmt.Sprintf("/v0/runs/%s/batch", url.PathEscape(request.CallCtx.RunID))
		if err := fetchAPIData(groupCtx, client, opts, path, &events); err != nil {
			return fmt.Errorf("failed to retrieve event batch: %w", err)
		}
		return nil
	})
	group.Go(func() error {
		path := fmt.Sprintf("/v0/runs/%s/actions", url.PathEscape(request.CallCtx.RunID))
		if err := fetchAPIData(groupCtx, client, opts, path, &steps); err != nil {
			return fmt.Errorf("failed to retrieve step state: %w", err)
		}
		return nil
	})
	if err := group.Wait(); err != nil {
		return err
	}

	request.Events = events
	request.Steps = steps
	return nil
}

func fetchAPIData(ctx context.Context, client *http.Client, opts LoadFromAPIOpts, path string, target any) error {
	endpoint := strings.TrimRight(opts.APIBaseURL, "/") + path
	tokens := []string{opts.AuthToken}
	if opts.AuthTokenFallback != "" && opts.AuthTokenFallback != opts.AuthToken {
		tokens = append(tokens, opts.AuthTokenFallback)
	}

	for i, token := range tokens {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return fmt.Errorf("failed to create API request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("API request failed: %w", err)
		}

		if (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) && i+1 < len(tokens) {
			_ = resp.Body.Close()
			continue
		}

		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
			_ = resp.Body.Close()
			return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		err = json.NewDecoder(resp.Body).Decode(target)
		_ = resp.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to decode API response: %w", err)
		}
		return nil
	}

	return fmt.Errorf("API request failed")
}
