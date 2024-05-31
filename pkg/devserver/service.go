package devserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coocood/freecache"
	"github.com/eko/gocache/lib/v4/cache"
	freecachestore "github.com/eko/gocache/store/freecache/v4"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/api/apiv1"
	"github.com/inngest/inngest/pkg/cli"
	"github.com/inngest/inngest/pkg/coreapi"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/deploy"
	"github.com/inngest/inngest/pkg/devserver/discovery"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest/log"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/pubsub"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/service"
	"github.com/mattn/go-isatty"
)

func newService(opts StartOpts, runner runner.Runner, data cqrs.Manager, pb pubsub.Publisher, stepLimitOverrides map[string]int) *devserver {
	return &devserver{
		data:               data,
		runner:             runner,
		opts:               opts,
		handlerLock:        &sync.Mutex{},
		publisher:          pb,
		stepLimitOverrides: stepLimitOverrides,
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

	stepLimitOverrides map[string]int

	// runner stores the runner
	runner    runner.Runner
	tracker   *runner.Tracker
	state     state.Manager
	queue     queue.Queue
	executor  execution.Executor
	publisher pubsub.Publisher

	apiservice service.Service

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

	devAPI.Route("/v1", func(r chi.Router) {
		// Add the V1 API to our dev server API.
		cache := cache.New[[]byte](freecachestore.NewFreecache(freecache.NewCache(1024 * 1024)))
		caching := apiv1.NewCacheMiddleware(cache)

		apiv1.AddRoutes(r, apiv1.Opts{
			CachingMiddleware: caching,
			EventReader:       d.data,
			FunctionReader:    d.data,
			FunctionRunReader: d.data,
			JobQueueReader:    d.queue.(queue.JobQueueReader),
			Executor:          d.executor,
		})
	})

	// d.opts.Config.EventStream.Service.TopicName()

	core, err := coreapi.NewCoreApi(coreapi.Options{
		Data:         d.data,
		Config:       d.opts.Config,
		Logger:       logger.From(ctx),
		Runner:       d.runner,
		Tracker:      d.tracker,
		State:        d.state,
		Queue:        d.queue,
		EventHandler: d.handleEvent,
		Executor:     d.executor,
	})
	if err != nil {
		return err
	}
	// Create a new data API directly in the devserver.  This allows us to inject
	// the data API into the dev server port, providing a single router for the dev
	// server UI, events, and API for loading data.
	//
	// Merge the dev server API (for handling files & registration) with the data
	// API into the event API router.
	d.apiservice = api.NewService(
		d.opts.Config,
		api.Mount{At: "/", Router: devAPI},
		api.Mount{At: "/v0", Router: core.Router},
		api.Mount{At: "/debug", Handler: middleware.Profiler()},
	)

	// Autodiscover the URLs that are hosting Inngest SDKs on the local machine.
	go d.runDiscovery(ctx)

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
			if d.opts.Autodiscover {
				fmt.Printf("\tScanning for available serve handlers.\n")
				fmt.Printf("\tTo disable scanning run `inngest dev` with flags: --no-discovery -u <your-serve-url>")
				fmt.Println("")
			}
			fmt.Println("")
		}()
	}

	return d.apiservice.Run(ctx)
}

func (d *devserver) Stop(ctx context.Context) error {
	return d.apiservice.Stop(ctx)
}

// runDiscovery attempts to run autodiscovery while the dev server is running.
//
// This lets the dev server start and wait for the SDK server to come up at

// any point.
func (d *devserver) runDiscovery(ctx context.Context) {
	logger.From(ctx).Info().Msg("autodiscovering locally hosted SDKs")
	pollInterval := time.Duration(d.opts.PollInterval) * time.Second
	for {
		if ctx.Err() != nil {
			return
		}

		if d.opts.Autodiscover {
			_ = discovery.Autodiscover(ctx)
		}

		<-time.After(pollInterval)
	}
}

