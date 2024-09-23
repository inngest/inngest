package commands

import (
	"fmt"
	"os"
	"strconv"

	"github.com/inngest/inngest/cmd/commands/internal/localconfig"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/lite"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCmdLite() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "lite",
		Short:   "[Experimental] Run the Inngest lite server",
		Example: "inngest lite",
		Run:     doLite,
	}

	cmd.Flags().String("config", "", "Path to the configuration file")
	cmd.Flags().String("host", "", "host to run the API on")
	cmd.Flags().StringP("port", "p", "8288", "port to run the API on")
	cmd.Flags().String("signing-key", "", "Signing key")

	return cmd
}

func doLite(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	// TODO Likely need a `Lite()`
	conf, err := config.Dev(ctx)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if err = localconfig.InitStartConfig(ctx, cmd); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	port, err := strconv.Atoi(viper.GetString("port"))
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	conf.EventAPI.Port = port

	host := viper.GetString("host")
	if host != "" {
		conf.EventAPI.Addr = host
	}

	if err := itrace.NewUserTracer(ctx, itrace.TracerOpts{
		ServiceName:   "lite",
		Type:          itrace.TracerTypeOTLPHTTP,
		TraceEndpoint: "localhost:8288",
		TraceURLPath:  "/dev/traces",
	}); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer func() {
		_ = itrace.CloseUserTracer(ctx)
	}()

	opts := lite.StartOpts{
		Config: *conf,
	}

	signingKey := viper.GetString("signing-key")
	if signingKey != "" {
		opts.SigningKey = &signingKey
	}

	err = lite.New(ctx, opts)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
