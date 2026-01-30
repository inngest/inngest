package debounce

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
		Usage:     "Get debounce information for a function",
		ArgsUsage: "<function-uuid>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "key",
				Aliases: []string{"k"},
				Value:   "",
				Usage:   "Debounce key (optional, for keyed debounces)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("function UUID is required")
			}

			functionID := cmd.Args().Get(0)
			debounceKey := cmd.String("key")

			debugCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			req := &dbgpb.DebounceInfoRequest{
				FunctionId:  functionID,
				DebounceKey: debounceKey,
			}

			resp, err := debugCtx.Client.GetDebounceInfo(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to get debounce info: %w", err)
			}

			if !resp.HasDebounce {
				fmt.Println("No active debounce found")
				return nil
			}

			fmt.Printf("Debounce ID:   %s\n", resp.DebounceId)
			fmt.Printf("Event ID:      %s\n", resp.EventId)
			fmt.Printf("Function ID:   %s\n", resp.FunctionId)
			fmt.Printf("Timeout:       %d ms\n", resp.Timeout)

			if len(resp.EventData) > 0 {
				var eventData map[string]any
				if err := json.Unmarshal(resp.EventData, &eventData); err == nil {
					prettyJSON, _ := json.MarshalIndent(eventData, "", "  ")
					fmt.Printf("\nEvent Data:\n%s\n", string(prettyJSON))
				}
			}

			return nil
		},
	}
}
