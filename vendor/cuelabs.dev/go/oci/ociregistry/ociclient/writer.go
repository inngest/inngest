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
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/opencontainers/go-digest"

	"cuelabs.dev/go/oci/ociregistry"
	"cuelabs.dev/go/oci/ociregistry/internal/ocirequest"
	"cuelabs.dev/go/oci/ociregistry/ociauth"
)

// This file implements the ociregistry.Writer methods.

func (c *client) PushManifest(ctx context.Context, repo string, tag string, contents []byte, mediaType string) (ociregistry.Descriptor, error) {
	if mediaType == "" {
		return ociregistry.Descriptor{}, fmt.Errorf("PushManifest called with empty mediaType")
	}
	desc := ociregistry.Descriptor{
		Digest:    digest.FromBytes(contents),
		Size:      int64(len(contents)),
		MediaType: mediaType,
	}

	rreq := &ocirequest.Request{
		Kind:   ocirequest.ReqManifestPut,
		Repo:   repo,
		Tag:    tag,
		Digest: string(desc.Digest),
	}
	req, err := newRequest(ctx, rreq, bytes.NewReader(contents))
	if err != nil {
		return ociregistry.Descriptor{}, err
	}
	req.Header.Set("Content-Type", mediaType)
	req.ContentLength = desc.Size
	resp, err := c.do(req, http.StatusCreated)
	if err != nil {
		return ociregistry.Descriptor{}, err
	}
	resp.Body.Close()
	return desc, nil
}

