// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package fileuri

import (
	"fmt"
	"net/url"
	pathpkg "path"
	"path/filepath"
	"strings"
)

// FileURI is a parsed local file URI. It keeps the original URL form so path
// joining can preserve hierarchical and opaque file URI representations.
type FileURI struct {
	parsed url.URL
	path   string
}

// Parse parses hierarchical and opaque file URIs using the same lexical rules
// for every FileIO consumer.
func Parse(name string) (FileURI, error) {
	parsed, err := url.Parse(name)
	if err != nil {
		return FileURI{}, err
	}
	if !strings.EqualFold(parsed.Scheme, "file") {
		return FileURI{}, fmt.Errorf("unsupported file URI scheme %q", parsed.Scheme)
	}

	localPath := parsed.Path
	if parsed.Opaque != "" {
		localPath, err = url.PathUnescape(parsed.Opaque)
		if err != nil {
			return FileURI{}, err
		}
	}

	return FileURI{parsed: *parsed, path: localPath}, nil
}

// Host returns the URI authority without changing its spelling.
func (f FileURI) Host() string {
	return f.parsed.Host
}

// LocalPath returns the decoded local path. When windows is true, Windows drive
// authorities and /C:/ URI paths are converted to native drive-shaped paths.
func (f FileURI) LocalPath(windows bool) string {
	if !windows {
		return f.path
	}
	if IsWindowsDriveHost(f.parsed.Host) {
		return f.parsed.Host + f.path
	}
	if IsWindowsDriveURIPath(f.path) {
		return f.path[1:]
	}

	return f.path
}

// LocalPathForOS returns the decoded local path using the running host's file
// system semantics.
func (f FileURI) LocalPathForOS() string {
	return f.LocalPath(filepath.Separator == '\\')
}

// JoinPath appends path elements while retaining the URI's hierarchical or
// opaque representation.
func (f FileURI) JoinPath(elem ...string) string {
	joined := f.parsed
	if joined.Opaque != "" {
		segments := append([]string{joined.Opaque}, elem...)
		joined.Opaque = pathpkg.Join(segments...)

		return joined.String()
	}

	return joined.JoinPath(elem...).String()
}

// HasWindowsDrivePrefix reports whether path begins with a Windows drive
// designator such as C: independently of the host operating system.
func HasWindowsDrivePrefix(path string) bool {
	return len(path) >= 2 && isDriveLetter(path[0]) && path[1] == ':'
}

// IsWindowsDriveURIPath reports whether path uses the /C:/ form produced by a
// hierarchical Windows file URI.
func IsWindowsDriveURIPath(path string) bool {
	return len(path) >= 3 && path[0] == '/' && isDriveLetter(path[1]) && path[2] == ':'
}

// IsWindowsDriveHost reports whether a file URI authority is a drive such as C:.
func IsWindowsDriveHost(host string) bool {
	return len(host) == 2 && isDriveLetter(host[0]) && host[1] == ':'
}

func isDriveLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
