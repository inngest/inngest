package batch

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/pkg/cli/output"
	debugpkg "github.com/inngest/inngest/pkg/debug"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/urfave/cli/v3"
	"google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"
)

func RunCommand() *cli.Command {
	return &cli.Command{
		Name:    "run",
		Aliases: []string{"r"},
		Usage:   "Trigger immediate execution of a batch",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "function-id",
				Aliases:  []string{"fn"},
				Usage:    "The function ID (UUID)",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "batch-key",
				Aliases: []string{"key"},
				Usage:   "The batch key (defaults to 'default' if not specified)",
			},
			&cli.StringFlag{
				Name:     "account-id",
				Aliases:  []string{"acct"},
				Usage:    "The account ID for shard selection",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "workspace-id",
				Aliases:  []string{"ws"},
				Usage:    "The workspace ID",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "app-id",
				Aliases:  []string{"app"},
				Usage:    "The app ID",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			functionID := cmd.String("function-id")
			batchKey := cmd.String("batch-key")
			accountID := cmd.String("account-id")
			workspaceID := cmd.String("workspace-id")
			appID := cmd.String("app-id")

			dbgCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			req := &pb.RunBatchRequest{
				FunctionId:  functionID,
				BatchKey:    batchKey,
				AccountId:   accountID,
				WorkspaceId: workspaceID,
				AppId:       appID,
			}

			resp, err := dbgCtx.Client.RunBatch(ctx, req)
			if err != nil {
				st, ok := grpcStatus.FromError(err)
				if !ok {
					return fmt.Errorf("failed to run batch: %w", err)
				}

				switch st.Code() {
				case codes.NotFound:
					fmt.Println("no batch found to run")
					return nil
				}

				return fmt.Errorf("failed to run batch: %w", err)
			}

			return output.TextRunBatch(resp)
		},
	}
}
