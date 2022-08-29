package main

import "context"

// mockfetcher is an in-memory mock fetcher.  In the real world, you might
// fetch your information from an API, a database, or an external service.
type mockfetcher struct{}

func (mockfetcher) Fetch(ctx context.Context) ([]Summary, error) {
	return nil, nil
}
