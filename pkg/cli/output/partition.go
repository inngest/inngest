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

		w.Write([]Row{
			{Key: "ID", Value: pt.Id},
			{Key: "Slug", Value: pt.Slug},
		})

		// t.AppendRows([]table.Row{
		// 	{"ID", pt.Id},
		// 	{"Slug", pt.Slug},
		// })
		// t.AppendSeparator()
		// t.AppendRow(table.Row{strings.ToUpper("Tenant")}, rowAutoMerge)
		// t.AppendSeparator()
		// t.AppendRows([]table.Row{
		// 	{"Account", pt.Tenant.AccountId},
		// 	{"Environment", pt.Tenant.EnvId},
		// 	{"App", pt.Tenant.AppId},
		// 	{"Queue Shard", "TODO"},
		// 	{"Account Concurrency", "TODO"},
		// 	{"Function Concurrency", "TODO"},
		// })
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
