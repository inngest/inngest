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
	"bytes"
	"context"
	"io"
	"io/fs"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	memBuckets   = map[string]*MemFS{}
	memBucketsMu sync.Mutex
)

func openOrCreateMemBucket(bucket string) *MemFS {
	memBucketsMu.Lock()
	defer memBucketsMu.Unlock()
	if b, ok := memBuckets[bucket]; ok {
		return b
	}
	b := &MemFS{files: map[string][]byte{}}
	memBuckets[bucket] = b

	return b
}

func init() {
	Register("mem", func(_ context.Context, parsed *url.URL, _ map[string]string) (IO, error) {
		return openOrCreateMemBucket(parsed.Host), nil
	})
}

// MemFS is a simple in-memory IO implementation used for testing and lightweight workloads.
// Files are stored per named bucket and persist for the lifetime of the process.
// Use the "mem://bucket/path" URI scheme to access files.
type MemFS struct {
	mu    sync.RWMutex
	files map[string][]byte // keyed by full URI
}

// NewMemFS creates an initialized MemFS ready for use.
func NewMemFS() *MemFS {
	return &MemFS{files: make(map[string][]byte)}
}

func (m *MemFS) Open(name string) (File, error) {
	m.mu.RLock()
	data, ok := m.files[name]
	m.mu.RUnlock()
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	cp := make([]byte, len(data))
	copy(cp, data)

	return &memFile{data: cp, name: path.Base(name)}, nil
}

func (m *MemFS) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.files[name]; !ok {
		return &fs.PathError{Op: "remove", Path: name, Err: fs.ErrNotExist}
	}

	delete(m.files, name)

	return nil
}

func (m *MemFS) Create(name string) (FileWriter, error) {
	return &memWriter{fs: m, name: name}, nil
}

func (m *MemFS) WriteFile(name string, content []byte) error {
	cp := make([]byte, len(content))
	copy(cp, content)
	m.mu.Lock()
	m.files[name] = cp
	m.mu.Unlock()

	return nil
}

func (m *MemFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	type walkEntry struct {
		path  string
		size  int64
		isDir bool
	}

	root = strings.TrimRight(root, "/")

	m.mu.RLock()
	entriesByPath := make(map[string]walkEntry)
	addEntry := func(entry walkEntry) {
		current, exists := entriesByPath[entry.path]
		if !exists || (!entry.isDir && current.isDir) {
			entriesByPath[entry.path] = entry
		}
	}
	for key, data := range m.files {
		if !memPathInRoot(key, root) {
			continue
		}

		if key == root {
			addEntry(walkEntry{path: key, size: int64(len(data))})

			continue
		}

		addEntry(walkEntry{path: root, isDir: true})
		relative := strings.TrimPrefix(key, root+"/")
		parts := strings.Split(relative, "/")
		for i := range parts {
			entryPath := strings.Join(parts[:i+1], "/")
			if root != "" {
				entryPath = root + "/" + entryPath
			}
			entry := walkEntry{path: entryPath, isDir: i < len(parts)-1}
			if !entry.isDir {
				entry.size = int64(len(data))
			}
			addEntry(entry)
		}
	}
	m.mu.RUnlock()
	if len(entriesByPath) == 0 {
		err := fn(root, nil, &fs.PathError{Op: "lstat", Path: root, Err: fs.ErrNotExist})
		if err == fs.SkipDir || err == fs.SkipAll {
			return nil
		}

		return err
	}
	if rootEntry, ok := entriesByPath[root]; ok && !rootEntry.isDir {
		entriesByPath = map[string]walkEntry{root: rootEntry}
	}

	paths := make([]string, 0, len(entriesByPath))
	for entryPath := range entriesByPath {
		paths = append(paths, entryPath)
	}
	sort.Slice(paths, func(i, j int) bool {
		if paths[i] == root {
			return true
		}
		if paths[j] == root {
			return false
		}

		left := strings.Split(strings.TrimPrefix(paths[i], root+"/"), "/")
		right := strings.Split(strings.TrimPrefix(paths[j], root+"/"), "/")
		for index := range min(len(left), len(right)) {
			if left[index] != right[index] {
				return left[index] < right[index]
			}
		}

		return len(left) < len(right)
	})

	var skipPrefix string
	for _, entryPath := range paths {
		if skipPrefix != "" && strings.HasPrefix(entryPath, skipPrefix) {
			continue
		}

		entry := entriesByPath[entryPath]
		info := &memFileInfo{name: path.Base(entry.path), size: entry.size, isDir: entry.isDir}
		err := fn(entry.path, fs.FileInfoToDirEntry(info), nil)
		switch err {
		case nil:
		case fs.SkipAll:
			return nil
		case fs.SkipDir:
			if entry.isDir {
				skipPrefix = entry.path + "/"
			} else {
				skipPrefix = path.Dir(entry.path) + "/"
			}
		default:
			return err
		}
	}

	return nil
}

func memPathInRoot(path, root string) bool {
	return root == "" || path == root || strings.HasPrefix(path, root+"/")
}

type memFile struct {
	data []byte
	name string
	pos  int64
}

func (f *memFile) Read(p []byte) (int, error) {
	if f.pos >= int64(len(f.data)) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.pos:])
	f.pos += int64(n)

	return n, nil
}

func (f *memFile) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = f.pos + offset
	case io.SeekEnd:
		abs = int64(len(f.data)) + offset
	default:
		return 0, &fs.PathError{Op: "seek", Path: f.name, Err: fs.ErrInvalid}
	}
	if abs < 0 {
		return 0, &fs.PathError{Op: "seek", Path: f.name, Err: fs.ErrInvalid}
	}
	f.pos = abs

	return f.pos, nil
}

func (f *memFile) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, &fs.PathError{Op: "readat", Path: f.name, Err: fs.ErrInvalid}
	}
	if off >= int64(len(f.data)) {
		return 0, io.EOF
	}
	n := copy(p, f.data[off:])
	if n < len(p) {
		return n, io.EOF
	}

	return n, nil
}

func (f *memFile) Close() error { return nil }

func (f *memFile) Stat() (fs.FileInfo, error) {
	return &memFileInfo{name: f.name, size: int64(len(f.data))}, nil
}

type memFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (fi *memFileInfo) Name() string { return fi.name }
func (fi *memFileInfo) Size() int64  { return fi.size }
func (fi *memFileInfo) Mode() fs.FileMode {
	if fi.isDir {
		return fs.ModeDir
	}

	return 0
}
func (fi *memFileInfo) ModTime() time.Time { return time.Time{} }
func (fi *memFileInfo) IsDir() bool        { return fi.isDir }
func (fi *memFileInfo) Sys() any           { return nil }

type memWriter struct {
	fs   *MemFS
	name string
	buf  bytes.Buffer
}

func (w *memWriter) Write(p []byte) (int, error)         { return w.buf.Write(p) }
func (w *memWriter) ReadFrom(r io.Reader) (int64, error) { return w.buf.ReadFrom(r) }
func (w *memWriter) Close() error {
	return w.fs.WriteFile(w.name, w.buf.Bytes())
}
