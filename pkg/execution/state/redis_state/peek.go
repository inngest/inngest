package redis_state

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
	"unsafe"

	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/inngest/inngest/pkg/util"
	"github.com/redis/rueidis"
)

type peeker[T any] struct {
	q      *queue
	max    int64
	opName string

	// if ignoreUntil is provided, the entire count is returned and items are peeked even
	// if the score exceeds the until value (usually the current time).
	ignoreUntil bool

	isMillisecondPrecision bool

	handleMissingItems func(pointers []string) error
	maker              func() *T
	keyMetadataHash    string

	// fromTime provides an optional start time for peeks
	// instead of the default -INF
	fromTime *time.Time
}

var ErrPeekerPeekExceedsMaxLimits = fmt.Errorf("provided limit exceeds max configured limit")

// peek peeks up to <limit> items from the given ZSET up to until, in order if sequential is true, otherwise randomly.
func (p *peeker[T]) peek(ctx context.Context, keyOrderedPointerSet string, sequential bool, until time.Time, limit int64, opts ...osqueue.PeekOpt) (*osqueue.PeekResult[T], error) {
	l := logger.StdlibLogger(ctx)

	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, p.opName), redis_telemetry.ScopeQueue)

	opt := osqueue.PeekOption{}
	for _, apply := range opts {
		apply(&opt)
	}

	if p.maker == nil {
		return nil, fmt.Errorf("missing 'maker' argument")
	}

	rc := p.q.RedisClient.Client()

	if limit > p.max {
		return nil, ErrPeekerPeekExceedsMaxLimits
	}
	if limit <= 0 {
		limit = p.max
	}

	var script string
	var rawArgs []any
	if p.ignoreUntil {
		script = "peekOrderedSet"
		rawArgs = []any{
			limit,
		}
	} else {
		script = "peekOrderedSetUntil"
		ms := until.UnixMilli()

		fromTime := "-inf"
		if p.fromTime != nil && !p.fromTime.IsZero() {
			fromTime = strconv.Itoa(int(p.fromTime.UnixMilli()))
		}

		untilTime := until.Unix()
		if p.isMillisecondPrecision {
			untilTime = until.UnixMilli()
		}

		isSequential := 0
		if sequential {
			isSequential = 1
		}

		rawArgs = []any{
			fromTime,
			untilTime,
			ms,
			limit,
			isSequential,
		}
	}

	args, err := StrSlice(rawArgs)
	if err != nil {
		return nil, fmt.Errorf("could not convert args: %w", err)
	}

	peekRet, err := scripts[fmt.Sprintf("queue/%s", script)].Exec(
		redis_telemetry.WithScriptName(ctx, script),
		rc,
		[]string{
			p.keyMetadataHash,
			keyOrderedPointerSet,
		},
		args,
	).ToAny()
	// NOTE: We use ToAny to force return a []any, allowing us to update the slice value with
	// a JSON-decoded item without allocations
	if err != nil {
		return nil, fmt.Errorf("error peeking ordered pointer set: %w", err)
	}
	returnedSet, ok := peekRet.([]any)
	if !ok {
		return nil, fmt.Errorf("unknown return type from %s: %T", p.opName, peekRet)
	}

	var totalCount, cursor int64
	var potentiallyMissingItems, allPointerIDs []any
	if len(returnedSet) == 4 {
		totalCount, ok = returnedSet[0].(int64)
		if !ok {
			return nil, fmt.Errorf("unexpected first item in set returned from %s: %T", p.opName, returnedSet[0])
		}

		potentiallyMissingItems, ok = returnedSet[1].([]any)
		if !ok {
			return nil, fmt.Errorf("unexpected second item in set returned from %s: %T", p.opName, returnedSet[1])
		}

		allPointerIDs, ok = returnedSet[2].([]any)
		if !ok {
			return nil, fmt.Errorf("unexpected third item in set returned from %s: %T", p.opName, returnedSet[2])
		}

		cursor, ok = returnedSet[3].(int64)
		if !ok {
			return nil, fmt.Errorf("invalid cursor returned from %s: %T", p.opName, returnedSet[3])
		}
	} else if len(returnedSet) != 0 {
		return nil, fmt.Errorf("expected zero or four items in set returned by %s: %v", p.opName, returnedSet)
	}

	encoded := make([]any, 0)
	missingItems := make([]string, 0)
	if len(potentiallyMissingItems) > 0 {
		for idx, pointerID := range allPointerIDs {
			if potentiallyMissingItems[idx] == nil {
				if pointerID == nil {
					return nil, fmt.Errorf("encountered nil pointer in pointer queue")
				}

				str, ok := pointerID.(string)
				if !ok {
					return nil, fmt.Errorf("encountered non-string pointer in pointer queue")
				}

				missingItems = append(missingItems, str)
			} else {
				encoded = append(encoded, potentiallyMissingItems[idx])
			}
		}
	}

	// Use parallel decoding as per Peek
	items, err := util.ParallelDecode(encoded, func(val any, _ int) (*T, bool, error) {
		if val == nil {
			l.Error("encountered nil item in pointer queue",
				"encoded", encoded,
				"missing", missingItems,
				"key", keyOrderedPointerSet,
			)
			return nil, false, fmt.Errorf("encountered nil item in pointer queue")
		}

		str, ok := val.(string)
		if !ok {
			return nil, false, fmt.Errorf("unknown type in peekOrderedPointerSet: %T", val)
		}

		item := p.maker()

		if err := json.Unmarshal(unsafe.Slice(unsafe.StringData(str), len(str)), item); err != nil {
			return nil, false, fmt.Errorf("error reading item: %w", err)
		}

		return item, false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error decoding items: %w", err)
	}

	if !opt.IgnoreCleanup && p.handleMissingItems != nil && len(missingItems) > 0 {
		if err := p.handleMissingItems(missingItems); err != nil {
			return nil, fmt.Errorf("could not handle missing items: %w", err)
		}
	}

	return &osqueue.PeekResult[T]{
		Items:        items,
		TotalCount:   int(totalCount),
		RemovedCount: len(missingItems),
		Cursor:       cursor,
	}, nil
}

