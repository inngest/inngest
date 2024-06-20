package redis_state

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/expr"
	"github.com/inngest/inngest/pkg/config/registration"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

//go:embed lua/*
var embedded embed.FS

var (
	// scripts stores all embedded lua scripts on initialization
	scripts = map[string]*rueidis.Lua{}
	include = regexp.MustCompile(`-- \$include\(([\w.]+)\)`)

	// A number to version backend logic in order to prevent non-backward compatible
	// changes to break
	currentVersion = 1
)

func init() {
	// register the redis driver
	registration.RegisterState(func() any { return registration.StateConfig(&Config{}) })
	registration.RegisterQueue(func() any { return registration.QueueConfig(&queueConfig{}) })

	// read the lua scripts
	entries, err := embedded.ReadDir("lua")
	if err != nil {
		panic(fmt.Errorf("error reading redis lua dir: %w", err))
	}

	readRedisScripts("lua", entries)
}

func readRedisScripts(path string, entries []fs.DirEntry) {
	for _, e := range entries {
		// NOTE: When using embed go always uses forward slashes as a path
		// prefix. filepath.Join uses OS-specific prefixes which fails on
		// windows, so we construct the path using Sprintf for all platforms
		if e.IsDir() {
			entries, _ := embedded.ReadDir(fmt.Sprintf("%s/%s", path, e.Name()))
			readRedisScripts(path+"/"+e.Name(), entries)
			continue
		}

		byt, err := embedded.ReadFile(fmt.Sprintf("%s/%s", path, e.Name()))
		if err != nil {
			panic(fmt.Errorf("error reading redis lua script: %w", err))
		}

		name := path + "/" + e.Name()
		name = strings.TrimPrefix(name, "lua/")
		name = strings.TrimSuffix(name, ".lua")
		val := string(byt)

		// Add any includes.
		items := include.FindAllStringSubmatch(val, -1)
		if len(items) > 0 {
			// Replace each include
			for _, include := range items {
				byt, err = embedded.ReadFile(fmt.Sprintf("lua/includes/%s", include[1]))
				if err != nil {
					panic(fmt.Errorf("error reading redis lua include: %w", err))
				}
				val = strings.ReplaceAll(val, include[0], string(byt))
			}
		}
		scripts[name] = rueidis.NewLuaScript(val)
	}
}

type queueConfig struct{}

func (c queueConfig) QueueName() string             { return "redis" }
func (c queueConfig) Queue() (osqueue.Queue, error) { return nil, nil }
func (c queueConfig) Consumer() osqueue.Consumer    { return nil }
func (c queueConfig) Producer() osqueue.Producer    { return nil }

// Config registers the configuration for the in-memory state store,
// and provides a factory for the state manager based off of the config.
type Config struct {
	// DSN contains the entire configuration in a single string, if
	// provided (eg. redis://user:pass@host:port/db)
	// DSN *string

	Host       string
	Port       int
	DB         int
	Username   string
	Password   string
	MaxRetries *int
	PoolSize   *int

	KeyPrefix string

	// Expiry represents the expiration time on values stored in state.
	// This defaults to 0, ie. no expiry TTL.
	Expiry time.Duration
}

func (c Config) StateName() string { return "redis" }

func (c Config) Manager(ctx context.Context) (state.Manager, error) {
	opts, err := c.ConnectOpts()
	if err != nil {
		return nil, err
	}

	return New(
		ctx,
		WithConnectOpts(opts),
	)
}

func (c Config) ConnectOpts() (rueidis.ClientOption, error) {
	opts := rueidis.ClientOption{
		InitAddress: []string{fmt.Sprintf("%s:%d", c.Host, c.Port)},
		ShuffleInit: true,
		SelectDB:    c.DB,
		Username:    c.Username,
		Password:    c.Password,
	}
	return opts, nil
}

// Opt represents an option to use when creating a redis-backed state store.
type Opt func(r *mgr)

// New returns a state manager which uses Redis as the backing state store.
//
// By default, this connects to a local Redis server.  Use WithConnectOpts to
// change how we connect to Redis.
func New(ctx context.Context, opts ...Opt) (state.Manager, error) {
	m := &mgr{}

	for _, opt := range opts {
		opt(m)
	}

	if m.s == nil {
		r, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress: []string{"localhost:6379"},
			Password:    "",
		})
		if err != nil {
			return m, err
		}

		m.s = NewShardedClient(r)
	}

	if m.u == nil {
		r, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress: []string{"localhost:6379"},
			Password:    "",
		})
		if err != nil {
			return m, err
		}

		m.u = NewUnshardedClient(r)
	}

	if m.pauseR == nil {
		m.pauseR = m.u.r
	}

	m.shardedMgr = shardedMgr{
		s: m.s,
	}

	m.unshardedMgr = unshardedMgr{
		u:      m.u,
		pauseR: m.pauseR,
	}

	return m, nil
}

// WithShardedClient uses an already connected redis client.
func WithShardedClient(s *ShardedClient) Opt {
	return func(m *mgr) {
		m.s = s
	}
}

// WithUnshardedClient uses an already connected redis client.
func WithUnshardedClient(u *UnshardedClient) Opt {
	return func(m *mgr) {
		m.u = u
	}
}

// WithPauseRedisClient uses an already connected redis client for managing pauses.
func WithPauseRedisClient(r rueidis.Client) Opt {
	return func(m *mgr) {
		m.pauseR = r
	}
}

type mgr struct {
	shardedMgr
	unshardedMgr
}

type shardedMgr struct {
	s *ShardedClient
}

type unshardedMgr struct {
	u *UnshardedClient

	// this is the redis client for managing pauses.
	pauseR rueidis.Client
}

type CompositePauseID struct {
	PauseID uuid.UUID
	RunID   *ulid.ULID
}

