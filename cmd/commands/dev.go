package commands

import (
	"fmt"
	"os"

	"github.com/inngest/inngestctl/pkg/api"
	"github.com/spf13/cobra"
)

func NewCmdDev() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dev",
		Short:   "Run the inngest dev server",
		Example: "inngest dev -p 1234",
		Run:     doDev,
	}

	cmd.Flags().StringP("port", "p", "9999", "port to run the API on")
	return cmd
}

func doDev(cmd *cobra.Command, args []string) {
	fmt.Println("Running!")

	err := api.NewAPI(api.Opts{
		Port:         cmd.Flag("port").Value.String(),
		EventHandler: api.BasicEventHandler,
		Output:       api.StdoutOutputWriter,
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
