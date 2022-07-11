package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/inngest/inngest-cli/pkg/api"
	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/inngest/inngest-cli/pkg/coreapi"
	"github.com/inngest/inngest-cli/pkg/execution/executor"
	"github.com/inngest/inngest-cli/pkg/execution/runner"
	"github.com/inngest/inngest-cli/pkg/service"
	"github.com/spf13/cobra"

	// Import the default drivers, queues, and state stores.
	_ "github.com/inngest/inngest-cli/pkg/config/defaults"
)

const (
	ServeExecutor = "executor"
	ServeRunner   = "runner"
	ServeEventAPI = "events-api"
	ServeCoreAPI  = "core-api"
)

var (
	serveConf = ""
	serveArgs = []string{ServeExecutor, ServeRunner, ServeEventAPI, ServeCoreAPI}
)

func NewCmdServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "serve [component]",
		Short:     "Start an Inngest service",
		Example:   fmt.Sprintf("inngest serve %s", strings.Join(serveArgs, " ")),
		Run:       serve,
		Args:      cobra.OnlyValidArgs,
		ValidArgs: serveArgs,
	}

	cmd.Flags().StringVarP(&serveConf, "config", "c", "", "The config file location (defaults to ./inngest.(cue|json) or /etc/inngest.(cue|json)")

	return cmd
}

func serve(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	locs := []string{}
	if serveConf != "" {
		locs = []string{serveConf}
	}
	conf, err := config.Load(ctx, locs...)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
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
		case ServeCoreAPI:
			svc = append(svc, coreapi.NewService(*conf))
		default:
			fmt.Println("Not implemented")
			os.Exit(1)
		}
	}

	if err := service.StartAll(ctx, svc...); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
