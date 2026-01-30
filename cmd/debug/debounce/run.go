package debounce

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
		Usage:     "Execute a pending debounce immediately",
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

			req := &dbgpb.RunDebounceRequest{
				FunctionId:  functionID,
				DebounceKey: debounceKey,
			}

			resp, err := debugCtx.Client.RunDebounce(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to run debounce: %w", err)
			}

			if !resp.Scheduled {
				fmt.Println("No debounce found to run")
				return nil
			}

			fmt.Printf("Scheduled debounce: %s\n", resp.DebounceId)
			fmt.Printf("Event ID:           %s\n", resp.EventId)

			return nil
		},
	}
}
