package constraintapi

import (
	"context"
	"fmt"

	debugpkg "github.com/inngest/inngest/pkg/debug"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/urfave/cli/v3"
)

func CountAccountsCommand() *cli.Command {
	return &cli.Command{
		Name:    "count-accounts",
		Aliases: []string{"ca"},
		Usage:   "Count accounts in the top-level scavenger zset",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			dbgCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			resp, err := dbgCtx.Client.CountAccounts(ctx, &pb.CountAccountsRequest{})
			if err != nil {
				return fmt.Errorf("failed to count accounts: %w", err)
			}

			fmt.Printf("Account count: %d\n", resp.Count)
			return nil
		},
	}
}
