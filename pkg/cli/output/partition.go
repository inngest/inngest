package output

import (
	pb "github.com/inngest/inngest/proto/gen/debug/v1"
)

func PartitionTable(pt *pb.PartitionResponse, pts *pb.PartitionStatusResponse) error {
	w := NewTextWriter()

	if pt != nil {
		// conf, err := json.MarshalIndent([]byte(`{"hello": "world"}`), "", "  ")
		// if err != nil {
		// 	return fmt.Errorf("error marshalling partition config: %w", err)
		// }

		w.Write(map[string]any{
			"Type": "Partition",
			"ID":   pt.Id,
			"Slug": pt.Slug,
			"Tenant": map[string]any{
				"Account":     pt.Tenant.AccountId,
				"Environment": pt.Tenant.EnvId,
				"App":         pt.Tenant.AppId,
				"Queue Shard": "TODO",
			},
			// TODO: implement this
			"Concurrency": map[string]any{
				"Account":  0,
				"Function": 0,
			},
		},
			WithTextOptLeadSpace(true),
		)
	}

	// Status
	// t.AppendSeparator()
	// t.AppendRow(table.Row{strings.ToUpper("status")}, rowAutoMerge)
	// t.AppendSeparator()

	// // TODO: key queues
	// if pts != nil {
	// 	t.AppendRows([]table.Row{
	// 		{"Paused", pts.Paused},
	// 		{"Migrate", pts.Migrate},
	// 	})
	// } else {
	// 	t.AppendRows([]table.Row{
	// 		{"Paused", false},
	// 		{"Migrate", false},
	// 	})
	// }

	// t.Render()

	return nil
}
