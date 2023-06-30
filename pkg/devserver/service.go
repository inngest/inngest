package devserver

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/cli"
	"github.com/inngest/inngest/pkg/coreapi"
	"github.com/inngest/inngest/pkg/coredata/inmemory"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/service"
	"github.com/mattn/go-isatty"
)

const (
	SDKPollInterval = 5 * time.Second
)

func newService(opts StartOpts, loader *inmemory.ReadWriter, runner runner.Runner, data cqrs.Manager) *devserver {
	return &devserver{
		data:        data,
		runner:      runner,
		loader:      loader,
		opts:        opts,
		urls:        opts.URLs,
		urlLock:     &sync.Mutex{},
		handlerLock: &sync.Mutex{},
	}
}

// devserver is an individual service which operates development-specific APIs.
//
// Usually, you would have the event API hosted separately to any other APIs.
// In the dev server, we only want one port open:  all APIs are hosted together
// in a single router on a single port.  This simplifies the CLI args (--port) and
// SDKs, as they can test and use a single URL.
type devserver struct {
	opts StartOpts

	data cqrs.Manager

	// runner stores the runner
	runner  runner.Runner
	tracker *runner.Tracker
	state   state.Manager

	apiservice service.Service

	// urls are the URLs that host SDKs
	urls    []string
	urlLock *sync.Mutex

	// loader stores all registered functions in the dev server.
	loader *inmemory.ReadWriter

	// handlers are updated by the API (d.apiservice) when registering functions.
	handlers    []SDKHandler
	handlerLock *sync.Mutex
}

func (devserver) Name() string {
	return "devserver"
}

func (d *devserver) Pre(ctx context.Context) error {
	// Create a new API endpoint which hosts SDK-related functionality for
	// registering functions.
	devAPI := newDevAPI(d)

	datarw := d.loader
	core, err := coreapi.NewCoreApi(coreapi.Options{
		Data:          d.data,
		Config:        d.opts.Config,
		Logger:        logger.From(ctx),
		APIReadWriter: datarw,
		Runner:        d.runner,
		Tracker:       d.tracker,
		State:         d.state,
	})
	if err != nil {
		return err
	}

	// Create a new data API directly in the devserver.  This allows us to inject
	// the data API into the dev server port, providing a single router for the dev
	// server UI, events, and API for loading data.

	// Merge the dev server API (for handling files & registration) with the data
	// API into the event API router.
	d.apiservice = api.NewService(
		d.opts.Config,
		api.Mount{At: "/", Router: devAPI},
		api.Mount{At: "/v0", Router: core.Router},
		api.Mount{At: "/debug", Handler: middleware.Profiler()},
	)

	if d.opts.Autodiscover {
		// Autodiscover the URLs that are hosting Inngest SDKs on the local machine.
		go d.autodiscover(ctx)
	}

	return d.apiservice.Pre(ctx)
}

func (d *devserver) Run(ctx context.Context) error {
	// Start polling the SDKs as the APIs are going live.
	go d.pollSDKs(ctx)

	// Add a nice output to the terminal.
	if isatty.IsTerminal(os.Stdout.Fd()) {
		go func() {
			<-time.After(25 * time.Millisecond)
			addr := fmt.Sprintf("%s:%d", d.opts.Config.EventAPI.Addr, d.opts.Config.EventAPI.Port)
			fmt.Println("")
			fmt.Println("")
			fmt.Print(cli.BoldStyle.Render("\tInngest dev server online "))
			fmt.Printf(cli.TextStyle.Render(fmt.Sprintf("at %s, visible at the following URLs:", addr)) + "\n\n")
			for n, ip := range localIPs() {
				style := cli.BoldStyle
				if n > 0 {
					style = cli.TextStyle
				}
				fmt.Print(style.Render(fmt.Sprintf("\t - http://%s:%d", ip.IP.String(), d.opts.Config.EventAPI.Port)))
				if ip.IP.IsLoopback() {
					fmt.Print(cli.TextStyle.Render(fmt.Sprintf(" (http://localhost:%d)", d.opts.Config.EventAPI.Port)))
				}
				fmt.Println("")
			}
			fmt.Println("")
			fmt.Println("")
		}()
	}

	return d.apiservice.Run(ctx)
}

func (d *devserver) Stop(ctx context.Context) error {
	return d.apiservice.Stop(ctx)
}

// Autodiscover attempts to run autodiscovery while the dev server is running.
//
// This lets the dev server start and wait for the SDK server to come up at
// any point.
func (d *devserver) autodiscover(ctx context.Context) {
	logger.From(ctx).Info().Msg("autodiscovering locally hosted SDKs")
	for {
		if ctx.Err() != nil {
			return
		}
		d.urlLock.Lock()
		d.urls = Autodiscover(ctx)
		d.urlLock.Unlock()
		<-time.After(5 * time.Second)
	}
}

// pollSDKs hits each SDK's register endpoint, asking them to communicate with
// the dev server to re-register their functions.
func (d *devserver) pollSDKs(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		d.urlLock.Lock()
		for _, u := range d.urls {
			// Make a new PUT request to the URL, indicating that the
			// SDK should push functions to the dev server.
			req, _ := http.NewRequest(http.MethodPut, u, nil)
			resp, err := hc.Do(req)
			if err != nil {
				logger.From(ctx).Error().Err(err).Str("url", u).Msg("unable to connect to the SDK")
				continue
			}
			if resp.StatusCode == 200 {
				continue
			}
			// Log an error that we were unable to connect to the SDK.
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			logger.From(ctx).Error().
				Int("status", resp.StatusCode).
				Str("url", u).
				Str("response", string(body)).
				Msg("unable to connect to the SDK")
		}
		d.urlLock.Unlock()

		<-time.After(SDKPollInterval)
	}
}

// SDKHandler represents a handler that has registered with the dev server.
type SDKHandler struct {
	Functions []string            `json:"functionIDs"`
	SDK       sdk.RegisterRequest `json:"sdk"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
}

func localIPs() []*net.IPNet {
	ips := []*net.IPNet{}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet)
			}
		}
	}
	return ips
}
