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
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"cuelabs.dev/go/oci/ociregistry"
)

// errorBodySizeLimit holds the maximum number of response bytes aallowed in
// the server's error response. A typical error message is around 200
// bytes. Hence, 8 KiB should be sufficient.
const errorBodySizeLimit = 8 * 1024

// makeError forms an error from a non-OK response.
//
// It reads but does not close resp.Body.
func makeError(resp *http.Response) error {
	var data []byte
	var err error
	if resp.Body != nil {
		data, err = io.ReadAll(io.LimitReader(resp.Body, errorBodySizeLimit+1))
		if err != nil {
			err = fmt.Errorf("cannot read error body: %v", err)
		} else if len(data) > errorBodySizeLimit {
			err = fmt.Errorf("error body too large")
		} else {
			err = makeError1(resp, data)
		}
	}
	// We always include the status code and response in the error.
	return ociregistry.NewHTTPError(err, resp.StatusCode, resp, data)
}

func makeError1(resp *http.Response, bodyData []byte) error {
	if resp.Request.Method == "HEAD" {
		// When we've made a HEAD request, we can't see any of
		// the actual error, so we'll have to make up something
		// from the HTTP status.
		// TODO would we do better if we interpreted the HTTP status
		// relative to the actual method that was called in order
		// to come up with a more plausible error?
		var err error
		switch resp.StatusCode {
		case http.StatusNotFound:
			err = ociregistry.ErrNameUnknown
		case http.StatusUnauthorized:
			err = ociregistry.ErrUnauthorized
		case http.StatusForbidden:
			err = ociregistry.ErrDenied
		case http.StatusTooManyRequests:
			err = ociregistry.ErrTooManyRequests
		case http.StatusBadRequest:
			err = ociregistry.ErrUnsupported
		default:
			// Our caller will turn this into a non-nil error.
			return nil
		}
		return err
	}
	if ctype := resp.Header.Get("Content-Type"); !isJSONMediaType(ctype) {
		return fmt.Errorf("non-JSON error response %q; body %q", ctype, bodyData)
	}
	var errs ociregistry.WireErrors
	if err := json.Unmarshal(bodyData, &errs); err != nil {
		return fmt.Errorf("%s: malformed error response: %v", resp.Status, err)
	}
	if len(errs.Errors) == 0 {
		return fmt.Errorf("%s: no errors in body (probably a server issue)", resp.Status)
	}
	return &errs
}

// isJSONMediaType reports whether the content type implies
// that the content is JSON.
func isJSONMediaType(contentType string) bool {
	mediaType, _, _ := mime.ParseMediaType(contentType)
	m := strings.TrimPrefix(mediaType, "application/")
	if len(m) == len(mediaType) {
		return false
	}
	// Look for +json suffix. See https://tools.ietf.org/html/rfc6838#section-4.2.8
	// We recognize multiple suffixes too (e.g. application/something+json+other)
	// as that seems to be a possibility.
	for {
		i := strings.Index(m, "+")
		if i == -1 {
			return m == "json"
		}
		if m[0:i] == "json" {
			return true
		}
		m = m[i+1:]
	}
}
