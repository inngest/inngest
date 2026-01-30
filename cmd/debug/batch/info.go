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

			if resp.BatchId == "" {
				fmt.Println("No active batch found")
				return nil
			}

			fmt.Printf("Batch ID:    %s\n", resp.BatchId)
			fmt.Printf("Status:      %s\n", resp.Status)
			fmt.Printf("Item Count:  %d\n", resp.ItemCount)

			if len(resp.Items) > 0 {
				fmt.Println("\nBatch Items:")
				for i, item := range resp.Items {
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

			return nil
		},
	}
}
