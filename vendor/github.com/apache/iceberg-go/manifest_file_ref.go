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

import "github.com/apache/iceberg-go/internal"

// ManifestFilePartitionRef returns the manifest's partition summaries without
// copying for trusted in-module callers. The returned slice and all nested
// bounds alias the manifest and must be treated as read-only for the current
// operation.
func (m *manifestFile) ManifestFilePartitionRef(_ internal.ManifestFileRef) []FieldSummary {
	if m.PartitionList == nil {
		return nil
	}

	// The builder and decoder maintain that a non-nil PartitionList points to
	// a non-nil slice.
	return *m.PartitionList
}
