package healthcheck

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/connect"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"
)

const (
	defaultHost      = "127.0.0.1"
	defaultScheme    = "http"
	defaultTimeout   = 5 * time.Second
	verboseBodyLimit = 200
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "healthcheck",
		Usage: "Probe local inngest HTTP endpoints; exit non-zero on failure",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "host",
				Value:   defaultHost,
				Sources: cli.EnvVars("INNGEST_HOST"),
				Usage:   "Inngest server host",
			},
			&cli.IntFlag{
				Name:    "port",
				Value:   api.DefaultAPIPort,
				Sources: cli.EnvVars("INNGEST_PORT"),
				Usage:   "Inngest server port",
			},
			&cli.IntFlag{
				Name:    "connect-gateway-port",
				Value:   connect.DefaultGatewayPort,
				Sources: cli.EnvVars("INNGEST_CONNECT_GATEWAY_PORT"),
				Usage:   "Connect Gateway port",
			},
			&cli.StringFlag{
				Name:  "scheme",
				Value: defaultScheme,
				Usage: "URL scheme: http or https",
			},
			&cli.DurationFlag{
				Name:  "timeout",
				Value: defaultTimeout,
				Usage: "Per-probe HTTP timeout",
			},
			&cli.BoolFlag{
				Name:  "skip-connect-gateway",
				Value: false,
				Usage: "Skip the Connect Gateway probe",
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Value: false,
				Usage: "On failure, also print response status and a snippet of the body",
			},
		},
		Action: run,
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	host := cmd.String("host")
	scheme := cmd.String("scheme")
	timeout := cmd.Duration("timeout")
	verbose := cmd.Bool("verbose")

	type probe struct {
		component string
		url       string
	}

	probes := []probe{
		{
			component: "api",
			url:       fmt.Sprintf("%s://%s:%d%s", scheme, host, cmd.Int("port"), api.HealthPath),
		},
	}
	if !cmd.Bool("skip-connect-gateway") {
		probes = append(probes, probe{
			component: "connect-gateway",
			url:       fmt.Sprintf("%s://%s:%d%s", scheme, host, cmd.Int("connect-gateway-port"), connect.ReadyPath),
		})
	}

	client := &http.Client{Timeout: timeout}
	eg, egCtx := errgroup.WithContext(ctx)
	errs := make([]error, len(probes))
	for i, p := range probes {
		eg.Go(func() error {
			if err := probeOnce(egCtx, client, p.url, verbose); err != nil {
				errs[i] = fmt.Errorf("healthcheck: %s %s: %w", p.component, p.url, err)
			}
			return nil
		})
	}
	_ = eg.Wait()

	var failed int
	for _, err := range errs {
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			failed++
		}
	}
	if failed > 0 {
		return cli.Exit("", 1)
	}
	return nil
}

func probeOnce(ctx context.Context, client *http.Client, url string, verbose bool) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	if verbose {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, verboseBodyLimit))
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	return fmt.Errorf("status %d", resp.StatusCode)
}
