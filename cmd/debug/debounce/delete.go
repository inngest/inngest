package debounce

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

func DeleteCommand() *cli.Command {
	return &cli.Command{
		Name:    "delete",
		Aliases: []string{"d"},
		Usage:   "Delete a debounce for a function",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "function-id",
				Aliases:  []string{"fn"},
				Usage:    "The function ID (UUID)",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "debounce-key",
				Aliases: []string{"key"},
				Usage:   "The debounce key (from the key expression or function_id)",
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
			debounceKey := cmd.String("debounce-key")
			accountID := cmd.String("account-id")

			dbgCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			req := &pb.DeleteDebounceRequest{
				FunctionId:  functionID,
				DebounceKey: debounceKey,
				AccountId:   accountID,
			}

			resp, err := dbgCtx.Client.DeleteDebounce(ctx, req)
			if err != nil {
				st, ok := grpcStatus.FromError(err)
				if !ok {
					return fmt.Errorf("failed to delete debounce: %w", err)
				}

				switch st.Code() {
				case codes.NotFound:
					fmt.Println("no debounce found to delete")
					return nil
				}

				return fmt.Errorf("failed to delete debounce: %w", err)
			}

			return output.TextDeleteDebounce(resp)
		},
	}
}
