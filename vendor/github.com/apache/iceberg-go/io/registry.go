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
	"context"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"sync"

	"github.com/apache/iceberg-go/internal/schemes"
)

type registry map[string]SchemeFactory

var (
	regMutex        sync.RWMutex
	defaultRegistry = registry{}
)

// SchemeFactory is a function that creates an IO implementation for a given URI and properties.
type SchemeFactory func(ctx context.Context, parsed *url.URL, props map[string]string) (IO, error)

// Register adds a new scheme factory to the registry. If the scheme is already registered, it will panic.
func Register(scheme string, factory SchemeFactory) {
	if factory == nil {
		panic("io: Register factory is nil")
	}

	regMutex.Lock()
	defer regMutex.Unlock()

	if _, dup := defaultRegistry[scheme]; dup {
		panic("io: Register called twice for scheme " + scheme)
	}
	defaultRegistry[scheme] = factory
}

// Unregister removes the requested scheme factory from the registry.
func Unregister(scheme string) {
	regMutex.Lock()
	defer regMutex.Unlock()
	delete(defaultRegistry, scheme)
}

// GetRegisteredSchemes returns the list of registered scheme names.
func GetRegisteredSchemes() []string {
	regMutex.RLock()
	defer regMutex.RUnlock()

	return slices.Collect(maps.Keys(defaultRegistry))
}

func init() {
	// Register local filesystem schemes
	localFSFactory := func(ctx context.Context, parsed *url.URL, props map[string]string) (IO, error) {
		return LocalFS{}, nil
	}
	Register("file", localFSFactory)
	Register("", localFSFactory)
}

func schemeImportHint(backend string) string {
	return "hint: import the matching IO module for side-effect registration: " +
		`_ "github.com/apache/iceberg-go/io/gocloud/` + backend + `"`
}

func schemeRegistrationHint(scheme string) string {
	backend := schemes.BackendFor(scheme)
	if backend == "" {
		return ""
	}

	return schemeImportHint(backend)
}

func inferFileIOFromScheme(ctx context.Context, path string, props map[string]string) (IO, error) {
	parsed, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	regMutex.RLock()
	factory, ok := defaultRegistry[parsed.Scheme]
	regMutex.RUnlock()

	if !ok {
		hint := schemeRegistrationHint(parsed.Scheme)
		if hint != "" {
			return nil, fmt.Errorf("%w for path %q (scheme: %s)\n%s", ErrIOSchemeNotFound, path, parsed.Scheme, hint)
		}

		return nil, fmt.Errorf("%w for path %q (scheme: %s)", ErrIOSchemeNotFound, path, parsed.Scheme)
	}

	return factory(ctx, parsed, props)
}
