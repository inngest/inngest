package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/coreapi"
	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/inngest/inngest/pkg/service"
	"github.com/spf13/cobra"
)

const (
	ServeExecutor = "executor"
	ServeRunner   = "runner"
	ServeEventAPI = "event-api"
	ServeCoreAPI  = "core-api"
)

var (
	serveConf = ""
	serveArgs = []string{ServeExecutor, ServeRunner, ServeEventAPI, ServeCoreAPI}
)

func NewCmdServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "serve [component]",
		Short:     "Start an Inngest service for self hosting",
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

	if len(args) == 0 {
		fmt.Println("\nYou must supply one of the following services to serve:")
		for _, svc := range serveArgs {
			fmt.Printf("\t%s\n", svc)
		}
		fmt.Println("")
		os.Exit(1)
	}

	conf, err := config.Dev(ctx)
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