func (m shardedMgr) New(ctx context.Context, input state.Input) (state.State, error) {
	// We marshal this ahead of creating a redis transaction as it's necessary
	// every time and reduces the duration that the lock is held.
	events, err := json.Marshal(input.EventBatchData)
	if err != nil {
		return nil, err
	}

	metadata := runMetadata{
		Identifier:     input.Identifier,
		Debugger:       input.Debugger,
		Version:        currentVersion,
		RequestVersion: consts.RequestVersionUnknown, // Always use -1 to indicate unset hash version until first request.
		Context:        input.Context,
		Status:         enums.RunStatusScheduled,
		SpanID:         input.SpanID,
		EventSize:      len(events),
	}
	if input.RunType != nil {
		metadata.RunType = *input.RunType
	}

	var stepsByt []byte
	if len(input.Steps) > 0 {
		stepsByt, err = json.Marshal(input.Steps)
		if err != nil {
			return nil, fmt.Errorf("error storing run state in redis: %w", err)
		}
	}

	// Add total state size, including size of input steps.
	metadata.StateSize = len(events) + len(stepsByt)

	metadataByt, err := json.Marshal(metadata.Map())
	if err != nil {
		return nil, fmt.Errorf("error storing run state in redis: %w", err)
	}

	args, err := StrSlice([]any{
		events,
		metadataByt,
		stepsByt,
	})
	if err != nil {
		return nil, err
	}

	status, err := scripts["new"].Exec(
		ctx,
		m.s.r,
		[]string{
			m.s.kg.Idempotency(ctx, input.Identifier),
			m.s.kg.Events(ctx, input.Identifier),
			m.s.kg.RunMetadata(ctx, input.Identifier.RunID),
			m.s.kg.Actions(ctx, input.Identifier),
		},
		args,
	).AsInt64()

	if err != nil {
		return nil, fmt.Errorf("error storing run state in redis: %w", err)
	}

	if status == 1 {
		return nil, state.ErrIdentifierExists
	}

	return state.NewStateInstance(
			input.Identifier,
			metadata.Metadata(),
			input.EventBatchData,
			input.Steps,
			make([]string, 0),
		),
		nil
}

func (m shardedMgr) UpdateMetadata(ctx context.Context, runID ulid.ULID, md state.MetadataUpdate) error {
	input := []string{
		"0", // Force planning / disable immediate execution
		strconv.Itoa(consts.RequestVersionUnknown), // Request version
		"0", // start time default value
	}
	if md.DisableImmediateExecution {
		input[0] = "1"
	}
	if md.RequestVersion != consts.RequestVersionUnknown {
		input[1] = strconv.Itoa(md.RequestVersion)
	}
	if !md.StartedAt.IsZero() {
		input[2] = strconv.FormatInt(md.StartedAt.UnixMilli(), 10)
	}
	status, err := scripts["updateMetadata"].Exec(
		ctx,
		m.s.r,
		[]string{
			m.s.kg.RunMetadata(ctx, runID),
		},
		input,
	).AsInt64()
	if err != nil {
		return err
	}
	if status != 0 {
		return fmt.Errorf("unknown response updating metadata: %w", err)
	}
	return nil
}

func (m shardedMgr) IsComplete(ctx context.Context, runID ulid.ULID) (bool, error) {
	cmd := m.s.r.B().Hget().Key(m.s.kg.RunMetadata(ctx, runID)).Field("status").Build()
	val, err := m.s.r.Do(ctx, cmd).AsBytes()
	if err != nil {
		return false, err
	}
	return !bytes.Equal(val, []byte("0")), nil
}

func (m shardedMgr) Exists(ctx context.Context, runID ulid.ULID) (bool, error) {
	cmd := m.s.r.B().Exists().Key(m.s.kg.RunMetadata(ctx, runID)).Build()
	return m.s.r.Do(ctx, cmd).AsBool()
}

func (m shardedMgr) metadata(ctx context.Context, runID ulid.ULID) (*runMetadata, error) {
	cmd := m.s.r.B().Hgetall().Key(m.s.kg.RunMetadata(ctx, runID)).Build()
	val, err := m.s.r.Do(ctx, cmd).AsStrMap()
	if err != nil {
		return nil, err
	}
	return newRunMetadata(val)
}

func (m shardedMgr) Cancel(ctx context.Context, id state.Identifier) error {
	status, err := scripts["cancel"].Exec(
		ctx,
		m.s.r,
		[]string{m.s.kg.RunMetadata(ctx, id.RunID)},
		[]string{},
	).AsInt64()
	if err != nil && !rueidis.IsRedisNil(err) {
		return fmt.Errorf("error cancelling: %w", err)
	}
	switch status {
	case 0:
		return nil
	case 1:
		return state.ErrFunctionComplete
	case 2:
		return state.ErrFunctionFailed
	case 3:
		return state.ErrFunctionCancelled
	}
	return fmt.Errorf("unknown return value cancelling function: %d", status)
}

func (m shardedMgr) SetStatus(ctx context.Context, id state.Identifier, status enums.RunStatus) error {
	args, err := StrSlice([]any{
		int(status),
	})
	if err != nil {
		return err
	}

	_, err = scripts["setStatus"].Exec(
		ctx,
		m.s.r,
		[]string{m.s.kg.RunMetadata(ctx, id.RunID)},
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error cancelling: %w", err)
	}
	return nil
}

func (m shardedMgr) Metadata(ctx context.Context, runID ulid.ULID) (*state.Metadata, error) {
	metadata, err := m.metadata(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to load metadata: %w", err)
	}
	meta := metadata.Metadata()
	return &meta, nil
}

func (m shardedMgr) LoadEvents(ctx context.Context, fnID uuid.UUID, runID ulid.ULID) ([]json.RawMessage, error) {
	var (
		events []json.RawMessage
		v1id   = state.Identifier{
			RunID:      runID,
			WorkflowID: fnID,
		}
	)

	cmd := m.s.r.B().Get().Key(m.s.kg.Events(ctx, v1id)).Build()
	byt, err := m.s.r.Do(ctx, cmd).AsBytes()
	if err == nil {
		if err := json.Unmarshal(byt, &events); err != nil {
			return nil, fmt.Errorf("failed to unmarshal batch; %w", err)
		}
		return events, nil
	}

	// Pre-batch days for backcompat.
	cmd = m.s.r.B().Get().Key(m.s.kg.Event(ctx, v1id)).Build()
	byt, err = m.s.r.Do(ctx, cmd).AsBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get event; %w", err)
	}
	return []json.RawMessage{byt}, nil
}

