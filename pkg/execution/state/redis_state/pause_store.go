package redis_state

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/expr"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

// PauseStore implements pause operations using Redis.
type PauseStore struct {
	// unsharded is used for all pause-specific operations.
	unsharded *UnshardedClient
}

// NewPauseStore creates a new PauseStore with the given clients.
func NewPauseStore(unsharded *UnshardedClient) *PauseStore {
	return &PauseStore{unsharded: unsharded}
}

// PauseCreatedAt returns the timestamp a pause was created, using the given
// workspace <> event Index.
func (s *PauseStore) PauseCreatedAt(ctx context.Context, workspaceID uuid.UUID, event string, pauseID uuid.UUID) (time.Time, error) {
	pc := s.unsharded.Pauses()
	idx := pc.kg.PauseIndex(ctx, "add", workspaceID, event)
	result, err := pc.Client().Do(ctx, pc.Client().B().Zmscore().Key(idx).Member(pauseID.String()).Build()).ToArray()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return time.Time{}, state.ErrPauseNotFound
		}
		return time.Time{}, err
	}

	if len(result) == 0 {
		return time.Time{}, state.ErrPauseNotFound
	}

	// ZMSCORE returns nil for non-existent members
	if result[0].IsNil() {
		return time.Time{}, state.ErrPauseNotFound
	}

	ts, err := result[0].AsInt64()
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(ts, 0), nil
}

func (s *PauseStore) SavePause(ctx context.Context, p state.Pause) (int64, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "SavePause"), redis_telemetry.ScopePauses)

	// `evt` is used to search for pauses based on event names. We only want to
	// do this if this pause is not part of an invoke. If it is, we don't want
	// to index it by event name as the pause will be processed by correlation
	// ID.
	evt := ""
	if p.Event != nil && (p.InvokeCorrelationID == nil || *p.InvokeCorrelationID == "") {
		evt = *p.Event
	}

	invokeCorrId := ""
	if p.InvokeCorrelationID != nil {
		invokeCorrId = *p.InvokeCorrelationID
	}

	signalCorrId := ""
	if p.SignalID != nil {
		signalCorrId = *p.SignalID
	}

	extendedExpiry := time.Until(p.Expires.Time().Add(10 * time.Minute)).Seconds()

	createdAt := p.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	nowUnixSeconds := createdAt.Unix()
	p.CreatedAt = createdAt

	packed, err := json.Marshal(p)
	if err != nil {
		return 0, err
	}

	pause := s.unsharded.Pauses()

	// Warning: We need to access global keys, which must be colocated on the same Redis cluster
	global := s.unsharded.Global()

	keys := []string{
		pause.kg.Pause(ctx, p.ID),
		pause.kg.PauseEvent(ctx, p.WorkspaceID, evt),
		global.kg.Invoke(ctx, p.WorkspaceID),
		global.kg.Signal(ctx, p.WorkspaceID),
		pause.kg.PauseIndex(ctx, "add", p.WorkspaceID, evt),
		pause.kg.PauseIndex(ctx, "exp", p.WorkspaceID, evt),
		pause.kg.RunPauses(ctx, p.Identifier.RunID),
		pause.kg.GlobalPauseIndex(ctx),
	}

	replaceSignalOnConflict := "0"
	if p.ReplaceSignalOnConflict {
		replaceSignalOnConflict = "1"
	}

	args, err := StrSlice([]any{
		string(packed),
		p.ID.String(),
		evt,
		invokeCorrId,
		signalCorrId,
		// Add at least 10 minutes to this pause, allowing us to process the
		// pause by ID for 10 minutes past expiry.
		int(extendedExpiry),
		nowUnixSeconds,
		replaceSignalOnConflict,
	})
	if err != nil {
		return 0, err
	}

	status, err := scripts["savePause"].Exec(
		redis_telemetry.WithScriptName(ctx, "savePause"),
		pause.Client(),
		keys,
		args,
	).AsInt64()
	if err != nil {
		if err.Error() == "ErrSignalConflict" {
			return 0, state.ErrSignalConflict
		}

		return 0, fmt.Errorf("error finalizing: %w", err)
	}

	switch status {
	case -1:
		return status, state.ErrPauseAlreadyExists
	default:
		return status, nil
	}
}

