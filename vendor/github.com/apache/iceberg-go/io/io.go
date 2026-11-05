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

// Package io provides an interface for IO implementations along with
// a registry for registering IO implementations for different URI schemes.
//
// Subpackages of this package provide implementations for cloud storage providers
// which will register themselves if imported. For instance, adding the following
// import:
//
//	import _ "github.com/apache/iceberg-go/io/gocloud"
//
// Will register S3, GCS and Azure, and link all three cloud SDKs.
// Importing io/gocloud/s3, io/gocloud/gcs or io/gocloud/azure instead registers only that backend.
// The local filesystem (file:// and empty scheme) is always registered.
package io

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
)

// IO is an interface to a hierarchical file system.
//
// The IO interface is the minimum implementation required for a file
// system to utilize an iceberg table. A file system may implement
// additional interfaces, such as ReadFileIO, to provide additional or
// optimized functionality.
type IO interface {
	// Open opens the named file.
	//
	// When Open returns an error, it should be of type *PathError
	// with the Op field set to "open", the Path field set to name,
	// and the Err field describing the problem.
	//
	// Open should reject attempts to open names that do not satisfy
	// fs.ValidPath(name), returning a *PathError with Err set to
	// ErrInvalid or ErrNotExist.
	Open(name string) (File, error)

	// Remove removes the named file or (empty) directory.
	//
	// If there is an error, it will be of type *PathError.
	// Implementations must be safe for concurrent use by multiple goroutines.
	Remove(name string) error
}

// ReadFileIO is the interface implemented by a file system that
// provides an optimized implementation of ReadFile.
type ReadFileIO interface {
	IO

	// ReadFile reads the named file and returns its contents.
	// A successful call returns a nil error, not io.EOF.
	// (Because ReadFile reads the whole file, the expected EOF
	// from the final Read is not treated as an error to be reported.)
	//
	// The caller is permitted to modify the returned byte slice.
	// This method should return a copy of the underlying data.
	ReadFile(name string) ([]byte, error)
}

// WriteFileIO is the interface implemented by a file system that
// provides an optimized implementation of WriteFile
type WriteFileIO interface {
	IO

	// Create attempts to create the named file and return a writer
	// for it.
	Create(name string) (FileWriter, error)

	// WriteFile writes p to the named file.
	WriteFile(name string, p []byte) error
}

// ListableIO is the interface implemented by a file system that
// supports directory walking. This replaces the need for reflect-based
// access to internal file system structures (e.g., blob storage buckets).
//
// The root parameter is a full URI (e.g., "s3://bucket/path" or
// "/local/path") as used throughout the iceberg-go codebase.
// The fn callback receives full URI paths consistent with the root's
// scheme and authority.
type ListableIO interface {
	IO

	// WalkDir walks the file tree rooted at root, calling fn for each
	// file or directory in the tree, including root. Paths passed to fn
	// are full URIs consistent with the root's scheme and authority.
	WalkDir(root string, fn fs.WalkDirFunc) error
}

// BulkRemovableIO is an optional interface for IO implementations that
// support deleting multiple files in a single batch operation.
// Cloud object stores (S3 DeleteObjects, GCS batch, Azure batch) can
// implement this for significantly better throughput than single-file Remove.
type BulkRemovableIO interface {
	IO

	// DeleteFiles deletes all named files. Implementations should make a
	// best-effort attempt to delete as many files as possible, returning
	// the list of successfully deleted paths and a joined error
	// (via [errors.Join]) for any individual failures.
	//
	// Missing files are not considered errors — if a path does not exist,
	// it should be treated as a successful deletion.
	//
	// An empty paths slice is a no-op and must not error.
	DeleteFiles(ctx context.Context, paths []string) (deleted []string, err error)
}

// A File provides access to a single file. The File interface is the
// minimum implementation required for Iceberg to interact with a file.
// Directory files should also implement
type File interface {
	fs.File
	io.ReadSeekCloser
	io.ReaderAt
}

// A FileWriter represents an open writable file.
type FileWriter interface {
	io.WriteCloser
	io.ReaderFrom
}

// A ReadDirFile is a directory file whose entries can be read with the
// ReadDir method. Every directory file should implement this interface.
// (It is permissible for any file to implement this interface, but
// if so ReadDir should return an error for non-directories.)
type ReadDirFile interface {
	File

	// ReadDir read the contents of the directory and returns a slice
	// of up to n DirEntry values in directory order. Subsequent calls
	// on the same file will yield further DirEntry values.
	//
	// If n > 0, ReadDir returns at most n DirEntry structures. In this
	// case, if ReadDir returns an empty slice, it will return a non-nil
	// error explaining why.
	//
	// At the end of a directory, the error is io.EOF. (ReadDir must return
	// io.EOF itself, not an error wrapping io.EOF.)
	//
	// If n <= 0, ReadDir returns all the DirEntry values from the directory
	// in a single slice. In this case, if ReadDir succeeds (reads all the way
	// to the end of the directory), it returns the slice and a nil error.
	// If it encounters an error before the end of the directory, ReadDir
	// returns the DirEntry list read until that point and a non-nil error.
	ReadDir(n int) ([]fs.DirEntry, error)
}

// StatIO is an extension of IO interface that includes the Stat
// method for retrieving file or directory information
type StatIO interface {
	IO

	// The Stat method returns a FileInfo describing the named file or directory,
	// or an error satisfying errors.Is(err, fs.ErrNotExist) if the file does not exist
	Stat(path string) (fs.FileInfo, error)
}

