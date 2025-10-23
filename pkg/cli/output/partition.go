package output

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/inngest"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

func TextPartition(pt *pb.PartitionResponse, pts *pb.PartitionStatusResponse) error {
	w := NewTextWriter()

	if pt != nil {
		var fn inngest.Function
		if len(pt.GetConfig()) > 0 {
			if err := json.Unmarshal(pt.GetConfig(), &fn); err != nil {
				return fmt.Errorf("error unmarshalling partition: %w", err)
			}
		}

		var shard *OrderedMap
		if pt.GetQueueShard() != nil {
			s := pt.GetQueueShard()
			shard = OrderedData(
				"Name", s.GetName(),
				"Kind", s.GetKind(),
			)
		}

		if err := w.WriteOrdered(OrderedData(
			"Type", "Partition",
			"ID", pt.Id,
			"Slug", pt.Slug,
			"Tenant", OrderedData(
				"Account", pt.Tenant.AccountId,
				"Environment", pt.Tenant.EnvId,
				"App", pt.Tenant.AppId,
				"Queue Shard", shard,
			),
			"Triggers", fn.Triggers,
			"Concurrency", OrderedData(
				"Account", 0,
				"Function", 0,
			),
			"Configuration", OrderedData(
				"Name", fn.Name,
				"Version", fn.FunctionVersion,
				"Priority", fn.Priority,
				"Timeouts", fn.Timeouts,
				"Concurrency", fn.Concurrency,
				"Debounce", fn.Debounce,
				"Batching", fn.EventBatch,
				"RateLimit", fn.RateLimit,
				"Throttle", fn.Throttle,
				"Cancel", fn.Cancel,
				"Singleton", fn.Singleton,
				"URI", fn.Steps,
			),
		),
			WithTextOptLeadSpace(true),
		); err != nil {
			return err
		}
	}

	// Status
	if pts != nil {
		if err := w.WriteOrdered(OrderedData(
			"Paused", pts.Paused,
			"Migrate", pts.Migrate,
		)); err != nil {
			return err
		}
		if err := w.WriteOrdered(OrderedData(
			"Account Active", pts.AccountActive,
			"Account In-progress", pts.AccountInProgress,
			"Ready", pts.Ready,
			"In-progress", pts.InProgress,
			"Active", pts.Active,
			"Future", pts.Future,
			"Backlogs", pts.Backlogs,
		)); err != nil {
			return err
		}
	}

	return w.Flush()
}

func TextQueueItem(item *queue.QueueItem) error {
	if item == nil {
		fmt.Println("no item found")
		return nil
	}

	w := NewTextWriter()

	data := item.Data

	if err := w.WriteOrdered(OrderedData(
		"ID", item.ID,
		"EarliestPeekTime", item.EarliestPeekTime,
		"At", time.UnixMilli(item.AtMS).Format(time.RFC3339),
		"WallTime", time.Duration(item.WallTimeMS)*time.Millisecond,
		"WorkspaceID", item.WorkspaceID,
		"FunctionID", item.FunctionID,
		"LeaseID", item.LeaseID,
		"QueueName", item.QueueName,
		"IdempotencyPeriod", item.IdempotencyPeriod,
		"RefilledFrom", item.RefilledFrom,
		"RefilledAt", time.UnixMilli(item.RefilledAt).Format(time.RFC3339),
		"EnqueuedAt", time.UnixMilli(item.EnqueuedAt).Format(time.RFC3339),
		"Data", OrderedData(
			"JobID", data.JobID,
			"GroupID", data.GroupID,
			"Kind", data.Kind,
			"Identifier", OrderedData(
				"AccountID", data.Identifier.AccountID,
				"WorkspaceID", data.Identifier.WorkspaceID,
				"AppID", data.Identifier.AppID,
				"FunctionID", data.Identifier.WorkflowID,
				"FunctionVersion", data.Identifier.WorkflowVersion,
				"EventIDs", data.Identifier.EventIDs,
				"RunID", data.Identifier.RunID,
				"Key", data.Identifier.Key,
				"ReplayID", data.Identifier.ReplayID,
				"OriginalRunID", data.Identifier.OriginalRunID,
				"PriorityFactor", data.Identifier.PriorityFactor,
			),
			"Attempt", data.Attempt,
			"MaxAttempts", data.GetMaxAttempts(),
			"Payload", data.Payload,
			"Metadata", data.Metadata,
			"ParallelMode", data.ParallelMode,
		),
	), WithTextOptLeadSpace(true)); err != nil {
		return err
	}

	return w.Flush()
}
