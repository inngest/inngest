package output

import (
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/execution/state"
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

func TextPause(item *state.Pause) error {
	if item == nil {
		fmt.Println("no item found")
		return nil
	}

	w := NewTextWriter()

	if err := w.WriteOrdered(OrderedData(
		"ID", item.ID,
		"WorkspaceID", item.WorkspaceID,
		"Identifier", OrderedData(
			"RunID", item.Identifier.RunID,
			"FunctionID", item.Identifier.FunctionID,
			"AccountID", item.Identifier.AccountID,
		),
		"Outgoing", item.Outgoing,
		"Incoming", item.Incoming,
		"StepName", item.StepName,
		"Opcode", item.Opcode,
		"Expires", fmt.Sprintf("%d (%s)", time.Time(item.Expires).UTC().UnixMilli(), time.Time(item.Expires).UTC().Format(time.RFC3339)),
		"Event", item.Event,
		"Expression", item.Expression,
		"InvokeCorrelationID", item.InvokeCorrelationID,
		"InvokeTargetFnID", item.InvokeTargetFnID,
		"SignalID", item.SignalID,
		"ReplaceSignalOnConflict", item.ReplaceSignalOnConflict,
		"OnTimeout", item.OnTimeout,
		"DataKey", item.DataKey,
		"Cancel", item.Cancel,
		"MaxAttempts", item.MaxAttempts,
		"GroupID", item.GroupID,
		"TriggeringEventID", item.TriggeringEventID,
		"Metadata", item.Metadata,
		"ParallelMode", item.ParallelMode,
		"CreatedAt", fmt.Sprintf("%d (%s)", item.CreatedAt.UTC().UnixMilli(), item.CreatedAt.UTC().Format(time.RFC3339)),
	), WithTextOptLeadSpace(true)); err != nil {
		return err
	}

	return w.Flush()
}

func TextIndex(resp *pb.IndexResponse) error {
	if resp == nil {
		fmt.Println("no index found")
		return nil
	}

	w := NewTextWriter()

	// Write index summary first
	if err := w.WriteOrdered(OrderedData(
		"WorkspaceID", resp.WorkspaceId,
		"EventName", resp.EventName,
		"BufferLength", resp.BufferLength,
		"BlockCount", len(resp.Blocks),
	), WithTextOptLeadSpace(true)); err != nil {
		return err
	}

	if err := w.Flush(); err != nil {
		return err
	}

	// Write block information
	if len(resp.Blocks) > 0 {
		fmt.Printf("\nBlocks:\n")

		for i, block := range resp.Blocks {
			fmt.Printf("\n  Block %d:\n", i+1)
			fmt.Printf("    ID:             %s\n", block.Id)
			fmt.Printf("    Length:         %d\n", block.Length)
			fmt.Printf("    FirstTimestamp: %d (%s)\n", block.FirstTimestamp, time.UnixMilli(block.FirstTimestamp).UTC().Format(time.RFC3339))
			fmt.Printf("    LastTimestamp:  %d (%s)\n", block.LastTimestamp, time.UnixMilli(block.LastTimestamp).UTC().Format(time.RFC3339))
			fmt.Printf("    DeleteCount:    %d\n", block.DeleteCount)
		}
	} else {
		fmt.Printf("\nNo blocks found\n")
	}

	return nil
}

func TextBlockPeek(resp *pb.BlockPeekResponse) error {
	if resp == nil {
		fmt.Println("no block data found")
		return nil
	}

	w := NewTextWriter()

	// Write summary
	if err := w.WriteOrdered(OrderedData(
		"BlockID", resp.BlockId,
		"TotalCount", resp.TotalCount,
		"Compacted", resp.Compacted,
	), WithTextOptLeadSpace(true)); err != nil {
		return err
	}

	if err := w.Flush(); err != nil {
		return err
	}

	// Write pause IDs
	if len(resp.PauseIds) > 0 {
		fmt.Printf("\nPause IDs:\n")
		// Calculate padding width based on total count
		width := len(fmt.Sprintf("%d", len(resp.PauseIds)))
		for i, pauseID := range resp.PauseIds {
			fmt.Printf("  %*d: %s\n", width, i+1, pauseID)
		}

		if resp.Compacted {
			fmt.Printf("\n... and %d more (use --limit to show more)\n", resp.TotalCount-int64(len(resp.PauseIds)))
		}
	} else {
		fmt.Printf("\nNo pause IDs found\n")
	}

	return nil
}

func TextBlockDeleted(resp *pb.BlockDeletedResponse) error {
	if resp == nil {
		fmt.Println("no block data found")
		return nil
	}

	w := NewTextWriter()

	// Write summary
	if err := w.WriteOrdered(OrderedData(
		"BlockID", resp.BlockId,
		"TotalCount", resp.TotalCount,
		"Compacted", resp.Compacted,
	), WithTextOptLeadSpace(true)); err != nil {
		return err
	}

	if err := w.Flush(); err != nil {
		return err
	}

	// Write deleted IDs
	if len(resp.DeletedIds) > 0 {
		fmt.Printf("\nDeleted IDs:\n")
		// Calculate padding width based on total count
		width := len(fmt.Sprintf("%d", len(resp.DeletedIds)))
		for i, deletedID := range resp.DeletedIds {
			fmt.Printf("  %*d: %s\n", width, i+1, deletedID)
		}

		if resp.Compacted {
			fmt.Printf("\n... and %d more (use --limit to show more)\n", resp.TotalCount-int64(len(resp.DeletedIds)))
		}
	} else {
		fmt.Printf("\nNo deleted IDs found\n")
	}

	return nil
}