func (p *peeker[T]) peekPointer(ctx context.Context, keyOrderedPointerSet string, sequential bool, until time.Time, limit int64) ([]string, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, p.opName), redis_telemetry.ScopeQueue)

	if limit > p.max {
		return nil, ErrPeekerPeekExceedsMaxLimits
	}
	if limit <= 0 {
		limit = p.max
	}

	ms := until.UnixMilli()

	untilTime := until.Unix()
	if p.isMillisecondPrecision {
		untilTime = until.UnixMilli()
	}

	isSequential := 0
	if sequential {
		isSequential = 1
	}

	args, err := StrSlice([]any{
		untilTime,
		ms,
		limit,
		isSequential,
	})
	if err != nil {
		return nil, err
	}

	pointers, err := scripts["queue/peekPointerUntil"].Exec(
		redis_telemetry.WithScriptName(ctx, "peekPointerUntil"),
		p.q.RedisClient.unshardedRc,
		[]string{
			keyOrderedPointerSet,
		},
		args,
	).AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("error peeking pointers in %s: %w", p.opName, err)
	}

	return pointers, nil
}

func (p *peeker[T]) peekUUIDPointer(ctx context.Context, keyOrderedPointerSet string, sequential bool, until time.Time, limit int64) ([]uuid.UUID, error) {
	pointers, err := p.peekPointer(ctx, keyOrderedPointerSet, sequential, until, limit)
	if err != nil {
		return nil, fmt.Errorf("could not peek pointers: %w", err)
	}

	items := make([]uuid.UUID, len(pointers))
	for i, s := range pointers {
		parsed, err := uuid.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("could not parse uuid from ordered queue: %w", err)
		}

		items[i] = parsed
	}

	return items, nil
}

func CleanupMissingPointers(ctx context.Context, key string, client rueidis.Client, log logger.Logger) func(pointers []string) error {
	return func(pointers []string) error {
		cmd := client.B().Zrem().Key(key).Member(pointers...).Build()

		err := client.Do(ctx, cmd).Error()
		if err != nil {
			log.Warn("could not clean up missing items", "err", err, "missing", pointers, "source", key)
		}

		return nil
	}
}
