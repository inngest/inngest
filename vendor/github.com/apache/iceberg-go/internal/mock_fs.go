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

package internal

import (
	"bytes"
	"errors"
	sio "io"
	"io/fs"

	"github.com/apache/iceberg-go/io"
	"github.com/stretchr/testify/mock"
)

type MockFS struct {
	mock.Mock
}

func (m *MockFS) Open(name string) (io.File, error) {
	args := m.Called(name)

	return args.Get(0).(io.File), args.Error(1)
}

func (m *MockFS) Create(name string) (io.FileWriter, error) {
	args := m.Called(name)

	return args.Get(0).(io.FileWriter), args.Error(1)
}

func (m *MockFS) WriteFile(name string, content []byte) error {
	return m.Called(name, content).Error(0)
}

func (m *MockFS) Remove(name string) error {
	return m.Called(name).Error(0)
}

type MockFSReadFile struct {
	MockFS
	ErrOnClose bool
}

func (m *MockFSReadFile) ReadFile(name string) ([]byte, error) {
	args := m.Called(name)

	return args.Get(0).([]byte), args.Error(1)
}

type MockFile struct {
	Contents   *bytes.Reader
	ErrOnClose bool

	closed bool
}

func (m *MockFile) Stat() (fs.FileInfo, error) {
	return nil, nil
}

func (m *MockFile) Read(p []byte) (int, error) {
	return m.Contents.Read(p)
}

func (m *MockFile) ReadFrom(r sio.Reader) (n int64, err error) {
	return 0, nil
}

func (m *MockFile) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *MockFile) Close() error {
	if m.ErrOnClose {
		return errors.New("error on close")
	}
	if m.closed {
		return errors.New("already closed")
	}
	m.closed = true

	return nil
}

func (m *MockFile) ReadAt(p []byte, off int64) (n int, err error) {
	if m.closed {
		return 0, errors.New("already closed")
	}

	return m.Contents.ReadAt(p, off)
}

func (m *MockFile) Seek(offset int64, whence int) (n int64, err error) {
	if m.closed {
		return 0, errors.New("already closed")
	}

	return m.Contents.Seek(offset, whence)
}
