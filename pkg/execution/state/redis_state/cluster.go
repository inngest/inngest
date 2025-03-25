package redis_state

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"github.com/redis/rueidis"
	"runtime"
	"time"
)

type RetriableClient interface {
	Do(ctx context.Context, cmd func(client rueidis.Client) rueidis.Completed) (resp rueidis.RedisResult)
}

type noopRetriableClient struct {
	r rueidis.Client
}

func (r noopRetriableClient) Do(ctx context.Context, cmd func(client rueidis.Client) rueidis.Completed) (resp rueidis.RedisResult) {
	return r.r.Do(ctx, cmd(r.r))
}

func NewNoopRetriableClient(client rueidis.Client) RetriableClient {
	return noopRetriableClient{client}
}

type retryClusterDownClient struct {
	r rueidis.Client
}

func (r retryClusterDownClient) do(ctx context.Context, cmd func(client rueidis.Client) rueidis.Completed, attempts int) rueidis.RedisResult {
	resp := r.r.Do(ctx, cmd(r.r))

	if err := resp.Error(); err != nil {
		if ret, ok := rueidis.IsRedisErr(err); ok {
			// retry on CLUSTERDOWN (in case we're scaling up/down)
			if ret.IsClusterDown() {
				if attempts == 5 {
					return resp
				}

				time.Sleep(100 * time.Millisecond)
				return r.do(ctx, cmd, attempts+1)
			}
		}
	}

	return resp
}

func (r retryClusterDownClient) Do(ctx context.Context, cmd func(client rueidis.Client) rueidis.Completed) (resp rueidis.RedisResult) {
	return r.do(ctx, cmd, 0)
}

func newRetryClusterDownClient(r rueidis.Client) RetriableClient {
	return &retryClusterDownClient{r: r}
}

// NewClusterLuaScript creates a Lua instance whose Lua.Exec uses EVALSHA and EVAL.
func NewClusterLuaScript(script string) *RetriableLua {
	sum := sha1.Sum([]byte(script))
	return &RetriableLua{script: script, sha1: hex.EncodeToString(sum[:]), maxp: runtime.GOMAXPROCS(0)}
}

// Lua represents a redis lua script. It should be created from the NewLuaScript() or NewLuaScriptReadOnly()
type RetriableLua struct {
	script   string
	sha1     string
	maxp     int
	readonly bool
}

// Exec the script to the given Client.
// It will first try with the EVALSHA/EVALSHA_RO and then EVAL/EVAL_RO if first try failed.
// Cross slot keys are prohibited if the Client is a cluster client.
func (s *RetriableLua) Exec(ctx context.Context, c RetriableClient, keys, args []string) (resp rueidis.RedisResult) {
	if s.readonly {
		resp = c.Do(ctx, func(client rueidis.Client) rueidis.Completed {
			return client.B().EvalshaRo().Sha1(s.sha1).Numkeys(int64(len(keys))).Key(keys...).Arg(args...).Build()
		})
	} else {
		resp = c.Do(ctx, func(client rueidis.Client) rueidis.Completed {
			return client.B().Evalsha().Sha1(s.sha1).Numkeys(int64(len(keys))).Key(keys...).Arg(args...).Build()
		})
	}
	if err, ok := rueidis.IsRedisErr(resp.Error()); ok && err.IsNoScript() {
		if s.readonly {
			resp = c.Do(ctx, func(client rueidis.Client) rueidis.Completed {
				return client.B().EvalRo().Script(s.script).Numkeys(int64(len(keys))).Key(keys...).Arg(args...).Build()
			})
		} else {
			resp = c.Do(ctx, func(client rueidis.Client) rueidis.Completed {
				return client.B().Eval().Script(s.script).Numkeys(int64(len(keys))).Key(keys...).Arg(args...).Build()
			})
		}
	}
	return resp
}
