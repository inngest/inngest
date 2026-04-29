package main

import (
	"github.com/inngest/inngest/cmd/debug"
	"github.com/urfave/cli/v3"
)

func alpha() *cli.Command {
	return &cli.Command{
		Name:   "alpha",
		Hidden: true,
		Usage:  "experimental CLI commands",
		Commands: []*cli.Command{
			debug.Command(),
		},
	}
}