func (c *client) MountBlob(ctx context.Context, fromRepo, toRepo string, dig ociregistry.Digest) (ociregistry.Descriptor, error) {
	rreq := &ocirequest.Request{
		Kind:     ocirequest.ReqBlobMount,
		Repo:     toRepo,
		FromRepo: fromRepo,
		Digest:   string(dig),
	}
	resp, err := c.doRequest(ctx, rreq, http.StatusCreated, http.StatusAccepted)
	if err != nil {
		return ociregistry.Descriptor{}, err
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusAccepted {
		// Mount isn't supported and technically the upload session has begun,
		// but we aren't in a great position to be able to continue it, so let's just
		// return Unsupported.
		return ociregistry.Descriptor{}, fmt.Errorf("registry does not support mounts: %w", ociregistry.ErrUnsupported)
	}
	// TODO: is it OK to omit the size from the returned descriptor here?
	return descriptorFromResponse(resp, dig, requireDigest)
}

func (c *client) PushBlob(ctx context.Context, repo string, desc ociregistry.Descriptor, r io.Reader) (_ ociregistry.Descriptor, _err error) {
	// TODO use the single-post blob-upload method (ReqBlobUploadBlob)
	// See:
	//	https://github.com/distribution/distribution/issues/4065
	//	https://github.com/golang/go/issues/63152
	rreq := &ocirequest.Request{
		Kind: ocirequest.ReqBlobStartUpload,
		Repo: repo,
	}
	req, err := newRequest(ctx, rreq, nil)
	if err != nil {
		return ociregistry.Descriptor{}, err
	}
	resp, err := c.do(req, http.StatusAccepted)
	if err != nil {
		return ociregistry.Descriptor{}, err
	}
	resp.Body.Close()
	location, err := locationFromResponse(resp)
	if err != nil {
		return ociregistry.Descriptor{}, err
	}

	// We've got the upload location. Now PUT the content.

	ctx = ociauth.ContextWithRequestInfo(ctx, ociauth.RequestInfo{
		RequiredScope: scopeForRequest(rreq),
	})
	// Note: we can't use ocirequest.Request here because that's
	// specific to the ociserver implementation in this case.
	req, err = http.NewRequestWithContext(ctx, "PUT", "", r)
	if err != nil {
		return ociregistry.Descriptor{}, err
	}
	req.URL = urlWithDigest(location, string(desc.Digest))
	req.ContentLength = desc.Size
	req.Header.Set("Content-Type", "application/octet-stream")
	// TODO: per the spec, the content-range header here is unnecessary.
	req.Header.Set("Content-Range", ocirequest.RangeString(0, desc.Size))
	resp, err = c.do(req, http.StatusCreated)
	if err != nil {
		return ociregistry.Descriptor{}, err
	}
	defer closeOnError(&_err, resp.Body)
	resp.Body.Close()
	return desc, nil
}

// TODO is this a reasonable default? We have to
// weigh up in-memory cost vs round-trip overhead.
// TODO: make this default configurable.
const defaultChunkSize = 64 * 1024

func (c *client) PushBlobChunked(ctx context.Context, repo string, chunkSize int) (ociregistry.BlobWriter, error) {
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}
	resp, err := c.doRequest(ctx, &ocirequest.Request{
		Kind: ocirequest.ReqBlobStartUpload,
		Repo: repo,
	}, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	location, err := locationFromResponse(resp)
	if err != nil {
		return nil, err
	}
	ctx = ociauth.ContextWithRequestInfo(ctx, ociauth.RequestInfo{
		RequiredScope: ociauth.NewScope(ociauth.ResourceScope{
			ResourceType: "repository",
			Resource:     repo,
			Action:       "push",
		}),
	})
	return &blobWriter{
		ctx:       ctx,
		client:    c,
		chunkSize: chunkSizeFromResponse(resp, chunkSize),
		chunk:     make([]byte, 0, chunkSize),
		location:  location,
	}, nil
}

func (c *client) PushBlobChunkedResume(ctx context.Context, repo string, id string, offset int64, chunkSize int) (ociregistry.BlobWriter, error) {
	if id == "" {
		return nil, fmt.Errorf("id must be non-empty to resume a chunked upload")
	}
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}
	var location *url.URL
	switch {
	case offset == -1:
		// Try to find what offset we're meant to be writing at
		// by doing a GET to the location.
		// TODO does resuming an upload require push or pull scope or both?
		ctx := ociauth.ContextWithRequestInfo(ctx, ociauth.RequestInfo{
			RequiredScope: ociauth.NewScope(ociauth.ResourceScope{
				ResourceType: "repository",
				Resource:     repo,
				Action:       "push",
			}, ociauth.ResourceScope{
				ResourceType: "repository",
				Resource:     repo,
				Action:       "pull",
			}),
		})
		req, err := http.NewRequestWithContext(ctx, "GET", id, nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.do(req, http.StatusNoContent)
		if err != nil {
			return nil, fmt.Errorf("cannot recover chunk offset: %v", err)
		}
		location, err = locationFromResponse(resp)
		if err != nil {
			return nil, fmt.Errorf("cannot get location from response: %v", err)
		}
		rangeStr := resp.Header.Get("Range")
		p0, p1, ok := ocirequest.ParseRange(rangeStr)
		if !ok {
			return nil, fmt.Errorf("invalid range %q in response", rangeStr)
		}
		if p0 != 0 {
			return nil, fmt.Errorf("range %q does not start with 0", rangeStr)
		}
		chunkSize = chunkSizeFromResponse(resp, chunkSize)
		offset = p1
	case offset < 0:
		return nil, fmt.Errorf("invalid offset; must be -1 or non-negative")
	default:
		var err error
		location, err = url.Parse(id) // Note that this mirrors [BlobWriter.ID].
		if err != nil {
			return nil, fmt.Errorf("provided ID is not a valid location URL")
		}
		if !strings.HasPrefix(location.Path, "/") {
			// Our BlobWriter.ID method always returns a fully
			// qualified absolute URL, so this must be a mistake
			// on the part of the caller.
			// We allow a relative URL even though we don't
			// ever return one to make things a bit easier for tests.
			return nil, fmt.Errorf("provided upload ID %q has unexpected relative URL path", id)
		}
	}
	ctx = ociauth.ContextWithRequestInfo(ctx, ociauth.RequestInfo{
		RequiredScope: ociauth.NewScope(ociauth.ResourceScope{
			ResourceType: "repository",
			Resource:     repo,
			Action:       "push",
		}),
	})
	return &blobWriter{
		ctx:       ctx,
		client:    c,
		chunkSize: chunkSize,
		size:      offset,
		flushed:   offset,
		location:  location,
	}, nil
}

type blobWriter struct {
	client    *client
	chunkSize int
	ctx       context.Context

	// mu guards the fields below it.
	mu       sync.Mutex
	closed   bool
	chunk    []byte
	closeErr error

	// size holds the size of the entire upload as seen from the
	// client perspective. Each call to Write increases this immediately.
	size int64

	// flushed holds the size of the upload as flushed to the server.
	// Each successfully flushed chunk increases this.
	flushed  int64
	location *url.URL
}

func (w *blobWriter) Write(buf []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// We use > rather than >= here so that using a chunk size of 100
	// and writing 100 bytes does not actually flush, which would result in a PATCH
	// then followed by an empty-bodied PUT with the call to Commit.
	// Instead, we want the writes to not flush at all, and Commit to PUT the entire chunk.
	if len(w.chunk)+len(buf) > w.chunkSize {
		if err := w.flush(buf, ""); err != nil {
			return 0, err
		}
	} else {
		if w.chunk == nil {
			w.chunk = make([]byte, 0, w.chunkSize)
		}
		w.chunk = append(w.chunk, buf...)
	}
	w.size += int64(len(buf))
	return len(buf), nil
}

// flush flushes any outstanding upload data to the server.
// If commitDigest is non-empty, this is the final segment of data in the blob:
// the blob is being committed and the digest should hold the digest of the entire blob content.
func (w *blobWriter) flush(buf []byte, commitDigest ociregistry.Digest) error {
	if commitDigest == "" && len(buf)+len(w.chunk) == 0 {
		return nil
	}
	// Start a new PATCH request to send the currently outstanding data.
	method := "PATCH"
	expect := http.StatusAccepted
	reqURL := w.location
	if commitDigest != "" {
		// This is the final piece of data, so send it as the final PUT request
		// (committing the whole blob) which avoids an extra round trip.
		method = "PUT"
		expect = http.StatusCreated
		reqURL = urlWithDigest(reqURL, string(commitDigest))
	}
	req, err := http.NewRequestWithContext(w.ctx, method, "", concatBody(w.chunk, buf))
	if err != nil {
		return fmt.Errorf("cannot make PATCH request: %v", err)
	}
	req.URL = reqURL
	req.ContentLength = int64(len(w.chunk) + len(buf))
	// TODO: per the spec, the content-range header here is unnecessary
	// if we are doing a final PUT without a body.
	req.Header.Set("Content-Range", ocirequest.RangeString(w.flushed, w.flushed+req.ContentLength))
	resp, err := w.client.do(req, expect)
	if err != nil {
		return err
	}
	resp.Body.Close()
	location, err := locationFromResponse(resp)
	if err != nil {
		return fmt.Errorf("bad Location in response: %v", err)
	}
	// TODO is there something we could be doing with the Range header in the response?
	w.location = location
	w.flushed += req.ContentLength
	w.chunk = w.chunk[:0]
	return nil
}

func concatBody(b1, b2 []byte) io.Reader {
	if len(b1)+len(b2) == 0 {
		return nil // note that net/http treats a nil request body differently
	}
	if len(b1) == 0 {
		return bytes.NewReader(b2)
	}
	if len(b2) == 0 {
		return bytes.NewReader(b1)
	}
	return io.MultiReader(
		bytes.NewReader(b1),
		bytes.NewReader(b2),
	)
}

func (w *blobWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return w.closeErr
	}
	err := w.flush(nil, "")
	w.closed = true
	w.closeErr = err
	return err
}

