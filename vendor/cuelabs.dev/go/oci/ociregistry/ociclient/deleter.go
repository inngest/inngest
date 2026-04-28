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
	"net/http"

	"cuelabs.dev/go/oci/ociregistry"
	"cuelabs.dev/go/oci/ociregistry/internal/ocirequest"
)

func (c *client) DeleteBlob(ctx context.Context, repoName string, digest ociregistry.Digest) error {
	return c.delete(ctx, &ocirequest.Request{
		Kind:   ocirequest.ReqBlobDelete,
		Repo:   repoName,
		Digest: string(digest),
	})
}

func (c *client) DeleteManifest(ctx context.Context, repoName string, digest ociregistry.Digest) error {
	return c.delete(ctx, &ocirequest.Request{
		Kind:   ocirequest.ReqManifestDelete,
		Repo:   repoName,
		Digest: string(digest),
	})
}

func (c *client) DeleteTag(ctx context.Context, repoName string, tagName string) error {
	return c.delete(ctx, &ocirequest.Request{
		Kind: ocirequest.ReqManifestDelete,
		Repo: repoName,
		Tag:  tagName,
	})
}

func (c *client) delete(ctx context.Context, rreq *ocirequest.Request) error {
	resp, err := c.doRequest(ctx, rreq, http.StatusAccepted)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
