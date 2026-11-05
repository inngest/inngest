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

// Package blobfs implements the iceberg-go/io FileIO interfaces on top of a gocloud.dev blob bucket.
// It links no cloud SDK of its own.
// The backend packages under io/gocloud each supply a bucket for their cloud.
package blobfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	pathpkg "path"
	"slices"
	"strings"
	"sync"
	"time"

	icebergio "github.com/apache/iceberg-go/io"
	"gocloud.dev/blob"
	"gocloud.dev/gcerrors"
)

// blobOpenFile describes a single open blob as a File.
// It implements the iceberg-go/io.File interface.
// It is based on gocloud.dev/blob.iofsOpenFile which:
// - Doesn't support the `io.ReaderAt` interface
// - Is not externally accessible, so copied here
type blobOpenFile struct {
	*blob.Reader

	name, key string
	b         *FileIO
	ctx       context.Context
}

func (f *blobOpenFile) ReadAt(p []byte, off int64) (n int, err error) {
	var rdr io.ReadCloser
	if f.b.newRangeReader != nil {
		rdr, err = f.b.newRangeReader(f.ctx, f.key, off, int64(len(p)))
	} else {
		rdr, err = f.b.NewRangeReader(f.ctx, f.key, off, int64(len(p)), nil)
	}
	if err != nil {
		return 0, err
	}
	// not using internal.CheckedClose due to import cycle
	defer func() { err = errors.Join(err, rdr.Close()) }()

	return io.ReadFull(rdr, p)
}

// Functions to implement the `Stat()` function in the `io/fs.File` interface

func (f *blobOpenFile) Name() string               { return f.name }
func (f *blobOpenFile) Mode() fs.FileMode          { return fs.ModeIrregular }
func (f *blobOpenFile) Sys() any                   { return f.b }
func (f *blobOpenFile) IsDir() bool                { return false }
func (f *blobOpenFile) Stat() (fs.FileInfo, error) { return f, nil }

// KeyExtractor extracts the object key from an input path.
type KeyExtractor func(path string) (string, error)

// ErrEmptyObjectKey is returned by object location extractors when a URI names
// a bucket or container but no object key within it.
var ErrEmptyObjectKey = errors.New("object key is empty")

// ErrUnsupportedObjectAuthority is returned when this single-bucket FileIO is
// asked to access a URI for another bucket or container. Load a FileIO for that
// URI to access it; this backend does not route across authorities.
var ErrUnsupportedObjectAuthority = errors.New("object URI authority is not supported by this FileIO")

// ObjectLocation is a parsed object-store URI:
// the scheme, the authority owning the object, and the object key within that authority.
type ObjectLocation struct {
	scheme       string
	authority    string
	key          string
	uriPrefix    string
	hasAuthority bool
}

func splitObjectLocation(location string) (ObjectLocation, error) {
	scheme, rest, ok := strings.Cut(location, "://")
	if !ok {
		return ObjectLocation{key: location}, nil
	}

	authorityEnd := strings.IndexAny(rest, "/?#")
	if authorityEnd == -1 {
		authorityEnd = len(rest)
	}

	authority := rest[:authorityEnd]
	if authority == "" {
		return ObjectLocation{}, fmt.Errorf("URI authority is empty: %s", location)
	}

	key := ""
	if authorityEnd < len(rest) {
		if rest[authorityEnd] != '/' {
			return ObjectLocation{}, fmt.Errorf("URI authority %q must be followed by an object path: %s",
				authority, location)
		}

		// Keep object keys raw. Cloud object names are opaque strings, so this
		// intentionally does not URL-unescape literal %, spaces, ? or #.
		key = strings.TrimPrefix(rest[authorityEnd:], "/")
	}

	return NewObjectLocation(scheme, authority, key), nil
}

// Scheme returns the URI scheme, empty for a location parsed from a bare key.
func (o ObjectLocation) Scheme() string { return o.scheme }

// Authority returns the bucket or container owning the object,
// empty for a location parsed from a bare key.
func (o ObjectLocation) Authority() string { return o.authority }

// Key returns the object key within the authority.
func (o ObjectLocation) Key() string { return o.key }

// NewObjectLocation builds a location for a URI carrying an explicit authority.
// Backends parsing their own URI grammar use it so the URI prefix stays
// consistent with the scheme and authority they parsed.
func NewObjectLocation(scheme, authority, key string) ObjectLocation {
	return ObjectLocation{
		scheme:       scheme,
		authority:    authority,
		key:          key,
		uriPrefix:    scheme + "://" + authority + "/",
		hasAuthority: true,
	}
}

