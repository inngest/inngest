package pause

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/cli/output"
	debugpkg "github.com/inngest/inngest/pkg/debug"
	"github.com/inngest/inngest/pkg/execution/state"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/urfave/cli/v3"
	"google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"
)

func PauseCommand() *cli.Command {
	return &cli.Command{
		Name:    "item",
		Aliases: []string{"i"},
		Usage:   "Get pause data",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "id",
				Usage: "pause item id to reference",
			},
			&cli.StringFlag{
				Name:    "event",
				Aliases: []string{"ev"},
				Usage:   "The event name that the pause is related to",
			},
			&cli.StringFlag{
				Name:    "workspace-id",
				Aliases: []string{"ws"},
				Usage:   "The workspace id that the pause belongs to",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			itemID := cmd.String("id")
			eventName := cmd.String("event")
			workspaceID := cmd.String("workspace-id")

			if itemID == "" || eventName == "" || workspaceID == "" {
				return fmt.Errorf("--id / --event / --workspace-id  required")
			}

			dbgCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			req := pb.PauseRequest{
				ItemId:      itemID,
				EventName:   eventName,
				WorkspaceId: workspaceID,
			}

			resp, err := dbgCtx.Client.GetPause(ctx, &req)
			if err != nil {
				st, ok := grpcStatus.FromError(err)
				if !ok {
					return fmt.Errorf("failed to retrieve queue item: %w", err)
				}

				switch st.Code() {
				case codes.NotFound:
					fmt.Println("no pause found")
					return nil
				}

				return fmt.Errorf("failed to retrieve queue item: %w", err)
			}

			var item state.Pause
			if err := json.Unmarshal(resp.GetData(), &item); err != nil {
				return fmt.Errorf("error unmarshalling item: %w", err)
			}

			return output.TextPause(&item)
		},
	}
}
