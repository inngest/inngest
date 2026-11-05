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

package schemes

import "slices"

// S3 covers oss because Alibaba OSS is reached through its S3-compatible API and the AWS SDK.
// Unlike Java, which has a dedicated OSSFileIO.
// OSS users here configure s3.* credential keys, not oss.*.
var (
	S3    = []string{"s3", "s3a", "s3n", "oss"}
	GCS   = []string{"gs"}
	Azure = []string{"abfs", "abfss", "wasb", "wasbs"}
)

// BackendFor returns the io/gocloud subpackage that registers scheme, or
// an empty string if no backend claims it.
func BackendFor(scheme string) string {
	switch {
	case slices.Contains(S3, scheme):
		return "s3"
	case slices.Contains(GCS, scheme):
		return "gcs"
	case slices.Contains(Azure, scheme):
		return "azure"
	default:
		return ""
	}
}