func (s *PauseStore) LeasePause(ctx context.Context, id uuid.UUID) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "LeasePause"), redis_telemetry.ScopePauses)

	args, err := StrSlice([]any{
		time.Now().UnixMilli(),
		state.PauseLeaseDuration.Seconds(),
	})
	if err != nil {
		return err
	}

	pause := s.unsharded.Pauses()

	status, err := scripts["leasePause"].Exec(
		redis_telemetry.WithScriptName(ctx, "leasePause"),
		pause.Client(),
		// keys will be sharded/unsharded depending on RunID
		[]string{pause.kg.PauseLease(ctx, id)},
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
	// case 2:
	//  NOTE: This is now not possible, as we flush blocks from redis to a backing block store
	//  meaning that pauses may never be found,
	// 	return state.ErrPauseNotFound
	default:
		return fmt.Errorf("unknown response leasing pause: %d", status)
	}
}

func (s *PauseStore) DeletePauseByID(ctx context.Context, pauseID uuid.UUID, workspaceID uuid.UUID) error {
	// Attempt to fetch this pause.
	pause, err := s.PauseByID(ctx, pauseID)
	if err != nil {
		if err == state.ErrPauseNotFound {
			// pause doesn't exist, nothing to delete
			return nil
		}
		// bubble the error up we can safely retry the whole process
		return err
	}
	return s.DeletePause(ctx, *pause)
}

func (s *PauseStore) PauseIDsForRun(ctx context.Context, runID ulid.ULID) ([]uuid.UUID, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PauseIDsForRun"), redis_telemetry.ScopePauses)

	pause := s.unsharded.Pauses()
	pauseIDStrs, err := pause.Client().Do(ctx, pause.Client().B().Smembers().Key(pause.kg.RunPauses(ctx, runID)).Build()).AsStrSlice()
	if err != nil {
		return nil, err
	}

	pauseIDs := make([]uuid.UUID, 0, len(pauseIDStrs))
	for _, id := range pauseIDStrs {
		pauseID, err := uuid.Parse(id)
		if err != nil {
			logger.StdlibLogger(ctx).Error("invalid pause ID in run pause set", "error", err, "pauseID", id, "runID", runID)
			continue
		}
		pauseIDs = append(pauseIDs, pauseID)
	}

	return pauseIDs, nil
}

func (s *PauseStore) DeleteRunPausesIndex(ctx context.Context, runID ulid.ULID) error {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "DeleteRunPausesIndex"), redis_telemetry.ScopePauses)

	pause := s.unsharded.Pauses()
	return pause.Client().Do(ctx, pause.Client().B().Del().Key(pause.kg.RunPauses(ctx, runID)).Build()).Error()
}

func (s *PauseStore) DeletePausesForRun(ctx context.Context, runID ulid.ULID, workspaceID uuid.UUID) error {
	pauseIDs, err := s.PauseIDsForRun(ctx, runID)
	if err != nil {
		return err
	}

	for _, pauseID := range pauseIDs {
		if err := s.DeletePauseByID(ctx, pauseID, workspaceID); err != nil {
			return err
		}
	}

	return s.DeleteRunPausesIndex(ctx, runID)
}

func (s *PauseStore) DeletePause(ctx context.Context, p state.Pause, options ...state.DeletePauseOpt) error {
	opts := state.DeletePauseOpts{}
	for _, fn := range options {
		fn(&opts)
	}

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "DeletePause"), redis_telemetry.ScopePauses)

	pause := s.unsharded.Pauses()

	global := s.unsharded.Global()

	// Add a default event here, which is null and overwritten by everything.  This is necessary
	// to keep the same cluster key.
	eventKey := pause.kg.PauseEvent(ctx, p.WorkspaceID, "-")
	if p.Event != nil {
		eventKey = pause.kg.PauseEvent(ctx, p.WorkspaceID, *p.Event)
	}

	evt := ""
	if p.Event != nil && (p.InvokeCorrelationID == nil || *p.InvokeCorrelationID == "") {
		evt = *p.Event
	}

	invokeCorrId := ""
	if p.InvokeCorrelationID != nil && *p.InvokeCorrelationID != "" {
		invokeCorrId = *p.InvokeCorrelationID
	}

	signalCorrId := ""
	if p.SignalID != nil {
		signalCorrId = *p.SignalID
	}

	pauseKey := pause.kg.Pause(ctx, p.ID)
	runPausesKey := pause.kg.RunPauses(ctx, p.Identifier.RunID)

	keys := []string{
		pauseKey,
		eventKey,
		// Warning: We need to access global keys, which must be colocated on the same Redis cluster
		global.kg.Invoke(ctx, p.WorkspaceID),
		global.kg.Signal(ctx, p.WorkspaceID),
		pause.kg.PauseIndex(ctx, "add", p.WorkspaceID, evt),
		pause.kg.PauseIndex(ctx, "exp", p.WorkspaceID, evt),
		runPausesKey,
		pause.kg.GlobalPauseIndex(ctx),
		pause.kg.PauseBlockIndex(ctx, p.ID),
	}

	// Marshal WriteBlockIndex to JSON if it has content, otherwise pass empty string
	blockIndexJSON := ""
	if opts.WriteBlockIndex.BlockID != "" {
		if blockIndexBytes, err := json.Marshal(opts.WriteBlockIndex); err == nil {
			blockIndexJSON = string(blockIndexBytes)
		}
	}

	status, err := scripts["deletePause"].Exec(
		redis_telemetry.WithScriptName(ctx, "deletePause"),
		pause.Client(),
		keys,
		[]string{
			p.ID.String(),
			invokeCorrId,
			signalCorrId,
			blockIndexJSON,
		},
	).AsInt64()
	if err != nil {
		return fmt.Errorf("error deleting pause: %w", err)
	}

	switch status {
	case 0:
		return nil
	case 1:
		return state.ErrPauseNotInBuffer
	default:
		return fmt.Errorf("unknown response deleting pause: %d", status)
	}
}

