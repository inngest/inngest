package table

import (
	"fmt"
	"os"

	prettytable "github.com/jedib0t/go-pretty/v6/table"
)

type Row = prettytable.Row

type Table struct {
	prettytable.Writer
}

func (t Table) Render() {
	fmt.Println("")
	t.Writer.Render()
	fmt.Println("")
}

func New(header Row) Table {
	t := prettytable.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.Style().Options.DrawBorder = false
	t.Style().Box.PaddingLeft = "  "
	t.Style().Box.PaddingRight = "  "

	table := Table{Writer: t}
	table.AppendHeader(header)

	return table
}
