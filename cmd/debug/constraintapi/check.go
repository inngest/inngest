package constraintapi

import (
	"context"
	"fmt"

	cpb "github.com/inngest/inngest/proto/gen/constraintapi/v1"

	"github.com/inngest/inngest/pkg/cli/output"
	debugpkg "github.com/inngest/inngest/pkg/debug"
	"github.com/urfave/cli/v3"
)

// Common flags shared across Constraint API commands
var (
	accountFlag = &cli.StringFlag{
		Name:    "account-id",
		Aliases: []string{"acc"},
		Usage:   "The account id",
	}
	workspaceFlag = &cli.StringFlag{
		Name:    "workspace-id",
		Aliases: []string{"ws"},
		Usage:   "The workspace id",
	}
	functionFlag = &cli.StringFlag{
		Name:    "function-id",
		Aliases: []string{"fn"},
		Usage:   "The function id",
	}
)

func CheckCommand() *cli.Command {
	return &cli.Command{
		Name:    "check",
		Aliases: []string{"c"},
		Usage:   "Check constraints",
		Flags: []cli.Flag{
			accountFlag,
			workspaceFlag,
			functionFlag,
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			workspaceID := cmd.String("workspace-id")
			accountID := cmd.String("account-id")
			functionID := cmd.String("function-id")

			dbgCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			req := &cpb.CapacityCheckRequest{
				AccountId:  accountID,
				EnvId:      workspaceID,
				FunctionId: functionID,
			}

			resp, err := dbgCtx.Client.CheckConstraints(ctx, req)
			if err != nil {
				return fmt.Errorf("failed to check constraints: %w", err)
			}

			return output.TextCheckConstraints(resp)
		},
	}
}
