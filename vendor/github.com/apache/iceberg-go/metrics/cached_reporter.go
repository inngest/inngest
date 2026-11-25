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

package metrics

import "sync"

// CachedReporter builds a [Reporter] from catalog properties once and caches
// it, so a catalog holds a single reporter for its lifetime — matching Java's
// per-catalog MetricsReporter — rather than constructing a fresh one on every
// table load or commit. That per-operation construction is what makes a
// stateful reporter (an HTTP-backed one holding a shared client or a background
// dispatch worker) leak: a new one per load with no owner to close it.
//
// The zero value is ready to use and safe for concurrent use. Close releases
// the built reporter, giving a catalog a single place to clean up at shutdown.
type CachedReporter struct {
	mu     sync.Mutex
	built  bool
	closed bool
	rep    Reporter
	err    error
}

// Get returns the cached reporter, building it from props on the first call via
// [FromProperties]. The first call's result — reporter and error — is cached and
// returned to every later caller; props supplied on subsequent calls is ignored,
// because a catalog's reporter configuration does not change over its lifetime.
//
// After Close, Get returns [NopReporter] with no error: the built reporter has
// been released, so handing it back would violate the "no Report after Close"
// contract, and returning nil would force every caller to nil-check.
func (c *CachedReporter) Get(props map[string]string) (Reporter, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return NopReporter{}, nil
	}
	if !c.built {
		c.rep, c.err = FromProperties(props)
		c.built = true
	}

	return c.rep, c.err
}

// Close closes the built reporter, if one was ever built, and is a no-op
// otherwise (including when Get was never called or returned an error). Close is
// idempotent: it nils the cached reporter after closing, so a second Close does
// not re-close an underlying reporter that is not obliged to tolerate it, and a
// post-Close Get hands back [NopReporter] rather than a released reporter.
//
// A factory returning a typed nil (e.g. (*Custom)(nil)) yields a non-nil
// [Reporter] interface that passes the guard below, so Close still invokes its
// Close — [Factory] implementations must return a usable reporter or an error,
// never a typed-nil reporter. Close is expected at catalog shutdown, after
// operations have quiesced; it is safe for concurrent use, but a Get racing a
// Close has no defined ordering.
func (c *CachedReporter) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	rep := c.rep
	c.rep = nil
	if rep != nil {
		return rep.Close()
	}

	return nil
}
