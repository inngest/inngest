package output

import (
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/inngest"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

func Partition(pt *pb.PartitionResponse, pts *pb.PartitionStatusResponse) error {
	w := NewTextWriter()

	if pt != nil {
		var fn inngest.Function
		if len(pt.GetConfig()) > 0 {
			if err := json.Unmarshal(pt.GetConfig(), &fn); err != nil {
				return fmt.Errorf("error unmarshalling partition: %w", err)
			}
		}

		if err := w.WriteOrdered(OrderedData(
			"Type", "Partition",
			"ID", pt.Id,
			"Slug", pt.Slug,
			"Tenant", OrderedData(
				"Account", pt.Tenant.AccountId,
				"Environment", pt.Tenant.EnvId,
				"App", pt.Tenant.AppId,
				"Queue Shard", "TODO",
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
			"PauseEnqueue", pts.PauseEnqueue,
		)); err != nil {
			return err
		}
		if err := w.WriteOrdered(OrderedData(
			"PauseRefill", pts.PauseRefill,
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
