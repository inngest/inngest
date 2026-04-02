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
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"cuelabs.dev/go/oci/ociregistry"
	"cuelabs.dev/go/oci/ociregistry/ociref"
)

var (
	errBadlyFormedDigest = ociregistry.NewError("badly formed digest", ociregistry.ErrDigestInvalid.Code(), nil)
	errMethodNotAllowed  = httpErrorf(http.StatusMethodNotAllowed, "method not allowed")
	errNotFound          = httpErrorf(http.StatusNotFound, "page not found")
)

func badRequestf(f string, a ...any) error {
	return httpErrorf(http.StatusBadRequest, f, a...)
}

func httpErrorf(statusCode int, f string, a ...any) error {
	return ociregistry.NewHTTPError(fmt.Errorf(f, a...), statusCode, nil, nil)
}

type Request struct {
	Kind Kind

	// Repo holds the repository name. Valid for all request kinds
	// except ReqCatalogList and ReqPing.
	Repo string

	// Digest holds the digest being used in the request.
	// Valid for:
	//	ReqBlobMount
	//	ReqBlobUploadBlob
	//	ReqBlobGet
	//	ReqBlobHead
	//	ReqBlobDelete
	//	ReqBlobCompleteUpload
	//	ReqReferrersList
	//
	// Valid for these manifest requests when they're referring to a digest
	// rather than a tag:
	//	ReqManifestGet
	//	ReqManifestHead
	//	ReqManifestPut
	//	ReqManifestDelete
	Digest string

	// Tag holds the tag being used in the request. Valid for
	// these manifest requests when they're referring to a tag:
	//	ReqManifestGet
	//	ReqManifestHead
	//	ReqManifestPut
	//	ReqManifestDelete
	Tag string

	// FromRepo holds the repository name to mount from
	// for ReqBlobMount.
	FromRepo string

	// UploadID holds the upload identifier as used for
	// chunked uploads.
	// Valid for:
	//	ReqBlobUploadInfo
	//	ReqBlobUploadChunk
	UploadID string

	// ListN holds the maximum count for listing.
	// It's -1 to specify that all items should be returned.
	//
	// Valid for:
	//	ReqTagsList
	//	ReqCatalog
	ListN int

	// ListLast holds the item to start just after
	// when listing.
	//
	// Valid for:
	//	ReqTagsList
	//	ReqCatalog
	ListLast string

	// ArtifactType holds the artifact type to filter by when
	// listing.
	//
	// Valid for:
	//	ReqReferrersList
	ArtifactType string
}

type Kind int

const (
	// end-1	GET	/v2/	200	404/401
	ReqPing = Kind(iota)

	// Blob-related endpoints

	// end-2	GET	/v2/<name>/blobs/<digest>	200	404
	ReqBlobGet

	// end-2	HEAD	/v2/<name>/blobs/<digest>	200	404
	ReqBlobHead

	// end-10	DELETE	/v2/<name>/blobs/<digest>	202	404/405
	ReqBlobDelete

	// end-4a	POST	/v2/<name>/blobs/uploads/	202	404
	ReqBlobStartUpload

	// end-4b	POST	/v2/<name>/blobs/uploads/?digest=<digest>	201/202	404/400
	ReqBlobUploadBlob

	// end-11	POST	/v2/<name>/blobs/uploads/?mount=<digest>&from=<other_name>	201	404
	ReqBlobMount

	// end-13	GET	/v2/<name>/blobs/uploads/<reference>	204	404
	// NOTE: despite being described in the distribution spec, this
	// isn't really part of the OCI spec.
	ReqBlobUploadInfo

	// end-5	PATCH	/v2/<name>/blobs/uploads/<reference>	202	404/416
	// NOTE: despite being described in the distribution spec, this
	// isn't really part of the OCI spec.
	ReqBlobUploadChunk

	// end-6	PUT	/v2/<name>/blobs/uploads/<reference>?digest=<digest>	201	404/400
	// NOTE: despite being described in the distribution spec, this
	// isn't really part of the OCI spec.
	ReqBlobCompleteUpload

	// Manifest-related endpoints

	// end-3	GET	/v2/<name>/manifests/<tagOrDigest>	200	404
	ReqManifestGet

	// end-3	HEAD	/v2/<name>/manifests/<tagOrDigest>	200	404
	ReqManifestHead

	// end-7	PUT	/v2/<name>/manifests/<tagOrDigest>	201	404
	ReqManifestPut

	// end-9	DELETE	/v2/<name>/manifests/<tagOrDigest>	202	404/400/405
	ReqManifestDelete

	// Tag-related endpoints

	// end-8a	GET	/v2/<name>/tags/list	200	404
	// end-8b	GET	/v2/<name>/tags/list?n=<integer>&last=<integer>	200	404
	ReqTagsList

	// Referrer-related endpoints

	// end-12a	GET	/v2/<name>/referrers/<digest>	200	404/400
	ReqReferrersList

	// Catalog endpoints (out-of-spec)
	// 	GET	/v2/_catalog
	ReqCatalogList
)

