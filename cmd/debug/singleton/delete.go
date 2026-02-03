package singleton

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
		Usage:     "Delete a singleton lock for a function",
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

			req := &dbgpb.DeleteSingletonLockRequest{
				FunctionId:   functionID,
				SingletonKey: singletonKey,
			}

			resp, err := debugCtx.Client.DeleteSingletonLock(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to delete singleton lock: %w", err)
			}

			if !resp.Deleted {
				fmt.Println("No singleton lock found to delete")
				return nil
			}

			fmt.Printf("Deleted singleton lock\n")
			fmt.Printf("Previous run ID: %s\n", resp.RunId)

			return nil
		},
	}
}
