package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/cli/output"
	debugpkg "github.com/inngest/inngest/pkg/debug"
	"github.com/inngest/inngest/pkg/execution/queue"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/urfave/cli/v3"
	"google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"
)

func ItemCommand() *cli.Command {
	return &cli.Command{
		Name:    "item",
		Aliases: []string{"i"},
		Usage:   "Get queue item data",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "id",
				Usage:    "queue item ID to reference",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "run-id",
				Usage:    "run ID to reference",
				Required: false,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			itemID := cmd.String("id")
			runID := cmd.String("run-id")

			if itemID == "" && runID == "" {
				return fmt.Errorf("either --id or --run-id is required")
			}

			// Use itemID if provided, otherwise use runID
			queryID := itemID
			if queryID == "" {
				queryID = runID
			}

			dbgCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			resp, err := dbgCtx.Client.GetQueueItem(ctx, &pb.QueueItemRequest{Id: queryID})
			if err != nil {
				st, ok := grpcStatus.FromError(err)
				if !ok {
					return fmt.Errorf("failed to retrieve queue item: %w", err)
				}

				switch st.Code() {
				case codes.NotFound:
					fmt.Println("no queue item", queryID)
					return nil
				}

				return fmt.Errorf("failed to retrieve queue item: %w", err)
			}

			var item queue.QueueItem
			if err := json.Unmarshal(resp.GetData(), &item); err != nil {
				return fmt.Errorf("error unmarshalling item: %w", err)
			}

			return output.TextQueueItem(&item)
		},
	}
}
