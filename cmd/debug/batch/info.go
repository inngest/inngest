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

func InfoCommand() *cli.Command {
	return &cli.Command{
		Name:    "info",
		Aliases: []string{"i"},
		Usage:   "Get batch information for a function",
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
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			functionID := cmd.String("function-id")
			batchKey := cmd.String("batch-key")
			accountID := cmd.String("account-id")

			dbgCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			req := &pb.BatchInfoRequest{
				FunctionId: functionID,
				BatchKey:   batchKey,
				AccountId:  accountID,
			}

			resp, err := dbgCtx.Client.GetBatchInfo(ctx, req)
			if err != nil {
				st, ok := grpcStatus.FromError(err)
				if !ok {
					return fmt.Errorf("failed to retrieve batch info: %w", err)
				}

				switch st.Code() {
				case codes.NotFound:
					fmt.Println("no batch found")
					return nil
				}

				return fmt.Errorf("failed to retrieve batch info: %w", err)
			}

			return output.TextBatchInfo(resp)
		},
	}
}
