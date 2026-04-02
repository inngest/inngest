package rueidis

import (
	"context"
	"errors"
	"iter"
	"sync/atomic"
	"time"

	intl "github.com/redis/rueidis/internal/cmds"
)

// MGetCache is a helper that consults the client-side caches with multiple keys by grouping keys within the same slot into multiple GETs
func MGetCache(client Client, ctx context.Context, ttl time.Duration, keys []string) (ret map[string]RedisMessage, err error) {
	if len(keys) == 0 {
		return make(map[string]RedisMessage), nil
	}
	if isCacheDisabled(client) {
		return MGet(client, ctx, keys)
	}
	cmds := mgetcachecmdsp.Get(len(keys), len(keys))
	defer mgetcachecmdsp.Put(cmds)
	for i := range cmds.s {
		cmds.s[i] = CT(client.B().Get().Key(keys[i]).Cache(), ttl)
	}
	return doMultiCache(client, ctx, cmds.s, keys)
}

func isCacheDisabled(client Client) bool {
	switch c := client.(type) {
	case *singleClient:
		return c.DisableCache
	case *standalone:
		return c.primary.Load().DisableCache
	case *sentinelClient:
		return c.mOpt != nil && c.mOpt.DisableCache
	case *clusterClient:
		return c.opt != nil && c.opt.DisableCache
	}
	return false
}

// MGet is a helper that consults the redis directly with multiple keys by grouping keys within the same slot into MGET or multiple GETs
func MGet(client Client, ctx context.Context, keys []string) (ret map[string]RedisMessage, err error) {
	if len(keys) == 0 {
		return make(map[string]RedisMessage), nil
	}

	switch client.(type) {
	case *singleClient, *standalone, *sentinelClient:
		return clientMGet(client, ctx, client.B().Mget().Key(keys...).Build(), keys)
	}

	return clusterMGet(client, ctx, keys)
}

// MSet is a helper that consults the redis directly with multiple keys by grouping keys within the same slot into MSETs or multiple SETs
func MSet(client Client, ctx context.Context, kvs map[string]string) map[string]error {
	if len(kvs) == 0 {
		return make(map[string]error)
	}

	switch client.(type) {
	case *singleClient, *standalone, *sentinelClient:
		return clientMSet(client, ctx, "MSET", kvs, make(map[string]error, len(kvs)))
	}

	cmds := mgetcmdsp.Get(0, len(kvs))
	defer mgetcmdsp.Put(cmds)
	for k, v := range kvs {
		cmds.s = append(cmds.s, client.B().Set().Key(k).Value(v).Build().Pin())
	}
	return doMultiSet(client, ctx, cmds.s)
}

// MDel is a helper that consults the redis directly with multiple keys by grouping keys within the same slot into DELs
func MDel(client Client, ctx context.Context, keys []string) map[string]error {
	if len(keys) == 0 {
		return make(map[string]error)
	}

	switch client.(type) {
	case *singleClient, *standalone, *sentinelClient:
		return clientMDel(client, ctx, keys)
	}

	cmds := mgetcmdsp.Get(len(keys), len(keys))
	defer mgetcmdsp.Put(cmds)
	for i, k := range keys {
		cmds.s[i] = client.B().Del().Key(k).Build().Pin()
	}
	return doMultiSet(client, ctx, cmds.s)
}

// MSetNX is a helper that consults the redis directly with multiple keys by grouping keys within the same slot into MSETNXs or multiple SETNXs
func MSetNX(client Client, ctx context.Context, kvs map[string]string) map[string]error {
	if len(kvs) == 0 {
		return make(map[string]error)
	}

	switch client.(type) {
	case *singleClient, *standalone, *sentinelClient:
		return clientMSet(client, ctx, "MSETNX", kvs, make(map[string]error, len(kvs)))
	}

	cmds := mgetcmdsp.Get(0, len(kvs))
	defer mgetcmdsp.Put(cmds)
	for k, v := range kvs {
		cmds.s = append(cmds.s, client.B().Set().Key(k).Value(v).Nx().Build().Pin())
	}
	return doMultiSet(client, ctx, cmds.s)
}

// JsonMGetCache is a helper that consults the client-side caches with multiple keys by grouping keys within the same slot into multiple JSON.GETs
func JsonMGetCache(client Client, ctx context.Context, ttl time.Duration, keys []string, path string) (ret map[string]RedisMessage, err error) {
	if len(keys) == 0 {
		return make(map[string]RedisMessage), nil
	}
	cmds := mgetcachecmdsp.Get(len(keys), len(keys))
	defer mgetcachecmdsp.Put(cmds)
	for i := range cmds.s {
		cmds.s[i] = CT(client.B().JsonGet().Key(keys[i]).Path(path).Cache(), ttl)
	}
	return doMultiCache(client, ctx, cmds.s, keys)
}

