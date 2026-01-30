package output

import (
	"encoding/json"
	"fmt"
	"time"

	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

func TextDebounceInfo(resp *pb.DebounceInfoResponse) error {
	if resp == nil {
		fmt.Println("no debounce info found")
		return nil
	}

	w := NewTextWriter()

	if !resp.HasDebounce {
		fmt.Println("no pending debounce found")
		return nil
	}

	// Format timeout as both unix millis and human-readable
	timeoutStr := fmt.Sprintf("%d (%s)", resp.Timeout, time.UnixMilli(resp.Timeout).UTC().Format(time.RFC3339))

	// Try to parse event data as JSON for better display
	var eventDataDisplay any
	if len(resp.EventData) > 0 {
		if err := json.Unmarshal(resp.EventData, &eventDataDisplay); err != nil {
			eventDataDisplay = string(resp.EventData)
		}
	}

	if err := w.WriteOrdered(OrderedData(
		"HasDebounce", resp.HasDebounce,
		"DebounceID", resp.DebounceId,
		"EventID", resp.EventId,
		"Timeout", timeoutStr,
		"AccountID", resp.AccountId,
		"WorkspaceID", resp.WorkspaceId,
		"FunctionID", resp.FunctionId,
		"EventData", eventDataDisplay,
	), WithTextOptLeadSpace(true)); err != nil {
		return err
	}

	return w.Flush()
}

func TextDeleteDebounce(resp *pb.DeleteDebounceResponse) error {
	if resp == nil {
		fmt.Println("no response received")
		return nil
	}

	w := NewTextWriter()

	if !resp.Deleted {
		fmt.Println("no debounce was deleted")
		return nil
	}

	if err := w.WriteOrdered(OrderedData(
		"Deleted", resp.Deleted,
		"DebounceID", resp.DebounceId,
		"EventID", resp.EventId,
	), WithTextOptLeadSpace(true)); err != nil {
		return err
	}

	return w.Flush()
}

func TextRunDebounce(resp *pb.RunDebounceResponse) error {
	if resp == nil {
		fmt.Println("no response received")
		return nil
	}

	w := NewTextWriter()

	if !resp.Scheduled {
		fmt.Println("no debounce was scheduled for execution")
		return nil
	}

	if err := w.WriteOrdered(OrderedData(
		"Scheduled", resp.Scheduled,
		"DebounceID", resp.DebounceId,
		"EventID", resp.EventId,
	), WithTextOptLeadSpace(true)); err != nil {
		return err
	}

	return w.Flush()
}
