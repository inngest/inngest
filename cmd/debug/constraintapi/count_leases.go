package constraintapi

import (
	"context"
	"fmt"

	debugpkg "github.com/inngest/inngest/pkg/debug"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/urfave/cli/v3"
)

func CountAccountLeasesCommand() *cli.Command {
	return &cli.Command{
		Name:    "count-leases",
		Aliases: []string{"cl"},
		Usage:   "Count items in an account's leaseq zset",
		Flags: []cli.Flag{
			accountFlag,
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			accountID := cmd.String("account-id")
			if accountID == "" {
				return fmt.Errorf("account-id is required")
			}

			dbgCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			resp, err := dbgCtx.Client.CountAccountLeases(ctx, &pb.CountAccountLeasesRequest{
				AccountId: accountID,
			})
			if err != nil {
				return fmt.Errorf("failed to count account leases: %w", err)
			}

			fmt.Printf("Lease count: %d\n", resp.Count)
			return nil
		},
	}
}
