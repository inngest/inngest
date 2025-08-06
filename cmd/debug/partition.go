package debug

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/cli/output"
	dbgpb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/urfave/cli/v3"
	"google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"
)

func partitionCommand() *cli.Command {
	return &cli.Command{
		Name:      "partition",
		Aliases:   []string{"pt"},
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
				st, ok := grpcStatus.FromError(err)
				if !ok {
					return fmt.Errorf("failed to get partition status: %w", err)

				}

				switch st.Code() {
				case codes.NotFound:
					// no-op
				}
			}

			return output.Partition(partition, status)
		},
	}
}
