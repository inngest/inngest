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
		Short:   "Run the inngest dev server",
		Example: "inngest dev -p 1234",
		Run:     doDev,
	}

	cmd.Flags().String("host", "", "host to run the API on")
	cmd.Flags().StringP("port", "p", "9999", "port to run the API on")
	cmd.Flags().String("dir", ".", "directory to load functions from")

	return cmd
}

func doDev(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	conf, err := config.Default(ctx)
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
	dir := cmd.Flag("dir").Value.String()

	err = devserver.NewDevServer(ctx, *conf, dir)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
