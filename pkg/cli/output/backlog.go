package output

import (
	"fmt"

	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

func TextShadowPartition(sp *pb.ShadowPartitionResponse) error {
	if sp == nil {
		fmt.Println("no shadow partition found")
		return nil
	}

	w := NewTextWriter()
	if err := w.WriteOrdered(OrderedData(
		"Type", "Shadow Partition",
		"Partition ID", sp.PartitionId,
		"Function Version", sp.FunctionVersion,
		"Lease ID", sp.LeaseId,
		"Function ID", sp.FunctionId,
		"Env ID", sp.EnvId,
		"Account ID", sp.AccountId,
		"System Queue Name", sp.SystemQueueName,
		"Backlog Count", sp.BacklogCount,
	), WithTextOptLeadSpace(true)); err != nil {
		return err
	}
	return w.Flush()
}

func TextBacklogList(resp *pb.BacklogsResponse) error {
	if resp == nil || len(resp.Backlogs) == 0 {
		fmt.Println("no backlogs found")
		return nil
	}

	w := NewTextWriter()

	if err := w.WriteOrdered(OrderedData(
		"Total Backlogs", resp.TotalCount,
	), WithTextOptLeadSpace(true)); err != nil {
		return err
	}

	for _, bl := range resp.Backlogs {
		data := OrderedData(
			"Backlog ID", bl.BacklogId,
			"Shadow Partition ID", bl.ShadowPartitionId,
			"Function Version", bl.EarliestFunctionVersion,
			"Start", bl.Start,
			"Item Count", bl.ItemCount,
		)

		if len(bl.ConcurrencyKeys) > 0 {
			ckMap := NewOrderedMap()
			for i, ck := range bl.ConcurrencyKeys {
				ckMap.Set(fmt.Sprintf("Key #%d", i+1), OrderedData(
					"Canonical Key ID", ck.CanonicalKeyId,
					"Scope", ck.Scope,
					"Entity ID", ck.EntityId,
					"Key Expression", ck.HashedKeyExpression,
					"Hashed Value", ck.HashedValue,
					"Unhashed Value", ck.UnhashedValue,
					"Mode", ck.ConcurrencyMode,
				))
			}
			data.Set("Concurrency Keys", ckMap)
		}

		if bl.Throttle != nil {
			data.Set("Throttle", OrderedData(
				"Key", bl.Throttle.ThrottleKey,
				"Raw Value", bl.Throttle.ThrottleKeyRawValue,
				"Expression Hash", bl.Throttle.ThrottleKeyExpressionHash,
			))
		}

		if err := w.WriteOrdered(data, WithTextOptLeadSpace(true)); err != nil {
			return err
		}
	}

	return w.Flush()
}

func TextBacklogSize(resp *pb.BacklogSizeResponse) error {
	if resp == nil {
		fmt.Println("no backlog found")
		return nil
	}

	w := NewTextWriter()
	data := OrderedData(
		"Backlog ID", resp.BacklogId,
		"Item Count", resp.ItemCount,
	)

	if resp.Backlog != nil {
		data.Set("Shadow Partition ID", resp.Backlog.ShadowPartitionId)
		data.Set("Start", resp.Backlog.Start)

		if len(resp.Backlog.ConcurrencyKeys) > 0 {
			ckMap := NewOrderedMap()
			for i, ck := range resp.Backlog.ConcurrencyKeys {
				ckMap.Set(fmt.Sprintf("Key #%d", i+1), OrderedData(
					"Canonical Key ID", ck.CanonicalKeyId,
					"Scope", ck.Scope,
					"Unhashed Value", ck.UnhashedValue,
					"Mode", ck.ConcurrencyMode,
				))
			}
			data.Set("Concurrency Keys", ckMap)
		}

		if resp.Backlog.Throttle != nil {
			data.Set("Throttle", OrderedData(
				"Key", resp.Backlog.Throttle.ThrottleKey,
				"Raw Value", resp.Backlog.Throttle.ThrottleKeyRawValue,
			))
		}
	}

	if err := w.WriteOrdered(data, WithTextOptLeadSpace(true)); err != nil {
		return err
	}
	return w.Flush()
}