// JsonMGet is a helper that consults redis directly with multiple keys by grouping keys within the same slot into JSON.MGETs or multiple JSON.GETs
func JsonMGet(client Client, ctx context.Context, keys []string, path string) (ret map[string]RedisMessage, err error) {
	if len(keys) == 0 {
		return make(map[string]RedisMessage), nil
	}

	switch client.(type) {
	case *singleClient, *standalone, *sentinelClient:
		return clientMGet(client, ctx, client.B().JsonMget().Key(keys...).Path(path).Build(), keys)
	}

	return clusterJsonMGet(client, ctx, keys, path)
}

// JsonMSet is a helper that consults redis directly with multiple keys by grouping keys within the same slot into JSON.MSETs or multiple JSON.SETs
func JsonMSet(client Client, ctx context.Context, kvs map[string]string, path string) map[string]error {
	if len(kvs) == 0 {
		return make(map[string]error)
	}

	switch client.(type) {
	case *singleClient, *standalone, *sentinelClient:
		return clientJSONMSet(client, ctx, kvs, path, make(map[string]error, len(kvs)))
	}

	cmds := mgetcmdsp.Get(0, len(kvs))
	defer mgetcmdsp.Put(cmds)
	for k, v := range kvs {
		cmds.s = append(cmds.s, client.B().JsonSet().Key(k).Path(path).Value(v).Build().Pin())
	}
	return doMultiSet(client, ctx, cmds.s)
}