// Parse parses the given HTTP method and URL as an OCI registry request.
// It understands the endpoints described in the [distribution spec].
//
// If it returns an error, it will be of type [ociregistry.Error] or [ociregistry.HTTPError].
//
// [distribution spec]: https://github.com/opencontainers/distribution-spec/blob/main/spec.md#endpoints
func Parse(method string, u *url.URL) (*Request, error) {
	path := u.Path
	urlq, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, badRequestf("invalid query parameters: %v", err)
	}

	var rreq Request
	if path == "/v2" || path == "/v2/" {
		rreq.Kind = ReqPing
		return &rreq, nil
	}
	path, ok := strings.CutPrefix(path, "/v2/")
	if !ok {
		return nil, ociregistry.NewError("unknown URL path", ociregistry.ErrNameUnknown.Code(), nil)
	}
	if path == "_catalog" {
		if method != "GET" {
			return nil, errMethodNotAllowed
		}
		rreq.Kind = ReqCatalogList
		setListQueryParams(&rreq, urlq)
		return &rreq, nil
	}
	uploadPath, ok := strings.CutSuffix(path, "/blobs/uploads/")
	if !ok {
		uploadPath, ok = strings.CutSuffix(path, "/blobs/uploads")
	}
	if ok {
		rreq.Repo = uploadPath
		if !ociref.IsValidRepository(rreq.Repo) {
			return nil, ociregistry.ErrNameInvalid
		}
		if method != "POST" {
			return nil, errMethodNotAllowed
		}
		if d := urlq.Get("mount"); d != "" {
			// end-11
			rreq.Digest = d
			if !ociref.IsValidDigest(rreq.Digest) {
				return nil, ociregistry.ErrDigestInvalid
			}
			rreq.FromRepo = urlq.Get("from")
			if rreq.FromRepo == "" {
				// There's no "from" argument so fall back to
				// a regular chunked upload.
				rreq.Kind = ReqBlobStartUpload
				// TODO does the "mount" query argument actually take effect in some way?
				rreq.Digest = ""
				return &rreq, nil
			}
			if !ociref.IsValidRepository(rreq.FromRepo) {
				return nil, ociregistry.ErrNameInvalid
			}
			rreq.Kind = ReqBlobMount
			return &rreq, nil
		}
		if d := urlq.Get("digest"); d != "" {
			// end-4b
			rreq.Digest = d
			if !ociref.IsValidDigest(d) {
				return nil, errBadlyFormedDigest
			}
			rreq.Kind = ReqBlobUploadBlob
			return &rreq, nil
		}
		// end-4a
		rreq.Kind = ReqBlobStartUpload
		return &rreq, nil
	}
	path, last, ok := cutLast(path, "/")
	if !ok {
		return nil, errNotFound
	}
	path, lastButOne, ok := cutLast(path, "/")
	if !ok {
		return nil, errNotFound
	}
	switch lastButOne {
	case "blobs":
		rreq.Repo = path
		if !ociref.IsValidDigest(last) {
			return nil, errBadlyFormedDigest
		}
		if !ociref.IsValidRepository(rreq.Repo) {
			return nil, ociregistry.ErrNameInvalid
		}
		rreq.Digest = last
		switch method {
		case "GET":
			rreq.Kind = ReqBlobGet
		case "HEAD":
			rreq.Kind = ReqBlobHead
		case "DELETE":
			rreq.Kind = ReqBlobDelete
		default:
			return nil, errMethodNotAllowed
		}
		return &rreq, nil
	case "uploads":
		// Note: this section is all specific to ociserver and
		// isn't part of the OCI registry spec.
		repo, ok := strings.CutSuffix(path, "/blobs")
		if !ok {
			return nil, errNotFound
		}
		rreq.Repo = repo
		if !ociref.IsValidRepository(rreq.Repo) {
			return nil, ociregistry.ErrNameInvalid
		}
		uploadID64 := last
		if uploadID64 == "" {
			return nil, errNotFound
		}
		uploadID, err := base64.RawURLEncoding.DecodeString(uploadID64)
		if err != nil {
			return nil, badRequestf("invalid upload ID %q (cannot decode)", uploadID64)
		}
		if !utf8.Valid(uploadID) {
			return nil, badRequestf("upload ID %q decoded to invalid utf8", uploadID64)
		}
		rreq.UploadID = string(uploadID)

		switch method {
		case "GET":
			rreq.Kind = ReqBlobUploadInfo
		case "PATCH":
			rreq.Kind = ReqBlobUploadChunk
		case "PUT":
			rreq.Kind = ReqBlobCompleteUpload
			rreq.Digest = urlq.Get("digest")
			if !ociref.IsValidDigest(rreq.Digest) {
				return nil, errBadlyFormedDigest
			}
		default:
			return nil, errMethodNotAllowed
		}
		return &rreq, nil
	case "manifests":
		rreq.Repo = path
		if !ociref.IsValidRepository(rreq.Repo) {
			return nil, ociregistry.ErrNameInvalid
		}
		switch {
		case ociref.IsValidDigest(last):
			rreq.Digest = last
		case ociref.IsValidTag(last):
			rreq.Tag = last
		default:
			return nil, errNotFound
		}
		switch method {
		case "GET":
			rreq.Kind = ReqManifestGet
		case "HEAD":
			rreq.Kind = ReqManifestHead
		case "PUT":
			rreq.Kind = ReqManifestPut
		case "DELETE":
			rreq.Kind = ReqManifestDelete
		default:
			return nil, errMethodNotAllowed
		}
		return &rreq, nil

	case "tags":
		if last != "list" {
			return nil, errNotFound
		}
		if err := setListQueryParams(&rreq, urlq); err != nil {
			return nil, err
		}
		if method != "GET" {
			return nil, errMethodNotAllowed
		}
		rreq.Repo = path
		if !ociref.IsValidRepository(rreq.Repo) {
			return nil, ociregistry.ErrNameInvalid
		}
		rreq.Kind = ReqTagsList
		return &rreq, nil
	case "referrers":
		if !ociref.IsValidDigest(last) {
			return nil, errBadlyFormedDigest
		}
		if method != "GET" {
			return nil, errMethodNotAllowed
		}
		rreq.Repo = path
		if !ociref.IsValidRepository(rreq.Repo) {
			return nil, ociregistry.ErrNameInvalid
		}
		// Unlike other list-oriented endpoints, there appears to be no defined way for the client
		// to indicate the desired number of results, but set ListN anyway to be future-proof.
		rreq.ListN = -1
		rreq.Digest = last
		rreq.ArtifactType = urlq.Get("artifactType")
		rreq.Kind = ReqReferrersList
		return &rreq, nil
	}
	return nil, errNotFound
}

