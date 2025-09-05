package peek

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/redis_telemetry"
	"github.com/inngest/inngest/pkg/util"
	"github.com/redis/rueidis"
)

type Option func(p *peekOption)

type peekOption struct {
	client rueidis.Client

	// fromTime provides an optional start time for peeks
	// instead of the default -INF
	from *time.Time

	// fromTime provides an optional end time for peeks
	// instead of the default +INF
	until *time.Time

	// limit determines how many items should be peeked
	limit int

	// sequential determines whether peeks should respect FIFO rules
	sequential bool
}

func WithClient(c rueidis.Client) Option {
	return func(p *peekOption) {
		p.client = c
	}
}

func From(from time.Time) Option {
	return func(p *peekOption) {
		p.from = &from
	}
}

func Until(until time.Time) Option {
	return func(p *peekOption) {
		p.from = &until
	}
}

func Limit(limit int) Option {
	return func(p *peekOption) {
		p.limit = limit
	}
}

func Sequential(sequential bool) Option {
	return func(p *peekOption) {
		p.sequential = sequential
	}
}

var ErrPeekerPeekExceedsMaxLimits = fmt.Errorf("provided limit exceeds max configured limit")

// Peek peeks up to <limit> items from the given ZSET up to until, in order if sequential is true, otherwise randomly.
func (p *peeker[T]) Peek(ctx context.Context, keyOrderedPointerSet string, opts ...Option) (*Result[T], error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, p.opName), redis_telemetry.ScopeQueue)

	l := logger.StdlibLogger(ctx)

	opt := peekOption{}
	for _, apply := range opts {
		apply(&opt)
	}

	client := p.client
	if opt.client != nil {
		client = opt.client
	}

	if p.maker == nil {
		return nil, fmt.Errorf("missing 'maker' argument")
	}

	limit := int64(opt.limit)
	if limit > p.max {
		return nil, ErrPeekerPeekExceedsMaxLimits
	}
	if limit <= 0 {
		limit = p.max
	}

	var script string
	var rawArgs []any

	until := opt.until
	if until == nil {
		script = "peekOrderedSet"
		rawArgs = []any{
			limit,
		}
	} else {
		script = "peekOrderedSetUntil"
		ms := until.UnixMilli()

		fromTime := "-inf"
		if opt.from != nil && !opt.from.IsZero() {
			fromTime = strconv.Itoa(int(opt.from.UnixMilli()))
		}

		untilTime := until.Unix()
		if p.isMillisecondPrecision {
			untilTime = until.UnixMilli()
		}

		isSequential := 0
		if opt.sequential {
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

	args, err := util.StrSlice(rawArgs)
	if err != nil {
		return nil, fmt.Errorf("could not convert args: %w", err)
	}

	peekRet, err := scripts[script].Exec(
		redis_telemetry.WithScriptName(ctx, script),
		client,
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

	var totalCount int64
	var potentiallyMissingItems, allPointerIDs []any
	if len(returnedSet) == 3 {
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

	} else if len(returnedSet) != 0 {
		return nil, fmt.Errorf("expected zero or three items in set returned by %s: %v", p.opName, returnedSet)
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

	if p.handleMissingItems != nil && len(missingItems) > 0 {
		if err := p.handleMissingItems(ctx, missingItems); err != nil {
			return nil, fmt.Errorf("could not handle missing items: %w", err)
		}
	}

	return &Result[T]{
		Items:        items,
		TotalCount:   int(totalCount),
		RemovedCount: len(missingItems),
	}, nil
}

func (p *peeker[T]) PeekPointer(ctx context.Context, keyOrderedPointerSet string, opts ...Option) ([]string, error) {
	ctx = redis_telemetry.WithScope(redis_telemetry.WithOpName(ctx, p.opName), redis_telemetry.ScopeQueue)

	opt := peekOption{}
	for _, apply := range opts {
		apply(&opt)
	}

	client := p.client
	if opt.client != nil {
		client = opt.client
	}

	limit := int64(opt.limit)
	if limit > p.max {
		return nil, ErrPeekerPeekExceedsMaxLimits
	}
	if limit <= 0 {
		limit = p.max
	}

	until := opt.until
	if until == nil {
		return nil, fmt.Errorf("until is required in peek pointer")
	}

	ms := until.UnixMilli()

	untilTime := until.Unix()
	if p.isMillisecondPrecision {
		untilTime = until.UnixMilli()
	}

	isSequential := 0
	if opt.sequential {
		isSequential = 1
	}

	args, err := util.StrSlice([]any{
		untilTime,
		ms,
		limit,
		isSequential,
	})
	if err != nil {
		return nil, err
	}

	pointers, err := scripts["peekPointerUntil"].Exec(
		redis_telemetry.WithScriptName(ctx, "peekPointerUntil"),
		client,
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

func (p *peeker[T]) PeekUUIDPointer(ctx context.Context, keyOrderedPointerSet string, opts ...Option) ([]uuid.UUID, error) {
	pointers, err := p.PeekPointer(ctx, keyOrderedPointerSet, opts...)
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

func CleanupMissingPointers(key string, client rueidis.Client, log logger.Logger) MissingItemHandler {
	return func(ctx context.Context, pointers []string) error {
		cmd := client.B().Zrem().Key(key).Member(pointers...).Build()

		err := client.Do(ctx, cmd).Error()
		if err != nil {
			log.Warn("could not clean up missing items", "err", err, "missing", pointers, "source", key)
		}

		return nil
	}
}

func init() {
	// read the lua scripts
	entries, err := embedded.ReadDir("lua")
	if err != nil {
		panic(fmt.Errorf("error reading redis lua dir: %w", err))
	}

	readRedisScripts("lua", entries)
}