// ObjectLocationExtractor resolves an input path to the object location this
// FileIO acts on, rejecting paths belonging to another authority.
type ObjectLocationExtractor func(location string) (ObjectLocation, error)

// KeyExtractorFromObjectLocation adapts an extractor to a KeyExtractor by
// keeping only the object key.
func KeyExtractorFromObjectLocation(extract ObjectLocationExtractor) KeyExtractor {
	return func(location string) (string, error) {
		parsed, err := extract(location)
		if err != nil {
			return "", err
		}

		return parsed.key, nil
	}
}

// DefaultObjectLocationExtractor extracts locations from URIs of the form scheme://bucket/key,
// accepting only the given schemes and bucket.
func DefaultObjectLocationExtractor(bucketName string, allowedSchemes ...string) ObjectLocationExtractor {
	return func(location string) (ObjectLocation, error) {
		parsed, err := splitObjectLocation(location)
		if err != nil {
			return ObjectLocation{}, err
		}

		if parsed.hasAuthority {
			if len(allowedSchemes) > 0 && !slices.Contains(allowedSchemes, parsed.scheme) {
				return ObjectLocation{}, fmt.Errorf("URI scheme %q is not supported by this FileIO (allowed: %s): %s",
					parsed.scheme, strings.Join(allowedSchemes, ", "), location)
			}
			if parsed.authority != bucketName {
				return ObjectLocation{}, fmt.Errorf("%w: URI authority %q does not match configured authority %q",
					ErrUnsupportedObjectAuthority,
					parsed.authority, bucketName)
			}
		} else {
			parsed.key = strings.TrimPrefix(location, bucketName+"/")
		}

		if parsed.key == "" {
			return parsed, fmt.Errorf("%w: %s", ErrEmptyObjectKey, location)
		}

		return parsed, nil
	}
}

// defaultKeyExtractor extracts the object key by removing the scheme and bucket name from the URI.
// e.g., s3://bucket/path/file -> path/file.
func defaultKeyExtractor(bucketName string, allowedSchemes ...string) KeyExtractor {
	return KeyExtractorFromObjectLocation(DefaultObjectLocationExtractor(bucketName, allowedSchemes...))
}

// FileIO is the iceberg-go/io implementation backed by a gocloud.dev bucket.
type FileIO struct {
	*blob.Bucket

	extractObject ObjectLocationExtractor
	ctx           context.Context

	// newRangeReader is an optional hook for testing.
	// It allows injecting a mock reader to verify Close calls.
	newRangeReader func(ctx context.Context, key string, offset, length int64) (io.ReadCloser, error)
}

var (
	_ icebergio.ListableIO      = (*FileIO)(nil)
	_ icebergio.BulkRemovableIO = (*FileIO)(nil)
)

// deleteFilesMaxConcurrency bounds the number of in-flight object-store
// deletes without creating one goroutine per path for large cleanup jobs.
const deleteFilesMaxConcurrency = 16

type blobFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	sys     any
}

// blobFileInfo implements the fs.FileInfo interface for a blob object
var _ fs.FileInfo = (*blobFileInfo)(nil)

func (f blobFileInfo) Name() string       { return f.name }
func (f blobFileInfo) Size() int64        { return f.size }
func (f blobFileInfo) Mode() fs.FileMode  { return f.mode }
func (f blobFileInfo) ModTime() time.Time { return f.modTime }
func (f blobFileInfo) IsDir() bool        { return f.mode.IsDir() }
func (f blobFileInfo) Sys() any           { return f.sys }

// preprocess returns the object key from an input path
func (bfs *FileIO) preprocess(path string) (string, error) {
	location, err := bfs.resolveLocation(path)
	if err != nil {
		return "", err
	}

	return location.key, nil
}

// blobErrToFsErr converts a gocloud blob error to an error from the fs package; this is
// necessary for comply with certain iceberg-go/io interface functions that expect fs.ErrNotExist
// for missing files, rather than the gocloud error.
func blobErrToFsErr(op, name string, err error) error {
	if gcerrors.Code(err) == gcerrors.NotFound {
		err = fs.ErrNotExist
	}

	return &fs.PathError{Op: op, Path: name, Err: err}
}

// directoryMarker turns a key into a string which can be interpreted as a directory marker
// for path based operations
func directoryMarker(key string) string {
	key = strings.TrimRight(key, "/")
	if key == "" {
		return ""
	}

	return key + "/"
}

// directoryName returns the last component of a key, which is interpreted as the directory name
func directoryName(key string) string {
	key = strings.TrimRight(key, "/")
	if key == "" {
		return "."
	}

	return pathpkg.Base(key)
}

