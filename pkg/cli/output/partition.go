package output

import (
	"os"
	"strings"

	pb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/jedib0t/go-pretty/v6/table"
)

func PartitionTable(pt *pb.PartitionResponse, pts *pb.PartitionStatusResponse) error {
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.SetOutputMirror(os.Stdout)

	rowAutoMerge := table.RowConfig{AutoMerge: true}
	// colConf := []table.ColumnConfig{}
	// for i := range 2 {
	// 	colConf = append(colConf, table.ColumnConfig{Number: i + 1, AutoMerge: true})
	// }
	// t.SetColumnConfigs(colConf)

	if pt != nil {
		// conf, err := json.MarshalIndent([]byte(`{"hello": "world"}`), "", "  ")
		// if err != nil {
		// 	return fmt.Errorf("error marshalling partition config: %w", err)
		// }

		t.AppendRows([]table.Row{
			{"ID", pt.Id},
			{"Slug", pt.Slug},
		})
		t.AppendSeparator()
		t.AppendRow(table.Row{strings.ToUpper("Tenant")}, rowAutoMerge)
		t.AppendSeparator()
		t.AppendRows([]table.Row{
			{"Account", pt.Tenant.AccountId},
			{"Environment", pt.Tenant.EnvId},
			{"App", pt.Tenant.AppId},
			{"Queue Shard", "TODO"},
			{"Account Concurrency", "TODO"},
			{"Function Concurrency", "TODO"},
		})
	}

	// Status
	t.AppendSeparator()
	t.AppendRow(table.Row{strings.ToUpper("status")}, rowAutoMerge)
	t.AppendSeparator()

	// TODO: key queues
	if pts != nil {
		t.AppendRows([]table.Row{
			{"Paused", pts.Paused},
			{"Migrate", pts.Migrate},
		})
	} else {
		t.AppendRows([]table.Row{
			{"Paused", false},
			{"Migrate", false},
		})
	}

	t.Render()

	return nil
}