func (m shardedMgr) LoadSteps(ctx context.Context, fnID uuid.UUID, runID ulid.ULID) (map[string]json.RawMessage, error) {
	var (
		steps = map[string]json.RawMessage{}
		v1id  = state.Identifier{
			RunID:      runID,
			WorkflowID: fnID,
		}
	)

	// Load the actions.  This is a map of step IDs to JSON-encoded results.
	cmd := m.s.r.B().Hgetall().Key(m.s.kg.Actions(ctx, v1id)).Build()
	rmap, err := m.s.r.Do(ctx, cmd).AsStrMap()
	if err != nil {
		return nil, fmt.Errorf("failed loading actions; %w", err)
	}
	for stepID, marshalled := range rmap {
		steps[stepID] = json.RawMessage(marshalled)
	}
	return steps, nil
}

func (m shardedMgr) Load(ctx context.Context, runID ulid.ULID) (state.State, error) {
	// XXX: Use a pipeliner to improve speed.
	metadata, err := m.metadata(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to load metadata; %w", err)
	}

	id := metadata.Identifier

	// Load events.
	events := []map[string]any{}
	switch metadata.Version {
	case 0: // pre-batch days
		cmd := m.s.r.B().Get().Key(m.s.kg.Event(ctx, id)).Build()
		byt, err := m.s.r.Do(ctx, cmd).AsBytes()
		if err != nil {
			if err == rueidis.Nil {
				return nil, state.ErrEventNotFound
			}
			return nil, fmt.Errorf("failed to get event; %w", err)
		}
		event := map[string]any{}
		if err := json.Unmarshal(byt, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event; %w", err)
		}
		events = []map[string]any{event}
	default: // current default is 1
		// Load the batch of events
		cmd := m.s.r.B().Get().Key(m.s.kg.Events(ctx, id)).Build()
		byt, err := m.s.r.Do(ctx, cmd).AsBytes()
		if err != nil {
			return nil, fmt.Errorf("failed to get batch; %w", err)
		}
		if err := json.Unmarshal(byt, &events); err != nil {
			return nil, fmt.Errorf("failed to unmarshal batch; %w", err)
		}
	}

	// Load the actions.  This is a map of step IDs to JSON-encoded results.
	cmd := m.s.r.B().Hgetall().Key(m.s.kg.Actions(ctx, id)).Build()
	rmap, err := m.s.r.Do(ctx, cmd).AsStrMap()
	if err != nil {
		return nil, fmt.Errorf("failed loading actions; %w", err)
	}
	actions := map[string]any{}
	for stepID, marshalled := range rmap {
		var data any
		err = json.Unmarshal([]byte(marshalled), &data)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal step \"%s\" with data \"%s\"; %w", stepID, marshalled, err)
		}
		actions[stepID] = data
	}

	meta := metadata.Metadata()

	stack, err := m.stack(ctx, id.RunID)
	if err != nil {
		return nil, fmt.Errorf("error fetching stack: %w", err)
	}

	return state.NewStateInstance(id, meta, events, actions, stack), nil
}

func (m shardedMgr) stack(ctx context.Context, runID ulid.ULID) ([]string, error) {
	cmd := m.s.r.B().Lrange().Key(m.s.kg.Stack(ctx, runID)).Start(0).Stop(-1).Build()
	stack, err := m.s.r.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("error fetching stack: %w", err)
	}
	return stack, nil
}

func (m shardedMgr) StackIndex(ctx context.Context, runID ulid.ULID, stepID string) (int, error) {
	cmd := m.s.r.B().Lrange().Key(m.s.kg.Stack(ctx, runID)).Start(0).Stop(-1).Build()
	stack, err := m.s.r.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return 0, err
	}
	if len(stack) == 0 {
		return 0, nil
	}
	for n, i := range stack {
		if i == stepID {
			return n + 1, nil
		}

	}
	return 0, fmt.Errorf("step not found in stack: %s", stepID)
}

func (m shardedMgr) SaveResponse(ctx context.Context, i state.Identifier, stepID, marshalledOuptut string) error {
	keys := []string{
		m.s.kg.Actions(ctx, i),
		m.s.kg.RunMetadata(ctx, i.RunID),
		m.s.kg.Stack(ctx, i.RunID),
	}
	args := []string{stepID, marshalledOuptut}

	index, err := scripts["saveResponse"].Exec(
		ctx,
		m.s.r,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error saving response: %w", err)
	}
	if index == -1 {
		// This is a duplicate response, so we don't need to do anything.
		return state.ErrDuplicateResponse
	}
	return nil
}