func (bfs *FileIO) Open(path string) (icebergio.File, error) {
	originalPath := path
	var err error
	path, err = bfs.preprocess(path)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: originalPath, Err: err}
	}
	if !fs.ValidPath(path) {
		return nil, &fs.PathError{Op: "open", Path: originalPath, Err: fs.ErrInvalid}
	}

	key, name := path, pathpkg.Base(path)

	r, err := bfs.NewReader(bfs.ctx, key, nil)
	if err != nil {
		return nil, blobErrToFsErr("open", originalPath, err)
	}

	return &blobOpenFile{Reader: r, name: name, key: key, b: bfs, ctx: bfs.ctx}, nil
}

func (bfs *FileIO) Remove(name string) error {
	key, err := bfs.preprocess(name)
	if err != nil {
		return &fs.PathError{Op: "remove", Path: name, Err: err}
	}

	if err := bfs.Delete(bfs.ctx, key); err != nil {
		if gcerrors.Code(err) == gcerrors.NotFound {
			marker := directoryMarker(key)
			if marker != "" {
				if markerErr := bfs.Delete(bfs.ctx, marker); markerErr == nil {
					return nil
				} else if gcerrors.Code(markerErr) != gcerrors.NotFound {
					return &fs.PathError{Op: "remove", Path: name, Err: markerErr}
				}
			}

			err = fs.ErrNotExist
		}

		return &fs.PathError{Op: "remove", Path: name, Err: err}
	}

	return nil
}

func (bfs *FileIO) Create(name string) (icebergio.FileWriter, error) {
	return bfs.NewWriter(bfs.ctx, name, true, nil)
}

func (bfs *FileIO) WriteFile(name string, content []byte) error {
	key, err := bfs.preprocess(name)
	if err != nil {
		return &fs.PathError{Op: "write file", Path: name, Err: err}
	}

	return bfs.WriteAll(bfs.ctx, key, content, nil)
}

// NewWriter returns a Writer that writes to the blob stored at path.
// A nil WriterOptions is treated the same as the zero value.
//
// If overwrite is disabled and a blob with this path already exists,
// an error will be returned.
//
// The caller must call Close on the returned Writer, even if the write is
// aborted.
func (bfs *FileIO) NewWriter(ctx context.Context, path string, overwrite bool, opts *blob.WriterOptions) (w *Writer, err error) {
	key, err := bfs.preprocess(path)
	if err != nil {
		return nil, &fs.PathError{Op: "new writer", Path: path, Err: err}
	}
	if !fs.ValidPath(key) {
		return nil, &fs.PathError{Op: "new writer", Path: path, Err: fs.ErrInvalid}
	}

	if !overwrite {
		if exists, err := bfs.Exists(ctx, key); err != nil || exists {
			if err != nil {
				return nil, &fs.PathError{Op: "new writer", Path: path, Err: err}
			}

			return nil, &fs.PathError{Op: "new writer", Path: path, Err: fs.ErrInvalid}
		}
	}
	bw, err := bfs.Bucket.NewWriter(ctx, key, opts)
	if err != nil {
		return nil, err
	}

	return &Writer{
			Writer: bw,
			name:   key,
		},
		nil
}

// New returns a FileIO backed by bucket,
// using extractObject to map input paths to object keys within it.
func New(ctx context.Context, bucket *blob.Bucket, extractObject ObjectLocationExtractor) *FileIO {
	return &FileIO{Bucket: bucket, extractObject: extractObject, ctx: ctx}
}

func (bfs *FileIO) resolveLocation(root string) (ObjectLocation, error) {
	if bfs.extractObject == nil {
		return ObjectLocation{}, errors.New("blob file IO missing object location extractor")
	}

	return bfs.extractObject(root)
}

func walkedURIPath(location ObjectLocation, walked string) string {
	if walked == "." {
		return location.uriPrefix
	}

	if walked == "" {
		return location.uriPrefix
	}

	return location.uriPrefix + walked
}

// isDirectoryMarker reports whether dirEntry is the marker object for the
// path passed to WalkDir
func isDirectoryMarker(walkRootKey string, dirEntry fs.DirEntry) bool {
	// Directory entries and nil entries are never the marker object itself.
	if dirEntry == nil || dirEntry.IsDir() {
		return false
	}

	info, err := dirEntry.Info()
	if err != nil {
		return false
	}

	// blob fs is the only implementation that uses directory markers
	// so we can use a type assertion to check
	obj, ok := info.Sys().(*blob.ListObject)

	return ok && obj.Key == directoryMarker(walkRootKey)
}

