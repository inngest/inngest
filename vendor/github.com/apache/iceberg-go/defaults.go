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

import "slices"

// CloneDefaultValue returns a deep copy of an Iceberg initial or write default value.
func CloneDefaultValue(value any) any {
	switch value := value.(type) {
	case []byte:
		return slices.Clone(value)
	case BinaryLiteral:
		return BinaryLiteral(slices.Clone([]byte(value)))
	case FixedLiteral:
		return FixedLiteral(slices.Clone([]byte(value)))
	case []any:
		cloned := make([]any, len(value))
		for i, item := range value {
			cloned[i] = CloneDefaultValue(item)
		}

		return cloned
	case map[string]any:
		cloned := make(map[string]any, len(value))
		for key, item := range value {
			cloned[key] = CloneDefaultValue(item)
		}

		return cloned
	default:
		// Iceberg scalar defaults are value types (bool, numeric values,
		// strings, UUIDs, and Decimals). Any mutable reference type must get
		// an explicit deep-copy case above.
		return value
	}
}