func (m mgr) SavePause(ctx context.Context, p state.Pause) error {
	packed, err := json.Marshal(p)
	if err != nil {
		return err
	}

	// `evt` is used to search for pauses based on event names. We only want to
	// do this if this pause is not part of an invoke. If it is, we don't want
	// to index it by event name as the pause will be processed by correlation
	// ID.
	evt := ""
	if p.Event != nil && (p.InvokeCorrelationID == nil || *p.InvokeCorrelationID == "") {
		evt = *p.Event
	}

	keys := []string{
		m.unshardedMgr.u.kg.PauseID(ctx, p.ID),
		m.unshardedMgr.u.kg.PauseEvent(ctx, p.WorkspaceID, evt),
		m.unshardedMgr.u.kg.Invoke(ctx, p.WorkspaceID),
		m.unshardedMgr.u.kg.PauseIndex(ctx, "add", p.WorkspaceID, evt),
		m.unshardedMgr.u.kg.PauseIndex(ctx, "exp", p.WorkspaceID, evt),
		// This key uses sharding, so it cannot be included in the Lua script.
		// Instead, we execute the command separately below.
		// 		m.u.kg.RunPauses(ctx, p.Identifier.RunID),
	}

	// Add 1 second because int will truncate the float. Otherwise, timeouts
	// will be 1 second less than configured.
	ttl := int(time.Until(p.Expires.Time()).Seconds()) + 1

	// Ensure the TTL is at least 1 second. This probably will always be true
	// since we're adding 1 second above. But you never know if some code
	// between expiry creation and here will take longer than expected.
	if ttl < 1 {
		ttl = 1
	}

	corrId := ""
	if p.InvokeCorrelationID != nil {
		corrId = *p.InvokeCorrelationID
	}

	extendedExpiry := time.Until(p.Expires.Time().Add(10 * time.Minute)).Seconds()
	nowUnixSeconds := time.Now().Unix()

	args, err := StrSlice([]any{
		string(packed),
		p.ID.String(),
		evt,
		corrId,
		ttl,
		// Add at least 10 minutes to this pause, allowing us to process the
		// pause by ID for 10 minutes past expiry.
		int(extendedExpiry),
		nowUnixSeconds,
	})
	if err != nil {
		return err
	}

	status, err := scripts["savePause"].Exec(
		ctx,
		m.unshardedMgr.pauseR,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error finalizing: %w", err)
	}

	// Add an index of when the pause expires.  This lets us manually
	// garbage collect expired pauses from the HSET below.
	// Note: This command was extracted from savePause. SADD is idempotent, so this can run without a lock, outside the Lua script
	cmd := m.shardedMgr.s.r.B().Sadd().Key(m.shardedMgr.s.kg.RunPauses(ctx, p.Identifier.RunID)).Member(p.ID.String()).Build()
	err = m.shardedMgr.s.r.Do(ctx, cmd).Error()
	if err != nil {
		return fmt.Errorf("error finalizing: %w", err)
	}

	switch status {
	case 0:
		return nil
	case 1:
		return state.ErrPauseAlreadyExists
	}
	return fmt.Errorf("unknown response saving pause: %d", status)
}

func (m unshardedMgr) LeasePause(ctx context.Context, id uuid.UUID) error {
	args, err := StrSlice([]any{
		time.Now().UnixMilli(),
		state.PauseLeaseDuration.Seconds(),
	})
	if err != nil {
		return err
	}

	status, err := scripts["leasePause"].Exec(
		ctx,
		m.pauseR,
		[]string{m.u.kg.PauseID(ctx, id), m.u.kg.PauseLease(ctx, id)},
		args,
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error leasing pause: %w", err)
	}
	switch status {
	case 0:
		return nil
	case 1:
		return state.ErrPauseLeased
	case 2:
		return state.ErrPauseNotFound
	default:
		return fmt.Errorf("unknown response leasing pause: %d", status)
	}
}

// Delete deletes state from the state store.  Previously, we would handle this in a
// lifecycle.  Now, state stores must account for deletion directly.  Note that if the
// state store is queue-aware, it must delete queue items for the run also.  This may
// not always be the case.
//
// Returns a boolean indicating whether it performed deletion. If the run had
// parallel steps then it may be false, since parallel steps cause the function
// end to be reached multiple times in a single run
func (m mgr) Delete(ctx context.Context, i state.Identifier) (bool, error) {
	// Ensure this context isn't cancelled;  this is called in a goroutine.
	callCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	key := i.Key
	if i.Key == "" {
		if md, err := m.shardedMgr.Metadata(ctx, i.RunID); err == nil {
			key = m.shardedMgr.s.kg.Idempotency(ctx, md.Identifier)
		}
	} else {
		key = m.shardedMgr.s.kg.Idempotency(ctx, i)
	}

	cmd := m.shardedMgr.s.r.B().Expire().Key(key).Seconds(int64(consts.FunctionIdempotencyPeriod.Seconds())).Build()
	if err := m.shardedMgr.s.r.Do(callCtx, cmd).Error(); err != nil {
		return false, err
	}

	// Fetch all pauses for the run
	if pauseIDs, err := m.shardedMgr.s.r.Do(callCtx, m.shardedMgr.s.r.B().Smembers().Key(m.shardedMgr.s.kg.RunPauses(ctx, i.RunID)).Build()).AsStrSlice(); err == nil {
		for _, id := range pauseIDs {
			pauseID, _ := uuid.Parse(id)
			_ = m.DeletePauseByID(ctx, pauseID)
		}
	}

	// Clear all other data for a job.
	keys := []string{
		m.s.kg.Actions(ctx, i),
		m.s.kg.RunMetadata(ctx, i.RunID),
		m.s.kg.Events(ctx, i),
		m.s.kg.Stack(ctx, i.RunID),

		// XXX: remove these in a state store refactor.
		m.s.kg.Event(ctx, i),
		m.s.kg.History(ctx, i.RunID),
		m.s.kg.Errors(ctx, i),
		m.s.kg.RunPauses(ctx, i.RunID),
	}

	performedDeletion := false
	for _, k := range keys {
		cmd := m.s.r.B().Del().Key(k).Build()
		result := m.s.r.Do(callCtx, cmd)

		// We should check a single key rather than all keys, to avoid races.
		// We'll somewhat arbitrarily pick RunMetadata
		if k == m.s.kg.RunMetadata(ctx, i.RunID) {
			if count, _ := result.ToInt64(); count > 0 {
				performedDeletion = true
			}
		}

		if err := result.Error(); err != nil {
			return false, err
		}
	}

	return performedDeletion, nil
}

func (m mgr) DeletePauseByID(ctx context.Context, pauseID uuid.UUID) error {
	// Attempt to fetch this pause.
	pause, err := m.PauseByID(ctx, pauseID)
	if err == nil && pause != nil {
		return m.DeletePause(ctx, *pause)
	}

	// This won't delete event keys nicely, but still gets the pause yeeted.
	return m.DeletePause(ctx, state.Pause{
		ID: pauseID,
	})
}

