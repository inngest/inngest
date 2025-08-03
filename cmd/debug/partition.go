package debug

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/cmd/internal/table"
	dbgpb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/urfave/cli/v3"
)

func partitionCommand() *cli.Command {
	return &cli.Command{
		Name:      "partition",
		Aliases:   []string{"p"},
		Usage:     "Get partition information and status",
		ArgsUsage: "<partition-uuid>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("partition UUID is required")
			}

			partitionID := cmd.Args().Get(0)

			debugCtx, ok := ctx.Value(dbgCtxKey).(*DebugContext)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			// Get partition information
			partitionReq := &dbgpb.PartitionRequest{Id: partitionID}

			partition, err := debugCtx.Client.GetPartition(ctx, partitionReq)
			if err != nil {
				return fmt.Errorf("failed to get partition: %w", err)
			}

			// Get partition status
			status, err := debugCtx.Client.GetPartitionStatus(ctx, partitionReq)
			if err != nil {
				return fmt.Errorf("failed to get partition status: %w", err)
			}

			// Display partition information
			fmt.Printf("Partition: %s\n", partition.Id)
			fmt.Printf("Slug: %s\n", partition.Slug)
			if partition.Tenant != nil {
				fmt.Printf("Account ID: %s\n", partition.Tenant.AccountId)
				fmt.Printf("Environment ID: %s\n", partition.Tenant.EnvId)
				fmt.Printf("App ID: %s\n", partition.Tenant.AppId)
			}
			fmt.Printf("Version: %d\n", partition.Version)

			// Display partition status in a table
			statusTable := table.New(table.Row{"Property", "Value"})
			statusTable.AppendRow(table.Row{"Paused", fmt.Sprintf("%t", status.Paused)})
			statusTable.AppendRow(table.Row{"Migrate", fmt.Sprintf("%t", status.Migrate)})
			statusTable.AppendRow(table.Row{"Pause Refill", fmt.Sprintf("%t", status.PauseRefill)})
			statusTable.AppendRow(table.Row{"Pause Enqueue", fmt.Sprintf("%t", status.PauseEnqueue)})

			fmt.Println("\nStatus:")
			statusTable.Render()

			// Display queue metrics in a table
			metricsTable := table.New(table.Row{"Metric", "Count"})
			metricsTable.AppendRow(table.Row{"Account Active", fmt.Sprintf("%d", status.AccountActive)})
			metricsTable.AppendRow(table.Row{"Account In Progress", fmt.Sprintf("%d", status.AccountInProgress)})
			metricsTable.AppendRow(table.Row{"Ready", fmt.Sprintf("%d", status.Ready)})
			metricsTable.AppendRow(table.Row{"In Progress", fmt.Sprintf("%d", status.InProgress)})
			metricsTable.AppendRow(table.Row{"Active", fmt.Sprintf("%d", status.Active)})
			metricsTable.AppendRow(table.Row{"Future", fmt.Sprintf("%d", status.Future)})
			metricsTable.AppendRow(table.Row{"Backlogs", fmt.Sprintf("%d", status.Backlogs)})

			fmt.Println("Queue Metrics:")
			metricsTable.Render()

			return nil
		},
	}
}
