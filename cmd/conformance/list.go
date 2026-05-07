package conformance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/cli/output"
	conf "github.com/inngest/inngest/pkg/conformance"
	"github.com/urfave/cli/v3"
)

func listCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List available conformance suites, cases, features, and transports.",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_ = ctx
			registry := conf.DefaultRegistry()

			if cmd.Bool("json") {
				byt, err := json.MarshalIndent(struct {
					Transports []conf.Transport        `json:"transports"`
					Suites     map[string]conf.Suite   `json:"suites"`
					Cases      map[string]conf.Case    `json:"cases"`
					Features   map[string]conf.Feature `json:"features"`
				}{
					Transports: conf.ValidTransports(),
					Suites:     registry.Suites,
					Cases:      registry.Cases,
					Features:   registry.Features,
				}, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(byt))
				return nil
			}

			return output.TextConformanceCatalog(registry)
		},
	}
}
