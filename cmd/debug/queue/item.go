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
				Name:  "id",
				Usage: "queue item ID to reference (does not work with --run-id)",
			},
			&cli.StringFlag{
				Name:  "run-id",
				Usage: "run ID to reference (does not work with --id)",
			},
			&cli.StringFlag{
				Name:    "queue-shard",
				Aliases: []string{"qs"},
				Usage:   "The queue shard to specify",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			itemID := cmd.String("id")
			runID := cmd.String("run-id")

			if itemID == "" && runID == "" {
				return fmt.Errorf("either --id or --run-id is required")
			}

			if itemID != "" && runID != "" {
				return fmt.Errorf("--id and --run-id are mutually exclusive")
			}

			dbgCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			// Use itemID if provided, otherwise use runID
			req := pb.QueueItemRequest{
				QueueShard: cmd.String("queue-shard"),
			}
			if itemID != "" {
				req.ItemId = itemID
			} else {
				req.RunId = runID
			}

			resp, err := dbgCtx.Client.GetQueueItem(ctx, &req)
			if err != nil {
				st, ok := grpcStatus.FromError(err)
				if !ok {
					return fmt.Errorf("failed to retrieve queue item: %w", err)
				}

				switch st.Code() {
				case codes.NotFound:
					fmt.Println("no queue item found")
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
