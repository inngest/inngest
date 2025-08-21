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
		Name:      "item",
		Aliases:   []string{"i"},
		Usage:     "Get queue item data",
		ArgsUsage: "<item-id>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() == 0 {
				return fmt.Errorf("item ID is required")
			}

			itemID := cmd.Args().Get(0)

			dbgCtx, ok := ctx.Value(debugpkg.CtxKey).(*debugpkg.Context)
			if !ok {
				return fmt.Errorf("debug context not found")
			}

			resp, err := dbgCtx.Client.GetQueueItem(ctx, &pb.QueueItemRequest{Id: itemID})
			if err != nil {
				st, ok := grpcStatus.FromError(err)
				if !ok {
					return fmt.Errorf("failed to retrieve queue item: %w", err)
				}

				switch st.Code() {
				case codes.NotFound:
					fmt.Println("no queue item", itemID)
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
