package output

import (
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

func PartitionTable(pt *pb.PartitionResponse, pts *pb.PartitionStatusResponse) error {
	w := NewTextWriter()

	if pt != nil {
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
			"Concurrency", OrderedData(
				"Account", 0,
				"Function", 0,
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
