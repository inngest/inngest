package commands

import (
	"fmt"
	"os"
	"strconv"

	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/devserver"
	"github.com/spf13/cobra"
)

func NewCmdDev() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dev",
		Short:   "Run the Inngest dev server",
		Example: "inngest dev -u http://localhost:3000/api/inngest",
		Run:     doDev,
	}

	cmd.Flags().String("host", "", "host to run the API on")
	cmd.Flags().StringP("port", "p", "8288", "port to run the API on")
	cmd.Flags().StringSliceP("sdk-url", "u", []string{}, "SDK URLs to load functions from")
	cmd.Flags().Bool("no-discovery", false, "Disable autodiscovery")
	cmd.Flags().Bool("no-poll", false, "Disable polling of apps for updates")

	return cmd
}

func doDev(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	conf, err := config.Dev(ctx)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	port, err := strconv.Atoi(cmd.Flag("port").Value.String())
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	conf.EventAPI.Port = port

	host := cmd.Flag("host").Value.String()
	if host != "" {
		conf.EventAPI.Addr = host
	}

	urls, _ := cmd.Flags().GetStringSlice("sdk-url")

	// Run auto-discovery unless we've explicitly disabled it.
	noDiscovery, _ := cmd.Flags().GetBool("no-discovery")
	noPoll, _ := cmd.Flags().GetBool("no-poll")

	opts := devserver.StartOpts{
		Config:       *conf,
		URLs:         urls,
		Autodiscover: !noDiscovery,
		Poll:         !noPoll,
	}

	err = devserver.New(ctx, opts)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
