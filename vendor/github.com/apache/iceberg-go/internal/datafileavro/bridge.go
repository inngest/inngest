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

// Package datafileavro is an internal bridge that lets the
// github.com/apache/iceberg-go/codec package consume the iceberg
// package's manifest-entry Avro decoder without iceberg having to
// export it publicly.
//
// The iceberg package's init populates [Unmarshal] with a closure
// over its unexported decode function. The codec package calls
// [Unmarshal] through this indirection. The any-typed parameters
// break what would otherwise be a circular import between iceberg
// and codec.
package datafileavro

// Unmarshal is set by the iceberg package's init() function. It
// invokes the iceberg package's unexported manifest-entry Avro
// decoder and returns an iceberg.DataFile (typed as any to avoid an
// import cycle); callers must type-assert it back.
//
// spec carries an iceberg.PartitionSpec; schema carries a
// *iceberg.Schema. version is the Iceberg format version (1, 2, or 3).
var Unmarshal func(data []byte, spec, schema any, version int) (any, error)
