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
	"bytes"
	"encoding/json"
	"fmt"
)

// validatePlanningTaskEnvelope mirrors the Java response validation for the
// status-discriminated planning responses. Empty arrays are still present on
// the wire, so inspect the raw JSON instead of relying only on decoded slice
// lengths when rejecting task fields before planning completes.
func validatePlanningTaskEnvelope(data []byte, status PlanStatus, tasks ScanTasks, endpoint string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	for _, name := range []string{"plan-tasks", "file-scan-tasks", "delete-files"} {
		raw, ok := fields[name]
		if !ok {
			continue
		}
		if isJSONNull(raw) {
			return fmt.Errorf("%w: %s response field %s must not be null", ErrRESTError, endpoint, name)
		}
		if status != PlanStatusCompleted {
			return fmt.Errorf("%w: %s response includes %s for status %q", ErrRESTError, endpoint, name, status)
		}
	}

	if len(tasks.DeleteFiles) > 0 && len(tasks.FileScanTasks) == 0 {
		return fmt.Errorf(
			"%w: %s response has delete-files without file-scan-tasks",
			ErrRESTError,
			endpoint,
		)
	}

	return nil
}

func (r *FetchScanTasksResponse) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return fmt.Errorf("%w: fetchScanTasks response must be an object", ErrRESTError)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	planTasks, hasPlanTasks := fields["plan-tasks"]
	fileScanTasks, hasFileScanTasks := fields["file-scan-tasks"]
	deleteFiles, hasDeleteFiles := fields["delete-files"]

	isNull := func(raw json.RawMessage) bool {
		return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
	}
	if hasPlanTasks && isNull(planTasks) {
		return fmt.Errorf("%w: fetchScanTasks response field plan-tasks must not be null", ErrRESTError)
	}
	if hasFileScanTasks && isNull(fileScanTasks) {
		return fmt.Errorf("%w: fetchScanTasks response field file-scan-tasks must not be null", ErrRESTError)
	}
	if hasDeleteFiles && isNull(deleteFiles) {
		return fmt.Errorf("%w: fetchScanTasks response field delete-files must not be null", ErrRESTError)
	}

	type responseAlias FetchScanTasksResponse
	var decoded responseAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if len(decoded.DeleteFiles) > 0 && len(decoded.FileScanTasks) == 0 {
		return fmt.Errorf("%w: fetchScanTasks response has delete-files without file-scan-tasks", ErrRESTError)
	}

	*r = FetchScanTasksResponse(decoded)

	return nil
}
