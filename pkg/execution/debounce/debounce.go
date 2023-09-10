package debounce

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/xhit/go-str2duration/v2"
)

//go:embed lua/*
var embedded embed.FS

var (
	ErrDebounceExists   = fmt.Errorf("a debounce exists for this function")
	ErrDebounceNotFound = fmt.Errorf("debounce not found")
)

var (
	buffer = 2 * time.Second
	// scripts stores all embedded lua scripts on initialization
	scripts = map[string]*rueidis.Lua{}
	include = regexp.MustCompile(`-- \$include\(([\w.]+)\)`)
)

func init() {
	// read the lua scripts
	entries, err := embedded.ReadDir("lua")
	if err != nil {
		panic(fmt.Errorf("error reading redis lua dir: %w", err))
	}
	readRedisScripts("lua", entries)
}

// The general strategy for debounce:
//
// 1. Create a new debounce key.
// 2. Store the current event in the debounce key.
// 3. Create a new queue item for the debounce, linking to the debounce key

// DebounceItem represents a debounce stored within the debounce manager.
//
// DebounceItem fulfils event.TrackedEvent, allowing the use of the entire DebounceItem
// as the triggering event data passed to executor.Schedule.
type DebounceItem struct {
	// AccountID represents the account for the debounce item
	AccountID uuid.UUID `json:"aID"`
	// WorkspaceID represents the account for the debounce item
	WorkspaceID uuid.UUID `json:"wsID"`
	// FunctionID represents the function ID that this debounce is for.
	FunctionID uuid.UUID `json:"fnID"`
	// EventID represents the internal event ID that triggers the function.
	EventID ulid.ULID `json:"eID"`
	// Event represents the event data which triggers the function.
	Event event.Event `json:"e"`
}

func (d DebounceItem) QueuePayload() DebouncePayload {
	return DebouncePayload{
		AccountID:   d.AccountID,
		WorkspaceID: d.WorkspaceID,
		FunctionID:  d.FunctionID,
	}
}

func (d DebounceItem) GetInternalID() ulid.ULID {
	return d.EventID
}

func (d DebounceItem) GetEvent() event.Event {
	return d.Event
}

// DebouncePayload represents the data stored within the queue's payload.
type DebouncePayload struct {
	DebounceID ulid.ULID `json:"debounceID"`
	// AccountID represents the account for the debounce item
	AccountID uuid.UUID `json:"aID"`
	// WorkspaceID represents the account for the debounce item
	WorkspaceID uuid.UUID `json:"wsID"`
	// FunctionID represents the function ID that this debounce is for.
	FunctionID uuid.UUID `json:"fnID"`
}

type Debouncer interface {
	Debounce(ctx context.Context, d DebounceItem, fn inngest.Function) error
	GetDebounceItem(ctx context.Context, debounceID ulid.ULID) (*DebounceItem, error)
}

func NewRedisDebouncer(r rueidis.Client, k redis_state.DebounceKeyGenerator, q redis_state.QueueManager) Debouncer {
	return debouncer{
		r: r,
		k: k,
		q: q,
	}
}

type debouncer struct {
	r rueidis.Client
	k redis_state.DebounceKeyGenerator
	q redis_state.QueueManager
}

func (d debouncer) GetDebounceItem(ctx context.Context, debounceID ulid.ULID) (*DebounceItem, error) {
	keyDbc := d.k.Debounce(ctx)

	cmd := d.r.B().Hget().Key(keyDbc).Field(debounceID.String()).Build()
	byt, err := d.r.Do(ctx, cmd).AsBytes()
	if rueidis.IsRedisNil(err) {
		return nil, ErrDebounceNotFound
	}

	di := &DebounceItem{}
	if err := json.Unmarshal(byt, &di); err != nil {
		return nil, fmt.Errorf("error unmarshalling debounce item: %w", err)
	}
	return di, nil
}

func (d debouncer) Debounce(ctx context.Context, di DebounceItem, fn inngest.Function) error {
	if fn.Debounce == nil {
		return fmt.Errorf("fn has no debounce config")
	}
	ttl, err := str2duration.ParseDuration(fn.Debounce.Period)
	if err != nil {
		return fmt.Errorf("invalid debounce duration: %w", err)
	}
	return d.debounce(ctx, di, fn, ttl, 0)
}

func (d debouncer) debounce(ctx context.Context, di DebounceItem, fn inngest.Function, ttl time.Duration, n int) error {
	// Call new debounce immediately.  If this returns ErrDebounceExists then
	// update the debounce.  This ensures that checking and creating a debounce
	// is atomic, and two individual threads/workers cannot create debounces simultaneously.
	debounceID, err := d.newDebounce(ctx, di, fn, ttl)
	if err == nil {
		return nil
	}
	if err != ErrDebounceExists {
		// There was an unkown error creating the debounce.
		return err
	}
	if debounceID == nil {
		return fmt.Errorf("expected debounce ID when debounce exists")
	}

	// A debounce must already exist for this fn.  Update it.
	err = d.updateDebounce(ctx, di, fn, ttl, *debounceID)
	if err == context.DeadlineExceeded {
		if n == 4 {
			// Only recurse 5 times.
			return fmt.Errorf("unable to update debounce: %w", err)
		}
		// Re-invoke this to see if we need to extend the debounce or continue.
		return d.debounce(ctx, di, fn, ttl, n+1)
	}

	return err
}

