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

package table

import (
	"github.com/apache/iceberg-go"
	iceberginternal "github.com/apache/iceberg-go/internal"
)

// manifestFilePartitions returns partition summaries without copying for the
// built-in manifest file and falls back to the public getter for external
// implementations. Callers must treat the returned slice and nested bounds as
// read-only and must not retain them beyond the current operation.
func manifestFilePartitions(file iceberg.ManifestFile) []iceberg.FieldSummary {
	return iceberginternal.BorrowedManifestFilePartitions(file)
}
