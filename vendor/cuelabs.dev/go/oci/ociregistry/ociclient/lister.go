// Copyright 2023 CUE Labs AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ociclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"slices"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"cuelabs.dev/go/oci/ociregistry"
	"cuelabs.dev/go/oci/ociregistry/internal/ocirequest"
)

func (c *client) Repositories(ctx context.Context, startAfter string) iter.Seq2[string, error] {
	return pager(ctx, c, &ocirequest.Request{
		Kind:     ocirequest.ReqCatalogList,
		ListN:    c.listPageSize,
		ListLast: startAfter,
	}, true, func(resp *http.Response) ([]string, error) {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var catalog struct {
			Repos []string `json:"repositories"`
		}
		if err := json.Unmarshal(data, &catalog); err != nil {
			return nil, fmt.Errorf("cannot unmarshal catalog response: %v", err)
		}
		return catalog.Repos, nil
	})
}

func (c *client) Tags(ctx context.Context, repoName, startAfter string) iter.Seq2[string, error] {
	return pager(ctx, c, &ocirequest.Request{
		Kind:     ocirequest.ReqTagsList,
		Repo:     repoName,
		ListN:    c.listPageSize,
		ListLast: startAfter,
	}, true, func(resp *http.Response) ([]string, error) {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var tagsResponse struct {
			Repo string   `json:"name"`
			Tags []string `json:"tags"`
		}
		if err := json.Unmarshal(data, &tagsResponse); err != nil {
			return nil, fmt.Errorf("cannot unmarshal tags list response: %v", err)
		}
		return tagsResponse.Tags, nil
	})
}

func (c *client) Referrers(ctx context.Context, repoName string, digest ociregistry.Digest, artifactType string) iter.Seq2[ociregistry.Descriptor, error] {
	return pager(ctx, c, &ocirequest.Request{
		Kind:         ocirequest.ReqReferrersList,
		Repo:         repoName,
		Digest:       string(digest),
		ListN:        c.listPageSize,
		ArtifactType: artifactType,
	}, false, func(resp *http.Response) ([]ociregistry.Descriptor, error) {
		body := resp.Body
		if resp.StatusCode == http.StatusNotFound {
			body.Close()
			body = nil
			// Fall back to the referrers tag API.
			// From https://github.com/opencontainers/distribution-spec/blob/main/spec.md#unavailable-referrers-api :
			//	A client querying the referrers API and receiving a
			//	404 Not Found MUST fallback to using an image index
			//	pushed to a tag described by the referrers tag
			//	schema.
			r, err := c.GetTag(ctx, repoName, referrersTag(digest))
			if err != nil {
				if errors.Is(err, ociregistry.ErrManifestUnknown) {
					return nil, nil
				}
				return nil, err
			}
			body = r
		}
		data, err := io.ReadAll(body)
		body.Close()
		if err != nil {
			return nil, err
		}
		var referrersResponse ocispec.Index
		if err := json.Unmarshal(data, &referrersResponse); err != nil {
			return nil, fmt.Errorf("cannot unmarshal referrers response: %v", err)
		}
		if artifactType == "" || resp.Header.Get("OCI-Filters-Applied") == "artifactType" {
			return referrersResponse.Manifests, nil
		}
		// The server hasn't filtered the responses, so we must.
		// TODO is it OK to assume that the index contains correctly populated
		// artifact type and attributes fields when we've fallen back to the referrer tags API?
		// If not, we might have to retrieve all the individual manifests to check that info.
		manifests := slices.DeleteFunc(referrersResponse.Manifests, func(desc ociregistry.Descriptor) bool {
			return desc.ArtifactType != artifactType
		})
		return manifests, nil
	}, http.StatusOK, http.StatusNotFound)
}