func (s *PauseStore) EventHasPauses(ctx context.Context, workspaceID uuid.UUID, event string) (bool, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "EventHasPauses"), redis_telemetry.ScopePauses)

	pause := s.unsharded.Pauses()
	key := pause.kg.PauseEvent(ctx, workspaceID, event)
	cmd := pause.Client().B().Exists().Key(key).Build()
	return pause.Client().Do(ctx, cmd).AsBool()
}

func (s *PauseStore) PauseByID(ctx context.Context, pauseID uuid.UUID) (*state.Pause, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PauseByID"), redis_telemetry.ScopePauses)

	pauses := s.unsharded.Pauses()
	cmd := pauses.Client().B().Get().Key(pauses.kg.Pause(ctx, pauseID)).Build()
	str, err := pauses.Client().Do(ctx, cmd).ToString()
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

func (s *PauseStore) PauseByInvokeCorrelationID(ctx context.Context, wsID uuid.UUID, correlationID string) (*state.Pause, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PauseByInvokeCorrelationID"), redis_telemetry.ScopePauses)

	global := s.unsharded.Global()
	key := global.kg.Invoke(ctx, wsID)
	cmd := global.Client().B().Hget().Key(key).Field(correlationID).Build()
	pauseIDstr, err := global.Client().Do(ctx, cmd).ToString()
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
	return s.PauseByID(ctx, pauseID)
}

func (s *PauseStore) PauseBySignalID(ctx context.Context, wsID uuid.UUID, signalID string) (*state.Pause, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PauseBySignalID"), redis_telemetry.ScopePauses)

	global := s.unsharded.Global()
	key := global.kg.Signal(ctx, wsID)
	cmd := global.Client().B().Hget().Key(key).Field(signalID).Build()
	pauseIDstr, err := global.Client().Do(ctx, cmd).ToString()
	if err == rueidis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get signalID: %w", err)
	}

	pauseID, err := uuid.Parse(pauseIDstr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pauseID UUID: %w", err)
	}

	p, err := s.PauseByID(ctx, pauseID)
	if err != nil {
		if err == state.ErrPauseNotFound {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get pause by ID: %w", err)
	}

	return p, nil
}

func (s *PauseStore) PauseLen(ctx context.Context, workspaceID uuid.UUID, event string) (int64, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PuaseLen"), redis_telemetry.ScopePauses)
	pauses := s.unsharded.Pauses()
	key := pauses.kg.PauseEvent(ctx, workspaceID, event)
	cntCmd := pauses.Client().B().Hlen().Key(key).Build()
	return pauses.Client().Do(ctx, cntCmd).AsInt64()
}

// PausesByEvent returns all pauses for a given event within a workspace.
func (s *PauseStore) PausesByEvent(ctx context.Context, workspaceID uuid.UUID, event string) (state.PauseIterator, error) {
	return s.pausesByEvent(ctx, workspaceID, event, time.Time{})
}

// pausesByEvent returns all pauses for a given event within a workspace.
func (s *PauseStore) pausesByEvent(ctx context.Context, workspaceID uuid.UUID, event string, aggregateStart time.Time) (state.PauseIterator, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PausesByEvent"), redis_telemetry.ScopePauses)

	pauses := s.unsharded.Pauses()

	key := pauses.kg.PauseEvent(ctx, workspaceID, event)
	// If there are > 1000 keys in the hmap, use scanning

	cntCmd := pauses.Client().B().Hlen().Key(key).Build()
	cnt, err := pauses.Client().Do(ctx, cntCmd).AsInt64()

	if err != nil || cnt > 1000 {
		key := pauses.kg.PauseEvent(ctx, workspaceID, event)
		iter := &scanIter{
			count:          cnt,
			r:              pauses.Client(),
			aggregateStart: aggregateStart,
		}
		err := iter.init(ctx, key, 1000)
		return iter, err
	}

	// If there are less than a thousand items, query the keys
	// for iteration.
	iter := &bufIter{r: pauses.Client(), aggregateStart: aggregateStart}
	err = iter.init(ctx, key)
	return iter, err
}

func (s *PauseStore) PausesByEventSince(ctx context.Context, workspaceID uuid.UUID, event string, since time.Time) (state.PauseIterator, error) {
	start := time.Now()

	if since.IsZero() {
		return s.pausesByEvent(ctx, workspaceID, event, start)
	}

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PausesByEventSince"), redis_telemetry.ScopePauses)

	pauses := s.unsharded.Pauses()

	// Load all items in the set.
	cmd := pauses.Client().B().
		Zrangebyscore().
		Key(pauses.kg.PauseIndex(ctx, "add", workspaceID, event)).
		Min(strconv.Itoa(int(since.Unix()))).
		Max("+inf").
		Build()
	ids, err := pauses.Client().Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return nil, err
	}

	iter := &keyIter{
		r:     pauses.Client(),
		kf:    pauses.kg,
		start: start,
	}
	err = iter.init(ctx, ids, []float64{}, 100)
	return iter, err
}

