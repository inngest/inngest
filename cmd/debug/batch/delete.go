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

			// Older servers only fill the top-level fields.
			stores := resp.Stores
			if len(stores) == 0 {
				stores = []*dbgpb.DeleteBatchStoreResult{{
					Deleted:   resp.Deleted,
					BatchId:   resp.BatchId,
					ItemCount: resp.ItemCount,
				}}
			}

			for i, store := range stores {
				if i > 0 {
					fmt.Println()
				}
				if store.Store != "" {
					fmt.Printf("Store:         %s\n", store.Store)
				}
				if !store.Deleted {
					fmt.Println("No batch found to delete")
					continue
				}
				fmt.Printf("Deleted batch: %s\n", store.BatchId)
				fmt.Printf("Items removed: %d\n", store.ItemCount)
			}

			return nil
		},
	}
}
