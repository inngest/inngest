package conformance

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

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

			fmt.Println("Transports:")
			for _, transport := range conf.ValidTransports() {
				fmt.Printf("  - %s\n", transport)
			}
			fmt.Println("")

			suiteIDs := make([]string, 0, len(registry.Suites))
			for suiteID := range registry.Suites {
				suiteIDs = append(suiteIDs, suiteID)
			}
			sort.Strings(suiteIDs)

			fmt.Println("Suites:")
			for _, suiteID := range suiteIDs {
				suite := registry.Suites[suiteID]
				fmt.Printf("  - %s: %s\n", suite.ID, suite.Label)
				if suite.Description != "" {
					fmt.Printf("      %s\n", suite.Description)
				}
				for _, caseID := range suite.CaseIDs {
					testCase := registry.Cases[caseID]
					fmt.Printf("      case: %s\n", testCase.ID)
				}
			}
			fmt.Println("")

			featureIDs := make([]string, 0, len(registry.Features))
			for featureID := range registry.Features {
				featureIDs = append(featureIDs, featureID)
			}
			sort.Strings(featureIDs)

			fmt.Println("Features:")
			for _, featureID := range featureIDs {
				feature := registry.Features[featureID]
				fmt.Printf("  - %s: %s\n", feature.ID, feature.Label)
			}

			return nil
		},
	}
}
