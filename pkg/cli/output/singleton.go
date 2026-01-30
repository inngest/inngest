package output

import (
	"fmt"

	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

func TextSingletonInfo(resp *pb.SingletonInfoResponse) error {
	if resp == nil {
		fmt.Println("no singleton info found")
		return nil
	}

	w := NewTextWriter()

	if !resp.HasLock {
		fmt.Println("no singleton lock currently held")
		return nil
	}

	if err := w.WriteOrdered(OrderedData(
		"HasLock", resp.HasLock,
		"CurrentRunID", resp.CurrentRunId,
	), WithTextOptLeadSpace(true)); err != nil {
		return err
	}

	return w.Flush()
}

func TextDeleteSingletonLock(resp *pb.DeleteSingletonLockResponse) error {
	if resp == nil {
		fmt.Println("no response received")
		return nil
	}

	w := NewTextWriter()

	if !resp.Deleted {
		fmt.Println("no singleton lock was deleted")
		return nil
	}

	if err := w.WriteOrdered(OrderedData(
		"Deleted", resp.Deleted,
		"RunID", resp.RunId,
	), WithTextOptLeadSpace(true)); err != nil {
		return err
	}

	return w.Flush()
}