func (m mgr) DeletePause(ctx context.Context, p state.Pause) error {
	// Add a default event here, which is null and overwritten by everything.  This is necessary
	// to keep the same cluster key.
	eventKey := m.u.kg.PauseEvent(ctx, p.WorkspaceID, "-")
	if p.Event != nil {
		eventKey = m.u.kg.PauseEvent(ctx, p.WorkspaceID, *p.Event)
	}

	evt := ""
	if p.Event != nil && (p.InvokeCorrelationID == nil || *p.InvokeCorrelationID == "") {
		evt = *p.Event
	}

	keys := []string{
		m.unshardedMgr.u.kg.PauseID(ctx, p.ID),
		// PauseStep is a sharded key, so we need to
		// use a separate command below the Lua script
		// m.u.kg.PauseStep(ctx, p.Identifier, p.Incoming),
		eventKey,
		m.unshardedMgr.u.kg.Invoke(ctx, p.WorkspaceID),
		m.unshardedMgr.u.kg.PauseIndex(ctx, "add", p.WorkspaceID, evt),
		m.unshardedMgr.u.kg.PauseIndex(ctx, "exp", p.WorkspaceID, evt),
		// RunPauses is a sharded key, so we need to
		// use a separate command below the Lua script
		// m.u.kg.RunPauses(ctx, p.Identifier.RunID),
	}
	corrId := ""
	if p.InvokeCorrelationID != nil && *p.InvokeCorrelationID != "" {
		corrId = *p.InvokeCorrelationID
	}
	status, err := scripts["deletePause"].Exec(
		ctx,
		m.unshardedMgr.pauseR,
		keys,
		[]string{
			p.ID.String(),
			corrId,
		},
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error deleting pause: %w", err)
	}

	cmd := m.shardedMgr.s.r.B().Srem().Key(m.shardedMgr.s.kg.RunPauses(ctx, p.Identifier.RunID)).Member(p.ID.String()).Build()
	err = m.shardedMgr.s.r.Do(ctx, cmd).Error()
	if err != nil {
		return fmt.Errorf("error deleting pause: %w", err)
	}

	cmd = m.shardedMgr.s.r.B().Del().Key(m.shardedMgr.s.kg.PauseStep(ctx, p.Identifier, p.Incoming)).Build()
	err = m.shardedMgr.s.r.Do(ctx, cmd).Error()
	if err != nil {
		return fmt.Errorf("error deleting pause: %w", err)
	}

	switch status {
	case 0:
		return nil
	default:
		return fmt.Errorf("unknown response deleting pause: %d", status)
	}
}

func (m *unshardedMgr) CleanUpPause(ctx context.Context, p *state.Pause, runID ulid.ULID) error {
	// Add a default event here, which is null and overwritten by everything.
	// This is necessary to keep the same cluster key.
	eventKey := u.u.kg.PauseEvent(ctx, p.WorkspaceID, "-")
	if p.Event != nil {
		eventKey = m.u.kg.PauseEvent(ctx, p.WorkspaceID, *p.Event)
	}

	// For pause indexes.
	evt := ""
	if p.Event != nil && (p.InvokeCorrelationID == nil || *p.InvokeCorrelationID == "") {
		evt = *p.Event
	}

	keys := []string{
		m.u.kg.PauseStep(ctx, p.Identifier, p.Incoming),
		eventKey,
		m.u.kg.Invoke(ctx, p.WorkspaceID),
		m.u.kg.PauseIndex(ctx, "add", p.WorkspaceID, evt),
		m.u.kg.PauseIndex(ctx, "exp", p.WorkspaceID, evt),
		m.u.kg.RunPauses(ctx, p.Identifier.RunID),
	}

	corrId := ""
	if p.InvokeCorrelationID != nil && *p.InvokeCorrelationID != "" {
		corrId = *p.InvokeCorrelationID
	}

	args, err := StrSlice([]any{
		p.ID.String(),
		corrId,
	})
	if err != nil {
		return fmt.Errorf("error generating arguments for running cleanUpPause script: %w", err)
	}

	err = scripts["cleanUpPause"].Exec(
		ctx,
		m.u.r,
		keys,
		args,
	).Error()
	if err != nil {
		return fmt.Errorf("error running cleanUpPause script: %w", err)
	}

	return nil
}

func (m *shardedMgr) ConsumePause(ctx context.Context, pauseID uuid.UUID, runID ulid.ULID, data any) (*state.Pause, error) {
	p, err := m.PauseByID(ctx, pauseID, runID)
	if err != nil {
		return p, err
	}

	marshalledData, err := json.Marshal(data)
	if err != nil {
		return p, fmt.Errorf("cannot marshal data to store in state: %w", err)
	}

	keys := []string{
		m.s.kg.PauseID(ctx, pauseID, runID),
		m.s.kg.Actions(ctx, p.Identifier),
		m.s.kg.Stack(ctx, p.Identifier.RunID),
		m.s.kg.RunMetadata(ctx, p.Identifier.RunID),
	}

	args, err := StrSlice([]any{
		// pauseID.String(),
		// corrId,
		p.DataKey,
		string(marshalledData),
	})
	if err != nil {
		return p, err
	}

	status, err := scripts["consumePause"].Exec(
		ctx,
		m.s.r,
		keys,
		args,
	).AsInt64()
	if err != nil {
		return p, fmt.Errorf("error consuming pause: %w", err)
	}
	switch status {
	case 0:
		return p, nil
	case 1:
		return p, state.ErrPauseNotFound // 🤔
	default:
		return p, fmt.Errorf("unknown response leasing pause: %d", status)
	}
}

func (m mgr) ConsumePause(ctx context.Context, pauseID uuid.UUID, runID ulid.ULID, data any) error {
	p, err := m.shardedMgr.ConsumePause(ctx, pauseID, runID, data)
	if err != nil {
		return err
	}

	// The pause was now consumed, so let's clean up
	return m.unshardedMgr.CleanUpPause(ctx, p, runID)
}

func (m unshardedMgr) EventHasPauses(ctx context.Context, workspaceID uuid.UUID, event string) (bool, error) {
	key := m.u.kg.PauseEvent(ctx, workspaceID, event)
	cmd := m.pauseR.B().Exists().Key(key).Build()
	return m.pauseR.Do(ctx, cmd).AsBool()
}

