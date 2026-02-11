package singleton

import (
	"context"
	"fmt"

	debugpkg "github.com/inngest/inngest/pkg/debug"
	dbgpb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/urfave/cli/v3"
)

func InfoCommand() *cli.Command {
	return &cli.Command{
		Name:      "info",
		Aliases:   []string{"i"},
		Usage:     "Get singleton lock information for a function",
		ArgsUsage: "<function-uuid>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "key",
				Aliases: []string{"k"},
				Value:   "",
				Usage:   "Singleton key suffix (optional, for keyed singletons)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("function UUID is required")
			}

			functionID := cmd.Args().Get(0)
			singletonKey := cmd.String("key")

			debugCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			req := &dbgpb.SingletonInfoRequest{
				FunctionId:   functionID,
				SingletonKey: singletonKey,
			}

			resp, err := debugCtx.Client.GetSingletonInfo(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to get singleton info: %w", err)
			}

			if !resp.HasLock {
				fmt.Println("No active singleton lock")
				return nil
			}

			fmt.Printf("Lock held:     yes\n")
			fmt.Printf("Current Run:   %s\n", resp.CurrentRunId)

			return nil
		},
	}
}
