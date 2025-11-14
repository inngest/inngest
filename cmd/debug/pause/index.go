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

func DescribeCommand() *cli.Command {
	return &cli.Command{
		Name:    "describe",
		Aliases: []string{"d"},
		Usage:   "Get pause index block information",
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
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			eventName := cmd.String("event")
			workspaceID := cmd.String("workspace-id")

			if eventName == "" || workspaceID == "" {
				return fmt.Errorf("--event / --workspace-id required")
			}

			dbgCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			req := pb.IndexRequest{
				EventName:   eventName,
				WorkspaceId: workspaceID,
			}

			resp, err := dbgCtx.Client.GetIndex(ctx, &req)
			if err != nil {
				st, ok := grpcStatus.FromError(err)
				if !ok {
					return fmt.Errorf("failed to retrieve index: %w", err)
				}

				switch st.Code() {
				case codes.NotFound:
					fmt.Println("no index found")
					return nil
				case codes.InvalidArgument:
					return fmt.Errorf("invalid arguments: %s", st.Message())
				}

				return fmt.Errorf("failed to retrieve index: %w", err)
			}

			return output.TextIndex(resp)
		},
	}
}

