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

package ocirequest

import (
	"encoding/base64"
	"fmt"
	"net/url"
)

func (req *Request) Construct() (method string, ustr string, err error) {
	method, ustr = req.construct()
	u, err := url.Parse(ustr)
	if err != nil {
		return "", "", fmt.Errorf("invalid OCI request: %v", err)
	}
	if _, err := Parse(method, u); err != nil {
		return "", "", fmt.Errorf("invalid OCI request: %v", err)
	}
	return method, ustr, nil
}

func (req *Request) MustConstruct() (method string, ustr string) {
	method, ustr, err := req.Construct()
	if err != nil {
		panic(err)
	}
	return method, ustr
}

func (req *Request) construct() (method string, urlStr string) {
	switch req.Kind {
	case ReqPing:
		return "GET", "/v2/"
	case ReqBlobGet:
		return "GET", "/v2/" + req.Repo + "/blobs/" + req.Digest
	case ReqBlobHead:
		return "HEAD", "/v2/" + req.Repo + "/blobs/" + req.Digest
	case ReqBlobDelete:
		return "DELETE", "/v2/" + req.Repo + "/blobs/" + req.Digest
	case ReqBlobStartUpload:
		return "POST", "/v2/" + req.Repo + "/blobs/uploads/"
	case ReqBlobUploadBlob:
		return "POST", "/v2/" + req.Repo + "/blobs/uploads/?digest=" + req.Digest
	case ReqBlobMount:
		return "POST", "/v2/" + req.Repo + "/blobs/uploads/?mount=" + req.Digest + "&from=" + req.FromRepo
	case ReqBlobUploadInfo:
		// Note: this is specific to the ociserver implementation.
		return "GET", req.uploadPath()
	case ReqBlobUploadChunk:
		// Note: this is specific to the ociserver implementation.
		return "PATCH", req.uploadPath()
	case ReqBlobCompleteUpload:
		// Note: this is specific to the ociserver implementation.
		// TODO this is bogus when the upload ID contains query parameters.
		return "PUT", req.uploadPath() + "?digest=" + req.Digest
	case ReqManifestGet:
		return "GET", "/v2/" + req.Repo + "/manifests/" + req.tagOrDigest()
	case ReqManifestHead:
		return "HEAD", "/v2/" + req.Repo + "/manifests/" + req.tagOrDigest()
	case ReqManifestPut:
		return "PUT", "/v2/" + req.Repo + "/manifests/" + req.tagOrDigest()
	case ReqManifestDelete:
		return "DELETE", "/v2/" + req.Repo + "/manifests/" + req.tagOrDigest()
	case ReqTagsList:
		return "GET", "/v2/" + req.Repo + "/tags/list" + req.listParams()
	case ReqReferrersList:
		p := "/v2/" + req.Repo + "/referrers/" + req.Digest
		if req.ArtifactType != "" {
			p += "?" + url.Values{"artifactType": {req.ArtifactType}}.Encode()
		}
		return "GET", p
	case ReqCatalogList:
		return "GET", "/v2/_catalog" + req.listParams()
	default:
		panic("invalid request kind")
	}
}

func (req *Request) uploadPath() string {
	return "/v2/" + req.Repo + "/blobs/uploads/" + base64.RawURLEncoding.EncodeToString([]byte(req.UploadID))
}

func (req *Request) listParams() string {
	q := make(url.Values)
	if req.ListN >= 0 {
		q.Set("n", fmt.Sprint(req.ListN))
	}
	if req.ListLast != "" {
		q.Set("last", req.ListLast)
	}
	if len(q) > 0 {
		return "?" + q.Encode()
	}
	return ""
}

func (req *Request) tagOrDigest() string {
	if req.Tag != "" {
		return req.Tag
	}
	return req.Digest
}
