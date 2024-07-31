package devserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cli"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/deploy"
	"github.com/inngest/inngest/pkg/devserver/discovery"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest/log"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/pubsub"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/service"
	"github.com/inngest/inngest/pkg/telemetry"
	"github.com/mattn/go-isatty"
	"github.com/redis/rueidis"
	"go.opentelemetry.io/otel/propagation"
)

func NewService(opts StartOpts, runner runner.Runner, data cqrs.Manager, pb pubsub.Publisher, stepLimitOverrides map[string]int, stateSizeLimitOverrides map[string]int, rc rueidis.Client, hw history.Driver) *devserver {
	return &devserver{
		data:                    data,
		runner:                  runner,
		opts:                    opts,
		handlerLock:             &sync.Mutex{},
		snapshotLock:            &sync.Mutex{},
		publisher:               pb,
		stepLimitOverrides:      stepLimitOverrides,
		stateSizeLimitOverrides: stateSizeLimitOverrides,
		redisClient:             rc,
		historyWriter:           hw,
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

	stepLimitOverrides      map[string]int
	stateSizeLimitOverrides map[string]int

	// runner stores the runner
	runner      runner.Runner
	tracker     *runner.Tracker
	state       state.Manager
	queue       queue.Queue
	executor    execution.Executor
	publisher   pubsub.Publisher
	redisClient rueidis.Client

	apiservice service.Service

	historyWriter history.Driver

	// handlers are updated by the API (d.apiservice) when registering functions.
	handlers    []SDKHandler
	handlerLock *sync.Mutex

	// Used to lock the snapshotting process.
	snapshotLock *sync.Mutex
}

func (devserver) Name() string {
	return "devserver"
}

func (d *devserver) Pre(ctx context.Context) error {
	// Import Redis if we can
	d.importRedisSnapshot(ctx)

	// Autodiscover the URLs that are hosting Inngest SDKs on the local machine.
	if d.opts.Autodiscover {
		go d.runDiscovery(ctx)
	}

	return nil
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

	<-ctx.Done()

	return nil
}

func (d *devserver) Stop(ctx context.Context) error {
	d.exportRedisSnapshot(ctx)

	return nil
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
		_ = discovery.Autodiscover(ctx)

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

	carrier := telemetry.NewTraceCarrier()
	telemetry.UserTracer().Propagator().Inject(ctx, propagation.MapCarrier(carrier.Context))

	err = d.publisher.Publish(
		ctx,
		d.opts.Config.EventStream.Service.TopicName(),
		pubsub.Message{
			Name:      event.EventReceivedName,
			Data:      string(byt),
			Timestamp: time.Now(),
			Metadata: map[string]any{
				consts.OtelPropagationKey: carrier,
			},
		},
	)

	return trackedEvent.GetInternalID().String(), err
}

type SnapshotValue struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

func (d *devserver) exportRedisSnapshot(ctx context.Context) (err error) {
	d.snapshotLock.Lock()
	defer d.snapshotLock.Unlock()

	snapshot := make(map[string]SnapshotValue)

	l := logger.From(ctx).With().Str("caller", "devserver").Logger()
	l.Info().Msg("exporting Redis snapshot")
	defer func() {
		if err != nil {
			l.Error().Err(err).Msg("error exporting Redis snapshot")
		} else {
			jsonData, _ := json.Marshal(snapshot)
			humanSize := fmt.Sprintf("%.2fKB", float64(len(jsonData))/1024)
			l.Info().Str("size", humanSize).Msg("exported Redis snapshot")
		}
	}()

	// Get a dedicated client for this operation, which should block all other
	// operations if we only have a pool size of 1.
	rc, _ := d.redisClient.Dedicate()
	defer func() {
		// We'd usually call `done()` to release the client to the pool, but
		// let's just close the entire connection here to ensure nothing else
		// can write.
		rc.Close()
	}()

	// Give an arbitrary amount of time to allow for any writes to finish
	<-time.After(150 * time.Millisecond)

	cmd := rc.B().Keys().Pattern("*").Build()
	keys, err := rc.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		err = fmt.Errorf("error getting keys: %w", err)
		return
	}

	for _, key := range keys {
		typeCmd := rc.B().Type().Key(key).Build()
		var typ string
		typ, err = rc.Do(ctx, typeCmd).ToString()
		if err != nil {
			err = fmt.Errorf("error getting type for key %s: %w", key, err)
			return
		}

		switch typ {
		case "string":
			getCmd := rc.B().Get().Key(key).Build()
			var val string
			val, err = rc.Do(ctx, getCmd).ToString()
			if err != nil {
				err = fmt.Errorf("error getting value for string key %s: %w", key, err)
				return
			}
			snapshot[key] = SnapshotValue{
				Type:  typ,
				Value: val,
			}
		case "list":
			lrangeCmd := rc.B().Lrange().Key(key).Start(0).Stop(-1).Build()
			var vals []string
			vals, err = rc.Do(ctx, lrangeCmd).AsStrSlice()
			if err != nil {
				err = fmt.Errorf("error getting values for list key %s: %w", key, err)
				return
			}
			snapshot[key] = SnapshotValue{
				Type:  typ,
				Value: vals,
			}
		case "set":
			smembersCmd := rc.B().Smembers().Key(key).Build()
			var vals []string
			vals, err = rc.Do(ctx, smembersCmd).AsStrSlice()
			if err != nil {
				err = fmt.Errorf("error getting values for set key %s: %w", key, err)
				return
			}
			snapshot[key] = SnapshotValue{
				Type:  typ,
				Value: vals,
			}
		case "zset":
			zrangeCmd := rc.B().Zrange().Key(key).Min("-inf").Max("+inf").Byscore().Withscores().Build()
			var vals []string
			vals, err = rc.Do(ctx, zrangeCmd).AsStrSlice()
			if err != nil {
				err = fmt.Errorf("error getting values for zset key %s: %w", key, err)
				return
			}
			snapshot[key] = SnapshotValue{
				Type:  typ,
				Value: vals,
			}
		case "hash":
			hgetallCmd := rc.B().Hgetall().Key(key).Build()
			var rawVals map[string]rueidis.RedisMessage
			rawVals, err = rc.Do(ctx, hgetallCmd).AsMap()
			if err != nil {
				err = fmt.Errorf("error getting values for hash key %s: %w", key, err)
				return
			}
			vals := make(map[string]string, len(rawVals))
			for k, v := range rawVals {
				strVal, _ := v.ToString()
				vals[k] = strVal
			}
			snapshot[key] = SnapshotValue{
				Type:  typ,
				Value: vals,
			}
		case "none":
			// the key was deleted between fetching keys and fetching its
			// type. For now we continue and ignore it; we should make sure
			// the client is read-only before we try to dump.
		default:
			err = fmt.Errorf("unsupported type: %s", typ)
			return
		}
	}

	var file *os.File
	file, err = os.Create(fmt.Sprintf("%s/%s", consts.DevServerTempDir, consts.DevServerRdbFile))
	if err != nil {
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(snapshot)
	if err != nil {
		return
	}

	return nil
}

func (d *devserver) importRedisSnapshot(ctx context.Context) (err error, imported bool) {
	file, err := os.Open(fmt.Sprintf("%s/%s", consts.DevServerTempDir, consts.DevServerRdbFile))
	if err != nil && os.IsNotExist(err) {
		err = nil
		return
	}
	defer file.Close()

	var snapshot map[string]SnapshotValue

	l := logger.From(ctx).With().Str("caller", "devserver").Logger()
	l.Info().Msg("importing Redis snapshot")
	defer func() {
		if err != nil {
			l.Error().Err(err).Msg("error importing Redis snapshot")
		} else {
			jsonData, _ := json.Marshal(snapshot)
			humanSize := fmt.Sprintf("%.2fKB", float64(len(jsonData))/1024)
			l.Info().Str("size", humanSize).Msg("imported Redis snapshot")
		}
	}()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&snapshot)
	if err != nil {
		err = fmt.Errorf("error decoding snapshot: %w", err)
		return
	}

	rc, done := d.redisClient.Dedicate()
	defer done()

	for key, data := range snapshot {
		switch data.Type {
		case "string":
			strVal := data.Value.(string)
			setCmd := rc.B().Set().Key(key).Value(strVal).Build()
			err = rc.Do(ctx, setCmd).Error()
			if err != nil {
				err = fmt.Errorf("error setting string key %s: %w", key, err)
				return
			}

		case "list":
			vals := data.Value.([]interface{})
			strValues := make([]string, len(vals))
			for i, v := range vals {
				strVal, _ := v.(string)
				strValues[i] = strVal
			}
			rpushCmd := rc.B().Rpush().Key(key).Element(strValues...).Build()
			err = rc.Do(ctx, rpushCmd).Error()
			if err != nil {
				err = fmt.Errorf("error pushing to list key %s: %w", key, err)
				return
			}

		case "set":
			strValues := data.Value.([]string)
			// err = rc.SAdd(ctx, key, strValues...).Err()
			saddCmd := rc.B().Sadd().Key(key).Member(strValues...).Build()
			err = rc.Do(ctx, saddCmd).Error()
			if err != nil {
				err = fmt.Errorf("error adding to set key %s: %w", key, err)
				return
			}

		case "zset":
			vals := data.Value.([]interface{})
			zaddCmd := rc.B().Zadd().Key(key).ScoreMember()
			for i := 0; i < len(vals); i += 2 {
				member := vals[i].(string)
				score, _ := strconv.ParseFloat(vals[i+1].(string), 64)
				zaddCmd = zaddCmd.ScoreMember(score, member)
			}
			err = rc.Do(ctx, zaddCmd.Build()).Error()
			if err != nil {
				err = fmt.Errorf("error adding to zset key %s: %w", key, err)
				return
			}

		case "hash":
			values := data.Value.(map[string]interface{})
			hmsetCmd := rc.B().Hmset().Key(key).FieldValue()
			for k, v := range values {
				strVal, _ := v.(string)
				hmsetCmd = hmsetCmd.FieldValue(k, strVal)
			}
			err = rc.Do(ctx, hmsetCmd.Build()).Error()
			if err != nil {
				err = fmt.Errorf("error setting hash key %s: %w", key, err)
				return
			}

		default:
			err = fmt.Errorf("unsupported key type: %s", data.Type)
			return
		}
	}

	imported = true

	return
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
