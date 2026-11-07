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
	"net/url"
	"strings"

	"github.com/apache/iceberg-go"
)

type partitionPathField struct {
	escapedName string
	transform   iceberg.Transform
	resultType  iceberg.Type
}

// partitionPathPlan contains the spec and schema state needed to format a
// partition path. The values are immutable after construction, so a plan can
// be shared by all workers handling the same table write.
type partitionPathPlan struct {
	fields        []partitionPathField
	estimatedSize int
}

func (p partitionPathPlan) format(data iceberg.StructLike) string {
	if len(p.fields) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow(p.estimatedSize)
	for i, field := range p.fields {
		if i > 0 {
			sb.WriteByte('/')
		}

		sb.WriteString(field.escapedName)
		sb.WriteByte('=')
		value := field.transform.ToHumanStrType(field.resultType, data.Get(i))
		sb.WriteString(url.QueryEscape(value))
	}

	return sb.String()
}