func (d debouncer) queueItem(ctx context.Context, di DebounceItem, debounceID ulid.ULID) queue.Item {
	jobID := debounceID.String()
	payload := di.QueuePayload()
	payload.DebounceID = debounceID
	return queue.Item{
		JobID:       &jobID,
		WorkspaceID: di.WorkspaceID,
		Identifier: state.Identifier{
			AccountID:   di.AccountID,
			WorkspaceID: di.WorkspaceID,
			WorkflowID:  di.FunctionID,
		},
		Kind:    queue.KindDebounce,
		Payload: payload,
	}
}

func (d debouncer) newDebounce(ctx context.Context, di DebounceItem, fn inngest.Function, ttl time.Duration) (*ulid.ULID, error) {
	now := time.Now()
	debounceID := ulid.MustNew(ulid.Now(), rand.Reader)

	key, err := d.debounceKey(ctx, di, fn)
	if err != nil {
		return nil, err
	}

	keyPtr := d.k.DebouncePointer(ctx, fn.ID, key)
	keyDbc := d.k.Debounce(ctx)

	byt, err := json.Marshal(di)
	if err != nil {
		return nil, fmt.Errorf("error marshalling debounce: %w", err)
	}

	out, err := scripts["newDebounce"].Exec(
		ctx,
		d.r,
		[]string{keyPtr, keyDbc},
		[]string{debounceID.String(), string(byt), strconv.Itoa(int(ttl.Seconds()))},
	).ToString()
	if err != nil {
		return nil, fmt.Errorf("error creating debounce: %w", err)
	}

	if out == "0" {
		// Enqueue the debounce job with extra buffer *plus* one second.  This ensures that we never
		// attempt to start a debounce during the debounce's expiry (race conditions), and the extra
		// second lets an updateDebounce call on TTL 0 finish, as the buffer is the updateDebounce
		// deadline.
		qi := d.queueItem(ctx, di, debounceID)
		err = d.q.Enqueue(ctx, qi, now.Add(ttl).Add(buffer).Add(time.Second))
		if err != nil {
			return &debounceID, fmt.Errorf("error enqueueing debounce job: %w", err)
		}
		return &debounceID, nil
	}

	existingID, err := ulid.Parse(out)
	if err != nil {
		// This was not a ULID, so we have no idea what was returned.
		return nil, fmt.Errorf("unknown new debounce return value: %s", out)
	}
	return &existingID, ErrDebounceExists
}

// updateDebounce updates the currently pending debounce to point to the new event ID.  It pushes
// out the debounce's TTL, and re-enqueues the job to initialize fns from the debounce.
func (d debouncer) updateDebounce(ctx context.Context, di DebounceItem, fn inngest.Function, ttl time.Duration, debounceID ulid.ULID) error {
	now := time.Now()

	key, err := d.debounceKey(ctx, di, fn)
	if err != nil {
		return err
	}

	// NOTE: This functioon has a deadline to complete.  If this fn doesn't complete within the deadline,
	// eg, network issues, we must check if the debounce expired and re-attempt the entire thing.
	ctx, cancel := context.WithTimeout(ctx, buffer)
	defer cancel()

	keyPtr := d.k.DebouncePointer(ctx, fn.ID, key)
	keyDbc := d.k.Debounce(ctx)
	byt, err := json.Marshal(di)
	if err != nil {
		return fmt.Errorf("error marshalling debounce: %w", err)
	}

	out, err := scripts["updateDebounce"].Exec(
		ctx,
		d.r,
		[]string{keyPtr, keyDbc},
		[]string{debounceID.String(), string(byt), strconv.Itoa(int(ttl.Seconds()))},
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error creating debounce: %w", err)
	}
	switch out {
	case 0:
		err = d.q.RequeueByJobID(
			ctx,
			fn.ID.String(),
			debounceID.String(),
			now.Add(ttl).Add(buffer).Add(time.Second),
		)
		if err != nil {
			return fmt.Errorf("error requeueing debounce job '%s': %w", debounceID, err)
		}
		return nil
	default:
		return fmt.Errorf("unknown update debounce return value: %d", out)
	}
}

func (d debouncer) debounceKey(ctx context.Context, evt event.TrackedEvent, fn inngest.Function) (string, error) {
	out, _, err := expressions.Evaluate(ctx, fn.Debounce.Key, map[string]any{"event": evt.GetEvent().Map()})
	if err != nil {
		return "", fmt.Errorf("invalid debounce expression: %w", err)
	}
	if str, ok := out.(string); ok {
		return str, nil
	}
	return fmt.Sprintf("%v", out), nil
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
