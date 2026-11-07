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
	"fmt"

	"github.com/apache/iceberg-go/internal"
	"github.com/twmb/avro"
	"github.com/twmb/avro/atype"
)

func partitionTypeToAvroSchema(t *StructType) (*avro.Schema, error) {
	fields := make([]avro.SchemaField, len(t.FieldList))
	for i, f := range t.FieldList {
		var node avro.SchemaNode
		switch typ := f.Type.(type) {
		case Int32Type:
			node = internal.NullableNode(internal.IntNode)
		case Int64Type:
			node = internal.NullableNode(internal.LongNode)
		case Float32Type:
			node = internal.NullableNode(internal.FloatNode)
		case Float64Type:
			node = internal.NullableNode(internal.DoubleNode)
		case StringType:
			node = internal.NullableNode(internal.StringNode)
		case DateType:
			node = internal.NullableNode(internal.DateNode)
		case TimeType:
			node = internal.NullableNode(internal.TimeNode)
		case TimestampType:
			node = internal.NullableNode(internal.TimestampNode)
		case TimestampTzType:
			node = internal.NullableNode(internal.TimestampTzNode)
		case TimestampNsType:
			node = internal.NullableNode(internal.TimestampNsNode)
		case TimestampTzNsType:
			node = internal.NullableNode(internal.TimestampTzNsNode)
		case UUIDType:
			node = internal.NullableNode(internal.UUIDNode)
		case BooleanType:
			node = internal.NullableNode(internal.BoolNode)
		case BinaryType:
			node = internal.NullableNode(internal.BytesNode)
		case FixedType:
			node = internal.NullableNode(internal.FixedNode(typ.Len()))
		case DecimalType:
			node = internal.NullableNode(internal.DecimalNode(typ.precision, typ.scale))
		case UnknownType:
			node = internal.NullNode
		default:
			return nil, fmt.Errorf("unsupported partition type: %s", f.Type.String())
		}

		fields[i] = avro.SchemaField{
			Name:  f.Name,
			Type:  node,
			Props: internal.WithFieldID(f.ID),
		}
	}

	node := avro.SchemaNode{
		Type:   atype.Record,
		Name:   "r102",
		Fields: fields,
	}

	return node.Schema()
}