func (m *shardedMgr) PauseByID(ctx context.Context, pauseID uuid.UUID, runID ulid.ULID) (*state.Pause, error) {
	cmd := m.s.r.B().Get().Key(m.s.kg.PauseID(ctx, pauseID, runID)).Build()
	str, err := m.s.r.Do(ctx, cmd).ToString()
	if err == rueidis.Nil {
		return nil, state.ErrPauseNotFound
	}
	if err != nil {
		return nil, err
	}
	pause := &state.Pause{}
	err = json.Unmarshal([]byte(str), pause)
	return pause, err
}

func (m unshardedMgr) PauseByInvokeCorrelationID(ctx context.Context, wsID uuid.UUID, correlationID string) (*state.Pause, error) {
	key := m.u.kg.Invoke(ctx, wsID)
	cmd := m.pauseR.B().Hget().Key(key).Field(correlationID).Build()
	pauseIDstr, err := m.pauseR.Do(ctx, cmd).ToString()
	if err == rueidis.Nil {
		return nil, state.ErrInvokePauseNotFound
	}
	if err != nil {
		return nil, err
	}

	pauseID, err := uuid.Parse(pauseIDstr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pauseID UUID: %w", err)
	}
	return m.PauseByID(ctx, pauseID)
}

func (m unshardedMgr) PausesByID(ctx context.Context, ids ...uuid.UUID) ([]*state.Pause, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	keys := make([]string, len(ids))
	for n, id := range ids {
		keys[n] = m.u.kg.PauseID(ctx, id)
	}

	cmd := m.pauseR.B().Mget().Key(keys...).Build()
	strings, err := m.pauseR.Do(ctx, cmd).AsStrSlice()
	if err == rueidis.Nil {
		return nil, state.ErrPauseNotFound
	}
	if err != nil {
		return nil, err
	}

	var merr error

	pauses := []*state.Pause{}
	for _, item := range strings {
		if len(item) == 0 {
			continue
		}

		pause := &state.Pause{}
		err = json.Unmarshal([]byte(item), pause)
		if err != nil {
			merr = errors.Join(merr, err)
			continue
		}
		pauses = append(pauses, pause)
	}

	return pauses, merr
}

// PauseByStep returns a specific pause for a given workflow run, from a given step.
//
// This is required when continuing a step function from an async step, ie. one that
// has deferred results which must be continued by resuming the specific pause set
// up for the given step ID.
func (m mgr) PauseByStep(ctx context.Context, i state.Identifier, actionID string) (*state.Pause, error) {
	// Access sharded value first
	cmd := m.shardedMgr.s.r.B().Get().Key(m.shardedMgr.s.kg.PauseStep(ctx, i, actionID)).Build()
	str, err := m.shardedMgr.s.r.Do(ctx, cmd).ToString()

	if err == rueidis.Nil {
		return nil, state.ErrPauseNotFound
	}
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(str)
	if err != nil {
		return nil, err
	}

	// Then access unsharded value
	cmd = m.unshardedMgr.pauseR.B().Get().Key(m.u.kg.PauseID(ctx, id)).Build()
	byt, err := m.unshardedMgr.pauseR.Do(ctx, cmd).AsBytes()

	if err == rueidis.Nil {
		return nil, state.ErrPauseNotFound
	}
	if err != nil {
		return nil, err
	}

	pause := &state.Pause{}
	err = json.Unmarshal(byt, pause)
	return pause, err
}

// PausesByEvent returns all pauses for a given event within a workspace.
func (m unshardedMgr) PausesByEvent(ctx context.Context, workspaceID uuid.UUID, event string) (state.PauseIterator, error) {
	key := m.u.kg.PauseEvent(ctx, workspaceID, event)
	// If there are > 1000 keys in the hmap, use scanning

	cntCmd := m.pauseR.B().Hlen().Key(key).Build()
	cnt, err := m.pauseR.Do(ctx, cntCmd).AsInt64()

	if err != nil || cnt > 1000 {
		key := m.u.kg.PauseEvent(ctx, workspaceID, event)
		iter := &scanIter{
			count: cnt,
			r:     m.pauseR,
		}
		err := iter.init(ctx, key, 1000)
		return iter, err
	}

	// If there are less than a thousand items, query the keys
	// for iteration.
	iter := &bufIter{r: m.pauseR}
	err = iter.init(ctx, key)
	return iter, err
}

func (m unshardedMgr) PausesByEventSince(ctx context.Context, workspaceID uuid.UUID, event string, since time.Time) (state.PauseIterator, error) {
	if since.IsZero() {
		return m.PausesByEvent(ctx, workspaceID, event)
	}

	// Load all items in the set.
	cmd := m.u.r.B().
		Zrangebyscore().
		Key(m.u.kg.PauseIndex(ctx, "add", workspaceID, event)).
		Min(strconv.Itoa(int(since.Unix()))).
		Max("+inf").
		Build()
	ids, err := m.u.r.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return nil, err
	}

	iter := &keyIter{
		r:  m.u.r,
		kf: m.u.kg,
	}
	err = iter.init(ctx, ids, 100)
	return iter, err
}

func (m unshardedMgr) EvaluablesByID(ctx context.Context, ids ...uuid.UUID) ([]expr.Evaluable, error) {
	items, err := m.PausesByID(ctx, ids...)
	if err != nil {
		return nil, err
	}
	evaluables := make([]expr.Evaluable, len(items))
	for n, i := range items {
		evaluables[n] = i
	}
	return evaluables, nil
}

func (m mgr) LoadEvaluablesSince(ctx context.Context, workspaceID uuid.UUID, eventName string, since time.Time, do func(context.Context, expr.Evaluable) error) error {

	// Keep a list of pauses that should be deleted because they've expired.
	//
	// Note that we don't do this in the iteration loop, as redis can use either HSCAN or
	// MGET;  deleting during iteration may lead to skipped items.
	expired := []*state.Pause{}

	it, err := m.PausesByEventSince(ctx, workspaceID, eventName, since)
	if err != nil {
		return err
	}
	for it.Next(ctx) {
		pause := it.Val(ctx)
		if pause == nil {
			continue
		}

		if pause.Expires.Time().Before(time.Now()) {
			expired = append(expired, pause)
			continue
		}

		if err := do(ctx, pause); err != nil {
			return err
		}
	}

	// GC pauses on fetch.
	for _, pause := range expired {
		_ = m.DeletePause(ctx, *pause)
	}

	if it.Error() != context.Canceled && it.Error() != scanDoneErr {
		return it.Error()
	}

	return nil
}