// RenameIO is an extension of IO interface that includes the Rename
// method for renaming (moving) one file or object. This is atomic on
// a local filesystem but is not atomic in other implementations like blob stores
type RenameIO interface {
	IO

	Rename(oldpath, newpath string) error
}

// RenameNoReplaceIO is an extension of IO interface that includes the
// RenameNoReplace method for moving files only when the destination
// path does not already exist. Implementations may require oldpath and newpath
// to live on the same filesystem. This is atomic on a local filesystem but
// is not atomic in other implementations like blob stores
type RenameNoReplaceIO interface {
	IO

	RenameNoReplace(oldpath, newpath string) error
}

// RemoveAllIO is an extension of IO interface that includes the RemoveAll
// method for removing a file path and any children recursively
type RemoveAllIO interface {
	IO

	RemoveAll(name string) error
}

// MkdirAllIO is an extension of IO interface that includes the MkdirAll
// method for creating a directory path recursively
type MkdirAllIO interface {
	IO

	// MkdirAll should not raise an error if the directory already exists,
	// and should create any parent directories in the path if they
	// do not already exist.
	MkdirAll(path string) error
}

// FS wraps an io/fs.FS as an IO interface.
func FS(fsys fs.FS) IO {
	if _, ok := fsys.(fs.ReadFileFS); ok {
		return readFileFS{ioFS{fsys, nil}}
	}

	return ioFS{fsys, nil}
}

// FSPreProcName wraps an io/fs.FS like FS, only if fn is non-nil then
// it is called to preprocess any filenames before they are passed to
// the underlying fsys.
func FSPreProcName(fsys fs.FS, fn func(string) string) IO {
	if _, ok := fsys.(fs.ReadFileFS); ok {
		return readFileFS{ioFS{fsys, fn}}
	}

	return ioFS{fsys, fn}
}

type readFileFS struct {
	ioFS
}

func (r readFileFS) ReadFile(name string) ([]byte, error) {
	rfs, ok := r.fsys.(fs.ReadFileFS)
	if !ok {
		return nil, errMissingReadFile
	}

	return rfs.ReadFile(r.processName(name))
}

type ioFS struct {
	fsys fs.FS

	preProcessName func(string) string
}

func (f ioFS) Open(name string) (File, error) {
	name = f.processName(name)
	file, err := f.fsys.Open(name)
	if err != nil {
		return nil, err
	}

	return ioFile{file}, nil
}

func (f ioFS) processName(name string) string {
	if f.preProcessName != nil {
		name = f.preProcessName(name)
	}

	if name == "/" {
		name = "."
	} else {
		name = strings.TrimPrefix(name, "/")
	}

	return name
}

func (f ioFS) Remove(name string) error {
	r, ok := f.fsys.(interface{ Remove(name string) error })
	if !ok {
		return errMissingRemove
	}

	return r.Remove(f.processName(name))
}

var (
	ErrIOSchemeNotFound = errors.New("io scheme not registered")

	errMissingReadDir  = errors.New("fs.File directory missing ReadDir method")
	errMissingSeek     = errors.New("fs.File missing Seek method")
	errMissingReadAt   = errors.New("fs.File missing ReadAt")
	errMissingRemove   = errors.New("fs.FS missing Remove method")
	errMissingReadFile = errors.New("fs.FS missing ReadFile method")
)

type ioFile struct {
	file fs.File
}

func (f ioFile) Close() error               { return f.file.Close() }
func (f ioFile) Read(b []byte) (int, error) { return f.file.Read(b) }
func (f ioFile) Stat() (fs.FileInfo, error) { return f.file.Stat() }
func (f ioFile) Seek(offset int64, whence int) (int64, error) {
	s, ok := f.file.(io.Seeker)
	if !ok {
		return 0, errMissingSeek
	}

	return s.Seek(offset, whence)
}

func (f ioFile) ReadAt(p []byte, off int64) (n int, err error) {
	r, ok := f.file.(io.ReaderAt)
	if !ok {
		return 0, errMissingReadAt
	}

	return r.ReadAt(p, off)
}

func (f ioFile) ReadDir(count int) ([]fs.DirEntry, error) {
	d, ok := f.file.(fs.ReadDirFile)
	if !ok {
		return nil, errMissingReadDir
	}

	return d.ReadDir(count)
}

// LoadFS takes a map of properties and an optional URI location
// and attempts to infer an IO object from it using the registered
// scheme factories.
//
// The scheme is extracted from the location URI and used to look up
// the appropriate factory from the registry. The local filesystem
// (file:// or empty scheme) is registered by default.
//
// Additional schemes can be registered by importing subpackages.
// Import the backend for each cloud in use, or io/gocloud for all of them:
//
//	import _ "github.com/apache/iceberg-go/io/gocloud/s3"
func LoadFS(ctx context.Context, props map[string]string, location string) (IO, error) {
	if location == "" {
		location = props["warehouse"]
	}

	iofs, err := inferFileIOFromScheme(ctx, location, props)
	if err != nil {
		return nil, err
	}

	if iofs == nil {
		iofs = LocalFS{}
	}

	return iofs, nil
}

// LoadFSFunc is a helper function to create IO Func factory for later usage
func LoadFSFunc(props map[string]string, location string) func(ctx context.Context) (IO, error) {
	return func(ctx context.Context) (IO, error) {
		iofs, err := LoadFS(ctx, props, location)
		if err != nil {
			return nil, fmt.Errorf("failed to load metadata file at %s: %w", location, err)
		}

		return iofs, nil
	}
}
