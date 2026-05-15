package batch

import (
	"context"
	"fmt"

	debugpkg "github.com/inngest/inngest/pkg/debug"
	dbgpb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/urfave/cli/v3"
)

func RunCommand() *cli.Command {
	return &cli.Command{
		Name:      "run",
		Aliases:   []string{"r"},
		Usage:     "Execute a pending batch immediately",
		ArgsUsage: "<function-uuid>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "key",
				Aliases: []string{"k"},
				Value:   "",
				Usage:   "Batch key (defaults to 'default' if not specified)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("function UUID is required")
			}

			functionID := cmd.Args().Get(0)
			batchKey := cmd.String("key")

			debugCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			req := &dbgpb.RunBatchRequest{
				FunctionId: functionID,
				BatchKey:   batchKey,
			}

			resp, err := debugCtx.Client.RunBatch(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to run batch: %w", err)
			}

			if !resp.Scheduled {
				fmt.Println("No batch found to run")
				return nil
			}

			fmt.Printf("Scheduled batch: %s\n", resp.BatchId)
			fmt.Printf("Items in batch:  %d\n", resp.ItemCount)

			return nil
		},
	}
}