func (bfs *FileIO) WalkDir(root string, fn fs.WalkDirFunc) error {
	location, err := bfs.resolveLocation(root)
	var walkPath string
	if err != nil {
		if !errors.Is(err, ErrEmptyObjectKey) {
			return &fs.PathError{Op: "walk dir", Path: root, Err: err}
		}

		walkPath = "."
	} else {
		walkPath = location.key
	}

	return fs.WalkDir(bfs.Bucket, walkPath, func(path string, d fs.DirEntry, err error) error {
		// Skip the marker for the path passed to WalkDir. Otherwise walking
		// "warehouse/ns" reports "warehouse/ns/" as a child file.
		if err == nil && isDirectoryMarker(walkPath, d) {
			return nil
		}

		if location.hasAuthority {
			path = walkedURIPath(location, path)
		}

		return fn(path, d, err)
	})
}

func (bfs *FileIO) deleteFile(ctx context.Context, p string) (bool, error) {
	key, err := bfs.preprocess(p)
	if err != nil {
		return false, fmt.Errorf("failed to delete %s: %w", p, err)
	}

	if err := bfs.Delete(ctx, key); err != nil {
		// Missing files are not errors per the interface contract.
		if gcerrors.Code(err) == gcerrors.NotFound {
			return true, nil
		}

		return false, fmt.Errorf("failed to delete %s: %w", p, err)
	}

	return true, nil
}

func (bfs *FileIO) DeleteFiles(ctx context.Context, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	type result struct {
		deleted bool
		err     error
	}

	results := make([]result, len(paths))
	workers := len(paths)
	if workers > deleteFilesMaxConcurrency {
		workers = deleteFilesMaxConcurrency
	}

	jobs := make(chan int)
	var wg sync.WaitGroup
	wg.Add(workers)

	for range workers {
		go func() {
			defer wg.Done()

			for idx := range jobs {
				results[idx].deleted, results[idx].err = bfs.deleteFile(ctx, paths[idx])
			}
		}()
	}

	for idx := range paths {
		jobs <- idx
	}
	close(jobs)
	wg.Wait()

	deleted := make([]string, 0, len(paths))
	var errs error
	for idx, result := range results {
		if result.deleted {
			deleted = append(deleted, paths[idx])
		}

		if result.err != nil {
			errs = errors.Join(errs, result.err)
		}
	}

	return deleted, errs
}

// MkdirAll mimics creating a directory by creating a zero-length object for each component of the path
func (bfs *FileIO) MkdirAll(path string) error {
	key, err := bfs.preprocess(path)
	if err != nil {
		return &fs.PathError{Op: "mkdir", Path: path, Err: err}
	}

	key = strings.Trim(key, "/")
	if key == "" || key == "." {
		return nil
	}

	parts := strings.Split(key, "/")
	for idx := range parts {
		marker := strings.Join(parts[:idx+1], "/") + "/"
		if err := bfs.WriteAll(bfs.ctx, marker, nil, nil); err != nil {
			return &fs.PathError{Op: "mkdir", Path: path, Err: err}
		}
	}

	return nil
}

// ReadFile reads the contents of the file at the given path and returns it as a byte slice.
func (bfs *FileIO) ReadFile(path string) ([]byte, error) {
	key, err := bfs.preprocess(path)
	if err != nil {
		return nil, &fs.PathError{Op: "ReadFile", Path: path, Err: err}
	}

	data, err := bfs.ReadAll(bfs.ctx, key)
	if err != nil {
		return nil, blobErrToFsErr("ReadFile", path, err)
	}

	return data, nil
}

