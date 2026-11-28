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

// This file contains the table-side seam for REST server-side scan planning
// (apache/iceberg-go#1178): the scan option, request/result types, and the
// ScanPlanner interface implemented by catalog/rest.

package table

import (
	"context"
	"io"

	"github.com/apache/iceberg-go"
	icebergio "github.com/apache/iceberg-go/io"
)

// ScanPlanningMode is the user-facing scan option: three values
// (local/remote/auto) selecting how (*Scan).PlanFiles plans a scan. Local
// planning remains the default; remote is opt-in via WithScanPlanningMode.
//
// This is deliberately distinct from the REST table-config key
// `scan-planning-mode` (values `client`/`server`), which is a server directive
// resolved separately (OQ4): a `client` table forces local planning, a `server`
// table forces remote planning, and explicit conflicting scan options fail
// fast. There is intentionally no fourth `server` value here; the directive
// lives in the table config, not the user option.
type ScanPlanningMode string

const (
	// ScanPlanningLocal always plans locally by reading manifests through the
	// table's FileIO. This is the default and current behavior.
	ScanPlanningLocal ScanPlanningMode = "local"
	// ScanPlanningRemote requires a planner that advertises remote capability
	// and fails loudly if remote planning is unavailable.
	ScanPlanningRemote ScanPlanningMode = "remote"
	// ScanPlanningAuto uses remote planning when available, otherwise it falls
	// back to local.
	ScanPlanningAuto ScanPlanningMode = "auto"
)

// WithScanPlanningMode sets the scan-planning mode for a scan. The default is
// ScanPlanningLocal.
func WithScanPlanningMode(mode ScanPlanningMode) ScanOption {
	return func(scan *Scan) { scan.planningMode = mode }
}

// ScanPlanningMetadata is the subset of table.Metadata a ScanPlanner needs:
// schema binding, snapshot resolution, partition decode, and property-based
// mode resolution. Narrowing from the full Metadata interface keeps the seam
// contract honest — planners do not depend on metadata logs, sort orders, or
// file lists. table.Metadata satisfies it.
type ScanPlanningMetadata interface {
	CurrentSchema() *iceberg.Schema
	Schemas() []*iceberg.Schema
	PartitionSpec() iceberg.PartitionSpec
	PartitionSpecByID(int) *iceberg.PartitionSpec
	CurrentSnapshot() *Snapshot
	SnapshotByID(int64) *Snapshot
	Properties() iceberg.Properties
}

// Compile-time guard that the full Metadata interface still satisfies the
// narrowed planner view, so callers can pass a table.Metadata directly.
var _ ScanPlanningMetadata = (Metadata)(nil)

// Compile-time guard that Scan exposes the lifecycle contract as io.Closer.
var _ io.Closer = (*Scan)(nil)

// ScanPlanningRequest is the input a Scan hands to a ScanPlanner. It carries
// the resolved scan state a planner needs without depending on catalog/rest.
//
// Schema is the schema resolved for this scan. It is kept separate from
// Metadata.CurrentSchema because a historical/tag scan may use a snapshot
// schema while a branch scan uses the table schema. Planners should use this
// same schema for filter binding and returned residual decoding.
type ScanPlanningRequest struct {
	Identifier Identifier
	// Metadata is the narrowed planner view of table metadata (see
	// ScanPlanningMetadata); MetadataLocation is kept separate.
	Metadata         ScanPlanningMetadata
	Schema           *iceberg.Schema
	MetadataLocation string
	// FileIOProperties are the table-scoped properties used to build the
	// table's normal FileIO. A remote planner can overlay plan-scoped
	// credentials on these properties without serializing them into the plan
	// request.
	FileIOProperties iceberg.Properties
	SnapshotID       *int64
	SelectedFields   []string
	RowFilter        iceberg.BooleanExpression
	MinRowsRequested *int64
	StatsFields      []string
	// MaxConcurrency is the maximum number of concurrent operations a planner
	// may use for this scan. Zero means use the planner's default.
	MaxConcurrency int
	// CaseSensitive must carry the Scan's value (which defaults to true), not
	// Go's false zero value, or the wire request would flip the spec default.
	// Nil means use the scan default.
	CaseSensitive *bool
	// UseSnapshotSchema is a pointer to distinguish the spec default from an
	// explicit false when the scanner-delegation phase binds it to table config.
	UseSnapshotSchema *bool
}

// PlanIO lazily loads the FileIO used to read a planned scan and closes any
// resources it holds (e.g. plan-scoped credentials) once the plan is replaced.
// Nil means the scan keeps using the table's normal FileIO. Remote planners may
// return a PlanIO backed by plan-scoped storage credentials. Implementations
// must use a comparable dynamic type with stable identity, normally a pointer.
//
// Delivery contract (OQ1): a returned ScanPlanningResult.IO is stored on the
// Scan that planned it; every ReadTasks call loads from it instead of the
// table's FileIO. Replacing the plan releases that Scan's ownership; the old IO
// closes after its remaining scan owners and active record iterators finish.
// Call Scan.Close when the scan is no longer needed, including when planning
// succeeds but its tasks are not read. Close is idempotent and waits for active
// ReadTasks iterators before the plan IO is closed.
// This ties a plan-scoped scan to PlanFiles -> ReadTasks: tasks from a remote
// plan must be read by the Scan that produced them or one of its derived scans.
// A Scan carrying plan-scoped IO is not safe for concurrent PlanFiles or
// ReadTasks calls, but consuming an existing iterator while replanning is safe.
type PlanIO interface {
	Load(context.Context) (icebergio.IO, error)
	Close() error
}

// ScanPlanningResult is what a ScanPlanner returns.
type ScanPlanningResult struct {
	Tasks []FileScanTask
	IO    PlanIO
}

// ScanPlanner plans scans for a table. rest.Catalog implements it; non-REST
// catalogs leave it nil and planning stays local.
//
// SupportsRemoteScanPlanning reports whether the planner can submit a remote
// plan for the requested scan. A planner with separate continuation
// capabilities can implement FullRemoteScanPlanner so auto mode can require
// the complete remote scan flow.
type ScanPlanner interface {
	SupportsRemoteScanPlanning() bool
	PlanFiles(context.Context, ScanPlanningRequest) (ScanPlanningResult, error)
}

// FullRemoteScanPlanner is an optional capability extension for ScanPlanner.
// A planner that exposes separate submission and continuation capabilities can
// implement SupportsFullRemoteScanPlanning so auto mode only selects it when
// the complete remote scan flow is available. Planners that do not implement
// this extension retain the original ScanPlanner behavior.
type FullRemoteScanPlanner interface {
	SupportsFullRemoteScanPlanning() bool
}

// Scan integration:
//
//	type Scan struct {
//		// ...existing fields...
//		planningMode ScanPlanningMode // set by WithScanPlanningMode; default ScanPlanningLocal
//		planner      ScanPlanner      // non-nil only when the catalog supplies one
//	}
//
// table.New sets Table.planner when the supplied CatalogIO also implements
// ScanPlanner; Table.Scan copies that planner into Scan. This chooses the
// Catalog -> Table -> Scan wiring now.
//
// (*Scan).PlanFiles resolves planningMode and, for remote/auto with a capable
// planner, delegates to planner.PlanFiles and stores any returned PlanIO on the
// Scan for ReadTasks. Auto without a capable planner runs the existing local
// path unchanged; remote without one fails loudly instead of silently planning
// locally. The compile-time `var _ table.ScanPlanner = (*Catalog)(nil)` in
// catalog/rest proves the REST catalog satisfies the planner interface.
