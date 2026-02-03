package debounce

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
		Usage:     "Delete a debounce for a function",
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

			req := &dbgpb.DeleteDebounceRequest{
				FunctionId:  functionID,
				DebounceKey: debounceKey,
			}

			resp, err := debugCtx.Client.DeleteDebounce(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to delete debounce: %w", err)
			}

			if !resp.Deleted {
				fmt.Println("No debounce found to delete")
				return nil
			}

			fmt.Printf("Deleted debounce: %s\n", resp.DebounceId)
			fmt.Printf("Event ID:         %s\n", resp.EventId)

			return nil
		},
	}
}