// Stat interprets the input path as a directory or file and returns the corresponding FileInfo.
// If the path does not exist, it returns fs.ErrNotExist
func (bfs *FileIO) Stat(path string) (fs.FileInfo, error) {
	key, err := bfs.preprocess(path)
	if err != nil {
		return nil, &fs.PathError{Op: "Stat", Path: path, Err: err}
	}

	// If the path ends with a slash, we know by path semantics that we can
	// check for a directory immediately and skip needing to check it as both an object
	// or a directory marker.
	if strings.HasSuffix(key, "/") {
		if attrs, markerErr := bfs.Attributes(bfs.ctx, key); markerErr == nil {
			return blobFileInfo{
				name:    directoryName(key),
				mode:    fs.ModeDir,
				modTime: attrs.ModTime,
				sys:     attrs,
			}, nil
		} else if gcerrors.Code(markerErr) != gcerrors.NotFound {
			return nil, blobErrToFsErr("Stat", path, markerErr)
		}
	}

	attrs, err := bfs.Attributes(bfs.ctx, key)
	// if there is no error and we have attributes, we can return a FileInfo for object
	if err == nil {
		return blobFileInfo{
			name:    pathpkg.Base(key),
			size:    attrs.Size,
			mode:    fs.ModeIrregular,
			modTime: attrs.ModTime,
			sys:     attrs,
		}, nil
	}

	if gcerrors.Code(err) != gcerrors.NotFound {
		return nil, blobErrToFsErr("Stat", path, err)
	}

	marker := directoryMarker(key)
	// if the marker exists, we can return a FileInfo for the directory
	if marker != "" {
		if attrs, markerErr := bfs.Attributes(bfs.ctx, marker); markerErr == nil {
			return blobFileInfo{
				name:    directoryName(key),
				mode:    fs.ModeDir,
				modTime: attrs.ModTime,
				sys:     attrs,
			}, nil
		} else if gcerrors.Code(markerErr) != gcerrors.NotFound {
			return nil, blobErrToFsErr("Stat", path, markerErr)
		}
	}

	prefix := marker
	if prefix == "" {
		prefix = key
	}

	iter := bfs.List(&blob.ListOptions{Prefix: prefix})
	if obj, listErr := iter.Next(bfs.ctx); listErr == nil {
		return blobFileInfo{
			name: directoryName(key),
			mode: fs.ModeDir,
			sys:  obj,
		}, nil
	} else if listErr != io.EOF {
		return nil, blobErrToFsErr("Stat", path, listErr)
	}

	return nil, &fs.PathError{Op: "Stat", Path: path, Err: fs.ErrNotExist}
}

// Rename renames one file from oldpath to newpath, replacing newpath if it already exists.
func (bfs *FileIO) Rename(oldpath, newpath string) error {
	oldKey, err := bfs.preprocess(oldpath)
	if err != nil {
		return &fs.PathError{Op: "Rename", Path: oldpath, Err: err}
	}

	newKey, err := bfs.preprocess(newpath)
	if err != nil {
		return &fs.PathError{Op: "Rename", Path: newpath, Err: err}
	}

	if err := bfs.Copy(bfs.ctx, newKey, oldKey, nil); err != nil {
		return blobErrToFsErr("Rename", oldpath, err)
	}

	if err := bfs.Delete(bfs.ctx, oldKey); err != nil && gcerrors.Code(err) != gcerrors.NotFound {
		return blobErrToFsErr("Rename", oldpath, err)
	}

	return nil
}

// RenameNoReplace renames one file/object (non-recursive) from oldpath to newpath, returning an error if newpath already exists.
func (bfs *FileIO) RenameNoReplace(oldpath, newpath string) error {
	if _, err := bfs.Stat(newpath); err == nil {
		return &fs.PathError{Op: "RenameNoReplace", Path: newpath, Err: fs.ErrExist}
		// if the error is just that the file doesn't exist, we can continue with the rename
		// since we will be creating the new file. If the error is something else, we should return it.
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	return bfs.Rename(oldpath, newpath)
}

// RemoveAll removes either a single file or interprets the path as and removes both
// it and and all its children.
func (bfs *FileIO) RemoveAll(name string) error {
	key, err := bfs.preprocess(name)
	if err != nil {
		return &fs.PathError{Op: "RemoveAll", Path: name, Err: err}
	}

	if err := bfs.Delete(bfs.ctx, key); err != nil && gcerrors.Code(err) != gcerrors.NotFound {
		return &fs.PathError{Op: "RemoveAll", Path: name, Err: err}
	}

	prefix := directoryMarker(key)
	if prefix == "" {
		prefix = key
	}

	iter := bfs.List(&blob.ListOptions{Prefix: prefix})
	for {
		obj, err := iter.Next(bfs.ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return &fs.PathError{Op: "RemoveAll", Path: name, Err: err}
		}
		// If the err in question was NotFound, that just means the object was already deleted,
		// so we can ignore it and continue deleting the rest of the objects.
		if err := bfs.Delete(bfs.ctx, obj.Key); err != nil && gcerrors.Code(err) != gcerrors.NotFound {
			return &fs.PathError{Op: "RemoveAll", Path: name, Err: err}
		}
	}

	return nil
}

// Writer is the FileWriter returned by NewWriter.
// It exposes the underlying gocloud.dev writer for callers that need its options.
type Writer struct {
	*blob.Writer
	name string
	b    *FileIO
}

func (f *Writer) Name() string                { return f.name }
func (f *Writer) Sys() any                    { return f.b }
func (f *Writer) Close() error                { return f.Writer.Close() }
func (f *Writer) Write(p []byte) (int, error) { return f.Writer.Write(p) }
