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

// Package s3 provides the FileIO backend for S3 and S3-compatible object stores.
// Import it for its side effects to register the s3, s3a, s3n and oss schemes without linking the other clouds'
// SDKs:
//
//	import _ "github.com/apache/iceberg-go/io/gocloud/s3"
package s3

import (
	"context"
	"net/url"

	"github.com/apache/iceberg-go/internal/schemes"
	"github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/io/gocloud/blobfs"
)

func init() {
	factory := func(ctx context.Context, parsed *url.URL, props map[string]string) (io.IO, error) {
		bucket, err := createS3Bucket(ctx, parsed, props)
		if err != nil {
			return nil, err
		}

		return blobfs.New(ctx, bucket, blobfs.DefaultObjectLocationExtractor(parsed.Host, schemes.S3...)), nil
	}

	for _, scheme := range schemes.S3 {
		io.Register(scheme, factory)
	}
}
