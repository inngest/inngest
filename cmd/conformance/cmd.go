package conformance

import "github.com/urfave/cli/v3"

func Command() *cli.Command {
	return &cli.Command{
		Name:    "conformance",
		Aliases: []string{"cft"},
		Usage:   "Run and inspect SDK conformance suites.",
		Commands: []*cli.Command{
			runCommand(),
			listCommand(),
			doctorCommand(),
			reportCommand(),
		},
	}
}
