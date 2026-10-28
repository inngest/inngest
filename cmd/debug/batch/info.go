package batch

import (
	"context"
	"encoding/json"
	"fmt"

	debugpkg "github.com/inngest/inngest/pkg/debug"
	dbgpb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/urfave/cli/v3"
)

func InfoCommand() *cli.Command {
	return &cli.Command{
		Name:      "info",
		Aliases:   []string{"i"},
		Usage:     "Get batch information for a function",
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

			req := &dbgpb.BatchInfoRequest{
				FunctionId: functionID,
				BatchKey:   batchKey,
			}

			resp, err := debugCtx.Client.GetBatchInfo(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to get batch info: %w", err)
			}

			// Older servers only fill the top-level fields.
			stores := resp.Stores
			if len(stores) == 0 {
				stores = []*dbgpb.BatchStoreInfo{{
					BatchId:   resp.BatchId,
					ItemCount: resp.ItemCount,
					Items:     resp.Items,
					Status:    resp.Status,
				}}
			}

			for i, store := range stores {
				if i > 0 {
					fmt.Println()
				}
				if store.Store != "" {
					fmt.Printf("Store:       %s\n", store.Store)
				}
				printBatchInfo(store)
			}

			return nil
		},
	}
}

func printBatchInfo(store *dbgpb.BatchStoreInfo) {
	if store.BatchId == "" {
		fmt.Println("No active batch found")
		return
	}

	fmt.Printf("Batch ID:    %s\n", store.BatchId)
	fmt.Printf("Status:      %s\n", store.Status)
	fmt.Printf("Item Count:  %d\n", store.ItemCount)

	if len(store.Items) > 0 {
		fmt.Println("\nBatch Items:")
		for i, item := range store.Items {
			fmt.Printf("  [%d] Event ID: %s\n", i+1, item.EventId)
			if len(item.EventData) > 0 {
				var eventData map[string]any
				if err := json.Unmarshal(item.EventData, &eventData); err == nil {
					prettyJSON, _ := json.MarshalIndent(eventData, "      ", "  ")
					fmt.Printf("      Data: %s\n", string(prettyJSON))
				}
			}
		}
	}
}
