package rueidis

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rueian/rueidis/internal/cmds"
	"github.com/rueian/rueidis/internal/util"
)

// MGetCache is a helper that consults the client-side caches with multiple keys by grouping keys within same slot into MGETs
func MGetCache(client Client, ctx context.Context, ttl time.Duration, keys []string) (ret map[string]RedisMessage, err error) {
	if len(keys) == 0 {
		return make(map[string]RedisMessage), nil
	}
	if _, ok := client.(*singleClient); ok {
		return clientMGetCache(client, ctx, ttl, client.B().Mget().Key(keys...).Cache(), keys)
	}
	return parallelMGetCache(client, ctx, ttl, cmds.MGets(keys), keys)
}

// MGet is a helper that consults the redis directly with multiple keys by grouping keys within same slot into MGETs
func MGet(client Client, ctx context.Context, keys []string) (ret map[string]RedisMessage, err error) {
	if len(keys) == 0 {
		return make(map[string]RedisMessage), nil
	}
	if _, ok := client.(*singleClient); ok {
		return clientMGet(client, ctx, client.B().Mget().Key(keys...).Build(), keys)
	}
	return parallelMGet(client, ctx, cmds.MGets(keys), keys)
}

// MSet is a helper that consults the redis directly with multiple keys by grouping keys within same slot into MSETs
func MSet(client Client, ctx context.Context, kvs map[string]string) map[string]error {
	if len(kvs) == 0 {
		return make(map[string]error)
	}
	if _, ok := client.(*singleClient); ok {
		return clientMSet(client, ctx, "MSET", kvs, make(map[string]error, len(kvs)))
	}
	return parallelMSet(client, ctx, cmds.MSets(kvs), make(map[string]error, len(kvs)))
}

// MSetNX is a helper that consults the redis directly with multiple keys by grouping keys within same slot into MSETNXs
func MSetNX(client Client, ctx context.Context, kvs map[string]string) map[string]error {
	if len(kvs) == 0 {
		return make(map[string]error)
	}
	if _, ok := client.(*singleClient); ok {
		return clientMSet(client, ctx, "MSETNX", kvs, make(map[string]error, len(kvs)))
	}
	return parallelMSet(client, ctx, cmds.MSetNXs(kvs), make(map[string]error, len(kvs)))
}

// JsonMGetCache is a helper that consults the client-side caches with multiple keys by grouping keys within same slot into JSON.MGETs
func JsonMGetCache(client Client, ctx context.Context, ttl time.Duration, keys []string, path string) (ret map[string]RedisMessage, err error) {
	if len(keys) == 0 {
		return make(map[string]RedisMessage), nil
	}
	if _, ok := client.(*singleClient); ok {
		return clientMGetCache(client, ctx, ttl, client.B().JsonMget().Key(keys...).Path(path).Cache(), keys)
	}
	return parallelMGetCache(client, ctx, ttl, cmds.JsonMGets(keys, path), keys)
}

// JsonMGet is a helper that consults redis directly with multiple keys by grouping keys within same slot into JSON.MGETs
func JsonMGet(client Client, ctx context.Context, keys []string, path string) (ret map[string]RedisMessage, err error) {
	if len(keys) == 0 {
		return make(map[string]RedisMessage), nil
	}
	if _, ok := client.(*singleClient); ok {
		return clientMGet(client, ctx, client.B().JsonMget().Key(keys...).Path(path).Build(), keys)
	}
	return parallelMGet(client, ctx, cmds.JsonMGets(keys, path), keys)
}

func clientMGetCache(client Client, ctx context.Context, ttl time.Duration, cmd Cacheable, keys []string) (ret map[string]RedisMessage, err error) {
	arr, err := client.DoCache(ctx, cmd, ttl).ToArray()
	if err != nil {
		return nil, err
	}
	return arrayToKV(make(map[string]RedisMessage, len(keys)), arr, keys), nil
}

func clientMGet(client Client, ctx context.Context, cmd Completed, keys []string) (ret map[string]RedisMessage, err error) {
	arr, err := client.Do(ctx, cmd).ToArray()
	if err != nil {
		return nil, err
	}
	return arrayToKV(make(map[string]RedisMessage, len(keys)), arr, keys), nil
}

func clientMSet(client Client, ctx context.Context, mset string, kvs map[string]string, ret map[string]error) map[string]error {
	cmd := client.B().Arbitrary(mset)
	for k, v := range kvs {
		cmd = cmd.Args(k, v)
	}
	ok, err := client.Do(ctx, cmd.Build()).AsBool()
	if err == nil && !ok {
		err = ErrMSetNXNotSet
	}
	for k := range kvs {
		ret[k] = err
	}
	return ret
}

func parallelMGetCache(cc Client, ctx context.Context, ttl time.Duration, mgets map[uint16]Completed, keys []string) (ret map[string]RedisMessage, err error) {
	return doMGets(make(map[string]RedisMessage, len(keys)), mgets, func(cmd Completed) RedisResult {
		return cc.DoCache(ctx, Cacheable(cmd), ttl)
	})
}

func parallelMGet(cc Client, ctx context.Context, mgets map[uint16]Completed, keys []string) (ret map[string]RedisMessage, err error) {
	return doMGets(make(map[string]RedisMessage, len(keys)), mgets, func(cmd Completed) RedisResult {
		return cc.Do(ctx, cmd)
	})
}

func parallelMSet(cc Client, ctx context.Context, msets map[uint16]Completed, ret map[string]error) map[string]error {
	var mu sync.Mutex
	for _, cmd := range msets {
		cmd.Pin()
	}
	util.ParallelVals(msets, func(cmd Completed) {
		ok, err := cc.Do(ctx, cmd).AsBool()
		err2 := err
		if err2 == nil && !ok {
			err2 = ErrMSetNXNotSet
		}
		mu.Lock()
		for i := 1; i < len(cmd.Commands()); i += 2 {
			ret[cmd.Commands()[i]] = err2
		}
		mu.Unlock()
		if err == nil {
			cmds.Put(cmds.CompletedCS(cmd))
		}
	})
	return ret
}

func doMGets(m map[string]RedisMessage, mgets map[uint16]Completed, fn func(cmd Completed) RedisResult) (ret map[string]RedisMessage, err error) {
	var mu sync.Mutex
	for _, cmd := range mgets {
		cmd.Pin()
	}
	util.ParallelVals(mgets, func(cmd Completed) {
		arr, err2 := fn(cmd).ToArray()
		mu.Lock()
		if err2 != nil {
			err = err2
		} else {
			arrayToKV(m, arr, cmd.Commands()[1:])
		}
		mu.Unlock()
	})
	if err != nil {
		return nil, err
	}
	for _, cmd := range mgets {
		cmds.Put(cmds.CompletedCS(cmd))
	}
	return m, nil
}

func arrayToKV(m map[string]RedisMessage, arr []RedisMessage, keys []string) map[string]RedisMessage {
	for i, resp := range arr {
		m[keys[i]] = resp
	}
	return m
}

// ErrMSetNXNotSet is used in the MSetNX helper when the underlying MSETNX response is 0.
// Ref: https://redis.io/commands/msetnx/
var ErrMSetNXNotSet = errors.New("MSETNX: no key was set")
