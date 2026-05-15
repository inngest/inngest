package batch

import (
	"context"
	"fmt"

	debugpkg "github.com/inngest/inngest/pkg/debug"
	dbgpb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/urfave/cli/v3"
)

func DeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Aliases:   []string{"d", "rm"},
		Usage:     "Delete a batch for a function",
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

			req := &dbgpb.DeleteBatchRequest{
				FunctionId: functionID,
				BatchKey:   batchKey,
			}

			resp, err := debugCtx.Client.DeleteBatch(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to delete batch: %w", err)
			}

			if !resp.Deleted {
				fmt.Println("No batch found to delete")
				return nil
			}

			fmt.Printf("Deleted batch: %s\n", resp.BatchId)
			fmt.Printf("Items removed: %d\n", resp.ItemCount)

			return nil
		},
	}
}