func (w *blobWriter) Size() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.size
}

func (w *blobWriter) ChunkSize() int {
	return w.chunkSize
}

func (w *blobWriter) ID() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.location.String()
}

func (w *blobWriter) Commit(digest ociregistry.Digest) (ociregistry.Descriptor, error) {
	if digest == "" {
		return ociregistry.Descriptor{}, fmt.Errorf("cannot commit with an empty digest")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.flush(nil, digest); err != nil {
		return ociregistry.Descriptor{}, fmt.Errorf("cannot flush data before commit: %w", err)
	}
	return ociregistry.Descriptor{
		MediaType: "application/octet-stream",
		Size:      w.size,
		Digest:    digest,
	}, nil
}

func (w *blobWriter) Cancel() error {
	return nil
}

// urlWithDigest returns u with the digest query parameter set, taking care not
// to disrupt the initial URL (thus avoiding the charge of "manually
// assembing the location; see [here].
//
// [here]: https://github.com/opencontainers/distribution-spec/blob/main/spec.md#post-then-put
func urlWithDigest(u0 *url.URL, digest string) *url.URL {
	u := *u0
	digest = url.QueryEscape(digest)
	switch {
	case u.ForceQuery:
		// The URL already ended in a "?" with no actual query parameters.
		u.RawQuery = "digest=" + digest
		u.ForceQuery = false
	case u.RawQuery != "":
		// There's already a query parameter present.
		u.RawQuery += "&digest=" + digest
	default:
		u.RawQuery = "digest=" + digest
	}
	return &u
}

// See https://github.com/opencontainers/distribution-spec/blob/main/spec.md#pushing-a-blob-in-chunks
func chunkSizeFromResponse(resp *http.Response, chunkSize int) int {
	minChunkSize, err := strconv.Atoi(resp.Header.Get("OCI-Chunk-Min-Length"))
	if err == nil && minChunkSize > chunkSize {
		return minChunkSize
	}
	return chunkSize
}
