package pause

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

func BSelectCommand() *cli.Command {
	return &cli.Command{
		Name:    "bselect",
		Aliases: []string{"bs"},
		Usage:   "Select pause IDs from a block",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "event",
				Aliases: []string{"ev"},
				Usage:   "The event name for the pause index",
			},
			&cli.StringFlag{
				Name:    "workspace-id",
				Aliases: []string{"ws"},
				Usage:   "The workspace id for the pause index",
			},
			&cli.StringFlag{
				Name:    "block-id",
				Aliases: []string{"bid"},
				Usage:   "The block ID to select from (ULID)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			eventName := cmd.String("event")
			workspaceID := cmd.String("workspace-id")
			blockID := cmd.String("block-id")

			if eventName == "" || workspaceID == "" || blockID == "" {
				return fmt.Errorf("--event / --workspace-id / --block-id required")
			}

			dbgCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			req := pb.BlockPeekRequest{
				EventName:   eventName,
				WorkspaceId: workspaceID,
				BlockId:     blockID,
			}

			resp, err := dbgCtx.Client.BlockPeek(ctx, &req)
			if err != nil {
				st, ok := grpcStatus.FromError(err)
				if !ok {
					return fmt.Errorf("failed to select from block: %w", err)
				}

				switch st.Code() {
				case codes.NotFound:
					fmt.Println("block not found")
					return nil
				case codes.InvalidArgument:
					return fmt.Errorf("invalid arguments: %s", st.Message())
				}

				return fmt.Errorf("failed to select from block: %w", err)
			}

			return output.TextBlockPeek(resp)
		},
	}
}

