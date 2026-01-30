package output

import (
	"encoding/json"
	"fmt"

	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

func TextBatchInfo(resp *pb.BatchInfoResponse) error {
	if resp == nil {
		fmt.Println("no batch info found")
		return nil
	}

	w := NewTextWriter()

	if resp.BatchId == "" {
		fmt.Println("no active batch found")
		return nil
	}

	if err := w.WriteOrdered(OrderedData(
		"BatchID", resp.BatchId,
		"Status", resp.Status,
		"ItemCount", resp.ItemCount,
	), WithTextOptLeadSpace(true)); err != nil {
		return err
	}

	if err := w.Flush(); err != nil {
		return err
	}

	// Write batch items
	if len(resp.Items) > 0 {
		fmt.Printf("\nBatch Items:\n")

		for i, item := range resp.Items {
			fmt.Printf("\n  Item %d:\n", i+1)
			fmt.Printf("    EventID:         %s\n", item.EventId)
			fmt.Printf("    AccountID:       %s\n", item.AccountId)
			fmt.Printf("    WorkspaceID:     %s\n", item.WorkspaceId)
			fmt.Printf("    AppID:           %s\n", item.AppId)
			fmt.Printf("    FunctionID:      %s\n", item.FunctionId)
			fmt.Printf("    FunctionVersion: %d\n", item.FunctionVersion)

			if len(item.EventData) > 0 {
				var eventData any
				if err := json.Unmarshal(item.EventData, &eventData); err == nil {
					prettyJSON, _ := json.MarshalIndent(eventData, "    ", "  ")
					fmt.Printf("    EventData:       %s\n", string(prettyJSON))
				} else {
					fmt.Printf("    EventData:       %s\n", string(item.EventData))
				}
			}
		}
	} else {
		fmt.Printf("\nNo batch items found\n")
	}

	return nil
}

func TextDeleteBatch(resp *pb.DeleteBatchResponse) error {
	if resp == nil {
		fmt.Println("no response received")
		return nil
	}

	w := NewTextWriter()

	if !resp.Deleted {
		fmt.Println("no batch was deleted")
		return nil
	}

	if err := w.WriteOrdered(OrderedData(
		"Deleted", resp.Deleted,
		"BatchID", resp.BatchId,
		"ItemCount", resp.ItemCount,
	), WithTextOptLeadSpace(true)); err != nil {
		return err
	}

	return w.Flush()
}

func TextRunBatch(resp *pb.RunBatchResponse) error {
	if resp == nil {
		fmt.Println("no response received")
		return nil
	}

	w := NewTextWriter()

	if !resp.Scheduled {
		fmt.Println("no batch was scheduled for execution")
		return nil
	}

	if err := w.WriteOrdered(OrderedData(
		"Scheduled", resp.Scheduled,
		"BatchID", resp.BatchId,
		"ItemCount", resp.ItemCount,
	), WithTextOptLeadSpace(true)); err != nil {
		return err
	}

	return w.Flush()
}
