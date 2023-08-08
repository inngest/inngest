package rueidis

import (
	"context"
	"errors"
	"time"

	intl "github.com/redis/rueidis/internal/cmds"
)

// MGetCache is a helper that consults the client-side caches with multiple keys by grouping keys within same slot into multiple GETs
func MGetCache(client Client, ctx context.Context, ttl time.Duration, keys []string) (ret map[string]RedisMessage, err error) {
	if len(keys) == 0 {
		return make(map[string]RedisMessage), nil
	}
	cmds := make([]CacheableTTL, len(keys))
	for i := range cmds {
		cmds[i] = CT(client.B().Get().Key(keys[i]).Cache(), ttl)
	}
	return doMultiCache(client, ctx, cmds, keys)
}

// MGet is a helper that consults the redis directly with multiple keys by grouping keys within same slot into MGET or multiple GETs
func MGet(client Client, ctx context.Context, keys []string) (ret map[string]RedisMessage, err error) {
	if len(keys) == 0 {
		return make(map[string]RedisMessage), nil
	}
	switch client.(type) {
	case *singleClient, *sentinelClient:
		return clientMGet(client, ctx, client.B().Mget().Key(keys...).Build(), keys)
	}

	cmds := make([]Completed, len(keys))
	for i := range cmds {
		cmds[i] = client.B().Get().Key(keys[i]).Build()
	}
	return doMultiGet(client, ctx, cmds, keys)
}

// MSet is a helper that consults the redis directly with multiple keys by grouping keys within same slot into MSETs or multiple SETs
func MSet(client Client, ctx context.Context, kvs map[string]string) map[string]error {
	if len(kvs) == 0 {
		return make(map[string]error)
	}

	switch client.(type) {
	case *singleClient, *sentinelClient:
		return clientMSet(client, ctx, "MSET", kvs, make(map[string]error, len(kvs)))
	}

	cmds := make([]Completed, 0, len(kvs))
	keys := make([]string, 0, len(kvs))
	for k, v := range kvs {
		cmds = append(cmds, client.B().Set().Key(k).Value(v).Build())
		keys = append(keys, k)
	}
	return doMultiSet(client, ctx, cmds, keys)
}

// MDel is a helper that consults the redis directly with multiple keys by grouping keys within same slot into DELs
func MDel(client Client, ctx context.Context, keys []string) map[string]error {
	if len(keys) == 0 {
		return make(map[string]error)
	}

	switch client.(type) {
	case *singleClient, *sentinelClient:
		return clientMDel(client, ctx, keys)
	}

	cmds := make([]Completed, len(keys))
	for i, k := range keys {
		cmds[i] = client.B().Del().Key(k).Build()
	}
	return doMultiSet(client, ctx, cmds, keys)
}

// MSetNX is a helper that consults the redis directly with multiple keys by grouping keys within same slot into MSETNXs or multiple SETNXs
func MSetNX(client Client, ctx context.Context, kvs map[string]string) map[string]error {
	if len(kvs) == 0 {
		return make(map[string]error)
	}

	switch client.(type) {
	case *singleClient, *sentinelClient:
		return clientMSet(client, ctx, "MSETNX", kvs, make(map[string]error, len(kvs)))
	}

	cmds := make([]Completed, 0, len(kvs))
	keys := make([]string, 0, len(kvs))
	for k, v := range kvs {
		cmds = append(cmds, client.B().Set().Key(k).Value(v).Nx().Build())
		keys = append(keys, k)
	}
	return doMultiSet(client, ctx, cmds, keys)
}

// JsonMGetCache is a helper that consults the client-side caches with multiple keys by grouping keys within same slot into multiple JSON.GETs
func JsonMGetCache(client Client, ctx context.Context, ttl time.Duration, keys []string, path string) (ret map[string]RedisMessage, err error) {
	if len(keys) == 0 {
		return make(map[string]RedisMessage), nil
	}
	cmds := make([]CacheableTTL, len(keys))
	for i := range cmds {
		cmds[i] = CT(client.B().JsonGet().Key(keys[i]).Path(path).Cache(), ttl)
	}
	return doMultiCache(client, ctx, cmds, keys)
}

