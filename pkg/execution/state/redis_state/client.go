package redis_state

import (
	"context"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
)

const StateDefaultKey = "estate"
const QueueDefaultKey = "queue"

type FunctionRunStateClient struct {
	kg            RunStateKeyGenerator
	client        RetriableClient
	unshardedConn RetriableClient
	isSharded     IsShardedWithRunIdFn
}

func (f *FunctionRunStateClient) KeyGenerator() RunStateKeyGenerator {
	return f.kg
}

func (f *FunctionRunStateClient) Client(ctx context.Context, accountId uuid.UUID, runId ulid.ULID) (RetriableClient, bool) {
	if f.isSharded(ctx, accountId, runId) {
		return f.client, true
	}
	return f.unshardedConn, false
}

func (f *FunctionRunStateClient) ForceShardedClient() RetriableClient {
	return f.client
}

func NewFunctionRunStateClient(r rueidis.Client, u *UnshardedClient, stateDefaultKey string, isSharded IsShardedWithRunIdFn) *FunctionRunStateClient {
	if r == nil {
		panic("missing function run state client")
	}
	return &FunctionRunStateClient{
		kg:            &runStateKeyGenerator{stateDefaultKey: stateDefaultKey},
		client:        newRetryClusterDownClient(r),
		unshardedConn: NewNoopRetriableClient(u.unshardedConn),
		isSharded:     isSharded,
	}
}

type BatchClient struct {
	kg     BatchKeyGenerator
	client RetriableClient
}

func (b *BatchClient) KeyGenerator() BatchKeyGenerator {
	return b.kg
}

func (b *BatchClient) Client() RetriableClient {
	return b.client
}

func NewBatchClient(r rueidis.Client, queueDefaultKey string) *BatchClient {
	if r == nil {
		panic("missing batch redis client")
	}
	return &BatchClient{
		kg:     batchKeyGenerator{queueDefaultKey: queueDefaultKey, queueItemKeyGenerator: queueItemKeyGenerator{queueDefaultKey: queueDefaultKey}},
		client: newRetryClusterDownClient(r),
	}
}

type ShardedClient struct {
	fnRunState *FunctionRunStateClient
	batch      *BatchClient
}

type IsShardedWithRunIdFn func(ctx context.Context, accountId uuid.UUID, runId ulid.ULID) bool

func AlwaysShardOnRun(ctx context.Context, accountId uuid.UUID, runId ulid.ULID) bool {
	return true
}

func NeverShardOnRun(ctx context.Context, accountId uuid.UUID, runId ulid.ULID) bool {
	return false
}

type ShardedClientOpts struct {
	UnshardedClient *UnshardedClient

	FunctionRunStateClient rueidis.Client
	BatchClient            rueidis.Client

	StateDefaultKey string
	QueueDefaultKey string

	FnRunIsSharded IsShardedWithRunIdFn
}

func NewShardedClient(opts ShardedClientOpts) *ShardedClient {
	return &ShardedClient{
		fnRunState: NewFunctionRunStateClient(opts.FunctionRunStateClient, opts.UnshardedClient, opts.StateDefaultKey, opts.FnRunIsSharded),
		batch:      NewBatchClient(opts.BatchClient, opts.QueueDefaultKey),
	}
}

func (s *ShardedClient) FunctionRunState() *FunctionRunStateClient {
	return s.fnRunState
}

func (s *ShardedClient) Batch() *BatchClient {
	return s.batch
}

type PauseClient struct {
	kg          PauseKeyGenerator
	unshardedRc rueidis.Client
}

func (p *PauseClient) KeyGenerator() PauseKeyGenerator {
	return p.kg
}

func (p *PauseClient) Client() rueidis.Client {
	return p.unshardedRc
}

func NewPauseClient(r rueidis.Client, stateDefaultKey string) *PauseClient {
	return &PauseClient{
		kg:          pauseKeyGenerator{stateDefaultKey: stateDefaultKey},
		unshardedRc: r,
	}
}

type QueueClient struct {
	kg          QueueKeyGenerator
	unshardedRc rueidis.Client
}

func (q *QueueClient) KeyGenerator() QueueKeyGenerator {
	return q.kg
}

func (q *QueueClient) Client() rueidis.Client {
	return q.unshardedRc
}

func NewQueueClient(r rueidis.Client, queueDefaultKey string) *QueueClient {
	return &QueueClient{
		kg:          queueKeyGenerator{queueDefaultKey: queueDefaultKey, queueItemKeyGenerator: queueItemKeyGenerator{queueDefaultKey: queueDefaultKey}},
		unshardedRc: r,
	}
}

type DebounceClient struct {
	kg          DebounceKeyGenerator
	unshardedRc rueidis.Client
}

func (d *DebounceClient) KeyGenerator() DebounceKeyGenerator {
	return d.kg
}

func (d *DebounceClient) Client() rueidis.Client {
	return d.unshardedRc
}

func NewDebounceClient(r rueidis.Client, queueDefaultKey string) *DebounceClient {
	return &DebounceClient{
		kg:          debounceKeyGenerator{queueDefaultKey: queueDefaultKey, queueItemKeyGenerator: queueItemKeyGenerator{queueDefaultKey: queueDefaultKey}},
		unshardedRc: r,
	}
}

type GlobalClient struct {
	kg          GlobalKeyGenerator
	unshardedRc rueidis.Client
}

func (g *GlobalClient) KeyGenerator() GlobalKeyGenerator {
	return g.kg
}

func (g *GlobalClient) Client() rueidis.Client {
	return g.unshardedRc
}

func NewGlobalClient(r rueidis.Client, stateDefaultKey string) *GlobalClient {
	return &GlobalClient{
		kg:          globalKeyGenerator{stateDefaultKey: stateDefaultKey},
		unshardedRc: r,
	}
}

type UnshardedClient struct {
	unshardedConn rueidis.Client

	pauses   *PauseClient
	queue    *QueueClient
	debounce *DebounceClient
	global   *GlobalClient
}

func (u *UnshardedClient) Pauses() *PauseClient {
	return u.pauses
}

func (u *UnshardedClient) Queue() *QueueClient {
	return u.queue
}

func (u *UnshardedClient) Debounce() *DebounceClient {
	return u.debounce
}

func (u *UnshardedClient) Global() *GlobalClient {
	return u.global
}

func NewUnshardedClient(r rueidis.Client, stateDefaultKey, queueDefaultKey string) *UnshardedClient {
	return &UnshardedClient{
		pauses:        NewPauseClient(r, stateDefaultKey),
		queue:         NewQueueClient(r, queueDefaultKey),
		debounce:      NewDebounceClient(r, queueDefaultKey),
		global:        NewGlobalClient(r, stateDefaultKey),
		unshardedConn: r,
	}
}