// PausesByEventSinceWithCreatedAt is for getting ordered pauses and ensuring that they are returned
// with their createdAt time even when the queue item doesn't have it.
func (s *PauseStore) PausesByEventSinceWithCreatedAt(ctx context.Context, workspaceID uuid.UUID, event string, since time.Time, limit int64) (state.PauseIterator, error) {
	start := time.Now()

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, "PausesByEventSinceWithCreatedAt"), redis_telemetry.ScopePauses)

	pauses := s.unsharded.Pauses()

	cmd := pauses.Client().B().
		Zrange().
		Key(pauses.kg.PauseIndex(ctx, "add", workspaceID, event)).
		Min(strconv.Itoa(int(since.Unix()))).
		Max("+inf").
		Byscore().
		Limit(0, limit).
		Withscores().
		Build()

	results, err := pauses.Client().Do(ctx, cmd).AsZScores()
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(results))
	scores := make([]float64, len(results))
	for i, result := range results {
		ids[i] = result.Member
		scores[i] = result.Score
	}

	iter := &keyIter{
		r:     pauses.Client(),
		kf:    pauses.kg,
		start: start,
	}
	err = iter.init(ctx, ids, scores, 100)
	return iter, err
}

func (s *PauseStore) LoadEvaluablesSince(ctx context.Context, workspaceID uuid.UUID, eventName string, since time.Time, do func(context.Context, expr.Evaluable) error) error {
	// Keep a list of pauses that should be deleted because they've expired.
	//
	// Note that we don't do this in the iteration loop, as redis can use either HSCAN or
	// MGET;  deleting during iteration may lead to skipped items.
	expired := []*state.Pause{}

	it, err := s.PausesByEventSince(ctx, workspaceID, eventName, since)
	if err != nil {
		return err
	}
	for it.Next(ctx) {
		pause := it.Val(ctx)
		if pause == nil {
			continue
		}

		if pause.Expires.Time().Before(time.Now()) {
			// runTS is the time that the run started.
			runTS := time.UnixMilli(int64(pause.Identifier.RunID.Time()))

			// isMaxAge returns whether the pause is greater than the max age allowed
			isMaxAge := time.Now().Add(-1 * consts.CancelTimeout).After(runTS)

			afterGrace := pause.Expires.Time().Add(consts.PauseExpiredDeletionGracePeriod).Before(time.Now())

			if isMaxAge || afterGrace {
				expired = append(expired, pause)
			}

			continue
		}

		if err := do(ctx, pause); err != nil {
			return err
		}
	}

	// GC pauses on fetch.
	for _, pause := range expired {
		logger.StdlibLogger(ctx).Debug("deleting expired pause in iterator", "pause", pause)
		_ = s.DeletePause(ctx, *pause)
	}

	if it.Error() != context.Canceled && it.Error() != errScanDone {
		return it.Error()
	}

	return nil
}
