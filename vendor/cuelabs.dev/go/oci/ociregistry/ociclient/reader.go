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
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"cuelabs.dev/go/oci/ociregistry"
	"cuelabs.dev/go/oci/ociregistry/internal/ocirequest"
	"github.com/opencontainers/go-digest"
)

func (c *client) GetBlob(ctx context.Context, repo string, digest ociregistry.Digest) (ociregistry.BlobReader, error) {
	return c.read(ctx, &ocirequest.Request{
		Kind:   ocirequest.ReqBlobGet,
		Repo:   repo,
		Digest: string(digest),
	})
}

func (c *client) GetBlobRange(ctx context.Context, repo string, digest ociregistry.Digest, o0, o1 int64) (_ ociregistry.BlobReader, _err error) {
	if o0 == 0 && o1 < 0 {
		return c.GetBlob(ctx, repo, digest)
	}
	rreq := &ocirequest.Request{
		Kind:   ocirequest.ReqBlobGet,
		Repo:   repo,
		Digest: string(digest),
	}
	req, err := newRequest(ctx, rreq, nil)
	if err != nil {
		return nil, err
	}
	if o1 < 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", o0))
	} else {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", o0, o1-1))
	}
	resp, err := c.do(req, http.StatusOK, http.StatusPartialContent)
	if err != nil {
		return nil, err
	}
	// TODO this is wrong when the server returns a 200 response.
	// Fix that either by returning ErrUnsupported or by reading the whole
	// blob and returning only the required portion.
	defer closeOnError(&_err, resp.Body)
	desc, err := descriptorFromResponse(resp, ociregistry.Digest(rreq.Digest), requireSize)
	if err != nil {
		return nil, fmt.Errorf("invalid descriptor in response: %v", err)
	}
	return newBlobReaderUnverified(resp.Body, desc), nil
}

func (c *client) ResolveBlob(ctx context.Context, repo string, digest ociregistry.Digest) (ociregistry.Descriptor, error) {
	return c.resolve(ctx, &ocirequest.Request{
		Kind:   ocirequest.ReqBlobHead,
		Repo:   repo,
		Digest: string(digest),
	})
}

func (c *client) ResolveManifest(ctx context.Context, repo string, digest ociregistry.Digest) (ociregistry.Descriptor, error) {
	return c.resolve(ctx, &ocirequest.Request{
		Kind:   ocirequest.ReqManifestHead,
		Repo:   repo,
		Digest: string(digest),
	})
}

func (c *client) ResolveTag(ctx context.Context, repo string, tag string) (ociregistry.Descriptor, error) {
	return c.resolve(ctx, &ocirequest.Request{
		Kind: ocirequest.ReqManifestHead,
		Repo: repo,
		Tag:  tag,
	})
}

func (c *client) resolve(ctx context.Context, rreq *ocirequest.Request) (ociregistry.Descriptor, error) {
	resp, err := c.doRequest(ctx, rreq)
	if err != nil {
		return ociregistry.Descriptor{}, err
	}
	resp.Body.Close()
	desc, err := descriptorFromResponse(resp, ociregistry.Digest(rreq.Digest), requireSize|requireDigest)
	if err != nil {
		return ociregistry.Descriptor{}, fmt.Errorf("invalid descriptor in response: %v", err)
	}
	return desc, nil
}

func (c *client) GetManifest(ctx context.Context, repo string, digest ociregistry.Digest) (ociregistry.BlobReader, error) {
	return c.read(ctx, &ocirequest.Request{
		Kind:   ocirequest.ReqManifestGet,
		Repo:   repo,
		Digest: string(digest),
	})
}

func (c *client) GetTag(ctx context.Context, repo string, tagName string) (ociregistry.BlobReader, error) {
	return c.read(ctx, &ocirequest.Request{
		Kind: ocirequest.ReqManifestGet,
		Repo: repo,
		Tag:  tagName,
	})
}

// inMemThreshold holds the maximum number of bytes of manifest content
// that we'll hold in memory to obtain a digest before falling back do
// doing a HEAD request.
//
// This is hopefully large enough to be considerably larger than most
// manifests but small enough to fit comfortably into RAM on most
// platforms.
//
// Note: this is only used when talking to registries that fail to return
// a digest when doing a GET on a tag.
const inMemThreshold = 128 * 1024

func (c *client) read(ctx context.Context, rreq *ocirequest.Request) (_ ociregistry.BlobReader, _err error) {
	resp, err := c.doRequest(ctx, rreq)
	if err != nil {
		return nil, err
	}
	defer closeOnError(&_err, resp.Body)
	desc, err := descriptorFromResponse(resp, ociregistry.Digest(rreq.Digest), requireSize)
	if err != nil {
		return nil, fmt.Errorf("invalid descriptor in response: %v", err)
	}
	if desc.Digest == "" {
		// Returning a digest isn't mandatory according to the spec, and
		// at least one registry (AWS's ECR) fails to return a digest
		// when doing a GET of a tag.
		// We know the request must be a tag-getting
		// request because all other requests take a digest not a tag
		// but sanity check anyway.
		if rreq.Kind != ocirequest.ReqManifestGet {
			return nil, fmt.Errorf("internal error: no digest available for non-tag request")
		}

		// If the manifest is of a reasonable size, just read it into memory
		// and calculate the digest that way, otherwise issue a HEAD
		// request which should hopefully (and does in the ECR case)
		// give us the digest we need.
		if desc.Size <= inMemThreshold {
			data, err := io.ReadAll(io.LimitReader(resp.Body, desc.Size+1))
			if err != nil {
				return nil, fmt.Errorf("failed to read body to determine digest: %v", err)
			}
			if int64(len(data)) != desc.Size {
				return nil, fmt.Errorf("body size mismatch")
			}
			desc.Digest = digest.FromBytes(data)
			resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewReader(data))
		} else {
			rreq1 := rreq
			rreq1.Kind = ocirequest.ReqManifestHead
			resp1, err := c.doRequest(ctx, rreq1)
			if err != nil {
				return nil, err
			}
			resp1.Body.Close()
			desc, err = descriptorFromResponse(resp1, ociregistry.Digest(rreq1.Digest), requireSize|requireDigest)
			if err != nil {
				return nil, err
			}
		}
	}
	return newBlobReader(resp.Body, desc), nil
}
