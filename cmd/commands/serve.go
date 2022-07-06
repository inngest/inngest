package commands

import (
	"fmt"

	"github.com/inngest/inngest-cli/pkg/api"
	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/inngest/inngest-cli/pkg/server"
	"github.com/spf13/cobra"
)

const (
	ServeExecutor = "executor"
	ServeEventAPI = "events"
)

var (
	serveConf = ""
	serveArgs = []string{ServeExecutor, ServeEventAPI}
)

func NewCmdServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "serve [component]",
		Short:     "Start an Inngest server",
		Example:   "inngestctl serve executor",
		RunE:      serve,
		Args:      cobra.ExactValidArgs(1),
		ValidArgs: serveArgs,
	}

	cmd.Flags().StringVarP(&serveConf, "config", "c", "", "The config file location (defaults to ./inngest.(cue|json) or /etc/inngest.(cue|json)")

	return cmd
}

func serve(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	var s server.Server

	locs := []string{}
	if serveConf != "" {
		locs = []string{serveConf}
	}
	conf, err := config.Load(ctx, locs...)
	if err != nil {
		return err
	}

	switch args[0] {
	case ServeEventAPI:
		s = api.NewServer(conf)
	default:
		return fmt.Errorf("Not implemented")
	}

	return server.Start(ctx, s)
}
