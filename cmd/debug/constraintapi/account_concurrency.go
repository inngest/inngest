package constraintapi

import (
	"context"
	"fmt"

	debugpkg "github.com/inngest/inngest/pkg/debug"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/urfave/cli/v3"
)

func GetAccountConcurrencyCommand() *cli.Command {
	return &cli.Command{
		Name:    "account-concurrency",
		Aliases: []string{"ac"},
		Usage:   "Get in-progress concurrency count for an account",
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

			resp, err := dbgCtx.Client.GetAccountConcurrency(ctx, &pb.GetAccountConcurrencyRequest{
				AccountId: accountID,
			})
			if err != nil {
				return fmt.Errorf("failed to get account concurrency: %w", err)
			}

			fmt.Printf("In-progress: %d\n", resp.InProgress)
			return nil
		},
	}
}
