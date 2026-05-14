package debounce

import (
	"context"
	"fmt"
	"strings"

	"github.com/inngest/inngest/cmd/debug/debugflags"
	debugpkg "github.com/inngest/inngest/pkg/debug"
	dbgpb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/urfave/cli/v3"
)

func DeleteByIDCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete-by-id",
		Aliases:   []string{"rmid"},
		Usage:     "Delete debounces directly by their IDs",
		ArgsUsage: "<debounce-id> [debounce-id...]",
		Flags: append(
			debugflags.AccountEnvFlags(),
			debugflags.FunctionFlag(),
		),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("at least one debounce ID is required")
			}

			ids := make([]string, cmd.NArg())
			for i := 0; i < cmd.NArg(); i++ {
				ids[i] = cmd.Args().Get(i)
			}

			accountID, envID, err := debugflags.AccountEnv(cmd)
			if err != nil {
				return err
			}

			functionID, err := debugflags.RequiredUUID(cmd, "function-id")
			if err != nil {
				return err
			}

			debugCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			req := &dbgpb.DeleteDebounceByIDRequest{
				DebounceIds: ids,
				AccountId:   accountID,
				EnvId:       envID,
				FunctionId:  functionID,
			}

			resp, err := debugCtx.Client.DeleteDebounceByID(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to delete debounces by ID: %w", err)
			}

			if len(resp.DeletedIds) == 0 {
				fmt.Println("No debounces deleted")
				return nil
			}

			fmt.Printf("Deleted count:     %d\n", len(resp.DeletedIds))
			fmt.Printf("Deleted debounces: %s\n", strings.Join(resp.DeletedIds, ", "))

			return nil
		},
	}
}
