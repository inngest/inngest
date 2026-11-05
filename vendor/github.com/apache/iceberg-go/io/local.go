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

package io

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/apache/iceberg-go/internal/fileuri"
)

// LocalFS is an implementation of IO that implements interaction with
// the local file system.
type LocalFS struct{}

func localPath(name string) (string, error) {
	if filepath.VolumeName(name) != "" {
		return name, nil
	}

	schemeEnd := strings.IndexByte(name, ':')
	firstSeparator := strings.IndexAny(name, `/\`)
	if schemeEnd < 0 || (firstSeparator >= 0 && firstSeparator < schemeEnd) {
		return name, nil
	}
	scheme := name[:schemeEnd]
	// A colon is valid in a native POSIX filename. Treat non-file names as
	// URIs only when they have an authority delimiter, preserving paths such as
	// "partition:2026/data.parquet" while still rejecting s3:// and similar
	// locations passed directly to LocalFS.
	if !strings.EqualFold(scheme, "file") && !strings.HasPrefix(name[schemeEnd+1:], "//") {
		return name, nil
	}
	if !strings.EqualFold(scheme, "file") {
		return "", fmt.Errorf("unsupported local filesystem scheme %q", scheme)
	}

	parsed, err := fileuri.Parse(name)
	if err != nil {
		return "", fmt.Errorf("invalid local file path %q: %w", name, err)
	}
	if parsed.Host() != "" && !strings.EqualFold(parsed.Host(), "localhost") {
		if filepath.Separator != '\\' || !fileuri.IsWindowsDriveHost(parsed.Host()) {
			return "", fmt.Errorf("unsupported file URI authority %q", parsed.Host())
		}
	}

	path := parsed.LocalPathForOS()

	return filepath.FromSlash(path), nil
}

func (LocalFS) Open(name string) (File, error) {
	path, err := localPath(name)
	if err != nil {
		return nil, err
	}

	return os.Open(path)
}

func (LocalFS) ReadFile(name string) ([]byte, error) {
	path, err := localPath(name)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(path)
}

func (LocalFS) Create(name string) (FileWriter, error) {
	filename, err := localPath(name)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return nil, err
	}

	return os.Create(filename)
}

func (LocalFS) WriteFile(name string, content []byte) error {
	filename, err := localPath(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}

	return os.WriteFile(filename, content, 0o644)
}

func (LocalFS) Remove(name string) error {
	path, err := localPath(name)
	if err != nil {
		return err
	}

	return os.Remove(path)
}

func (LocalFS) RemoveAll(name string) error {
	path, err := localPath(name)
	if err != nil {
		return err
	}

	return os.RemoveAll(path)
}

func (LocalFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	path, err := localPath(root)
	if err != nil {
		return err
	}

	return filepath.WalkDir(path, fn)
}

func (LocalFS) ReadDir(name string) ([]fs.DirEntry, error) {
	path, err := localPath(name)
	if err != nil {
		return nil, err
	}

	return os.ReadDir(path)
}

func (LocalFS) MkdirAll(name string) error {
	path, err := localPath(name)
	if err != nil {
		return err
	}

	return os.MkdirAll(path, 0o755)
}

func (LocalFS) Mkdir(name string) error {
	path, err := localPath(name)
	if err != nil {
		return err
	}

	return os.Mkdir(path, 0o755)
}

func (LocalFS) Stat(name string) (fs.FileInfo, error) {
	path, err := localPath(name)
	if err != nil {
		return nil, err
	}

	return os.Stat(path)
}

func (LocalFS) Rename(oldpath, newpath string) error {
	src, err := localPath(oldpath)
	if err != nil {
		return err
	}
	dst, err := localPath(newpath)
	if err != nil {
		return err
	}

	return os.Rename(src, dst)
}

func (LocalFS) RenameNoReplace(oldpath, newpath string) error {
	src, err := localPath(oldpath)
	if err != nil {
		return err
	}
	dst, err := localPath(newpath)
	if err != nil {
		return err
	}

	if err = os.Link(src, dst); err != nil {
		return err
	}

	// Publishing already succeeded once the hard link exists. Removing the
	// source temp file is best-effort cleanup.
	_ = os.Remove(src)

	return nil
}
