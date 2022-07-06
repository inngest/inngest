package commands

import (
	"fmt"

	"github.com/inngest/inngest-cli/pkg/api"
	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/inngest/inngest-cli/pkg/service"
	"github.com/spf13/cobra"
)

const (
	ServeExecutor = "executor"
	ServeEventAPI = "events-api"
)

var (
	serveConf = ""
	serveArgs = []string{ServeExecutor, ServeEventAPI}
)

func NewCmdServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "serve [component]",
		Short:     "Start an Inngest service",
		Example:   "inngest serve executor",
		RunE:      serve,
		Args:      cobra.ExactValidArgs(1),
		ValidArgs: serveArgs,
	}

	cmd.Flags().StringVarP(&serveConf, "config", "c", "", "The config file location (defaults to ./inngest.(cue|json) or /etc/inngest.(cue|json)")

	return cmd
}

func serve(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	var s service.Service

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
		s = api.NewService(conf)
	default:
		return fmt.Errorf("Not implemented")
	}

	return service.Start(ctx, s)
}
