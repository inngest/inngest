package singleton

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
		Usage:   "Delete a singleton lock",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "singleton-key",
				Aliases:  []string{"key"},
				Usage:    "The singleton key (function_id-hash or just function_id)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "account-id",
				Aliases:  []string{"acct"},
				Usage:    "The account ID for shard selection",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			singletonKey := cmd.String("singleton-key")
			accountID := cmd.String("account-id")

			dbgCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			req := &pb.DeleteSingletonLockRequest{
				SingletonKey: singletonKey,
				AccountId:    accountID,
			}

			resp, err := dbgCtx.Client.DeleteSingletonLock(ctx, req)
			if err != nil {
				st, ok := grpcStatus.FromError(err)
				if !ok {
					return fmt.Errorf("failed to delete singleton lock: %w", err)
				}

				switch st.Code() {
				case codes.NotFound:
					fmt.Println("no singleton lock found to delete")
					return nil
				}

				return fmt.Errorf("failed to delete singleton lock: %w", err)
			}

			return output.TextDeleteSingletonLock(resp)
		},
	}
}
