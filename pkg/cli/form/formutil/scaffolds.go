package formutil

import (
	"context"

	"github.com/charmbracelet/bubbles/list"
	"github.com/inngest/inngest/pkg/scaffold"
)

type ScaffoldGetter struct{}

func (s *ScaffoldGetter) Items() []list.Item {
	mapping, _ := scaffold.Parse(context.Background())
	items := []list.Item{}
	for k := range mapping.Languages {
		items = append(items, BasicListItem{
			Name: k,
		})
	}
	items = append(items, BasicListItem{Name: "Another language"})
	return items
}

type BasicListItem struct {
	Name string
	Desc string
}

func (i BasicListItem) Title() string       { return i.Name }
func (i BasicListItem) Description() string { return i.Desc }
func (i BasicListItem) FilterValue() string { return i.Name }