// JsonMGet is a helper that consults redis directly with multiple keys by grouping keys within same slot into JSON.MGETs or multiple JSON.GETs
func JsonMGet(client Client, ctx context.Context, keys []string, path string) (ret map[string]RedisMessage, err error) {
	if len(keys) == 0 {
		return make(map[string]RedisMessage), nil
	}

	switch client.(type) {
	case *singleClient, *sentinelClient:
		return clientMGet(client, ctx, client.B().JsonMget().Key(keys...).Path(path).Build(), keys)
	}

	cmds := make([]Completed, len(keys))
	for i := range cmds {
		cmds[i] = client.B().JsonGet().Key(keys[i]).Path(path).Build()
	}
	return doMultiGet(client, ctx, cmds, keys)
}

// JsonMSet is a helper that consults redis directly with multiple keys by grouping keys within same slot into JSON.MSETs or multiple JOSN.SETs
func JsonMSet(client Client, ctx context.Context, kvs map[string]string, path string) map[string]error {
	if len(kvs) == 0 {
		return make(map[string]error)
	}

	switch client.(type) {
	case *singleClient, *sentinelClient:
		return clientJSONMSet(client, ctx, kvs, path, make(map[string]error, len(kvs)))
	}

	cmds := make([]Completed, 0, len(kvs))
	keys := make([]string, 0, len(kvs))
	for k, v := range kvs {
		cmds = append(cmds, client.B().JsonSet().Key(k).Path(path).Value(v).Build())
		keys = append(keys, k)
	}
	return doMultiSet(client, ctx, cmds, keys)
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

func clientJSONMSet(client Client, ctx context.Context, kvs map[string]string, path string, ret map[string]error) map[string]error {
	cmd := intl.JsonMsetTripletValue(client.B().JsonMset())
	for k, v := range kvs {
		cmd = cmd.Key(k).Path(path).Value(v)
	}
	err := client.Do(ctx, cmd.Build()).Error()
	for k := range kvs {
		ret[k] = err
	}
	return ret
}

func clientMDel(client Client, ctx context.Context, keys []string) map[string]error {
	err := client.Do(ctx, client.B().Del().Key(keys...).Build()).Error()
	ret := make(map[string]error, len(keys))
	for _, k := range keys {
		ret[k] = err
	}
	return ret
}

func doMultiCache(cc Client, ctx context.Context, cmds []CacheableTTL, keys []string) (ret map[string]RedisMessage, err error) {
	ret = make(map[string]RedisMessage, len(keys))
	resps := cc.DoMultiCache(ctx, cmds...)
	defer resultsp.Put(&redisresults{s: resps})
	for i, resp := range resps {
		if err := resp.NonRedisError(); err != nil {
			return nil, err
		}
		ret[keys[i]] = resp.val
	}
	return ret, nil
}

func doMultiGet(cc Client, ctx context.Context, cmds []Completed, keys []string) (ret map[string]RedisMessage, err error) {
	ret = make(map[string]RedisMessage, len(keys))
	resps := cc.DoMulti(ctx, cmds...)
	defer resultsp.Put(&redisresults{s: resps})
	for i, resp := range resps {
		if err := resp.NonRedisError(); err != nil {
			return nil, err
		}
		ret[keys[i]] = resp.val
	}
	return ret, nil
}

func doMultiSet(cc Client, ctx context.Context, cmds []Completed, keys []string) (ret map[string]error) {
	ret = make(map[string]error, len(keys))
	resps := cc.DoMulti(ctx, cmds...)
	for i, resp := range resps {
		ret[keys[i]] = resp.Error()
	}
	resultsp.Put(&redisresults{s: resps})
	return ret
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