type bufIter struct {
	r     rueidis.Client
	items []string

	val *state.Pause
	err error

	l sync.Mutex
}

func (i *bufIter) init(ctx context.Context, key string) error {
	var err error
	// If there are less than a thousand items, query the keys
	// for iteration.
	cmd := i.r.B().Hvals().Key(key).Build()
	i.items, err = i.r.Do(ctx, cmd).AsStrSlice()
	i.l = sync.Mutex{}
	return err
}

func (i *bufIter) Count() int {
	return len(i.items)
}

func (i *bufIter) Next(ctx context.Context) bool {
	i.l.Lock()
	defer i.l.Unlock()

	if len(i.items) == 0 {
		i.err = context.Canceled
		return false
	}

	pause := &state.Pause{}
	i.err = json.Unmarshal([]byte(i.items[0]), pause)
	i.val = pause
	// Remove one from the slice.
	i.items = i.items[1:]
	return i.err == nil
}

// Buffer by running an MGET to get the values of the pauses.
func (i *bufIter) Val(ctx context.Context) *state.Pause {
	return i.val
}

func (i *bufIter) Error() error {
	return i.err
}

var scanDoneErr = fmt.Errorf("scan done")

type scanIter struct {
	r   rueidis.Client
	key string
	// chunk is the size of scans to load in one.
	chunk int64

	// count is the cached number of items to return in Count(),
	// ie the hlen result when creating the iterator.
	count int64

	// iterator fields
	i      int
	cursor int
	vals   rueidis.ScanEntry
	err    error

	l sync.Mutex
}

func (i *scanIter) Error() error {
	return i.err
}

func (i *scanIter) init(ctx context.Context, key string, chunk int64) error {
	i.key = key
	i.chunk = chunk
	cmd := i.r.B().Hscan().Key(key).Cursor(0).Count(i.chunk).Build()
	scan, err := i.r.Do(ctx, cmd).AsScanEntry()
	if err != nil {
		i.err = err
		return err
	}
	i.cursor = int(scan.Cursor)
	i.vals = scan
	i.i = -1
	i.l = sync.Mutex{}
	return nil
}

func (i *scanIter) Count() int {
	return int(i.count)
}

func (i *scanIter) fetch(ctx context.Context) error {
	// Reset the index.
	i.i = -1

	if i.cursor == 0 {
		// We're done, no need to fetch.
		return scanDoneErr
	}

	// Scan 100 times up until there are values
	for scans := 0; scans < 100; scans++ {
		cmd := i.r.B().Hscan().
			Key(i.key).
			Cursor(uint64(i.cursor)).
			Count(i.chunk).
			Build()

		scan, err := i.r.Do(ctx, cmd).AsScanEntry()
		if err != nil {
			return err
		}

		i.cursor = int(scan.Cursor)
		i.vals = scan

		if len(i.vals.Elements) > 0 {
			return nil
		}
	}

	return fmt.Errorf("Scanned max times without finding pause values")
}

func (i *scanIter) Next(ctx context.Context) bool {
	i.l.Lock()
	defer i.l.Unlock()

	if i.i >= (len(i.vals.Elements) - 1) {
		err := i.fetch(ctx)
		if err == scanDoneErr {
			// No more present.
			i.err = context.Canceled
			return false
		}
		if err != nil {
			i.err = err
			// Stop iterating, set error.
			return false
		}
	}
	// Skip the ID
	i.i++
	// Get the value.
	i.i++
	return true
}

func (i *scanIter) Val(ctx context.Context) *state.Pause {
	if i.i == -1 || i.i >= len(i.vals.Elements) {
		return nil
	}

	val := i.vals.Elements[i.i]
	if val == "" {
		return nil
	}

	pause := &state.Pause{}
	err := json.Unmarshal([]byte(val), pause)
	if err != nil {
		return nil
	}
	return pause
}

func newRunMetadata(data map[string]string) (*runMetadata, error) {
	var err error
	m := &runMetadata{}

	// The V1 state identifier is the most important thing to be stored in state.  We must have this
	// as it contains tenant information.
	val, ok := data["id"]
	if !ok || val == "" {
		return nil, state.ErrRunNotFound
	}
	id := state.Identifier{}
	if err := json.Unmarshal([]byte(val), &id); err != nil {
		return nil, fmt.Errorf("unable to unmarshal metadata identifier: %s", val)
	}
	m.Identifier = id

	// Handle everything else optimistically
	v, ok := data["status"]
	if !ok {
		return nil, fmt.Errorf("no status stored in metadata")
	}
	status, err := strconv.Atoi(v)
	if err != nil {
		return nil, fmt.Errorf("invalid function status stored in run metadata: %#v", v)
	}
	m.Status = enums.RunStatus(status)

	parseInt := func(v string) (int, error) {
		str, ok := data[v]
		if !ok {
			return 0, fmt.Errorf("no '%s' stored in run metadata", v)
		}
		val, err := strconv.Atoi(str)
		if err != nil {
			return 0, fmt.Errorf("invalid '%s' stored in run metadata", v)
		}
		return val, nil
	}

	m.StateSize, _ = parseInt("state_size")
	m.EventSize, _ = parseInt("event_size")
	m.StepCount, _ = parseInt("step_count")

	if val, ok := data["version"]; ok && val != "" {
		v, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid metadata version detected: %#v", val)
		}

		m.Version = v
	}

	if val, ok := data["rv"]; ok && val != "" {
		v, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid hash version detected: %#v", val)
		}
		m.RequestVersion = v
	}

	if val, ok := data["sat"]; ok && val != "" {
		v, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid started at timestamp detected: %#v", val)
		}
		m.StartedAt = v
	}

	// The below fields are optional
	if val, ok := data["debugger"]; ok {
		if val == "true" || val == "1" {
			m.Debugger = true
		}
	}
	if val, ok := data["runType"]; ok {
		m.RunType = val
	}
	if val, ok := data["id"]; ok && val != "" {
		id := state.Identifier{}
		if err := json.Unmarshal([]byte(val), &id); err != nil {
			return nil, fmt.Errorf("unable to unmarshal metadata identifier: %s", val)
		}
		m.Identifier = id
	}
	if val, ok := data["ctx"]; ok && val != "" {
		ctx := map[string]any{}
		if err := json.Unmarshal([]byte(val), &ctx); err != nil {
			return nil, fmt.Errorf("unable to unmarshal metadata context: %s", val)
		}
		m.Context = ctx
	}
	if val, ok := data["die"]; ok {
		if val == "true" || val == "1" {
			m.DisableImmediateExecution = true
		}
	}
	if val, ok := data["sid"]; ok {
		m.SpanID = val
	}

	return m, nil
}

