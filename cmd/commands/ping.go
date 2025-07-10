package commands

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/inngest/inngest/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func NewCmdPing(rootCmd *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Ping the Inngest server",
		Run:   doPing,
	}
	baseFlags := pflag.NewFlagSet("base", pflag.ExitOnError)
	baseFlags.String("host", "localhost", "Inngest server hostname")
	baseFlags.StringP("port", "p", "8288", "Inngest server port")
	cmd.Flags().AddFlagSet(baseFlags)

	return cmd
}

func doPing(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	_, err := config.Dev(ctx)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	err = errors.Join(err, viper.BindPFlag("host", cmd.Flags().Lookup("host")))
	err = errors.Join(err, viper.BindPFlag("port", cmd.Flags().Lookup("port")))
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	viper.SetEnvPrefix("INNGEST")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	portString := viper.GetString("port")

	port, err := strconv.Atoi(portString)
	if err != nil {
		fmt.Println("Error parsing port:", err)
		os.Exit(1)
	}

	url := fmt.Sprintf("http://%s:%d/ping", viper.GetString("host"), port)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error pinging the Inngest server:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error pinging the Inngest server:", resp.Status)
		os.Exit(1)
		return
	}

	os.Exit(0)
}
