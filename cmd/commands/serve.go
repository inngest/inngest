package commands

import (
	"fmt"

	"github.com/inngest/inngest-cli/pkg/api"
	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/inngest/inngest-cli/pkg/execution/executor"
	"github.com/inngest/inngest-cli/pkg/execution/runner"
	"github.com/inngest/inngest-cli/pkg/service"
	"github.com/spf13/cobra"
)

const (
	ServeExecutor = "executor"
	ServeRunner   = "runner"
	ServeEventAPI = "events-api"
)

var (
	serveConf = ""
	serveArgs = []string{ServeExecutor, ServeRunner, ServeEventAPI}
)

func NewCmdServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "serve [component]",
		Short:     "Start an Inngest service",
		Example:   "inngest serve executor",
		RunE:      serve,
		Args:      cobra.OnlyValidArgs,
		ValidArgs: serveArgs,
	}

	cmd.Flags().StringVarP(&serveConf, "config", "c", "", "The config file location (defaults to ./inngest.(cue|json) or /etc/inngest.(cue|json)")

	return cmd
}

func serve(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	locs := []string{}
	if serveConf != "" {
		locs = []string{serveConf}
	}
	conf, err := config.Load(ctx, locs...)
	if err != nil {
		return err
	}

	svc := []service.Service{}
	for _, name := range args {
		switch name {
		case ServeEventAPI:
			svc = append(svc, api.NewService(*conf))
		case ServeRunner:
			svc = append(svc, runner.NewService(*conf))
		case ServeExecutor:
			svc = append(svc, executor.NewService(*conf))
		default:
			return fmt.Errorf("Not implemented")
		}
	}

	return service.StartAll(ctx, svc...)
}
