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
	"errors"
	"fmt"
)

var (
	ErrInvalidTypeString       = errors.New("invalid type")
	ErrNotImplemented          = errors.New("not implemented")
	ErrInvalidArgument         = errors.New("invalid argument")
	ErrInvalidFormatVersion    = fmt.Errorf("%w: invalid format version", ErrInvalidArgument)
	ErrInvalidSchema           = errors.New("invalid schema")
	ErrInvalidPartitionSpec    = errors.New("invalid partition spec")
	ErrEmptyManifest           = errors.New("empty manifest file has been written")
	ErrInvalidTransform        = errors.New("invalid transform syntax")
	ErrType                    = errors.New("type error")
	ErrBadCast                 = errors.New("could not cast value")
	ErrBadLiteral              = errors.New("invalid literal value")
	ErrInvalidBinSerialization = errors.New("invalid binary serialization")
	ErrInvalidFixedLength      = errors.New("invalid fixed literal length")
	ErrResolve                 = errors.New("cannot resolve type")
	// ErrBBoxNotSerializable is returned when marshaling a geospatial bbox
	// predicate to REST expression JSON: bbox predicates exist only for local
	// scan planning and have no representation in the REST expression grammar.
	// It wraps ErrNotImplemented (not ErrInvalidArgument) because this is a
	// contract/not-implemented condition, not a bad caller argument - matching
	// how the substrait backstop surfaces the same predicate.
	ErrBBoxNotSerializable = fmt.Errorf("%w: geospatial bbox predicates cannot be serialized to REST expression JSON", ErrNotImplemented)
	// ErrExtractNotSerializable is returned when marshaling a variant extract term
	// to REST expression JSON: extract terms have no representation in the REST
	// expression grammar and would otherwise encode to an empty object.
	ErrExtractNotSerializable = fmt.Errorf("%w: variant extract terms cannot be serialized to REST expression JSON", ErrNotImplemented)
)
