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

package metrics

// Unit identifies what a counter measures. The wire values match Java's
// MetricsContext.Unit display names.
type Unit string

const (
	UnitUndefined Unit = "undefined"
	UnitCount     Unit = "count"
	UnitBytes     Unit = "bytes"
)

// TimeUnit values for a TimerResult, matching Java's lowercased
// java.util.concurrent.TimeUnit names. iceberg-go records durations in
// nanoseconds.
const (
	TimeUnitNanoseconds = "nanoseconds"
)

// CounterResult is the serializable snapshot of a single counter. It maps to
// the spec's CounterResult schema.
type CounterResult struct {
	Unit  Unit  `json:"unit"`
	Value int64 `json:"value"`
}

// NewCounterResult returns a CounterResult with the given unit and value.
func NewCounterResult(unit Unit, value int64) *CounterResult {
	return &CounterResult{Unit: unit, Value: value}
}

// TimerResult is the serializable snapshot of a single timer. It maps to the
// spec's TimerResult schema. TotalDuration is expressed in TimeUnit.
type TimerResult struct {
	TimeUnit      string `json:"time-unit"`
	Count         int64  `json:"count"`
	TotalDuration int64  `json:"total-duration"`
}

// NewNanosTimerResult returns a TimerResult recording totalNanos nanoseconds
// observed over count samples.
func NewNanosTimerResult(count, totalNanos int64) *TimerResult {
	return &TimerResult{TimeUnit: TimeUnitNanoseconds, Count: count, TotalDuration: totalNanos}
}
