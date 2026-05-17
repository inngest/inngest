package main

import (
	"github.com/inngest/inngest/cmd/conformance"
	"github.com/inngest/inngest/cmd/debug"
	"github.com/inngest/inngest/cmd/doctor"
	"github.com/urfave/cli/v3"
)

func alpha() *cli.Command {
	return &cli.Command{
		Name:   "alpha",
		Hidden: true,
		Usage:  "experimental CLI commands",
		Commands: []*cli.Command{
			conformance.Command(),
			debug.Command(),
			doctor.Command(),
		},
	}
}