// keyIter loads all pauses in batches given a list of IDs
type keyIter struct {
	r  rueidis.Client
	kf UnshardedKeyGenerator
	// chunk is the size of scans to load in one.
	chunk int64
	// keys stores pause IDs to fetch in batches
	keys []string
	// vals stores pauses as strings from MGET
	vals []string
	err  error
}

func (i *keyIter) Error() error {
	return i.err
}

func (i *keyIter) init(ctx context.Context, keys []string, chunk int64) error {
	i.keys = keys
	i.chunk = chunk
	err := i.fetch(ctx)
	if err == scanDoneErr {
		return nil
	}
	return err
}

func (i *keyIter) Count() int {
	return len(i.keys)
}

func (i *keyIter) fetch(ctx context.Context) error {
	if len(i.keys) == 0 {
		// No more present.
		i.err = context.Canceled
		return scanDoneErr
	}

	var load []string
	if len(i.keys) > int(i.chunk) {
		load = i.keys[0:i.chunk]
		i.keys = i.keys[i.chunk:]
	} else {
		load = i.keys[:]
		i.keys = []string{}
	}

	for n, id := range load {
		load[n] = i.kf.PauseID(ctx, uuid.MustParse(id))
	}

	cmd := i.r.B().Mget().Key(load...).Build()
	i.vals, i.err = i.r.Do(ctx, cmd).AsStrSlice()
	if rueidis.IsRedisNil(i.err) {
		// Somehow none of these pauses no longer exist, which is okay:
		// another concurrent thread may have already consumed it.
		i.err = nil
	}
	return i.err
}

func (i *keyIter) Next(ctx context.Context) bool {
	if len(i.vals) > 0 {
		return true
	}

	err := i.fetch(ctx)
	if err == scanDoneErr {
		return false
	}
	return err == nil
}

func (i *keyIter) Val(ctx context.Context) *state.Pause {
	if len(i.vals) == 0 {
		return nil
	}

	val := i.vals[0]
	i.vals = i.vals[1:]
	if val == "" {
		return nil
	}

	pause := &state.Pause{}
	err := json.Unmarshal([]byte(val), pause)
	if err != nil {
		return nil
	}
	return pause
}

// runMetadata is stored for each invocation of a function.  This is inserted when
// creating a new run, and stores the triggering event as well as workflow-specific
// metadata for the invocation.
type runMetadata struct {
	Identifier state.Identifier `json:"id"`
	Status     enums.RunStatus  `json:"status"`
	// These are the fields for standard state metadata.
	StateSize                 int            `json:"state_size"`
	EventSize                 int            `json:"event_size"`
	StepCount                 int            `json:"step_count"`
	Debugger                  bool           `json:"debugger"`
	RunType                   string         `json:"runType,omitempty"`
	ReplayID                  string         `json:"rID,omitempty"`
	Version                   int            `json:"version"`
	RequestVersion            int            `json:"rv"`
	Context                   map[string]any `json:"ctx,omitempty"`
	DisableImmediateExecution bool           `json:"die,omitempty"`
	SpanID                    string         `json:"sid"`
	StartedAt                 int64          `json:"sat,omitempty"`
}

func (r runMetadata) Map() map[string]any {
	return map[string]any{
		"id":       r.Identifier,
		"status":   int(r.Status), // Always store this as an int
		"debugger": r.Debugger,
		"runType":  r.RunType,
		"version":  r.Version,
		"rv":       r.RequestVersion,
		"ctx":      r.Context,
		"die":      r.DisableImmediateExecution,
		"sid":      r.SpanID,
		"sat":      r.StartedAt,
	}
}

func (r runMetadata) Metadata() state.Metadata {
	m := state.Metadata{
		Identifier:                r.Identifier,
		Debugger:                  r.Debugger,
		Status:                    r.Status,
		Version:                   r.Version,
		RequestVersion:            r.RequestVersion,
		Context:                   r.Context,
		DisableImmediateExecution: r.DisableImmediateExecution,
		SpanID:                    r.SpanID,
	}
	// 0 != time.IsZero
	// only convert to time if runMetadata's StartedAt is > 0
	if r.StartedAt > 0 {
		m.StartedAt = time.UnixMilli(r.StartedAt)
	}

	if r.RunType != "" {
		m.RunType = &r.RunType
	}
	return m
}

func StrSlice(args []any) ([]string, error) {
	res := make([]string, len(args))
	for i, item := range args {
		if s, ok := item.(fmt.Stringer); ok {
			res[i] = s.String()
			continue
		}

		switch v := item.(type) {
		case string:
			res[i] = v
		case []byte:
			res[i] = rueidis.BinaryString(v)
		case int:
			res[i] = strconv.Itoa(v)
		case bool:
			// Use 1 and 0 to signify true/false.
			if v {
				res[i] = "1"
			} else {
				res[i] = "0"
			}
		default:
			byt, err := json.Marshal(item)
			if err != nil {
				return nil, err
			}
			res[i] = rueidis.BinaryString(byt)
		}
	}
	return res, nil
}
