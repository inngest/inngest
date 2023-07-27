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
	"github.com/inngest/inngest/pkg/telemetry"
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

	var svcName string
	services := make([]service.Service, 0)
	for _, name := range args {
		switch name {
		case ServeEventAPI:
			svc := api.NewService(*conf)
			svcName = svc.Name()
			services = append(services, svc)
		case ServeRunner:
			svc := runner.NewService(*conf)
			svcName = svc.Name()
			services = append(services, svc)
		case ServeExecutor:
			svc := executor.NewService(*conf)
			svcName = svc.Name()
			services = append(services, svc)
		case ServeCoreAPI:
			svc := coreapi.NewService(*conf)
			svcName = svc.Name()
			services = append(services, svc)
		default:
			fmt.Println("Not implemented")
			os.Exit(1)
		}
	}

	close, err := telemetry.TracerSetup(svcName, telemetry.TracerTypeIO)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer close()

	if err := service.StartAll(ctx, services...); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
