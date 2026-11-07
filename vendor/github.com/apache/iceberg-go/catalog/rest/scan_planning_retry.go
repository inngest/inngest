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

package rest

import (
	"context"
	"errors"
	"time"

	"github.com/apache/iceberg-go/internal/scanmetrics"
)

// retryScanPlanning retries only explicit transient HTTP responses. In
// particular, it never replays an ambiguous transport failure or a malformed
// successful response. POST callers create their idempotency header before
// entering this helper and reuse it (and the request body) for every attempt.
func retryScanPlanning[T any](ctx context.Context, operation string, request func() (T, error)) (T, error) {
	const (
		maxRetries = 3
		minDelay   = 100 * time.Millisecond
		maxDelay   = 5 * time.Second
	)
	_, hasDeadline := ctx.Deadline()
	delay := minDelay
	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			var zero T

			return zero, err
		}
		result, err := request()
		var responseError errorResponse
		if !errors.As(err, &responseError) {
			return result, err
		}
		retryAfter, retryable := scanPlanPollRetry(err)
		if !retryable || attempt == maxRetries {
			return result, err
		}
		delay = nextScanPlanBackoff(delay, minDelay, maxDelay)
		wait := applyRetryAfter(delay, retryAfter, minDelay, maxDelay, hasDeadline)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			var zero T

			return zero, ctx.Err()
		case <-timer.C:
			scanmetrics.Retry(ctx, operation)
		}
	}
}