func setListQueryParams(rreq *Request, urlq url.Values) error {
	rreq.ListN = -1
	if nstr := urlq.Get("n"); nstr != "" {
		n, err := strconv.Atoi(nstr)
		if err != nil {
			return badRequestf("query parameter n is not a valid integer")
		}
		rreq.ListN = n
	}
	rreq.ListLast = urlq.Get("last")
	return nil
}

func cutLast(s, sep string) (before, after string, found bool) {
	if i := strings.LastIndex(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return "", s, false
}

// ParseRange extracts the start and end offsets from a Content-Range string.
// The resulting start is inclusive and the end exclusive, to match Go convention,
// whereas Content-Range is inclusive on both ends.
func ParseRange(s string) (start, end int64, ok bool) {
	p0s, p1s, ok := strings.Cut(s, "-")
	if !ok {
		return 0, 0, false
	}
	p0, err0 := strconv.ParseInt(p0s, 10, 64)
	p1, err1 := strconv.ParseInt(p1s, 10, 64)
	if p1 > 0 {
		p1++
	}
	return p0, p1, err0 == nil && err1 == nil
}

// RangeString formats a pair of start and end offsets in the Content-Range form.
// The input start is inclusive and the end exclusive, to match Go convention,
// whereas Content-Range is inclusive on both ends.
func RangeString(start, end int64) string {
	end--
	if end < 0 {
		end = 0
	}
	return fmt.Sprintf("%d-%d", start, end)
}
