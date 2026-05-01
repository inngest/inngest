package constraintapi

import (
	"context"
	"fmt"

	debugpkg "github.com/inngest/inngest/pkg/debug"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/urfave/cli/v3"
)

func GetFunctionConcurrencyCommand() *cli.Command {
	return &cli.Command{
		Name:    "function-concurrency",
		Aliases: []string{"fc"},
		Usage:   "Get in-progress concurrency count for a function",
		Flags: []cli.Flag{
			accountFlag,
			functionFlag,
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			accountID := cmd.String("account-id")
			if accountID == "" {
				return fmt.Errorf("account-id is required")
			}

			functionID := cmd.String("function-id")
			if functionID == "" {
				return fmt.Errorf("function-id is required")
			}

			dbgCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			resp, err := dbgCtx.Client.GetFunctionConcurrency(ctx, &pb.GetFunctionConcurrencyRequest{
				AccountId:  accountID,
				FunctionId: functionID,
			})
			if err != nil {
				return fmt.Errorf("failed to get function concurrency: %w", err)
			}

			fmt.Printf("In-progress: %d\n", resp.InProgress)
			return nil
		},
	}
}
