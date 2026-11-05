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

package iceberg

import (
	"maps"
	"sync"
)

const (
	// EnvironmentEngineNameKey identifies the engine name property.
	EnvironmentEngineNameKey = "engine-name"
	// EnvironmentEngineVersionKey identifies the engine version property.
	EnvironmentEngineVersionKey = "engine-version"

	environmentContextIcebergVersionKey = "iceberg-version"
)

var environmentContext = struct {
	mu         sync.RWMutex
	properties map[string]string
}{
	properties: map[string]string{
		environmentContextIcebergVersionKey: fullVersion(),
	},
}

// EnvironmentContext returns an independent snapshot of the process-wide
// context used to populate report metadata and persisted snapshot summaries.
// The returned map may be modified by the caller without changing the stored
// context.
func EnvironmentContext() map[string]string {
	environmentContext.mu.RLock()
	defer environmentContext.mu.RUnlock()

	return maps.Clone(environmentContext.properties)
}

// SetEnvironmentProperty sets one process-wide environment context property.
func SetEnvironmentProperty(key, value string) {
	environmentContext.mu.Lock()
	defer environmentContext.mu.Unlock()

	environmentContext.properties[key] = value
}

// RemoveEnvironmentProperty removes one process-wide environment context
// property.
func RemoveEnvironmentProperty(key string) {
	environmentContext.mu.Lock()
	defer environmentContext.mu.Unlock()

	delete(environmentContext.properties, key)
}