// DecodeSliceOfJSON is a helper that struct-scans each RedisMessage into dest, which must be a slice of the pointer.
func DecodeSliceOfJSON[T any](result RedisResult, dest *[]T) error {
	values, err := result.ToArray()
	if err != nil {
		return err
	}

	ts := make([]T, len(values))
	for i, v := range values {
		var t T
		if err = v.DecodeJSON(&t); err != nil {
			if IsRedisNil(err) {
				continue
			}
			return err
		}
		ts[i] = t
	}
	*dest = ts
	return nil
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

func doMultiSet(cc Client, ctx context.Context, cmds []Completed) (ret map[string]error) {
	ret = make(map[string]error, len(cmds))
	resps := cc.DoMulti(ctx, cmds...)
	for i, resp := range resps {
		if ret[cmds[i].Commands()[1]] = resp.Error(); resp.NonRedisError() == nil {
			intl.PutCompletedForce(cmds[i])
		}
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

func clusterMGet(client Client, ctx context.Context, keys []string) (ret map[string]RedisMessage, err error) {
	ret = make(map[string]RedisMessage, len(keys))
	if len(keys) == 0 {
		return ret, nil
	}

	// Map slot -> index in cmds.s
	hint := len(keys) / 2
	slotIdx := make(map[uint16]int, hint)
	cmds := mgetcmdsp.Get(0, hint)
	defer mgetcmdsp.Put(cmds)

	for _, key := range keys {
		slot := intl.Slot(key)
		idx, ok := slotIdx[slot]
		if !ok {
			slotIdx[slot] = len(cmds.s)
			cmds.s = append(cmds.s, client.B().Mget().Key(key).Build().Pin())
			continue
		}
		intl.AppendCompleted(cmds.s[idx], key)
	}

	resps := client.DoMulti(ctx, cmds.s...)
	defer resultsp.Put(&redisresults{s: resps})

	for i, resp := range resps {
		arr, err := resp.ToArray()
		if err != nil {
			return nil, err
		}
		for j, val := range arr {
			ret[cmds.s[i].Commands()[j+1]] = val
		}
	}

	for i := range cmds.s {
		intl.PutCompletedForce(cmds.s[i])
	}
	return ret, nil
}

func clusterJsonMGet(client Client, ctx context.Context, keys []string, path string) (ret map[string]RedisMessage, err error) {
	ret = make(map[string]RedisMessage, len(keys))
	if len(keys) == 0 {
		return ret, nil
	}

	// Map slot -> index in cmds.s
	hint := len(keys) / 2
	slotIdx := make(map[uint16]int, hint)
	cmds := mgetcmdsp.Get(0, hint)
	defer mgetcmdsp.Put(cmds)

	for _, key := range keys {
		slot := intl.Slot(key)
		idx, ok := slotIdx[slot]
		if !ok {
			slotIdx[slot] = len(cmds.s)
			cmds.s = append(cmds.s, client.B().Arbitrary("JSON.MGET").Keys(key).MultiGet().Pin())
			continue
		}
		intl.AppendCompleted(cmds.s[idx], key)
	}

	for _, c := range cmds.s {
		intl.AppendCompleted(c, path)
	}

	resps := client.DoMulti(ctx, cmds.s...)
	defer resultsp.Put(&redisresults{s: resps})

	for i, resp := range resps {
		arr, err := resp.ToArray()
		if err != nil {
			return nil, err
		}
		for j, val := range arr {
			ret[cmds.s[i].Commands()[j+1]] = val
		}
	}

	for i := range cmds.s {
		intl.PutCompletedForce(cmds.s[i])
	}
	return ret, nil
}

// ErrMSetNXNotSet is used in the MSetNX helper when the underlying MSETNX response is 0.
// Ref: https://redis.io/commands/msetnx/
var ErrMSetNXNotSet = errors.New("MSETNX: no key was set")

type Scanner struct {
	next func(cursor uint64) (ScanEntry, error)
	err  error
}

func NewScanner(next func(cursor uint64) (ScanEntry, error)) *Scanner {
	return &Scanner{next: next}
}

func (s *Scanner) scan() iter.Seq[[]string] {
	return func(yield func([]string) bool) {
		var e ScanEntry
		for e, s.err = s.next(0); s.err == nil && yield(e.Elements) && e.Cursor != 0; {
			e, s.err = s.next(e.Cursor)
		}
	}
}

func (s *Scanner) Iter() iter.Seq[string] {
	return func(yield func(string) bool) {
		for vs := range s.scan() {
			for _, v := range vs {
				if !yield(v) {
					return
				}
			}
		}
	}
}

func (s *Scanner) Iter2() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for vs := range s.scan() {
			for i := 0; i+1 < len(vs); i += 2 {
				if !yield(vs[i], vs[i+1]) {
					return
				}
			}
		}
	}
}

func (s *Scanner) Err() error {
	return s.err
}

// PreferReplicaNodeSelector prioritizes reading from any replica using Round-Robin.
// If no replicas are available, it falls back to the primary.
func PreferReplicaNodeSelector() ReadNodeSelectorFunc {
	var counter atomic.Uint32
	return func(_ uint16, nodes []NodeInfo) int {
		length := uint32(len(nodes))
		if length > 1 {
			c := counter.Add(1)
			return int(c%(length-1)) + 1
		}
		return -1
	}
}

// AZAffinityNodeSelector prioritizes replicas in the same AZ using Round-Robin.
func AZAffinityNodeSelector(clientAZ string) ReadNodeSelectorFunc {
	return newAZSelector(clientAZ, 1)
}

// AZAffinityReplicasAndPrimaryNodeSelector prioritizes:
// 1. Same-AZ Replicas
// 2. Same-AZ Primary
// 3. Any Replica
// 4. Primary
func AZAffinityReplicasAndPrimaryNodeSelector(clientAZ string) ReadNodeSelectorFunc {
	var counter atomic.Uint32
	return func(_ uint16, nodes []NodeInfo) int {
		// Same-AZ Replicas
		if idx := pickAZ(nodes, clientAZ, 1, &counter); idx != -1 {
			return idx
		}

		length := uint32(len(nodes))

		// Same-AZ Primary
		if length > 0 && nodes[0].AZ == clientAZ {
			return 0
		}

		// Any Replica
		if length > 1 {
			c := counter.Add(1)
			return int(c%(length-1)) + 1
		}
		return -1
	}
}

// newAZSelector creates the internal selector closure with a specific start index.
func newAZSelector(clientAZ string, startIdx int) func(uint16, []NodeInfo) int {
	var counter atomic.Uint32
	return func(_ uint16, nodes []NodeInfo) int {
		// Round-Robin on Same-AZ Replicas
		if idx := pickAZ(nodes, clientAZ, startIdx, &counter); idx != -1 {
			return idx
		}

		// Round-Robin on ALL available nodes
		if count := uint32(len(nodes) - startIdx); count > 0 {
			c := counter.Add(1)
			return int(c%count) + startIdx
		}

		return -1
	}
}

// pickAZ selects a node index from nodes[startIdx:] that matches the clientAZ.
func pickAZ(nodes []NodeInfo, clientAZ string, startIdx int, counter *atomic.Uint32) int {
	n := len(nodes)
	if n <= startIdx {
		return -1
	}

	// We cap the search at 255 nodes
	limit := min(n, 255)
	var matches [8]uint8
	var count uint32 = 0

	for i := startIdx; i < limit; i++ {
		if nodes[i].AZ == clientAZ {
			matches[count] = uint8(i)
			count++
			if count == 8 {
				break
			}
		}
	}

	if count == 0 {
		return -1
	}

	// Round-Robin Selection
	c := counter.Add(1)
	k := c % count

	return int(matches[k])
}