// pollSDKs hits each SDK's register endpoint, asking them to communicate with
// the dev server to re-register their functions.
func (d *devserver) pollSDKs(ctx context.Context) {
	pollInterval := time.Duration(d.opts.PollInterval) * time.Second

	// Initially, add every app started with the `-u` flag
	for _, url := range d.opts.URLs {
		// URLs must contain a protocol. If not, add http since very few apps
		// use https during development
		if !strings.Contains(url, "://") {
			url = "http://" + url
		}

		// Create a new app which holds the error message.
		params := cqrs.InsertAppParams{
			ID:  uuid.New(),
			Url: url,
			Error: sql.NullString{
				Valid:  true,
				String: deploy.DeployErrUnreachable.Error(),
			},
		}
		if _, err := d.data.InsertApp(ctx, params); err != nil {
			log.From(ctx).Error().Err(err).Msg("error inserting app from scan")
		}
	}

	// Then poll for every added app (including apps added via the `-u` flag and via the
	// UI), plus run autodiscovery.
	for {
		if ctx.Err() != nil {
			return
		}

		urls := map[string]struct{}{}
		if apps, err := d.data.GetApps(ctx); err == nil {
			for _, app := range apps {
				// We've seen this URL.
				urls[app.Url] = struct{}{}

				if !d.opts.Poll && len(app.Error.String) == 0 {
					continue
				}

				// Make a new PUT request to each app, indicating that the
				// SDK should push functions to the dev server.
				res := deploy.Ping(ctx, app.Url)
				if res.Err != nil {
					_, _ = d.data.UpdateAppError(ctx, cqrs.UpdateAppErrorParams{
						ID: app.ID,
						Error: sql.NullString{
							String: res.Err.Error(),
							Valid:  true,
						},
					})
				}
			}
		}

		// Attempt to add new apps for each discovered URL that's _not_ already
		// an app.
		if d.opts.Autodiscover {
			for u := range discovery.URLs() {
				if _, ok := urls[u]; ok {
					continue
				}

				res := deploy.Ping(ctx, u)

				// If there was an SDK error then we should still ensure the app
				// exists. Otherwise, users will have a harder time figuring out
				// why the Dev Server can't find their app.
				if res.Err != nil && res.IsSDK {
					upsertErroredApp(ctx, d.data, u, res.Err)
				}
			}
		}
		<-time.After(pollInterval)
	}
}

func (d *devserver) handleEvent(ctx context.Context, e *event.Event) (string, error) {
	// ctx is the request context, so we need to re-add
	// the caller here.
	l := logger.From(ctx).With().Str("caller", "devserver").Logger()
	ctx = logger.With(ctx, l)

	l.Debug().Str("event", e.Name).Msg("handling event")

	trackedEvent := event.NewOSSTrackedEvent(*e)

	byt, err := json.Marshal(trackedEvent)
	if err != nil {
		l.Error().Err(err).Msg("error unmarshalling event as JSON")
		return "", err
	}

	l.Info().
		Str("event_name", trackedEvent.GetEvent().Name).
		Str("internal_id", trackedEvent.GetInternalID().String()).
		Str("external_id", trackedEvent.GetEvent().ID).
		Interface("event", trackedEvent.GetEvent()).
		Msg("publishing event")

	err = d.publisher.Publish(
		ctx,
		d.opts.Config.EventStream.Service.TopicName(),
		pubsub.Message{
			Name:      event.EventReceivedName,
			Data:      string(byt),
			Timestamp: time.Now(),
		},
	)

	return trackedEvent.GetInternalID().String(), err
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

func upsertErroredApp(
	ctx context.Context,
	mgr cqrs.Manager,
	appURL string,
	pingError error,
) {
	tx, err := mgr.WithTx(ctx)
	if err != nil {
		logger.From(ctx).Error().Err(err).Msg("error creating transaction")
		return
	}

	rollback := func(ctx context.Context) {
		if err := tx.Rollback(ctx); err != nil {
			logger.From(ctx).Error().Err(err).Msg("error rolling back transaction")
		}
	}

	appID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(appURL))
	_, err = tx.GetAppByID(ctx, appID)
	if err == sql.ErrNoRows {
		// App doesn't exist so create it.

		_, err = tx.InsertApp(ctx, cqrs.InsertAppParams{
			Error: sql.NullString{
				String: pingError.Error(),
				Valid:  true,
			},
			ID:  appID,
			Url: appURL,
		})
		if err != nil {
			logger.From(ctx).Error().Err(err).Msg("error inserting app")
			rollback(ctx)
			return
		}

		if err = tx.Commit(ctx); err != nil {
			logger.From(ctx).Error().Err(err).Msg("error inserting app")
			rollback(ctx)
			return
		}

		return
	}

	if err != nil {
		logger.From(ctx).Error().Err(err).Msg("error getting app")
		rollback(ctx)
		return
	}
	_, err = tx.UpdateAppError(ctx, cqrs.UpdateAppErrorParams{
		ID: appID,
		Error: sql.NullString{
			String: pingError.Error(),
			Valid:  true,
		},
	})
	if err != nil {
		logger.From(ctx).Error().Err(err).Msg("error updating app")
		rollback(ctx)
		return
	}

	if err = tx.Commit(ctx); err != nil {
		logger.From(ctx).Error().Err(err).Msg("error updating app")
		rollback(ctx)
		return
	}
}