// pager returns an iterator for a list entry point. It starts by sending the given
// initial request and parses each response into its component items using
// parseResponse. It tries to use the Link header in each response to continue
// the iteration, falling back to using the "last" query parameter if
// canUseLast is true.
func pager[T any](ctx context.Context, c *client, initialReq *ocirequest.Request, canUseLast bool, parseResponse func(*http.Response) ([]T, error), okStatuses ...int) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		// We assume that the same auth scope is applicable to all page requests.
		req, err := newRequest(ctx, initialReq, nil)
		if err != nil {
			yield(*new(T), err)
			return
		}
		for {
			resp, err := c.do(req, okStatuses...)
			if err != nil {
				yield(*new(T), err)
				return
			}
			items, err := parseResponse(resp)
			resp.Body.Close()
			if err != nil {
				yield(*new(T), err)
				return
			}
			for _, item := range items {
				if !yield(item, nil) {
					return
				}
			}
			if len(items) < initialReq.ListN {
				// From the distribution spec:
				//     The response to such a request MAY return fewer than <int> results,
				//     but only when the total number of tags attached to the repository
				//     is less than <int>.
				return
			}
			req, err = nextLink(ctx, resp, initialReq, canUseLast, items[len(items)-1])
			if err != nil {
				yield(*new(T), fmt.Errorf("invalid Link header in response: %v", err))
				return
			}
			if req == nil {
				// No link found; assume there are no more items.
				return
			}
		}
	}
}

// nextLink ttries to form a request that can be sent to obtain the next page
// in a set of list results.
// The given response holds the response received from the previous
// list request; initialReq holds the request that initiated the listing,
// and last holds the final item returned in the previous response.
func nextLink[T any](ctx context.Context, resp *http.Response, initialReq *ocirequest.Request, canUseLast bool, last T) (*http.Request, error) {
	link0 := resp.Header.Get("Link")
	if link0 == "" {
		if !canUseLast {
			return nil, nil
		}
		// This is beyond the first page and there was no Link
		// in the previous response (the standard doesn't mandate
		// one), so add a "last" parameter to the initial request.
		rreq := *initialReq
		rreq.ListLast = fmt.Sprint(last)
		req, err := newRequest(ctx, &rreq, nil)
		if err != nil {
			// Given that we could form the initial request, this should
			// never happen.
			return nil, fmt.Errorf("cannot form next request: %v", err)
		}
		return req, nil
	}
	// Parse the link header according to RFC 5988.
	// TODO perhaps we shouldn't ignore the relation type?
	link, ok := strings.CutPrefix(link0, "<")
	if !ok {
		return nil, fmt.Errorf("no initial < character in Link=%q", link0)
	}
	link, _, ok = strings.Cut(link, ">")
	if !ok {
		return nil, fmt.Errorf("no > character in Link=%q", link0)
	}
	// Parse it with respect to the originating request, as it's probably relative.
	linkURL, err := resp.Request.URL.Parse(link)
	if err != nil {
		return nil, fmt.Errorf("invalid URL in Link=%q", link0)
	}
	return http.NewRequestWithContext(ctx, "GET", linkURL.String(), nil)
}

// referrersTag returns the referrers tag for the given digest, as described
// in https://github.com/opencontainers/distribution-spec/blob/main/spec.md#referrers-tag-schema
func referrersTag(digest ociregistry.Digest) string {
	// It's hard to know what the spec means by "with any characters not allowed by <reference> tags replaced with -",
	// because different characters are allowed in different contexts (for example, a dot character
	// is allowed except when it's at the start.
	// In practice, however, the set of characters is very limited, and the only
	// disallowed character in common use is :, so just use a naive algorithm.
	return truncateAndMap(digest.Algorithm().String(), 32) + "-" + truncateAndMap(digest.Encoded(), 64)
}

func truncateAndMap(s string, n int) string {
	// regexp: [a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}

	s = strings.Map(func(r rune) rune {
		switch {
		case 'a' <= r && r <= 'z':
			return r
		case 'A' <= r && r <= 'Z':
			return r
		case '0' <= r && r <= '9':
			return r
		case r == '.' || r == '_' || r == '-':
			return r
		}
		return '-'
	}, s)
	// Note: it's OK to use n as a byte index because the
	// above Map has eliminated all non-ASCII characters.
	if len(s) <= n {
		return s
	}
	return s[:n]
}
